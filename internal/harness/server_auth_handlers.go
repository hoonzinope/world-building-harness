package harness

import (
	"net/http"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

func (s *webServer) addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; form-action 'self'; connect-src 'self'")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
}

func (s *webServer) mustCSRFToken(w http.ResponseWriter, r *http.Request) string {
	return mustCSRFToken(w, r)
}

func (s *webServer) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if !requireCSRF(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (s *webServer) requireStoryTaskCSRF(w http.ResponseWriter, r *http.Request) bool {
	if requireCSRF(r) {
		return true
	}
	if isJSONStoryTaskRequest(r) {
		formToken := strings.TrimSpace(r.FormValue("csrf_token"))
		if formToken != "" {
			headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if headerToken == "" || headerToken == formToken {
				return true
			}
		}
	}
	if wantsJSONResponse(r) {
		writeJSONResponse(w, http.StatusForbidden, map[string]any{"error": "csrf token mismatch"})
		return false
	}
	http.Error(w, "forbidden", http.StatusForbidden)
	return false
}

func (s *webServer) userFromRequest(r *http.Request) (*authUser, bool) {
	if s.auth == nil {
		return nil, false
	}
	c, err := r.Cookie(sessionCookieName)
	if err == nil && c.Value != "" {
		if u, err := s.auth.userForToken(c.Value); err == nil {
			return u, true
		}
	}
	return nil, false
}

func (s *webServer) requireAuth(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	if u, ok := s.userFromRequest(r); ok {
		return u, true
	}
	http.Redirect(w, r, s.base(r)+"/login", http.StatusSeeOther)
	return nil, false
}

func (s *webServer) requireLoggedInUser(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	if u := currentUser(r); u != nil {
		return u, true
	}
	if s.auth == nil {
		http.NotFound(w, r)
		return nil, false
	}
	return s.requireAuth(w, r)
}

func isAdminUser(u *authUser) bool { return u != nil && u.Role == "admin" }

func (s *webServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.render(w, r, "Login", ui.LoginTemplate, map[string]any{"Base": s.base(r), "CSRFToken": mustCSRFToken(w, r)})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	u, err := s.auth.authenticate(strings.TrimSpace(r.FormValue("username")), r.FormValue("password"))
	if err != nil {
		s.render(w, r, "Login", ui.LoginTemplate, map[string]any{"Base": s.base(r), "Error": "로그인 정보를 확인할 수 없습니다.", "CSRFToken": mustCSRFToken(w, r)})
		return
	}
	token, expires, err := s.auth.createSession(u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, expires, r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	http.Redirect(w, r, s.base(r)+"/stories", http.StatusSeeOther)
}

func (s *webServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireLoggedInUser(w, r); !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.auth.revokeToken(c.Value)
	}
	clearSessionCookie(w)
	http.Redirect(w, r, s.base(r)+"/login", http.StatusSeeOther)
}
