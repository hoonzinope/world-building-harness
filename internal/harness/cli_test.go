package harness

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryExportDraftCreatesWrappedDraft(t *testing.T) {
	root := t.TempDir()
	if _, err := initWorld(root, "world_one"); err != nil {
		t.Fatal(err)
	}

	bundleDir := filepath.Join(t.TempDir(), "bundle")
	if err := os.MkdirAll(bundleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	mustWriteJSON(t, filepath.Join(bundleDir, "source_manifest.json"), map[string]any{
		"id":    "storylet_trade_conflict_001",
		"title": "거래 충돌",
	})
	mustWriteJSON(t, filepath.Join(bundleDir, "turn_hashes.json"), map[string]any{
		"story_id": "storylet_trade_conflict_001",
		"turns":    []any{},
	})
	mustWriteJSON(t, filepath.Join(bundleDir, "export_manifest.json"), map[string]any{
		"story_id":                "storylet_trade_conflict_001",
		"draft_target_suggestion": "drafts/storylets/storylet_trade_conflict_001.md",
		"status":                  "draft_pending",
	})
	mustWriteFile(t, filepath.Join(bundleDir, "summary.md"), []byte("bundle summary\n"))
	mustWriteFile(t, filepath.Join(bundleDir, "storylet.md"), []byte(`# 거래 충돌

bundle body
`))

	env, code := runToolJSON(t, []string{
		"story", "export-draft",
		"--root", root,
		"--world-id", "world_one",
		"--bundle", bundleDir,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !env.OK || env.Command != "story.export-draft" {
		t.Fatalf("env = %#v", env)
	}
	if got := env.Data.(map[string]any)["draft_path"]; got != "drafts/storylets/storylet_trade_conflict_001.md" {
		t.Fatalf("draft_path = %#v", got)
	}
	if got := env.Data.(map[string]any)["status"]; got != "draft_created" {
		t.Fatalf("status = %#v", got)
	}
	actions := env.AvailableActions
	if !containsString(actions, "world_read_draft") || !containsString(actions, "world_validate_draft") || !containsString(actions, "world_diff_draft") {
		t.Fatalf("available_actions = %#v", actions)
	}

	draftPath := filepath.Join(root, "drafts", "storylets", "storylet_trade_conflict_001.md")
	draftBytes, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	draftDoc, err := parseMarkdown("drafts/storylets/storylet_trade_conflict_001.md", draftBytes)
	if err != nil {
		t.Fatal(err)
	}
	if draftDoc.Type() != "storylet" || draftDoc.Status() != "draft" {
		t.Fatalf("draft doc = %#v", draftDoc.Meta)
	}
	if draftDoc.ID() != "storylet_trade_conflict_001" {
		t.Fatalf("draft id = %q", draftDoc.ID())
	}
	if draftDoc.Title() != "거래 충돌" {
		t.Fatalf("draft title = %q", draftDoc.Title())
	}
	if !strings.Contains(draftDoc.Body, "bundle body") {
		t.Fatalf("draft body = %q", draftDoc.Body)
	}
	if got := metaString(draftDoc.Meta, "source_run_id"); got == "" {
		t.Fatal("missing source_run_id")
	}
	if got := metaString(draftDoc.Meta, "change_type"); got != "create" {
		t.Fatalf("change_type = %q", got)
	}
	if got := metaStringList(draftDoc.Meta, "tags"); len(got) != 0 {
		t.Fatalf("tags = %#v", got)
	}

	var summary map[string]any
	if err := readJSON(filepath.Join(root, "runs", env.RunID.(string), "summary.json"), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["bundle_path"] != bundleDir {
		t.Fatalf("bundle_path = %#v", summary["bundle_path"])
	}
	if summary["draft_path"] != "drafts/storylets/storylet_trade_conflict_001.md" {
		t.Fatalf("summary draft_path = %#v", summary["draft_path"])
	}
	if summary["story_id"] != "storylet_trade_conflict_001" {
		t.Fatalf("story_id = %#v", summary["story_id"])
	}
	if summary["status"] != "draft_created" {
		t.Fatalf("summary status = %#v", summary["status"])
	}
}

func TestStoryExportDraftValidatesBundleAndTargetPath(t *testing.T) {
	t.Run("missing storylet", func(t *testing.T) {
		root := t.TempDir()
		if _, err := initWorld(root, "world_one"); err != nil {
			t.Fatal(err)
		}
		bundleDir := filepath.Join(t.TempDir(), "bundle")
		if err := os.MkdirAll(bundleDir, 0o700); err != nil {
			t.Fatal(err)
		}
		mustWriteJSON(t, filepath.Join(bundleDir, "source_manifest.json"), map[string]any{
			"id":    "storylet_missing",
			"title": "Missing Storylet",
		})
		env, code := runToolJSON(t, []string{
			"story", "export-draft",
			"--root", root,
			"--world-id", "world_one",
			"--bundle", bundleDir,
		})
		if code != 2 {
			t.Fatalf("exit code = %d", code)
		}
		if env.OK || env.Error == nil || env.Error.Code != "INVALID_ARGUMENT" {
			t.Fatalf("env = %#v", env)
		}
	})

	t.Run("invalid target", func(t *testing.T) {
		root := t.TempDir()
		if _, err := initWorld(root, "world_one"); err != nil {
			t.Fatal(err)
		}
		bundleDir := filepath.Join(t.TempDir(), "bundle")
		if err := os.MkdirAll(bundleDir, 0o700); err != nil {
			t.Fatal(err)
		}
		mustWriteJSON(t, filepath.Join(bundleDir, "source_manifest.json"), map[string]any{
			"id":    "storylet_trade_conflict_001",
			"title": "거래 충돌",
		})
		mustWriteJSON(t, filepath.Join(bundleDir, "export_manifest.json"), map[string]any{
			"story_id": "storylet_trade_conflict_001",
		})
		mustWriteFile(t, filepath.Join(bundleDir, "storylet.md"), []byte("bundle body\n"))
		env, code := runToolJSON(t, []string{
			"story", "export-draft",
			"--root", root,
			"--world-id", "world_one",
			"--bundle", bundleDir,
			"--target-draft", "content/storylet_trade_conflict_001.md",
		})
		if code != 2 {
			t.Fatalf("exit code = %d", code)
		}
		if env.OK || env.Error == nil || env.Error.Code != "PATH_SCOPE_DENIED" {
			t.Fatalf("env = %#v", env)
		}
	})
}

func TestStoryRecoverValidatesStoriesRoot(t *testing.T) {
	root := t.TempDir()
	if _, err := initWorld(root, "world_one"); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(t.TempDir(), "worlds.yaml")
	mustWriteFile(t, registryPath, []byte("default: world_one\nworlds:\n  world_one:\n    title: World One\n    root: "+root+"\n"))
	t.Setenv("WORLD_TOOL_REGISTRY", registryPath)

	env, code := runToolJSON(t, []string{
		"story", "recover",
		"--world", "world_one",
		"--story-id", "story_missing",
		"--json",
	})
	if code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	if env.OK || env.Error == nil || env.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("env = %#v", env)
	}
}

func TestStoryRecoverRepairsStoreAndReturnsReport(t *testing.T) {
	root := t.TempDir()
	if _, err := initWorld(root, "world_one"); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(t.TempDir(), "worlds.yaml")
	mustWriteFile(t, registryPath, []byte("default: world_one\nworlds:\n  world_one:\n    title: World One\n    root: "+root+"\n"))
	t.Setenv("WORLD_TOOL_REGISTRY", registryPath)

	storiesRoot := filepath.Join(root, "data", "stories")
	if err := os.MkdirAll(filepath.Join(storiesRoot, "story_recover"), 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(storiesRoot, "story_recover")
	event1, err := json.Marshal(map[string]any{"type": "story_created", "at": "2026-06-07T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	event2, err := json.Marshal(map[string]any{"type": "story_updated", "at": "2026-06-07T00:01:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), append(append(event1, '\n'), event2[:len(event2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}
	turn1 := storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: "user_admin", InputID: "input_1", Source: "setup", SceneTitle: "Turn 1", SceneBody: "one", CurrentSituation: "situation 1", CreatedAt: "2026-06-07T00:00:00Z"}
	turn2 := storyTurn{TurnID: 2, BranchID: "branch_main", ParentTurnID: 1, ActorID: "user_admin", InputID: "input_2", Source: "choice", SceneTitle: "Turn 2", SceneBody: "two", CurrentSituation: "situation 2", CreatedAt: "2026-06-07T00:01:00Z"}
	t1, err := json.Marshal(turn1)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := json.Marshal(turn2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), append(append(t1, '\n'), t2[:len(t2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}
	q1 := storyQuestion{ID: "q1", ActorID: "user_admin", Question: "where?", Answer: "here", TurnID: 1, CreatedAt: "2026-06-07T00:00:00Z"}
	q2 := storyQuestion{ID: "q2", ActorID: "user_admin", Question: "why?", Answer: "because", TurnID: 2, CreatedAt: "2026-06-07T00:01:00Z"}
	qb1, err := json.Marshal(q1)
	if err != nil {
		t.Fatal(err)
	}
	qb2, err := json.Marshal(q2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "qa.jsonl"), append(append(qb1, '\n'), qb2[:len(qb2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}
	lock := map[string]any{"story_id": "story_recover", "reason": "stale", "actor_id": "user_admin", "acquired_at": "2020-01-01T00:00:00Z"}
	mustWriteJSON(t, filepath.Join(dir, "lock.json"), lock)

	env, code := runToolJSON(t, []string{
		"story", "recover",
		"--world", "world_one",
		"--story-id", "story_recover",
		"--stories-root", storiesRoot,
		"--json",
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !env.OK || env.Command != "story.recover" {
		t.Fatalf("env = %#v", env)
	}
	data := env.Data.(map[string]any)
	if data["story_id"] != "story_recover" {
		t.Fatalf("story_id = %#v", data["story_id"])
	}
	if data["recovery_status"] != "recovered" {
		t.Fatalf("recovery_status = %#v", data["recovery_status"])
	}
	if got := data["lock_removed"]; got != true {
		t.Fatalf("lock_removed = %#v", got)
	}
	repaired := data["repaired_items"].([]any)
	if len(repaired) != 3 {
		t.Fatalf("repaired_items = %#v", repaired)
	}
}

func runToolJSON(t *testing.T, args []string) (Envelope, int) {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()
	code := Run(args)
	_ = w.Close()
	os.Stdout = origStdout
	out := <-done
	var env Envelope
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, string(out))
	}
	return env, code
}

func mustWriteJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, append(b, '\n'))
}

func mustWriteFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
