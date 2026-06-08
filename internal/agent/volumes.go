package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type VolumeManager struct {
	docker *docker.Client
}

func NewVolumeManager(cli *docker.Client) *VolumeManager {
	return &VolumeManager{docker: cli}
}

func (m *VolumeManager) ListVolumes(ctx context.Context) ([]api.VolumeInfo, error) {
	resp, err := m.docker.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get local containers to check for in-use volumes
	containers, _ := m.docker.ContainerList(ctx, container.ListOptions{All: true})
	inUseMap := make(map[string]bool)
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == "volume" {
				inUseMap[m.Name] = true
			}
		}
	}

	volumes := make([]api.VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		stack := v.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}

		volType := "local"
		if v.Driver != "local" {
			volType = v.Driver
		} else if v.Options != nil {
			if t, ok := v.Options["type"]; ok && t != "" {
				volType = t
			} else if o, ok := v.Options["o"]; ok && strings.Contains(o, "bind") {
				volType = "bind"
			}
		}

		if volType == "local" && strings.Contains(v.Mountpoint, "nfs") {
			volType = "nfs"
		}

		var createdAt time.Time
		if v.CreatedAt != "" {
			createdAt, _ = time.Parse(time.RFC3339, v.CreatedAt)
		}

		labels := v.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		volumes = append(volumes, api.VolumeInfo{
			Name:       v.Name,
			Stack:      stack,
			Driver:     v.Driver,
			Type:       volType,
			Mountpoint: v.Mountpoint,
			CreatedAt:  createdAt,
			InUse:      inUseMap[v.Name],
			Labels:     labels,
		})
	}

	return volumes, nil
}

func (m *VolumeManager) PruneVolumes(ctx context.Context) (types.VolumesPruneReport, error) {
	pruneFilters := filters.NewArgs()
	pruneFilters.Add("all", "1")
	return m.docker.VolumesPrune(ctx, pruneFilters)
}

func (m *VolumeManager) RemoveVolume(ctx context.Context, name string, force bool) error {
	return m.docker.VolumeRemove(ctx, name, force)
}

func (m *VolumeManager) BrowseVolume(ctx context.Context, volumeName string, subPath string) ([]api.VolumeFileEntry, error) {
	vol, err := m.docker.VolumeInspect(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}

	mountpoint := vol.Mountpoint
	if mountpoint == "" {
		return nil, fmt.Errorf("volume %s has no mountpoint", volumeName)
	}

	targetPath := filepath.Clean(filepath.Join(mountpoint, subPath))
	// Basic directory traversal prevention
	if !strings.HasPrefix(targetPath, filepath.Clean(mountpoint)) {
		return nil, fmt.Errorf("invalid path: escapes volume mountpoint")
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var fileEntries []api.VolumeFileEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		fileEntries = append(fileEntries, api.VolumeFileEntry{
			Name:       entry.Name(),
			Path:       filepath.ToSlash(filepath.Join(subPath, entry.Name())),
			IsDir:      entry.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			CreatedAt:  info.ModTime(), // Fallback as Go doesn't easily expose birthtime across all Linux FS
			Permission: info.Mode().String(),
		})
	}

	return fileEntries, nil
}

