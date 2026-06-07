package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseHectorDraftLatestTurn(t *testing.T) {
	body := `---
title: '헥터: 첫 잔명 대조'
---

## Turn 18

### 판정
old

## Turn 19

### 선택
B. old choice

### 판정
latest scene

### 확인된 정보
- fact one
- fact two

### 다음 갈림길
A. choice A
B. choice B
C. choice C
D. choice D
`
	parsed, err := parseHectorDraft(body)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "헥터: 첫 잔명 대조" {
		t.Fatalf("title = %q", parsed.Title)
	}
	if parsed.TurnID != 19 {
		t.Fatalf("turn = %d", parsed.TurnID)
	}
	if parsed.SceneBody != "latest scene" {
		t.Fatalf("scene = %q", parsed.SceneBody)
	}
	if len(parsed.Facts) != 2 || parsed.Facts[0] != "fact one" {
		t.Fatalf("facts = %#v", parsed.Facts)
	}
	if len(parsed.Choices) != 4 || parsed.Choices[3].ID != "D" || parsed.Choices[3].Text != "choice D" {
		t.Fatalf("choices = %#v", parsed.Choices)
	}
	if len(parsed.Turns) != 2 || parsed.Turns[0].TurnID != 18 || parsed.Turns[1].TurnID != 19 {
		t.Fatalf("turns = %#v", parsed.Turns)
	}
}

func TestImportHectorUpdatesExistingStoryForSameSourcePath(t *testing.T) {
	root := t.TempDir()
	packs := filepath.Join(root, "packs")
	sourceDir := filepath.Join(packs, "lumen-federation", "drafts", "storylets")
	if err := os.MkdirAll(sourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `---
title: '헥터: 첫 잔명 대조'
---

## Turn 19

### 판정
scene

### 확인된 정보
- fact

### 다음 갈림길
A. choice A
B. choice B
C. choice C
D. choice D
`
	if err := os.WriteFile(filepath.Join(sourceDir, "hector_first_residual_check.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openStoryStore(filepath.Join(root, "data", "stories"), packs)
	if err != nil {
		t.Fatal(err)
	}
	first, existed, err := store.importHector("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatal("first import should not report existing story")
	}
	second, existed, err := store.importHector("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Fatal("second import should report existing story")
	}
	if first != second {
		t.Fatalf("expected same story id, got %q and %q", first, second)
	}
	firstManifest, err := store.readManifest(first)
	if err != nil {
		t.Fatal(err)
	}
	firstHash := firstManifest.SourceHash
	updatedBody := `---
title: '헥터: 첫 잔명 대조 / 개정'
---

## Turn 19

### 판정
scene updated

### 확인된 정보
- fact

### 다음 갈림길
A. choice A
B. choice B
C. choice C
D. choice D
`
	if err := os.WriteFile(filepath.Join(sourceDir, "hector_first_residual_check.md"), []byte(updatedBody), 0o600); err != nil {
		t.Fatal(err)
	}
	third, existed, err := store.importHector("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Fatal("reimport after source change should report existing story")
	}
	if third != first {
		t.Fatalf("expected same story id after source change, got %q and %q", first, third)
	}
	secondManifest, err := store.readManifest(first)
	if err != nil {
		t.Fatal(err)
	}
	if secondManifest.SourceHash == firstHash {
		t.Fatalf("expected refreshed source hash, got %q", secondManifest.SourceHash)
	}
	if secondManifest.Title != "헥터: 첫 잔명 대조 / 개정" {
		t.Fatalf("title was not refreshed: %#v", secondManifest)
	}
	turn20 := storyTurn{
		TurnID:           20,
		BranchID:         "branch_main",
		ParentTurnID:     19,
		ActorID:          "user_admin",
		InputID:          "runtime_turn_20",
		Source:           "story_turn",
		SelectedChoiceID: "A",
		CustomInputMode:  "action",
		CustomText:       "runtime progress",
		SceneTitle:       "runtime 20",
		SceneBody:        "runtime scene 20",
		CurrentSituation: "runtime summary 20",
		CreatedAt:        "2026-06-07T00:50:00Z",
	}
	if err := appendJSONL(filepath.Join(root, "data", "stories", first, "turns.jsonl"), turn20); err != nil {
		t.Fatal(err)
	}
	secondManifest.CurrentTurn = 20
	secondManifest.LatestSummary = "runtime summary 20"
	secondManifest.UpdatedAt = "2026-06-07T00:50:00Z"
	if err := writeJSONAtomic(filepath.Join(root, "data", "stories", first, "manifest.json"), secondManifest); err != nil {
		t.Fatal(err)
	}
	if err := store.ensureSeedStories("user_admin"); err != nil {
		t.Fatal(err)
	}
	turns, err := store.readTurns(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) == 0 || turns[len(turns)-1].TurnID != 20 || !strings.Contains(turns[len(turns)-1].SceneBody, "runtime scene 20") {
		t.Fatalf("runtime turn was not preserved: %#v", turns)
	}
	finalManifest, err := store.readManifest(first)
	if err != nil {
		t.Fatal(err)
	}
	if finalManifest.CurrentTurn != 20 {
		t.Fatalf("current turn regressed: %#v", finalManifest)
	}
	stories, err := store.listStories()
	if err != nil {
		t.Fatal(err)
	}
	if len(stories) != 1 || stories[0].ID != first {
		t.Fatalf("listStories should stay deduped, got %#v", stories)
	}
}

func TestListStoriesCollapsesDuplicateSourceDraftPath(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.ToSlash(filepath.Join("lumen-federation", "drafts", "storylets", "hector_first_residual_check.md"))
	older := storyManifest{
		ID:              "story_older_active",
		Title:           "older active",
		WorldID:         "lumen-federation",
		Status:          "active",
		Phase:           "waiting_for_choice",
		CreatedBy:       "user_admin",
		CreatedAt:       "2026-06-07T00:00:00Z",
		UpdatedAt:       "2026-06-07T00:10:00Z",
		SourceDraftPath: sourcePath,
		SourceHash:      "sha256:old",
		LatestSummary:   "older active summary",
	}
	newerDeleted := storyManifest{
		ID:              "story_newer_deleted",
		Title:           "newer deleted",
		WorldID:         "lumen-federation",
		Status:          "deleted",
		Phase:           "waiting_for_choice",
		CreatedBy:       "user_admin",
		CreatedAt:       "2026-06-07T00:30:00Z",
		UpdatedAt:       "2026-06-07T00:40:00Z",
		SourceDraftPath: sourcePath,
		SourceHash:      "sha256:new",
		LatestSummary:   "newer deleted summary",
	}
	for _, m := range []storyManifest{older, newerDeleted} {
		dir := filepath.Join(storyRoot, m.ID)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
			t.Fatal(err)
		}
	}
	stories, err := store.listStories()
	if err != nil {
		t.Fatal(err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected one visible story, got %#v", stories)
	}
	if stories[0].ID != older.ID {
		t.Fatalf("expected non-deleted story to win, got %#v", stories[0])
	}
}

func TestParseHectorHistoryMergesRunInbox(t *testing.T) {
	root := t.TempDir()
	packs := filepath.Join(root, "packs")
	inbox := filepath.Join(packs, "lumen-federation", "runs", "inbox")
	drafts := filepath.Join(packs, "lumen-federation", "drafts", "storylets")
	if err := os.MkdirAll(inbox, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(drafts, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inbox, "20260603-body.md"), []byte(`# 헥터: 첫 잔명 대조

## Turn 1

### 상황
start

### 선택지
- A. old A

### 현재 결과
result 1
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(drafts, "hector_first_residual_check.md"), []byte(`---
title: '헥터: 첫 잔명 대조'
---

## Turn 2

### 판정
latest

### 확인된 정보
- fact

### 다음 갈림길
A. choice A
`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openStoryStore(filepath.Join(root, "data", "stories"), packs)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := store.parseHectorHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Turns) != 2 || parsed.TurnID != 2 {
		t.Fatalf("history = %#v", parsed)
	}
	if parsed.Turns[0].Choices[0].Text != "old A" || parsed.Turns[1].RevealedFacts[0] != "fact" {
		t.Fatalf("merged turns = %#v", parsed.Turns)
	}
}

func TestNewStoryProgressionUsesItsOwnState(t *testing.T) {
	root := t.TempDir()
	store, err := openStoryStore(filepath.Join(root, "stories"), filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.appendChoice(id, &authUser{ID: "user_admin", Role: "admin"}, "", "action", "베이르가 아닌 루세라에 있어야 한다고 주장한다"); err != nil {
		t.Fatal(err)
	}
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("turn count = %d", len(turns))
	}
	if !strings.Contains(turns[0].SceneBody, "르네") || !strings.Contains(turns[0].SceneBody, "루세라") {
		t.Fatalf("initial scene does not preserve setup: %q", turns[0].SceneBody)
	}
	for _, want := range []string{"루멘 연방", "회복 공공재", "공공 수선", "저안개"} {
		if !strings.Contains(turns[0].SceneBody, want) {
			t.Fatalf("initial scene missing world term %q: %q", want, turns[0].SceneBody)
		}
	}
	if !strings.Contains(turns[1].SceneBody, "르네") || !strings.Contains(turns[1].SceneBody, "루멘 연방") {
		t.Fatalf("progression does not use story state: %q", turns[1].SceneBody)
	}
	st, err := store.readState(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"루멘 연방", "회복 공공재", "공공 수선"} {
		found := false
		for _, fact := range st.Facts {
			if strings.Contains(fact, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("state facts missing world seed %q: %#v", want, st.Facts)
		}
	}
	for _, leaked := range []string{"헥터", "라우", "아델", "감리단"} {
		if strings.Contains(turns[0].SceneBody, leaked) {
			t.Fatalf("initial scene leaked hector seed term %q: %q", leaked, turns[0].SceneBody)
		}
		if strings.Contains(turns[1].SceneBody, leaked) {
			t.Fatalf("progression leaked hector seed term %q: %q", leaked, turns[1].SceneBody)
		}
	}
}

func TestStoryReadTurnsRepairsFinalPartialTail(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_partial_tail"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	turn1 := storyTurn{
		TurnID:           1,
		BranchID:         "branch_main",
		ActorID:          "user_admin",
		InputID:          "input_1",
		Source:           "setup",
		SceneTitle:       "Turn 1",
		SceneBody:        "first",
		CurrentSituation: "situation 1",
		CreatedAt:        "2026-06-07T00:00:00Z",
	}
	turn2 := storyTurn{
		TurnID:           2,
		BranchID:         "branch_main",
		ParentTurnID:     1,
		ActorID:          "user_admin",
		InputID:          "input_2",
		Source:           "choice",
		SceneTitle:       "Turn 2",
		SceneBody:        "second",
		CurrentSituation: "situation 2",
		CreatedAt:        "2026-06-07T00:01:00Z",
	}
	b1, err := json.Marshal(turn1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(turn2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), append(append(b1, '\n'), b2[:len(b2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].TurnID != 1 {
		t.Fatalf("recovered turns = %#v", turns)
	}

	data, err := os.ReadFile(filepath.Join(dir, "turns.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, append(b1, '\n')) {
		t.Fatalf("turns.jsonl was not truncated: %q", string(data))
	}

	var sawRecovery bool
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_recovered" {
			sawRecovery = true
			if ev["recovered_path"] != "turns.jsonl" {
				t.Fatalf("recovered_path = %#v", ev["recovered_path"])
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawRecovery {
		t.Fatal("missing story_recovered event after truncating turns.jsonl")
	}
}

func TestStoryReadTurnsRejectsMalformedMiddleLine(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_bad_middle"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	good, err := json.Marshal(storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: "user_admin", InputID: "input_1", Source: "setup", SceneTitle: "Turn 1", SceneBody: "first", CurrentSituation: "situation 1", CreatedAt: "2026-06-07T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	bad := []byte(`{"turn_id":2,"branch_id":"branch_main"`)
	third, err := json.Marshal(storyTurn{TurnID: 3, BranchID: "branch_main", ParentTurnID: 2, ActorID: "user_admin", InputID: "input_3", Source: "choice", SceneTitle: "Turn 3", SceneBody: "third", CurrentSituation: "situation 3", CreatedAt: "2026-06-07T00:02:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	body.Write(good)
	body.WriteByte('\n')
	body.Write(bad)
	body.WriteByte('\n')
	body.Write(third)
	body.WriteByte('\n')
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := store.readTurns(id); err == nil {
		t.Fatal("expected malformed middle line to fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "events.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("events.jsonl should not be created on middle-line failure: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "turns.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, bad) {
		t.Fatalf("turns.jsonl was unexpectedly repaired: %q", string(data))
	}
}

func TestStoryRecoverRepairsPartialTailsAndRemovesStaleLock(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_recover"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	event1, err := json.Marshal(map[string]any{"type": "story_created", "at": "2026-06-07T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	event2, err := json.Marshal(map[string]any{"type": "story_updated", "at": "2026-06-07T00:01:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), append(append(event1, '\n'), event2[:len(event2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	turn1 := storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: "user_admin", InputID: "input_1", Source: "setup", SceneTitle: "Turn 1", SceneBody: "one", CurrentSituation: "situation 1", CreatedAt: "2026-06-07T00:00:00Z"}
	turn2 := storyTurn{TurnID: 2, BranchID: "branch_main", ParentTurnID: 1, ActorID: "user_admin", InputID: "input_2", Source: "choice", SceneTitle: "Turn 2", SceneBody: "two", CurrentSituation: "situation 2", CreatedAt: "2026-06-07T00:01:00Z"}
	t1, err := json.Marshal(turn1)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := json.Marshal(turn2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), append(append(t1, '\n'), t2[:len(t2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	q1 := storyQuestion{ID: "q1", ActorID: "user_admin", Question: "where?", Answer: "here", TurnID: 1, CreatedAt: "2026-06-07T00:00:00Z"}
	q2 := storyQuestion{ID: "q2", ActorID: "user_admin", Question: "why?", Answer: "because", TurnID: 2, CreatedAt: "2026-06-07T00:01:00Z"}
	qb1, err := json.Marshal(q1)
	if err != nil {
		t.Fatal(err)
	}
	qb2, err := json.Marshal(q2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "qa.jsonl"), append(append(qb1, '\n'), qb2[:len(qb2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	lock := map[string]any{"story_id": id, "reason": "stuck", "actor_id": "user_admin", "acquired_at": "2020-01-01T00:00:00Z"}
	if err := writeJSONAtomic(filepath.Join(dir, "lock.json"), lock); err != nil {
		t.Fatal(err)
	}

	report, err := store.recoverStory(id)
	if err != nil {
		t.Fatal(err)
	}
	if report.RecoveryStatus != "recovered" {
		t.Fatalf("recovery status = %q", report.RecoveryStatus)
	}
	if !report.LockRemoved {
		t.Fatal("expected stale lock to be removed")
	}
	if len(report.CheckedFiles) != 3 {
		t.Fatalf("checked files = %#v", report.CheckedFiles)
	}
	wantRepaired := map[string]bool{"events.jsonl": true, "turns.jsonl": true, "qa.jsonl": true}
	for _, got := range report.RepairedItems {
		delete(wantRepaired, got)
	}
	if len(wantRepaired) != 0 {
		t.Fatalf("missing repaired items: %#v", wantRepaired)
	}
	if _, err := os.Stat(filepath.Join(dir, "lock.json")); !os.IsNotExist(err) {
		t.Fatalf("lock.json should have been removed: %v", err)
	}

	var sawSummary bool
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_recovered" && ev["story_id"] == id {
			if got := ev["recovery_status"]; got != "recovered" {
				t.Fatalf("recovery_status = %#v", got)
			}
			if got := ev["lock_removed"]; got != true {
				t.Fatalf("lock_removed = %#v", got)
			}
			if got := ev["checked_files"]; len(got.([]any)) != 3 {
				t.Fatalf("checked_files = %#v", got)
			}
			sawSummary = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawSummary {
		t.Fatal("missing story_recovered summary event")
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
	jobID, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-idem-create", "A", "", "")
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
	if _, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-idem-stale-1", "A", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-idem-stale-2", "A", "", ""); err == nil {
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
	jobID1, err := store.submitQuestionJob(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "question-idem", "루세라는 어디야?")
	if err != nil {
		t.Fatal(err)
	}
	jobID2, err := store.submitQuestionJob(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "question-idem", "루세라는 어디야?")
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

func TestExportStoryBundleWritesHandoffMetadataAndAuditEvent(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	bundleDir, err := store.exportStoryBundle(id, &authUser{ID: "user_admin", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"source_manifest.json", "turn_hashes.json", "storylet.md", "summary.md", "export_manifest.json"} {
		if _, err := os.Stat(filepath.Join(bundleDir, name)); err != nil {
			t.Fatalf("missing bundle file %q: %v", name, err)
		}
	}

	var exportManifest map[string]any
	if err := readJSON(filepath.Join(bundleDir, "export_manifest.json"), &exportManifest); err != nil {
		t.Fatal(err)
	}
	if got := exportManifest["story_id"]; got != id {
		t.Fatalf("story_id = %#v", got)
	}
	if got := exportManifest["status"]; got != "draft_pending" {
		t.Fatalf("status = %#v", got)
	}
	if got := exportManifest["exported_by"]; got != "user_admin" {
		t.Fatalf("exported_by = %#v", got)
	}
	if got := exportManifest["draft_target_suggestion"]; got != filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md")) {
		t.Fatalf("draft target = %#v", got)
	}
	if got := exportManifest["turn_hashes_path"]; got != "turn_hashes.json" {
		t.Fatalf("turn_hashes_path = %#v", got)
	}
	if got := exportManifest["storylet_path"]; got != "storylet.md" {
		t.Fatalf("storylet_path = %#v", got)
	}
	if got := exportManifest["summary_path"]; got != "summary.md" {
		t.Fatalf("summary_path = %#v", got)
	}

	var sawExportEvent bool
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_export_handoff" {
			sawExportEvent = true
			if ev["actor_id"] != "user_admin" {
				t.Fatalf("actor_id = %#v", ev["actor_id"])
			}
			if ev["bundle_path"] != bundleDir {
				t.Fatalf("bundle_path = %#v", ev["bundle_path"])
			}
			if ev["target_draft_suggestion"] != filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md")) {
				t.Fatalf("target_draft_suggestion = %#v", ev["target_draft_suggestion"])
			}
			if ev["status"] != "draft_pending" {
				t.Fatalf("status = %#v", ev["status"])
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawExportEvent {
		t.Fatal("missing story_export_handoff audit event")
	}
}

func TestStoryLifecycleActionsAuditAndGuardActiveJobs(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.adminUpdateStory(id, "user_admin", "paused", "user_friend"); err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "paused" || m.ActiveDriverID != "user_friend" {
		t.Fatalf("generic admin update failed: %#v", m)
	}

	if err := store.archiveStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.restoreStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.deleteStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}

	m, err = store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "deleted" {
		t.Fatalf("delete did not tombstone story: %#v", m)
	}
	if _, err := os.Stat(filepath.Join(storyRoot, id, "turns.jsonl")); err != nil {
		t.Fatalf("story data was removed: %v", err)
	}

	var eventTypes []string
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if typ, _ := ev["type"].(string); typ != "" {
			eventTypes = append(eventTypes, typ)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"story_status_changed", "story_archived", "story_restored", "story_deleted"} {
		found := false
		for _, got := range eventTypes {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing audit event %q in %#v", want, eventTypes)
		}
	}

	busyID, err := store.createStoryWithPrologueJob("user_admin", "busy", "조사극", "르네", "")
	if err != nil {
		t.Fatal(err)
	}
	busyManifest, err := store.readManifest(busyID)
	if err != nil {
		t.Fatal(err)
	}
	busyManifest.Phase = "waiting_for_choice"
	if err := writeJSONAtomic(filepath.Join(storyRoot, busyID, "manifest.json"), busyManifest); err != nil {
		t.Fatal(err)
	}
	if err := store.archiveStory(busyID, "user_admin"); err == nil {
		t.Fatal("expected archive to be blocked by active GM job")
	}
	if _, err := store.exportStoryBundle(busyID, &authUser{ID: "user_admin", Role: "admin"}); err == nil {
		t.Fatal("expected export to be blocked by active GM job")
	}
	if err := store.editCurrentTurn(busyID, "user_admin", "edited", "summary"); err == nil {
		t.Fatal("expected edit to be blocked by active GM job")
	}
	if err := store.rollbackStoryToTurn(busyID, "user_admin", 1); err == nil {
		t.Fatal("expected rollback to be blocked by active GM job")
	}
}

func TestStoryAdminTurnEditAndRollbackUpdateProjection(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.appendChoice(id, &authUser{ID: "user_admin", Role: "admin"}, "", "action", "베이르가 아닌 루세라에 있어야 한다고 주장한다"); err != nil {
		t.Fatal(err)
	}

	if err := store.editCurrentTurn(id, "user_admin", "edited scene body", "edited current situation"); err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentTurn != 2 || m.LatestSummary != "edited current situation" {
		t.Fatalf("manifest not updated after edit: %#v", m)
	}
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 || turns[1].SceneBody != "edited scene body" || turns[1].CurrentSituation != "edited current situation" {
		t.Fatalf("turn projection not updated after edit: %#v", turns)
	}
	summary, err := os.ReadFile(filepath.Join(storyRoot, id, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(summary)); got != "edited current situation" {
		t.Fatalf("summary.md = %q", string(summary))
	}

	if err := store.rollbackStoryToTurn(id, "user_admin", 1); err != nil {
		t.Fatal(err)
	}
	m, err = store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentTurn != 1 || m.LatestSummary == "edited current situation" {
		t.Fatalf("manifest not rolled back: %#v", m)
	}
	turns, err = store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].TurnID != 1 {
		t.Fatalf("turn projection not rolled back: %#v", turns)
	}
	summary, err = os.ReadFile(filepath.Join(storyRoot, id, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(summary)); got != turns[0].CurrentSituation {
		t.Fatalf("summary.md was not rolled back: %q", string(summary))
	}

	var eventTypes []string
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if typ, _ := ev["type"].(string); typ != "" {
			eventTypes = append(eventTypes, typ)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !containsString(eventTypes, "turn_edited_by_admin") {
		t.Fatalf("missing edit audit event: %#v", eventTypes)
	}
	if !containsString(eventTypes, "story_rolled_back_by_admin") {
		t.Fatalf("missing rollback audit event: %#v", eventTypes)
	}
}

func TestStoryReadEventsRepairsFinalPartialTail(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	_, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_events_tail"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(map[string]any{"type": "turn_committed", "at": "2026-06-07T00:00:00Z", "turn": map[string]any{"turn_id": 1}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(map[string]any{"type": "turn_committed", "at": "2026-06-07T00:01:00Z", "turn": map[string]any{"turn_id": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), append(append(first, '\n'), second[:len(second)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	var events []map[string]any
	if err := readStoryJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0]["type"] != "turn_committed" {
		t.Fatalf("unexpected repaired read: %#v", events)
	}
	events = nil
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected recovery event in events.jsonl, got %#v", events)
	}
	if events[1]["type"] != "story_recovered" {
		t.Fatalf("missing story_recovered event: %#v", events)
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
	if err := store.askQuestion(id, &authUser{ID: "user_admin", Role: "admin"}, "루세라는 어디야?"); err != nil {
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
	req := gmRequest{
		Job: gmJob{
			ID:        "job_mock_prologue",
			StoryID:   "story_mock_prologue",
			JobType:   "prologue",
			Setup:     &storySetup{Title: "르네의 이야기", Style: "생존극", CharacterName: "르네", Traits: "루세라의 간호사"},
			TurnID:    1,
			CreatedAt: "2026-06-07T00:00:00Z",
		},
		Manifest: storyManifest{WorldID: "lumen-federation"},
		State:    storyState{Location: "미정", ActiveCharacters: []string{"르네"}},
	}
	req.WorldContext = storyWorldContextSeedForRequest(req.Manifest, req.State, req.Job)
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

func containsString(in []string, want string) bool {
	for _, got := range in {
		if got == want {
			return true
		}
	}
	return false
}
