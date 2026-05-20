package manager

import (
	"context"
	"fmt"
	"log"
	"time"
)

type GitWorker struct {
	syncMgr     *GitSyncManager
	gitMgr      *GitManager
	settingsMgr *SettingsManager
	stackMgr    *StackManager
}

func NewGitWorker(syncMgr *GitSyncManager, gitMgr *GitManager, settingsMgr *SettingsManager, stackMgr *StackManager) *GitWorker {
	return &GitWorker{
		syncMgr:     syncMgr,
		gitMgr:      gitMgr,
		settingsMgr: settingsMgr,
		stackMgr:    stackMgr,
	}
}

func (w *GitWorker) Start(ctx context.Context) {
	// Start with a default ticker, it will be adjusted
	interval := w.settingsMgr.GetSettings().GitSyncInterval
	if interval <= 0 {
		interval = 5
	}

	lastSync := time.Now()
	lastRepoTest := time.Now()
	checkTicker := time.NewTicker(10 * time.Second)
	defer checkTicker.Stop()

	log.Printf("Worker: Starting Git reconciliation loop (Interval: %d min)", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTicker.C:
			// Re-check interval in case it changed
			newInterval := w.settingsMgr.GetSettings().GitSyncInterval
			if newInterval > 0 && newInterval != interval {
				log.Printf("Worker: Adjusting interval from %d to %d min", interval, newInterval)
				interval = newInterval
			}

			// If it's time to sync
			if time.Since(lastSync) >= time.Duration(interval)*time.Minute {
				w.processSyncs(ctx)
				lastSync = time.Now()
			}

			// If it's time to test repo connections (every 24 hours)
			if time.Since(lastRepoTest) >= 24*time.Hour {
				w.testAllRepos(ctx)
				lastRepoTest = time.Now()
			}
		}
	}
}

func (w *GitWorker) testAllRepos(ctx context.Context) {
	log.Printf("Worker: Starting scheduled connection tests for all repositories")
	repos := w.gitMgr.ListRepos()
	for _, r := range repos {
		err := w.gitMgr.TestRepo(r.ID)
		if err != nil {
			log.Printf("Worker: Scheduled test failed for repo %s (%s): %v", r.Name, r.ID, err)
		} else {
			log.Printf("Worker: Scheduled test successful for repo %s (%s)", r.Name, r.ID)
		}
	}
}

func (w *GitWorker) processSyncs(ctx context.Context) {
	syncs := w.syncMgr.ListSyncs()
	for _, s := range syncs {
		if !s.AutoSync {
			continue
		}

		repo, ok := w.gitMgr.GetRepo(s.RepositoryID)
		if !ok {
			log.Printf("Worker: Repo %s not found for sync %s", s.RepositoryID, s.ID)
			continue
		}

		// Fetch latest SHA
		latestSHA, err := w.gitMgr.GetLatestSHA(repo, s.Branch)
		if err != nil {
			log.Printf("Worker: Failed to get latest SHA for sync %s: %v", s.ID, err)
			w.syncMgr.UpdateSyncStatus(s.ID, "Failed", fmt.Sprintf("Fetch failed: %v", err), s.LastAppliedSHA)
			continue
		}

		if latestSHA != s.LastAppliedSHA {
			log.Printf("Worker: New SHA detected for sync %s: %s -> %s", s.StackName, s.LastAppliedSHA, latestSHA)

			// Set status to "Out of Date" first so UI turns red
			w.syncMgr.UpdateSyncStatus(s.ID, "Out of Date", "", s.LastAppliedSHA)

			err := w.syncMgr.PerformSync(ctx, s.ID, "Automatic reconciliation triggered by SHA change")
			if err != nil {
				log.Printf("Worker: Sync failed for %s: %v", s.StackName, err)
			} else {
				log.Printf("Worker: Sync successful for %s", s.StackName)
			}
		} else {
			// Update last sync time even if no change
			w.syncMgr.UpdateSyncTime(s.ID)
		}
	}
}
