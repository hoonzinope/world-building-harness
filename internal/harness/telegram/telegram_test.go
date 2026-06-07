package telegram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTelegramPlainMessageStoresIdeaWithoutCodex(t *testing.T) {
	packsRoot := t.TempDir()
	ctx, err := initWorld(filepath.Join(packsRoot, "lumen-federation"), "lumen-federation")
	if err != nil {
		t.Fatal(err)
	}
	bot := &telegramBot{packsRoot: packsRoot, defaultPack: "lumen-federation"}
	var update telegramUpdate
	update.Message.MessageID = 42
	update.Message.Date = 1710000000
	update.Message.Text = "베이르 잔명 정정망에 작은 사건 메모"

	reply := bot.handle(update)
	if !strings.Contains(reply, "Codex는 실행하지 않았습니다") {
		t.Fatalf("expected no-codex reply, got %q", reply)
	}
	if !strings.Contains(reply, "ideas/inbox/") {
		t.Fatalf("expected idea path in reply, got %q", reply)
	}

	files, err := filepath.Glob(filepath.Join(ctx.Root, "ideas", "inbox", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one idea file, got %d", len(files))
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "source: telegram") || !strings.Contains(content, update.Message.Text) {
		t.Fatalf("idea content missing metadata or body:\n%s", content)
	}

	listReply := bot.cmdIdeas("")
	if !strings.Contains(listReply, "최근 idea inbox") || !strings.Contains(listReply, update.Message.Text) {
		t.Fatalf("expected idea list preview, got %q", listReply)
	}
}

func TestTelegramUnknownSlashCommandDoesNotStoreIdea(t *testing.T) {
	packsRoot := t.TempDir()
	ctx, err := initWorld(filepath.Join(packsRoot, "lumen-federation"), "lumen-federation")
	if err != nil {
		t.Fatal(err)
	}
	bot := &telegramBot{packsRoot: packsRoot, defaultPack: "lumen-federation"}
	var update telegramUpdate
	update.Message.MessageID = 43
	update.Message.Text = "/oops 메모처럼 보이지만 명령어 오타"

	reply := bot.handle(update)
	if !strings.Contains(reply, "알 수 없는 명령") {
		t.Fatalf("expected unknown command reply, got %q", reply)
	}
	files, err := filepath.Glob(filepath.Join(ctx.Root, "ideas", "inbox", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no idea files, got %d", len(files))
	}
}

func TestTelegramDraftCreatesPackDraftWithSourceRunID(t *testing.T) {
	packsRoot := t.TempDir()
	ctx, err := initWorld(filepath.Join(packsRoot, "lumen-federation"), "lumen-federation")
	if err != nil {
		t.Fatal(err)
	}
	bot := &telegramBot{packsRoot: packsRoot, defaultPack: "lumen-federation"}

	var update telegramUpdate
	update.Message.Text = "/draft character aria | Aria Prime | First draft body"
	reply := bot.handle(update)
	if !strings.Contains(reply, "draft 생성 요청을 처리했습니다") {
		t.Fatalf("expected draft success reply, got %q", reply)
	}

	files, err := filepath.Glob(filepath.Join(ctx.Root, "drafts", "characters", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one draft file, got %d", len(files))
	}
	doc, err := parseMarkdown("drafts/characters/"+filepath.Base(files[0]), mustReadFile(t, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := doc.Meta["title"].(string); got != "Aria Prime" {
		t.Fatalf("expected title from input, got %q", got)
	}
	if got, _ := doc.Meta["source_run_id"].(string); strings.TrimSpace(got) == "" {
		t.Fatal("expected non-empty source_run_id")
	}
	if !strings.Contains(doc.Body, "# Aria Prime") || !strings.Contains(doc.Body, "First draft body") {
		t.Fatalf("expected preserved legacy body format, got %q", doc.Body)
	}
	runID, _ := doc.Meta["source_run_id"].(string)
	summaryPath := filepath.Join(ctx.Root, "runs", runID, "summary.json")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("expected run summary to exist at %s: %v", summaryPath, err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
