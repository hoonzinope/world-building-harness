package story

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) listJobs(storyID string) ([]GMJob, error) {
	entries, err := os.ReadDir(filepath.Join(s.storyDir(storyID), "jobs"))
	if err != nil {
		return nil, err
	}
	var out []GMJob
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var j GMJob
		if err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", e.Name()), &j); err == nil {
			out = append(out, j)
		}
	}
	return out, nil
}

func (s *Store) findJobByIdempotencyKey(storyID, jobType, key string) (GMJob, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return GMJob{}, false, nil
	}
	jobs, err := s.listJobs(storyID)
	if err != nil {
		return GMJob{}, false, err
	}
	for _, job := range jobs {
		if job.IdempotencyKey == key && (jobType == "" || job.JobType == jobType) {
			return job, true, nil
		}
	}
	return GMJob{}, false, nil
}

func (s *Store) idempotencyMatchesStoryInput(job GMJob, actorID, choiceID, customMode, customText string, turnID int) bool {
	if job.ActorID != actorID || job.JobType != "story_turn" || job.TurnID != turnID || job.Input == nil {
		return false
	}
	return job.Input.SelectedChoiceID == choiceID && job.Input.CustomInputMode == customMode && job.Input.CustomText == strings.TrimSpace(customText)
}

func (s *Store) idempotencyMatchesQuestion(job GMJob, actorID, question string, turnID int) bool {
	if job.ActorID != actorID || job.JobType != "question_answer" || job.TurnID != turnID || job.Question == nil {
		return false
	}
	return job.Question.Question == strings.TrimSpace(question)
}

func (s *Store) failedProgressionJob(storyID string) (GMJob, bool, error) {
	m, err := s.readManifest(storyID)
	if err != nil {
		return GMJob{}, false, err
	}
	if m.Phase != "failed_waiting_retry" || m.ActiveJobID == "" {
		return GMJob{}, false, nil
	}
	job, err := s.readJob(storyID, m.ActiveJobID)
	if err != nil {
		return GMJob{}, false, err
	}
	return job, true, nil
}

func (s *Store) resumeFailedJob(storyID string, u *Actor) (string, error) {
	unlock, err := s.acquireLock(storyID, "resume_failed_job", u.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	if m.Phase != "failed_waiting_retry" || m.ActiveJobID == "" {
		return "", errors.New("no failed job to resume")
	}
	job, err := s.readJob(storyID, m.ActiveJobID)
	if err != nil {
		return "", err
	}
	if u.Role != "admin" && u.ID != job.ActorID {
		return "", errors.New("not allowed to resume this job")
	}
	if job.JobType != "story_turn" && job.JobType != "prologue" {
		return "", errors.New("failed job cannot be resumed")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	newJob := job
	newJob.ID = "job_" + randomID()
	newJob.Status = "queued"
	newJob.Attempt++
	newJob.CreatedAt = now
	newJob.StartedAt = ""
	newJob.CompletedAt = ""
	newJob.ErrorCode = ""
	newJob.ErrorMessage = ""
	newJob.Provider = ""
	newJob.Model = ""
	newJob.RawOutputPath = ""
	newJob.IdempotencyKey = ""
	newJob.ContextHash = storyContextHash(m, mustReadTurns(s, storyID))
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_resumed", "at": now, "job": newJob, "from_job_id": job.ID}); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "jobs", newJob.ID+".json"), newJob); err != nil {
		return "", err
	}
	m.Phase = "gm_generating"
	m.ActiveJobID = newJob.ID
	m.UpdatedAt = now
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return "", err
	}
	return newJob.ID, nil
}

func (s *Store) cancelFailedJob(storyID string, u *Actor) error {
	unlock, err := s.acquireLock(storyID, "cancel_failed_job", u.ID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.Phase != "failed_waiting_retry" || m.ActiveJobID == "" {
		return errors.New("no failed job to cancel")
	}
	job, err := s.readJob(storyID, m.ActiveJobID)
	if err != nil {
		return err
	}
	if u.Role != "admin" && u.ID != job.ActorID {
		return errors.New("not allowed to cancel this job")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_canceled", "at": now, "job_id": job.ID, "actor_id": u.ID}); err != nil {
		return err
	}
	m.Phase = "waiting_for_choice"
	m.ActiveJobID = ""
	m.UpdatedAt = now
	return writeJSONAtomic(filepath.Join(dir, "manifest.json"), m)
}

func (s *Store) storyHasBlockingGMJob(m Manifest) bool {
	switch m.Phase {
	case "gm_generating", "validating_output", "applying_turn":
		return true
	}
	if m.ActiveJobID == "" {
		return false
	}
	job, err := s.readJob(m.ID, m.ActiveJobID)
	if err != nil {
		return true
	}
	switch job.Status {
	case "queued", "running", "validating", "applying":
		return true
	default:
		return false
	}
}
