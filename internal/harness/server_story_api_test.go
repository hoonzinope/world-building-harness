package harness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestStoryInputCompletionStatusIncludesCompletedJob(t *testing.T) {
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

	if _, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-complete-idem", choiceID, "action", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
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
	if status.IsProcessing {
		t.Fatalf("expected completed status, got %#v", status)
	}
	if status.CurrentTurn != 2 {
		t.Fatalf("expected current turn to advance, got %#v", status)
	}
	if status.LastCompletedJobID == "" || status.LastCompletedJobType != "story_turn" || status.LastCompletedJobTurnID != 2 || status.LastCompletedJobStatus != "completed" {
		t.Fatalf("unexpected completed job payload: %#v", status)
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
