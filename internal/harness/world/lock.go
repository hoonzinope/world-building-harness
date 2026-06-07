package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonzi/world-harness/internal/harness/core"
)

func AcquireWorldLock(ctx *Context, command string) (func(), error) {
	lockPath := filepath.Join(ctx.Root, "runs", ".lock")
	if err := core.EnsureDir(filepath.Dir(lockPath)); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("LOCK_BUSY: %s is locked", ctx.ID)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(f, "pid=%d\ncommand=%s\ncreated_at=%s\n", os.Getpid(), command, timeNowRFC3339())
	_ = f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}

func UnresolvedRecovery(ctx *Context) (string, bool) {
	entries, _ := os.ReadDir(filepath.Join(ctx.Root, "runs"))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "inbox" {
			continue
		}
		path := filepath.Join(ctx.Root, "runs", entry.Name(), "recovery.json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(b, &m) == nil && m["resolved"] != true && fmt.Sprint(m["recovery_status"]) != "resolved" {
			return entry.Name(), true
		}
	}
	return "", false
}
