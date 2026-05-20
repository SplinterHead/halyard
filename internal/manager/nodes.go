package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gorilla/websocket"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type NodeManager struct {
	docker *docker.Client
	client *http.Client
	db     *DB
}

func NewNodeManager(cli *docker.Client, db *DB) *NodeManager {
	return &NodeManager{
		docker: cli,
		client: &http.Client{Timeout: 2 * time.Second},
		db:     db,
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

	// Get agent tasks to find their IPs
	log.Println("Listing agent tasks...")
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		log.Printf("Error listing agent tasks: %v", err)
		return nil, err
	}
	log.Printf("Found %d agent tasks", len(agentTasks))

	// Map NodeID -> Task IP
	nodeToIP := make(map[string]string)
	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			log.Printf("Processing task %s on node %s", task.ID, task.NodeID)
			// Find the IP on the halyard_default network
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					ip := net.Addresses[0]
					for i := range ip {
						if ip[i] == '/' {
							cleanIP := ip[:i]
							nodeToIP[task.NodeID] = cleanIP
							log.Printf("Mapped node %s to agent IP %s", task.NodeID, cleanIP)
							break
						}
					}
				}
			}
		}
	}

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

			if ip, ok := nodeToIP[n.ID]; ok {
				// Fetch real stats from agent
				url := fmt.Sprintf("http://%s:9090/stats", ip)
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
						log.Printf("Successfully received stats for node %s (%s)", n.ID, stats.Hostname)
					} else {
						log.Printf("Failed to decode stats from agent %s: %v", ip, err)
					}
				}
			} else {
				log.Printf("No agent IP found for node %s", n.ID)
			}

			mu.Lock()
			result = append(result, stats)
			mu.Unlock()
		}(node)
	}

	wg.Wait()
	return result, nil
}

func (m *NodeManager) StreamStatsWS(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	// Get agent tasks to find their IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return err
	}

	// Map Task IP -> NodeID for injection
	ipToNodeID := make(map[string]string)
	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					ip := net.Addresses[0]
					if i := strings.Index(ip, "/"); i != -1 {
						ipToNodeID[ip[:i]] = task.NodeID
						break
					}
				}
			}
		}
	}

	errChan := make(chan error, 1)
	mu := sync.Mutex{}

	for ip, nodeID := range ipToNodeID {
		go func(agentIP, id string) {
			agentURL := fmt.Sprintf("ws://%s:9090/stats/stream", agentIP)
			agentConn, _, err := websocket.DefaultDialer.Dial(agentURL, nil)
			if err != nil {
				log.Printf("Failed to connect to agent at %s: %v", agentIP, err)
				return
			}
			defer agentConn.Close()

			for {
				var stats api.NodeStats
				err := agentConn.ReadJSON(&stats)
				if err != nil {
					log.Printf("Error reading stats from agent %s: %v", agentIP, err)
					return
				}

				stats.NodeID = id // Inject NodeID so UI can map it
				stats.Timestamp = time.Now()

				mu.Lock()
				if err := clientConn.WriteJSON(stats); err != nil {
					mu.Unlock()
					errChan <- err
					return
				}
				mu.Unlock()
			}
		}(ip, nodeID)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
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
		url := fmt.Sprintf("http://%s:9090/stats", agentIP)
		resp, err := m.client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			var agentStats api.NodeStats
			if err := json.NewDecoder(resp.Body).Decode(&agentStats); err == nil {
				detail.CPUUsage = agentStats.CPUUsage
				detail.MemoryUsage = agentStats.MemoryUsage
				detail.MemoryTotal = agentStats.MemoryTotal
				detail.Uptime = agentStats.Uptime
			}
		}
	}

	return detail, nil
}

func (m *NodeManager) getAgentIPForNode(ctx context.Context, nodeID string) (string, error) {
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return "", err
	}

	for _, task := range agentTasks {
		if task.NodeID == nodeID && task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					ip := net.Addresses[0]
					if i := strings.Index(ip, "/"); i != -1 {
						return ip[:i], nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("agent not found on node %s", nodeID)
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

func (m *NodeManager) PruneCluster(ctx context.Context, req api.PruneRequest) error {
	log.Println("Starting cluster-wide prune with custom options...")
	
	// 1. Prune manager node
	if err := m.docker.Prune(ctx, req.Containers, req.Networks, req.Volumes, req.Images, req.ImagesAll); err != nil {
		log.Printf("Failed to prune manager node: %v", err)
		// Continue to agents anyway
	}

	// 2. Get agent IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return err
	}

	agentIPs := make([]string, 0)
	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					ip := net.Addresses[0]
					if i := strings.Index(ip, "/"); i != -1 {
						agentIPs = append(agentIPs, ip[:i])
						break
					}
				}
			}
		}
	}

	// 3. Serialize options to JSON for the agents
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize prune options: %w", err)
	}
	bodyString := string(bodyBytes)

	// 4. Prune all agents concurrently
	var wg sync.WaitGroup
	for _, ip := range agentIPs {
		wg.Add(1)
		go func(agentIP string) {
			defer wg.Done()
			url := fmt.Sprintf("http://%s:9090/prune", agentIP)
			log.Printf("Triggering prune on agent at %s with options", url)
			resp, err := m.client.Post(url, "application/json", strings.NewReader(bodyString))
			if err != nil {
				log.Printf("Failed to prune agent %s: %v", agentIP, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				log.Printf("Agent %s returned unexpected status: %d", agentIP, resp.StatusCode)
			}
		}(ip)
	}

	wg.Wait()
	log.Println("Cluster-wide prune completed")
	return nil
}
