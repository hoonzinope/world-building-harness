package story

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

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
	joinedFacts := strings.Join(out.StatePatch.FactsAdd, "\n")
	for _, want := range []string{"루멘 연방", "회복 공공재", "공공 수선"} {
		if !strings.Contains(joinedFacts, want) {
			t.Fatalf("state patch missing world seed %q: %#v", want, out.StatePatch.FactsAdd)
		}
	}
}
