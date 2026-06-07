package harness

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

func wantsJSONResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	if strings.Contains(accept, "application/json") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Requested-With")), "XMLHttpRequest")
}

func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *webServer) writeStoryTaskError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if wantsJSONResponse(r) {
		writeJSONResponse(w, status, storyTaskErrorPayload(message))
		return
	}
	http.Error(w, message, status)
}

func (s *webServer) writeStoryTaskAccepted(w http.ResponseWriter, r *http.Request, storyID, jobType string, turnID int, jobID string, progress storyProgressView) {
	if jobID == "" {
		http.Error(w, "missing job id", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusAccepted, storyTaskAcceptedPayload(s.base(r), storyID, jobType, turnID, jobID, progress))
}

func (s *webServer) storyRoomProgressSnapshot(id string, m storyManifest, u *authUser) storyProgressView {
	progress := storyProgressView{
		StoryID:     id,
		Status:      m.Status,
		Phase:       m.Phase,
		CurrentTurn: m.CurrentTurn,
		CanDrive:    canDriveStory(m, u),
		CanQuestion: u != nil && canQuestionStory(m),
		StatusLabel: friendlyStatusLabel(m),
		StepIndex:   3,
		StepLabel:   "ready",
		NextPollMS:  0,
	}
	progress.ProgressMessage = storyProgressDefaultMessage(progress.CanQuestion, progress.CanDrive, u == nil)
	if progress.CanQuestion && !progress.CanDrive {
		progress.ProgressMessage = "질문은 현재 턴에 대해 보낼 수 있습니다."
	}
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		progress.HasProgressMeta = true
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			progress.ActiveJobID, progress.ActiveJobType, progress.ActiveJobStatus, progress.ActiveJobTurnID = job.ID, job.JobType, job.Status, job.TurnID
			progress.JobStartedAt, progress.JobCompletedAt, progress.JobErrorCode, progress.JobErrorMessage = job.StartedAt, job.CompletedAt, job.ErrorCode, job.ErrorMessage
			progress.ProgressMessage = "GM 작업이 실패했습니다. 복구 또는 취소가 필요합니다."
			progress.StepIndex, progress.StepLabel = 4, "failed"
		}
		progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
		return progress
	}
	if m.ActiveJobID != "" {
		progress.HasProgressMeta = true
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			progress.ActiveJobID, progress.ActiveJobType, progress.ActiveJobStatus, progress.ActiveJobTurnID = job.ID, job.JobType, job.Status, job.TurnID
			progress.JobStartedAt, progress.JobCompletedAt, progress.JobErrorCode, progress.JobErrorMessage = job.StartedAt, job.CompletedAt, job.ErrorCode, job.ErrorMessage
			progress.IsProcessing = job.Status == "queued" || job.Status == "running" || job.Status == "validating" || job.Status == "applying"
			progress.NextPollMS = 2500
			switch job.Status {
			case "queued":
				progress.StepIndex, progress.StepLabel, progress.ProgressMessage = 0, "queued", "작업이 대기열에 들어갔습니다. 잠시만 기다려 주세요."
			case "running":
				progress.StepIndex, progress.StepLabel, progress.ProgressMessage = 1, "generating", "GM이 장면을 생성 중입니다. 잠시만 기다려 주세요."
			case "validating", "applying":
				progress.StepIndex, progress.StepLabel, progress.ProgressMessage = 2, "applying", "생성 결과를 반영하는 중입니다."
			case "failed":
				progress.StepIndex, progress.StepLabel, progress.ProgressMessage = 4, "failed", "GM 작업이 실패했습니다. 복구 또는 취소가 필요합니다."
			default:
				progress.ProgressMessage = "GM 작업 상태를 확인 중입니다."
			}
			progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
			return progress
		}
	}
	pending := s.storyProgressPendingQuestions(id)
	if len(pending) > 0 {
		progress.PendingQuestions = pending
		progress.ActiveJobID, progress.ActiveJobType, progress.ActiveJobStatus, progress.ActiveJobTurnID = pending[0].JobID, "question_answer", pending[0].Status, pending[0].TurnID
		progress.IsProcessing, progress.NextPollMS = true, 2500
		switch pending[0].Status {
		case "queued":
			progress.StepIndex, progress.StepLabel = 0, "queued"
		case "running":
			progress.StepIndex, progress.StepLabel = 1, "generating"
		default:
			progress.StepIndex, progress.StepLabel = 1, "generating"
		}
		progress.ProgressMessage = "질문 답변을 준비 중입니다. 잠시만 기다려 주세요."
		progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
	}
	return progress
}

func storyTaskErrorPayload(message string) map[string]any {
	return map[string]any{
		"error": message,
	}
}

func storyTaskAcceptedPayload(base, storyID, jobType string, turnID int, jobID string, progress storyProgressView) map[string]any {
	return map[string]any{
		"story_id":          storyID,
		"job_id":            jobID,
		"job_type":          jobType,
		"turn_id":           firstNonZero(progress.ActiveJobTurnID, turnID),
		"status_url":        base + "/stories/" + url.PathEscape(storyID) + "/status",
		"next_poll_ms":      progress.NextPollMS,
		"status_label":      progress.StatusLabel,
		"progress_message":  progress.ProgressMessage,
		"step_index":        progress.StepIndex,
		"step_label":        progress.StepLabel,
		"is_processing":     progress.IsProcessing,
		"active_job_id":     progress.ActiveJobID,
		"active_job_type":   progress.ActiveJobType,
		"active_job_status": progress.ActiveJobStatus,
		"current_turn":      progress.CurrentTurn,
	}
}

func storyProgressDefaultMessage(canQuestion, canDrive, anonymous bool) string {
	switch {
	case anonymous:
		return "로그인하면 진행, 질문, 진행권, 관리 기능을 사용할 수 있습니다."
	case canQuestion && !canDrive:
		return "질문은 현재 턴에 대해 보낼 수 있습니다."
	default:
		return "대기 중입니다. 새 입력을 제출할 수 있는 상태입니다."
	}
}

func (s *webServer) storyProgressPendingQuestions(id string) []storyProgressQuestionView {
	jobs, err := s.stories.listJobs(id)
	if err != nil {
		return nil
	}
	out := make([]storyProgressQuestionView, 0, len(jobs))
	for _, job := range jobs {
		if job.JobType != "question_answer" || (job.Status != "queued" && job.Status != "running") {
			continue
		}
		text := ""
		if job.Question != nil {
			text = job.Question.Question
		}
		out = append(out, storyProgressQuestionView{JobID: job.ID, Status: job.Status, TurnID: job.TurnID, Question: text, CreatedAt: job.CreatedAt})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].JobID < out[j].JobID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}
