package harness

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type gmJob struct {
	ID                   string         `json:"id"`
	StoryID              string         `json:"story_id"`
	JobType              string         `json:"job_type"`
	Status               string         `json:"status"`
	Attempt              int            `json:"attempt"`
	ActorID              string         `json:"actor_id"`
	ActorRole            string         `json:"actor_role"`
	Input                *storyInput    `json:"input,omitempty"`
	Setup                *storySetup    `json:"setup,omitempty"`
	Question             *storyQuestion `json:"question,omitempty"`
	TurnID               int            `json:"turn_id"`
	ParentTurnID         int            `json:"parent_turn_id"`
	ContextHash          string         `json:"context_hash"`
	ErrorCode            string         `json:"error_code,omitempty"`
	ErrorMessage         string         `json:"error_message,omitempty"`
	Provider             string         `json:"provider,omitempty"`
	Model                string         `json:"model,omitempty"`
	CreatedAt            string         `json:"created_at"`
	StartedAt            string         `json:"started_at,omitempty"`
	CompletedAt          string         `json:"completed_at,omitempty"`
	RawOutputPath        string         `json:"raw_output_path,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ExclusiveProgression bool           `json:"exclusive_progression"`
}

type storyInput struct {
	ID               string `json:"id"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
	CustomText       string `json:"custom_text,omitempty"`
}

type storySetup struct {
	Title         string `json:"title"`
	Style         string `json:"style"`
	CharacterName string `json:"character_name"`
	Traits        string `json:"traits"`
}

type gmRequest struct {
	Job      gmJob         `json:"job"`
	Manifest storyManifest `json:"manifest"`
	State    storyState    `json:"state"`
	Turns    []storyTurn   `json:"recent_turns"`
}

type gmOutput struct {
	SchemaVersion    string              `json:"schema_version"`
	StoryID          string              `json:"story_id"`
	Turn             gmOutputTurn        `json:"turn"`
	SceneTitle       string              `json:"scene_title"`
	SceneBody        string              `json:"scene_body"`
	Answer           string              `json:"answer,omitempty"`
	CurrentSituation string              `json:"current_situation"`
	RevealedFacts    []string            `json:"revealed_facts"`
	StatePatch       gmStatePatch        `json:"state_patch"`
	Resolution       string              `json:"resolution"`
	Choices          []storyChoice       `json:"choices"`
	GMNotes          map[string][]string `json:"gm_notes,omitempty"`
}

type gmOutputTurn struct {
	BranchID         string `json:"branch_id"`
	TurnID           int    `json:"turn_id"`
	ParentTurnID     int    `json:"parent_turn_id"`
	InputID          string `json:"input_id"`
	JobID            string `json:"job_id"`
	Source           string `json:"source"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
}

type gmStatePatch struct {
	LocationSet         string   `json:"location_set,omitempty"`
	ActiveCharactersSet []string `json:"active_characters_set,omitempty"`
	FactsAdd            []string `json:"facts_add,omitempty"`
	FactsRemove         []string `json:"facts_remove,omitempty"`
	OpenThreadsAdd      []string `json:"open_threads_add,omitempty"`
	OpenThreadsResolve  []string `json:"open_threads_resolve,omitempty"`
	RisksAdd            []string `json:"risks_add,omitempty"`
	RisksRemove         []string `json:"risks_remove,omitempty"`
	FlagsAdd            []string `json:"flags_add,omitempty"`
	FlagsRemove         []string `json:"flags_remove,omitempty"`
	SummaryPatch        string   `json:"summary_patch,omitempty"`
}

type gmProvider interface {
	Generate(context.Context, gmRequest) (gmOutput, string, string, string, error)
}

func newGMProvider(name string) gmProvider {
	switch strings.TrimSpace(name) {
	case "codex_cli":
		return codexCLIProvider{}
	default:
		return mockGMProvider{}
	}
}

func (s *storyStore) submitStoryInput(storyID string, u *authUser, choiceID, customMode, customText string) (string, error) {
	unlock, err := s.acquireLock(storyID, "submit_input", u.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return "", errors.New("story is not waiting for input")
	}
	if u.Role != "admin" && (m.ActiveDriverID == "" || m.ActiveDriverID != u.ID) {
		return "", errors.New("only active driver can progress this story")
	}
	turns, err := s.readTurns(storyID)
	if err != nil || len(turns) == 0 {
		return "", errors.New("story has no turns")
	}
	prev := turns[len(turns)-1]
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
	input := storyInput{ID: "input_" + randomID(), SelectedChoiceID: choiceID, CustomInputMode: customMode, CustomText: strings.TrimSpace(customText)}
	job := gmJob{
		ID: "job_" + randomID(), StoryID: storyID, JobType: "story_turn", Status: "queued", Attempt: 1,
		ActorID: u.ID, ActorRole: u.Role, Input: &input, TurnID: prev.TurnID + 1, ParentTurnID: prev.TurnID,
		ContextHash: storyContextHash(m, turns), CreatedAt: now, ExclusiveProgression: true,
	}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "input_submitted", "at": now, "input": input, "actor_id": u.ID}); err != nil {
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

func (s *storyStore) createStoryWithPrologueJob(actorID, title, style, characterName, traits string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := "story_" + randomID()
	name := firstNonEmpty(strings.TrimSpace(characterName), "새 인물")
	style = firstNonEmpty(strings.TrimSpace(style), "조사극")
	traits = strings.TrimSpace(traits)
	setup := storySetup{Title: firstNonEmpty(strings.TrimSpace(title), name+"의 이야기"), Style: style, CharacterName: name, Traits: traits}
	job := gmJob{
		ID: "job_" + randomID(), StoryID: id, JobType: "prologue", Status: "queued", Attempt: 1,
		ActorID: actorID, ActorRole: "admin", Setup: &setup, TurnID: 1, ParentTurnID: 0,
		ContextHash: "sha256:setup", CreatedAt: now, ExclusiveProgression: true,
	}
	m := storyManifest{ID: id, Title: setup.Title, WorldID: "lumen-federation", Status: "active", Phase: "gm_generating", CurrentTurn: 0, ActiveDriverID: actorID, ActiveJobID: job.ID, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now, LatestSummary: "프롤로그 생성 중"}
	st := storyState{Location: "미정", ActiveCharacters: []string{name}, Facts: []string{"주인공은 " + name + "이다.", "아직 canon이 아닌 runtime story 상태다."}, OpenThreads: []string{"프롤로그 생성"}, Risks: []string{"GM 생성 결과 검증 전까지 장면은 확정되지 않았다."}, Flags: []string{"runtime_story_created"}}
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

func (s *storyStore) submitQuestionJob(storyID string, u *authUser, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("question is empty")
	}
	unlock, err := s.acquireLock(storyID, "submit_question", u.ID)
	if err != nil {
		return "", err
	}
	defer unlock()
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
	now := time.Now().UTC().Format(time.RFC3339)
	q := storyQuestion{ID: "question_" + randomID(), ActorID: u.ID, Question: text, Answer: "답변 생성 중", TurnID: m.CurrentTurn, CreatedAt: now}
	job := gmJob{
		ID: "job_" + randomID(), StoryID: storyID, JobType: "question_answer", Status: "queued", Attempt: 1,
		ActorID: u.ID, ActorRole: u.Role, Question: &q, TurnID: m.CurrentTurn, ParentTurnID: m.CurrentTurn,
		ContextHash: "sha256:question", CreatedAt: now, ExclusiveProgression: false,
	}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "question_asked", "at": now, "question": q}); err != nil {
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

func (s *storyStore) startGMWorker(ctx context.Context, provider gmProvider) {
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

func (s *storyStore) processOneGMJob(ctx context.Context, provider gmProvider) error {
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

func (s *storyStore) runGMJob(ctx context.Context, provider gmProvider, job gmJob) error {
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
		_ = s.failGMJob(job, "GM_PROVIDER_ERROR", err.Error())
		return err
	}
	if err := s.applyGMOutput(job, out, raw, providerName, model); err != nil {
		_ = s.failGMJob(job, "GM_SCHEMA_INVALID", err.Error())
		return err
	}
	return nil
}

func (s *storyStore) markJobStarted(job gmJob) error {
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

func (s *storyStore) applyGMOutput(job gmJob, out gmOutput, raw, providerName, model string) error {
	if job.JobType == "question_answer" {
		return s.applyQuestionOutput(job, out, raw, providerName, model)
	}
	if err := validateGMOutput(job, out); err != nil {
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
	turn := storyTurn{
		TurnID: job.TurnID, BranchID: out.Turn.BranchID, ParentTurnID: job.ParentTurnID, ActorID: job.ActorID,
		InputID: out.Turn.InputID, Source: out.Turn.Source,
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
	st = applyGMStatePatch(st, out.StatePatch, out.RevealedFacts)
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

func (s *storyStore) applyQuestionOutput(job gmJob, out gmOutput, raw, providerName, model string) error {
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
	if err := rewriteQuestionProjection(filepath.Join(dir, "qa.jsonl"), q); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, "jobs", job.ID+".json"), job)
}

func (s *storyStore) failGMJob(job gmJob, code, message string) error {
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

func (s *storyStore) buildGMRequest(job gmJob) (gmRequest, error) {
	m, err := s.readManifest(job.StoryID)
	if err != nil {
		return gmRequest{}, err
	}
	st, _ := s.readState(job.StoryID)
	turns, _ := s.readTurns(job.StoryID)
	if len(turns) > 5 {
		turns = turns[len(turns)-5:]
	}
	return gmRequest{Job: job, Manifest: m, State: st, Turns: turns}, nil
}

func (s *storyStore) readJob(storyID, jobID string) (gmJob, error) {
	var j gmJob
	err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", jobID+".json"), &j)
	return j, err
}

func (s *storyStore) listQueuedJobs(storyID string) ([]gmJob, error) {
	entries, err := os.ReadDir(filepath.Join(s.storyDir(storyID), "jobs"))
	if err != nil {
		return nil, err
	}
	var out []gmJob
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var j gmJob
		if err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", e.Name()), &j); err == nil && j.Status == "queued" {
			out = append(out, j)
		}
	}
	return out, nil
}

func storyContextHash(m storyManifest, turns []storyTurn) string {
	b, _ := json.Marshal(map[string]any{"manifest": m, "turns": turns})
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type mockGMProvider struct{}

func (mockGMProvider) Generate(ctx context.Context, req gmRequest) (gmOutput, string, string, string, error) {
	if req.Job.JobType == "question_answer" {
		return mockQuestionOutput(req)
	}
	if req.Job.JobType == "prologue" {
		return mockPrologueOutput(req)
	}
	prev := req.Turns[len(req.Turns)-1]
	st := req.State
	input := firstNonEmpty(req.Job.Input.SelectedChoiceID, req.Job.Input.CustomText, "직접 행동")
	scene := generateLocalGMScene(prev, st, input, req.Job.Input.CustomInputMode)
	out := gmOutput{
		SchemaVersion: "story-gm-output.v1", StoryID: req.Job.StoryID,
		Turn:       gmOutputTurn{BranchID: "branch_main", TurnID: req.Job.TurnID, ParentTurnID: req.Job.ParentTurnID, InputID: req.Job.Input.ID, JobID: req.Job.ID, Source: "choice", SelectedChoiceID: req.Job.Input.SelectedChoiceID, CustomInputMode: req.Job.Input.CustomInputMode},
		SceneTitle: fmt.Sprintf("Turn %d의 여파", req.Job.TurnID), SceneBody: scene,
		CurrentSituation: generateCurrentSituation(st), RevealedFacts: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."},
		StatePatch: gmStatePatch{FactsAdd: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."}, OpenThreadsAdd: []string{"방금 선택의 절차적 후속 근거 확보"}, SummaryPatch: generateCurrentSituation(st)},
		Resolution: "accepted", Choices: generateNextChoices(req.Job.TurnID, st),
	}
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}

func mockQuestionOutput(req gmRequest) (gmOutput, string, string, string, error) {
	if req.Job.Question == nil {
		return gmOutput{}, "", "mock", "mock-story-gm", errors.New("missing question")
	}
	answer := "Turn 기준으로 답하면, " + summarizeQuestionAnswer(req.Job.Question.Question, req.State)
	out := gmOutput{SchemaVersion: "story-question-answer.v1", StoryID: req.Job.StoryID, Answer: answer}
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}

func mockPrologueOutput(req gmRequest) (gmOutput, string, string, string, error) {
	setup := req.Job.Setup
	if setup == nil {
		return gmOutput{}, "", "mock", "mock-story-gm", errors.New("missing setup")
	}
	name := firstNonEmpty(setup.CharacterName, "새 인물")
	subject := name + "는"
	location := "루세라 야간 진료동"
	scene := fmt.Sprintf("루세라의 야간 진료동은 새벽이 가까워질수록 더 조용해지지 않았다. 대기실의 의자는 이미 부족했고, 처치실 문틈으로는 젖은 소독포 냄새와 낮은 신음이 번갈아 밀려나왔다.\n\n%s 간호기록판을 팔에 끼운 채 잠깐 멈춰 섰다. 손가락 사이에는 늘 챙기던 펜이 있었고, 안경 너머의 눈 밑에는 오래 씻지 못한 피로가 그대로 남아 있었다. 방금 들어온 환자 셋의 기록은 서로 다른 증상을 말하고 있었지만, 병동의 빈 침상 수는 같은 답만 내놓았다. 더 받을 수 없다.\n\n그때 접수대 쪽에서 누군가 %s의 이름을 불렀다. 새 환자 하나가 쓰러졌고, 동시에 이미 누워 있던 아이의 보호자가 약속된 처치를 왜 미루냐고 묻기 시작했다. 둘 다 기다릴 수 없지만, %s의 손은 하나뿐이다.", subject, name, name)
	if setup.Traits != "" {
		scene += "\n\n초기 설정 메모: " + setup.Traits
	}
	summary := fmt.Sprintf("%s에서 %s 동시에 밀려든 두 긴급 상황 앞에 섰다.", location, subject)
	facts := []string{"주인공은 " + name + "이다.", subject + " 루세라의 간호사다.", "초기 배경은 " + location + "이다.", "아직 canon이 아닌 runtime story 상태다."}
	if setup.Traits != "" {
		facts = append(facts, "초기 설정: "+setup.Traits)
	}
	choices := []storyChoice{{ID: "A", Text: "새로 쓰러진 환자의 상태를 직접 확인한다.", RiskHint: "즉시 위험을 볼 수 있지만 기존 처치가 더 밀린다."}, {ID: "B", Text: "기존 아이 환자의 처치를 먼저 이어간다.", RiskHint: "약속된 처치를 지키지만 새 환자를 놓칠 수 있다."}, {ID: "C", Text: "보호자에게 짧게 설명하고 동료를 호출한다.", RiskHint: "시간을 벌 수 있지만 항의가 커질 수 있다."}, {ID: "D", Text: "기록판과 펜으로 우선순위를 빠르게 다시 계산한다.", RiskHint: "근거는 남지만 현장 반응이 늦어진다."}}
	out := gmOutput{
		SchemaVersion: "story-gm-output.v1", StoryID: req.Job.StoryID,
		Turn:             gmOutputTurn{BranchID: "branch_main", TurnID: 1, ParentTurnID: 0, InputID: "setup_" + req.Job.ID, JobID: req.Job.ID, Source: "setup"},
		SceneTitle:       firstNonEmpty(setup.Style, "조사극") + "의 시작",
		SceneBody:        scene,
		CurrentSituation: summary,
		RevealedFacts:    facts,
		StatePatch:       gmStatePatch{LocationSet: location, ActiveCharactersSet: []string{name}, FactsAdd: facts, OpenThreadsAdd: []string{"새 환자와 기존 환자 중 누구를 먼저 살릴지 결정", "병동의 부족한 자원을 어떻게 배분할지 판단"}, RisksAdd: []string{"어느 쪽을 선택해도 다른 쪽의 상태가 악화될 수 있다.", "과로와 환경 압박으로 판단 여력이 흔들릴 수 있다."}, SummaryPatch: summary},
		Resolution:       "accepted",
		Choices:          choices,
	}
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}

type codexCLIProvider struct{}

func (codexCLIProvider) Generate(ctx context.Context, req gmRequest) (gmOutput, string, string, string, error) {
	work := filepath.Join(os.TempDir(), "world-harness-gm-"+req.Job.ID)
	if err := os.MkdirAll(work, 0o700); err != nil {
		return gmOutput{}, "", "", "", err
	}
	contextPath := filepath.Join(work, "context.json")
	b, _ := json.MarshalIndent(req, "", "  ")
	if err := os.WriteFile(contextPath, b, 0o600); err != nil {
		return gmOutput{}, "", "", "", err
	}
	prompt := buildCodexGMPrompt(req, contextPath)
	outputPath := filepath.Join(work, "output.json")
	cmd := exec.CommandContext(ctx, "codex", "exec", "-C", work, "--add-dir", filepath.Join(os.Getenv("WORLD_HARNESS_PACKS_ROOT"), "lumen-federation"), "--sandbox", "read-only", "--skip-git-repo-check", "--ephemeral", "--output-last-message", outputPath, prompt)
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"CODEX_HOME=" + firstNonEmpty(os.Getenv("CODEX_HOME"), filepath.Join(os.Getenv("HOME"), ".codex")),
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"WORLD_HARNESS_PACKS_ROOT=" + os.Getenv("WORLD_HARNESS_PACKS_ROOT"),
		"PATH=" + os.Getenv("PATH"),
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return gmOutput{}, stdout.String(), "codex_cli", "", fmt.Errorf("%w: %s", err, tail(stderr.String(), 1200))
	}
	rawBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return gmOutput{}, stdout.String(), "codex_cli", "", err
	}
	raw := strings.TrimSpace(string(rawBytes))
	var out gmOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return gmOutput{}, raw, "codex_cli", "", err
	}
	return out, raw, "codex_cli", "codex-cli", nil
}

func buildCodexGMPrompt(req gmRequest, contextPath string) string {
	if req.Job.JobType == "question_answer" {
		return fmt.Sprintf(`You are the question-answer GM worker for world-harness Story Web UI.

Read the job context JSON at %s. Answer the user's non-progressing story question using only current story state, recent turns, and read-only world documents under the added pack directory if needed.

Do not advance the story. Do not change choices, state, summary, canon, or files.

Return exactly one JSON object. Do not use Markdown fences. Do not include explanations outside JSON.

Required schema:
{
  "schema_version": "story-question-answer.v1",
  "story_id": %q,
  "answer": "Korean answer, concise but useful"
}`, contextPath, req.Job.StoryID)
	}
	inputID := ""
	source := "choice or custom"
	if req.Job.Input != nil {
		inputID = req.Job.Input.ID
	} else {
		inputID = "setup_" + req.Job.ID
		source = "setup"
	}
	return fmt.Sprintf(`You are the GM worker for world-harness Story Web UI.

Read the job context JSON at %s. Use only the provided context and read-only world documents under the added pack directory if needed.

Return exactly one JSON object. Do not use Markdown fences. Do not include explanations outside JSON.

Required schema:
- schema_version: "story-gm-output.v1"
- story_id: %q
- turn.branch_id: "branch_main"
- turn.turn_id: %d
- turn.parent_turn_id: %d
- turn.input_id: %q
- turn.job_id: %q
- turn.source: %q
- scene_title, scene_body, current_situation: Korean strings
- revealed_facts: Korean string array
- state_patch: object with allowed add/set/remove fields
- resolution: "accepted", "partial", or "rejected"
- choices: 3 or 4 choices with id A-D, text, intent, risk_hint

The output should be interactive literary Korean prose, 1500-3000 Korean characters when possible. Do not change canon or files.`, contextPath, req.Job.StoryID, req.Job.TurnID, req.Job.ParentTurnID, inputID, req.Job.ID, source)
}

func validateGMOutput(job gmJob, out gmOutput) error {
	if out.SchemaVersion != "story-gm-output.v1" {
		return errors.New("schema_version mismatch")
	}
	expectedInputID := out.Turn.InputID
	if job.Input != nil {
		expectedInputID = job.Input.ID
	}
	if out.StoryID != job.StoryID || out.Turn.TurnID != job.TurnID || out.Turn.ParentTurnID != job.ParentTurnID || out.Turn.InputID != expectedInputID || out.Turn.JobID != job.ID {
		return errors.New("lineage mismatch")
	}
	if strings.TrimSpace(out.SceneBody) == "" || strings.TrimSpace(out.SceneTitle) == "" || strings.TrimSpace(out.CurrentSituation) == "" {
		return errors.New("empty required scene field")
	}
	if len(out.Choices) < 3 || len(out.Choices) > 4 {
		return errors.New("choices must be 3 or 4")
	}
	seen := map[string]bool{}
	for _, c := range out.Choices {
		if c.ID == "" || c.Text == "" || seen[c.ID] {
			return errors.New("invalid choice")
		}
		seen[c.ID] = true
	}
	return nil
}

func applyGMStatePatch(st storyState, p gmStatePatch, revealed []string) storyState {
	if p.LocationSet != "" {
		st.Location = p.LocationSet
	}
	if len(p.ActiveCharactersSet) > 0 {
		st.ActiveCharacters = p.ActiveCharactersSet
	}
	st.Facts = removeStrings(appendUnique(st.Facts, revealed...), p.FactsRemove...)
	st.Facts = appendUnique(st.Facts, p.FactsAdd...)
	st.OpenThreads = removeStrings(st.OpenThreads, p.OpenThreadsResolve...)
	st.OpenThreads = appendUnique(st.OpenThreads, p.OpenThreadsAdd...)
	st.Risks = removeStrings(st.Risks, p.RisksRemove...)
	st.Risks = appendUnique(st.Risks, p.RisksAdd...)
	st.Flags = removeStrings(st.Flags, p.FlagsRemove...)
	st.Flags = appendUnique(st.Flags, p.FlagsAdd...)
	return st
}

func removeStrings(in []string, vals ...string) []string {
	remove := map[string]bool{}
	for _, v := range vals {
		remove[strings.TrimSpace(v)] = true
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !remove[strings.TrimSpace(v)] {
			out = append(out, v)
		}
	}
	return out
}

func rewriteQuestionProjection(path string, updated storyQuestion) error {
	var qs []storyQuestion
	_ = readJSONL(path, func(b []byte) error {
		var q storyQuestion
		if err := json.Unmarshal(b, &q); err != nil {
			return err
		}
		if q.ID == updated.ID {
			q = updated
		}
		qs = append(qs, q)
		return nil
	})
	found := false
	for i := range qs {
		if qs[i].ID == updated.ID {
			qs[i] = updated
			found = true
		}
	}
	if !found {
		qs = append(qs, updated)
	}
	var b strings.Builder
	for _, q := range qs {
		line, _ := json.Marshal(q)
		b.Write(line)
		b.WriteByte('\n')
	}
	return writeAtomic(path, []byte(b.String()))
}
