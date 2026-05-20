package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/manager"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)


func main() {
	fmt.Println("Halyard Manager starting...")

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	db, err := manager.NewDB("/data/halyard.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sessionSecret, err := db.GetOrCreateSessionSecret()
	if err != nil {
		log.Fatalf("Failed to initialize session secret: %v", err)
	}

	svcMgr := manager.NewServiceManager(cli)
	nodeMgr := manager.NewNodeManager(cli, db)
	stackMgr := manager.NewStackManager(cli)
	volAgg := manager.NewVolumeAggregator(cli)
	netAgg := manager.NewNetworkAggregator(cli)
	contAgg := manager.NewContainerAggregator(cli)
	imgAgg := manager.NewImageAggregator(cli)
	gitMgr := manager.NewGitManager(db)
	registryMgr := manager.NewRegistryManager(db)
	syncMgr := manager.NewGitSyncManager(db, gitMgr, stackMgr, registryMgr)
	settingsMgr := manager.NewSettingsManager(db)
	secretMgr := manager.NewSecretManager(cli)
	configMgr := manager.NewConfigManager(cli)

	worker := manager.NewGitWorker(syncMgr, gitMgr, settingsMgr, stackMgr)
	go worker.Start(context.Background())

	statsWorker := manager.NewStatsWorker(nodeMgr, db)
	go statsWorker.Start(context.Background())

	// API Endpoints
	http.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		hasUsers, err := db.HasUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"has_users": hasUsers})
	})

	http.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req api.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := db.CreateUser(req.Username, req.RealName, req.Password)
		if err != nil {
			if strings.Contains(err.Error(), "registration is disabled") {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		token, err := manager.GenerateToken(user.Username, sessionSecret)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.AuthResponse{
			Token: token,
			User:  *user,
		})
	})

	http.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req api.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := db.AuthenticateUser(req.Username, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		token, err := manager.GenerateToken(user.Username, sessionSecret)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.AuthResponse{
			Token: token,
			User:  *user,
		})
	})

	http.HandleFunc("/api/swarm/tokens", func(w http.ResponseWriter, r *http.Request) {
		swarmInfo, err := cli.SwarmInspect(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Find manager IP
		managerIP := "127.0.0.1" // Fallback
		nodes, err := cli.NodeList(r.Context(), types.NodeListOptions{})
		if err == nil {
			for _, n := range nodes {
				if n.ManagerStatus != nil && n.ManagerStatus.Leader {
					// Format address (usually ip:port)
					managerIP = n.ManagerStatus.Addr
					break
				}
			}
		}

		// If leader not found, just use any ready manager's address
		if managerIP == "127.0.0.1" && err == nil {
			for _, n := range nodes {
				if n.ManagerStatus != nil && n.ManagerStatus.Addr != "" {
					managerIP = n.ManagerStatus.Addr
					break
				}
			}
		}

		response := map[string]string{
			"worker_token":    swarmInfo.JoinTokens.Worker,
			"manager_token":   swarmInfo.JoinTokens.Manager,
			"worker_command":  fmt.Sprintf("docker swarm join --token %s %s", swarmInfo.JoinTokens.Worker, managerIP),
			"manager_command": fmt.Sprintf("docker swarm join --token %s %s", swarmInfo.JoinTokens.Manager, managerIP),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/api/nodes", func(w http.ResponseWriter, r *http.Request) {
		nodes, err := nodeMgr.ListNodes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodes)
	})

	http.HandleFunc("/api/nodes/stream", func(w http.ResponseWriter, r *http.Request) {
		if err := nodeMgr.StreamStatsWS(r.Context(), w, r); err != nil {
			log.Printf("Error streaming node stats via WS: %v", err)
		}
	})

	http.HandleFunc("/api/nodes/history", func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("id")
		if nodeID == "" {
			http.Error(w, "node id is required", http.StatusBadRequest)
			return
		}
		stats, err := db.GetHistoricalNodeStats(nodeID, 24) // Default to 24 hours
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/api/nodes/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		detail, err := nodeMgr.GetNodeDetail(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/api/nodes/labels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID    string `json:"id"`
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.ID == "" || req.Key == "" {
			http.Error(w, "id and key are required", http.StatusBadRequest)
			return
		}

		err := nodeMgr.AddNodeLabel(r.Context(), req.ID, req.Key, req.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/nodes/labels/remove", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.ID == "" || req.Key == "" {
			http.Error(w, "id and key are required", http.StatusBadRequest)
			return
		}

		err := nodeMgr.RemoveNodeLabel(r.Context(), req.ID, req.Key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/stacks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			stacks, err := stackMgr.ListStacks(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stacks)
			return
		}
	})

	http.HandleFunc("/api/stacks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			name := strings.TrimPrefix(r.URL.Path, "/api/stacks/")
			err := stackMgr.RemoveStack(r.Context(), name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/stacks/detail", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		detail, err := stackMgr.GetStackDetail(r.Context(), name, svcMgr, contAgg, volAgg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/api/services/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		detail, err := svcMgr.GetServiceDetail(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/api/volumes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			node := r.URL.Query().Get("node")
			name := r.URL.Query().Get("name")
			if node == "" || name == "" {
				http.Error(w, "node and name are required", http.StatusBadRequest)
				return
			}
			err := volAgg.DeleteVolume(r.Context(), node, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		volumes, err := volAgg.ListAllVolumes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volumes)
	})

	http.HandleFunc("/api/volumes/prune", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		results, err := volAgg.PruneAllVolumes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	http.HandleFunc("/api/networks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			networks, err := netAgg.ListAllNetworks(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(networks)
			return
		}

		if r.Method == http.MethodPost {
			var req api.NetworkCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			id, err := netAgg.CreateNetwork(r.Context(), req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": id})
			return
		}

		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "id is required", http.StatusBadRequest)
				return
			}
			err := netAgg.DeleteNetwork(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/networks/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		detail, err := netAgg.GetNetworkDetail(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/api/containers", func(w http.ResponseWriter, r *http.Request) {
		containers, err := contAgg.ListAllContainers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	})

	http.HandleFunc("/api/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			images, err := imgAgg.ListAllImages(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(images)
			return
		}

		if r.Method == http.MethodDelete {
			nodeID := r.URL.Query().Get("node_id")
			id := r.URL.Query().Get("id")
			force := r.URL.Query().Get("force") == "true"
			if nodeID == "" || id == "" {
				http.Error(w, "node_id and id are required", http.StatusBadRequest)
				return
			}

			err := imgAgg.DeleteImage(r.Context(), nodeID, id, force)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/images/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		nodeID := r.URL.Query().Get("node_id")
		repo := r.URL.Query().Get("repository")
		tag := r.URL.Query().Get("tag")
		id := r.URL.Query().Get("id")

		if nodeID == "" || repo == "" || tag == "" || id == "" {
			http.Error(w, "node_id, repository, tag, and id are required", http.StatusBadRequest)
			return
		}

		// Look up matching credentials from saved registries
		registries, _ := registryMgr.ListRegistries()
		registryAuth := getEncodedRegistryAuth(repo, registries)

		upToDate, err := imgAgg.CheckImage(r.Context(), nodeID, repo, tag, id, registryAuth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]bool{
			"up_to_date": upToDate,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/api/containers/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		node := r.URL.Query().Get("node")
		if id == "" || node == "" {
			http.Error(w, "id and node are required", http.StatusBadRequest)
			return
		}
		detail, err := contAgg.GetContainerDetail(r.Context(), id, node)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/api/containers/logs", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		node := r.URL.Query().Get("node")
		if id == "" || node == "" {
			http.Error(w, "id and node are required", http.StatusBadRequest)
			return
		}
		if err := contAgg.StreamLogsWS(r.Context(), id, node, w, r); err != nil {
			log.Printf("Error streaming logs via WS: %v", err)
		}
	})

	http.HandleFunc("/api/repos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(gitMgr.ListRepos())
			return
		}
		if r.Method == http.MethodPost {
			var repo api.GitRepository
			if err := json.NewDecoder(r.Body).Decode(&repo); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			newRepo, err := gitMgr.AddRepo(repo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newRepo)
			return
		}
	})

	http.HandleFunc("/api/repos/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		repoID := parts[3]

		if r.Method == http.MethodPut {
			var repo api.GitRepository
			if err := json.NewDecoder(r.Body).Decode(&repo); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			repo.ID = repoID
			updated, err := gitMgr.UpdateRepo(repo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(updated)
			return
		}

		if r.Method == http.MethodDelete {
			if err := gitMgr.DeleteRepo(repoID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if len(parts) == 5 && parts[4] == "test" {
			if r.Method == http.MethodPost {
				err := gitMgr.TestRepo(repoID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if len(parts) == 5 && parts[4] == "branches" {
			repo, ok := gitMgr.GetRepo(repoID)
			if !ok {
				http.Error(w, "Repo not found", http.StatusNotFound)
				return
			}
			branches, err := gitMgr.ListBranches(repo)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(branches)
			return
		}

		if len(parts) == 5 && parts[4] == "files" {
			repo, ok := gitMgr.GetRepo(repoID)
			if !ok {
				http.Error(w, "Repo not found", http.StatusNotFound)
				return
			}
			branch := r.URL.Query().Get("branch")
			path := r.URL.Query().Get("path")
			if branch == "" {
				http.Error(w, "branch is required", http.StatusBadRequest)
				return
			}
			files, err := gitMgr.ListFiles(repo, branch, path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(files)
			return
		}
	})

	http.HandleFunc("/api/syncs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(syncMgr.ListSyncs())
			return
		}
		if r.Method == http.MethodPost {
			var sync api.GitSync
			if err := json.NewDecoder(r.Body).Decode(&sync); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			newSync, err := syncMgr.AddSync(sync)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newSync)
			return
		}
	})

	http.HandleFunc("/api/syncs/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		syncID := parts[3]

		if len(parts) == 5 && parts[4] == "sync" {
			if r.Method == http.MethodPost {
				err := syncMgr.ForceSync(syncID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if r.Method == http.MethodPut {
			var sync api.GitSync
			if err := json.NewDecoder(r.Body).Decode(&sync); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sync.ID = syncID
			updated, err := syncMgr.UpdateSync(sync)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(updated)
			return
		}

		if r.Method == http.MethodDelete {
			if err := syncMgr.DeleteSync(syncID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(syncMgr.ListDeploymentHistory())
			return
		}
	})

	http.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(settingsMgr.GetSettings())
			return
		}
		if r.Method == http.MethodPost {
			var s api.Settings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			settingsMgr.UpdateSettings(s)
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	http.HandleFunc("/api/services", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "id is required", http.StatusBadRequest)
				return
			}
			err := svcMgr.RemoveService(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		services, err := svcMgr.ListServices(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(services)
	})

	http.HandleFunc("/api/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			secrets, err := secretMgr.ListSecrets(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(secrets)
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				Name string `json:"name"`
				Data string `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			id, err := secretMgr.CreateSecret(r.Context(), req.Name, []byte(req.Data))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"id": id})
			return
		}
	})

	http.HandleFunc("/api/secrets/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := strings.TrimPrefix(r.URL.Path, "/api/secrets/")
			err := secretMgr.RemoveSecret(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			configs, err := configMgr.ListConfigs(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(configs)
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				Name string `json:"name"`
				Data string `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			id, err := configMgr.CreateConfig(r.Context(), req.Name, []byte(req.Data))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"id": id})
			return
		}
	})

	http.HandleFunc("/api/configs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := strings.TrimPrefix(r.URL.Path, "/api/configs/")
			err := configMgr.RemoveConfig(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/api/swarm/prune", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Set default values in case options are omitted
			req := api.PruneRequest{
				Containers: true,
				Networks:   true,
				Volumes:    true,
				Images:     true,
				ImagesAll:  false,
			}

			if r.ContentLength > 0 {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					log.Printf("Failed to decode swarm prune body: %v (using defaults)", err)
				}
			}

			err := nodeMgr.PruneCluster(r.Context(), req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})


	http.HandleFunc("/api/registries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			registries, err := registryMgr.ListRegistries()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(registries)
			return
		}
		if r.Method == http.MethodPost {
			var reg api.Registry
			if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			newReg, err := registryMgr.AddRegistry(reg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newReg)
		}
	})

	http.HandleFunc("/api/registries/dockerhub-limit", func(w http.ResponseWriter, r *http.Request) {
		var username, password string
		registries, err := registryMgr.ListRegistries()
		if err == nil {
			for _, reg := range registries {
				u := strings.ToLower(reg.URL)
				if u == "" || strings.Contains(u, "docker.io") || strings.Contains(u, "docker.com") {
					username = reg.Username
					password = reg.Password
					break
				}
			}
		}

		limitInfo, err := manager.FetchDockerHubRateLimit(r.Context(), username, password)
		if err != nil {
			log.Printf("Docker Hub authenticated rate limit fetch failed: %v. Retrying anonymously.", err)
			limitInfo, err = manager.FetchDockerHubRateLimit(r.Context(), "", "")
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(limitInfo)
	})

	http.HandleFunc("/api/registries/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/registries/")
		if r.Method == http.MethodPut {
			var reg api.Registry
			if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			reg.ID = id
			updated, err := registryMgr.UpdateRegistry(reg)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(updated)
			return
		}
		if r.Method == http.MethodDelete {
			if err := registryMgr.DeleteRegistry(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	// Serve static files from the VueJS frontend
	fs := http.FileServer(http.Dir("ui/dist"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If the request is for an API or the file exists, serve it
		// Otherwise, serve index.html for SPA routing
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			fs.ServeHTTP(w, r)
			return
		}

		// Simple check if file exists (could be improved)
		// For simplicity, if it's not /api/, try to serve file, then fallback
		if !strings.HasPrefix(path, "/api/") {
			// Check if file exists in ui/dist
			_, err := os.Stat("ui/dist" + path)
			if os.IsNotExist(err) {
				// Fallback to index.html
				http.ServeFile(w, r, "ui/dist/index.html")
				return
			}
			fs.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	log.Println("Server listening on :8080")
	handler := AuthMiddleware(db, sessionSecret, http.DefaultServeMux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func getEncodedRegistryAuth(repo string, registries []api.Registry) string {
	registryHost := "docker.io" // Default Docker Hub
	parts := strings.Split(repo, "/")
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		registryHost = parts[0]
	}

	var matchedUser, matchedPass, matchedURL string
	
	cleanRegistryURL := func(u string) string {
		u = strings.TrimPrefix(u, "https://")
		u = strings.TrimPrefix(u, "http://")
		u = strings.TrimSuffix(u, "/v1")
		u = strings.TrimSuffix(u, "/v1/")
		u = strings.TrimSuffix(u, "/v2")
		u = strings.TrimSuffix(u, "/v2/")
		u = strings.TrimSuffix(u, "/")
		return strings.ToLower(u)
	}

	cleanHost := cleanRegistryURL(registryHost)

	for _, reg := range registries {
		cleanURL := cleanRegistryURL(reg.URL)
		if strings.Contains(cleanURL, cleanHost) || strings.Contains(cleanHost, cleanURL) {
			matchedUser = reg.Username
			matchedPass = reg.Password
			matchedURL = reg.URL
			break
		}
	}

	// If it's docker.io / Docker Hub, and we found no matched credential, look for named Docker Hub registries
	if registryHost == "docker.io" && matchedUser == "" {
		for _, reg := range registries {
			cleanURL := cleanRegistryURL(reg.URL)
			if strings.Contains(cleanURL, "docker.io") || strings.Contains(strings.ToLower(reg.Name), "docker hub") {
				matchedUser = reg.Username
				matchedPass = reg.Password
				matchedURL = reg.URL
				break
			}
		}
	}

	if matchedUser == "" || matchedPass == "" {
		return ""
	}

	authConfig := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress,omitempty"`
	}{
		Username:      matchedUser,
		Password:      matchedPass,
		ServerAddress: matchedURL,
	}

	jsonData, err := json.Marshal(authConfig)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(jsonData)
}

func AuthMiddleware(db *manager.DB, secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Skip authentication for health check, static files, and auth endpoints
		if !strings.HasPrefix(path, "/api/") ||
			path == "/api/auth/status" ||
			path == "/api/auth/login" ||
			path == "/api/auth/register" {
			next.ServeHTTP(w, r)
			return
		}

		// Retrieve token from Authorization header or URL query parameter (for WebSockets)
		var tokenStr string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Validate token
		_, err := manager.ValidateToken(tokenStr, secret)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
