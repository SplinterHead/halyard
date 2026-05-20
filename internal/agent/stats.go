package agent

import (
	"context"
	"os"
	"time"

	"github.com/lewis-england/halyard/api"
	"github.com/lewis-england/halyard/internal/pkg/docker"
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
		Hostname:    hostname,
		CPUUsage:    cpuUsage,
		MemoryUsage: vm.Used,
		MemoryTotal: vm.Total,
		Uptime:      h.Uptime,
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
