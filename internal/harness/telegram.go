package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type telegramBot struct {
	token       string
	allowedChat string
	packsRoot   string
	defaultPack string
	repoRoot    string
	client      *http.Client
}

type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int    `json:"message_id"`
		Text      string `json:"text"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"message"`
}

func runTelegram(args []string) int {
	fs := flag.NewFlagSet("telegram", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	token := fs.String("token", os.Getenv("TELEGRAM_BOT_TOKEN"), "telegram bot token")
	allowedChat := fs.String("allowed-chat-id", os.Getenv("TELEGRAM_ALLOWED_CHAT_ID"), "allowed chat id")
	packsRoot := fs.String("packs-root", envDefault("WORLD_HARNESS_PACKS_ROOT", "packs"), "packs root")
	defaultPack := fs.String("default-pack", envDefault("WORLD_HARNESS_DEFAULT_PACK", "lumen-federation"), "default pack")
	repoRoot := fs.String("repo-root", envDefault("WORLD_HARNESS_REPO_ROOT", "."), "repo root")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *token == "" {
		fmt.Fprintln(os.Stderr, "TELEGRAM_BOT_TOKEN is required")
		return 2
	}
	bot := &telegramBot{
		token:       *token,
		allowedChat: *allowedChat,
		packsRoot:   *packsRoot,
		defaultPack: *defaultPack,
		repoRoot:    *repoRoot,
		client:      &http.Client{Timeout: 90 * time.Second},
	}
	return bot.loop()
}

func (b *telegramBot) loop() int {
	offset := 0
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			fmt.Fprintln(os.Stderr, "telegram getUpdates:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message.Text == "" {
				continue
			}
			chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
			if b.allowedChat != "" && chatID != b.allowedChat {
				_ = b.sendMessage(chatID, "이 봇은 허용된 채팅에서만 world-harness 명령을 처리합니다.")
				continue
			}
			reply := b.handle(update)
			if reply != "" {
				_ = b.sendMessage(chatID, reply)
			}
		}
	}
}

func (b *telegramBot) getUpdates(offset int) ([]telegramUpdate, error) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=60&offset=%d", url.PathEscape(b.token), offset)
	resp, err := b.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OK     bool             `json:"ok"`
		Result []telegramUpdate `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("telegram returned not ok: %s", string(body))
	}
	return parsed.Result, nil
}

func (b *telegramBot) sendMessage(chatID, text string) error {
	if len(text) > 3900 {
		text = text[:3900] + "\n..."
	}
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", url.PathEscape(b.token))
	resp, err := b.client.PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *telegramBot) handle(update telegramUpdate) string {
	text := strings.TrimSpace(update.Message.Text)
	switch {
	case text == "/start" || text == "/help":
		return strings.Join([]string{
			"world-harness Telegram commands:",
			"/packs - pack 목록",
			"/status [pack] - 상태 확인",
			"/search [pack] <query> - canon 문서 검색",
			"/codex [pack] <request> - Codex에게 draft/story 작업 요청",
			"/draft [pack] <type> <id> | <title> | <body> - Codex 없이 draft 생성",
		}, "\n")
	case text == "/packs":
		return b.cmdPacks()
	case strings.HasPrefix(text, "/status"):
		return b.cmdStatus(strings.TrimSpace(strings.TrimPrefix(text, "/status")))
	case strings.HasPrefix(text, "/search"):
		return b.cmdSearch(strings.TrimSpace(strings.TrimPrefix(text, "/search")))
	case strings.HasPrefix(text, "/draft"):
		return b.cmdDraft(strings.TrimSpace(strings.TrimPrefix(text, "/draft")))
	case strings.HasPrefix(text, "/codex"):
		return b.cmdCodex(update, strings.TrimSpace(strings.TrimPrefix(text, "/codex")))
	default:
		return "알 수 없는 명령입니다. /help 를 확인하세요."
	}
}

func (b *telegramBot) cmdPacks() string {
	server := &webServer{packsRoot: b.packsRoot}
	packs := server.packs()
	if len(packs) == 0 {
		return "등록된 pack이 없습니다."
	}
	lines := []string{"packs:"}
	for _, pack := range packs {
		lines = append(lines, fmt.Sprintf("- %s (%s)", pack["title"], pack["id"]))
	}
	return strings.Join(lines, "\n")
}

func (b *telegramBot) packFromArg(arg string) (string, string) {
	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return b.defaultPack, ""
	}
	candidate := fields[0]
	if _, err := os.Stat(filepath.Join(b.packsRoot, candidate, "harness.yaml")); err == nil {
		return candidate, strings.TrimSpace(strings.TrimPrefix(arg, candidate))
	}
	return b.defaultPack, arg
}

func (b *telegramBot) cmdStatus(arg string) string {
	pack, _ := b.packFromArg(arg)
	ctx, err := (&webServer{packsRoot: b.packsRoot}).packContext(pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	summary := worldStatus(ctx)["summary"].(map[string]any)
	return fmt.Sprintf("%s\ncontent: %v\ndrafts: %v\nruns: %v", ctx.ID, summary["content_documents"], summary["active_drafts"], summary["runs"])
}

func (b *telegramBot) cmdSearch(arg string) string {
	pack, query := b.packFromArg(arg)
	ctx, err := (&webServer{packsRoot: b.packsRoot}).packContext(pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	if strings.TrimSpace(query) == "" {
		return "검색어를 입력하세요."
	}
	results, err := searchDocuments(ctx, "content", query)
	if err != nil {
		return err.Error()
	}
	if len(results) > 8 {
		results = results[:8]
	}
	lines := []string{fmt.Sprintf("검색 결과: %s", query)}
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("- %s\n  %s", r["title"], r["path"]))
	}
	if len(results) == 0 {
		lines = append(lines, "결과 없음")
	}
	return strings.Join(lines, "\n")
}

func (b *telegramBot) cmdDraft(arg string) string {
	pack, rest := b.packFromArg(arg)
	parts := strings.Split(rest, "|")
	if len(parts) != 3 {
		return "형식: /draft [pack] <type> <id> | <title> | <body>"
	}
	left := strings.Fields(strings.TrimSpace(parts[0]))
	if len(left) != 2 {
		return "type과 id를 입력하세요."
	}
	ctx, err := (&webServer{packsRoot: b.packsRoot}).packContext(pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	titlePath, titleHash, err := stageText(ctx, "title", strings.TrimSpace(parts[1]))
	if err != nil {
		return err.Error()
	}
	bodyPath, bodyHash, err := stageText(ctx, "body", strings.TrimSpace(parts[2]))
	if err != nil {
		return err.Error()
	}
	code := cmdDraftCreate(commonFlags{}, ctx, []string{"--change-type", "create", "--type", left[0], "--id", left[1], "--title-file", titlePath, "--title-hash", titleHash, "--body-file", bodyPath, "--body-hash", bodyHash})
	if code != 0 {
		return "draft 생성에 실패했습니다. 서버 로그를 확인하세요."
	}
	return fmt.Sprintf("draft 생성 요청을 처리했습니다: %s / %s", pack, left[1])
}

func stageText(ctx *WorldContext, kind, text string) (string, string, error) {
	rid, _, err := createRun(ctx, "input.stage")
	if err != nil {
		return "", "", err
	}
	ext := ".txt"
	if kind == "body" {
		ext = ".md"
	}
	rel := filepath.ToSlash(filepath.Join("runs", "inbox", rid+"-"+kind+ext))
	abs, _, err := safeRel(ctx.Root, rel)
	if err != nil {
		return "", "", err
	}
	b := []byte(text)
	return rel, sha256Bytes(b), writeFileAtomic(abs, b, 0o600)
}

func (b *telegramBot) cmdCodex(update telegramUpdate, arg string) string {
	pack, request := b.packFromArg(arg)
	if strings.TrimSpace(request) == "" {
		return "요청 내용을 입력하세요."
	}
	packRoot, _ := filepath.Abs(filepath.Join(b.packsRoot, pack))
	prompt := fmt.Sprintf(`You are operating world-harness for a private worldbuilding pack.

Pack root: %s
Pack id: %s

Use the local world-tool CLI. Do not edit content/ directly. Create or update drafts first, validate them, and summarize the draft_path and validation status. If the user asks for story material, create storylet or event/character/place drafts as appropriate.

Telegram user: %s (%d)
Request:
%s`, packRoot, pack, update.Message.From.Username, update.Message.From.ID, request)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", "exec", "-C", b.repoRoot, "--add-dir", packRoot, "--sandbox", "danger-full-access", "--skip-git-repo-check", prompt)
	cmd.Env = append(os.Environ(), "WORLD_TOOL_REGISTRY="+filepath.Join(b.repoRoot, ".worlds.yaml"))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Codex 실행 실패: %v\n%s", err, tail(stderr.String(), 1200))
	}
	return tail(out.String(), 3500)
}

func tail(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}
