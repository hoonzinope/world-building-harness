package harness

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const sessionCookieName = "wh_session"
const csrfCookieName = "wh_csrf"

type authStore struct {
	db *sql.DB
}

type authUser struct {
	ID          string
	Username    string
	DisplayName string
	Role        string
	Status      string
	LastLoginAt string
}

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

func (s *authStore) authenticate(username, password string) (*authUser, error) {
	var u authUser
	var hash string
	err := s.db.QueryRow(`SELECT id, username, display_name, role, status, password_hash, COALESCE(last_login_at,'') FROM users WHERE username=?`, username).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Status, &hash, &u.LastLoginAt)
	if err != nil || u.Status != "active" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi0pPaWxn96p36y3Zsd8iQEe4CEu0dq"), []byte(password))
		return nil, errors.New("invalid credentials")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = s.db.Exec(`UPDATE users SET last_login_at=?, updated_at=? WHERE id=?`, now, now, u.ID)
	u.LastLoginAt = now
	return &u, nil
}

func (s *authStore) createSession(userID string) (string, time.Time, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().UTC().Add(14 * 24 * time.Hour)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"sess_"+randomID(), userID, tokenHash(token), now, expires.Format(time.RFC3339), now)
	return token, expires, err
}

func (s *authStore) userForToken(token string) (*authUser, error) {
	var u authUser
	now := time.Now().UTC()
	err := s.db.QueryRow(`SELECT u.id, u.username, u.display_name, u.role, u.status, COALESCE(u.last_login_at,'')
FROM sessions s JOIN users u ON u.id=s.user_id
WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at > ? AND u.status='active'`,
		tokenHash(token), now.Format(time.RFC3339)).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Status, &u.LastLoginAt)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec(`UPDATE sessions SET last_seen_at=? WHERE token_hash=?`, now.Format(time.RFC3339), tokenHash(token))
	return &u, nil
}

func (s *authStore) revokeToken(token string) {
	_, _ = s.db.Exec(`UPDATE sessions SET revoked_at=? WHERE token_hash=?`, time.Now().UTC().Format(time.RFC3339), tokenHash(token))
}

func (s *authStore) listUsers() ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT u.id, u.username, u.display_name, u.role, u.status, COALESCE(u.last_login_at,''), COUNT(s.id)
FROM users u LEFT JOIN sessions s ON s.user_id=u.id AND s.revoked_at IS NULL AND s.expires_at > ?
GROUP BY u.id ORDER BY u.username`, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, username, display, role, status, last string
		var sessions int
		if err := rows.Scan(&id, &username, &display, &role, &status, &last, &sessions); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "username": username, "display_name": display, "role": role, "status": status, "last_login_at": last, "active_sessions": sessions})
	}
	return out, rows.Err()
}

func (s *authStore) createUser(username, display, role, password string) error {
	if role != "admin" {
		role = "friend"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`INSERT INTO users (id, username, display_name, role, password_hash, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`,
		"user_"+slugID(username), username, firstNonEmpty(display, username), role, string(hash), now, now)
	return err
}

func (s *authStore) updateUser(id, role, status string) error {
	if role != "admin" {
		role = "friend"
	}
	if status != "disabled" {
		status = "active"
	}
	_, err := s.db.Exec(`UPDATE users SET role=?, status=?, updated_at=? WHERE id=?`, role, status, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *authStore) resetPassword(id, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`, string(hash), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *authStore) resetPasswordByUsername(username, password string) error {
	var id string
	if err := s.db.QueryRow(`SELECT id FROM users WHERE username=?`, username).Scan(&id); err != nil {
		return err
	}
	return s.resetPassword(id, password)
}

func (s *authStore) revokeUserSessions(id string) error {
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *authStore) firstActiveAdminID() (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM users WHERE role='admin' AND status='active' ORDER BY created_at, username LIMIT 1`).Scan(&id)
	return id, err
}

type authContextKey struct{}

func currentUser(r *http.Request) *authUser {
	u, _ := r.Context().Value(authContextKey{}).(*authUser)
	return u
}

func withUser(r *http.Request, u *authUser) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authContextKey{}, u))
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expires, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if c, err := r.Cookie(csrfCookieName); err == nil && strings.TrimSpace(c.Value) != "" {
		return c.Value, nil
	}
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	setCSRFCookie(w, token, r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	return token, nil
}

func requireCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return false
	}
	form := strings.TrimSpace(r.FormValue("csrf_token"))
	if form == "" {
		return false
	}
	return cookie.Value == form
}

func setCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().UTC().Add(14 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomID() string {
	t, _ := randomToken(12)
	return slugID(t)
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func slugID(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() == 0 || strings.HasSuffix(b.String(), "_") {
			continue
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "id_" + fmt.Sprint(time.Now().UnixNano())
	}
	return out
}
