package agent

import (
	"context"
	"os"
	"strconv"
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
	s.checkHostUpdates()
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.checkHostUpdates()
	}
}

func (s *StatsCollector) checkHostUpdates() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Check for updates
	out, err := s.docker.RunHostCommand(ctx, "apt list --upgradable 2>/dev/null | grep -v Listing | wc -l")
	if err == nil {
		count, err := strconv.Atoi(strings.TrimSpace(out))
		if err == nil {
			s.pendingUpdates.Store(int32(count))
		}
	}

	// Check for reboot requirement
	_, errReboot := s.docker.RunHostCommand(ctx, "test -f /run/reboot-required || test -f /run/reboot-required.pkgs")
	s.restartRequired.Store(errReboot == nil)
}

func (s *StatsCollector) ListPendingUpdates(ctx context.Context) ([]string, error) {
	out, err := s.docker.RunHostCommand(ctx, "apt list --upgradable 2>/dev/null | grep -v Listing")
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
		parts := strings.SplitN(line, "/", 2)
		if len(parts) == 2 {
			packages = append(packages, parts[0])
		}
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
