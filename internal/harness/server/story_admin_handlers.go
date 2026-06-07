package server

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

func (s *webServer) handleStoryAdmin(w http.ResponseWriter, r *http.Request, id string) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
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
	switch r.FormValue("action") {
	case "update":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.adminUpdateStory(id, u.ID, r.FormValue("status"), r.FormValue("active_driver_id")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "edit_turn":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.editCurrentTurn(id, u.ID, r.FormValue("scene_body"), r.FormValue("current_situation")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "rollback_turn":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		turnID := parseFormInt(r.FormValue("turn_id"))
		if turnID <= 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.stories.rollbackStoryToTurn(id, u.ID, turnID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "archive":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.archiveStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "restore":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.restoreStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "delete":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.deleteStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "export_bundle":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		bundlePath, err := s.stories.exportStoryBundle(id, u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		draftTarget := filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md"))
		redirectURL := s.base(r) + "/stories/" + url.PathEscape(id) + "?exported=" + url.QueryEscape(bundlePath) + "&export_status=" + url.QueryEscape("draft_pending") + "&export_draft_target=" + url.QueryEscape(draftTarget)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	case "recover_store":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		report, err := s.stories.recoverStory(id)
		if err != nil {
			redirectURL := s.base(r) + "/stories/" + url.PathEscape(id) + "?recovery_status=failed&recovery_message=" + url.QueryEscape(err.Error())
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}
		values := url.Values{}
		values.Set("recovery_status", report.RecoveryStatus)
		values.Set("recovery_checked", strings.Join(report.CheckedFiles, ","))
		values.Set("recovery_repaired", strings.Join(report.RepairedItems, ","))
		values.Set("recovery_lock_removed", fmt.Sprint(report.LockRemoved))
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"?"+values.Encode(), http.StatusSeeOther)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
	}
}

func (s *webServer) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if !isAdminUser(u) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !s.requireCSRF(w, r) {
			return
		}
		switch r.FormValue("action") {
		case "create":
			if err := s.auth.createUser(r.FormValue("username"), r.FormValue("display_name"), r.FormValue("role"), r.FormValue("password")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "update":
			if err := s.auth.updateUser(r.FormValue("id"), r.FormValue("role"), r.FormValue("status")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "reset":
			if err := s.auth.resetPassword(r.FormValue("id"), r.FormValue("password")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "revoke":
			if err := s.auth.revokeUserSessions(r.FormValue("id")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		http.Redirect(w, r, s.base(r)+"/admin/users", http.StatusSeeOther)
		return
	}
	users, err := s.auth.listUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := adminUsersTemplateData(s.base(r), u, users, mustCSRFToken(w, r))
	s.render(w, r, "Admin Users", ui.AdminUsersTemplate, data)
}

func adminUsersTemplateData(base string, u *authUser, users any, csrf string) map[string]any {
	return map[string]any{
		"Base":      base,
		"User":      u,
		"Users":     users,
		"CSRFToken": csrf,
	}
}
