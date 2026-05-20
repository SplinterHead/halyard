package manager

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ServiceManager struct {
	docker *docker.Client
}

func NewServiceManager(cli *docker.Client) *ServiceManager {
	return &ServiceManager{docker: cli}
}

func (m *ServiceManager) ListServices(ctx context.Context) ([]api.Service, error) {
	services, err := m.docker.ServiceList(ctx, types.ServiceListOptions{Status: true})
	if err != nil {
		return nil, err
	}

	result := make([]api.Service, 0, len(services))
	for _, s := range services {
		stack := s.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}

		mode := "replicated"
		replicas := uint64(0)
		if s.Spec.Mode.Global != nil {
			mode = "global"
		} else if s.Spec.Mode.Replicated != nil && s.Spec.Mode.Replicated.Replicas != nil {
			replicas = *s.Spec.Mode.Replicated.Replicas
		}

		running := uint64(0)
		if s.ServiceStatus != nil {
			running = s.ServiceStatus.RunningTasks
			if mode == "global" {
				replicas = s.ServiceStatus.DesiredTasks
			} else if replicas == 0 {
				// Fallback for replicated services if spec is missing but status is present
				replicas = s.ServiceStatus.DesiredTasks
			}
		}

		ports := make([]string, 0)
		if s.Endpoint.Ports != nil {
			for _, p := range s.Endpoint.Ports {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublishedPort, p.TargetPort, p.Protocol))
			}
		}

		result = append(result, api.Service{
			ID:        s.ID,
			Name:      s.Spec.Name,
			Stack:     stack,
			Image:     s.Spec.TaskTemplate.ContainerSpec.Image,
			Mode:      mode,
			Replicas:  replicas,
			Running:   running,
			Ports:     ports,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}

	return result, nil
}
func (m *ServiceManager) GetServiceDetail(ctx context.Context, id string) (*api.ServiceDetail, error) {
	s, _, err := m.docker.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		return nil, err
	}

	stack := s.Spec.Labels["com.docker.stack.namespace"]
	if stack == "" {
		stack = "-"
	}

	mode := "replicated"
	replicas := uint64(0)
	if s.Spec.Mode.Global != nil {
		mode = "global"
	} else if s.Spec.Mode.Replicated != nil && s.Spec.Mode.Replicated.Replicas != nil {
		replicas = *s.Spec.Mode.Replicated.Replicas
	}

	ports := make([]string, 0)
	if s.Endpoint.Ports != nil {
		for _, p := range s.Endpoint.Ports {
			ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublishedPort, p.TargetPort, p.Protocol))
		}
	}

	restartPolicy := "none"
	if s.Spec.TaskTemplate.RestartPolicy != nil {
		restartPolicy = string(s.Spec.TaskTemplate.RestartPolicy.Condition)
	}

	constraints := make([]string, 0)
	if s.Spec.TaskTemplate.Placement != nil {
		constraints = s.Spec.TaskTemplate.Placement.Constraints
	}

	// Get status/running count using ServiceList with filter
	services, err := m.docker.ServiceList(ctx, types.ServiceListOptions{
		Filters: docker.BuildFilters(map[string]string{"id": id}),
		Status:  true,
	})
	running := uint64(0)
	if err == nil && len(services) > 0 {
		if services[0].ServiceStatus != nil {
			running = services[0].ServiceStatus.RunningTasks
			if mode == "global" {
				replicas = services[0].ServiceStatus.DesiredTasks
			}
		}
	}

	return &api.ServiceDetail{
		Service: api.Service{
			ID:        s.ID,
			Name:      s.Spec.Name,
			Stack:     stack,
			Image:     s.Spec.TaskTemplate.ContainerSpec.Image,
			Mode:      mode,
			Replicas:  replicas,
			Running:   running,
			Ports:     ports,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		Labels:        s.Spec.Labels,
		RestartPolicy: restartPolicy,
		Env:           s.Spec.TaskTemplate.ContainerSpec.Env,
		Constraints:   constraints,
	}, nil
}

func (m *ServiceManager) RemoveService(ctx context.Context, id string) error {
	return m.docker.ServiceRemove(ctx, id)
}

