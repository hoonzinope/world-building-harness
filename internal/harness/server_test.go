package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type storyTaskResponse struct {
	StoryID         string `json:"story_id"`
	JobID           string `json:"job_id"`
	JobType         string `json:"job_type"`
	TurnID          int    `json:"turn_id"`
	StatusURL       string `json:"status_url"`
	NextPollMS      int    `json:"next_poll_ms"`
	StatusLabel     string `json:"status_label"`
	ProgressMessage string `json:"progress_message"`
	StepIndex       int    `json:"step_index"`
	StepLabel       string `json:"step_label"`
	IsProcessing    bool   `json:"is_processing"`
	ActiveJobID     string `json:"active_job_id"`
	ActiveJobType   string `json:"active_job_type"`
	ActiveJobStatus string `json:"active_job_status"`
	CurrentTurn     int    `json:"current_turn"`
}

type storyStatusResponse struct {
	StoryID          string                      `json:"story_id"`
	Status           string                      `json:"status"`
	Phase            string                      `json:"phase"`
	CurrentTurn      int                         `json:"current_turn"`
	ActiveJobID      string                      `json:"active_job_id"`
	ActiveJobType    string                      `json:"active_job_type"`
	ActiveJobStatus  string                      `json:"active_job_status"`
	IsProcessing     bool                        `json:"is_processing"`
	CanDrive         bool                        `json:"can_drive"`
	CanQuestion      bool                        `json:"can_question"`
	StatusLabel      string                      `json:"status_label"`
	ProgressMessage  string                      `json:"progress_message"`
	StepIndex        int                         `json:"step_index"`
	StepLabel        string                      `json:"step_label"`
	NextPollMS       int                         `json:"next_poll_ms"`
	PendingQuestions []storyProgressQuestionView `json:"pending_questions"`
}

func renderStoryRoomHTML(t *testing.T, s *webServer, id string, u *authUser, query string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/stories/"+id+query, nil)
	if u != nil {
		req = withUser(req, u)
	}
	rec := httptest.NewRecorder()
	s.renderStoryRoom(rec, req, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func renderStoryLobbyHTML(t *testing.T, s *webServer, u *authUser, query string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/stories"+query, nil)
	if u != nil {
		req = withUser(req, u)
	}
	rec := httptest.NewRecorder()
	s.renderStoryLobby(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func assertNoStoryIDInLobbyMeta(t *testing.T, html, id string) {
	t.Helper()
	metaRe := regexp.MustCompile(`(?s)<div class="story-card-meta">(.*?)</div>`)
	matches := metaRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		t.Fatal("missing story-card-meta block in rendered story lobby")
	}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if strings.Contains(match[1], id) {
			t.Fatalf("unexpected raw story id %q in story card meta", id)
		}
	}
}

func assertLobbyMetaLineFormatting(t *testing.T, html string) {
	t.Helper()
	metaRe := regexp.MustCompile(`(?s)<div class="story-card-meta">(.*?)</div>`)
	matches := metaRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		t.Fatal("missing story-card-meta block in rendered story lobby")
	}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		meta := match[1]
		if !strings.Contains(meta, " · ") {
			t.Fatalf("expected dot-separated lobby meta line, got %q", meta)
		}
		if strings.Contains(meta, "story_") {
			t.Fatalf("unexpected raw story id in lobby meta line %q", meta)
		}
		if strings.Contains(meta, "가져온 스토리관전") || strings.Contains(meta, "내 스토리턴") || strings.Contains(meta, "관전턴") {
			t.Fatalf("unexpected adjacent lobby meta labels in %q", meta)
		}
	}
}

func assertNoRawLobbyTimestamp(t *testing.T, html string) {
	t.Helper()
	rawTimestampRe := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	if rawTimestampRe.FindString(html) != "" {
		t.Fatal("unexpected raw ISO timestamp in rendered story lobby")
	}
	formattedTimestampRe := regexp.MustCompile(`업데이트 \d{4}\.\d{2}\.\d{2} \d{2}:\d{2}`)
	if !formattedTimestampRe.MatchString(html) {
		t.Fatal("missing localized lobby update timestamp")
	}
}

func newTestAuthStore(t *testing.T) *authStore {
	t.Helper()
	root := t.TempDir()
	t.Setenv("WORLD_HARNESS_ADMIN_USERNAME", "admin")
	t.Setenv("WORLD_HARNESS_ADMIN_PASSWORD", "password")
	store, err := openAuthStore(filepath.Join(root, "auth.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func addCookie(req *http.Request, name, value string) {
	req.AddCookie(&http.Cookie{Name: name, Value: value, Path: "/"})
}

func authenticatedRequest(t *testing.T, auth *authStore, method, target string, body *strings.Reader) (*http.Request, string) {
	t.Helper()
	token, _, err := auth.createSession("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, target, body)
	addCookie(req, sessionCookieName, token)
	return req, token
}

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
		`<label class="muted">세계관</label>`,
		`<label class="muted">제목</label>`,
		`<label class="muted">스타일</label>`,
		`<label class="muted">캐릭터 이름</label>`,
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

func TestPublicStoryRoomAccessibleWithoutLogin(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	auth := newTestAuthStore(t)
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store, auth: auth, authRequired: true, storyEnabled: true}

	req := httptest.NewRequest(http.MethodGet, "/stories/"+id, nil)
	rec := httptest.NewRecorder()
	srv.handle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()
	for _, want := range []string{
		`읽기 전용`,
		`로그인하면 진행, 질문, 진행권, 관리 기능을 사용할 수 있습니다.`,
		`로그인하면 진행권을 받고 직접 입력할 수 있습니다.`,
		`로그인하면 질문을 보낼 수 있습니다.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in public story room", want)
		}
	}
	for _, forbidden := range []string{
		`data-story-submit`,
		`data-story-question-textarea`,
		`name="action" value="claim"`,
		`name="action" value="release"`,
		`name="action" value="update"`,
		`name="action" value="edit_turn"`,
		`name="action" value="rollback_turn"`,
		`name="action" value="export_bundle"`,
		`name="action" value="recover_store"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("unexpected %q in public story room", forbidden)
		}
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/stories/"+id+"/status", nil)
	statusRec := httptest.NewRecorder()
	srv.handle(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status response %d: %s", statusRec.Code, statusRec.Body.String())
	}
	var status storyStatusResponse
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.CanDrive || status.CanQuestion {
		t.Fatalf("expected read-only status payload, got %#v", status)
	}
}

func TestUnauthenticatedStoryInputIsRejected(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	auth := newTestAuthStore(t)
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	beforeJobs, err := store.listJobs(id)
	if err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store, auth: auth, authRequired: true, storyEnabled: true}
	form := url.Values{}
	form.Set("csrf_token", "csrf-public")
	form.Set("turn_id", "1")
	form.Set("idempotency_key", "public-input-idem")
	form.Set("choice_id", "A")
	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addCookie(req, csrfCookieName, "csrf-public")
	rec := httptest.NewRecorder()
	srv.handle(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/login" {
		t.Fatalf("unexpected redirect location %q", got)
	}
	afterJobs, err := store.listJobs(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterJobs) != len(beforeJobs) {
		t.Fatalf("expected no job creation, before=%d after=%d", len(beforeJobs), len(afterJobs))
	}
}

func TestAuthenticatedStoryProgressionStillWorks(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	auth := newTestAuthStore(t)
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	srv := &webServer{stories: store, auth: auth, authRequired: true, storyEnabled: true}

	sessionToken, _, err := auth.createSession("user_admin")
	if err != nil {
		t.Fatal(err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/stories/"+id, nil)
	addCookie(getReq, sessionCookieName, sessionToken)
	getRec := httptest.NewRecorder()
	srv.handle(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected GET status %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), `data-story-submit-kind="input"`) {
		t.Fatalf("missing input form in authenticated story room")
	}

	csrf := "csrf-auth"
	form := url.Values{}
	form.Set("csrf_token", csrf)
	form.Set("turn_id", "1")
	form.Set("idempotency_key", "auth-input-idem")
	form.Set("choice_id", "A")
	postReq := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Accept", "application/json")
	addCookie(postReq, sessionCookieName, sessionToken)
	addCookie(postReq, csrfCookieName, csrf)
	postRec := httptest.NewRecorder()
	srv.handle(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("unexpected POST status %d: %s", postRec.Code, postRec.Body.String())
	}
	var resp storyTaskResponse
	if err := json.Unmarshal(postRec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.StoryID != id || resp.JobID == "" || resp.JobType != "story_turn" || resp.TurnID != 2 || !resp.IsProcessing {
		t.Fatalf("unexpected accepted payload: %#v", resp)
	}
}

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
		`story-composer`,
		`mode-tabs`,
		`progress-loader`,
		`data-story-progress`,
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
		`[hidden] { display:none !important; }`,
		`<strong data-story-progress-label>입력 가능</strong>`,
		`data-step-label="ready"`,
		`이번 턴에서 확인된 정보`,
		`누적 확인 정보`,
		`현재 턴`,
		`입력/질문`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing %q in rendered story room", want)
		}
	}
	if strings.Contains(htmlOpen, `story-progress-steps`) {
		t.Fatalf("unexpected visible progress step list in rendered story room")
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
	for _, want := range []string{
		`.story-room-shell,`,
		`.story-room-header > *,`,
		`.story-room-grid > *,`,
		`.current-turn-column > *,`,
		`.turn-sidebar > *,`,
		`.dossier-stack,`,
		`.dossier-panel,`,
		`.story-composer,`,
		`.story-composer-panel,`,
		`.story-choice-submit-panel { display:grid; gap:12px; }`,
		`.story-choice-form { margin:0; }`,
		`.story-input-form { display:grid; gap:12px; margin:0; }`,
		`.story-custom-input-panel,`,
		`.story-input-divider { height:1px; background:rgba(17,27,24,.08); }`,
		`.story-composer-panel-head { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; justify-content:space-between; }`,
		`.story-composer-actions { margin-top:0; }`,
		`.story-progress,`,
		`.turn-timeline a,`,
		`.previous-turn,`,
		`.choice-card,`,
		`.choice-card-archived { min-width:0; }`,
		`.story-room-grid { display:grid; grid-template-columns:240px minmax(0, 1fr) 320px; grid-template-areas:"timeline current dossier"; gap:24px; align-items:start; }`,
		`.current-turn-panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:18px; scroll-margin-top:18px; }`,
		`.current-turn-flow { display:grid; gap:14px; border-left:4px solid var(--info); background:rgba(49,95,153,.05); border-radius:6px; padding:14px 14px 14px 16px; }`,
		`.turn-timeline-link { display:grid; gap:3px; border:1px solid var(--line); border-left:3px solid var(--deep); border-radius:6px; padding:9px 10px; background:rgba(255,255,255,.72); color:var(--ink); text-decoration:none; box-shadow:none; word-break:keep-all; overflow-wrap:anywhere; }`,
		`.previous-turn summary::after { content:"열기";`,
		`.choice-card-risk { font:12px ui-sans-serif, system-ui, sans-serif; color:var(--accent); word-break:keep-all; overflow-wrap:anywhere; }`,
		`.story-progress-message { margin:0; font:15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }`,
		`.story-progress:not([aria-busy="true"]) .progress-loader-copy { display:none; }`,
		`.story-choice-submit-panel { display:grid; gap:12px; }`,
		`.story-custom-input-panel { display:grid; gap:12px; }`,
		`.story-input-divider { height:1px; background:rgba(17,27,24,.08); }`,
		`.story-choice-submit-panel .choice-list { gap:8px; }`,
		`.story-choice-form .choice-card { background:rgba(255,255,255,.72); border-color:var(--line); }`,
		`.story-custom-input-panel textarea { min-height:120px; }`,
		`@media (max-width:820px){`,
		`.story-choice-submit-head,`,
		`.story-composer-panel-head{align-items:flex-start;}`,
		`.story-choice-submit-panel .choice-list,`,
		`.story-custom-input-panel,`,
		`.story-composer-actions{width:100%;}`,
		`.story-choice-form .choice-card,`,
		`.story-custom-input-panel textarea,`,
		`.story-composer-actions > *{width:100%;}`,
		`.current-turn-body{display:flex; flex-direction:column;}`,
		`.current-turn-flow{order:1;}`,
		`.current-turn-body .scene{order:2;}`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing css safeguard %q in rendered story room", want)
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

func TestStoryRoomHectorImportOmitsDuplicatedTurnTitles(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	sourceRel := filepath.Join("lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
	sourcePath := filepath.Join("..", "..", "packs", sourceRel)
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(packRoot, sourceRel)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, existed, err := store.importHector("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatalf("expected fresh hector import")
	}
	html := renderStoryRoomHTML(t, &webServer{stories: store}, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if !strings.Contains(html, `세션 기록`) {
		t.Fatalf("missing fallback session title for duplicated Hector turn")
	}
	for _, want := range []string{
		`turn-timeline-title">Turn 19`,
		`current-turn-title">Turn 19`,
		`previous-turn-title">Turn 19`,
	} {
		if strings.Contains(html, want) {
			t.Fatalf("unexpected duplicated Hector title fragment %q", want)
		}
	}
}

func TestStoryRoomProcessingStateOmitsMetaRefreshAndShowsBusyProgress(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createStoryWithPrologueJob("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if strings.Contains(html, `http-equiv="refresh"`) {
		t.Fatalf("unexpected meta refresh in processing story room")
	}
	for _, want := range []string{
		`id="story-room"`,
		`data-story-input-panel`,
		`aria-busy="true"`,
		`id="story-progress" role="status" aria-live="polite" aria-atomic="true" aria-busy="true"`,
		`data-story-progress`,
		`GM 생성 중`,
		`완료되면 자동으로 최신 턴이 갱신됩니다.`,
		`잠시만 기다려 주세요.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in processing story room", want)
		}
	}
	if strings.Contains(html, `active job:`) {
		t.Fatalf("unexpected active job copy in processing story room")
	}
	if strings.Contains(html, `story-progress-steps`) {
		t.Fatalf("unexpected progress step list in processing story room")
	}
}

func TestStoryRoomAssetRouteServed(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/assets/story-room.js", nil)
	rec := httptest.NewRecorder()
	srv.handle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("unexpected content type %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`async function submitForm(form, submitter)`,
		`submitForm(form, event.submitter || null);`,
		`data-story-submit`,
		`WeakMap`,
		`captureInitialControlState`,
		`restoreInitialControlState`,
		`initialControlState`,
		`control.disabled = initial.disabled`,
		`ariaDisabled`,
		`pollStatus`,
		`inputPanel.setAttribute('aria-busy'`,
		`data-story-progress-meta`,
		`const metaNode = progress.querySelector('[data-story-progress-meta]');`,
		`metaNode.hidden = !visible;`,
		`const actionURL = new URL(form.action, window.location.href);`,
		`const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;`,
		`Object.fromEntries(data.entries())`,
		`requestPayload`,
		`responsePayload`,
		`'Content-Type': 'application/json'`,
		`JSON.stringify(requestPayload)`,
		`credentials: 'include'`,
		`'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || '',`,
		`progress.dataset.stepIndex = '0';`,
		`progress.dataset.stepLabel = 'queued';`,
		`statusLabel.textContent = friendlyStepLabel('queued');`,
		`setStep('queued');`,
		`제출 응답을 JSON으로 받지 못했습니다`,
		`fetch(activeTask.status_url`,
		`function getReloadTarget(payload)`,
		`function scheduleStoryReload(payload)`,
		`window.history.replaceState(null, '', target);`,
		`window.location.reload()`,
		`새 내용이 준비되었습니다. 자동으로 최신 화면을 불러옵니다.`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in story-room.js asset", want)
		}
	}
}

func TestStoryInputReturnsJSONAndStatus(t *testing.T) {
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
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	choiceID := turns[0].Choices[0].ID

	token := "csrf-story-input"
	form := url.Values{}
	form.Set("csrf_token", token)
	form.Set("turn_id", "1")
	form.Set("idempotency_key", "input-json-idem")
	form.Set("choice_id", choiceID)
	form.Set("mode", "action")
	form.Set("custom_text", "")

	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryInput(rec, req, id)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var resp storyTaskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.StoryID != id || resp.JobID == "" || resp.JobType != "story_turn" || resp.TurnID != 2 || !strings.HasSuffix(resp.StatusURL, "/stories/"+id+"/status") || resp.NextPollMS <= 0 || !resp.IsProcessing {
		t.Fatalf("unexpected input JSON response: %#v", resp)
	}
	if resp.StepLabel != "queued" || resp.StepIndex != 0 {
		t.Fatalf("unexpected step response: %#v", resp)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/stories/"+id+"/status", nil)
	statusReq = withUser(statusReq, &authUser{ID: "user_admin", Role: "admin"})
	statusRec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryStatus(statusRec, statusReq, id)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status response code %d: %s", statusRec.Code, statusRec.Body.String())
	}
	var status storyStatusResponse
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.IsProcessing || status.ActiveJobID == "" || status.ActiveJobType != "story_turn" || status.StepLabel == "" {
		t.Fatalf("unexpected status payload: %#v", status)
	}
}

func TestStoryQuestionReturnsJSONAndStatus(t *testing.T) {
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

	token := "csrf-story-question"
	form := url.Values{}
	form.Set("csrf_token", token)
	form.Set("turn_id", "1")
	form.Set("idempotency_key", "question-json-idem")
	form.Set("question", "지금 무슨 상황이야?")

	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/question", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryQuestion(rec, req, id)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var resp storyTaskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.StoryID != id || resp.JobID == "" || resp.JobType != "question_answer" || resp.TurnID != 1 || !strings.HasSuffix(resp.StatusURL, "/stories/"+id+"/status") || resp.NextPollMS <= 0 || !resp.IsProcessing {
		t.Fatalf("unexpected question JSON response: %#v", resp)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/stories/"+id+"/status", nil)
	statusReq = withUser(statusReq, &authUser{ID: "user_admin", Role: "admin"})
	statusRec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryStatus(statusRec, statusReq, id)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("unexpected status response code %d: %s", statusRec.Code, statusRec.Body.String())
	}
	var status storyStatusResponse
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.IsProcessing || status.ActiveJobID != resp.JobID || status.ActiveJobType != "question_answer" || len(status.PendingQuestions) == 0 {
		t.Fatalf("unexpected question status payload: %#v", status)
	}
}

func TestStoryInputAndQuestionReturnJSONBodyTokenWithoutCSRF(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	inputID, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	questionID, err := store.createDemoStory("user_admin", "르네의 질문", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	turns, err := store.readTurns(inputID)
	if err != nil {
		t.Fatal(err)
	}
	choiceID := turns[0].Choices[0].ID

	run := func(path string, body map[string]string) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
		rec := httptest.NewRecorder()
		srv := &webServer{stories: store}
		if strings.HasSuffix(path, "/question") {
			srv.handleStoryQuestion(rec, req, questionID)
		} else {
			srv.handleStoryInput(rec, req, inputID)
		}
		return rec
	}

	inputRec := run("/stories/"+inputID+"/input", map[string]string{
		"csrf_token":      "csrf-body-token",
		"turn_id":         "1",
		"idempotency_key": "csrf-json-idem",
		"choice_id":       choiceID,
		"mode":            "action",
		"custom_text":     "test",
	})
	if inputRec.Code != http.StatusAccepted {
		t.Fatalf("unexpected input status %d: %s", inputRec.Code, inputRec.Body.String())
	}

	questionRec := run("/stories/"+questionID+"/question", map[string]string{
		"csrf_token":      "csrf-body-token",
		"turn_id":         "1",
		"idempotency_key": "question-json-idem",
		"question":        "무슨 일?",
	})
	if questionRec.Code != http.StatusAccepted {
		t.Fatalf("unexpected question status %d: %s", questionRec.Code, questionRec.Body.String())
	}
}

func TestStoryTaskJSONAllowsHeaderAndBodyCsrfWithoutCookieMatch(t *testing.T) {
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
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	choiceID := turns[0].Choices[0].ID

	body := map[string]string{
		"csrf_token":      "csrf-form-token",
		"turn_id":         "1",
		"idempotency_key": "header-csrf-idem",
		"choice_id":       choiceID,
		"mode":            "action",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-CSRF-Token", "csrf-form-token")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryInput(rec, req, id)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var resp storyTaskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.JobID == "" || resp.JobType != "story_turn" || resp.StatusURL == "" {
		t.Fatalf("unexpected JSON submit response: %#v", resp)
	}
}

func TestStoryTaskJSONRejectsMalformedBody(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(`{"csrf_token":`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryInput(rec, req, id)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStoryTaskFormPostStillRequiresCookieMatch(t *testing.T) {
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
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	choiceID := turns[0].Choices[0].ID

	form := url.Values{}
	form.Set("csrf_token", "csrf-form-token")
	form.Set("turn_id", "1")
	form.Set("idempotency_key", "form-csrf-idem")
	form.Set("choice_id", choiceID)
	form.Set("mode", "action")

	req := httptest.NewRequest(http.MethodPost, "/stories/"+id+"/input", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "csrf-wrong"})
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleStoryInput(rec, req, id)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "forbidden") {
		t.Fatalf("expected forbidden response, got %q", rec.Body.String())
	}
}

func TestNewStoryStillRequiresStandardCSRF(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("csrf_token", "csrf-form-token")
	form.Set("title", "새 이야기")
	form.Set("style", "조사극")
	form.Set("character_name", "르네")
	form.Set("traits", "테스트")

	req := httptest.NewRequest(http.MethodPost, "/stories/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-CSRF-Token", "csrf-form-token")
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	(&webServer{stories: store}).handleNewStory(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "forbidden") {
		t.Fatalf("expected forbidden response, got %q", rec.Body.String())
	}
}

func TestStoryRoomAdminTurnControlsBlockWhileGMJobActive(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createStoryWithPrologueJob("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if strings.Contains(html, `name="action" value="edit_turn"`) || strings.Contains(html, `name="action" value="rollback_turn"`) {
		t.Fatalf("admin turn controls should be hidden while GM job is active")
	}
	if !strings.Contains(html, `편집과 롤백을 막습니다`) {
		t.Fatalf("missing GM blocking note in admin panel")
	}
}

func TestStoryRoomAdminLifecycleButtonsFollowStoryStatus(t *testing.T) {
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
	srv := &webServer{stories: store}

	activeHTML := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	for _, want := range []string{
		`name="action" value="archive"`,
		`name="action" value="delete"`,
		`보관`,
		`삭제`,
		`번들 내보내기`,
		`저장소 복구`,
	} {
		if !strings.Contains(activeHTML, want) {
			t.Fatalf("missing %q in active admin panel", want)
		}
	}
	if strings.Contains(activeHTML, `name="action" value="restore"`) {
		t.Fatalf("restore should not be shown for active story")
	}

	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	m.Status = "archived"
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	archivedHTML := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if !strings.Contains(archivedHTML, `name="action" value="restore"`) {
		t.Fatalf("missing restore action in archived admin panel")
	}
	if !strings.Contains(archivedHTML, `복구`) {
		t.Fatalf("missing restore label in archived admin panel")
	}
	if strings.Contains(archivedHTML, `name="action" value="archive"`) {
		t.Fatalf("archive should not be shown for archived story")
	}
	if !strings.Contains(archivedHTML, `name="action" value="delete"`) {
		t.Fatalf("delete should remain available for archived story")
	}

	m.Status = "deleted"
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	deletedHTML := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if !strings.Contains(deletedHTML, `name="action" value="restore"`) {
		t.Fatalf("missing restore action in deleted admin panel")
	}
	if !strings.Contains(deletedHTML, `복구`) {
		t.Fatalf("missing restore label in deleted admin panel")
	}
	if strings.Contains(deletedHTML, `name="action" value="delete"`) {
		t.Fatalf("delete should be hidden for deleted story")
	}
}

func TestStoryRoomRecoveryControlsAndStatusPanel(t *testing.T) {
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
	srv := &webServer{stories: store}

	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "?recovery_status=recovered&recovery_checked=events.jsonl,turns.jsonl,qa.jsonl&recovery_repaired=turns.jsonl,qa.jsonl&recovery_lock_removed=true")
	for _, want := range []string{
		`name="action" value="recover_store"`,
		`Store recovery`,
		`Recovery status: <span class="badge">recovered</span>`,
		`<code>events.jsonl</code>`,
		`<code>turns.jsonl</code>`,
		`<code>qa.jsonl</code>`,
		`Repaired items:`,
		`Stale lock.json was removed.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in recovery panel", want)
		}
	}
}

func TestStoryRoomHidesTurnNavWithoutTurnsAndShowsRecoveryAndExportPanels(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	store, err := openStoryStore(storyRoot, packRoot)
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
	m.Phase = "failed_waiting_retry"
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "?exported=/tmp/bundle/story.zip&export_status=draft_pending&export_draft_target=drafts/storylets/story_123.md")

	for _, want := range []string{
		`GM 생성 실패`,
		`name="action" value="resume"`,
		`name="action" value="cancel"`,
		`Export handoff`,
		`Bundle exported to <code>/tmp/bundle/story.zip</code>`,
		`Draft creation is pending/manual via the admin writer path.`,
		`Target draft: <code>drafts/storylets/story_123.md</code>`,
		`<span class="badge">draft_pending</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in failed-job story room", want)
		}
	}
	if strings.Contains(html, `href="#turn-0"`) {
		t.Fatalf("unexpected turn-0 dock link in no-turn story room")
	}
	if strings.Contains(html, `aria-label="turn list"`) {
		t.Fatalf("unexpected turn nav in no-turn story room")
	}
	if !strings.Contains(html, `href="#input-panel"`) {
		t.Fatalf("missing input-panel dock link in no-turn story room")
	}
}

func TestStoryRoomShowsLatestTurnFirst(t *testing.T) {
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
	if err := store.appendChoice(id, &authUser{ID: "user_admin", Role: "admin"}, "", "action", "베이르가 아닌 루세라에 있어야 한다고 주장한다"); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}

	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")

	firstCurrent := strings.Index(html, `class="current-turn-panel" id="turn-2"`)
	firstPrevious := strings.Index(html, `class="previous-turns-panel panel"`)
	firstOlder := strings.Index(html, `class="previous-turn" id="turn-1"`)
	if firstCurrent == -1 || firstPrevious == -1 || firstOlder == -1 {
		t.Fatalf("missing turn sections in story room: current=%d previous=%d old=%d", firstCurrent, firstPrevious, firstOlder)
	}
	if firstCurrent > firstPrevious {
		t.Fatalf("expected current turn before previous turns, got current at %d after previous section at %d", firstCurrent, firstPrevious)
	}
	if strings.Contains(html, `class="previous-turn" id="turn-2"`) {
		t.Fatalf("latest turn should not be rendered inside previous turns")
	}
	if strings.Contains(html, `class="previous-turn" id="turn-1" open`) {
		t.Fatalf("previous turn should be collapsed by default")
	}
	if rawISO := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`); rawISO.MatchString(html) {
		t.Fatal("unexpected raw ISO timestamp in rendered story room")
	}
	formattedKST := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}`)
	if got := len(formattedKST.FindAllString(html, -1)); got < 2 {
		t.Fatalf("expected localized timestamps for current and previous turns, got %d matches", got)
	}
}

func TestStoryRoomFormatsQuestionAndTurnTimestampsInKST(t *testing.T) {
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
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-idem-create", "A", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if err := store.askQuestion(id, &authUser{ID: "user_admin", Role: "admin"}, "루세라는 어디야?"); err != nil {
		t.Fatal(err)
	}

	html := renderStoryRoomHTML(t, &webServer{stories: store}, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if rawISO := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`); rawISO.MatchString(html) {
		t.Fatal("unexpected raw ISO timestamp in rendered story room")
	}
	formattedKST := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}`)
	if got := len(formattedKST.FindAllString(html, -1)); got < 3 {
		t.Fatalf("expected localized timestamps for current, previous, and QA entries, got %d matches", got)
	}
}

func TestAdminUsersTemplateWiresCSRF(t *testing.T) {
	srv := &webServer{}
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req = withUser(req, &authUser{ID: "user_admin", Role: "admin"})
	rec := httptest.NewRecorder()
	users := []map[string]any{
		{
			"username":        "alice",
			"display_name":    "Alice",
			"role":            "friend",
			"status":          "active",
			"last_login_at":   "",
			"active_sessions": 2,
			"id":              "user_alice",
		},
	}
	srv.render(rec, req, "Admin Users", adminUsersTemplate, map[string]any{
		"Base":      "",
		"User":      &authUser{ID: "user_admin", Role: "admin"},
		"Users":     users,
		"CSRFToken": "csrf-test",
	})
	html := rec.Body.String()
	for _, want := range []string{
		`name="csrf_token" value="csrf-test"`,
		`name="action" value="create"`,
		`name="action" value="update"`,
		`name="action" value="reset"`,
		`name="action" value="revoke"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in admin users template", want)
		}
	}
}
