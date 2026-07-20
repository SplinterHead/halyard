package agent

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type StatsCollector struct {
	docker *docker.Client
}

func NewStatsCollector(cli *docker.Client) *StatsCollector {
	return &StatsCollector{docker: cli}
}

func checkPendingRestart() bool {
	if _, err := os.Stat("/host/var/run/reboot-required"); err == nil {
		return true
	}
	if _, err := os.Stat("/host/var/run/reboot-required.pkgs"); err == nil {
		return true
	}
	return false
}

func checkPendingUpdates() int {
	data, err := os.ReadFile("/host/var/lib/update-notifier/updates-available")
	if err == nil {
		// Try parsing standard Ubuntu message
		re := regexp.MustCompile(`(?m)^(\d+)\s+updates?`)
		matches := re.FindStringSubmatch(string(data))
		if len(matches) > 1 {
			count, _ := strconv.Atoi(matches[1])
			return count
		}

		// Try parsing alternative package message
		rePkg := regexp.MustCompile(`(?m)^(\d+)\s+packages?`)
		matchesPkg := rePkg.FindStringSubmatch(string(data))
		if len(matchesPkg) > 1 {
			count, _ := strconv.Atoi(matchesPkg[1])
			return count
		}
	}
	return 0
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
		PendingUpdates:  checkPendingUpdates(),
		RestartRequired: checkPendingRestart(),
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
