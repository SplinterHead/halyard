package agent

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type NetworkManager struct {
	docker *docker.Client
}

func NewNetworkManager(cli *docker.Client) *NetworkManager {
	return &NetworkManager{docker: cli}
}

func (m *NetworkManager) ListNetworks(ctx context.Context) ([]api.NetworkInfo, error) {
	resp, err := m.docker.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	networks := make([]api.NetworkInfo, len(resp))
	var wg sync.WaitGroup
	for i, n := range resp {
		wg.Add(1)
		go func(idx int, netResource types.NetworkResource) {
			defer wg.Done()
			stack := netResource.Labels["com.docker.stack.namespace"]
			if stack == "" {
				stack = "-"
			}

			subnet := ""
			if len(netResource.IPAM.Config) > 0 {
				subnet = netResource.IPAM.Config[0].Subnet
			}

			nw, err := m.docker.NetworkInspect(ctx, netResource.ID, types.NetworkInspectOptions{Verbose: true})
			if err != nil {
				nw, _ = m.docker.NetworkInspect(ctx, netResource.ID, types.NetworkInspectOptions{Verbose: false})
			}

			usedIPs, totalIPs := docker.CalculateNetworkIPUsage(nw)
			if subnet == "" && len(nw.IPAM.Config) > 0 {
				subnet = nw.IPAM.Config[0].Subnet
			}

			networks[idx] = api.NetworkInfo{
				ID:         netResource.ID,
				Name:       netResource.Name,
				Scope:      netResource.Scope,
				Subnet:     subnet,
				Stack:      stack,
				CreatedAt:  netResource.Created,
				Internal:   netResource.Internal,
				Attachable: netResource.Attachable,
				UsedIPs:    usedIPs,
				TotalIPs:   totalIPs,
			}
		}(i, n)
	}
	wg.Wait()

	return networks, nil
}

func (m *NetworkManager) GetNetworkDetail(ctx context.Context, id string) (api.NetworkDetail, error) {
	nw, err := m.docker.NetworkInspect(ctx, id, types.NetworkInspectOptions{Verbose: true})
	if err != nil {
		// Fallback without verbose in case it is a system/host network where verbose isn't supported
		nw, err = m.docker.NetworkInspect(ctx, id, types.NetworkInspectOptions{Verbose: false})
		if err != nil {
			return api.NetworkDetail{}, err
		}
	}

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
		Driver:     nw.Driver,
		Options:    nw.Options,
		Labels:     nw.Labels,
		Ingress:    nw.Ingress,
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

	return detail, nil
}

