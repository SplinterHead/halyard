package docker

import (
	"context"
	"io"
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
		if _, err := c.VolumesPrune(ctx, filters.Args{}); err != nil {
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
