package harness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing %q in rendered story room", want)
		}
	}
	for _, forbidden := range []string{
		`async function submitForm(form)`,
		`const actionURL = new URL(form.action, window.location.href);`,
		`fetch(activeTask.status_url`,
	} {
		if strings.Contains(htmlOpen, forbidden) {
			t.Fatalf("unexpected inline story-room JS %q in rendered story room", forbidden)
		}
	}
	for _, want := range []string{
		`data-story-submit-kind="input"`,
		`data-story-custom-textarea`,
		`name="mode"`,
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
		`data-story-progress`,
		`GM 생성 중`,
		`완료되면 새 내용 표시 버튼으로 최신 턴을 갱신할 수 있습니다.`,
		`보통 10초-2분`,
		`active job:`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in processing story room", want)
		}
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
		`async function submitForm(form)`,
		`data-story-submit`,
		`WeakMap`,
		`captureInitialControlState`,
		`restoreInitialControlState`,
		`initialControlState`,
		`control.disabled = initial.disabled`,
		`ariaDisabled`,
		`pollStatus`,
		`inputPanel.setAttribute('aria-busy'`,
		`const actionURL = new URL(form.action, window.location.href);`,
		`const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;`,
		`Object.fromEntries(data.entries())`,
		`requestPayload`,
		`responsePayload`,
		`'Content-Type': 'application/json'`,
		`JSON.stringify(requestPayload)`,
		`credentials: 'include'`,
		`'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || '',`,
		`제출 응답을 JSON으로 받지 못했습니다`,
		`fetch(activeTask.status_url`,
		`새 내용 표시`,
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

	firstLatest := strings.Index(html, `details class="turn-card" id="turn-2"`)
	firstOlder := strings.Index(html, `details class="turn-card" id="turn-1"`)
	if firstLatest == -1 || firstOlder == -1 {
		t.Fatalf("missing turn cards in story room: turn-2=%d turn-1=%d", firstLatest, firstOlder)
	}
	if firstLatest > firstOlder {
		t.Fatalf("expected latest turn first, got turn-2 at %d after turn-1 at %d", firstLatest, firstOlder)
	}
	if !strings.Contains(html, `details class="turn-card" id="turn-2" open`) {
		t.Fatalf("latest turn is not rendered open")
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
