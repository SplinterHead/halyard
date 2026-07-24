package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type VolumeAggregator struct {
	docker      *docker.Client
	client      *http.Client
	pruneClient *http.Client
	agentDir    *AgentDirectory
}

func NewVolumeAggregator(cli *docker.Client, agentDir *AgentDirectory) *VolumeAggregator {
	token := os.Getenv("HALYARD_AGENT_TOKEN")
	return &VolumeAggregator{
		docker:      cli,
		client:      agentclient.NewClient(token),
		pruneClient: agentclient.NewPruneClient(token),
		agentDir:    agentDir,
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

	// Get active agents from directory
	activeAgents := m.agentDir.GetAllAgents()

	mutator := func(item *api.VolumeInfo, nodeID string) {
		item.Node = nodeMap[nodeID]
	}
	allVolumes := GatherFromAgents[api.VolumeInfo](m.client, "/volumes", activeAgents, mutator)
	return allVolumes, nil
}

func (m *VolumeAggregator) PruneAllVolumes(ctx context.Context) (map[string]interface{}, error) {
	// Get active agents from directory
	activeAgents := m.agentDir.GetAllAgents()

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]interface{})

	for _, agent := range activeAgents {
		wg.Add(1)
		go func(taskIP, nodeID string) {
			defer wg.Done()
			fmt.Printf("Pruning volumes on node %s at %s\n", nodeID, taskIP)
			resp, err := m.pruneClient.Post(fmt.Sprintf("https://%s:9090/volumes/prune", taskIP), "application/json", nil)
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
		}(agent.IP, agent.NodeID)
	}

	wg.Wait()
	return results, nil
}

func (m *VolumeAggregator) DeleteVolume(ctx context.Context, nodeHostname string, volumeName string) error {
	ip, ok := m.agentDir.GetAgentIPByHostname(nodeHostname)
	if !ok {
		return fmt.Errorf("halyard agent not found or not running on node %s in AgentDirectory", nodeHostname)
	}

	// Send DELETE request to local agent
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("https://%s:9090/volumes?name=%s", ip, volumeName), nil)
	if err != nil {
		return fmt.Errorf("failed to create agent delete request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute agent delete request: %w", err)
	}
	defer resp.Body.Close()

	if err := agentclient.ParseError(resp, http.StatusNoContent, http.StatusOK); err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	return nil
}

func (m *VolumeAggregator) BrowseVolume(ctx context.Context, nodeHostname string, volumeName string, path string) ([]api.VolumeFileEntry, error) {
	ip, ok := m.agentDir.GetAgentIPByHostname(nodeHostname)
	if !ok {
		return nil, fmt.Errorf("halyard agent not found or not running on node %s in AgentDirectory", nodeHostname)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://%s:9090/volumes/browse?name=%s&path=%s", ip, volumeName, path), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent browse request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute agent browse request: %w", err)
	}
	defer resp.Body.Close()

	if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("agent error: %w", err)
	}

	var entries []api.VolumeFileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode agent response: %w", err)
	}

	return entries, nil
}

