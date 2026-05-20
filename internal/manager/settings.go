package manager

import (
	"encoding/json"
	"sync"

	"github.com/SplinterHead/halyard/api"
)

type SettingsManager struct {
	db *DB
	mu sync.RWMutex
}

func NewSettingsManager(db *DB) *SettingsManager {
	return &SettingsManager{
		db: db,
	}
}

func (m *SettingsManager) GetSettings() api.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var val string
	err := m.db.QueryRow("SELECT value FROM settings WHERE key = 'git_sync_concurrency'").Scan(&val)
	if err != nil {
		return api.Settings{GitSyncConcurrency: 5} // Default
	}

	var s api.Settings
	if err := json.Unmarshal([]byte(val), &s); err != nil {
		return api.Settings{GitSyncConcurrency: 5}
	}
	return s
}

func (m *SettingsManager) UpdateSettings(s api.Settings) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, _ := json.Marshal(s)
	m.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('git_sync_concurrency', ?)", string(data))
}
