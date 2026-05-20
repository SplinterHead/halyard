package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type VolumeAggregator struct {
	docker      *docker.Client
	client      *http.Client
	pruneClient *http.Client
}

func NewVolumeAggregator(cli *docker.Client) *VolumeAggregator {
	return &VolumeAggregator{
		docker:      cli,
		client:      &http.Client{Timeout: 5 * time.Second},
		pruneClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (m *VolumeAggregator) ListAllVolumes(ctx context.Context) ([]api.VolumeInfo, error) {
	// Get nodes to know hostnames
	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]string)
	for _, n := range nodes {
		nodeMap[n.ID] = n.Description.Hostname
	}

	// Get agent tasks to find their IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	allVolumes := make([]api.VolumeInfo, 0)

	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			var ip string
			// Look for IP in the network attachments
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					fullIP := net.Addresses[0]
					if i := strings.Index(fullIP, "/"); i != -1 {
						ip = fullIP[:i]
						// Favor IPs that are not 10.0.0.x (ingress often uses 10.255.x.x)
						// But for now, any valid IP is better than none.
						break
					}
				}
			}

			if ip != "" {
				wg.Add(1)
				go func(taskIP, nodeID string) {
					defer wg.Done()
					resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/volumes", taskIP))
					if err != nil {
						fmt.Printf("Error fetching volumes from %s: %v\n", taskIP, err)
						return
					}
					defer resp.Body.Close()

					var vols []api.VolumeInfo
					if err := json.NewDecoder(resp.Body).Decode(&vols); err == nil {
						mu.Lock()
						for i := range vols {
							vols[i].Node = nodeMap[nodeID]
							allVolumes = append(allVolumes, vols[i])
						}
						mu.Unlock()
					}
				}(ip, task.NodeID)
			}
		}
	}

	wg.Wait()
	return allVolumes, nil
}

func (m *VolumeAggregator) PruneAllVolumes(ctx context.Context) (map[string]interface{}, error) {
	// Get agent tasks to find their IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]interface{})

	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			var ip string
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					fullIP := net.Addresses[0]
					if i := strings.Index(fullIP, "/"); i != -1 {
						ip = fullIP[:i]
						break
					}
				}
			}

			if ip != "" {
				wg.Add(1)
				go func(taskIP, nodeID string) {
					defer wg.Done()
					fmt.Printf("Pruning volumes on node %s at %s\n", nodeID, taskIP)
					resp, err := m.pruneClient.Post(fmt.Sprintf("http://%s:9090/volumes/prune", taskIP), "application/json", nil)
					if err != nil {
						fmt.Printf("Failed to prune volumes on node %s: %v\n", nodeID, err)
						return
					}
					defer resp.Body.Close()

					var report interface{}
					if err := json.NewDecoder(resp.Body).Decode(&report); err == nil {
						mu.Lock()
						results[nodeID] = report
						mu.Unlock()
						fmt.Printf("Successfully pruned volumes on node %s\n", nodeID)
					} else {
						fmt.Printf("Failed to decode prune report from node %s: %v\n", nodeID, err)
					}
				}(ip, task.NodeID)
			}
		}
	}

	wg.Wait()
	return results, nil
}

func (m *VolumeAggregator) DeleteVolume(ctx context.Context, nodeHostname string, volumeName string) error {
	// Get nodes to find node ID from hostname
	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	var matchedNodeID string
	for _, n := range nodes {
		if n.Description.Hostname == nodeHostname || n.ID == nodeHostname {
			matchedNodeID = n.ID
			break
		}
	}

	if matchedNodeID == "" {
		return fmt.Errorf("node not found with hostname: %s", nodeHostname)
	}

	// Find running agent task on that node
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(
			filters.Arg("service", "halyard_agent"),
			filters.Arg("desired-state", "running"),
			filters.Arg("node", matchedNodeID),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to list agent tasks on node %s: %w", matchedNodeID, err)
	}

	var ip string
	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					fullIP := net.Addresses[0]
					if i := strings.Index(fullIP, "/"); i != -1 {
						ip = fullIP[:i]
						break
					}
				}
			}
		}
		if ip != "" {
			break
		}
	}

	if ip == "" {
		return fmt.Errorf("halyard agent not found or not running on node %s", nodeHostname)
	}

	// Send DELETE request to local agent
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("http://%s:9090/volumes?name=%s", ip, volumeName), nil)
	if err != nil {
		return fmt.Errorf("failed to create agent delete request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute agent delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var errMsg string
		var temp struct {
			Error string `json:"error"`
		}
		if json.NewDecoder(resp.Body).Decode(&temp) == nil && temp.Error != "" {
			errMsg = temp.Error
		} else {
			errMsg = fmt.Sprintf("agent responded with status %d", resp.StatusCode)
		}
		return fmt.Errorf("agent error: %s", errMsg)
	}

	return nil
}

