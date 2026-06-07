package harness

import (
	"net/http"
	"net/url"
	"strings"
)

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
