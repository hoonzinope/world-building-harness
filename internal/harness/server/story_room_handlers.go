package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

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
	s.render(w, r, m.Title, ui.StoryRoomTemplate, data)
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

func (s *webServer) handleStoryRoomAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(ui.StoryRoomAssetJS))
}
