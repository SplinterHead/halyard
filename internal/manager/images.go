package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type ImageAggregator struct {
	docker *docker.Client
	client *http.Client
}

func NewImageAggregator(cli *docker.Client) *ImageAggregator {
	return &ImageAggregator{
		docker: cli,
		client: &http.Client{Timeout: 10 * time.Second},
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

	// Get agent tasks to find their IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	allImages := make([]api.ImageInfo, 0)

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
					resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/images", taskIP))
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
				}(ip, task.NodeID)
			}
		}
	}

	wg.Wait()
	return allImages, nil
}

func (m *ImageAggregator) DeleteImage(ctx context.Context, nodeID string, imageID string, force bool) error {
	// Find task IP for the specified nodeID
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return err
	}

	var ip string
	for _, task := range agentTasks {
		if task.NodeID == nodeID && task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
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
		return fmt.Errorf("agent not found or not running on node %s", nodeID)
	}

	url := fmt.Sprintf("http://%s:9090/images?id=%s&force=%t", ip, imageID, force)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		// Try to read error message
		var buf [512]byte
		n, _ := resp.Body.Read(buf[:])
		errMsg := string(buf[:n])
		if errMsg == "" {
			errMsg = fmt.Sprintf("agent returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("failed to delete image on agent: %s", errMsg)
	}

	return nil
}

func (m *ImageAggregator) CheckImage(ctx context.Context, nodeID string, repository string, tag string, imageID string, registryAuth string) (bool, error) {
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return false, err
	}

	var ip string
	for _, task := range agentTasks {
		if task.NodeID == nodeID && task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
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
		return false, fmt.Errorf("agent not found or not running on node %s", nodeID)
	}

	checkURL := fmt.Sprintf(
		"http://%s:9090/images/check?repository=%s&tag=%s&id=%s",
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

	if resp.StatusCode != http.StatusOK {
		var buf [512]byte
		n, _ := resp.Body.Read(buf[:])
		errMsg := string(buf[:n])
		if errMsg == "" {
			errMsg = fmt.Sprintf("agent returned status %d", resp.StatusCode)
		}
		return false, fmt.Errorf("agent check error: %s", errMsg)
	}

	var result struct {
		UpToDate bool   `json:"up_to_date"`
		Error    string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if result.Error != "" {
		return false, fmt.Errorf(result.Error)
	}

	return result.UpToDate, nil
}

