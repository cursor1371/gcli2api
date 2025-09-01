package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages persistence of derived metadata like token_key -> project_id.
type Store struct {
	db     *sql.DB
	mem    map[string]string // fallback when db unavailable
	mu     sync.RWMutex
	closed bool
}

// Open opens a SQLite database at path and ensures schema. If opening fails, a
// memory-only store is returned with db == nil.
func Open(path string) (*Store, error) {
	s := &Store{mem: make(map[string]string)}
	// Ensure parent directory exists if path contains directories
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return s, fmt.Errorf("prepare sqlite dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return s, fmt.Errorf("open sqlite: %w", err)
	}
	// Apply busy timeout and WAL pragmas for robustness
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		// Non-fatal
	}
	if err := s.init(db); err != nil {
		// fall back to mem-only if schema fails
		_ = db.Close()
		return s, nil
	}
	s.db = db
	return s, nil
}

func (s *Store) init(db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS token_project (
  token_key TEXT PRIMARY KEY,
  provider TEXT,
  client_id TEXT,
  project_id TEXT NOT NULL,
  last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_token_project_client ON token_project(client_id);
CREATE INDEX IF NOT EXISTS idx_token_project_last_used ON token_project(last_used_at);
`
	_, err := db.Exec(ddl)
	return err
}

// Close closes the underlying DB if present.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetProjectID returns the project id for tokenKey, and whether it was found.
func (s *Store) GetProjectID(ctx context.Context, tokenKey string) (string, bool, error) {
	if s.db == nil {
		s.mu.RLock()
		pid, ok := s.mem[tokenKey]
		s.mu.RUnlock()
		return pid, ok, nil
	}
	var pid string
	err := s.db.QueryRowContext(ctx, `SELECT project_id FROM token_project WHERE token_key = ?`, tokenKey).Scan(&pid)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	// Best-effort last_used update, ignore error
	_, _ = s.db.ExecContext(ctx, `UPDATE token_project SET last_used_at = ? WHERE token_key = ?`, time.Now(), tokenKey)
	return pid, true, nil
}

// UpsertProjectID stores or updates the mapping for tokenKey.
func (s *Store) UpsertProjectID(ctx context.Context, tokenKey, provider, clientID, projectID string) error {
	if s.db == nil {
		s.mu.Lock()
		s.mem[tokenKey] = projectID
		s.mu.Unlock()
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO token_project (token_key, provider, client_id, project_id, last_used_at)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(token_key) DO UPDATE SET project_id=excluded.project_id, last_used_at=excluded.last_used_at`,
		tokenKey, provider, clientID, projectID, time.Now())
	return err
}

// ComputeTokenKey returns a stable digest for a credential identity.
func ComputeTokenKey(provider, clientID, identityValue string) string {
	h := sha256.Sum256([]byte(provider + ":" + clientID + ":" + identityValue))
	return hex.EncodeToString(h[:])
}
