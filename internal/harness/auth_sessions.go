package harness

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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

func (s *authStore) revokeUserSessions(id string) error {
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, time.Now().UTC().Format(time.RFC3339), id)
	return err
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

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expires, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}
