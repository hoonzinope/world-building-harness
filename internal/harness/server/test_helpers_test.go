package server

import (
	"net/http"
	"net/http/httptest"
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
	StoryID                string                      `json:"story_id"`
	Status                 string                      `json:"status"`
	Phase                  string                      `json:"phase"`
	CurrentTurn            int                         `json:"current_turn"`
	ActiveJobID            string                      `json:"active_job_id"`
	ActiveJobType          string                      `json:"active_job_type"`
	ActiveJobStatus        string                      `json:"active_job_status"`
	LastCompletedJobID     string                      `json:"last_completed_job_id"`
	LastCompletedJobType   string                      `json:"last_completed_job_type"`
	LastCompletedJobTurnID int                         `json:"last_completed_job_turn_id"`
	LastCompletedJobStatus string                      `json:"last_completed_job_status"`
	IsProcessing           bool                        `json:"is_processing"`
	CanDrive               bool                        `json:"can_drive"`
	CanQuestion            bool                        `json:"can_question"`
	StatusLabel            string                      `json:"status_label"`
	ProgressMessage        string                      `json:"progress_message"`
	StepIndex              int                         `json:"step_index"`
	StepLabel              string                      `json:"step_label"`
	NextPollMS             int                         `json:"next_poll_ms"`
	PendingQuestions       []storyProgressQuestionView `json:"pending_questions"`
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
