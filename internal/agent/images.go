package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ImageManager struct {
	docker *docker.Client
}

func NewImageManager(cli *docker.Client) *ImageManager {
	return &ImageManager{docker: cli}
}

func parseRepoTag(ref string) (repo, tag string) {
	if atIdx := strings.Index(ref, "@"); atIdx != -1 {
		ref = ref[:atIdx]
	}
	lastColon := strings.LastIndex(ref, ":")
	lastSlash := strings.LastIndex(ref, "/")
	if lastColon > lastSlash && lastColon != -1 {
		return ref[:lastColon], ref[lastColon+1:]
	}
	return ref, "latest"
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

	// 3. Get in-use image IDs and image strings from all local containers
	containers, _ := m.docker.ContainerList(ctx, container.ListOptions{All: true})

	return buildImageInfos(list, containers, arch), nil
}

type imageKey struct {
	id   string
	repo string
	tag  string
}

func buildImageInfos(list []image.Summary, containers []types.Container, arch string) []api.ImageInfo {
	inUseMap := make(map[string]bool)
	containerImageMap := make(map[string]string)
	for _, c := range containers {
		inUseMap[c.ImageID] = true
		if c.Image != "" {
			containerImageMap[c.ImageID] = c.Image
		}
	}

	seen := make(map[imageKey]bool)
	images := make([]api.ImageInfo, 0, len(list))
	appendUnique := func(info api.ImageInfo) {
		key := imageKey{id: info.ID, repo: info.Repository, tag: info.Tag}
		if seen[key] {
			return
		}
		seen[key] = true
		images = append(images, info)
	}

	for _, img := range list {
		inUse := inUseMap[img.ID]

		// Resolve tags for this image
		tags := make([]string, 0)
		for _, rt := range img.RepoTags {
			if rt != "" && rt != "<none>:<none>" {
				tags = append(tags, rt)
			}
		}

		// If no valid RepoTags (common for Swarm digest-pulled images), fallback to container image or RepoDigests
		if len(tags) == 0 {
			if cImg, ok := containerImageMap[img.ID]; ok && cImg != "" {
				tags = append(tags, cImg)
			} else if len(img.RepoDigests) > 0 && img.RepoDigests[0] != "" {
				tags = append(tags, img.RepoDigests[0])
			}
		}

		// If still completely untagged
		if len(tags) == 0 {
			appendUnique(api.ImageInfo{
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

		// List an entry for each unique (ID, repo, tag) associated with the image
		for _, repoTag := range tags {
			repo, tag := parseRepoTag(repoTag)
			if (repo == "" || repo == "<none>") && containerImageMap[img.ID] != "" {
				repo, tag = parseRepoTag(containerImageMap[img.ID])
			}

			appendUnique(api.ImageInfo{
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

	return images
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

	target := localImageID
	if target == "" {
		target = imageTag
	}

	img, _, err := m.docker.ImageInspectWithRaw(ctx, target)
	if err != nil {
		// Image is not locally present on this node; pull required
		return false, nil
	}

	for _, d := range img.RepoDigests {
		if strings.Contains(d, remoteDigest) {
			return true, nil
		}
	}
	if strings.Contains(img.ID, remoteDigest) {
		return true, nil
	}

	return false, nil
}

func (m *ImageManager) PullImage(ctx context.Context, repository string, tag string, encodedAuth string) error {
	imageTag := repository + ":" + tag
	reader, err := m.docker.ImagePull(ctx, imageTag, image.PullOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Consume entire response stream to ensure pull completes
	_, err = io.Copy(io.Discard, reader)
	return err
}

