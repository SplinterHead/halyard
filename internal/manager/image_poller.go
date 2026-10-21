package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

type ImagePoller struct {
	docker      *docker.Client
	imgAgg      *ImageAggregator
	svcMgr      *ServiceManager
	contAgg     *ContainerAggregator
	registryMgr *RegistryManager
	settingsMgr *SettingsManager
	db          *DB

	mu         sync.RWMutex
	isPolling  bool
	lastResult api.ImagePollResult
}

func NewImagePoller(
	cli *docker.Client,
	imgAgg *ImageAggregator,
	svcMgr *ServiceManager,
	contAgg *ContainerAggregator,
	registryMgr *RegistryManager,
	settingsMgr *SettingsManager,
	db *DB,
) *ImagePoller {
	return &ImagePoller{
		docker:      cli,
		imgAgg:      imgAgg,
		svcMgr:      svcMgr,
		contAgg:     contAgg,
		registryMgr: registryMgr,
		settingsMgr: settingsMgr,
		db:          db,
		lastResult: api.ImagePollResult{
			Status: "idle",
		},
	}
}

// Start runs the timer loop checking periodically if a poll is due
func (p *ImagePoller) Start(ctx context.Context) {
	log.Printf("ImagePoller: Background service started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastRun time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			settings := p.settingsMgr.GetSettings()
			intervalHours := settings.ImageUpdateInterval
			if intervalHours <= 0 {
				continue
			}

			intervalDuration := time.Duration(intervalHours) * time.Hour
			if lastRun.IsZero() || time.Since(lastRun) >= intervalDuration {
				_, err := p.Poll(ctx)
				if err != nil {
					log.Printf("ImagePoller: Scheduled poll returned error: %v", err)
				}
				lastRun = time.Now()
			}
		}
	}
}

func (p *ImagePoller) GetStatus() api.ImagePollResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastResult
}

type uniqueImageEntry struct {
	Repository   string
	Tag          string
	Architecture string
	Nodes        []string          // list of node IDs
	NodeMap      map[string]string // nodeID -> localImageID
	NodeNames    map[string]string // nodeID -> hostname
}

// Poll inspects unique image+platform pairs in the Swarm, pulls updates, and optionally restarts services
func (p *ImagePoller) Poll(ctx context.Context) (*api.ImagePollResult, error) {
	p.mu.Lock()
	if p.isPolling {
		p.mu.Unlock()
		return nil, fmt.Errorf("an image update poll is already running")
	}
	p.isPolling = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.isPolling = false
		p.mu.Unlock()
	}()

	startTime := time.Now()
	result := api.ImagePollResult{
		CheckedAt:         startTime,
		Status:            "running",
		RestartedServices: make([]string, 0),
		Details:           make([]string, 0),
	}

	log.Println("Triggering global image update poll")

	// 1. Gather all images from all nodes
	allImages, err := p.imgAgg.ListAllImages(ctx)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to list swarm images: %v", err)
		p.saveResult(result)
		log.Printf("ImagePoller: Error listing swarm images: %v", err)
		return &result, err
	}

	// 2. Gather Swarm services, nodes, and running tasks
	services, err := p.svcMgr.ListServices(ctx)
	if err != nil {
		log.Printf("ImagePoller: Warning - failed to list swarm services: %v", err)
		services = []api.Service{}
	}

	nodes, _ := p.docker.NodeList(ctx, types.NodeListOptions{})
	nodeHostnameMap := make(map[string]string)
	nodeArchMap := make(map[string]string)
	allActiveNodeIDs := make([]string, 0)
	for _, n := range nodes {
		nodeHostnameMap[n.ID] = n.Description.Hostname
		arch := n.Description.Platform.Architecture
		if arch == "" {
			arch = "unknown"
		}
		nodeArchMap[n.ID] = arch
		if string(n.Status.State) == "ready" && string(n.Spec.Availability) == "active" {
			allActiveNodeIDs = append(allActiveNodeIDs, n.ID)
		}
	}
	if len(allActiveNodeIDs) == 0 {
		for _, n := range nodes {
			allActiveNodeIDs = append(allActiveNodeIDs, n.ID)
		}
	}

	// Map service ID -> set of node IDs where service tasks run
	tasks, _ := p.docker.TaskList(ctx, types.TaskListOptions{})
	serviceNodesMap := make(map[string]map[string]bool)
	for _, t := range tasks {
		if t.NodeID == "" || t.ServiceID == "" {
			continue
		}
		if t.DesiredState == "running" || t.Status.State == "running" {
			if _, ok := serviceNodesMap[t.ServiceID]; !ok {
				serviceNodesMap[t.ServiceID] = make(map[string]bool)
			}
			serviceNodesMap[t.ServiceID][t.NodeID] = true
		}
	}

	// Index node local image IDs: nodeID -> normalized(repo:tag) -> imageID
	nodeImageMap := make(map[string]map[string]string)
	for _, img := range allImages {
		if img.Repository == "" || img.Repository == "<none>" || img.Tag == "" || img.Tag == "<none>" {
			continue
		}
		r, t := normalizeImageRef(img.Repository + ":" + img.Tag)
		if r == "" || t == "" {
			continue
		}
		if _, ok := nodeImageMap[img.NodeID]; !ok {
			nodeImageMap[img.NodeID] = make(map[string]string)
		}
		nodeImageMap[img.NodeID][r+":"+t] = img.ID
	}

	uniqueImages := make(map[string]*uniqueImageEntry)

	// 3a. Discover images defined across all Swarm Services across all stacks
	for _, svc := range services {
		if svc.Image == "" {
			continue
		}
		repo, tag := normalizeImageRef(svc.Image)
		if repo == "" || repo == "<none>" || tag == "" || tag == "<none>" {
			continue
		}

		targetNodeIDs := make([]string, 0)
		if svc.Mode == "global" {
			targetNodeIDs = allActiveNodeIDs
		} else if nodeSet, ok := serviceNodesMap[svc.ID]; ok && len(nodeSet) > 0 {
			for nID := range nodeSet {
				targetNodeIDs = append(targetNodeIDs, nID)
			}
		} else {
			targetNodeIDs = allActiveNodeIDs
		}

		for _, nodeID := range targetNodeIDs {
			arch := nodeArchMap[nodeID]
			if arch == "" {
				arch = "unknown"
			}
			key := fmt.Sprintf("%s:%s|%s", repo, tag, arch)
			entry, ok := uniqueImages[key]
			if !ok {
				entry = &uniqueImageEntry{
					Repository:   repo,
					Tag:          tag,
					Architecture: arch,
					Nodes:        make([]string, 0),
					NodeMap:      make(map[string]string),
					NodeNames:    make(map[string]string),
				}
				uniqueImages[key] = entry
			}

			if _, exists := entry.NodeNames[nodeID]; !exists {
				entry.Nodes = append(entry.Nodes, nodeID)
				nodeName := nodeHostnameMap[nodeID]
				if nodeName == "" {
					nodeName = nodeID
				}
				entry.NodeNames[nodeID] = nodeName
			}
			if localID, ok := nodeImageMap[nodeID][repo+":"+tag]; ok && localID != "" {
				entry.NodeMap[nodeID] = localID
			}
		}
	}

	// 3b. Also discover any standalone or unmanaged images present on nodes
	for _, img := range allImages {
		if img.Repository == "" || img.Repository == "<none>" || img.Tag == "" || img.Tag == "<none>" {
			continue
		}
		repo, tag := normalizeImageRef(img.Repository + ":" + img.Tag)
		if repo == "" || repo == "<none>" || tag == "" || tag == "<none>" {
			continue
		}

		arch := img.Architecture
		if arch == "" || arch == "unknown" {
			arch = nodeArchMap[img.NodeID]
		}
		if arch == "" {
			arch = "unknown"
		}

		key := fmt.Sprintf("%s:%s|%s", repo, tag, arch)
		entry, ok := uniqueImages[key]
		if !ok {
			entry = &uniqueImageEntry{
				Repository:   repo,
				Tag:          tag,
				Architecture: arch,
				Nodes:        make([]string, 0),
				NodeMap:      make(map[string]string),
				NodeNames:    make(map[string]string),
			}
			uniqueImages[key] = entry
		}

		if _, exists := entry.NodeNames[img.NodeID]; !exists {
			entry.Nodes = append(entry.Nodes, img.NodeID)
			nodeName := img.Node
			if nodeName == "" {
				nodeName = nodeHostnameMap[img.NodeID]
			}
			entry.NodeNames[img.NodeID] = nodeName
		}
		if img.ID != "" {
			entry.NodeMap[img.NodeID] = img.ID
		}
	}

	result.UniqueImagesCount = len(uniqueImages)

	registries, _ := p.registryMgr.ListRegistries()
	settings := p.settingsMgr.GetSettings()

	updatedCount := 0

	// 4. Iterate over each unique image+platform
	for _, entry := range uniqueImages {
		if len(entry.Nodes) == 0 {
			continue
		}

		imageRef := fmt.Sprintf("%s:%s", entry.Repository, entry.Tag)
		auth := GetEncodedRegistryAuth(entry.Repository, registries)

		// Pick the first node for checking remote digest
		checkNodeID := entry.Nodes[0]
		localImageID := entry.NodeMap[checkNodeID]

		upToDate, checkErr := p.imgAgg.CheckImage(ctx, checkNodeID, entry.Repository, entry.Tag, localImageID, auth)
		if checkErr != nil {
			log.Printf("ImagePoller: Check failed for %s (%s): %v", imageRef, entry.Architecture, checkErr)
			result.Details = append(result.Details, fmt.Sprintf("Error checking %s (%s): %v", imageRef, entry.Architecture, checkErr))
			continue
		}

		if upToDate {
			result.Details = append(result.Details, fmt.Sprintf("%s (%s) is up to date", imageRef, entry.Architecture))
			continue
		}

		// An update is available!
		updatedCount++

		// Pull the updated image to all relevant nodes with this platform
		pulledNodes := make([]string, 0)
		for _, nodeID := range entry.Nodes {
			nodeName := entry.NodeNames[nodeID]
			log.Printf("Found new image for %s, pulling to node %s", imageRef, nodeName)
			if pullErr := p.imgAgg.PullImage(ctx, nodeID, entry.Repository, entry.Tag, auth); pullErr != nil {
				errMsg := fmt.Sprintf("Failed pulling %s to node %s: %v", imageRef, nodeName, pullErr)
				log.Printf("ImagePoller: %s", errMsg)
				result.Details = append(result.Details, errMsg)
			} else {
				pulledNodes = append(pulledNodes, nodeName)
			}
		}

		detailEntry := fmt.Sprintf("Pulled %s (%s) to %d nodes: %s", imageRef, entry.Architecture, len(pulledNodes), strings.Join(pulledNodes, ", "))

		// If UpdateStacks is enabled, restart matching services
		if settings.UpdateStacks {
			restartedForImage := p.restartMatchingServices(ctx, entry.Repository, entry.Tag, services, nodeHostnameMap)
			if len(restartedForImage) > 0 {
				result.RestartedServices = append(result.RestartedServices, restartedForImage...)
				detailEntry += fmt.Sprintf(" | Restarted: %s", strings.Join(restartedForImage, ", "))
			}
		}

		result.Details = append(result.Details, detailEntry)

		// Record to DB deployment_history audit trail
		p.recordDeploymentEvent(imageRef, entry.Architecture, detailEntry)
	}

	result.UpdatedImagesCount = updatedCount
	result.Status = "success"
	p.saveResult(result)

	log.Printf("ImagePoller: Poll complete. Checked: %d, Updated: %d, Restarted: %d service(s)", len(uniqueImages), updatedCount, len(result.RestartedServices))

	return &result, nil
}

// restartMatchingServices handles restarting both replicated and global services
func (p *ImagePoller) restartMatchingServices(
	ctx context.Context,
	repo string,
	tag string,
	services []api.Service,
	nodeHostnameMap map[string]string,
) []string {
	restarted := make([]string, 0)
	targetImage := repo + ":" + tag

	for _, svc := range services {
		if !imageMatches(svc.Image, repo, tag) {
			continue
		}

		log.Printf("ImagePoller: Service %s (mode: %s) uses updated image %s:%s, triggering restart...",
			svc.Name, svc.Mode, repo, tag)

		// If the service's image is pinned to an old digest (@sha256:...), update it to the clean repo:tag
		// so Docker Swarm resolves and runs the updated image!
		if strings.Contains(svc.Image, "@sha256:") {
			p.updateServiceImage(ctx, svc.ID, targetImage)
		}

		if svc.Mode == "global" {
			// Global services: kill the service node by node
			p.restartGlobalServiceNodeByNode(ctx, svc, nodeHostnameMap)
			restarted = append(restarted, fmt.Sprintf("%s (global)", svc.Name))
		} else {
			// Replicated services: scale -1 then +1
			p.restartReplicatedService(ctx, svc)
			restarted = append(restarted, fmt.Sprintf("%s (replicated)", svc.Name))
		}
	}

	return restarted
}

func (p *ImagePoller) updateServiceImage(ctx context.Context, serviceID, newImage string) {
	service, _, err := p.docker.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		log.Printf("ImagePoller: Failed to inspect service %s for image update: %v", serviceID, err)
		return
	}
	service.Spec.TaskTemplate.ContainerSpec.Image = newImage
	service.Spec.TaskTemplate.ForceUpdate++
	_, err = p.docker.ServiceUpdate(ctx, serviceID, service.Version, service.Spec, types.ServiceUpdateOptions{
		QueryRegistry: true,
	})
	if err != nil {
		log.Printf("ImagePoller: Failed to update service %s image to %s: %v", serviceID, newImage, err)
	}
}

// restartReplicatedService scales -1 then +1 for replicated services
func (p *ImagePoller) restartReplicatedService(ctx context.Context, svc api.Service) {
	svcDetail, err := p.svcMgr.GetServiceDetail(ctx, svc.ID)
	if err != nil {
		log.Printf("ImagePoller: Failed to inspect replicated service %s: %v", svc.Name, err)
		return
	}

	currentReplicas := svcDetail.Replicas
	if currentReplicas == 0 {
		log.Printf("ImagePoller: Replicated service %s is currently at 0 replicas; skipping restart", svc.Name)
		return
	}

	if currentReplicas == 1 {
		log.Printf("ImagePoller: Scaling service %s (1 -> 0 -> 1)...", svc.Name)
		if err := p.svcMgr.ScaleService(ctx, svc.ID, 0); err != nil {
			log.Printf("ImagePoller: Error scaling down service %s: %v", svc.Name, err)
		}
		time.Sleep(2 * time.Second)
		if err := p.svcMgr.ScaleService(ctx, svc.ID, 1); err != nil {
			log.Printf("ImagePoller: Error scaling up service %s: %v", svc.Name, err)
		}
	} else {
		// Replicas > 1: scale -1 then +1, cycling each replica
		log.Printf("ImagePoller: Scaling service %s (%d -> %d -> %d)...", svc.Name, currentReplicas, currentReplicas-1, currentReplicas)
		for r := uint64(0); r < currentReplicas; r++ {
			if err := p.svcMgr.ScaleService(ctx, svc.ID, currentReplicas-1); err != nil {
				log.Printf("ImagePoller: Error scaling -1 on service %s: %v", svc.Name, err)
				break
			}
			time.Sleep(2 * time.Second)
			if err := p.svcMgr.ScaleService(ctx, svc.ID, currentReplicas); err != nil {
				log.Printf("ImagePoller: Error scaling +1 on service %s: %v", svc.Name, err)
				break
			}
			if r < currentReplicas-1 {
				time.Sleep(2 * time.Second)
			}
		}
	}

	log.Printf("ImagePoller: Successfully restarted replicated service %s", svc.Name)
}

// restartGlobalServiceNodeByNode kills the service node by node
func (p *ImagePoller) restartGlobalServiceNodeByNode(ctx context.Context, svc api.Service, nodeHostnameMap map[string]string) {
	tasks, err := p.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(
			filters.Arg("service", svc.ID),
			filters.Arg("desired-state", "running"),
		),
	})
	if err != nil {
		log.Printf("ImagePoller: Failed to list tasks for global service %s: %v", svc.Name, err)
		return
	}

	log.Printf("ImagePoller: Restarting global service %s across %d tasks node by node...", svc.Name, len(tasks))

	for i, task := range tasks {
		if task.Status.State != "running" {
			continue
		}

		containerID := task.Status.ContainerStatus.ContainerID
		nodeName := nodeHostnameMap[task.NodeID]
		if nodeName == "" {
			nodeName = task.NodeID
		}

		if containerID != "" {
			displayID := containerID
			if len(displayID) > 12 {
				displayID = displayID[:12]
			}
			log.Printf("ImagePoller: [%d/%d] Killing container %s on node %s for global service %s...",
				i+1, len(tasks), displayID, nodeName, svc.Name)

			if err := p.contAgg.DeleteContainer(ctx, containerID, nodeName, true); err != nil {
				log.Printf("ImagePoller: Error killing container %s on node %s: %v", displayID, nodeName, err)
			} else {
				log.Printf("ImagePoller: Killed container %s on node %s (Swarm will recreate with updated image)", displayID, nodeName)
			}

			// Wait between nodes so the global service updates progressively
			time.Sleep(3 * time.Second)
		}
	}

	log.Printf("ImagePoller: Completed node-by-node restart for global service %s", svc.Name)
}

func (p *ImagePoller) saveResult(res api.ImagePollResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastResult = res
}

func (p *ImagePoller) recordDeploymentEvent(imageRef, arch, details string) {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)

	_, err := p.db.Exec(
		"INSERT INTO deployment_history (id, sync_id, sha, status, logs, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		id,
		"image-poller",
		imageRef+" ("+arch+")",
		"Image Updated",
		details,
		time.Now(),
	)
	if err != nil {
		log.Printf("ImagePoller: Failed to save deployment audit event: %v", err)
	}
}

// Helpers for matching image names

func normalizeImageRef(imgRef string) (repo, tag string) {
	if atIdx := strings.Index(imgRef, "@"); atIdx != -1 {
		imgRef = imgRef[:atIdx]
	}
	parts := strings.Split(imgRef, ":")
	if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "/") {
		tag = parts[len(parts)-1]
		repo = strings.Join(parts[:len(parts)-1], ":")
	} else {
		repo = imgRef
		tag = "latest"
	}
	repo = cleanRepoName(repo)
	return repo, tag
}

func cleanRepoName(repo string) string {
	repo = strings.TrimPrefix(repo, "docker.io/library/")
	repo = strings.TrimPrefix(repo, "docker.io/")
	repo = strings.TrimPrefix(repo, "library/")
	return repo
}

func imageMatches(serviceImage, targetRepo, targetTag string) bool {
	sRepo, sTag := normalizeImageRef(serviceImage)
	tRepo, tTag := normalizeImageRef(targetRepo + ":" + targetTag)
	return sRepo == tRepo && sTag == tTag
}
