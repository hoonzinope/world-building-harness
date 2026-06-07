package server

import (
	"context"
	"net/http"
	"time"

	"github.com/hoonzi/world-harness/internal/harness/auth"
)

type authStore struct{ *auth.Store }
type authUser = auth.User

type authContextKey struct{}

const sessionCookieName = auth.SessionCookieName
const csrfCookieName = auth.CSRFCookieName

func openAuthStore(path string) (*authStore, error) {
	s, err := auth.OpenStore(path)
	if err != nil {
		return nil, err
	}
	return &authStore{Store: s}, nil
}

func currentUser(r *http.Request) *authUser {
	u, _ := r.Context().Value(authContextKey{}).(*authUser)
	return u
}

func withUser(r *http.Request, u *authUser) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authContextKey{}, u))
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request) (string, error) {
	return auth.EnsureCSRFToken(w, r)
}
func requireCSRF(r *http.Request) bool { return auth.RequireCSRF(r) }
func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	auth.SetSessionCookie(w, token, expires, secure)
}
func clearSessionCookie(w http.ResponseWriter) { auth.ClearSessionCookie(w) }
func setCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	auth.SetCSRFCookie(w, token, secure)
}

func (s *authStore) authenticate(username, password string) (*authUser, error) {
	return s.Store.Authenticate(username, password)
}

func (s *authStore) createSession(userID string) (string, time.Time, error) {
	return s.Store.CreateSession(userID)
}

func (s *authStore) userForToken(token string) (*authUser, error) {
	return s.Store.UserForToken(token)
}

func (s *authStore) revokeToken(token string) { s.Store.RevokeToken(token) }

func (s *authStore) revokeUserSessions(id string) error { return s.Store.RevokeUserSessions(id) }

func (s *authStore) listUsers() ([]map[string]any, error) { return s.Store.ListUsers() }

func (s *authStore) createUser(username, display, role, password string) error {
	return s.Store.CreateUser(username, display, role, password)
}

func (s *authStore) updateUser(id, role, status string) error {
	return s.Store.UpdateUser(id, role, status)
}

func (s *authStore) resetPassword(id, password string) error {
	return s.Store.ResetPassword(id, password)
}

func (s *authStore) resetPasswordByUsername(username, password string) error {
	return s.Store.ResetPasswordByUsername(username, password)
}

func (s *authStore) firstActiveAdminID() (string, error) { return s.Store.FirstActiveAdminID() }
