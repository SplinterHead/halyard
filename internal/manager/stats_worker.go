package manager

import (
	"context"
	"log"
	"time"
)

type StatsWorker struct {
	nodeMgr *NodeManager
	db      *DB
}

func NewStatsWorker(nodeMgr *NodeManager, db *DB) *StatsWorker {
	return &StatsWorker{
		nodeMgr: nodeMgr,
		db:      db,
	}
}

func (w *StatsWorker) Start(ctx context.Context) {
	log.Println("Starting background stats collector...")
	
	// Collect every 30 seconds for historical data
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Also prune every hour
	pruneTicker := time.NewTicker(1 * time.Hour)
	defer pruneTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.collect(ctx)
		case <-pruneTicker.C:
			w.prune()
		}
	}
}

func (w *StatsWorker) collect(ctx context.Context) {
	stats, err := w.nodeMgr.ListNodes(ctx)
	if err != nil {
		log.Printf("Worker failed to list nodes: %v", err)
		return
	}

	for _, s := range stats {
		s.Timestamp = time.Now()
		if err := w.db.SaveNodeStats(s); err != nil {
			log.Printf("Worker failed to save stats for node %s: %v", s.NodeID, err)
		}
	}
}

func (w *StatsWorker) prune() {
	// Keep 24 hours of data
	if err := w.db.PruneNodeStats(24); err != nil {
		log.Printf("Worker failed to prune stats: %v", err)
	}
}
