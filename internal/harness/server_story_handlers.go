package harness

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
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
	s.render(w, r, "스토리", storyLobbyTemplate, data)
}

func (s *webServer) handleNewStory(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		s.render(w, r, "새 스토리", newStoryTemplate, map[string]any{"Base": s.base(r), "User": u, "CSRFToken": mustCSRFToken(w, r)})
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

func (s *webServer) handleStoryRoute(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "":
		s.renderStoryRoom(w, r, id)
	case "input":
		s.handleStoryInput(w, r, id)
	case "question":
		s.handleStoryQuestion(w, r, id)
	case "status":
		s.handleStoryStatus(w, r, id)
	case "driver":
		s.handleStoryDriver(w, r, id)
	case "admin":
		s.handleStoryAdmin(w, r, id)
	case "recover":
		s.handleStoryRecovery(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) renderStoryRoom(w http.ResponseWriter, r *http.Request, id string) {
	u := currentUser(r)
	m, err := s.stories.readManifest(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	st, _ := s.stories.readState(id)
	turns, _ := s.stories.readTurns(id)
	qa, _ := s.stories.readQA(id)
	hasTurns := len(turns) > 0
	displayTurns := append([]storyTurn(nil), turns...)
	sort.SliceStable(displayTurns, func(i, j int) bool { return displayTurns[i].TurnID > displayTurns[j].TurnID })
	latestTurnID := 0
	var latestTurn any
	previousTurns := make([]storyTurn, 0, len(displayTurns))
	if hasTurns {
		latestTurnID = displayTurns[0].TurnID
		latestTurn = displayTurns[0]
		if len(displayTurns) > 1 {
			previousTurns = append(previousTurns, displayTurns[1:]...)
		}
	}
	progress := s.storyRoomProgressSnapshot(id, m, u)
	isProcessing := progress.IsProcessing
	canDrive := canDriveStory(m, u) && !isProcessing
	canClaim := u != nil && m.ActiveDriverID == "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canRelease := u != nil && (u.Role == "admin" || u.ID == m.ActiveDriverID) && m.ActiveDriverID != "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canQuestion := u != nil && canQuestionStory(m) && !isProcessing
	canAdminMutate := isAdminUser(u) && hasTurns && !isProcessing
	driverLabel := friendlyDriverLabel(m, u)
	var failedJob *failedJobView
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			failedJob = &failedJobView{Job: job, CanRecover: isAdminUser(u) || (u != nil && u.ID == job.ActorID), ActorLabel: job.ActorID}
		}
	}
	data := storyRoomTemplateData(
		s.base(r),
		id,
		u,
		m,
		st,
		displayTurns,
		previousTurns,
		qa,
		canDrive,
		canClaim,
		canRelease,
		canQuestion,
		isAdminUser(u),
		canAdminMutate,
		latestTurnID,
		latestTurn,
		hasTurns,
		driverLabel,
		isProcessing,
		progress,
		failedJob,
		r.URL.Query(),
		mustCSRFToken(w, r),
	)
	s.render(w, r, m.Title, storyRoomTemplate, data)
}

func (s *webServer) handleStoryInput(w http.ResponseWriter, r *http.Request, id string) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := parseStoryTaskRequest(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireStoryTaskCSRF(w, r) {
		return
	}
	mode := strings.TrimSpace(r.FormValue("mode"))
	turnID := parseFormInt(r.FormValue("turn_id"))
	idem := strings.TrimSpace(r.FormValue("idempotency_key"))
	var jobID string
	var err error
	if mode == "question" {
		jobID, err = s.stories.submitQuestionJob(id, u, turnID, idem, strings.TrimSpace(r.FormValue("custom_text")))
	} else {
		jobID, err = s.stories.submitStoryInput(id, u, turnID, idem, r.FormValue("choice_id"), mode, strings.TrimSpace(r.FormValue("custom_text")))
	}
	if err != nil {
		s.writeStoryTaskError(w, r, err.Error(), http.StatusForbidden)
		return
	}
	if wantsJSONResponse(r) {
		m, readErr := s.stories.readManifest(id)
		if readErr != nil {
			s.writeStoryTaskError(w, r, readErr.Error(), http.StatusInternalServerError)
			return
		}
		progress := s.storyRoomProgressSnapshot(id, m, u)
		jobType := "story_turn"
		if mode == "question" {
			jobType = "question_answer"
		}
		s.writeStoryTaskAccepted(w, r, id, jobType, turnID, jobID, progress)
		return
	}
	fragment := "#input-panel"
	if mode == "question" {
		fragment = "#qa"
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+fragment, http.StatusSeeOther)
}

func (s *webServer) handleStoryQuestion(w http.ResponseWriter, r *http.Request, id string) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := parseStoryTaskRequest(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireStoryTaskCSRF(w, r) {
		return
	}
	turnID := parseFormInt(r.FormValue("turn_id"))
	if turnID == 0 {
		if m, err := s.stories.readManifest(id); err == nil {
			turnID = m.CurrentTurn
		}
	}
	question := firstNonEmpty(strings.TrimSpace(r.FormValue("question")), strings.TrimSpace(r.FormValue("custom_text")))
	jobID, err := s.stories.submitQuestionJob(id, u, turnID, strings.TrimSpace(r.FormValue("idempotency_key")), question)
	if err != nil {
		s.writeStoryTaskError(w, r, err.Error(), http.StatusForbidden)
		return
	}
	if wantsJSONResponse(r) {
		m, readErr := s.stories.readManifest(id)
		if readErr != nil {
			s.writeStoryTaskError(w, r, readErr.Error(), http.StatusInternalServerError)
			return
		}
		progress := s.storyRoomProgressSnapshot(id, m, u)
		s.writeStoryTaskAccepted(w, r, id, "question_answer", turnID, jobID, progress)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#qa", http.StatusSeeOther)
}
func (s *webServer) handleStoryStatus(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u := currentUser(r)
	m, err := s.stories.readManifest(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSONResponse(w, http.StatusOK, s.storyRoomProgressSnapshot(id, m, u))
}
func (s *webServer) handleStoryDriver(w http.ResponseWriter, r *http.Request, id string) {
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
	if err := s.stories.updateDriver(id, u, r.FormValue("action")); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}
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
func (s *webServer) handleStoryRecovery(w http.ResponseWriter, r *http.Request, id string) {
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
	case "resume":
		if _, err := s.stories.resumeFailedJob(id, u); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#input-panel", http.StatusSeeOther)
	case "cancel":
		if err := s.stories.cancelFailedJob(id, u); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#input-panel", http.StatusSeeOther)
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
	s.render(w, r, "Admin Users", adminUsersTemplate, data)
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

func storyRoomTemplateData(base, id string, u *authUser, m storyManifest, st storyState, displayTurns, previousTurns []storyTurn, qa []storyQuestion, canDrive, canClaim, canRelease, canQuestion, isAdmin, canAdminMutate bool, latestTurnID int, latestTurn any, hasTurns bool, driverLabel string, isProcessing bool, progress storyProgressView, failedJob *failedJobView, query url.Values, csrf string) map[string]any {
	return map[string]any{
		"Base":                base,
		"User":                u,
		"IsAnonymous":         u == nil,
		"Story":               m,
		"State":               st,
		"Turns":               displayTurns,
		"PreviousTurns":       previousTurns,
		"QA":                  qa,
		"CanDrive":            canDrive,
		"CanClaim":            canClaim,
		"CanRelease":          canRelease,
		"CanQuestion":         canQuestion,
		"IsAdmin":             isAdmin,
		"CanAdminMutate":      canAdminMutate,
		"LatestTurnID":        latestTurnID,
		"LatestTurn":          latestTurn,
		"HasTurns":            hasTurns,
		"DriverLabel":         driverLabel,
		"IsProcessing":        isProcessing,
		"Progress":            progress,
		"StatusURL":           base + "/stories/" + url.PathEscape(id) + "/status",
		"FailedJob":           failedJob,
		"ExportedBundle":      strings.TrimSpace(query.Get("exported")),
		"ExportedStatus":      strings.TrimSpace(query.Get("export_status")),
		"ExportDraftTarget":   strings.TrimSpace(query.Get("export_draft_target")),
		"RecoveryStatus":      strings.TrimSpace(query.Get("recovery_status")),
		"RecoveryMessage":     strings.TrimSpace(query.Get("recovery_message")),
		"RecoveryChecked":     queryCSV(query.Get("recovery_checked")),
		"RecoveryRepaired":    queryCSV(query.Get("recovery_repaired")),
		"RecoveryLockRemoved": strings.EqualFold(strings.TrimSpace(query.Get("recovery_lock_removed")), "true"),
		"CSRFToken":           csrf,
	}
}

func adminUsersTemplateData(base string, u *authUser, users any, csrf string) map[string]any {
	return map[string]any{
		"Base":      base,
		"User":      u,
		"Users":     users,
		"CSRFToken": csrf,
	}
}
func (s *webServer) handleStoryRoomAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(storyRoomAssetJS))
}
