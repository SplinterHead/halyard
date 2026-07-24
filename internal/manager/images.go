package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ImageAggregator struct {
	docker   *docker.Client
	client   *http.Client
	agentDir *AgentDirectory
}

func NewImageAggregator(cli *docker.Client, agentDir *AgentDirectory) *ImageAggregator {
	token := os.Getenv("HALYARD_AGENT_TOKEN")
	return &ImageAggregator{
		docker:   cli,
		client:   agentclient.NewClient(token),
		agentDir: agentDir,
	}
}

func (m *ImageAggregator) ListAllImages(ctx context.Context) ([]api.ImageInfo, error) {
	// Get nodes to map ID to Hostname
	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]string)
	for _, n := range nodes {
		nodeMap[n.ID] = n.Description.Hostname
	}

	activeAgents := m.agentDir.GetAllAgents()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allImages := make([]api.ImageInfo, 0)

	for _, agent := range activeAgents {
		wg.Add(1)
		go func(taskIP, nodeID string) {
			defer wg.Done()
			resp, err := m.client.Get(fmt.Sprintf("https://%s:9090/images", taskIP))
			if err != nil {
				fmt.Printf("Error fetching images from %s: %v\n", taskIP, err)
				return
			}
			defer resp.Body.Close()

			var imgs []api.ImageInfo
			if err := json.NewDecoder(resp.Body).Decode(&imgs); err == nil {
				mu.Lock()
				for i := range imgs {
					imgs[i].Node = nodeMap[nodeID]
					imgs[i].NodeID = nodeID
					allImages = append(allImages, imgs[i])
				}
				mu.Unlock()
			}
		}(agent.IP, agent.NodeID)
	}

	wg.Wait()
	return allImages, nil
}

func (m *ImageAggregator) DeleteImage(ctx context.Context, nodeID string, imageID string, force bool) error {
	ip, ok := m.agentDir.GetAgentIP(nodeID)
	if !ok {
		return fmt.Errorf("agent not found or not running on node %s", nodeID)
	}

	url := fmt.Sprintf("https://%s:9090/images?id=%s&force=%t", ip, imageID, force)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := agentclient.ParseError(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("failed to delete image on agent: %w", err)
	}

	return nil
}

func (m *ImageAggregator) CheckImage(ctx context.Context, nodeID string, repository string, tag string, imageID string, registryAuth string) (bool, error) {
	ip, ok := m.agentDir.GetAgentIP(nodeID)
	if !ok {
		return false, fmt.Errorf("agent not found or not running on node %s", nodeID)
	}

	checkURL := fmt.Sprintf(
		"https://%s:9090/images/check?repository=%s&tag=%s&id=%s",
		ip,
		url.QueryEscape(repository),
		url.QueryEscape(tag),
		url.QueryEscape(imageID),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, checkURL, nil)
	if err != nil {
		return false, err
	}

	if registryAuth != "" {
		req.Header.Set("X-Registry-Auth", registryAuth)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
		return false, fmt.Errorf("agent check error: %w", err)
	}

	var result struct {
		UpToDate bool   `json:"up_to_date"`
		Error    string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, fmt.Errorf("%s", result.Error)
	}

	return result.UpToDate, nil
}

