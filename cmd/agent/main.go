package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/agent"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

func main() {
	fmt.Println("Halyard Agent starting...")

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

	go startHeartbeat(cli)

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	statsColl := agent.NewStatsCollector(cli)
	volMgr := agent.NewVolumeManager(cli)
	netMgr := agent.NewNetworkManager(cli)
	contMgr := agent.NewContainerManager(cli)
	imgMgr := agent.NewImageManager(cli)
	
	// Basic router for stats and health
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := statsColl.GetNodeStats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	http.HandleFunc("/stats/stream", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade agent stats stream: %v", err)
			return
		}
		defer conn.Close()

		statsChan := make(chan api.NodeStats)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Detect client disconnect immediately
		go func() {
			defer cancel()
			for {
				if _, _, err := conn.NextReader(); err != nil {
					break
				}
			}
		}()

		go statsColl.StreamStats(ctx, statsChan, 2*time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case stats := <-statsChan:
				if err := conn.WriteJSON(stats); err != nil {
					// Client disconnected (e.g. manager dropped connection)
					return
				}
			}
		}
	})

	http.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			name := r.URL.Query().Get("name")
			if name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}
			force := r.URL.Query().Get("force") == "true"
			if err := volMgr.RemoveVolume(r.Context(), name, force); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		volumes, err := volMgr.ListVolumes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volumes)
	})

	http.HandleFunc("/volumes/browse", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		path := r.URL.Query().Get("path")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if path == "" {
			path = "/"
		}

		entries, err := volMgr.BrowseVolume(r.Context(), name, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	})

	http.HandleFunc("/volumes/prune", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received volume prune request from %s", r.RemoteAddr)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		report, err := volMgr.PruneVolumes(r.Context())
		if err != nil {
			log.Printf("Failed to prune volumes: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Pruned %d volumes, reclaimed %d bytes", len(report.VolumesDeleted), report.SpaceReclaimed)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	http.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		networks, err := netMgr.ListNetworks(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(networks)
	})

	http.HandleFunc("/networks/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		detail, err := netMgr.GetNetworkDetail(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})


	http.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
		containers, err := contMgr.ListContainers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	})

	http.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "id is required", http.StatusBadRequest)
				return
			}
			force := r.URL.Query().Get("force") == "true"
			if err := imgMgr.RemoveImage(r.Context(), id, force); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		images, err := imgMgr.ListImages(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(images)
	})

	http.HandleFunc("/images/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.URL.Query().Get("repository")
		tag := r.URL.Query().Get("tag")
		id := r.URL.Query().Get("id")
		auth := r.Header.Get("X-Registry-Auth")

		if repo == "" || tag == "" || id == "" {
			http.Error(w, "repository, tag, and id are required", http.StatusBadRequest)
			return
		}

		upToDate, err := imgMgr.CheckImageUpToDate(r.Context(), repo, tag, id, auth)
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}

		response := map[string]interface{}{
			"up_to_date": upToDate,
			"error":      errMsg,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/containers/detail", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		detail, err := contMgr.GetContainerDetail(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	})

	http.HandleFunc("/containers/logs", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade agent logs connection: %v", err)
			return
		}
		defer conn.Close()
		
		stream, err := contMgr.StreamLogs(r.Context(), id)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
			return
		}
		defer stream.Close()

		// Detect client disconnect immediately
		go func() {
			for {
				if _, _, err := conn.NextReader(); err != nil {
					// Close stream to unblock DemuxLogs
					stream.Close()
					break
				}
			}
		}()

		// Helper to write log chunks to websocket
		writer := &wsWriter{conn: conn}
		docker.DemuxLogs(stream, writer)
	})

	http.HandleFunc("/containers/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := contMgr.StopContainer(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/containers/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := contMgr.StartContainer(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/containers/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := contMgr.RestartContainer(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/containers/remove", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		force := r.URL.Query().Get("force") == "true"
		if err := contMgr.RemoveContainer(r.Context(), id, force); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/containers/exec", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		shell := r.URL.Query().Get("shell")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if shell == "" {
			shell = "/bin/sh"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade agent exec connection: %v", err)
			return
		}
		defer conn.Close()

		err = contMgr.ExecContainerWS(r.Context(), id, shell, conn)
		if err != nil {
			log.Printf("Error during container exec WS: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte("\r\nError: "+err.Error()+"\r\n"))
		}
	})

	http.HandleFunc("/prune", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received global prune request from %s", r.RemoteAddr)
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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
				log.Printf("Failed to decode prune options: %v (using defaults)", err)
			}
		}

		if err := cli.Prune(r.Context(), req.Containers, req.Networks, req.Volumes, req.Images, req.ImagesAll); err != nil {
			log.Printf("Failed to prune node: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/host/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		log.Println("Received request to stream host updates via nsenter")
		
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		stream, err := cli.StreamHostCommand(ctx, "DEBIAN_FRONTEND=noninteractive apt-get update && DEBIAN_FRONTEND=noninteractive apt-get upgrade -y")
		if err != nil {
			fmt.Fprintf(w, "data: ERROR: %v\n\n", err)
			flusher.Flush()
			return
		}
		defer stream.Close()

		fmt.Fprintf(w, "data: Update started...\n\n")
		flusher.Flush()

		lineChan := make(chan string)
		errChan := make(chan error, 1)

		go func() {
			scanner := bufio.NewScanner(stream)
			for scanner.Scan() {
				lineChan <- scanner.Text()
			}
			errChan <- scanner.Err()
		}()

		keepAlive := time.NewTicker(15 * time.Second)
		defer keepAlive.Stop()

		for {
			select {
			case line := <-lineChan:
				fmt.Fprintf(w, "data: %s\n\n", line)
				flusher.Flush()
			case err := <-errChan:
				if err != nil {
					fmt.Fprintf(w, "data: ERROR: %v\n\n", err)
				} else {
					statsColl.CheckHostUpdates()
					fmt.Fprintf(w, "data: DONE\n\n")
				}
				flusher.Flush()
				return // End of stream
			case <-keepAlive.C:
				// Send an SSE comment to bypass proxy idle timeouts (e.g. Traefik/AWS)
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	})

	http.HandleFunc("/host/updates/list", func(w http.ResponseWriter, r *http.Request) {
		packages, err := statsColl.ListPendingUpdates(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(packages)
	})

	http.HandleFunc("/host/reboot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		log.Println("Received request to reboot host via nsenter")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			out, err := cli.RunHostCommand(ctx, "reboot")
			if err != nil {
				log.Printf("Error running host reboot: %v\nOutput: %s", err, string(out))
			}
		}()
		
		w.WriteHeader(http.StatusAccepted)
	})

	log.Println("Generating ephemeral TLS certificate...")
	cert, err := agentclient.GenerateSelfSignedCert()
	if err != nil {
		log.Fatalf("Failed to generate TLS certificate: %v", err)
	}

	token := os.Getenv("HALYARD_AGENT_TOKEN")
	handler := AuthMiddleware(token, http.DefaultServeMux)

	server := &http.Server{
		Addr:    ":9090",
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	log.Println("Agent listening securely on :9090")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func AuthMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func startHeartbeat(cli *docker.Client) {
	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Initial heartbeat immediately on startup
	sendHeartbeat(cli, managerURL)

	for range ticker.C {
		sendHeartbeat(cli, managerURL)
	}
}

func sendHeartbeat(cli *docker.Client, managerURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := cli.Info(ctx)
	if err != nil {
		log.Printf("Heartbeat error: failed to get docker info: %v", err)
		return
	}

	nodeID := info.Swarm.NodeID
	if nodeID == "" {
		log.Printf("Heartbeat error: node is not part of a swarm")
		return
	}

	hostname := os.Getenv("NODE_HOSTNAME")
	if hostname == "" {
		hostname = info.Name
	}

	ip, err := getOverlayIP(managerURL)
	if err != nil {
		log.Printf("Heartbeat error: failed to determine overlay IP: %v", err)
		return
	}

	payload := api.AgentHeartbeat{
		NodeID:   nodeID,
		IP:       ip,
		Hostname: hostname,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Heartbeat error: failed to marshal payload: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, managerURL+"/api/internal/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("Heartbeat error: failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Heartbeat error: failed to push to manager: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heartbeat error: manager returned status %d", resp.StatusCode)
	}
}

func getOverlayIP(managerURL string) (string, error) {
	hostPort := strings.TrimPrefix(managerURL, "http://")
	hostPort = strings.TrimPrefix(hostPort, "https://")
	if i := strings.Index(hostPort, "/"); i != -1 {
		hostPort = hostPort[:i]
	}

	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	if i := strings.LastIndex(localAddr, ":"); i != -1 {
		return localAddr[:i], nil
	}
	return localAddr, nil
}

