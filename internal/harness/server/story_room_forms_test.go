package server

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestStoryRoomFormsWireCSRFAndUniqueIdempotencyKeys(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	m.ActiveDriverID = ""
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	htmlOpen := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	htmlQuestion := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_friend", Role: "friend"}, "")
	m.ActiveDriverID = "user_admin"
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	htmlAssigned := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")

	for _, want := range []string{
		`data-story-room`,
		`story-room-shell`,
		`story-room-grid`,
		`grid-template-areas:"current composer" "current dossier" "timeline dossier"`,
		`current-turn-panel`,
		`current-turn-body`,
		`current-turn-flow`,
		`turn-sidebar`,
		`turn-timeline`,
		`previous-turns`,
		`previous-turn`,
		`choice-card`,
		`choice-card-risk`,
		`story-choice-submit-panel`,
		`story-choice-form`,
		`story-input-divider`,
		`story-custom-input-panel`,
		`story-composer-panel-head`,
		`story-composer-actions`,
		`dossier-stack`,
		`dossier-panel`,
		`플레이어에게 공개된 정보만 요약합니다.`,
		`story-composer`,
		`mobile-action-dock`,
		`mode-tabs`,
		`progress-loader`,
		`data-story-progress`,
		`data-story-step="queued"`,
		`data-story-step="generating"`,
		`data-story-step="applying"`,
		`data-story-step="ready"`,
		`data-story-submit`,
		`data-story-refresh`,
		`script defer src="`,
		`/assets/story-room.js`,
		`name="csrf_token" value="`,
		`name="action" value="claim"`,
		`name="action" value="update"`,
		`name="action" value="edit_turn"`,
		`name="action" value="rollback_turn"`,
		`name="action" value="export_bundle"`,
		`data-story-progress-meta hidden`,
		`<strong data-story-progress-label>입력 가능</strong>`,
		`data-step-label="ready"`,
		`role="status" aria-live="polite" aria-atomic="true"`,
		`진행 단계`,
		`최신 턴 준비`,
		`이번 턴에서 확인된 정보`,
		`누적 확인 정보`,
		`현재 턴`,
		`href="#input-panel" aria-label="입력`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing %q in rendered story room", want)
		}
	}
	for _, forbidden := range []string{
		`>ready<`,
		`>queued<`,
		`>generating<`,
		`>applying<`,
		`>failed<`,
		`>custom<`,
		`>setup<`,
		`session-rail`,
		`open으로`,
	} {
		if strings.Contains(htmlOpen, forbidden) {
			t.Fatalf("unexpected raw visible token %q in rendered story room", forbidden)
		}
	}
	if !strings.Contains(htmlOpen, `id="story-progress" role="status" aria-live="polite" aria-atomic="true" aria-busy="false"`) {
		t.Fatalf("missing idle aria-busy=false on story progress in rendered story room")
	}
	for _, forbidden := range []string{
		`async function submitForm(form)`,
		`submitForm(form, event.submitter || null);`,
		`if (submitter && submitter.name) {`,
		`data.set(submitter.name, submitter.value);`,
		`data.delete('custom_text');`,
		`data.delete('mode');`,
	} {
		if strings.Contains(htmlOpen, forbidden) {
			t.Fatalf("unexpected inline story-room JS %q in rendered story room", forbidden)
		}
	}
	for _, want := range []string{
		`data-story-submit-kind="input"`,
		`data-story-submit-kind="choice"`,
		`name="choice_id" value="A"`,
		`story-choice-form`,
		`data-story-choice-button`,
		`A/B/C/D를 바로 보낼 수 있습니다.`,
		`data-story-custom-textarea`,
		`참여 가능`,
		`name="mode" value="action"`,
		`name="mode" value="dialogue"`,
		`name="mode" value="question"`,
		`name="mode" value="narration"`,
	} {
		if !strings.Contains(htmlAssigned, want) {
			t.Fatalf("missing %q in assigned-driver story room", want)
		}
	}
	for _, want := range []string{
		`data-story-submit-kind="question"`,
		`data-story-question-textarea`,
		`질문 제출`,
	} {
		if !strings.Contains(htmlQuestion, want) {
			t.Fatalf("missing %q in non-driver question view", want)
		}
	}
	if !strings.Contains(htmlAssigned, `name="action" value="release"`) {
		t.Fatalf("missing release action in assigned-driver story room")
	}
	if strings.Contains(htmlOpen, `:has(`) {
		t.Fatalf("unexpected :has() selector in rendered story room")
	}
	for _, want := range []string{
		`위치`,
		`등장 인물`,
		`확인된 정보`,
		`열린 실마리`,
		`위험`,
		`턴`,
		`상태`,
		`진행자`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing localized label %q in rendered story room", want)
		}
	}
	for _, want := range []string{
		`입력 대기 · 진행 중`,
		`진행 중`,
		`입력 대기`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing friendly story-room label %q in rendered story room", want)
		}
	}
	if strings.Contains(htmlOpen, `waiting_for_choice · active`) {
		t.Fatalf("unexpected raw story-room status rail label in rendered story room")
	}
	for _, want := range []string{
		`관리`,
		`상태`,
		`진행자 ID`,
		fmt.Sprintf("현재 턴 %d 편집", m.CurrentTurn),
		`장면 본문`,
		`현재 상황`,
		`편집 저장`,
		`되돌릴 턴`,
		`되돌리기`,
		`보관`,
		`삭제`,
		`번들 내보내기`,
		`저장소 복구`,
		`턴 `,
		`<option value="active">진행 중</option>`,
		`<option value="paused">일시 정지</option>`,
		`<option value="completed">완료</option>`,
		`<option value="archived">보관됨</option>`,
		`진행자 비우기`,
	} {
		if !strings.Contains(htmlAssigned, want) {
			t.Fatalf("missing localized admin label %q in rendered story room", want)
		}
	}
	for _, forbidden := range []string{
		`Edit current turn 19Scene body`,
		`<h3>Admin</h3>`,
		`>Status<`,
		`>Active driver user id<`,
		`>Scene body<`,
		`>Current situation<`,
		`>save turn edit<`,
		`>Rollback to turn<`,
		`>rollback<`,
		`>archive<`,
		`>delete<`,
		`>export bundle<`,
		`>recover store<`,
		`open으로`,
		`>active<`,
		`>paused<`,
		`>completed<`,
		`>archived<`,
	} {
		if strings.Contains(htmlAssigned, forbidden) {
			t.Fatalf("unexpected old admin label %q in rendered story room", forbidden)
		}
	}

	re := regexp.MustCompile(`name="idempotency_key" value="([^"]+)"`)
	matches := re.FindAllStringSubmatch(htmlOpen, -1)
	if len(matches) < 5 {
		t.Fatalf("expected multiple idempotency keys, got %d", len(matches))
	}
	seen := map[string]bool{}
	for _, m := range matches {
		seen[m[1]] = true
	}
	if len(seen) != len(matches) {
		t.Fatalf("expected unique idempotency keys, got %d matches with %d unique values", len(matches), len(seen))
	}
}
