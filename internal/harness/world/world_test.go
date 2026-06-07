package world

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonzi/world-harness/internal/harness/core"
)

func TestSafeRelRejectsLeafSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := core.EnsureDir(filepath.Join(root, "content")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "content", "escape.md")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SafeRel(root, "content/escape.md"); err == nil {
		t.Fatal("expected leaf symlink escape to be rejected")
	}
}

func TestUnresolvedRecoveryDetection(t *testing.T) {
	root := t.TempDir()
	ctx := &Context{ID: "test", Root: root, RegistryRoot: root}
	if err := core.EnsureDir(filepath.Join(root, "runs", "run-1")); err != nil {
		t.Fatal(err)
	}
	if err := core.WriteJSON(filepath.Join(root, "runs", "run-1", "recovery.json"), map[string]any{"recovery_status": "unresolved", "resolved": false}); err != nil {
		t.Fatal(err)
	}
	if rid, ok := UnresolvedRecovery(ctx); !ok || rid != "run-1" {
		t.Fatalf("expected unresolved recovery run-1, got %q %v", rid, ok)
	}
	if err := core.WriteJSON(filepath.Join(root, "runs", "run-1", "recovery.json"), map[string]any{"recovery_status": "resolved", "resolved": true}); err != nil {
		t.Fatal(err)
	}
	if rid, ok := UnresolvedRecovery(ctx); ok {
		t.Fatalf("did not expect recovery, got %q", rid)
	}
}
