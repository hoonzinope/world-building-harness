package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func saveTelegramIdea(ctx *WorldContext, update telegramUpdate, text string) (string, error) {
	receivedAt := telegramMessageTime(update)
	id := fmt.Sprintf("%s-telegram-%d", receivedAt.UTC().Format("20060102T150405Z"), update.Message.MessageID)
	if update.Message.MessageID == 0 {
		id = fmt.Sprintf("%s-telegram-%s", receivedAt.UTC().Format("20060102T150405Z"), shortHash(text))
	}
	rel := filepath.ToSlash(filepath.Join("ideas", "inbox", id+".md"))
	abs, clean, err := safeRel(ctx.Root, rel)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		rel = filepath.ToSlash(filepath.Join("ideas", "inbox", id+"-"+shortHash(text)+".md"))
		abs, clean, err = safeRel(ctx.Root, rel)
		if err != nil {
			return "", err
		}
	}
	body := strings.TrimSpace(text)
	content := strings.Join([]string{
		"---",
		"source: telegram",
		"pack_id: " + ctx.ID,
		fmt.Sprintf("message_id: %d", update.Message.MessageID),
		"received_at: " + receivedAt.UTC().Format(time.RFC3339),
		"stored_at: " + time.Now().UTC().Format(time.RFC3339),
		"---",
		"",
		body,
		"",
	}, "\n")
	return clean, writeFileAtomic(abs, []byte(content), 0o600)
}

func telegramMessageTime(update telegramUpdate) time.Time {
	if update.Message.Date > 0 {
		return time.Unix(update.Message.Date, 0)
	}
	return time.Now().UTC()
}

func shortHash(text string) string {
	hash := strings.TrimPrefix(sha256Bytes([]byte(text)), "sha256:")
	if len(hash) > 10 {
		return hash[:10]
	}
	return hash
}

func ideaPreview(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	inFrontmatter := false
	for i, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > 80 {
			return string([]rune(trimmed)[:80]) + "..."
		}
		return trimmed
	}
	return ""
}
