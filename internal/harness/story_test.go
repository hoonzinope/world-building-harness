package harness

import (
	"context"
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

func TestImportHectorDeduplicatesSameSourceHash(t *testing.T) {
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
	if !strings.Contains(turns[1].SceneBody, "르네") || !strings.Contains(turns[1].SceneBody, "루세라") {
		t.Fatalf("progression does not use story state: %q", turns[1].SceneBody)
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
	jobID, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, "A", "", "")
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
