package story

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportStoryBundleWritesHandoffMetadataAndAuditEvent(t *testing.T) {
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
	bundleDir, err := store.exportStoryBundle(id, &Actor{ID: "user_admin", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"source_manifest.json", "turn_hashes.json", "storylet.md", "summary.md", "export_manifest.json"} {
		if _, err := os.Stat(filepath.Join(bundleDir, name)); err != nil {
			t.Fatalf("missing bundle file %q: %v", name, err)
		}
	}

	var exportManifest map[string]any
	if err := readJSON(filepath.Join(bundleDir, "export_manifest.json"), &exportManifest); err != nil {
		t.Fatal(err)
	}
	if got := exportManifest["story_id"]; got != id {
		t.Fatalf("story_id = %#v", got)
	}
	if got := exportManifest["status"]; got != "draft_pending" {
		t.Fatalf("status = %#v", got)
	}
	if got := exportManifest["exported_by"]; got != "user_admin" {
		t.Fatalf("exported_by = %#v", got)
	}
	if got := exportManifest["draft_target_suggestion"]; got != filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md")) {
		t.Fatalf("draft target = %#v", got)
	}
	if got := exportManifest["turn_hashes_path"]; got != "turn_hashes.json" {
		t.Fatalf("turn_hashes_path = %#v", got)
	}
	if got := exportManifest["storylet_path"]; got != "storylet.md" {
		t.Fatalf("storylet_path = %#v", got)
	}
	if got := exportManifest["summary_path"]; got != "summary.md" {
		t.Fatalf("summary_path = %#v", got)
	}

	var sawExportEvent bool
	if err := readStoryJSONL(filepath.Join(storyRoot, id, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_export_handoff" {
			sawExportEvent = true
			if ev["actor_id"] != "user_admin" {
				t.Fatalf("actor_id = %#v", ev["actor_id"])
			}
			if ev["bundle_path"] != bundleDir {
				t.Fatalf("bundle_path = %#v", ev["bundle_path"])
			}
			if ev["target_draft_suggestion"] != filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md")) {
				t.Fatalf("target_draft_suggestion = %#v", ev["target_draft_suggestion"])
			}
			if ev["status"] != "draft_pending" {
				t.Fatalf("status = %#v", ev["status"])
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawExportEvent {
		t.Fatal("missing story_export_handoff audit event")
	}
}
