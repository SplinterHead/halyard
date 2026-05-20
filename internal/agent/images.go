package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
)

type ImageManager struct {
	docker *docker.Client
}

func NewImageManager(cli *docker.Client) *ImageManager {
	return &ImageManager{docker: cli}
}

func (m *ImageManager) ListImages(ctx context.Context) ([]api.ImageInfo, error) {
	// 1. Get host system architecture from docker daemon Info
	info, err := m.docker.Info(ctx)
	arch := "unknown"
	if err == nil {
		arch = info.Architecture
	}

	// 2. List all local images
	list, err := m.docker.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return nil, err
	}

	// 3. Get in-use image IDs from all local containers
	containers, _ := m.docker.ContainerList(ctx, container.ListOptions{All: true})
	inUseMap := make(map[string]bool)
	for _, c := range containers {
		inUseMap[c.ImageID] = true
	}

	images := make([]api.ImageInfo, 0)
	for _, img := range list {
		inUse := inUseMap[img.ID]

		// Handle untagged images (<none>:<none>)
		if len(img.RepoTags) == 0 {
			images = append(images, api.ImageInfo{
				ID:           img.ID,
				Repository:   "<none>",
				Tag:          "<none>",
				Size:         img.Size,
				Architecture: arch,
				InUse:        inUse,
				CreatedAt:    time.Unix(img.Created, 0),
			})
			continue
		}

		// List an entry for each tag associated with the image
		for _, repoTag := range img.RepoTags {
			repo := repoTag
			tag := "latest"
			if parts := strings.SplitN(repoTag, ":", 2); len(parts) == 2 {
				repo = parts[0]
				tag = parts[1]
			}

			images = append(images, api.ImageInfo{
				ID:           img.ID,
				Repository:   repo,
				Tag:          tag,
				Size:         img.Size,
				Architecture: arch,
				InUse:        inUse,
				CreatedAt:    time.Unix(img.Created, 0),
			})
		}
	}

	return images, nil
}

func (m *ImageManager) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := m.docker.ImageRemove(ctx, id, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

func (m *ImageManager) CheckImageUpToDate(ctx context.Context, repository string, tag string, localImageID string, encodedAuth string) (bool, error) {
	imageTag := repository + ":" + tag
	if repository == "<none>" || tag == "<none>" || strings.Contains(imageTag, "sha256:") {
		return true, nil
	}

	distribution, err := m.docker.DistributionInspect(ctx, imageTag, encodedAuth)
	if err != nil {
		return false, err
	}
	remoteDigest := string(distribution.Descriptor.Digest)
	if remoteDigest == "" {
		return false, fmt.Errorf("remote registry returned an empty digest")
	}

	img, _, err := m.docker.ImageInspectWithRaw(ctx, localImageID)
	if err != nil {
		return false, err
	}

	for _, d := range img.RepoDigests {
		if strings.Contains(d, remoteDigest) {
			return true, nil
		}
	}

	return false, nil
}

