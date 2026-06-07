package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStoryRoomAdminTurnControlsBlockWhileGMJobActive(t *testing.T) {
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
	srv := &webServer{stories: store}
	html := renderStoryRoomHTML(t, srv, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if strings.Contains(html, `name="action" value="edit_turn"`) || strings.Contains(html, `name="action" value="rollback_turn"`) {
		t.Fatalf("admin turn controls should be hidden while GM job is active")
	}
	if !strings.Contains(html, `편집과 롤백을 막습니다`) {
		t.Fatalf("missing GM blocking note in admin panel")
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
		`보관`,
		`삭제`,
		`번들 내보내기`,
		`저장소 복구`,
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
	if !strings.Contains(archivedHTML, `복구`) {
		t.Fatalf("missing restore label in archived admin panel")
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
	if !strings.Contains(deletedHTML, `복구`) {
		t.Fatalf("missing restore label in deleted admin panel")
	}
	if strings.Contains(deletedHTML, `name="action" value="delete"`) {
		t.Fatalf("delete should be hidden for deleted story")
	}
}
