package world

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/core"
)

func PackContext(packsRoot, pack string) (*Context, error) {
	root, err := filepath.Abs(filepath.Join(packsRoot, pack))
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(root)
	h, err := ReadHarness(root)
	if err != nil {
		return nil, err
	}
	id := core.FirstNonEmpty(h.WorldID, pack)
	if id == "" {
		id = pack
	}
	return &Context{ID: id, Root: root, RegistryRoot: root, Harness: h}, nil
}

func Packs(packsRoot string) ([]map[string]any, error) {
	entries, err := os.ReadDir(packsRoot)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ctx, err := PackContext(packsRoot, e.Name())
		if err != nil {
			continue
		}
		status := Status(ctx)
		out = append(out, map[string]any{
			"id":      ctx.ID,
			"title":   PackTitle(ctx),
			"root":    ctx.Root,
			"summary": status["summary"],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return fmt.Sprint(out[i]["title"]) < fmt.Sprint(out[j]["title"])
	})
	return out, nil
}

func SafePackArg(arg string) (string, string) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", ""
	}
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		return "", ""
	}
	return parts[0], strings.TrimSpace(strings.TrimPrefix(arg, parts[0]))
}
