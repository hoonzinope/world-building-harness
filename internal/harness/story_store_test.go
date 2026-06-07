package harness

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoryReadTurnsRepairsFinalPartialTail(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_partial_tail"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	turn1 := storyTurn{
		TurnID:           1,
		BranchID:         "branch_main",
		ActorID:          "user_admin",
		InputID:          "input_1",
		Source:           "setup",
		SceneTitle:       "Turn 1",
		SceneBody:        "first",
		CurrentSituation: "situation 1",
		CreatedAt:        "2026-06-07T00:00:00Z",
	}
	turn2 := storyTurn{
		TurnID:           2,
		BranchID:         "branch_main",
		ParentTurnID:     1,
		ActorID:          "user_admin",
		InputID:          "input_2",
		Source:           "choice",
		SceneTitle:       "Turn 2",
		SceneBody:        "second",
		CurrentSituation: "situation 2",
		CreatedAt:        "2026-06-07T00:01:00Z",
	}
	b1, err := json.Marshal(turn1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(turn2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), append(append(b1, '\n'), b2[:len(b2)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	turns, err := store.readTurns(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].TurnID != 1 {
		t.Fatalf("recovered turns = %#v", turns)
	}

	data, err := os.ReadFile(filepath.Join(dir, "turns.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, append(b1, '\n')) {
		t.Fatalf("turns.jsonl was not truncated: %q", string(data))
	}

	var sawRecovery bool
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_recovered" {
			sawRecovery = true
			if ev["recovered_path"] != "turns.jsonl" {
				t.Fatalf("recovered_path = %#v", ev["recovered_path"])
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawRecovery {
		t.Fatal("missing story_recovered event after truncating turns.jsonl")
	}
}

func TestStoryReadTurnsRejectsMalformedMiddleLine(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_bad_middle"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	good, err := json.Marshal(storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: "user_admin", InputID: "input_1", Source: "setup", SceneTitle: "Turn 1", SceneBody: "first", CurrentSituation: "situation 1", CreatedAt: "2026-06-07T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	bad := []byte(`{"turn_id":2,"branch_id":"branch_main"`)
	third, err := json.Marshal(storyTurn{TurnID: 3, BranchID: "branch_main", ParentTurnID: 2, ActorID: "user_admin", InputID: "input_3", Source: "choice", SceneTitle: "Turn 3", SceneBody: "third", CurrentSituation: "situation 3", CreatedAt: "2026-06-07T00:02:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	body.Write(good)
	body.WriteByte('\n')
	body.Write(bad)
	body.WriteByte('\n')
	body.Write(third)
	body.WriteByte('\n')
	if err := os.WriteFile(filepath.Join(dir, "turns.jsonl"), body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := store.readTurns(id); err == nil {
		t.Fatal("expected malformed middle line to fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "events.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("events.jsonl should not be created on middle-line failure: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "turns.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, bad) {
		t.Fatalf("turns.jsonl was unexpectedly repaired: %q", string(data))
	}
}

func TestStoryRecoverRepairsPartialTailsAndRemovesStaleLock(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	store, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_recover"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

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

	lock := map[string]any{"story_id": id, "reason": "stuck", "actor_id": "user_admin", "acquired_at": "2020-01-01T00:00:00Z"}
	if err := writeJSONAtomic(filepath.Join(dir, "lock.json"), lock); err != nil {
		t.Fatal(err)
	}

	report, err := store.recoverStory(id)
	if err != nil {
		t.Fatal(err)
	}
	if report.RecoveryStatus != "recovered" {
		t.Fatalf("recovery status = %q", report.RecoveryStatus)
	}
	if !report.LockRemoved {
		t.Fatal("expected stale lock to be removed")
	}
	if len(report.CheckedFiles) != 3 {
		t.Fatalf("checked files = %#v", report.CheckedFiles)
	}
	wantRepaired := map[string]bool{"events.jsonl": true, "turns.jsonl": true, "qa.jsonl": true}
	for _, got := range report.RepairedItems {
		delete(wantRepaired, got)
	}
	if len(wantRepaired) != 0 {
		t.Fatalf("missing repaired items: %#v", wantRepaired)
	}
	if _, err := os.Stat(filepath.Join(dir, "lock.json")); !os.IsNotExist(err) {
		t.Fatalf("lock.json should have been removed: %v", err)
	}

	var sawSummary bool
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		if ev["type"] == "story_recovered" && ev["story_id"] == id {
			if got := ev["recovery_status"]; got != "recovered" {
				t.Fatalf("recovery_status = %#v", got)
			}
			if got := ev["lock_removed"]; got != true {
				t.Fatalf("lock_removed = %#v", got)
			}
			if got := ev["checked_files"]; len(got.([]any)) != 3 {
				t.Fatalf("checked_files = %#v", got)
			}
			sawSummary = true
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !sawSummary {
		t.Fatal("missing story_recovered summary event")
	}
}

func TestStoryReadEventsRepairsFinalPartialTail(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	_, err := openStoryStore(storyRoot, filepath.Join(root, "packs"))
	if err != nil {
		t.Fatal(err)
	}
	id := "story_events_tail"
	dir := filepath.Join(storyRoot, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(map[string]any{"type": "turn_committed", "at": "2026-06-07T00:00:00Z", "turn": map[string]any{"turn_id": 1}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(map[string]any{"type": "turn_committed", "at": "2026-06-07T00:01:00Z", "turn": map[string]any{"turn_id": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), append(append(first, '\n'), second[:len(second)/2]...), 0o600); err != nil {
		t.Fatal(err)
	}

	var events []map[string]any
	if err := readStoryJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0]["type"] != "turn_committed" {
		t.Fatalf("unexpected repaired read: %#v", events)
	}
	events = nil
	if err := readJSONL(filepath.Join(dir, "events.jsonl"), func(b []byte) error {
		var ev map[string]any
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected recovery event in events.jsonl, got %#v", events)
	}
	if events[1]["type"] != "story_recovered" {
		t.Fatalf("missing story_recovered event: %#v", events)
	}
}
