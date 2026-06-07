package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

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
