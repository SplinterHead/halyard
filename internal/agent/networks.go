package agent

import (
	"context"

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

	networks := make([]api.NetworkInfo, 0, len(resp))
	for _, n := range resp {
		stack := n.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}

		subnet := ""
		gateway := ""
		if len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
			gateway = n.IPAM.Config[0].Gateway
		}

		networks = append(networks, api.NetworkInfo{
			ID:        n.ID,
			Name:      n.Name,
			Driver:    n.Driver,
			Scope:     n.Scope,
			Subnet:    subnet,
			Gateway:   gateway,
			Stack:     stack,
			CreatedAt: n.Created,
		})
	}

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

	detail := api.NetworkDetail{
		NetworkInfo: api.NetworkInfo{
			ID:        nw.ID,
			Name:      nw.Name,
			Driver:    nw.Driver,
			Scope:     nw.Scope,
			CreatedAt: nw.Created,
		},
		Options:    nw.Options,
		Labels:     nw.Labels,
		Internal:   nw.Internal,
		Attachable: nw.Attachable,
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

