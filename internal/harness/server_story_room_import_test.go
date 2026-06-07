package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryRoomHectorImportOmitsDuplicatedTurnTitles(t *testing.T) {
	root := t.TempDir()
	storyRoot := filepath.Join(root, "stories")
	packRoot := filepath.Join(root, "packs")
	sourceRel := filepath.Join("lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
	sourcePath := filepath.Join("..", "..", "packs", sourceRel)
	b, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(packRoot, sourceRel)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := openStoryStore(storyRoot, packRoot)
	if err != nil {
		t.Fatal(err)
	}
	id, existed, err := store.importHector("user_admin")
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatalf("expected fresh hector import")
	}
	html := renderStoryRoomHTML(t, &webServer{stories: store}, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if !strings.Contains(html, `세션 기록`) {
		t.Fatalf("missing fallback session title for duplicated Hector turn")
	}
	for _, want := range []string{
		`turn-timeline-title">Turn 19`,
		`current-turn-title">Turn 19`,
		`previous-turn-title">Turn 19`,
	} {
		if strings.Contains(html, want) {
			t.Fatalf("unexpected duplicated Hector title fragment %q", want)
		}
	}
}
