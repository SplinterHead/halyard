package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type NetworkAggregator struct {
	docker   *docker.Client
	client   *http.Client
	agentDir *AgentDirectory
}

func NewNetworkAggregator(cli *docker.Client, agentDir *AgentDirectory) *NetworkAggregator {
	token := os.Getenv("HALYARD_AGENT_TOKEN")
	return &NetworkAggregator{
		docker:   cli,
		client:   agentclient.NewClient(token),
		agentDir: agentDir,
	}
}

func (m *NetworkAggregator) ListAllNetworks(ctx context.Context) ([]api.NetworkInfo, error) {
	activeAgents := m.agentDir.GetAllAgents()
	allNets := GatherFromAgents[api.NetworkInfo](m.client, "/networks", activeAgents, nil)

	// If no agents are active, fallback to local docker daemon
	if len(allNets) == 0 && m.docker != nil {
		localNets, err := m.docker.NetworkList(ctx, types.NetworkListOptions{})
		if err == nil {
			for _, n := range localNets {
				stack := n.Labels["com.docker.stack.namespace"]
				if stack == "" {
					stack = "-"
				}
				subnet := ""
				if len(n.IPAM.Config) > 0 {
					subnet = n.IPAM.Config[0].Subnet
				}
				nw, err := m.docker.NetworkInspect(ctx, n.ID, types.NetworkInspectOptions{Verbose: true})
				if err != nil {
					nw, _ = m.docker.NetworkInspect(ctx, n.ID, types.NetworkInspectOptions{Verbose: false})
				}
				usedIPs, totalIPs := docker.CalculateNetworkIPUsage(nw)
				if subnet == "" && len(nw.IPAM.Config) > 0 {
					subnet = nw.IPAM.Config[0].Subnet
				}
				allNets = append(allNets, api.NetworkInfo{
					ID:         n.ID,
					Name:       n.Name,
					Scope:      n.Scope,
					Subnet:     subnet,
					Stack:      stack,
					CreatedAt:  n.Created,
					Internal:   n.Internal,
					Attachable: n.Attachable,
					UsedIPs:    usedIPs,
					TotalIPs:   totalIPs,
				})
			}
		}
	}

	networkMap := make(map[string]api.NetworkInfo)
	for _, n := range allNets {
		key := n.Name
		existing, exists := networkMap[key]
		if !exists {
			networkMap[key] = n
		} else {
			if n.Scope == "swarm" && existing.Scope != "swarm" {
				networkMap[key] = n
			} else if n.UsedIPs > existing.UsedIPs {
				networkMap[key] = n
			}
		}
	}

	// For Swarm-scoped networks, inspect on manager if available to ensure full cluster IPAM visibility
	for key, n := range networkMap {
		if n.Scope == "swarm" && m.docker != nil {
			nw, err := m.docker.NetworkInspect(ctx, n.ID, types.NetworkInspectOptions{Verbose: true})
			if err == nil {
				used, total := docker.CalculateNetworkIPUsage(nw)
				if used > n.UsedIPs {
					n.UsedIPs = used
				}
				if total > 0 {
					n.TotalIPs = total
				}
				networkMap[key] = n
			}
		}
	}

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
	activeAgents := m.agentDir.GetAllAgents()
	if len(activeAgents) == 0 {
		return api.NetworkDetail{}, fmt.Errorf("network %s not found (no active agents)", id)
	}

	type result struct {
		detail api.NetworkDetail
		err    error
	}
	resChan := make(chan result, len(activeAgents))
	var wg sync.WaitGroup

	for _, agent := range activeAgents {
		wg.Add(1)
		go func(taskIP string) {
			defer wg.Done()
			resp, err := m.client.Get(fmt.Sprintf("https://%s:9090/networks/detail?id=%s", taskIP, id))
			if err != nil {
				resChan <- result{err: err}
				return
			}
			defer resp.Body.Close()

			if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
				resChan <- result{err: err}
				return
			}

			var detail api.NetworkDetail
			if err := json.NewDecoder(resp.Body).Decode(&detail); err == nil {
				resChan <- result{detail: detail}
				return
			}
			resChan <- result{err: fmt.Errorf("failed to decode detail")}
		}(agent.IP)
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
	stack := nw.Labels["com.docker.stack.namespace"]
	if stack == "" {
		stack = "-"
	}

	subnet := ""
	if len(nw.IPAM.Config) > 0 {
		subnet = nw.IPAM.Config[0].Subnet
	}

	usedIPs, totalIPs := docker.CalculateNetworkIPUsage(nw)

	detail := api.NetworkDetail{
		NetworkInfo: api.NetworkInfo{
			ID:         nw.ID,
			Name:       nw.Name,
			Scope:      nw.Scope,
			Subnet:     subnet,
			Stack:      stack,
			CreatedAt:  nw.Created,
			Internal:   nw.Internal,
			Attachable: nw.Attachable,
			UsedIPs:    usedIPs,
			TotalIPs:   totalIPs,
		},
		Driver:  nw.Driver,
		Options: nw.Options,
		Labels:  nw.Labels,
		Ingress: nw.Ingress,
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
