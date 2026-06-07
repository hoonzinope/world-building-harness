package harness

import (
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
