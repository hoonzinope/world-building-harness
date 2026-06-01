package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeRelRejectsLeafSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureDir(filepath.Join(root, "content")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "content", "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := safeRel(root, "content/escape.md"); err == nil {
		t.Fatal("expected leaf symlink escape to be rejected")
	}
}

func TestUnresolvedRecoveryDetection(t *testing.T) {
	root := t.TempDir()
	ctx := &WorldContext{ID: "test", Root: root, RegistryRoot: root}
	if err := ensureDir(filepath.Join(root, "runs", "run-1")); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(root, "runs", "run-1", "recovery.json"), map[string]any{"recovery_status": "unresolved", "resolved": false}); err != nil {
		t.Fatal(err)
	}
	if rid, ok := unresolvedRecovery(ctx); !ok || rid != "run-1" {
		t.Fatalf("expected unresolved recovery run-1, got %q %v", rid, ok)
	}
	if err := writeJSON(filepath.Join(root, "runs", "run-1", "recovery.json"), map[string]any{"recovery_status": "resolved", "resolved": true}); err != nil {
		t.Fatal(err)
	}
	if rid, ok := unresolvedRecovery(ctx); ok {
		t.Fatalf("did not expect recovery, got %q", rid)
	}
}
