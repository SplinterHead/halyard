package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/agent"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

func main() {
	fmt.Println("Halyard Agent starting...")

	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}

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

		go statsColl.StreamStats(ctx, statsChan, 2*time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case stats := <-statsChan:
				if err := conn.WriteJSON(stats); err != nil {
					log.Printf("Error writing stats to websocket: %v", err)
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

		// Helper to write log chunks to websocket
		writer := &wsWriter{conn: conn}
		docker.DemuxLogs(stream, writer)
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

	log.Println("Agent listening on :9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
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
