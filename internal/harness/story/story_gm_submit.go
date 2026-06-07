package story

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) submitStoryInput(storyID string, u *Actor, currentTurnID int, idempotencyKey, choiceID, customMode, customText string) (string, error) {
	choiceID = strings.TrimSpace(choiceID)
	customMode = strings.TrimSpace(customMode)
	customText = strings.TrimSpace(customText)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	unlock, err := s.acquireLock(storyID, "submit_input", u.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
	if job, found, err := s.findJobByIdempotencyKey(storyID, "story_turn", idempotencyKey); err != nil {
		return "", err
	} else if found {
		if s.idempotencyMatchesStoryInput(job, u.ID, choiceID, customMode, customText, currentTurnID) {
			return job.ID, nil
		}
		return "", errors.New("idempotency key conflict")
	}
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	turns, err := s.readTurns(storyID)
	if err != nil || len(turns) == 0 {
		return "", errors.New("story has no turns")
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return "", errors.New("story is not waiting for input")
	}
	if u.Role != "admin" && (m.ActiveDriverID == "" || m.ActiveDriverID != u.ID) {
		return "", errors.New("only active driver can progress this story")
	}
	prev := turns[len(turns)-1]
	if currentTurnID != prev.TurnID {
		return "", errors.New("stale turn")
	}
	if choiceID == "" && customText == "" {
		return "", errors.New("input is empty")
	}
	if choiceID != "" {
		found := false
		for _, c := range prev.Choices {
			if c.ID == choiceID {
				found = true
				break
			}
		}
		if !found {
			return "", errors.New("invalid choice")
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	input := Input{ID: "input_" + randomID(), SelectedChoiceID: choiceID, CustomInputMode: customMode, CustomText: strings.TrimSpace(customText)}
	job := GMJob{
		ID: "job_" + randomID(), StoryID: storyID, JobType: "story_turn", Status: "queued", Attempt: 1,
		ActorID: u.ID, ActorRole: u.Role, Input: &input, TurnID: currentTurnID + 1, ParentTurnID: currentTurnID,
		ContextHash: storyContextHash(m, turns), CreatedAt: now, ExclusiveProgression: true, IdempotencyKey: idempotencyKey,
	}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "input_submitted", "at": now, "input": input, "actor_id": u.ID, "idempotency_key": idempotencyKey}); err != nil {
		return "", err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_created", "at": now, "job": job}); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job); err != nil {
		return "", err
	}
	m.Phase = "gm_generating"
	m.ActiveJobID = job.ID
	m.UpdatedAt = now
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return "", err
	}
	return job.ID, nil
}

func (s *Store) createStoryWithPrologueJob(actorID, title, style, characterName, traits string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := "story_" + randomID()
	name := firstNonEmpty(strings.TrimSpace(characterName), "새 인물")
	style = firstNonEmpty(strings.TrimSpace(style), "조사극")
	traits = strings.TrimSpace(traits)
	setup := Setup{Title: firstNonEmpty(strings.TrimSpace(title), name+"의 이야기"), Style: style, CharacterName: name, Traits: traits}
	location, _, _, facts, openThreads, risks, _ := luceraPrologueSeed(name, traits)
	job := GMJob{
		ID: "job_" + randomID(), StoryID: id, JobType: "prologue", Status: "queued", Attempt: 1,
		ActorID: actorID, ActorRole: "admin", Setup: &setup, TurnID: 1, ParentTurnID: 0,
		ContextHash: "sha256:setup", CreatedAt: now, ExclusiveProgression: true,
	}
	m := Manifest{ID: id, Title: setup.Title, WorldID: "lumen-federation", Status: "active", Phase: "gm_generating", CurrentTurn: 0, ActiveDriverID: actorID, ActiveJobID: job.ID, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now, LatestSummary: "프롤로그 생성 중"}
	st := State{Location: location, ActiveCharacters: []string{name}, Facts: facts, OpenThreads: openThreads, Risks: risks, Flags: []string{"runtime_story_created"}}
	dir := s.storyDir(id)
	if err := os.MkdirAll(filepath.Join(dir, "jobs"), 0o700); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "memory-cards"), 0o700); err != nil {
		return "", err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "story_created", "at": now, "story": m, "setup": setup}); err != nil {
		return "", err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_created", "at": now, "job": job}); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return "", err
	}
	return id, writeAtomic(filepath.Join(dir, "summary.md"), []byte(m.LatestSummary+"\n"))
}

func (s *Store) submitQuestionJob(storyID string, u *Actor, currentTurnID int, idempotencyKey, text string) (string, error) {
	text = strings.TrimSpace(text)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if text == "" {
		return "", errors.New("question is empty")
	}
	unlock, err := s.acquireLock(storyID, "submit_question", u.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
	if job, found, err := s.findJobByIdempotencyKey(storyID, "question_answer", idempotencyKey); err != nil {
		return "", err
	} else if found {
		if s.idempotencyMatchesQuestion(job, u.ID, text, currentTurnID) {
			return job.ID, nil
		}
		return "", errors.New("idempotency key conflict")
	}
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	if m.Status != "active" && m.Status != "paused" {
		return "", errors.New("questions are closed for this story")
	}
	if m.Phase == "gm_generating" || m.Phase == "validating_output" || m.Phase == "applying_turn" {
		return "", errors.New("GM is generating")
	}
	if currentTurnID != m.CurrentTurn {
		return "", errors.New("stale turn")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q := Question{ID: "question_" + randomID(), ActorID: u.ID, Question: text, Answer: "답변 생성 중", TurnID: currentTurnID, CreatedAt: now}
	job := GMJob{
		ID: "job_" + randomID(), StoryID: storyID, JobType: "question_answer", Status: "queued", Attempt: 1,
		ActorID: u.ID, ActorRole: u.Role, Question: &q, TurnID: currentTurnID, ParentTurnID: currentTurnID,
		ContextHash: "sha256:question", CreatedAt: now, ExclusiveProgression: false, IdempotencyKey: idempotencyKey,
	}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "question_asked", "at": now, "question": q, "idempotency_key": idempotencyKey}); err != nil {
		return "", err
	}
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "gm_job_created", "at": now, "job": job}); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job); err != nil {
		return "", err
	}
	return job.ID, appendJSONL(filepath.Join(dir, "qa.jsonl"), q)
}
