package server

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

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

	firstCurrent := strings.Index(html, `class="current-turn-panel" id="turn-2"`)
	firstPrevious := strings.Index(html, `class="previous-turns-panel panel"`)
	firstOlder := strings.Index(html, `class="previous-turn" id="turn-1"`)
	if firstCurrent == -1 || firstPrevious == -1 || firstOlder == -1 {
		t.Fatalf("missing turn sections in story room: current=%d previous=%d old=%d", firstCurrent, firstPrevious, firstOlder)
	}
	if firstCurrent > firstPrevious {
		t.Fatalf("expected current turn before previous turns, got current at %d after previous section at %d", firstCurrent, firstPrevious)
	}
	if strings.Contains(html, `class="previous-turn" id="turn-2"`) {
		t.Fatalf("latest turn should not be rendered inside previous turns")
	}
	if strings.Contains(html, `class="previous-turn" id="turn-1" open`) {
		t.Fatalf("previous turn should be collapsed by default")
	}
	if rawISO := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`); rawISO.MatchString(html) {
		t.Fatal("unexpected raw ISO timestamp in rendered story room")
	}
	formattedKST := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}`)
	if got := len(formattedKST.FindAllString(html, -1)); got < 2 {
		t.Fatalf("expected localized timestamps for current and previous turns, got %d matches", got)
	}
}

func TestStoryRoomFormatsQuestionAndTurnTimestampsInKST(t *testing.T) {
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
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitStoryInput(id, &authUser{ID: "user_admin", Role: "admin"}, 1, "input-idem-create", "A", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.processOneGMJob(context.Background(), mockGMProvider{}); err != nil {
		t.Fatal(err)
	}
	if err := store.askQuestion(id, &authUser{ID: "user_admin", Role: "admin"}, "루세라는 어디야?"); err != nil {
		t.Fatal(err)
	}

	html := renderStoryRoomHTML(t, &webServer{stories: store}, id, &authUser{ID: "user_admin", Role: "admin"}, "")
	if rawISO := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`); rawISO.MatchString(html) {
		t.Fatal("unexpected raw ISO timestamp in rendered story room")
	}
	formattedKST := regexp.MustCompile(`\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}`)
	if got := len(formattedKST.FindAllString(html, -1)); got < 3 {
		t.Fatalf("expected localized timestamps for current, previous, and QA entries, got %d matches", got)
	}
}
