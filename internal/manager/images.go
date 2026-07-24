package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

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

	mutator := func(item *api.ImageInfo, nodeID string) {
		item.Node = nodeMap[nodeID]
		item.NodeID = nodeID
	}
	allImages := GatherFromAgents[api.ImageInfo](m.client, "/images", activeAgents, mutator)
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

