package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/SplinterHead/halyard/api"
)

type GitSyncManager struct {
	db          *DB
	gitMgr      *GitManager
	stackMgr    *StackManager
	registryMgr *RegistryManager
	mu          sync.RWMutex
}

func NewGitSyncManager(db *DB, gitMgr *GitManager, stackMgr *StackManager, registryMgr *RegistryManager) *GitSyncManager {
	return &GitSyncManager{
		db:          db,
		gitMgr:      gitMgr,
		stackMgr:    stackMgr,
		registryMgr: registryMgr,
	}
}

func (m *GitSyncManager) ListSyncs() []api.GitSync {
	rows, err := m.db.Query("SELECT id, name, source_id, stack_name, branch, file_path, pull_additional_files, auto_sync, last_applied_sha, last_sync_at, last_status, last_error, created_at FROM git_syncs")
	if err != nil {
		return []api.GitSync{}
	}
	defer rows.Close()

	list := make([]api.GitSync, 0)
	for rows.Next() {
		var s api.GitSync
		rows.Scan(&s.ID, &s.Name, &s.RepositoryID, &s.StackName, &s.Branch, &s.Path, &s.PullAdditionalFiles, &s.AutoSync, &s.LastAppliedSHA, &s.LastSyncAt, &s.LastStatus, &s.LastError, &s.CreatedAt)
		list = append(list, s)
	}
	return list
}

func (m *GitSyncManager) GetSync(id string) (api.GitSync, bool) {
	var s api.GitSync
	err := m.db.QueryRow("SELECT id, name, source_id, stack_name, branch, file_path, pull_additional_files, auto_sync, last_applied_sha, last_sync_at, last_status, last_error, created_at FROM git_syncs WHERE id = ?", id).
		Scan(&s.ID, &s.Name, &s.RepositoryID, &s.StackName, &s.Branch, &s.Path, &s.PullAdditionalFiles, &s.AutoSync, &s.LastAppliedSHA, &s.LastSyncAt, &s.LastStatus, &s.LastError, &s.CreatedAt)
	return s, err == nil
}

func (m *GitSyncManager) AddSync(sync api.GitSync) (api.GitSync, error) {
	ctx := context.Background()

	// Check for uniqueness
	var count int
	m.db.QueryRow("SELECT COUNT(*) FROM git_syncs WHERE stack_name = ?", sync.StackName).Scan(&count)
	if count > 0 {
		return api.GitSync{}, fmt.Errorf("a sync with stack name '%s' already exists", sync.StackName)
	}

	// 1. Get Repo
	repo, ok := m.gitMgr.GetRepo(sync.RepositoryID)
	if !ok {
		return api.GitSync{}, fmt.Errorf("repository not found")
	}

	// 2. Test File Existence
	err := m.gitMgr.CheckFile(repo, sync.Branch, sync.Path)
	if err != nil {
		sync.LastStatus = "Failed"
		sync.LastError = fmt.Sprintf("File check failed: %v", err)
	} else {
		sync.LastStatus = "Ready"
		sync.LastError = ""

		// 3. Check if stack exists
		exists, err := m.stackMgr.Exists(ctx, sync.StackName)
		if err != nil {
			log.Printf("SyncMgr: Failed to check if stack %s exists: %v", sync.StackName, err)
		} else if !exists {
			log.Printf("SyncMgr: Stack %s does not exist, deploying fresh...", sync.StackName)
			
			registries, _ := m.registryMgr.ListRegistries()
			if sync.PullAdditionalFiles {
				tmpDir, cleanup, err := m.gitMgr.CloneToDisk(repo, sync.Branch)
				if err != nil {
					sync.LastStatus = "Failed"
					sync.LastError = fmt.Sprintf("Failed to clone for fresh deploy: %v", err)
				} else {
					defer cleanup()
					err = m.stackMgr.DeployStack(ctx, sync.StackName, tmpDir, sync.Path, registries)
					if err != nil {
						sync.LastStatus = "Failed"
						sync.LastError = fmt.Sprintf("Fresh deploy failed: %v", err)
					} else {
						sync.LastStatus = "Synced"
					}
				}
			} else {
				content, err := m.gitMgr.GetFileContent(repo, sync.Branch, sync.Path)
				if err != nil {
					sync.LastStatus = "Failed"
					sync.LastError = fmt.Sprintf("Failed to get compose content: %v", err)
				} else {
					// Use a simpler deploy for single file
					err = m.stackMgr.DeployStackContent(ctx, sync.StackName, content, registries)
					if err != nil {
						sync.LastStatus = "Failed"
						sync.LastError = fmt.Sprintf("Fresh deploy failed: %v", err)
					} else {
						sync.LastStatus = "Synced"
					}
				}
			}
		}
	}

	// 4. Save
	b := make([]byte, 16)
	rand.Read(b)
	sync.ID = hex.EncodeToString(b)
	now := time.Now()
	sync.CreatedAt = &now

	_, err = m.db.Exec(`INSERT INTO git_syncs (id, name, source_id, stack_name, branch, file_path, pull_additional_files, auto_sync, last_applied_sha, last_sync_at, last_status, last_error, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sync.ID, sync.Name, sync.RepositoryID, sync.StackName, sync.Branch, sync.Path, sync.PullAdditionalFiles, sync.AutoSync, sync.LastAppliedSHA, sync.LastSyncAt, sync.LastStatus, sync.LastError, sync.CreatedAt)

	if err != nil {
		log.Printf("SyncMgr: Failed to insert sync: %v", err)
	}

	return sync, err
}

func (m *GitSyncManager) UpdateSync(sync api.GitSync) (api.GitSync, error) {
	// 1. Get Repo
	repo, ok := m.gitMgr.GetRepo(sync.RepositoryID)
	if !ok {
		return api.GitSync{}, fmt.Errorf("repository not found")
	}

	// 2. Test File Existence
	err := m.gitMgr.CheckFile(repo, sync.Branch, sync.Path)
	if err != nil {
		sync.LastStatus = "Failed"
		sync.LastError = fmt.Sprintf("File check failed: %v", err)
	} else {
		sync.LastStatus = "Ready"
		sync.LastError = ""
	}

	_, err = m.db.Exec(`UPDATE git_syncs SET name = ?, source_id = ?, stack_name = ?, branch = ?, file_path = ?, pull_additional_files = ?, auto_sync = ?, last_status = ?, last_error = ? WHERE id = ?`,
		sync.Name, sync.RepositoryID, sync.StackName, sync.Branch, sync.Path, sync.PullAdditionalFiles, sync.AutoSync, sync.LastStatus, sync.LastError, sync.ID)
	
	return sync, err
}

func (m *GitSyncManager) DeleteSync(id string) error {
	_, err := m.db.Exec("DELETE FROM git_syncs WHERE id = ?", id)
	return err
}

func (m *GitSyncManager) UpdateSyncStatus(id string, status, errStr, sha string) error {
	now := time.Now()
	_, err := m.db.Exec("UPDATE git_syncs SET last_status = ?, last_error = ?, last_applied_sha = ?, last_sync_at = ? WHERE id = ?", status, errStr, sha, now, id)
	return err
}

func (m *GitSyncManager) UpdateSyncTime(id string) error {
	now := time.Now()
	_, err := m.db.Exec("UPDATE git_syncs SET last_sync_at = ? WHERE id = ?", now, id)
	return err
}

func (m *GitSyncManager) ListDeploymentHistory() []api.DeploymentEvent {
	rows, err := m.db.Query(`
		SELECT h.id, h.sync_id, s.stack_name, h.sha, h.status, h.logs, h.timestamp 
		FROM deployment_history h
		LEFT JOIN git_syncs s ON h.sync_id = s.id
		ORDER BY h.timestamp DESC
	`)
	if err != nil {
		return []api.DeploymentEvent{}
	}
	defer rows.Close()

	list := make([]api.DeploymentEvent, 0)
	for rows.Next() {
		var e api.DeploymentEvent
		rows.Scan(&e.ID, &e.SyncID, &e.StackName, &e.SHA, &e.Status, &e.Logs, &e.Timestamp)
		list = append(list, e)
	}
	return list
}

func (m *GitSyncManager) AddDeploymentHistory(syncID string, sha string, status string, logs string) error {
	b := make([]byte, 16)
	rand.Read(b)
	id := hex.EncodeToString(b)
	
	_, err := m.db.Exec("INSERT INTO deployment_history (id, sync_id, sha, status, logs, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		id, syncID, sha, status, logs, time.Now())
	return err
}

func (m *GitSyncManager) PerformSync(ctx context.Context, id string, trigger string) error {
	sync, ok := m.GetSync(id)
	if !ok {
		return fmt.Errorf("sync not found")
	}

	repo, ok := m.gitMgr.GetRepo(sync.RepositoryID)
	if !ok {
		return fmt.Errorf("repository not found")
	}

	// 0. Set status to Syncing
	m.UpdateSyncStatus(sync.ID, "Syncing", "", sync.LastAppliedSHA)

	// 1. Get latest SHA and registries
	latestSHA, err := m.gitMgr.GetLatestSHA(repo, sync.Branch)
	if err != nil {
		latestSHA = sync.LastAppliedSHA // Fallback to current if fetch fails
	}

	registries, err := m.registryMgr.ListRegistries()
	if err != nil {
		log.Printf("SyncMgr: Failed to list registries for sync %s: %v", sync.ID, err)
		// Continue with empty registries
	}

	// 2. Prepare deployment
	var deployErr error
	if sync.PullAdditionalFiles {
		tmpDir, cleanup, err := m.gitMgr.CloneToDisk(repo, sync.Branch)
		if err != nil {
			deployErr = fmt.Errorf("clone failed: %v", err)
		} else {
			defer cleanup()
			deployErr = m.stackMgr.DeployStack(ctx, sync.StackName, tmpDir, sync.Path, registries)
		}
	} else {
		content, err := m.gitMgr.GetFileContent(repo, sync.Branch, sync.Path)
		if err != nil {
			deployErr = fmt.Errorf("fetch file failed: %v", err)
		} else {
			deployErr = m.stackMgr.DeployStackContent(ctx, sync.StackName, content, registries)
		}
	}

	// 3. Handle results
	if deployErr != nil {
		m.UpdateSyncStatus(sync.ID, "Failed", deployErr.Error(), sync.LastAppliedSHA)
		m.AddDeploymentHistory(sync.ID, latestSHA, "Failed", deployErr.Error())
		return deployErr
	}

	m.UpdateSyncStatus(sync.ID, "Synced", "", latestSHA)
	m.AddDeploymentHistory(sync.ID, latestSHA, "Success", trigger)
	return nil
}

func (m *GitSyncManager) ForceSync(id string) error {
	return m.PerformSync(context.Background(), id, "Manual sync triggered by user")
}
