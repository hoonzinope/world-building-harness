package harness

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type storyStore struct {
	root       string
	packsRoot  string
	exportRoot string
}

const storyLockTimeout = 10 * time.Minute

type storyManifest struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	WorldID         string `json:"world_id"`
	Status          string `json:"status"`
	Phase           string `json:"phase"`
	CurrentTurn     int    `json:"current_turn"`
	ActiveDriverID  string `json:"active_driver_id"`
	ActiveJobID     string `json:"active_job_id,omitempty"`
	SourceDraftPath string `json:"source_draft_path,omitempty"`
	SourceHash      string `json:"source_hash,omitempty"`
	CreatedBy       string `json:"created_by"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LatestSummary   string `json:"latest_summary"`
}

type storyState struct {
	Location         string   `json:"location"`
	ActiveCharacters []string `json:"active_characters"`
	Facts            []string `json:"facts"`
	OpenThreads      []string `json:"open_threads"`
	Risks            []string `json:"risks"`
	Flags            []string `json:"flags"`
}

type storyChoice struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Intent   string `json:"intent,omitempty"`
	RiskHint string `json:"risk_hint,omitempty"`
}

type storyTurn struct {
	TurnID           int           `json:"turn_id"`
	BranchID         string        `json:"branch_id"`
	ParentTurnID     int           `json:"parent_turn_id"`
	ActorID          string        `json:"actor_id"`
	InputID          string        `json:"input_id"`
	Source           string        `json:"source"`
	SelectedChoiceID string        `json:"selected_choice_id,omitempty"`
	CustomInputMode  string        `json:"custom_input_mode,omitempty"`
	CustomText       string        `json:"custom_text,omitempty"`
	SceneTitle       string        `json:"scene_title"`
	SceneBody        string        `json:"scene_body"`
	CurrentSituation string        `json:"current_situation"`
	RevealedFacts    []string      `json:"revealed_facts"`
	Choices          []storyChoice `json:"choices"`
	CreatedAt        string        `json:"created_at"`
}

type storyQuestion struct {
	ID        string `json:"id"`
	ActorID   string `json:"actor_id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	TurnID    int    `json:"turn_id"`
	CreatedAt string `json:"created_at"`
}

func openStoryStore(root, packsRoot string) (*storyStore, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &storyStore{root: root, packsRoot: packsRoot, exportRoot: filepath.Join(filepath.Dir(root), "exports")}, nil
}

func (s *storyStore) storyDir(id string) string {
	return filepath.Join(s.root, id)
}

func (s *storyStore) listStories() ([]storyManifest, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var out []storyManifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := s.readManifest(e.Name())
		if err == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	out = dedupeStoryManifests(out)
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

func dedupeStoryManifests(stories []storyManifest) []storyManifest {
	bySource := map[string]int{}
	out := make([]storyManifest, 0, len(stories))
	for _, m := range stories {
		sourcePath := strings.TrimSpace(m.SourceDraftPath)
		if sourcePath == "" {
			out = append(out, m)
			continue
		}
		if idx, ok := bySource[sourcePath]; ok {
			if betterStoryManifest(m, out[idx]) {
				out[idx] = m
			}
			continue
		}
		bySource[sourcePath] = len(out)
		out = append(out, m)
	}
	return out
}

func betterStoryManifest(candidate, current storyManifest) bool {
	candidateDeleted := storyManifestIsDeleted(candidate)
	currentDeleted := storyManifestIsDeleted(current)
	if candidateDeleted != currentDeleted {
		return !candidateDeleted && currentDeleted
	}
	if candidate.UpdatedAt != current.UpdatedAt {
		return candidate.UpdatedAt > current.UpdatedAt
	}
	if candidate.CreatedAt != current.CreatedAt {
		return candidate.CreatedAt > current.CreatedAt
	}
	return candidate.ID > current.ID
}

func storyManifestIsDeleted(m storyManifest) bool {
	switch m.Status {
	case "deleted", "archived", "completed":
		return true
	default:
		return false
	}
}

func (s *storyStore) readManifest(id string) (storyManifest, error) {
	var m storyManifest
	err := readJSON(filepath.Join(s.storyDir(id), "manifest.json"), &m)
	return m, err
}

func (s *storyStore) readState(id string) (storyState, error) {
	var st storyState
	err := readJSON(filepath.Join(s.storyDir(id), "state.json"), &st)
	return st, err
}

func (s *storyStore) readTurns(id string) ([]storyTurn, error) {
	var turns []storyTurn
	err := readStoryJSONL(filepath.Join(s.storyDir(id), "turns.jsonl"), func(b []byte) error {
		var t storyTurn
		if err := json.Unmarshal(b, &t); err != nil {
			return err
		}
		turns = append(turns, t)
		return nil
	})
	return turns, err
}

func (s *storyStore) readQA(id string) ([]storyQuestion, error) {
	var qa []storyQuestion
	err := readStoryJSONL(filepath.Join(s.storyDir(id), "qa.jsonl"), func(b []byte) error {
		var q storyQuestion
		if err := json.Unmarshal(b, &q); err != nil {
			return err
		}
		qa = append(qa, q)
		return nil
	})
	return qa, err
}

func (s *storyStore) exportStoryBundle(storyID string, actor *authUser) (string, error) {
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	if s.storyHasBlockingGMJob(m) {
		return "", errors.New("story has an active GM job")
	}
	st, err := s.readState(storyID)
	if err != nil {
		return "", err
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.exportRoot, 0o700); err != nil {
		return "", err
	}
	bundleID := time.Now().UTC().Format("20060102T150405Z") + "_" + randomID()
	dir := filepath.Join(s.exportRoot, storyID, bundleID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "source_manifest.json"), m); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "turn_hashes.json"), map[string]any{
		"story_id": storyID,
		"turns":    turnHashes(turns),
	}); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, "summary.md"), []byte(firstNonEmpty(strings.TrimSpace(m.LatestSummary), "No summary yet")+"\n")); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, "storylet.md"), []byte(renderStoryletBundle(m, st, turns))); err != nil {
		return "", err
	}
	exportedAt := time.Now().UTC().Format(time.RFC3339)
	exportManifest := map[string]any{
		"story_id":                    storyID,
		"world_id":                    m.WorldID,
		"exported_at":                 exportedAt,
		"status":                      "draft_pending",
		"draft_target_suggestion":     filepath.ToSlash(filepath.Join("drafts", "storylets", storyID+".md")),
		"source_files":                []string{"source_manifest.json", "turn_hashes.json", "storylet.md", "summary.md"},
		"turn_hashes_path":            "turn_hashes.json",
		"storylet_path":               "storylet.md",
		"summary_path":                "summary.md",
		"next_admin_cli_instructions": []string{"Copy the bundle into the admin writer workflow.", "Create the draft at the suggested target path.", "Mark the draft as ready before republishing or review."},
	}
	if actor != nil && strings.TrimSpace(actor.ID) != "" {
		exportManifest["exported_by"] = actor.ID
	}
	if err := writeJSONAtomic(filepath.Join(dir, "export_manifest.json"), exportManifest); err != nil {
		return "", err
	}
	actorID := ""
	if actor != nil {
		actorID = strings.TrimSpace(actor.ID)
	}
	if err := appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":                    "story_export_handoff",
		"at":                      exportedAt,
		"story_id":                storyID,
		"actor_id":                actorID,
		"bundle_path":             dir,
		"target_draft_suggestion": filepath.ToSlash(filepath.Join("drafts", "storylets", storyID+".md")),
		"status":                  "draft_pending",
	}); err != nil {
		return "", err
	}
	return dir, nil
}

func turnHashes(turns []storyTurn) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, turn := range turns {
		b, _ := json.Marshal(turn)
		sum := sha256.Sum256(b)
		out = append(out, map[string]any{
			"turn_id":     turn.TurnID,
			"branch_id":   turn.BranchID,
			"source":      turn.Source,
			"created_at":  turn.CreatedAt,
			"hash":        "sha256:" + hex.EncodeToString(sum[:]),
			"input_id":    turn.InputID,
			"actor_id":    turn.ActorID,
			"parent_turn": turn.ParentTurnID,
		})
	}
	return out
}

func renderStoryletBundle(m storyManifest, st storyState, turns []storyTurn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", m.Title)
	fmt.Fprintf(&b, "- story_id: `%s`\n- status: `%s`\n- phase: `%s`\n- current_turn: `%d`\n- active_driver: `%s`\n", m.ID, m.Status, m.Phase, m.CurrentTurn, firstNonEmpty(m.ActiveDriverID, "open"))
	if m.SourceDraftPath != "" {
		fmt.Fprintf(&b, "- source_draft_path: `%s`\n", m.SourceDraftPath)
	}
	fmt.Fprintf(&b, "\n## Summary\n\n%s\n\n## State\n\n- Location: %s\n- Active characters: %s\n\n### Facts\n", firstNonEmpty(strings.TrimSpace(m.LatestSummary), "No summary yet"), firstNonEmpty(st.Location, "미정"), strings.Join(st.ActiveCharacters, ", "))
	for _, fact := range st.Facts {
		fmt.Fprintf(&b, "- %s\n", fact)
	}
	b.WriteString("\n### Open threads\n")
	for _, thread := range st.OpenThreads {
		fmt.Fprintf(&b, "- %s\n", thread)
	}
	b.WriteString("\n### Risks\n")
	for _, risk := range st.Risks {
		fmt.Fprintf(&b, "- %s\n", risk)
	}
	b.WriteString("\n## Turns\n")
	for _, turn := range turns {
		fmt.Fprintf(&b, "\n### Turn %d\n\n", turn.TurnID)
		fmt.Fprintf(&b, "- actor: `%s`\n- source: `%s`\n- input_id: `%s`\n\n", turn.ActorID, turn.Source, turn.InputID)
		if turn.SceneTitle != "" {
			fmt.Fprintf(&b, "**%s**\n\n", turn.SceneTitle)
		}
		if turn.SceneBody != "" {
			b.WriteString(strings.TrimSpace(turn.SceneBody) + "\n\n")
		}
		if turn.CurrentSituation != "" {
			fmt.Fprintf(&b, "_Current situation: %s_\n\n", turn.CurrentSituation)
		}
		if len(turn.RevealedFacts) > 0 {
			b.WriteString("Facts:\n")
			for _, fact := range turn.RevealedFacts {
				fmt.Fprintf(&b, "- %s\n", fact)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (s *storyStore) listJobs(storyID string) ([]gmJob, error) {
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
		if err := readJSON(filepath.Join(s.storyDir(storyID), "jobs", e.Name()), &j); err == nil {
			out = append(out, j)
		}
	}
	return out, nil
}

func (s *storyStore) findJobByIdempotencyKey(storyID, jobType, key string) (gmJob, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return gmJob{}, false, nil
	}
	jobs, err := s.listJobs(storyID)
	if err != nil {
		return gmJob{}, false, err
	}
	for _, job := range jobs {
		if job.IdempotencyKey == key && (jobType == "" || job.JobType == jobType) {
			return job, true, nil
		}
	}
	return gmJob{}, false, nil
}

func (s *storyStore) idempotencyMatchesStoryInput(job gmJob, actorID, choiceID, customMode, customText string, turnID int) bool {
	if job.ActorID != actorID || job.JobType != "story_turn" {
		return false
	}
	if job.TurnID != turnID {
		return false
	}
	if job.Input == nil {
		return false
	}
	return job.Input.SelectedChoiceID == choiceID && job.Input.CustomInputMode == customMode && job.Input.CustomText == strings.TrimSpace(customText)
}

func (s *storyStore) idempotencyMatchesQuestion(job gmJob, actorID, question string, turnID int) bool {
	if job.ActorID != actorID || job.JobType != "question_answer" {
		return false
	}
	if job.TurnID != turnID {
		return false
	}
	if job.Question == nil {
		return false
	}
	return job.Question.Question == strings.TrimSpace(question)
}

func (s *storyStore) failedProgressionJob(storyID string) (gmJob, bool, error) {
	m, err := s.readManifest(storyID)
	if err != nil {
		return gmJob{}, false, err
	}
	if m.Phase != "failed_waiting_retry" || m.ActiveJobID == "" {
		return gmJob{}, false, nil
	}
	job, err := s.readJob(storyID, m.ActiveJobID)
	if err != nil {
		return gmJob{}, false, err
	}
	return job, true, nil
}

func (s *storyStore) resumeFailedJob(storyID string, u *authUser) (string, error) {
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

func (s *storyStore) cancelFailedJob(storyID string, u *authUser) error {
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

func mustReadTurns(s *storyStore, storyID string) []storyTurn {
	turns, _ := s.readTurns(storyID)
	return turns
}

func (s *storyStore) ensureSeedStories(actorID string) error {
	id, _, err := s.importHector(actorID)
	if err != nil {
		return err
	}
	return s.refreshHectorHistory(id, actorID)
}

func (s *storyStore) importHector(actorID string) (string, bool, error) {
	sourceRel := filepath.Join("lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
	sourcePath := filepath.Join(s.packsRoot, sourceRel)
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", false, err
	}
	hashBytes := sha256.Sum256(b)
	hash := "sha256:" + hex.EncodeToString(hashBytes[:])
	parsed, err := parseHectorDraft(string(b))
	if err != nil {
		return "", false, err
	}
	existing, _ := s.listStories()
	for _, m := range existing {
		if m.SourceDraftPath == filepath.ToSlash(sourceRel) {
			updated := false
			title := firstNonEmpty(parsed.Title, "헥터: 첫 잔명 대조")
			nextTurn := maxInt(m.CurrentTurn, parsed.TurnID)
			nextSummary := m.LatestSummary
			if nextTurn <= parsed.TurnID {
				nextSummary = hectorCurrentSituation()
			}
			if m.SourceHash != hash || m.Title != title || m.CurrentTurn != nextTurn || m.LatestSummary != nextSummary {
				now := time.Now().UTC().Format(time.RFC3339)
				m.Title = title
				m.SourceHash = hash
				m.UpdatedAt = now
				m.CurrentTurn = nextTurn
				m.LatestSummary = nextSummary
				updated = true
			}
			if updated {
				if err := writeJSONAtomic(filepath.Join(s.storyDir(m.ID), "manifest.json"), m); err != nil {
					return "", false, err
				}
			}
			return m.ID, true, nil
		}
	}
	id := "story_hector_first_residual_check"
	if _, err := os.Stat(s.storyDir(id)); err == nil {
		id = id + "_" + randomID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := storyManifest{
		ID: id, Title: firstNonEmpty(parsed.Title, "헥터: 첫 잔명 대조"), WorldID: "lumen-federation",
		Status: "active", Phase: "waiting_for_choice", CurrentTurn: parsed.TurnID,
		ActiveDriverID: actorID, SourceDraftPath: filepath.ToSlash(sourceRel), SourceHash: hash,
		CreatedBy: actorID, CreatedAt: now, UpdatedAt: now, LatestSummary: hectorCurrentSituation(),
	}
	st := storyState{
		Location:         "베이르 제3정정실 / 마라 베온 대기실 연결 채널",
		ActiveCharacters: []string{"헥터", "라우", "마라 베온", "아델 카이", "V-13 베른"},
		Facts: []string{
			"마라 베온은 의료 진정 상태로 안정화 중이다.",
			"대체 우안 동기화율은 하락하기 시작했다.",
			"라우는 생존 보호 명령 범위를 표본 안정자 내부 반응 조정까지 확장했다.",
			"감리단은 이 확장이 자기 고유 권한 침해라고 주장한다.",
			"17-B 기록군에는 23개의 사망 보상권, 9개의 신체 권리 분쟁, 4개의 생존 주장 잔류 건이 묶여 있다.",
		},
		OpenThreads: []string{"17-B 기록군 전체를 열 추가 연결점 확보", "감리단 권한 충돌 대응", "마라 베온의 생존 주장과 원 신체 권리 주장 확정", "17B-STB-EYE-03의 표본 안정자 지위 반박"},
		Risks:       []string{"감리단이 범위 확장 철회를 요구하고 있다.", "기록군 전체 재심사가 다수의 보상권과 신체 권리 분쟁을 흔들 수 있다.", "마라의 안정 상태는 의료 조치에 의존하고 있어 장기 안전이 확정되지 않았다."},
		Flags:       []string{"mara_medically_stabilized", "survival_protection_extended", "inspectorate_conflict_open", "record_group_17b_revealed"},
	}
	turns := parsed.Turns
	if len(turns) == 0 {
		turns = []storyTurn{{TurnID: parsed.TurnID, BranchID: "branch_main", ParentTurnID: 18, ActorID: actorID, InputID: "import_hector_turn_19", Source: "import", SelectedChoiceID: "B", SceneTitle: "첫 잔명 대조", SceneBody: parsed.SceneBody, CurrentSituation: hectorCurrentSituation(), RevealedFacts: parsed.Facts, Choices: parsed.Choices, CreatedAt: now}}
	}
	for i := range turns {
		turns[i].ActorID = actorID
		turns[i].CreatedAt = now
	}
	if err := s.createStory(m, st, turns); err != nil {
		return "", false, err
	}
	return id, false, nil
}

func (s *storyStore) refreshHectorHistory(storyID, actorID string) error {
	sourcePath := filepath.Join(s.packsRoot, "lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	hashBytes := sha256.Sum256(b)
	currentHash := "sha256:" + hex.EncodeToString(hashBytes[:])
	parsed, err := s.parseHectorHistory()
	if err != nil || len(parsed.Turns) == 0 {
		return err
	}
	existing, _ := s.readTurns(storyID)
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.SourceHash == currentHash && hectorTurnsMatch(existing, parsed.Turns) {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existingTurn := maxStoryTurn(existing)
	nextTurn := maxInt(m.CurrentTurn, maxInt(existingTurn, parsed.TurnID))
	m.CurrentTurn = nextTurn
	m.UpdatedAt = now
	m.SourceHash = currentHash
	if nextTurn <= parsed.TurnID {
		m.LatestSummary = hectorCurrentSituation()
	}
	if m.ActiveDriverID == "" {
		m.ActiveDriverID = actorID
	}
	if existingTurn > parsed.TurnID || m.CurrentTurn > parsed.TurnID {
		return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
	}
	st, _ := s.readState(storyID)
	for i := range parsed.Turns {
		parsed.Turns[i].ActorID = actorID
		parsed.Turns[i].CreatedAt = now
	}
	return s.replaceStory(m, st, parsed.Turns)
}

func maxInt(values ...int) int {
	max := 0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func maxStoryTurn(turns []storyTurn) int {
	max := 0
	for _, turn := range turns {
		if turn.TurnID > max {
			max = turn.TurnID
		}
	}
	return max
}

func (s *storyStore) parseHectorHistory() (hectorParsed, error) {
	sourcePath := filepath.Join(s.packsRoot, "lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
	paths, _ := filepath.Glob(filepath.Join(s.packsRoot, "lumen-federation", "runs", "inbox", "*-body.md"))
	paths = append(paths, sourcePath)
	sort.Strings(paths)
	byTurn := map[int]storyTurn{}
	title := "헥터: 첫 잔명 대조"
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		p, err := parseHectorDraft(string(b))
		if err != nil {
			continue
		}
		if p.Title != "" {
			title = p.Title
		}
		for _, t := range p.Turns {
			byTurn[t.TurnID] = t
		}
	}
	if len(byTurn) == 0 {
		return hectorParsed{}, errors.New("no hector history turns found")
	}
	ids := make([]int, 0, len(byTurn))
	for id := range byTurn {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	turns := make([]storyTurn, 0, len(ids))
	for _, id := range ids {
		t := byTurn[id]
		t.BranchID = "branch_main"
		t.ParentTurnID = id - 1
		if id == 1 {
			t.ParentTurnID = 0
		}
		t.InputID = fmt.Sprintf("import_hector_turn_%d", id)
		t.Source = "import"
		turns = append(turns, t)
	}
	latest := turns[len(turns)-1]
	return hectorParsed{Title: title, TurnID: latest.TurnID, SceneBody: latest.SceneBody, Facts: latest.RevealedFacts, Choices: latest.Choices, Turns: turns}, nil
}

func hectorTurnsMatch(existing, parsed []storyTurn) bool {
	if len(existing) != len(parsed) {
		return false
	}
	for i := range existing {
		if !hectorTurnMatch(existing[i], parsed[i]) {
			return false
		}
	}
	return true
}

func hectorTurnMatch(existing, parsed storyTurn) bool {
	if existing.TurnID != parsed.TurnID || existing.BranchID != parsed.BranchID || existing.ParentTurnID != parsed.ParentTurnID || existing.InputID != parsed.InputID || existing.Source != parsed.Source || existing.SelectedChoiceID != parsed.SelectedChoiceID || existing.CustomInputMode != parsed.CustomInputMode || existing.CustomText != parsed.CustomText || existing.SceneTitle != parsed.SceneTitle || existing.SceneBody != parsed.SceneBody || existing.CurrentSituation != parsed.CurrentSituation {
		return false
	}
	if strings.Join(existing.RevealedFacts, "\n") != strings.Join(parsed.RevealedFacts, "\n") {
		return false
	}
	if len(existing.Choices) != len(parsed.Choices) {
		return false
	}
	for i := range existing.Choices {
		if existing.Choices[i] != parsed.Choices[i] {
			return false
		}
	}
	return true
}

func (s *storyStore) replaceStory(m storyManifest, st storyState, turns []storyTurn) error {
	dir := s.storyDir(m.ID)
	for _, name := range []string{"events.jsonl", "turns.jsonl", "manifest.json", "state.json", "summary.md"} {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return s.createStory(m, st, turns)
}

func (s *storyStore) createDemoStory(actorID, title, style, characterName, traits string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := "story_" + randomID()
	name := firstNonEmpty(strings.TrimSpace(characterName), "새 인물")
	style = firstNonEmpty(strings.TrimSpace(style), "조사극")
	traits = strings.TrimSpace(traits)
	location, scene, summary, facts, openThreads, risks, choices := luceraPrologueSeed(name, traits)
	m := storyManifest{ID: id, Title: firstNonEmpty(strings.TrimSpace(title), name+"의 이야기"), WorldID: "lumen-federation", Status: "active", Phase: "waiting_for_choice", CurrentTurn: 1, ActiveDriverID: actorID, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now, LatestSummary: summary}
	st := storyState{Location: location, ActiveCharacters: []string{name}, Facts: facts, OpenThreads: openThreads, Risks: risks, Flags: []string{"runtime_story_created"}}
	turn := storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: actorID, InputID: "setup_" + randomID(), Source: "setup", SceneTitle: style + "의 시작", SceneBody: scene, CurrentSituation: m.LatestSummary, RevealedFacts: st.Facts, Choices: choices, CreatedAt: now, CustomText: traits}
	return id, s.createStory(m, st, []storyTurn{turn})
}

func luceraPrologueSeed(name, traits string) (string, string, string, []string, []string, []string, []storyChoice) {
	subject := name + "는"
	location := "루세라 야간 진료동"
	scene := fmt.Sprintf("루멘 연방의 의료 도시 루세라에서는 불빛이 생명을 살리지만, 같은 빛이 괴물의 길을 그어 버리기도 한다. 그래서 이 병동은 늘 저안개 차단막을 낮추고, 낮은 절차로 움직이며, 공공 수선과 회복 공공재 배분을 함께 계산해야 한다.\n\n%s 간호기록판을 팔에 끼운 채 잠깐 멈춰 섰다. 접수대 위에는 오늘의 환자 목록과 공공 수선 요청서, 그리고 행정 장부의 빈 칸이 겹쳐 놓여 있었다. 방금 들어온 환자 셋의 기록은 서로 다른 증상을 말하고 있었지만, 병동의 빈 침상 수는 같은 답만 내놓았다. 더 받을 수 없다.\n\n그때 접수대 쪽에서 누군가 %s의 이름을 불렀다. 새 환자 하나가 쓰러졌고, 동시에 이미 누워 있던 아이의 보호자가 약속된 처치를 왜 미루냐고 묻기 시작했다. 둘 다 기다릴 수 없지만, %s의 손은 하나뿐이다.\n\n병동 안에는 배분표를 다시 맞추는 사람도, 공공 수선 반에게 문의하는 사람도, 장부상 허가를 확인하는 사람도 있었다. 루세라에서 회복은 늘 누군가의 자원을 다시 계산하는 일과 붙어 다녔다.", subject, name, name)
	if traits != "" {
		scene += "\n\n초기 설정 메모: " + traits
	}
	summary := fmt.Sprintf("루멘 연방의 루세라 야간 진료동에서 공공 수선과 회복 공공재, 그리고 행정 장부가 서로를 밀어내고 있다.")
	facts := []string{
		"루멘 연방의 루세라는 병원, 약, 수면을 맡는 의료 도시다.",
		"병동 불빛은 회복 공공재이지만 저안개 차단과 낮은 절차로 다뤄야 한다.",
		"공공 수선과 자원 배분은 보험과 행정 장부의 허가를 거친다.",
		fmt.Sprintf("주인공은 %s이다.", name),
		fmt.Sprintf("%s 루세라의 간호사다.", subject),
		fmt.Sprintf("초기 배경은 %s이다.", location),
		"아직 canon이 아닌 runtime story 상태다.",
	}
	if traits != "" {
		facts = append(facts, "초기 설정: "+traits)
	}
	openThreads := []string{
		"새 환자와 기존 환자 중 누구를 먼저 살릴지 결정",
		"병동의 부족한 자원을 어떻게 배분할지 판단",
		"공공 수선과 행정 장부를 어떻게 맞출지 정리",
	}
	risks := []string{
		"어느 쪽을 선택해도 다른 쪽의 상태가 악화될 수 있다.",
		"과로와 저안개 절차 때문에 판단 여력이 흔들릴 수 있다.",
		"장부상 허가가 늦어지면 회복 공공재 배분이 밀릴 수 있다.",
	}
	choices := []storyChoice{
		{ID: "A", Text: "새로 쓰러진 환자의 상태를 직접 확인한다.", RiskHint: "즉시 위험을 볼 수 있지만 기존 처치가 더 밀린다."},
		{ID: "B", Text: "기존 아이 환자의 처치를 먼저 이어간다.", RiskHint: "약속된 처치를 지키지만 새 환자를 놓칠 수 있다."},
		{ID: "C", Text: "보호자에게 짧게 설명하고 동료를 호출한다.", RiskHint: "시간을 벌 수 있지만 항의가 커질 수 있다."},
		{ID: "D", Text: "기록판과 펜으로 우선순위를 빠르게 다시 계산한다.", RiskHint: "근거는 남지만 현장 반응이 늦어진다."},
	}
	return location, scene, summary, facts, openThreads, risks, choices
}

func luceraWorldContextSeed() string {
	return "루멘 연방의 루세라는 병원·약·수면의 도시다. 병동 불빛은 회복 공공재이지만 저안개 차단, 낮은 절차, 공공 수선, 자원 배분, 그리고 보험과 행정 장부의 마찰 속에서만 유지된다."
}

func (s *storyStore) createStory(m storyManifest, st storyState, turns []storyTurn) error {
	dir := s.storyDir(m.ID)
	if err := os.MkdirAll(filepath.Join(dir, "jobs"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "memory-cards"), 0o700); err != nil {
		return err
	}
	for _, t := range turns {
		if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "turn_committed", "at": t.CreatedAt, "turn": t}); err != nil {
			return err
		}
		if err := appendJSONL(filepath.Join(dir, "turns.jsonl"), t); err != nil {
			return err
		}
	}
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "summary.md"), []byte(m.LatestSummary+"\n"))
}

func (s *storyStore) writeStoryTurnsProjection(storyID string, turns []storyTurn) error {
	var b bytes.Buffer
	for _, turn := range turns {
		data, err := json.Marshal(turn)
		if err != nil {
			return err
		}
		if _, err := b.Write(data); err != nil {
			return err
		}
		if err := b.WriteByte('\n'); err != nil {
			return err
		}
	}
	return writeAtomic(filepath.Join(s.storyDir(storyID), "turns.jsonl"), b.Bytes())
}

func (s *storyStore) rewriteStorySummary(storyID, summary string) error {
	return writeAtomic(filepath.Join(s.storyDir(storyID), "summary.md"), []byte(firstNonEmpty(strings.TrimSpace(summary), "No summary yet")+"\n"))
}

func findStoryTurn(turns []storyTurn, turnID int) (int, bool) {
	for i := range turns {
		if turns[i].TurnID == turnID {
			return i, true
		}
	}
	return -1, false
}

func (s *storyStore) appendChoice(storyID string, u *authUser, choiceID, customMode, customText string) error {
	unlock, err := s.acquireLock(storyID, "turn_input", u.ID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return errors.New("story is not waiting for input")
	}
	if u.Role != "admin" && (m.ActiveDriverID == "" || m.ActiveDriverID != u.ID) {
		return errors.New("only active driver can progress this story")
	}
	turns, err := s.readTurns(storyID)
	if err != nil || len(turns) == 0 {
		return errors.New("story has no turns")
	}
	prev := turns[len(turns)-1]
	if choiceID != "" {
		found := false
		for _, c := range prev.Choices {
			if c.ID == choiceID {
				found = true
			}
		}
		if !found {
			return errors.New("invalid choice")
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	inputLabel := firstNonEmpty(choiceID, customText)
	st, _ := s.readState(storyID)
	scene := generateLocalGMScene(prev, st, inputLabel, customMode)
	nextChoices := generateNextChoices(prev.TurnID+1, st)
	situation := generateCurrentSituation(st)
	t := storyTurn{TurnID: prev.TurnID + 1, BranchID: "branch_main", ParentTurnID: prev.TurnID, ActorID: u.ID, InputID: "input_" + randomID(), Source: "choice", SelectedChoiceID: choiceID, CustomInputMode: customMode, CustomText: customText, SceneTitle: fmt.Sprintf("Turn %d의 여파", prev.TurnID+1), SceneBody: scene, CurrentSituation: situation, RevealedFacts: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."}, Choices: nextChoices, CreatedAt: now}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "turn_committed", "at": now, "turn": t}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "turns.jsonl"), t); err != nil {
		return err
	}
	st.Facts = appendUnique(st.Facts, t.RevealedFacts...)
	st.OpenThreads = appendUnique(st.OpenThreads, "방금 선택의 절차적 후속 근거 확보")
	m.CurrentTurn = t.TurnID
	m.UpdatedAt = now
	m.LatestSummary = t.CurrentSituation
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, "manifest.json"), m)
}

func (s *storyStore) askQuestion(storyID string, u *authUser, question string) error {
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	_, err = s.submitQuestionJob(storyID, u, m.CurrentTurn, "", question)
	return err
}

func (s *storyStore) storyHasBlockingGMJob(m storyManifest) bool {
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

func (s *storyStore) appendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":        eventType,
		"at":          now,
		"story_id":    storyID,
		"actor_id":    actorID,
		"from_status": fromStatus,
		"to_status":   toStatus,
	})
}

type storyRecoveryReport struct {
	StoryID        string   `json:"story_id"`
	RecoveryStatus string   `json:"recovery_status"`
	CheckedFiles   []string `json:"checked_files"`
	RepairedItems  []string `json:"repaired_items"`
	LockRemoved    bool     `json:"lock_removed"`
}

func (s *storyStore) changeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType string) error {
	unlock, err := s.acquireLock(storyID, "admin_update", "admin")
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	if m.Status == nextStatus {
		return nil
	}
	if err := s.appendStoryLifecycleEvent(storyID, actorID, eventType, m.Status, nextStatus); err != nil {
		return err
	}
	m.Status = nextStatus
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}

func (s *storyStore) archiveStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "archived", "story_archived")
}

func (s *storyStore) restoreStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "active", "story_restored")
}

func (s *storyStore) deleteStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "deleted", "story_deleted")
}

func (s *storyStore) adminUpdateStory(storyID, actorID, status, activeDriver string) error {
	unlock, err := s.acquireLock(storyID, "admin_update", actorID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	if status != "" && status != m.Status {
		if err := s.appendStoryLifecycleEvent(storyID, actorID, "story_status_changed", m.Status, status); err != nil {
			return err
		}
		m.Status = status
	}
	if activeDriver == "__open__" {
		m.ActiveDriverID = ""
	} else if activeDriver != "" {
		m.ActiveDriverID = activeDriver
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}

func (s *storyStore) editCurrentTurn(storyID, actorID, sceneBody, currentSituation string) error {
	unlock, err := s.acquireLock(storyID, "admin_turn_edit", actorID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return err
	}
	idx, ok := findStoryTurn(turns, m.CurrentTurn)
	if !ok {
		return errors.New("current turn not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	before := turns[idx]
	turns[idx].SceneBody = sceneBody
	turns[idx].CurrentSituation = currentSituation
	m.CurrentTurn = turns[idx].TurnID
	m.LatestSummary = currentSituation
	m.UpdatedAt = now
	if err := s.writeStoryTurnsProjection(storyID, turns); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m); err != nil {
		return err
	}
	if err := s.rewriteStorySummary(storyID, currentSituation); err != nil {
		return err
	}
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":                "turn_edited_by_admin",
		"at":                  now,
		"story_id":            storyID,
		"actor_id":            actorID,
		"turn_id":             before.TurnID,
		"previous_scene_body": before.SceneBody,
		"previous_situation":  before.CurrentSituation,
		"turn":                turns[idx],
	})
}

func (s *storyStore) rollbackStoryToTurn(storyID, actorID string, targetTurnID int) error {
	unlock, err := s.acquireLock(storyID, "admin_turn_rollback", actorID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return err
	}
	idx, ok := findStoryTurn(turns, targetTurnID)
	if !ok {
		return errors.New("selected turn not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	kept := append([]storyTurn(nil), turns[:idx+1]...)
	fromTurnID := m.CurrentTurn
	if fromTurnID == 0 && len(turns) > 0 {
		fromTurnID = turns[len(turns)-1].TurnID
	}
	m.CurrentTurn = targetTurnID
	m.LatestSummary = kept[len(kept)-1].CurrentSituation
	m.UpdatedAt = now
	if err := s.writeStoryTurnsProjection(storyID, kept); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m); err != nil {
		return err
	}
	if err := s.rewriteStorySummary(storyID, m.LatestSummary); err != nil {
		return err
	}
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":            "story_rolled_back_by_admin",
		"at":              now,
		"story_id":        storyID,
		"actor_id":        actorID,
		"from_turn_id":    fromTurnID,
		"to_turn_id":      targetTurnID,
		"kept_turn_count": len(kept),
		"removed_turn_ids": func() []int {
			if len(turns) <= len(kept) {
				return nil
			}
			removed := make([]int, 0, len(turns)-len(kept))
			for _, turn := range turns[len(kept):] {
				removed = append(removed, turn.TurnID)
			}
			return removed
		}(),
	})
}

func (s *storyStore) updateDriver(storyID string, u *authUser, action string) error {
	unlock, err := s.acquireLock(storyID, "driver_"+action, u.ID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return errors.New("driver can only change while story is waiting")
	}
	switch action {
	case "release":
		if u.Role != "admin" && m.ActiveDriverID != u.ID {
			return errors.New("only active driver can open this story")
		}
		m.ActiveDriverID = ""
	case "claim":
		if m.ActiveDriverID != "" && u.Role != "admin" {
			return errors.New("story already has an active driver")
		}
		m.ActiveDriverID = u.ID
	default:
		return errors.New("unknown driver action")
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}

func (s *storyStore) recoverStory(storyID string) (storyRecoveryReport, error) {
	dir := s.storyDir(storyID)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return storyRecoveryReport{}, errors.New("story not found")
		}
		return storyRecoveryReport{}, err
	}
	if !info.IsDir() {
		return storyRecoveryReport{}, errors.New("story path is not a directory")
	}

	lockRemoved := false
	lockPath := filepath.Join(dir, "lock.json")
	if b, err := os.ReadFile(lockPath); err == nil {
		var lock map[string]any
		if json.Unmarshal(b, &lock) == nil {
			if at, err := time.Parse(time.RFC3339, fmt.Sprint(lock["acquired_at"])); err == nil && !at.IsZero() {
				if time.Since(at) > storyLockTimeout {
					if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
						return storyRecoveryReport{}, err
					}
					_ = fsyncDir(dir)
					lockRemoved = true
				} else {
					return storyRecoveryReport{}, errors.New("story is locked")
				}
			}
		}
	}

	unlock, err := s.acquireLock(storyID, "store_recover", "admin")
	if err != nil {
		return storyRecoveryReport{}, err
	}
	defer unlock()

	checked := []string{"events.jsonl", "turns.jsonl", "qa.jsonl"}
	repaired := []string{}

	eventsPath := filepath.Join(dir, "events.jsonl")
	eventsBefore, err := readFileIfExists(eventsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if err := readStoryJSONL(eventsPath, func(b []byte) error {
		var v map[string]any
		return json.Unmarshal(b, &v)
	}); err != nil {
		return storyRecoveryReport{}, err
	}
	eventsAfter, err := readFileIfExists(eventsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(eventsBefore, eventsAfter) {
		repaired = append(repaired, "events.jsonl")
	}

	turnsPath := filepath.Join(dir, "turns.jsonl")
	turnsBefore, err := readFileIfExists(turnsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if _, err := s.readTurns(storyID); err != nil {
		return storyRecoveryReport{}, err
	}
	turnsAfter, err := readFileIfExists(turnsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(turnsBefore, turnsAfter) {
		repaired = append(repaired, "turns.jsonl")
	}

	qaPath := filepath.Join(dir, "qa.jsonl")
	qaBefore, err := readFileIfExists(qaPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if _, err := s.readQA(storyID); err != nil {
		return storyRecoveryReport{}, err
	}
	qaAfter, err := readFileIfExists(qaPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(qaBefore, qaAfter) {
		repaired = append(repaired, "qa.jsonl")
	}

	status := "checked"
	if len(repaired) > 0 || lockRemoved {
		status = "recovered"
	}
	if err := appendJSONL(eventsPath, map[string]any{
		"type":            "story_recovered",
		"at":              time.Now().UTC().Format(time.RFC3339),
		"story_id":        storyID,
		"checked_files":   checked,
		"repaired_items":  repaired,
		"lock_removed":    lockRemoved,
		"recovery_status": status,
	}); err != nil {
		return storyRecoveryReport{}, err
	}
	return storyRecoveryReport{
		StoryID:        storyID,
		RecoveryStatus: status,
		CheckedFiles:   checked,
		RepairedItems:  repaired,
		LockRemoved:    lockRemoved,
	}, nil
}

func (s *storyStore) acquireLock(storyID, reason, actorID string) (func(), error) {
	dir := s.storyDir(storyID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "lock.json")
	now := time.Now().UTC()
	if b, err := os.ReadFile(path); err == nil {
		var existing map[string]any
		_ = json.Unmarshal(b, &existing)
		if at, _ := time.Parse(time.RFC3339, fmt.Sprint(existing["acquired_at"])); !at.IsZero() && now.Sub(at) > storyLockTimeout {
			_ = os.Remove(path)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, errors.New("story is locked")
		}
		return nil, err
	}
	lock := map[string]any{"story_id": storyID, "reason": reason, "actor_id": actorID, "acquired_at": now.Format(time.RFC3339)}
	b, _ := json.MarshalIndent(lock, "", "  ")
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	fsyncDir(dir)
	return func() {
		_ = os.Remove(path)
		fsyncDir(dir)
	}, nil
}

type hectorParsed struct {
	Title     string
	TurnID    int
	SceneBody string
	Facts     []string
	Choices   []storyChoice
	Turns     []storyTurn
}

func parseHectorDraft(body string) (hectorParsed, error) {
	var p hectorParsed
	if strings.HasPrefix(body, "---") {
		if end := strings.Index(body[3:], "---"); end >= 0 {
			fm := body[3 : 3+end]
			for _, line := range strings.Split(fm, "\n") {
				if strings.HasPrefix(line, "title:") {
					p.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "title:")), "'\"")
				}
			}
		}
	}
	re := regexp.MustCompile(`(?m)^## Turn ([0-9]+)\s*$`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return p, errors.New("no turn heading")
	}
	last := matches[len(matches)-1]
	n, _ := strconv.Atoi(body[last[2]:last[3]])
	p.TurnID = n
	start := last[1]
	section := body[start:]
	if len(matches) > 1 {
		_ = matches
	}
	p.SceneBody = sectionBetween(section, "### 판정", "### 확인된 정보")
	facts := sectionBetween(section, "### 확인된 정보", "### 다음 갈림길")
	for _, line := range strings.Split(facts, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			p.Facts = append(p.Facts, line)
		}
	}
	choices := sectionBetween(section, "### 다음 갈림길", "")
	p.Choices = extractChoices(choices)
	p.Turns = parseHectorTurns(body)
	return p, nil
}

func parseHectorTurns(body string) []storyTurn {
	re := regexp.MustCompile(`(?m)^## Turn ([0-9]+)\s*$`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	var turns []storyTurn
	for i, m := range matches {
		id, _ := strconv.Atoi(body[m[2]:m[3]])
		start := m[1]
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		turns = append(turns, parseHectorTurnSection(id, strings.TrimSpace(body[start:end])))
	}
	return turns
}

func parseHectorTurnSection(id int, section string) storyTurn {
	facts := extractListSection(section, "### 확인된 정보", "### 다음 갈림길")
	choices := extractChoices(firstNonEmpty(sectionBetween(section, "### 다음 갈림길", ""), sectionBetween(section, "### 선택지", "### 선택")))
	scene := strings.TrimSpace(section)
	if block := sectionBetween(section, "### 확인된 정보", "### 다음 갈림길"); block != "" {
		scene = strings.Replace(scene, "### 확인된 정보\n"+block, "", 1)
	}
	if block := sectionBetween(section, "### 다음 갈림길", ""); block != "" {
		scene = strings.Replace(scene, "### 다음 갈림길\n"+block, "", 1)
	}
	situation := firstNonEmpty(sectionBetween(section, "### 현재 결과", ""), sectionBetween(section, "### 상황", "### 선택지"), hectorCurrentSituation())
	return storyTurn{TurnID: id, SceneTitle: fmt.Sprintf("Turn %d", id), SceneBody: strings.TrimSpace(scene), CurrentSituation: situation, RevealedFacts: facts, Choices: choices}
}

func extractListSection(body, startHeading, endHeading string) []string {
	block := sectionBetween(body, startHeading, endHeading)
	var out []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func extractChoices(block string) []storyChoice {
	var out []storyChoice
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if len(line) > 3 && line[1] == '.' {
			id := line[:1]
			if id >= "A" && id <= "D" {
				out = append(out, storyChoice{ID: id, Text: strings.TrimSpace(line[2:])})
			}
		}
	}
	return out
}

func sectionBetween(body, startHeading, endHeading string) string {
	start := strings.Index(body, startHeading)
	if start < 0 {
		return ""
	}
	start += len(startHeading)
	rest := strings.TrimSpace(body[start:])
	if endHeading != "" {
		if end := strings.Index(rest, endHeading); end >= 0 {
			rest = rest[:end]
		}
	}
	return strings.TrimSpace(rest)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func readFileIfExists(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

func readStoryJSONL(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var (
		offset      int64
		lastGood    int64
		lineNo      int
		lastLine    []byte
		needsRepair bool
	)
	sc := bufio.NewReader(f)
	for {
		line, err := sc.ReadBytes('\n')
		if len(line) == 0 && err == io.EOF {
			break
		}
		lineNo++
		offset += int64(len(line))
		trimmed := bytes.TrimRight(line, "\r\n")
		if len(bytes.TrimSpace(trimmed)) == 0 {
			if err == io.EOF {
				break
			}
			lastGood = offset
			continue
		}
		if fnErr := fn(trimmed); fnErr != nil {
			if err == io.EOF {
				needsRepair = true
				lastLine = append([]byte(nil), trimmed...)
				break
			}
			_ = f.Close()
			return fmt.Errorf("malformed JSONL line %d in %s: %w", lineNo, path, fnErr)
		}
		lastGood = offset
		if err == io.EOF {
			break
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if !needsRepair {
		return nil
	}
	if err := truncateStoryJSONL(path, lastGood); err != nil {
		return err
	}
	if err := appendStoryRecoveryEvent(path, lastGood, lastLine); err != nil {
		return err
	}
	return nil
}

func truncateStoryJSONL(path string, offset int64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Truncate(offset); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func appendStoryRecoveryEvent(path string, truncatedTo int64, repairedLine []byte) error {
	eventPath := path
	if filepath.Base(path) != "events.jsonl" {
		eventPath = filepath.Join(filepath.Dir(path), "events.jsonl")
	}
	return appendJSONL(eventPath, map[string]any{
		"type":           "story_recovered",
		"at":             time.Now().UTC().Format(time.RFC3339),
		"recovered_path": filepath.Base(path),
		"truncated_to":   truncatedTo,
		"repaired_tail":  string(repairedLine),
	})
}

func readJSONL(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 8*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		if err := fn([]byte(sc.Text())); err != nil {
			return err
		}
	}
	return sc.Err()
}

func appendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(b, '\n'))
}

func writeAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp." + randomID()
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func fsyncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func appendUnique(in []string, vals ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range append(in, vals...) {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func hectorCurrentSituation() string {
	return "마라 베온은 안정화 중이고, 감리단 권한 충돌이 열린 상태다. 17-B 기록군 전체를 열 추가 연결점이 필요하다."
}

func generateLocalGMScene(prev storyTurn, st storyState, inputLabel, mode string) string {
	if inputLabel == "" {
		inputLabel = "직접 행동"
	}
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	location := firstNonEmpty(st.Location, "현재 장면")
	prefix := fmt.Sprintf("%s의 선택이 장면에 기록된다", name)
	if mode == "dialogue" {
		prefix = fmt.Sprintf("%s의 말이 주변의 반응을 바꾼다", name)
	} else if mode == "narration" {
		prefix = "장면의 초점이 한 단계 좁아진다"
	}
	anchor := "아직 확정된 단서는 많지 않다."
	if len(st.Facts) > 0 {
		anchor = st.Facts[0]
	}
	thread := "다음에 무엇을 붙잡을지 정해야 한다."
	if len(st.OpenThreads) > 0 {
		thread = st.OpenThreads[0]
	}
	risk := "상황이 쉽게 안정되지 않는다."
	if len(st.Risks) > 0 {
		risk = st.Risks[0]
	}
	return fmt.Sprintf("%s.\n\n입력: %s\n\n%s에서 반응은 즉시 결론으로 이어지지 않는다. 먼저 %s의 현재 상태와 방금 입력이 서로 맞물리는 지점이 드러난다. 확인된 기준은 '%s'이고, 열린 문제는 '%s'이다.\n\n이 진행은 아직 실제 GM worker가 아니라 서버 내장 MVP 판정으로 생성된 runtime 장면이다. 선택은 append-only 이벤트로 보존되며, admin이 export하지 않는 한 world pack의 canon이나 draft를 바꾸지 않는다.\n\n그럼에도 장면 안에서는 압력이 이동했다. %s 이제 플레이어는 이전 Turn %d에서 남은 단서를 기준으로 다음 선택 앞에 선다.", prefix, inputLabel, location, name, anchor, thread, risk, prev.TurnID)
}

func generateNextChoices(turn int, st storyState) []storyChoice {
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	return []storyChoice{
		{ID: "A", Text: "눈앞의 기록과 사실만 차분히 정리한다.", RiskHint: "안전하지만 느리다."},
		{ID: "B", Text: name + "가 가장 불편해하는 지점을 직접 말한다.", RiskHint: "빠르지만 충돌이 커진다."},
		{ID: "C", Text: "주변 인물의 반응을 살펴 추가 단서를 확보한다.", RiskHint: "장면 압박이 관계 쪽으로 옮겨간다."},
		{ID: "D", Text: fmt.Sprintf("Turn %d의 기록을 정리하고 잠시 관망한다.", turn), RiskHint: "정보는 보존하지만 주도권을 잃을 수 있다."},
	}
}

func generateCurrentSituation(st storyState) string {
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	location := firstNonEmpty(st.Location, "현재 장면")
	return fmt.Sprintf("%s에서 %s의 선택 결과가 다음 압력으로 이어졌다.", location, name)
}

func summarizeQuestionAnswer(question string, st storyState) string {
	if strings.Contains(question, "마라") {
		return "마라 베온은 의료 진정 상태로 안정화 중이며, 대체 우안 동기화율은 하락하기 시작한 것으로 기록되어 있다. 장면 기준으로 장기 안전은 아직 확정되지 않았다."
	}
	if strings.Contains(question, "17-B") || strings.Contains(question, "기록군") {
		return "17-B 기록군은 사망 보상권, 신체 권리 분쟁, 생존 주장 잔류 건이 묶인 불안정군이다. 현재 인물들이 확인한 범위에서는 이 기록군을 열 추가 연결점이 필요하다."
	}
	if len(st.Facts) > 0 {
		return "확인된 정보는 '" + st.Facts[0] + "' 등으로 제한된다. 그 밖의 전개는 현재 장면 기준으로는 확인되지 않았다."
	}
	return "현재 장면에서 확인된 정보가 충분하지 않다. 추론과 canon 사실은 구분해서 다뤄야 한다."
}
