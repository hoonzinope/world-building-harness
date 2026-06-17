package story

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) startGMWorker(ctx context.Context, provider GMProvider) {
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = s.processOneGMJob(ctx, provider)
			}
		}
	}()
}

func (s *Store) processOneGMJob(ctx context.Context, provider GMProvider) error {
	stories, err := s.listStories()
	if err != nil {
		return err
	}
	for _, m := range stories {
		if m.ActiveJobID == "" || m.Phase != "gm_generating" {
			jobs, _ := s.listQueuedJobs(m.ID)
			for _, job := range jobs {
				if job.JobType == "question_answer" {
					return s.runGMJob(ctx, provider, job)
				}
			}
			continue
		}
		job, err := s.readJob(m.ID, m.ActiveJobID)
		if err == nil && job.Status == "queued" {
			return s.runGMJob(ctx, provider, job)
		}
	}
	return nil
}

func (s *Store) runGMJob(ctx context.Context, provider GMProvider, job GMJob) error {
	if err := s.markJobStarted(job); err != nil {
		return err
	}
	req, err := s.buildGMRequest(job)
	if err != nil {
		_ = s.failGMJob(job, "GM_CONTEXT_ERROR", err.Error())
		return err
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	out, raw, providerName, model, err := provider.Generate(timeoutCtx, req)
	if err != nil {
		if strings.TrimSpace(raw) != "" {
			rawPath, writeErr := s.writeFailedGMRaw(job, raw)
			if writeErr == nil {
				job.Provider = providerName
				job.Model = model
				job.RawOutputPath = rawPath
			}
		}
		_ = s.failGMJob(job, "GM_PROVIDER_ERROR", err.Error())
		return err
	}
	if err := s.applyGMOutput(job, out, raw, providerName, model); err != nil {
		_ = s.failGMJob(job, "GM_SCHEMA_INVALID", err.Error())
		return err
	}
	return nil
}

func (s *Store) markJobStarted(job GMJob) error {
	unlock, err := s.acquireLock(job.StoryID, "gm_started", "worker")
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(job.StoryID)
	if err != nil {
		return err
	}
	if job.ExclusiveProgression && (m.ActiveJobID != job.ID || m.Phase != "gm_generating") {
		return errors.New("job is no longer active")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	job.Status = "running"
	job.StartedAt = now
	dir := s.storyDir(job.StoryID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_started", "at": now, "job_id": job.ID}); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job)
}

func (s *Store) applyGMOutput(job GMJob, out GMOutput, raw, providerName, model string) error {
	if job.JobType == "question_answer" {
		return s.applyQuestionOutput(job, out, raw, providerName, model)
	}
	if err := ValidateGMOutput(job, out); err != nil {
		return err
	}
	unlock, err := s.acquireLock(job.StoryID, "gm_apply", "worker")
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(job.StoryID)
	if err != nil {
		return err
	}
	if m.ActiveJobID != job.ID || m.Phase != "gm_generating" {
		return errors.New("job is no longer active")
	}
	st, _ := s.readState(job.StoryID)
	now := time.Now().UTC().Format(time.RFC3339)
	turn := Turn{
		TurnID: job.TurnID, BranchID: out.Turn.BranchID, ParentTurnID: job.ParentTurnID, ActorID: job.ActorID,
		InputID: out.Turn.InputID, Source: outputSourceForJob(job),
		SceneGoal: out.SceneGoal, Conflict: out.Conflict, TurningPoint: out.TurningPoint, Consequence: out.Consequence,
		SceneTitle: out.SceneTitle, SceneBody: out.SceneBody, CurrentSituation: out.CurrentSituation,
		RevealedFacts: out.RevealedFacts, Choices: out.Choices, CreatedAt: now,
	}
	if job.Input != nil {
		turn.InputID = job.Input.ID
		turn.SelectedChoiceID = job.Input.SelectedChoiceID
		turn.CustomInputMode = job.Input.CustomInputMode
		turn.CustomText = job.Input.CustomText
	} else if job.Setup != nil {
		turn.CustomText = job.Setup.Traits
	}
	st = ApplyGMStatePatch(st, out.StatePatch, out.RevealedFacts)
	m.CurrentTurn = turn.TurnID
	m.Phase = "waiting_for_choice"
	m.ActiveJobID = ""
	m.UpdatedAt = now
	m.LatestSummary = firstNonEmpty(out.StatePatch.SummaryPatch, out.CurrentSituation)
	dir := s.storyDir(job.StoryID)
	rawPath := filepath.Join("jobs", job.ID+".raw.json")
	_ = writeAtomic(filepath.Join(dir, rawPath), []byte(raw))
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_validating", "at": now, "job_id": job.ID}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_applying", "at": now, "job_id": job.ID}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "turn_committed", "at": now, "turn": turn}); err != nil {
		return err
	}
	job.Status = "completed"
	job.Provider = providerName
	job.Model = model
	job.CompletedAt = now
	job.RawOutputPath = rawPath
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_completed", "at": now, "job": job}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "turns.jsonl"), turn); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "summary.md"), []byte(m.LatestSummary+"\n"))
}

func (s *Store) applyQuestionOutput(job GMJob, out GMOutput, raw, providerName, model string) error {
	if job.Question == nil {
		return errors.New("missing question")
	}
	if out.SchemaVersion != "story-question-answer.v1" || out.StoryID != job.StoryID || strings.TrimSpace(out.Answer) == "" {
		return errors.New("invalid question answer output")
	}
	unlock, err := s.acquireLock(job.StoryID, "question_apply", "worker")
	if err != nil {
		return err
	}
	defer unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	q := *job.Question
	q.Answer = strings.TrimSpace(out.Answer)
	dir := s.storyDir(job.StoryID)
	rawPath := filepath.Join("jobs", job.ID+".raw.json")
	_ = writeAtomic(filepath.Join(dir, rawPath), []byte(raw))
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_validating", "at": now, "job_id": job.ID}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "question_answered", "at": now, "question": q}); err != nil {
		return err
	}
	job.Status = "completed"
	job.Provider = providerName
	job.Model = model
	job.CompletedAt = now
	job.RawOutputPath = rawPath
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_completed", "at": now, "job": job}); err != nil {
		return err
	}
	if err := RewriteQuestionProjection(filepath.Join(dir, "qa.jsonl"), q); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job)
}

func (s *Store) failGMJob(job GMJob, code, message string) error {
	unlock, err := s.acquireLock(job.StoryID, "gm_failed", "worker")
	if err != nil {
		return err
	}
	defer unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	job.Status = "failed"
	job.ErrorCode = code
	job.ErrorMessage = message
	job.CompletedAt = now
	dir := s.storyDir(job.StoryID)
	m, _ := s.readManifest(job.StoryID)
	if job.ExclusiveProgression && m.ActiveJobID == job.ID {
		m.Phase = "failed_waiting_retry"
		m.UpdatedAt = now
		_ = writeJSONAtomic(filepath.Join(dir, "manifest.json"), m)
	}
	_ = appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_failed", "at": now, "job": job})
	return writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job)
}

func (s *Store) writeFailedGMRaw(job GMJob, raw string) (string, error) {
	dir := s.storyDir(job.StoryID)
	rawPath := filepath.Join("jobs", job.ID+".failed.txt")
	if err := writeAtomic(filepath.Join(dir, rawPath), []byte(raw)); err != nil {
		return "", err
	}
	return rawPath, nil
}

func (s *Store) buildGMRequest(job GMJob) (GMRequest, error) {
	m, err := s.readManifest(job.StoryID)
	if err != nil {
		return GMRequest{}, err
	}
	st, _ := s.readState(job.StoryID)
	turns, _ := s.readTurns(job.StoryID)
	if len(turns) > 8 {
		turns = turns[len(turns)-8:]
	}
	return GMRequest{Job: job, Manifest: m, State: st, Turns: turns, WorldContext: StoryWorldContextSeedForRequest(m, st, job)}, nil
}

func (s *Store) readJob(storyID, jobID string) (GMJob, error) {
	var j GMJob
	err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", jobID+".json"), &j)
	return j, err
}

func (s *Store) listQueuedJobs(storyID string) ([]GMJob, error) {
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
		if err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", e.Name()), &j); err == nil && j.Status == "queued" {
			out = append(out, j)
		}
	}
	return out, nil
}

func storyContextHash(m Manifest, turns []Turn) string {
	b, _ := json.Marshal(map[string]any{"manifest": m, "turns": turns})
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
