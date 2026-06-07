package world

import (
	"os"
	"path/filepath"
	"time"

	"github.com/hoonzi/world-harness/internal/harness/core"
)

func CreateRun(ctx *Context, command string) (string, string, error) {
	id := core.RunID()
	dir := filepath.Join(ctx.Root, "runs", id)
	if err := core.EnsureDir(dir); err != nil {
		return "", "", err
	}
	manifest := map[string]any{
		"run_id":     id,
		"command":    command,
		"world_id":   ctx.ID,
		"root":       ctx.Root,
		"created_at": timeNowRFC3339(),
		"redacted":   true,
	}
	if err := core.WriteJSON(filepath.Join(dir, "run.json"), manifest); err != nil {
		return "", "", err
	}
	return id, dir, nil
}

func timeNowRFC3339() string {
	if fixed := os.Getenv("WORLD_TOOL_FIXED_TIME"); fixed != "" {
		return fixed
	}
	return time.Now().UTC().Format(time.RFC3339)
}
