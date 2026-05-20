package manager

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type ConfigManager struct {
	docker *docker.Client
}

func NewConfigManager(cli *docker.Client) *ConfigManager {
	return &ConfigManager{docker: cli}
}

func (m *ConfigManager) ListConfigs(ctx context.Context) ([]api.ConfigInfo, error) {
	configs, err := m.docker.ConfigList(ctx, types.ConfigListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]api.ConfigInfo, 0, len(configs))
	for _, c := range configs {
		result = append(result, api.ConfigInfo{
			ID:        c.ID,
			Name:      c.Spec.Name,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Labels:    c.Spec.Labels,
		})
	}
	return result, nil
}

func (m *ConfigManager) CreateConfig(ctx context.Context, name string, data []byte) (string, error) {
	resp, err := m.docker.ConfigCreate(ctx, swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: name,
		},
		Data: data,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (m *ConfigManager) RemoveConfig(ctx context.Context, id string) error {
	return m.docker.ConfigRemove(ctx, id)
}
