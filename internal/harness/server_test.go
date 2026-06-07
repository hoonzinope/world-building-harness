package harness

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

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
	m.ActiveDriverID = "user_admin"
	if err := writeJSONAtomic(filepath.Join(storyRoot, id, "manifest.json"), m); err != nil {
		t.Fatal(err)
	}
	htmlAssigned := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")

	for _, want := range []string{
		`name="csrf_token" value="`,
		`name="action" value="claim"`,
		`name="action" value="update"`,
		`name="action" value="export_bundle"`,
		`name="mode"`,
		`value="question"`,
		`질문은 직접 입력에서 question 모드를 선택해 제출할 수 있습니다.`,
	} {
		if !strings.Contains(htmlOpen, want) {
			t.Fatalf("missing %q in rendered story room", want)
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
