package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

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
