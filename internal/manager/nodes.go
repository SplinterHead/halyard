package manager

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gorilla/websocket"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type NodeManager struct {
	docker   *docker.Client
	client   *http.Client
	db       *DB
	agentDir *AgentDirectory
}

func NewNodeManager(cli *docker.Client, db *DB, agentDir *AgentDirectory) *NodeManager {
	token := os.Getenv("HALYARD_AGENT_TOKEN")
	return &NodeManager{
		docker:   cli,
		client:   agentclient.NewClient(token),
		db:       db,
		agentDir: agentDir,
	}
}

func (m *NodeManager) ListNodes(ctx context.Context) ([]api.NodeStats, error) {
	log.Println("Listing swarm nodes...")
	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		log.Printf("Error listing nodes: %v", err)
		return nil, err
	}
	log.Printf("Found %d nodes in swarm", len(nodes))

	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make([]api.NodeStats, 0, len(nodes))

	for _, node := range nodes {
		wg.Add(1)
		go func(n swarm.Node) {
			defer wg.Done()

			stats := api.NodeStats{
				NodeID:       n.ID,
				Hostname:     n.Description.Hostname,
				Status:       string(n.Status.State),
				Availability: string(n.Spec.Availability),
				Role:         string(n.Spec.Role),
				Version:      n.Description.Engine.EngineVersion,
				IP:           n.Status.Addr,
			}

			if ip, ok := m.agentDir.GetAgentIP(n.ID); ok {
				// Fetch real stats from agent
				url := fmt.Sprintf("https://%s:9090/stats", ip)
				log.Printf("Fetching stats from agent at %s", url)
				resp, err := m.client.Get(url)
				if err != nil {
					log.Printf("Failed to fetch stats from agent %s: %v", ip, err)
				} else {
					defer resp.Body.Close()
					var agentStats api.NodeStats
					if err := json.NewDecoder(resp.Body).Decode(&agentStats); err == nil {
						stats.CPUUsage = agentStats.CPUUsage
						stats.MemoryUsage = agentStats.MemoryUsage
						stats.MemoryTotal = agentStats.MemoryTotal
						stats.Uptime = agentStats.Uptime
						stats.PendingUpdates = agentStats.PendingUpdates
						stats.RestartRequired = agentStats.RestartRequired
						log.Printf("Successfully received stats for node %s (%s)", n.ID, stats.Hostname)
					} else {
						log.Printf("Failed to decode stats from agent %s: %v", ip, err)
					}
				}
			} else {
				log.Printf("No agent IP found for node %s in AgentDirectory", n.ID)
			}

			mu.Lock()
			result = append(result, stats)
			mu.Unlock()
		}(node)
	}

	wg.Wait()
	return result, nil
}

func (m *NodeManager) StreamStatsWS(parentCtx context.Context, w http.ResponseWriter, r *http.Request) error {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Detect client disconnect immediately
	go func() {
		defer cancel()
		for {
			if _, _, err := clientConn.NextReader(); err != nil {
				break
			}
		}
	}()

	// Get active agents from directory
	activeAgents := m.agentDir.GetAllAgents()
	mu := sync.Mutex{}

	for _, agent := range activeAgents {
		go func(agentIP, id string) {
			agentURL := fmt.Sprintf("wss://%s:9090/stats/stream", agentIP)
			headers := http.Header{}
			if token := os.Getenv("HALYARD_AGENT_TOKEN"); token != "" {
				headers.Add("Authorization", "Bearer "+token)
			}
			dialer := &websocket.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			agentConn, _, err := dialer.Dial(agentURL, headers)
			if err != nil {
				log.Printf("Failed to connect to agent at %s: %v", agentIP, err)
				return
			}
			defer agentConn.Close()

			// Force ReadJSON to exit when context is cancelled
			go func() {
				<-ctx.Done()
				agentConn.Close()
			}()

			for {
				var stats api.NodeStats
				err := agentConn.ReadJSON(&stats)
				if err != nil {
					return // Exits gracefully on close or error
				}

				stats.NodeID = id // Inject NodeID so UI can map it
				stats.Timestamp = time.Now()

				mu.Lock()
				err = clientConn.WriteJSON(stats)
				mu.Unlock()
				if err != nil {
					cancel() // Signal all other goroutines to stop
					return
				}
			}
		}(agent.IP, agent.NodeID)
	}

	<-ctx.Done()
	return nil
}

func (m *NodeManager) GetNodeDetail(ctx context.Context, id string) (api.NodeDetail, error) {
	node, _, err := m.docker.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return api.NodeDetail{}, err
	}

	detail := api.NodeDetail{
		NodeStats: api.NodeStats{
			NodeID:       node.ID,
			Hostname:     node.Description.Hostname,
			Status:       string(node.Status.State),
			Availability: string(node.Spec.Availability),
			Role:         string(node.Spec.Role),
			Version:      node.Description.Engine.EngineVersion,
			IP:           node.Status.Addr,
		},
		OS:            node.Description.Platform.OS,
		Architecture:  node.Description.Platform.Architecture,
		CPUs:          int(node.Description.Resources.NanoCPUs / 1e9),
		Memory:        node.Description.Resources.MemoryBytes,
		Labels:        node.Spec.Labels,
		EngineVersion: node.Description.Engine.EngineVersion,
	}

	for _, p := range node.Description.Engine.Plugins {
		detail.Plugins = append(detail.Plugins, fmt.Sprintf("%s (%s)", p.Name, p.Type))
	}

	// Try to get real-time stats from agent if available
	agentIP, _ := m.getAgentIPForNode(ctx, id)
	if agentIP != "" {
		url := fmt.Sprintf("https://%s:9090/stats", agentIP)
		resp, err := m.client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			var agentStats api.NodeStats
			if err := json.NewDecoder(resp.Body).Decode(&agentStats); err == nil {
				detail.CPUUsage = agentStats.CPUUsage
				detail.MemoryUsage = agentStats.MemoryUsage
				detail.MemoryTotal = agentStats.MemoryTotal
				detail.Uptime = agentStats.Uptime
				detail.PendingUpdates = agentStats.PendingUpdates
				detail.RestartRequired = agentStats.RestartRequired
			}
		}
	}

	return detail, nil
}

func (m *NodeManager) getAgentIPForNode(ctx context.Context, nodeID string) (string, error) {
	if ip, ok := m.agentDir.GetAgentIP(nodeID); ok {
		return ip, nil
	}
	return "", fmt.Errorf("agent not found on node %s in AgentDirectory", nodeID)
}

func (m *NodeManager) AddNodeLabel(ctx context.Context, id, key, value string) error {
	node, _, err := m.docker.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return err
	}

	spec := node.Spec
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	spec.Labels[key] = value

	return m.docker.NodeUpdate(ctx, id, node.Version, spec)
}

func (m *NodeManager) RemoveNodeLabel(ctx context.Context, id, key string) error {
	node, _, err := m.docker.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return err
	}

	spec := node.Spec
	if spec.Labels == nil {
		return nil
	}
	delete(spec.Labels, key)

	return m.docker.NodeUpdate(ctx, id, node.Version, spec)
}

func (m *NodeManager) UpdateNode(ctx context.Context, id string, availability string, role string) error {
	node, _, err := m.docker.NodeInspectWithRaw(ctx, id)
	if err != nil {
		return err
	}

	spec := node.Spec
	updated := false

	if availability != "" {
		spec.Availability = swarm.NodeAvailability(availability)
		updated = true
	}

	if role != "" {
		spec.Role = swarm.NodeRole(role)
		updated = true
	}

	if !updated {
		return nil
	}

	return m.docker.NodeUpdate(ctx, id, node.Version, spec)
}


func (m *NodeManager) PruneCluster(ctx context.Context, req api.PruneRequest) error {
	log.Println("Starting cluster-wide prune with custom options...")
	
	// 1. Prune manager node
	if err := m.docker.Prune(ctx, req.Containers, req.Networks, req.Volumes, req.Images, req.ImagesAll); err != nil {
		log.Printf("Failed to prune manager node: %v", err)
		// Continue to agents anyway
	}

	// 2. Get active agents from directory
	activeAgents := m.agentDir.GetAllAgents()

	// 3. Serialize options to JSON for the agents
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize prune options: %w", err)
	}
	bodyString := string(bodyBytes)

	// 4. Prune all agents concurrently
	var wg sync.WaitGroup
	for _, agent := range activeAgents {
		wg.Add(1)
		go func(agentIP string) {
			defer wg.Done()
			url := fmt.Sprintf("https://%s:9090/prune", agentIP)
			log.Printf("Triggering prune on agent at %s with options", url)
			resp, err := m.client.Post(url, "application/json", strings.NewReader(bodyString))
			if err != nil {
				log.Printf("Failed to prune agent %s: %v", agentIP, err)
				return
			}
			defer resp.Body.Close()
			if err := agentclient.ParseError(resp, http.StatusNoContent); err != nil {
				log.Printf("Agent %s prune error: %v", agentIP, err)
			}
		}(agent.IP)
	}

	wg.Wait()
	log.Println("Cluster-wide prune completed")
	return nil
}

func (m *NodeManager) GetPendingUpdates(ctx context.Context, nodeID string) ([]string, error) {
	agentIP, err := m.getAgentIPForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s:9090/host/updates/list", agentIP)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var packages []string
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, err
	}
	return packages, nil
}

func (m *NodeManager) StreamHostUpdate(ctx context.Context, nodeID string, w http.ResponseWriter, r *http.Request) error {
	agentIP, err := m.getAgentIPForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s:9090/host/update", agentIP)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	// Important: We need a client without a timeout for streaming
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fmt.Fprintf(w, "%s\n", scanner.Text())
		flusher.Flush()
	}

	return scanner.Err()
}

func (m *NodeManager) HostReboot(ctx context.Context, nodeID string) error {
	agentIP, err := m.getAgentIPForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s:9090/host/reboot", agentIP)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return agentclient.ParseError(resp, http.StatusAccepted)
}
