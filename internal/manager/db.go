package manager

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SplinterHead/halyard/api"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and set busy timeout for better concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA busy_timeout=5000;")
	
	// SQLite only supports one writer at a time. 
	// Setting MaxOpenConns to 1 is often the most reliable way to avoid SQLITE_BUSY
	// especially when using modernc.org/sqlite which can be sensitive to concurrency.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{db}
	if err := d.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return d, nil
}

func (d *DB) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS git_sources (
			id TEXT PRIMARY KEY,
			name TEXT,
			url TEXT,
			description TEXT,
			username TEXT,
			token TEXT,
			last_status TEXT,
			last_error TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS git_syncs (
			id TEXT PRIMARY KEY,
			name TEXT,
			source_id TEXT,
			stack_name TEXT,
			branch TEXT,
			file_path TEXT,
			pull_additional_files BOOLEAN,
			last_applied_sha TEXT,
			last_status TEXT,
			last_error TEXT,
			created_at DATETIME,
			FOREIGN KEY(source_id) REFERENCES git_sources(id)
		);`,
		`CREATE TABLE IF NOT EXISTS deployment_history (
			id TEXT PRIMARY KEY,
			sync_id TEXT,
			sha TEXT,
			status TEXT,
			logs TEXT,
			timestamp DATETIME,
			FOREIGN KEY(sync_id) REFERENCES git_syncs(id)
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE,
			real_name TEXT,
			password_hash TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS registries (
			id TEXT PRIMARY KEY,
			name TEXT,
			url TEXT,
			username TEXT,
			password TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS node_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT,
			hostname TEXT,
			cpu_usage REAL,
			memory_usage INTEGER,
			memory_total INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, q := range queries {
		if _, err := d.Exec(q); err != nil {
			return err
		}
	}

	_, _ = d.Exec("CREATE INDEX IF NOT EXISTS idx_node_stats_node_id_timestamp ON node_stats(node_id, timestamp)")

	// Simple migrations
	_, _ = d.Exec("ALTER TABLE git_syncs ADD COLUMN stack_name TEXT")
	_, _ = d.Exec("ALTER TABLE git_syncs ADD COLUMN pull_additional_files BOOLEAN DEFAULT 0")
	_, _ = d.Exec("ALTER TABLE git_syncs ADD COLUMN auto_sync BOOLEAN DEFAULT 0")
	_, _ = d.Exec("ALTER TABLE git_syncs ADD COLUMN last_sync_at DATETIME")

	return nil
}

func (d *DB) SaveNodeStats(stats api.NodeStats) error {
	_, err := d.Exec(`INSERT INTO node_stats (node_id, hostname, cpu_usage, memory_usage, memory_total, timestamp) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		stats.NodeID, stats.Hostname, stats.CPUUsage, stats.MemoryUsage, stats.MemoryTotal, stats.Timestamp)
	return err
}

func (d *DB) GetHistoricalNodeStats(nodeID string, hours int) ([]api.NodeStats, error) {
	rows, err := d.Query(`SELECT node_id, hostname, cpu_usage, memory_usage, memory_total, timestamp 
		FROM node_stats WHERE node_id = ? AND timestamp >= datetime('now', '-' || ? || ' hours') 
		ORDER BY timestamp ASC`, nodeID, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []api.NodeStats
	for rows.Next() {
		var s api.NodeStats
		if err := rows.Scan(&s.NodeID, &s.Hostname, &s.CPUUsage, &s.MemoryUsage, &s.MemoryTotal, &s.Timestamp); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (d *DB) PruneNodeStats(retentionHours int) error {
	_, err := d.Exec(`DELETE FROM node_stats WHERE timestamp < datetime('now', '-' || ? || ' hours')`, retentionHours)
	return err
}

func (d *DB) GetOrCreateSessionSecret() ([]byte, error) {
	var secretHex string
	err := d.QueryRow("SELECT value FROM settings WHERE key = 'auth_token_secret'").Scan(&secretHex)
	if err == sql.ErrNoRows {
		// Generate random 32 bytes
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}
		secretHex = hex.EncodeToString(bytes)
		_, err = d.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('auth_token_secret', ?)", secretHex)
		if err != nil {
			return nil, fmt.Errorf("failed to save session secret: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query session secret: %w", err)
	}

	secretBytes, err := hex.DecodeString(secretHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session secret: %w", err)
	}
	return secretBytes, nil
}

