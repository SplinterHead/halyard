package manager

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type SecretManager struct {
	docker *docker.Client
}

func NewSecretManager(cli *docker.Client) *SecretManager {
	return &SecretManager{docker: cli}
}

func (m *SecretManager) ListSecrets(ctx context.Context) ([]api.SecretInfo, error) {
	secrets, err := m.docker.SecretList(ctx, types.SecretListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]api.SecretInfo, 0, len(secrets))
	for _, s := range secrets {
		result = append(result, api.SecretInfo{
			ID:        s.ID,
			Name:      s.Spec.Name,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			Labels:    s.Spec.Labels,
		})
	}
	return result, nil
}

func (m *SecretManager) CreateSecret(ctx context.Context, name string, data []byte) (string, error) {
	resp, err := m.docker.SecretCreate(ctx, swarm.SecretSpec{
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

func (m *SecretManager) RemoveSecret(ctx context.Context, id string) error {
	return m.docker.SecretRemove(ctx, id)
}
