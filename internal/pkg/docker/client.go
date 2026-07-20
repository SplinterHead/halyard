package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Client wraps the Docker SDK client to provide Swarm-specific functionality
type Client struct {
	*client.Client
}

// NewClient initializes a new Docker client connected to the local socket
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli}, nil
}

// CheckSwarm checks if the current node is part of a swarm
func (c *Client) CheckSwarm(ctx context.Context) (bool, error) {
	info, err := c.Info(ctx)
	if err != nil {
		return false, err
	}
	return info.Swarm.LocalNodeState == "active", nil
}

func (c *Client) Prune(ctx context.Context, containers, networks, volumes, images, imagesAll bool) error {
	if containers {
		if _, err := c.ContainersPrune(ctx, filters.Args{}); err != nil {
			return err
		}
	}
	if images {
		args := filters.NewArgs()
		if imagesAll {
			args.Add("dangling", "false")
		} else {
			args.Add("dangling", "true")
		}
		if _, err := c.ImagesPrune(ctx, args); err != nil {
			return err
		}
	}
	if networks {
		if _, err := c.NetworksPrune(ctx, filters.Args{}); err != nil {
			return err
		}
	}
	if volumes {
		args := filters.NewArgs()
		args.Add("all", "1")
		if _, err := c.VolumesPrune(ctx, args); err != nil {
			return err
		}
	}
	return nil
}

func ParsePort(portStr string) int {
	var port int
	for _, char := range portStr {
		if char >= '0' && char <= '9' {
			port = port*10 + int(char-'0')
		}
	}
	return port
}

func DemuxLogs(src io.Reader, dst io.Writer) error {
	_, err := stdcopy.StdCopy(dst, dst, src)
	return err
}

func BuildFilters(m map[string]string) filters.Args {
	args := filters.NewArgs()
	for k, v := range m {
		args.Add(k, v)
	}
	return args
}

// RunHostCommand creates an ephemeral privileged container to execute a command on the host.
func (c *Client) RunHostCommand(ctx context.Context, command string) (string, error) {
	imageName := os.Getenv("AGENT_IMAGE")
	if imageName == "" {
		// Dynamically discover the agent's image by finding a running agent container
		containers, err := c.ContainerList(ctx, container.ListOptions{})
		if err == nil {
			for _, cnt := range containers {
				if strings.Contains(cnt.Image, "halyard-agent") {
					imageName = cnt.Image
					break
				}
			}
		}
		if imageName == "" {
			imageName = "halyard-agent:latest"
		}
	}

	resp, err := c.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   []string{"nsenter", "-t", "1", "-m", "-u", "-n", "-i", "sh", "-c", command},
		Tty:   false,
	}, &container.HostConfig{
		Privileged: true,
		PidMode:    "host",
	}, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := c.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", err
	}

	statusCh, errCh := c.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return "", err
		}
	case status := <-statusCh:
		if status.Error != nil {
			c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
			return "", fmt.Errorf(status.Error.Message)
		}
		exitCode = status.StatusCode
	}

	out, err := c.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", err
	}
	defer out.Close()

	buf := new(bytes.Buffer)
	stdcopy.StdCopy(buf, buf, out) // demux docker logs

	c.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	if exitCode != 0 {
		return buf.String(), fmt.Errorf("exit code %d", exitCode)
	}

	return buf.String(), nil
}
