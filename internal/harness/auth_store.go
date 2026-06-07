package harness

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const sessionCookieName = "wh_session"
const csrfCookieName = "wh_csrf"

func openAuthStore(path string) (*authStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &authStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.bootstrapAdmin(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *authStore) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('admin','friend')),
  password_hash TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','disabled')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_login_at TEXT
);
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  revoked_at TEXT,
  last_seen_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
`)
	return err
}

func (s *authStore) bootstrapAdmin() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	username := strings.TrimSpace(os.Getenv("WORLD_HARNESS_ADMIN_USERNAME"))
	password := os.Getenv("WORLD_HARNESS_ADMIN_PASSWORD")
	if file := strings.TrimSpace(os.Getenv("WORLD_HARNESS_ADMIN_PASSWORD_FILE")); file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read WORLD_HARNESS_ADMIN_PASSWORD_FILE: %w", err)
		}
		password = strings.TrimSpace(string(b))
	}
	if username == "" || password == "" {
		return errors.New("active admin does not exist; set WORLD_HARNESS_ADMIN_USERNAME and WORLD_HARNESS_ADMIN_PASSWORD or WORLD_HARNESS_ADMIN_PASSWORD_FILE")
	}
	display := firstNonEmpty(strings.TrimSpace(os.Getenv("WORLD_HARNESS_ADMIN_DISPLAY_NAME")), username)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`INSERT INTO users (id, username, display_name, role, password_hash, status, created_at, updated_at) VALUES (?, ?, ?, 'admin', ?, 'active', ?, ?)`,
		"user_"+slugID(username), username, display, string(hash), now, now)
	return err
}
