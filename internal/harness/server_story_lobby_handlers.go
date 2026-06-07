package harness

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

func (s *webServer) renderStoryLobby(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	stories, err := s.stories.listStories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter")))
	if filter == "" {
		filter = "all"
	}
	rows := []lobbyStoryRow{}
	for _, m := range stories {
		row := storyLobbyRowFromManifest(m, u)
		if !storyMatchesLobbyFilter(row, filter) {
			continue
		}
		rows = append(rows, row)
	}
	data := storyLobbyTemplateData(s.base(r), u, filter, rows, mustCSRFToken(w, r))
	s.render(w, r, "스토리", ui.StoryLobbyTemplate, data)
}

func (s *webServer) handleNewStory(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		s.render(w, r, "새 스토리", ui.NewStoryTemplate, map[string]any{"Base": s.base(r), "User": u, "CSRFToken": mustCSRFToken(w, r)})
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
	id, err := s.stories.createStoryWithPrologueJob(u.ID, r.FormValue("title"), r.FormValue("style"), r.FormValue("character_name"), r.FormValue("traits"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleImportHector(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _, err := s.stories.importHector(u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func storyLobbyRowFromManifest(m storyManifest, u *authUser) lobbyStoryRow {
	return lobbyStoryRow{
		ID:          m.ID,
		Title:       m.Title,
		Status:      m.Status,
		Phase:       m.Phase,
		Turn:        m.CurrentTurn,
		MetaLine:    storyLobbyMetaLine(m, u),
		Summary:     m.LatestSummary,
		Updated:     storyLobbyUpdatedAt(m.UpdatedAt),
		MetaLabels:  storyLobbyMetaLabels(m, u),
		Imported:    m.SourceDraftPath != "",
		IsMine:      u != nil && (m.CreatedBy == u.ID || m.ActiveDriverID == u.ID),
		IsWatch:     (m.Status == "active" || m.Status == "paused") && !canDriveStory(m, u),
		IsArchived:  m.Status == "completed" || m.Status == "archived" || m.Status == "deleted",
		IsActive:    m.Status == "active",
		CanDrive:    canDriveStory(m, u),
		DriverLabel: friendlyDriverLabel(m, u),
		Permission:  friendlyPermissionLabel(m, u),
	}
}

func storyLobbyTemplateData(base string, u *authUser, filter string, rows []lobbyStoryRow, csrf string) map[string]any {
	return map[string]any{
		"Base":        base,
		"User":        u,
		"IsAnonymous": u == nil,
		"Stories":     rows,
		"Filter":      filter,
		"CSRFToken":   csrf,
	}
}
