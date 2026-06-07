package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryLifecycleActionsAuditAndGuardActiveJobs(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createDemoStory("user_admin", "르네의 이야기", "생존극", "르네", "루세라의 간호사")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.adminUpdateStory(id, "user_admin", "paused", "user_friend"); err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "paused" || m.ActiveDriverID != "user_friend" {
		t.Fatalf("generic admin update failed: %#v", m)
	}

	if err := store.archiveStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.restoreStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}
	if err := store.deleteStory(id, "user_admin"); err != nil {
		t.Fatal(err)
	}

	m, err = store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "deleted" {
		t.Fatalf("delete did not tombstone story: %#v", m)
	}
	if _, err := os.Stat(filepath.Join(storyRoot, id, "turns.jsonl")); err != nil {
		t.Fatalf("story data was removed: %v", err)
	}

	var eventTypes []string
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if typ, _ := ev["type"].(string); typ != "" {
			eventTypes = append(eventTypes, typ)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"story_status_changed", "story_archived", "story_restored", "story_deleted"} {
		found := false
		for _, got := range eventTypes {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing audit event %q in %#v", want, eventTypes)
		}
	}

	busyID, err := store.createStoryWithPrologueJob("user_admin", "busy", "조사극", "르네", "")
	if err != nil {
		t.Fatal(err)
	}
	busyManifest, err := store.readManifest(busyID)
	if err != nil {
		t.Fatal(err)
	}
	busyManifest.Phase = "waiting_for_choice"
	if err := writeJSONAtomic(filepath.Join(storyRoot, busyID, "manifest.json"), busyManifest); err != nil {
		t.Fatal(err)
	}
	if err := store.archiveStory(busyID, "user_admin"); err == nil {
		t.Fatal("expected archive to be blocked by active GM job")
	}
	if _, err := store.exportStoryBundle(busyID, &authUser{ID: "user_admin", Role: "admin"}); err == nil {
		t.Fatal("expected export to be blocked by active GM job")
	}
	if err := store.editCurrentTurn(busyID, "user_admin", "edited", "summary"); err == nil {
		t.Fatal("expected edit to be blocked by active GM job")
	}
	if err := store.rollbackStoryToTurn(busyID, "user_admin", 1); err == nil {
		t.Fatal("expected rollback to be blocked by active GM job")
	}
}

func TestStoryAdminTurnEditAndRollbackUpdateProjection(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
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

	if err := store.editCurrentTurn(id, "user_admin", "edited scene body", "edited current situation"); err != nil {
		t.Fatal(err)
	}
	m, err := store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentTurn != 2 || m.LatestSummary != "edited current situation" {
		t.Fatalf("manifest not updated after edit: %#v", m)
	}
	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 || turns[1].SceneBody != "edited scene body" || turns[1].CurrentSituation != "edited current situation" {
		t.Fatalf("turn projection not updated after edit: %#v", turns)
	}
	summary, err := os.ReadFile(filepath.Join(storyRoot, id, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(summary)); got != "edited current situation" {
		t.Fatalf("summary.md = %q", string(summary))
	}

	if err := store.rollbackStoryToTurn(id, "user_admin", 1); err != nil {
		t.Fatal(err)
	}
	m, err = store.readManifest(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.CurrentTurn != 1 || m.LatestSummary == "edited current situation" {
		t.Fatalf("manifest not rolled back: %#v", m)
	}
	turns, err = store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].TurnID != 1 {
		t.Fatalf("turn projection not rolled back: %#v", turns)
	}
	summary, err = os.ReadFile(filepath.Join(storyRoot, id, "summary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(summary)); got != turns[0].CurrentSituation {
		t.Fatalf("summary.md was not rolled back: %q", string(summary))
	}

	var eventTypes []string
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if typ, _ := ev["type"].(string); typ != "" {
			eventTypes = append(eventTypes, typ)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !containsString(eventTypes, "turn_edited_by_admin") {
		t.Fatalf("missing edit audit event: %#v", eventTypes)
	}
	if !containsString(eventTypes, "story_rolled_back_by_admin") {
		t.Fatalf("missing rollback audit event: %#v", eventTypes)
	}
}
