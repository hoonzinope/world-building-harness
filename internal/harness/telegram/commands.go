package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/core"
)

func (b *telegramBot) handle(update telegramUpdate) string {
	text := strings.TrimSpace(update.Message.Text)
	switch {
	case text == "/start" || text == "/help":
		return strings.Join([]string{
			"world-harness Telegram commands:",
			"/packs - pack 목록",
			"/status [pack] - 상태 확인",
			"/search [pack] <query> - canon 문서 검색",
			"/ideas [pack] - 최근 Telegram idea inbox 확인",
			"/codex [pack] <request> - Codex에게 draft/story 작업 요청",
			"/draft [pack] <type> <id> | <title> | <body> - Codex 없이 draft 생성",
			"일반 메시지 - Codex를 실행하지 않고 ideas/inbox에 저장",
		}, "\n")
	case text == "/packs":
		return b.cmdPacks()
	case strings.HasPrefix(text, "/status"):
		return b.cmdStatus(strings.TrimSpace(strings.TrimPrefix(text, "/status")))
	case strings.HasPrefix(text, "/search"):
		return b.cmdSearch(strings.TrimSpace(strings.TrimPrefix(text, "/search")))
	case strings.HasPrefix(text, "/ideas"):
		return b.cmdIdeas(strings.TrimSpace(strings.TrimPrefix(text, "/ideas")))
	case strings.HasPrefix(text, "/draft"):
		return b.cmdDraft(strings.TrimSpace(strings.TrimPrefix(text, "/draft")))
	case strings.HasPrefix(text, "/codex"):
		return b.cmdCodex(update, strings.TrimSpace(strings.TrimPrefix(text, "/codex")))
	case strings.HasPrefix(text, "/"):
		return "알 수 없는 명령입니다. /help 를 확인하세요."
	default:
		return b.cmdIdea(update, text)
	}
}

func (b *telegramBot) cmdPacks() string {
	packs, err := Packs(b.packsRoot)
	if err != nil {
		return err.Error()
	}
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
	ctx, err := PackContext(b.packsRoot, pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	summary := worldStatus(ctx)["summary"].(map[string]any)
	return fmt.Sprintf("%s\ncontent: %v\ndrafts: %v\nruns: %v", ctx.ID, summary["content_documents"], summary["active_drafts"], summary["runs"])
}

func (b *telegramBot) cmdSearch(arg string) string {
	pack, query := b.packFromArg(arg)
	ctx, err := PackContext(b.packsRoot, pack)
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

func (b *telegramBot) cmdIdeas(arg string) string {
	pack, _ := b.packFromArg(arg)
	ctx, err := PackContext(b.packsRoot, pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	relRoot := filepath.ToSlash(filepath.Join("ideas", "inbox"))
	absRoot, _, err := safeRel(ctx.Root, relRoot)
	if err != nil {
		return err.Error()
	}
	entries, err := os.ReadDir(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "저장된 idea가 없습니다: " + pack
		}
		return err.Error()
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return "저장된 idea가 없습니다: " + pack
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) > 8 {
		names = names[:8]
	}
	lines := []string{"최근 idea inbox:"}
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join(relRoot, name))
		preview := ideaPreview(filepath.Join(absRoot, name))
		if preview == "" {
			lines = append(lines, "- "+rel)
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s\n  %s", rel, preview))
	}
	return strings.Join(lines, "\n")
}

func (b *telegramBot) cmdIdea(update telegramUpdate, text string) string {
	pack, body := b.packFromArg(text)
	if strings.TrimSpace(body) == "" {
		body = text
	}
	ctx, err := PackContext(b.packsRoot, pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	rel, err := saveTelegramIdea(ctx, update, body)
	if err != nil {
		return "idea 저장 실패: " + err.Error()
	}
	return fmt.Sprintf("idea 저장됨: %s\n%s\nCodex는 실행하지 않았습니다. 초안화가 필요하면 여기서 정리하거나 /codex를 명시적으로 사용하세요.", pack, rel)
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
	ctx, err := PackContext(b.packsRoot, pack)
	if err != nil {
		return "pack을 찾을 수 없습니다: " + pack
	}
	titlePath, _, err := stageText(ctx, "title", strings.TrimSpace(parts[1]))
	if err != nil {
		return err.Error()
	}
	bodyPath, _, err := stageText(ctx, "body", strings.TrimSpace(parts[2]))
	if err != nil {
		return err.Error()
	}
	titleBytes, err := os.ReadFile(titlePath)
	if err != nil {
		return err.Error()
	}
	bodyBytes, err := os.ReadFile(bodyPath)
	if err != nil {
		return err.Error()
	}
	meta := map[string]any{
		"schema_version": core.DocSchemaVersion,
		"id":             left[1],
		"type":           left[0],
		"status":         "draft",
		"title":          strings.TrimSpace(string(titleBytes)),
		"tags":           []string{},
		"created_at":     core.NowDate(),
		"updated_at":     core.NowDate(),
		"related":        []string{},
		"relationships":  []any{},
		"source_run_id":  "",
		"change_type":    "create",
		"target_id":      nil,
		"retcon_reason":  nil,
	}
	out, err := buildMarkdown(meta, strings.TrimSpace(string(bodyBytes)))
	if err != nil {
		return err.Error()
	}
	targetRel, err := draftPath(left[0], left[1])
	if err != nil {
		return err.Error()
	}
	targetAbs, _, err := safeRel(ctx.Root, targetRel)
	if err != nil {
		return err.Error()
	}
	if err := writeFileAtomic(targetAbs, out, 0o644); err != nil {
		return err.Error()
	}
	return fmt.Sprintf("draft 생성 요청을 처리했습니다: %s / %s", pack, left[1])
}
