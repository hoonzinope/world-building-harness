package world

import (
	"os"
	"path/filepath"
	"strings"
)

func Status(ctx *Context) map[string]any {
	contentCount := countMarkdown(filepath.Join(ctx.Root, "content"))
	draftCount := countMarkdown(filepath.Join(ctx.Root, "drafts"))
	runCount := 0
	entries, _ := os.ReadDir(filepath.Join(ctx.Root, "runs"))
	for _, e := range entries {
		if e.IsDir() && e.Name() != "inbox" {
			runCount++
		}
	}
	return map[string]any{
		"world_id":      ctx.ID,
		"root":          ctx.Root,
		"registry_root": ctx.RegistryRoot,
		"summary": map[string]any{
			"content_documents": contentCount,
			"active_drafts":     draftCount,
			"runs":              runCount,
		},
	}
}

func countMarkdown(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == "archive" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			count++
		}
		return nil
	})
	return count
}
