package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func TestCalculateNetworkIPUsage(t *testing.T) {
	tests := []struct {
		name          string
		resource      types.NetworkResource
		expectedUsed  int
		expectedTotal int
	}{
		{
			name: "no IPAM config (host/none)",
			resource: types.NetworkResource{
				IPAM: network.IPAM{},
			},
			expectedUsed:  0,
			expectedTotal: 0,
		},
		{
			name: "IPv4 /24 subnet with containers and services",
			resource: types.NetworkResource{
				IPAM: network.IPAM{
					Config: []network.IPAMConfig{
						{Subnet: "10.0.1.0/24"},
					},
				},
				Containers: map[string]types.EndpointResource{
					"c1": {IPv4Address: "10.0.1.7/24"},
					"c2": {IPv4Address: "10.0.1.9/24"},
					"c3": {IPv4Address: "10.0.1.8/24"},
				},
				Services: map[string]network.ServiceInfo{
					"s1": {
						VIP: "10.0.1.2",
						Tasks: []network.Task{
							{EndpointIP: "10.0.1.7"}, // duplicate of c1
						},
					},
					"s2": {
						VIP: "10.0.1.5",
						Tasks: []network.Task{
							{EndpointIP: "10.0.1.9"}, // duplicate of c2
						},
					},
				},
			},
			expectedUsed:  5, // 10.0.1.2, 10.0.1.5, 10.0.1.7, 10.0.1.8, 10.0.1.9
			expectedTotal: 256,
		},
		{
			name: "IPv4 /16 subnet with no containers",
			resource: types.NetworkResource{
				IPAM: network.IPAM{
					Config: []network.IPAMConfig{
						{Subnet: "172.17.0.0/16"},
					},
				},
				Containers: map[string]types.EndpointResource{},
			},
			expectedUsed:  0,
			expectedTotal: 65536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, total := CalculateNetworkIPUsage(tt.resource)
			if used != tt.expectedUsed {
				t.Errorf("expected used %d, got %d", tt.expectedUsed, used)
			}
			if total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, total)
			}
		})
	}
}
