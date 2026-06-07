package server

import (
	"net/http"
	"strings"
)

func (s *webServer) handle(w http.ResponseWriter, r *http.Request) {
	s.addSecurityHeaders(w)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/health" || r.URL.Path == s.base(r)+"/health" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
		return
	}
	path := r.URL.Path
	base := s.base(r)
	if base != "" && strings.HasPrefix(path, base+"/") {
		path = strings.TrimPrefix(path, base)
	}
	if path == "/login" && s.auth != nil {
		s.handleLogin(w, r)
		return
	}
	if path == "/assets/story-room.js" {
		s.handleStoryRoomAsset(w, r)
		return
	}
	if s.pathRequiresAuth(path) {
		if s.auth == nil {
			http.NotFound(w, r)
			return
		}
		u, ok := s.requireAuth(w, r)
		if !ok {
			return
		}
		r = withUser(r, u)
	} else if u, ok := s.userFromRequest(r); ok {
		r = withUser(r, u)
	}
	switch {
	case path == "/" || path == "":
		if s.storyEnabled {
			http.Redirect(w, r, s.base(r)+"/stories", http.StatusSeeOther)
			return
		}
		s.renderIndex(w, r)
	case path == "/logout":
		s.handleLogout(w, r)
	case path == "/stories":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.renderStoryLobby(w, r)
	case path == "/stories/new":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleNewStory(w, r)
	case path == "/stories/import/hector":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleImportHector(w, r)
	case strings.HasPrefix(path, "/stories/"):
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleStoryRoute(w, r, strings.TrimPrefix(path, "/stories/"))
	case path == "/admin/users":
		s.handleAdminUsers(w, r)
	case strings.HasPrefix(path, "/packs/"):
		s.renderPackRoute(w, r, strings.TrimPrefix(path, "/packs/"))
	case path == "/api/packs":
		s.renderPackAPI(w)
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) pathRequiresAuth(path string) bool {
	switch path {
	case "/stories/new", "/stories/import/hector", "/admin/users", "/logout":
		return true
	}
	if strings.HasPrefix(path, "/stories/") {
		rest := strings.Trim(strings.TrimPrefix(path, "/stories/"), "/")
		if rest == "" {
			return false
		}
		parts := strings.Split(rest, "/")
		if len(parts) < 2 {
			return false
		}
		switch parts[1] {
		case "input", "question", "driver", "admin", "recover":
			return true
		}
	}
	return false
}
