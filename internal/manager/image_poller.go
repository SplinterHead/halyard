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
	log.Printf("[Image Poller] Background service started")
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
					log.Printf("[Image Poller] Scheduled poll returned error: %v", err)
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
		return nil, fmt.Errorf("[Image Poller] an image update poll is already running")
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

	deferredSelfRestarts := make([]deferredSelfRestart, 0)

	log.Println("[Image Poller] Triggering global image update poll")

	// 1. Gather all images from all nodes
	allImages, err := p.imgAgg.ListAllImages(ctx)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to list swarm images: %v", err)
		p.saveResult(result)
		log.Printf("[Image Poller] Error listing swarm images: %v", err)
		return &result, err
	}

	// 2. Gather Swarm services, nodes, and running tasks
	services, err := p.svcMgr.ListServices(ctx)
	if err != nil {
		log.Printf("[Image Poller] Warning - failed to list swarm services: %v", err)
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
			log.Printf("[Image Poller] Check failed for %s (%s): %v", imageRef, entry.Architecture, checkErr)
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
			log.Printf("[Image Poller] Found new image for %s, pulling to node %s", imageRef, nodeName)
			if pullErr := p.imgAgg.PullImage(ctx, nodeID, entry.Repository, entry.Tag, auth); pullErr != nil {
				errMsg := fmt.Sprintf("Failed pulling %s to node %s: %v", imageRef, nodeName, pullErr)
				log.Printf("[Image Poller] %s", errMsg)
				result.Details = append(result.Details, errMsg)
			} else {
				pulledNodes = append(pulledNodes, nodeName)
			}
		}

		detailEntry := fmt.Sprintf("Pulled %s (%s) to %d nodes: %s", imageRef, entry.Architecture, len(pulledNodes), strings.Join(pulledNodes, ", "))

		// If UpdateStacks is enabled, restart matching services
		if settings.UpdateStacks {
			restartedForImage, deferred := p.restartMatchingServices(ctx, entry.Repository, entry.Tag, services, auth)
			if len(restartedForImage) > 0 {
				result.RestartedServices = append(result.RestartedServices, restartedForImage...)
				detailEntry += fmt.Sprintf(" | Restarted: %s", strings.Join(restartedForImage, ", "))
			}
			deferredSelfRestarts = append(deferredSelfRestarts, deferred...)
		}

		result.Details = append(result.Details, detailEntry)

		// Record to DB deployment_history audit trail
		p.recordDeploymentEvent(imageRef, entry.Architecture, detailEntry)
	}

	result.UpdatedImagesCount = updatedCount
	result.Status = "success"
	p.saveResult(result)

	log.Printf("[Image Poller] Poll complete. Checked: %d, Updated: %d, Restarted: %d service(s)", len(uniqueImages), updatedCount, len(result.RestartedServices))

	// Trigger any deferred self-restarts (e.g. halyard_manager) last so the manager process is not killed midway through polling other stacks
	for _, def := range deferredSelfRestarts {
		log.Printf("[Image Poller] Triggering final deferred restart for self-service %s to %s:%s...", def.svc.Name, def.repo, def.tag)
		_ = p.restartServiceWithImage(ctx, def.svc, def.repo, def.tag, def.auth)
	}

	return &result, nil
}

type deferredSelfRestart struct {
	svc  api.Service
	repo string
	tag  string
	auth string
}

func isSelfService(svc api.Service) bool {
	name := strings.ToLower(svc.Name)
	return name == "halyard_manager" || name == "halyard-manager" || (strings.Contains(name, "halyard") && strings.Contains(name, "manager"))
}

// restartMatchingServices handles triggering Docker Swarm's rolling update for all matching services
func (p *ImagePoller) restartMatchingServices(
	ctx context.Context,
	repo string,
	tag string,
	services []api.Service,
	auth string,
) ([]string, []deferredSelfRestart) {
	restarted := make([]string, 0)
	deferred := make([]deferredSelfRestart, 0)

	for _, svc := range services {
		if !imageMatches(svc.Image, repo, tag) {
			continue
		}

		// Defer restarting halyard_manager itself until all other services finish
		if isSelfService(svc) {
			log.Printf("[Image Poller] Deferring restart of self-service %s until all other stack services finish updating", svc.Name)
			deferred = append(deferred, deferredSelfRestart{svc: svc, repo: repo, tag: tag, auth: auth})
			restarted = append(restarted, fmt.Sprintf("%s (%s - deferred)", svc.Name, svc.Mode))
			continue
		}

		log.Printf("[Image Poller] Service %s (mode: %s) uses updated image %s:%s, triggering rolling restart...",
			svc.Name, svc.Mode, repo, tag)

		if err := p.restartServiceWithImage(ctx, svc, repo, tag, auth); err != nil {
			log.Printf("[Image Poller] Failed rolling update for service %s: %v", svc.Name, err)
			continue
		}

		restarted = append(restarted, fmt.Sprintf("%s (%s)", svc.Name, svc.Mode))
	}

	return restarted, deferred
}

// restartServiceWithImage performs a zero-downtime rolling update on a Swarm service (replicated or global)
func (p *ImagePoller) restartServiceWithImage(ctx context.Context, svc api.Service, repo, tag, auth string) error {
	service, _, err := p.docker.ServiceInspectWithRaw(ctx, svc.ID, types.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect failed: %w", err)
	}

	// Determine clean target image reference without any stale @sha256 digest
	targetImage := svc.Image
	if atIdx := strings.Index(targetImage, "@"); atIdx != -1 {
		targetImage = targetImage[:atIdx]
	}
	if targetImage == "" || !strings.Contains(targetImage, tag) {
		targetImage = repo + ":" + tag
	}

	service.Spec.TaskTemplate.ContainerSpec.Image = targetImage
	service.Spec.TaskTemplate.ForceUpdate++

	// Attempt 1: QueryRegistry: true with auth so Swarm resolves the new remote digest
	opts := types.ServiceUpdateOptions{
		QueryRegistry: true,
	}
	if auth != "" {
		opts.EncodedRegistryAuth = auth
	}

	_, updateErr := p.docker.ServiceUpdate(ctx, svc.ID, service.Version, service.Spec, opts)
	if updateErr == nil {
		log.Printf("[Image Poller] Successfully restarted service %s to use updated image %s", svc.Name, targetImage)
		return nil
	}

	// Attempt 2: If QueryRegistry failed (e.g. registry auth missing from manager or network),
	// retry with QueryRegistry: false. Since Halyard already pulled the updated image to the nodes,
	// the nodes will use the freshly pulled image.
	// Re-inspect to get fresh service.Version to prevent "update out of sequence"
	freshSvc, _, inspectErr := p.docker.ServiceInspectWithRaw(ctx, svc.ID, types.ServiceInspectOptions{})
	if inspectErr == nil {
		service = freshSvc
		service.Spec.TaskTemplate.ContainerSpec.Image = targetImage
		service.Spec.TaskTemplate.ForceUpdate++
	}

	opts.QueryRegistry = false
	_, retryErr := p.docker.ServiceUpdate(ctx, svc.ID, service.Version, service.Spec, opts)
	if retryErr != nil {
		return fmt.Errorf("service update failed (registry: %v, fallback: %w)", updateErr, retryErr)
	}

	log.Printf("[Image Poller] Successfully restarted service %s using local image %s", svc.Name, targetImage)
	return nil
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
		log.Printf("[Image Poller] Failed to save deployment audit event: %v", err)
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
