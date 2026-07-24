package agent

import (
	"context"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type StatsCollector struct {
	docker          *docker.Client
	pendingUpdates  atomic.Int32
	restartRequired atomic.Bool
}

func NewStatsCollector(cli *docker.Client) *StatsCollector {
	s := &StatsCollector{docker: cli}
	go s.pollHostUpdates()
	return s
}

func (s *StatsCollector) pollHostUpdates() {
	// Give the Docker daemon a moment to fully register this container's state
	time.Sleep(15 * time.Second)
	s.checkHostUpdates()
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.checkHostUpdates()
	}
}

func (s *StatsCollector) checkHostUpdates() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Running host update and reboot checks...")

	// Check for updates
	packages, err := s.ListPendingUpdates(ctx)
	if err == nil {
		count := len(packages)
		s.pendingUpdates.Store(int32(count))
		log.Printf("Host updates check completed: %d pending updates found", count)
	} else {
		log.Printf("Host updates check failed: %v", err)
	}

	// Check for reboot requirement
	outReboot, errReboot := s.docker.RunHostCommand(ctx, "test -f /run/reboot-required || test -f /run/reboot-required.pkgs")
	requiresReboot := errReboot == nil
	s.restartRequired.Store(requiresReboot)
	log.Printf("Host reboot check completed: requires_reboot=%v (err: %v, output: %s)", requiresReboot, errReboot, outReboot)
}

func (s *StatsCollector) ListPendingUpdates(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	out, err := s.docker.RunHostCommand(ctx, "DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1 && DEBIAN_FRONTEND=noninteractive apt-get -s -q upgrade 2>/dev/null | grep '^Inst ' | awk '{print $2}'")
	if err != nil {
		return nil, err
	}

	var packages []string
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		packages = append(packages, line)
	}
	return packages, nil
}

func (s *StatsCollector) GetNodeStats(ctx context.Context) (api.NodeStats, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return api.NodeStats{}, err
	}

	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return api.NodeStats{}, err
	}

	h, err := host.InfoWithContext(ctx)
	if err != nil {
		return api.NodeStats{}, err
	}

	cpuUsage := 0.0
	if len(percentages) > 0 {
		cpuUsage = percentages[0]
	}

	hostname := h.Hostname
	if os.Getenv("NODE_HOSTNAME") != "" {
		hostname = os.Getenv("NODE_HOSTNAME")
	}

	return api.NodeStats{
		Hostname:        hostname,
		CPUUsage:        cpuUsage,
		MemoryUsage:     vm.Used,
		MemoryTotal:     vm.Total,
		Uptime:          h.Uptime,
		PendingUpdates:  int(s.pendingUpdates.Load()),
		RestartRequired: s.restartRequired.Load(),
	}, nil
}

func (s *StatsCollector) StreamStats(ctx context.Context, statsChan chan<- api.NodeStats, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := s.GetNodeStats(ctx)
			if err == nil {
				statsChan <- stats
			}
		}
	}
}
