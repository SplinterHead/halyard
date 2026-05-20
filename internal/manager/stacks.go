package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type StackManager struct {
	docker *docker.Client
}

func NewStackManager(cli *docker.Client) *StackManager {
	return &StackManager{docker: cli}
}

func (m *StackManager) ListStacks(ctx context.Context) ([]api.Stack, error) {
	services, err := m.docker.ServiceList(ctx, types.ServiceListOptions{Status: true})
	if err != nil {
		return nil, err
	}

	type stackInfo struct {
		services int
		running  uint64
		total    uint64
		updated  time.Time
	}

	stackMap := make(map[string]*stackInfo)
	for _, service := range services {
		stackName := service.Spec.Labels["com.docker.stack.namespace"]
		if stackName == "" {
			stackName = "Orphaned/Manual"
		}

		if _, ok := stackMap[stackName]; !ok {
			stackMap[stackName] = &stackInfo{}
		}

		info := stackMap[stackName]
		info.services++

		if service.ServiceStatus != nil {
			info.running += service.ServiceStatus.RunningTasks
			
			// For replicated services, use Spec replicas. For global, use DesiredTasks.
			if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
				info.total += *service.Spec.Mode.Replicated.Replicas
			} else {
				info.total += service.ServiceStatus.DesiredTasks
			}
		}

		if service.UpdatedAt.After(info.updated) {
			info.updated = service.UpdatedAt
		}
	}

	stacks := make([]api.Stack, 0, len(stackMap))
	for name, info := range stackMap {
		status := "Healthy"
		if info.running < info.total {
			status = "Degraded"
		} else if info.total == 0 && info.services > 0 {
			// Might be global services or still scaling
			status = "Updating"
		}

		stacks = append(stacks, api.Stack{
			Name:            name,
			Services:        info.services,
			RunningReplicas: info.running,
			TotalReplicas:   info.total,
			Status:          status,
			UpdatedAt:       info.updated,
		})
	}

	return stacks, nil
}

func (m *StackManager) Exists(ctx context.Context, name string) (bool, error) {
	stacks, err := m.ListStacks(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range stacks {
		if s.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (m *StackManager) DeployStack(ctx context.Context, name string, workingDir string, composePath string, registries []api.Registry) error {
	// Prepare Docker config if registries are provided
	var configDir string
	if len(registries) > 0 {
		var err error
		configDir, err = m.prepareDockerConfig(registries)
		if err != nil {
			return fmt.Errorf("failed to prepare docker config: %w", err)
		}
		defer os.RemoveAll(configDir)
	}

	args := []string{"stack", "deploy", "-c", composePath, name, "--with-registry-auth"}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = workingDir
	
	if configDir != "" {
		cmd.Env = append(os.Environ(), "DOCKER_CONFIG="+configDir)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("deploy failed: %v, output: %s", err, string(output))
	}
	return nil
}

func (m *StackManager) DeployStackContent(ctx context.Context, name string, composeContent string, registries []api.Registry) error {
	tmpFile, err := os.CreateTemp("", "halyard-stack-*.yml")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(composeContent); err != nil {
		return err
	}
	tmpFile.Close()

	return m.DeployStack(ctx, name, "", tmpFile.Name(), registries)
}

func (m *StackManager) prepareDockerConfig(registries []api.Registry) (string, error) {
	configDir, err := os.MkdirTemp("", "docker-config-*")
	if err != nil {
		return "", err
	}

	type authEntry struct {
		Auth string `json:"auth"`
	}
	type dockerConfig struct {
		Auths map[string]authEntry `json:"auths"`
	}

	config := dockerConfig{
		Auths: make(map[string]authEntry),
	}

	for _, r := range registries {
		auth := fmt.Sprintf("%s:%s", r.Username, r.Password)
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		entry := authEntry{Auth: encoded}

		// Helper to check if a URL represents Docker Hub
		isDockerHub := r.URL == "" || 
			strings.Contains(strings.ToLower(r.URL), "docker.io") || 
			strings.Contains(strings.ToLower(r.Name), "docker hub")

		if isDockerHub {
			// Populate all standard Docker Hub alias keys in config.json
			config.Auths["https://index.docker.io/v1/"] = entry
			config.Auths["registry-1.docker.io"] = entry
			config.Auths["docker.io"] = entry
			config.Auths["index.docker.io"] = entry
		} else {
			// Write the user-specified URL
			config.Auths[r.URL] = entry

			// Proactively generate standard normalized variations of the URL
			host := r.URL
			host = strings.TrimPrefix(host, "https://")
			host = strings.TrimPrefix(host, "http://")
			host = strings.Split(host, "/")[0] // Extract host domain (e.g. ghcr.io)

			if host != "" {
				config.Auths[host] = entry
				config.Auths["https://"+host] = entry
				config.Auths["https://"+host+"/"] = entry
				config.Auths["https://"+host+"/v1/"] = entry
				config.Auths["https://"+host+"/v2/"] = entry
			}
		}
	}

	configPath := filepath.Join(configDir, "config.json")
	f, err := os.Create(configPath)
	if err != nil {
		os.RemoveAll(configDir)
		return "", err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(config); err != nil {
		os.RemoveAll(configDir)
		return "", err
	}

	return configDir, nil
}

func (m *StackManager) RemoveStack(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "stack", "rm", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove failed: %v, output: %s", err, string(output))
	}
	return nil
}
func (m *StackManager) GetStackDetail(ctx context.Context, name string, svcMgr *ServiceManager, contAgg *ContainerAggregator, volAgg *VolumeAggregator) (*api.StackDetail, error) {
	stacks, err := m.ListStacks(ctx)
	if err != nil {
		return nil, err
	}

	var target api.Stack
	found := false
	for _, s := range stacks {
		if s.Name == name {
			target = s
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("stack %s not found", name)
	}

	// Get Services
	allServices, err := svcMgr.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	stackServices := make([]api.Service, 0)
	for _, s := range allServices {
		if s.Stack == name {
			stackServices = append(stackServices, s)
		}
	}

	// Get Containers
	allContainers, err := contAgg.ListAllContainers(ctx)
	if err != nil {
		return nil, err
	}
	stackContainers := make([]api.ContainerInfo, 0)
	for _, c := range allContainers {
		if c.Stack == name {
			stackContainers = append(stackContainers, c)
		}
	}

	// Get Volumes
	allVolumes, err := volAgg.ListAllVolumes(ctx)
	if err != nil {
		return nil, err
	}
	stackVolumes := make([]api.VolumeDetail, 0)
	seenVols := make(map[string]bool)
	for _, v := range allVolumes {
		if v.Stack == name || v.Labels["com.docker.stack.namespace"] == name {
			if seenVols[v.Name] {
				continue
			}
			seenVols[v.Name] = true
			
			// If it has the stack label, it was probably created by the stack
			// External volumes usually don't have it or have a different one
			isExternal := v.Labels["com.docker.stack.namespace"] == ""
			
			stackVolumes = append(stackVolumes, api.VolumeDetail{
				Name:     v.Name,
				External: isExternal,
			})
		}
	}

	// Get Tasks
	tasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "com.docker.stack.namespace="+name)),
	})
	if err != nil {
		return nil, err
	}

	// Get Nodes for mapping
	nodes, _ := m.docker.NodeList(ctx, types.NodeListOptions{})
	nodeMap := make(map[string]string)
	for _, node := range nodes {
		nodeMap[node.ID] = node.Description.Hostname
	}

	// Service name mapping
	svcMap := make(map[string]string)
	for _, svc := range stackServices {
		svcMap[svc.ID] = svc.Name
	}

	stackTasks := make([]api.TaskInfo, 0)
	for _, t := range tasks {
		stackTasks = append(stackTasks, api.TaskInfo{
			ID:           t.ID,
			ServiceID:    t.ServiceID,
			ServiceName:  svcMap[t.ServiceID],
			NodeID:       t.NodeID,
			NodeName:     nodeMap[t.NodeID],
			State:        string(t.Status.State),
			DesiredState: string(t.DesiredState),
			Error:        t.Status.Err,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}

	return &api.StackDetail{
		Stack:      target,
		Services:   stackServices,
		Containers: stackContainers,
		Volumes:    stackVolumes,
		Tasks:      stackTasks,
	}, nil
}
