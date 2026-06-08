package agent

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ContainerManager struct {
	docker      *docker.Client
	digestCache map[string]string
	cacheMu     sync.RWMutex
}

func NewContainerManager(cli *docker.Client) *ContainerManager {
	return &ContainerManager{
		docker:      cli,
		digestCache: make(map[string]string),
	}
}

func (m *ContainerManager) ListContainers(ctx context.Context) ([]api.ContainerInfo, error) {
	resp, err := m.docker.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	containers := make([]api.ContainerInfo, 0, len(resp))
	for _, c := range resp {
		stack := c.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}

		service := c.Labels["com.docker.swarm.service.name"]
		if service == "" {
			service = "-"
		}

		upToDate := m.checkImageUpToDate(ctx, c.Image, c.ImageID)

		var networks []string
		if c.NetworkSettings != nil && c.NetworkSettings.Networks != nil {
			for netName := range c.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}

		containers = append(containers, api.ContainerInfo{
			ID:        c.ID,
			Names:     c.Names,
			Image:     c.Image,
			State:     c.State,
			Status:    c.Status,
			Service:   service,
			Stack:     stack,
			Networks:  networks,
			UpToDate:  upToDate,
			CreatedAt: time.Unix(c.Created, 0),
		})
	}

	return containers, nil
}

func (m *ContainerManager) checkImageUpToDate(ctx context.Context, imageTag string, localImageID string) bool {
	// Skip if it's a SHA directly or very short
	if strings.Contains(imageTag, "sha256:") || len(imageTag) < 3 {
		return true
	}

	// 1. Get remote digest (with simple caching)
	m.cacheMu.RLock()
	remoteDigest, ok := m.digestCache[imageTag]
	m.cacheMu.RUnlock()

	if !ok {
		distribution, err := m.docker.DistributionInspect(ctx, imageTag, "")
		if err != nil {
			// If we can't inspect (e.g. auth), assume up to date to avoid false positives
			return true
		}
		remoteDigest = string(distribution.Descriptor.Digest)
		m.cacheMu.Lock()
		m.digestCache[imageTag] = remoteDigest
		// Basic cache expiration could be added here, but for now we'll keep it simple
		m.cacheMu.Unlock()
	}

	// 2. Local Image Inspect to get RepoDigests
	img, _, err := m.docker.ImageInspectWithRaw(ctx, localImageID)
	if err != nil {
		return true
	}

	// Compare remoteDigest with local RepoDigests
	for _, d := range img.RepoDigests {
		if strings.Contains(d, remoteDigest) {
			return true
		}
	}

	return false
}

func (m *ContainerManager) GetContainerDetail(ctx context.Context, id string) (api.ContainerDetail, error) {
	c, err := m.docker.ContainerInspect(ctx, id)
	if err != nil {
		return api.ContainerDetail{}, err
	}

	stack := c.Config.Labels["com.docker.stack.namespace"]
	if stack == "" {
		stack = "-"
	}

	service := c.Config.Labels["com.docker.swarm.service.name"]
	if service == "" {
		service = "-"
	}

	upToDate := m.checkImageUpToDate(ctx, c.Config.Image, c.Image)

	createdAt, _ := time.Parse(time.RFC3339Nano, c.Created)

	detail := api.ContainerDetail{
		ContainerInfo: api.ContainerInfo{
			ID:        c.ID,
			Names:     []string{c.Name},
			Image:     c.Config.Image,
			ImageID:   c.Image,
			State:     c.State.Status,
			Status:    c.State.Status,
			Service:   service,
			Stack:     stack,
			UpToDate:  upToDate,
			CreatedAt: createdAt,
		},
		Env:     c.Config.Env,
		Labels:  c.Config.Labels,
		Network: make([]string, 0),
		Mounts:  make([]api.Mount, 0),
		Ports:   make([]api.Port, 0),
	}

	for netName := range c.NetworkSettings.Networks {
		detail.Network = append(detail.Network, netName)
	}

	for _, m := range c.Mounts {
		detail.Mounts = append(detail.Mounts, api.Mount{
			Source:      m.Source,
			Destination: m.Destination,
			Type:        string(m.Type),
			RW:          m.RW,
		})
	}

	for p, bindings := range c.NetworkSettings.Ports {
		for _, b := range bindings {
			detail.Ports = append(detail.Ports, api.Port{
				IP:          b.HostIP,
				PrivatePort: uint16(p.Int()),
				PublicPort:  uint16(docker.ParsePort(b.HostPort)),
				Type:        p.Proto(),
			})
		}
	}

	return detail, nil
}

func (m *ContainerManager) StreamLogs(ctx context.Context, id string) (io.ReadCloser, error) {
	return m.docker.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
		Timestamps: true,
	})
}

func (m *ContainerManager) StopContainer(ctx context.Context, id string) error {
	return m.docker.ContainerStop(ctx, id, container.StopOptions{})
}

func (m *ContainerManager) StartContainer(ctx context.Context, id string) error {
	return m.docker.ContainerStart(ctx, id, container.StartOptions{})
}

func (m *ContainerManager) RestartContainer(ctx context.Context, id string) error {
	return m.docker.ContainerRestart(ctx, id, container.StopOptions{})
}

func (m *ContainerManager) RemoveContainer(ctx context.Context, id string, force bool) error {
	return m.docker.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

func (m *ContainerManager) ExecContainerWS(ctx context.Context, containerID string, shell string, wsConn *websocket.Conn) error {
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{shell},
	}

	execIDResp, err := m.docker.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return err
	}

	resp, err := m.docker.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		return err
	}
	defer resp.Close()

	errChan := make(chan error, 2)

	// Read from container stdout/stderr, write to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Reader.Read(buf)
			if n > 0 {
				if wErr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
					errChan <- wErr
					return
				}
			}
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Read from WebSocket (stdin & resize), write to container
	go func() {
		for {
			mt, message, err := wsConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if mt == websocket.TextMessage {
				var cmd struct {
					Type string `json:"type"`
					Data string `json:"data"`
					Cols uint   `json:"cols"`
					Rows uint   `json:"rows"`
				}
				if json.Unmarshal(message, &cmd) == nil {
					if cmd.Type == "resize" && cmd.Cols > 0 && cmd.Rows > 0 {
						m.docker.ContainerExecResize(ctx, execIDResp.ID, container.ResizeOptions{
							Height: cmd.Rows,
							Width:  cmd.Cols,
						})
						continue
					} else if cmd.Type == "stdin" {
						message = []byte(cmd.Data)
					}
				}
			}

			if _, wErr := resp.Conn.Write(message); wErr != nil {
				errChan <- wErr
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil && err != io.EOF && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return err
		}
		return nil
	}
}
