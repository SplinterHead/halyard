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
	"github.com/docker/docker/api/types/network"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type NetworkAggregator struct {
	docker *docker.Client
	client *http.Client
}

func NewNetworkAggregator(cli *docker.Client) *NetworkAggregator {
	return &NetworkAggregator{
		docker: cli,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (m *NetworkAggregator) ListAllNetworks(ctx context.Context) ([]api.NetworkInfo, error) {
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
	networkMap := make(map[string]api.NetworkInfo)

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
					resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/networks", taskIP))
					if err != nil {
						return
					}
					defer resp.Body.Close()

					var nets []api.NetworkInfo
					if err := json.NewDecoder(resp.Body).Decode(&nets); err == nil {
						mu.Lock()
						for _, n := range nets {
							nodeName := nodeMap[nodeID]
							key := n.Name

							existing, exists := networkMap[key]
							if !exists {
								// First time seeing this network name
								if n.Scope == "swarm" {
									n.Node = "Swarm"
								} else {
									n.Node = nodeName
								}
								networkMap[key] = n
							} else {
								// Already seen this network name
								if n.Scope == "swarm" && existing.Scope != "swarm" {
									// Upgrade to Swarm scope if found
									n.Node = "Swarm"
									networkMap[key] = n
								} else if n.Scope != "swarm" && existing.Scope != "swarm" {
									// Both local, mark as Multi-Node if from different nodes
									if existing.Node != nodeName && existing.Node != "Multi-Node" {
										existing.Node = "Multi-Node"
										networkMap[key] = existing
									}
								}
							}
						}
						mu.Unlock()
					}
				}(ip, task.NodeID)
			}
		}
	}

	wg.Wait()

	// Convert map to slice
	allNetworks := make([]api.NetworkInfo, 0, len(networkMap))
	for _, n := range networkMap {
		allNetworks = append(allNetworks, n)
	}

	return allNetworks, nil
}

func (m *NetworkAggregator) CreateNetwork(ctx context.Context, req api.NetworkCreateRequest) (string, error) {
	opts := types.NetworkCreate{
		Driver:     req.Driver,
		Attachable: req.Attachable,
		Internal:   req.Internal,
		Options:    req.Options,
		Labels:     req.Labels,
	}

	if req.Driver == "overlay" {
		opts.CheckDuplicate = true
	}

	if req.IPAM != nil && req.IPAM.Subnet != "" {
		opts.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  req.IPAM.Subnet,
					Gateway: req.IPAM.Gateway,
				},
			},
		}
	}

	resp, err := m.docker.NetworkCreate(ctx, req.Name, opts)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}
func (m *NetworkAggregator) GetNetworkDetail(ctx context.Context, id string) (api.NetworkDetail, error) {
	// 1. Try local inspect on manager first
	nw, err := m.docker.NetworkInspect(ctx, id, types.NetworkInspectOptions{Verbose: true})
	if err != nil {
		// Fallback without verbose (for system networks)
		nw, err = m.docker.NetworkInspect(ctx, id, types.NetworkInspectOptions{Verbose: false})
	}

	if err == nil {
		// Found locally on manager! Format and return
		return m.formatDetail(nw), nil
	}

	// 2. If not found locally, it could be node-local to a worker. Ask agents.
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return api.NetworkDetail{}, err
	}

	type result struct {
		detail api.NetworkDetail
		err    error
	}
	resChan := make(chan result, len(agentTasks))
	var wg sync.WaitGroup

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
				go func(taskIP string) {
					defer wg.Done()
					resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/networks/detail?id=%s", taskIP, id))
					if err != nil {
						resChan <- result{err: err}
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						var detail api.NetworkDetail
						if err := json.NewDecoder(resp.Body).Decode(&detail); err == nil {
							resChan <- result{detail: detail}
							return
						}
					}
					resChan <- result{err: fmt.Errorf("status %d", resp.StatusCode)}
				}(ip)
			}
		}
	}

	// Wait in a separate goroutine and close channel
	go func() {
		wg.Wait()
		close(resChan)
	}()

	// Look for first successful response
	for res := range resChan {
		if res.err == nil {
			return res.detail, nil
		}
	}

	return api.NetworkDetail{}, fmt.Errorf("network %s not found on any node", id)
}

func (m *NetworkAggregator) formatDetail(nw types.NetworkResource) api.NetworkDetail {
	detail := api.NetworkDetail{
		NetworkInfo: api.NetworkInfo{
			ID:        nw.ID,
			Name:      nw.Name,
			Driver:    nw.Driver,
			Scope:     nw.Scope,
			CreatedAt: nw.Created,
			Node:      "Swarm", // Since we inspected it on the manager, it's Swarm scope or manager-local
		},
		Options:    nw.Options,
		Labels:     nw.Labels,
		Internal:   nw.Internal,
		Attachable: nw.Attachable,
		Ingress:    nw.Ingress,
	}

	if nw.Scope != "swarm" {
		detail.Node = "Manager"
	}

	for _, cfg := range nw.IPAM.Config {
		detail.IPAMConfig = append(detail.IPAMConfig, api.IPAMConfig{
			Subnet:  cfg.Subnet,
			Gateway: cfg.Gateway,
		})
	}

	for containerID, container := range nw.Containers {
		detail.Containers = append(detail.Containers, api.NetworkContainer{
			ID:   containerID,
			Name: container.Name,
			IPv4: container.IPv4Address,
			IPv6: container.IPv6Address,
		})
	}

	return detail
}


func (m *NetworkAggregator) DeleteNetwork(ctx context.Context, id string) error {
	return m.docker.NetworkRemove(ctx, id)
}
