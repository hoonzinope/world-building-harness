package harness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
