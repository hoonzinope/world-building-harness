package harness

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFriendlyStoryLabels(t *testing.T) {
	for step, want := range map[string]string{
		"queued":     "대기열",
		"generating": "생성 중",
		"applying":   "반영 중",
		"ready":      "입력 가능",
		"failed":     "실패",
		"unknown":    "unknown",
	} {
		if got := friendlyStoryProgressStepLabel(step); got != want {
			t.Fatalf("step %q => %q, want %q", step, got, want)
		}
	}
	for kind, want := range map[string]string{
		"setup":    "설정",
		"choice":   "선택",
		"custom":   "입력",
		"question": "질문",
		"other":    "other",
	} {
		if got := friendlyStoryEventKindLabel(kind); got != want {
			t.Fatalf("kind %q => %q, want %q", kind, got, want)
		}
	}
}

func TestStoryLobbyUpdatedAtFormatting(t *testing.T) {
	if got := storyLobbyUpdatedAt("2026-06-07T06:20:43Z"); got != "2026.06.07 15:20" {
		t.Fatalf("unexpected localized timestamp %q", got)
	}
	if got := storyLobbyUpdatedAt("not-a-timestamp"); got != "업데이트 시각 확인 불가" {
		t.Fatalf("unexpected fallback timestamp %q", got)
	}
}

func TestStoryLobbyUsesKoreanFilterLabelsAndFriendlyStatusBadges(t *testing.T) {
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

	html := renderStoryLobbyHTML(t, &webServer{stories: store}, &authUser{ID: "user_admin", Role: "admin"}, "?filter=active")
	for _, want := range []string{
		`href="/stories"`,
		`>전체<`,
		`href="/stories?filter=active"`,
		`>진행 중<`,
		`aria-selected="true"`,
		`role="tab"`,
		`href="/stories?filter=mine"`,
		`>내 스토리<`,
		`href="/stories?filter=watch"`,
		`>관전<`,
		`href="/stories?filter=archived"`,
		`>보관됨<`,
		`href="/stories?filter=imported"`,
		`>가져온 스토리<`,
		`story-lobby-list`,
		`story-card`,
		`story-card-meta`,
		`story-card-summary`,
		`진행 중`,
		`응답 대기`,
		`참여 가능`,
		`입장하기`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in rendered story lobby", want)
		}
	}
	for _, forbidden := range []string{
		`story-lobby-table`,
		`story-lobby-status`,
		`story-lobby-turn`,
		`>active<`,
		`>waiting_for_choice<`,
		`activewaiting_for_choice`,
		`waiting_for_choice · active`,
		`입력 대기 · 진행 중`,
		`>입장<`,
		`진행 가능`,
		`입력 대기`,
		`open으로`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("unexpected raw status token %q in rendered story lobby", forbidden)
		}
	}
	if !strings.Contains(html, id) {
		t.Fatalf("missing created story row in rendered story lobby")
	}
	assertNoStoryIDInLobbyMeta(t, html, "story_")
	assertLobbyMetaLineFormatting(t, html)
	assertNoRawLobbyTimestamp(t, html)
}

func TestStoryLobbyAndNewStoryUseLocalizedLabels(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사"); err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	lobbyHTML := renderStoryLobbyHTML(t, srv, &authUser{ID: "user_admin", Role: "admin"}, "")
	for _, want := range []string{
		`<h1>스토리</h1>`,
		`story-lobby-list`,
		`story-card-title`,
		`story-card-meta`,
		`가져온 스토리`,
	} {
		if !strings.Contains(lobbyHTML, want) {
			t.Fatalf("missing %q in story lobby", want)
		}
	}
	for _, forbidden := range []string{
		`story-lobby-table`,
		`<th class="story-lobby-turn">턴</th>`,
		`헥터 가져오기`,
		`/stories/import/hector`,
		`>Stories<`,
		`>New Story<`,
		`>World<`,
		`>Title<`,
		`>Style<`,
		`>Character Name<`,
		`open으로`,
	} {
		if strings.Contains(lobbyHTML, forbidden) {
			t.Fatalf("unexpected English/raw label %q in story lobby", forbidden)
		}
	}
	assertNoStoryIDInLobbyMeta(t, lobbyHTML, "story_")
	assertLobbyMetaLineFormatting(t, lobbyHTML)
	assertNoRawLobbyTimestamp(t, lobbyHTML)

	req := httptest.NewRequest(http.MethodGet, "/stories/new", nil)
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	srv.render(rec, req, "새 스토리", newStoryTemplate, map[string]any{
		"Base":      "",
		"User":      &authUser{ID: "user_admin", Role: "admin"},
		"CSRFToken": "csrf-test",
	})
	newStoryHTML := rec.Body.String()
	for _, want := range []string{
		`<h1>새 스토리</h1>`,
		`<label class="muted" for="new-story-world">세계관</label>`,
		`id="new-story-world"`,
		`<label class="muted" for="new-story-title">제목</label>`,
		`id="new-story-title"`,
		`<label class="muted" for="new-story-style">스타일</label>`,
		`id="new-story-style"`,
		`<label class="muted" for="new-story-character-name">캐릭터 이름</label>`,
		`id="new-story-character-name"`,
		`<label class="muted" for="new-story-traits">특징 / 취향</label>`,
		`id="new-story-traits"`,
		`<option value="조사극">조사극</option>`,
	} {
		if !strings.Contains(newStoryHTML, want) {
			t.Fatalf("missing %q in new story page", want)
		}
	}
	for _, forbidden := range []string{
		`>Stories<`,
		`>New Story<`,
		`>World<`,
		`>Title<`,
		`>Style<`,
		`>Character Name<`,
		`open으로`,
	} {
		if strings.Contains(newStoryHTML, forbidden) {
			t.Fatalf("unexpected English/raw label %q in new story page", forbidden)
		}
	}
}

func TestPublicStoryLobbyAccessibleWithoutLogin(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	auth := newTestAuthStore(t)
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사"); err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store, auth: auth, authRequired: true, storyEnabled: true}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	srv.handle(rootRec, rootReq)
	if rootRec.Code != http.StatusSeeOther {
		t.Fatalf("unexpected root status %d: %s", rootRec.Code, rootRec.Body.String())
	}
	if got := rootRec.Header().Get("Location"); got != "/stories" {
		t.Fatalf("unexpected root redirect %q", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/stories", nil)
	rec := httptest.NewRecorder()
	srv.handle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()
	for _, want := range []string{
		`<h1>스토리</h1>`,
		`로그인`,
		`story-lobby-list`,
		`story-card-meta`,
		`관전`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in public story lobby", want)
		}
	}
	for _, forbidden := range []string{
		`story-lobby-table`,
		`/stories/new`,
		`name="action" value="create"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("unexpected %q in public story lobby", forbidden)
		}
	}
	assertNoStoryIDInLobbyMeta(t, html, "story_")
	assertLobbyMetaLineFormatting(t, html)
	assertNoRawLobbyTimestamp(t, html)
}
