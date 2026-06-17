package story

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func validGMOutputForValidation() GMOutput {
	return GMOutput{
		SchemaVersion:    "story-gm-output.v1",
		StoryID:          "story_1",
		Turn:             GMOutputTurn{TurnID: 2, ParentTurnID: 1, InputID: "input_1", JobID: "job_1"},
		SceneGoal:        "현재 장면의 다음 판단 기준을 정리한다.",
		Conflict:         "불확실한 정보와 시간 제약이 동시에 걸린다.",
		TurningPoint:     "우선순위를 다시 잡아야 하는 분기점이 생긴다.",
		Consequence:      "지금의 선택이 다음 턴의 상태 업데이트 범위를 바꾼다.",
		SceneTitle:       "제목",
		SceneBody:        strings.Repeat("장면 ", 60),
		CurrentSituation: "상황",
		StatePatch:       GMStatePatch{SummaryPatch: "요약", FactsAdd: []string{"사실"}},
		Choices: []Choice{
			{ID: "A", Text: "첫 번째 선택지를 취한다.", Intent: "장면의 위험도를 낮추기 위해 증거를 먼저 검증한다.", RiskHint: "안전하지만 즉시 반응은 느려질 수 있다."},
			{ID: "B", Text: "두 번째 선택지를 고른다.", Intent: "현재 흐름을 유지해 추가 혼선을 줄인다.", RiskHint: "기준을 고수하면 예기치 못한 반발이 남을 수 있다."},
			{ID: "C", Text: "세 번째 선택지를 검토한다.", Intent: "새 단서를 확인해 판단 근거를 확장한다.", RiskHint: "검토가 길어지면 기회가 흩어질 수 있다."},
		},
	}
}

func TestValidateGMOutputRequiresSceneQualityAndPatch(t *testing.T) {
	job := GMJob{ID: "job_1", StoryID: "story_1", TurnID: 2, ParentTurnID: 1, JobType: "story_turn", Input: &Input{ID: "input_1"}}
	out := validGMOutputForValidation()
	if err := ValidateGMOutput(job, out); err != nil {
		t.Fatalf("expected valid output, got %v", err)
	}
	out.SceneGoal = ""
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected missing scene_goal rejection")
	}
	out.SceneGoal = "목표"
	out.StatePatch = GMStatePatch{FactsAdd: []string{"  "}, FactsRemove: []string{"\t"}}
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected empty state_patch rejection")
	}
}

func TestValidateGMOutputRejectsNestedPatchContractWithoutEffectiveUpdate(t *testing.T) {
	job := GMJob{ID: "job_1", StoryID: "story_1", TurnID: 2, ParentTurnID: 1, JobType: "story_turn", Input: &Input{ID: "input_1"}}
	raw := map[string]any{
		"schema_version":    "story-gm-output.v1",
		"story_id":          "story_1",
		"turn":              map[string]any{"branch_id": "branch_main", "turn_id": 2, "parent_turn_id": 1, "input_id": "input_1", "job_id": "job_1", "source": "choice"},
		"scene_goal":        "현재 장면의 목표",
		"conflict":          "시간 압박",
		"turning_point":     "우선순위가 바뀐다",
		"consequence":       "흐름이 바뀐다",
		"scene_title":       "제목",
		"scene_body":        strings.Repeat("장면 ", 60),
		"current_situation": "상황",
		"revealed_facts":    []string{"사실"},
		"state_patch":       map[string]any{"add": map[string]any{"facts_add": []string{"새로운 사실"}}, "set": map[string]any{"location_set": "병동"}, "remove": map[string]any{"facts_remove": []string{"구식 사실"}}},
		"resolution":        "accepted",
		"choices": []map[string]string{
			{"id": "A", "text": "첫 번째 선택지를 취한다.", "intent": "우선순위를 보수적으로 맞춰 충돌 가능성을 낮춘다.", "risk_hint": "안전하지만 시간이 늘 수 있다."},
			{"id": "B", "text": "두 번째 선택지도 검토한다.", "intent": "균형을 맞추기 위해 조건을 보수적으로 다시 확인한다.", "risk_hint": "추가 확인이 필요해 처리 속도가 다소 느려질 수 있다."},
			{"id": "C", "text": "다른 경로로 빠르게 진행한다.", "intent": "현재 추정을 완화하고 예외 경로의 영향 범위를 분리한다.", "risk_hint": "시간 단축은 가능하지만 누락된 정보로 판단 오류가 생길 수 있다."},
		},
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var out GMOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected nested patch contract to be rejected")
	}
}

func TestValidateGMOutputRejectsWhitespaceChoiceID(t *testing.T) {
	job := GMJob{ID: "job_1", StoryID: "story_1", TurnID: 2, ParentTurnID: 1, JobType: "story_turn", Input: &Input{ID: "input_1"}}
	out := validGMOutputForValidation()
	out.Choices[0].ID = "   "
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected whitespace choice id rejection")
	}
}

func TestValidateGMOutputRejectsShortChoiceIntentAndRiskHint(t *testing.T) {
	job := GMJob{ID: "job_1", StoryID: "story_1", TurnID: 2, ParentTurnID: 1, JobType: "story_turn", Input: &Input{ID: "input_1"}}
	out := validGMOutputForValidation()
	out.Choices[0].Intent = "의도"
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected short intent rejection")
	}
	out = validGMOutputForValidation()
	out.Choices[1].RiskHint = "위험"
	if err := ValidateGMOutput(job, out); err == nil {
		t.Fatal("expected short risk hint rejection")
	}
	shortChoice := validGMOutputForValidation()
	if utf8.RuneCountInString(shortChoice.Choices[0].Intent) <= minChoiceMetaRunes {
		t.Fatal("intent fixture unexpectedly short")
	}
}

func TestGMOutputJSONRoundTripIncludesSceneQualityFields(t *testing.T) {
	valid := validGMOutputForValidation()
	raw := []byte(`{"schema_version":"story-gm-output.v1","story_id":"story_1","turn":{"branch_id":"branch_main","turn_id":2,"parent_turn_id":1,"input_id":"input_1","job_id":"job_1","source":"choice"},"scene_goal":"` + valid.SceneGoal + `","conflict":"` + valid.Conflict + `","turning_point":"` + valid.TurningPoint + `","consequence":"` + valid.Consequence + `","scene_title":"제목","scene_body":"` + strings.Repeat("장면 ", 60) + `","current_situation":"` + valid.CurrentSituation + `","revealed_facts":["사실"],"state_patch":{"summary_patch":"` + valid.StatePatch.SummaryPatch + `","facts_add":["사실"]},"resolution":"accepted","choices":[{"id":"A","text":"` + valid.Choices[0].Text + `","intent":"` + valid.Choices[0].Intent + `","risk_hint":"` + valid.Choices[0].RiskHint + `"},{"id":"B","text":"` + valid.Choices[1].Text + `","intent":"` + valid.Choices[1].Intent + `","risk_hint":"` + valid.Choices[1].RiskHint + `"},{"id":"C","text":"` + valid.Choices[2].Text + `","intent":"` + valid.Choices[2].Intent + `","risk_hint":"` + valid.Choices[2].RiskHint + `"}]}`)
	var out GMOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.SceneGoal == "" || out.Conflict == "" || out.TurningPoint == "" || out.Consequence == "" {
		t.Fatalf("missing scene quality fields: %#v", out)
	}
	if out.StatePatch.SummaryPatch == "" || len(out.StatePatch.FactsAdd) != 1 {
		t.Fatalf("missing state patch fields: %#v", out.StatePatch)
	}
}

func TestBuildCodexGMPromptUsesFlatStatePatchContract(t *testing.T) {
	req := GMRequest{
		Job: GMJob{ID: "job_1", StoryID: "story_1", JobType: "story_turn", TurnID: 2, ParentTurnID: 1, Input: &Input{ID: "input_1"}},
	}
	prompt := buildCodexGMPrompt(req, "/tmp/context.json")
	for _, want := range []string{"scene_goal", "conflict", "turning_point", "consequence", "location_set", "facts_add", "open_threads_add", "summary_patch"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
	for _, banned := range []string{`"add": {`, `"set": {`, `"remove": {`} {
		if strings.Contains(prompt, banned) {
			t.Fatalf("prompt still contains nested patch contract %q", banned)
		}
	}
}

func TestStoryWorldContextSeedRequiresPrologueOrLuceraLocation(t *testing.T) {
	storyTurn := GMJob{ID: "job_turn", StoryID: "story_1", JobType: "story_turn", TurnID: 2, ParentTurnID: 1, Input: &Input{ID: "input_1"}}
	if got := StoryWorldContextSeedForRequest(Manifest{WorldID: "lumen-federation"}, State{Location: "베이르 중앙역"}, storyTurn); got != "" {
		t.Fatalf("generic lumen-federation follow-up should not inject Lucera seed, got %q", got)
	}

	tests := []struct {
		name string
		m    Manifest
		st   State
		job  GMJob
	}{
		{
			name: "prologue in lumen federation",
			m:    Manifest{WorldID: "lumen-federation"},
			st:   State{Location: "미정"},
			job:  GMJob{ID: "job_prologue", StoryID: "story_1", JobType: "prologue", TurnID: 1},
		},
		{
			name: "korean lucera location",
			m:    Manifest{WorldID: "lumen-federation"},
			st:   State{Location: "루세라 야간 진료동"},
			job:  storyTurn,
		},
		{
			name: "english lucera location",
			m:    Manifest{WorldID: "lumen-federation"},
			st:   State{Location: "Lucera night clinic"},
			job:  storyTurn,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StoryWorldContextSeedForRequest(tt.m, tt.st, tt.job)
			if !strings.Contains(got, "루멘 연방의 루세라") || !strings.Contains(got, "회복 공공재") {
				t.Fatalf("expected Lucera anchor seed, got %q", got)
			}
		})
	}
}

func TestBuildCodexGMPromptReducesLuceraRepetitionOnFollowup(t *testing.T) {
	req := GMRequest{
		Job:      GMJob{ID: "job_2", StoryID: "story_1", JobType: "story_turn", TurnID: 2, ParentTurnID: 1, Input: &Input{ID: "input_1", SelectedChoiceID: "A"}},
		Manifest: Manifest{WorldID: "lumen-federation"},
		State: State{
			Location: "루세라 야간 진료동",
			Facts: []string{
				"루멘 연방의 루세라는 병원, 약, 수면을 맡는 의료 도시다.",
				"병동 불빛은 회복 공공재이지만 저안개 차단과 낮은 절차로 다뤄야 한다.",
			},
		},
		Turns: []Turn{{
			TurnID:        1,
			BranchID:      "branch_main",
			Source:        "setup",
			SceneBody:     "루세라 야간 진료동에서 공공 수선과 행정 장부의 마찰이 이미 소개되었다.",
			RevealedFacts: []string{"루멘 연방의 루세라는 병원, 약, 수면을 맡는 의료 도시다."},
			Choices:       []Choice{{ID: "A", Text: "새 환자를 확인한다.", Intent: "위험을 먼저 본다.", RiskHint: "기존 처치가 늦어진다."}},
		}},
	}
	req.WorldContext = StoryWorldContextSeedForRequest(req.Manifest, req.State, req.Job)
	if req.WorldContext == "" {
		t.Fatal("Lucera location should keep a world context anchor")
	}
	prompt := buildCodexGMPrompt(req, "/tmp/context.json")
	for _, banned := range []string{
		"For prologues and subsequent turns",
		"keep Lucera / Lumen Federation specifics visible",
		"required story anchor, not optional flavor",
	} {
		if strings.Contains(prompt, banned) {
			t.Fatalf("follow-up prompt still contains strong repeat guidance %q: %s", banned, prompt)
		}
	}
	for _, want := range []string{
		"Do not repeat already established Lucera / Lumen Federation baseline explanations",
		"revealed_facts must contain only newly discovered facts",
		"new action, conflict, and consequence",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("follow-up prompt missing anti-repetition rule %q: %s", want, prompt)
		}
	}
}

func TestBuildCodexGMPromptDoesNotMentionLuceraWithoutWorldContext(t *testing.T) {
	req := GMRequest{
		Job:      GMJob{ID: "job_2", StoryID: "story_1", JobType: "story_turn", TurnID: 2, ParentTurnID: 1, Input: &Input{ID: "input_1"}},
		Manifest: Manifest{WorldID: "lumen-federation"},
		State:    State{Location: "베이르 중앙역", Facts: []string{"베이르의 철도 운행이 지연되고 있다."}},
		Turns:    []Turn{{TurnID: 1, SceneBody: "베이르 중앙역의 지연이 이미 확인되었다."}},
	}
	req.WorldContext = StoryWorldContextSeedForRequest(req.Manifest, req.State, req.Job)
	if req.WorldContext != "" {
		t.Fatalf("non-Lucera follow-up should not have world context seed, got %q", req.WorldContext)
	}
	prompt := buildCodexGMPrompt(req, "/tmp/context.json")
	for _, banned := range []string{"Lucera", "루세라"} {
		if strings.Contains(prompt, banned) {
			t.Fatalf("prompt without world context should not inject %q: %s", banned, prompt)
		}
	}
	if !strings.Contains(prompt, "Do not repeat already established baseline world explanations") {
		t.Fatalf("prompt missing generic anti-repetition rule: %s", prompt)
	}

	qaReq := GMRequest{
		Job:      GMJob{ID: "job_q", StoryID: "story_1", JobType: "question_answer", Question: &Question{Question: "현재 상황은?"}},
		Manifest: req.Manifest,
		State:    req.State,
		Turns:    req.Turns,
	}
	qaPrompt := buildCodexGMPrompt(qaReq, "/tmp/context.json")
	for _, banned := range []string{"Lucera", "루세라"} {
		if strings.Contains(qaPrompt, banned) {
			t.Fatalf("QA prompt without world context should not inject %q: %s", banned, qaPrompt)
		}
	}
}

func TestBuildCodexGMPromptKeepsPrologueLuceraAnchor(t *testing.T) {
	req := GMRequest{
		Job:      GMJob{ID: "job_1", StoryID: "story_1", JobType: "prologue", TurnID: 1, Setup: &Setup{CharacterName: "르네", Style: "생존극", Traits: "루세라의 간호사"}},
		Manifest: Manifest{WorldID: "lumen-federation"},
		State:    State{Location: "미정"},
	}
	req.WorldContext = StoryWorldContextSeedForRequest(req.Manifest, req.State, req.Job)
	prompt := buildCodexGMPrompt(req, "/tmp/context.json")
	for _, want := range []string{
		"Use the world context seed as a prologue anchor",
		"Establish the Lucera / Lumen Federation baseline once",
		"루멘 연방의 루세라",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prologue prompt missing anchor %q: %s", want, prompt)
		}
	}
}

func TestBuildGMRequestKeepsMoreThanFiveTurns(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		if err := store.appendChoice(id, &Actor{ID: "user_admin", Role: "admin"}, "", "action", "상태를 확인한다"); err != nil {
			t.Fatal(err)
		}
		if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
			t.Fatal(err)
		}
	}
	job := GMJob{ID: "job_last", StoryID: id, JobType: "story_turn", TurnID: 9, ParentTurnID: 8, Input: &Input{ID: "input_last"}}
	req, err := store.buildGMRequest(job)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(req.Turns); got != 8 {
		t.Fatalf("expected 8 recent turns, got %d", got)
	}
}

func TestAppendChoiceProvidesLegacyQualityMetadataAndChoiceQuality(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.appendChoice(id, &Actor{ID: "user_admin", Role: "admin"}, "", "action", "상황을 정리한다"); err != nil {
		t.Fatal(err)
	}
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns after local append, got %d", len(turns))
	}
	turn := turns[1]
	if strings.TrimSpace(turn.SceneGoal) == "" || strings.TrimSpace(turn.Conflict) == "" || strings.TrimSpace(turn.TurningPoint) == "" || strings.TrimSpace(turn.Consequence) == "" {
		t.Fatalf("missing legacy scene quality metadata: %#v", turn)
	}
	if len(turn.Choices) == 0 {
		t.Fatal("local legacy turn should include choices")
	}
	for i, c := range turn.Choices {
		if strings.TrimSpace(c.Intent) == "" || strings.TrimSpace(c.RiskHint) == "" {
			t.Fatalf("choice %d missing intent/risk hint: %#v", i, c)
		}
		if utf8.RuneCountInString(strings.TrimSpace(c.Intent)) < minChoiceMetaRunes || utf8.RuneCountInString(strings.TrimSpace(c.RiskHint)) < minChoiceMetaRunes {
			t.Fatalf("choice %d intent/risk hint too short: %#v", i, c)
		}
	}
}

func TestStoryInputCreatesJobBeforeTurnCommit(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	jobID, err := store.submitStoryInput(id, &Actor{ID: "user_admin", Role: "admin"}, 1, "input-idem-create", "A", "", "")
	if err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Phase != "gm_generating" || m.ActiveJobID != jobID || m.CurrentTurn != 1 {
		t.Fatalf("manifest after submit = %#v", m)
	}
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("turn was committed synchronously: %#v", turns)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	m, _ = store.readManifest(id)
	turns, _ = store.readTurns(id)
	if m.Phase != "waiting_for_choice" || m.ActiveJobID != "" || m.CurrentTurn != 2 || len(turns) != 2 {
		t.Fatalf("job did not commit turn: manifest=%#v turns=%d", m, len(turns))
	}
}

func TestStoryInputRejectsStaleTurn(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitStoryInput(id, &Actor{ID: "user_admin", Role: "admin"}, 1, "input-idem-stale-1", "A", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitStoryInput(id, &Actor{ID: "user_admin", Role: "admin"}, 1, "input-idem-stale-2", "A", "", ""); err == nil {
		t.Fatal("expected stale turn rejection")
	}
}

func TestQuestionJobIsIdempotent(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	jobID1, err := store.submitQuestionJob(id, &Actor{ID: "user_admin", Role: "admin"}, 1, "question-idem", "루세라는 어디야?")
	if err != nil {
		t.Fatal(err)
	}
	jobID2, err := store.submitQuestionJob(id, &Actor{ID: "user_admin", Role: "admin"}, 1, "question-idem", "루세라는 어디야?")
	if err != nil {
		t.Fatal(err)
	}
	if jobID1 != jobID2 {
		t.Fatalf("expected duplicate question job id to match, got %q and %q", jobID1, jobID2)
	}
	qa, err := store.readQA(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(qa) != 1 {
		t.Fatalf("expected one queued question, got %#v", qa)
	}
}

func TestNewStoryCreatesPrologueJob(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createStoryWithPrologueJob("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Phase != "gm_generating" || m.CurrentTurn != 0 || m.ActiveJobID == "" {
		t.Fatalf("manifest after create = %#v", m)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	m, _ = store.readManifest(id)
	turns, _ := store.readTurns(id)
	if m.Phase != "waiting_for_choice" || m.CurrentTurn != 1 || len(turns) != 1 {
		t.Fatalf("prologue did not commit: manifest=%#v turns=%d", m, len(turns))
	}
	if !strings.Contains(turns[0].SceneBody, "르네") || !strings.Contains(turns[0].SceneBody, "루세라") {
		t.Fatalf("bad prologue: %q", turns[0].SceneBody)
	}
}

func TestQuestionUsesGMJobWithoutAdvancingTurn(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.askQuestion(id, &Actor{ID: "user_admin", Role: "admin"}, "루세라는 어디야?"); err != nil {
		t.Fatal(err)
	}
	m, _ := store.readManifest(id)
	if m.CurrentTurn != 1 || m.Phase != "waiting_for_choice" {
		t.Fatalf("question changed progression state: %#v", m)
	}
	qa, _ := store.readQA(id)
	if len(qa) != 1 || qa[0].Answer != "답변 생성 중" {
		t.Fatalf("pending qa = %#v", qa)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	m, _ = store.readManifest(id)
	turns, _ := store.readTurns(id)
	qa, _ = store.readQA(id)
	if m.CurrentTurn != 1 || len(turns) != 1 {
		t.Fatalf("question advanced turn: manifest=%#v turns=%d", m, len(turns))
	}
	if len(qa) != 1 || qa[0].Answer == "답변 생성 중" || !strings.Contains(qa[0].Answer, "Turn 기준") {
		t.Fatalf("answered qa = %#v", qa)
	}
}

func TestMockPrologueUsesWorldSeed(t *testing.T) {
	req := GMRequest{
		Job: GMJob{
			ID:        "job_mock_prologue",
			StoryID:   "story_mock_prologue",
			JobType:   "prologue",
			Setup:     &Setup{Title: "르네의 이야기", Style: "생존극", CharacterName: "르네", Traits: "루세라의 간호사"},
			TurnID:    1,
			CreatedAt: "2026-06-07T00:00:00Z",
		},
		Manifest: Manifest{WorldID: "lumen-federation"},
		State:    State{Location: "미정", ActiveCharacters: []string{"르네"}},
	}
	req.WorldContext = StoryWorldContextSeedForRequest(req.Manifest, req.State, req.Job)
	out, _, _, _, err := mockPrologueOutput(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"루멘 연방", "회복 공공재", "공공 수선", "저안개", "낮은 절차"} {
		if !strings.Contains(out.SceneBody, want) {
			t.Fatalf("scene body missing world term %q: %q", want, out.SceneBody)
		}
	}
	if out.SceneGoal == "" || out.Conflict == "" || out.TurningPoint == "" || out.Consequence == "" {
		t.Fatalf("missing quality metadata: %#v", out)
	}
	for i, choice := range out.Choices {
		if strings.TrimSpace(choice.Intent) == "" || strings.TrimSpace(choice.RiskHint) == "" {
			t.Fatalf("choice %d missing intent or risk hint: %#v", i, choice)
		}
	}
	joinedFacts := strings.Join(out.StatePatch.FactsAdd, "\n")
	for _, want := range []string{"루멘 연방", "회복 공공재", "공공 수선"} {
		if !strings.Contains(joinedFacts, want) {
			t.Fatalf("state patch missing world seed %q: %#v", want, out.StatePatch.FactsAdd)
		}
	}
}
