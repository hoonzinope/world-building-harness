package harness

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func resolveWorld(c commonFlags, command string) (*WorldContext, *ToolError) {
	if c.world != "" && c.root != "" {
		return nil, &ToolError{Code: "INVALID_ARGUMENT", Message: "--world and --root are mutually exclusive"}
	}
	if c.root != "" {
		root, err := normalizeRoot(c.root, true)
		if err != nil {
			return nil, &ToolError{Code: "WORLD_NOT_FOUND", Message: err.Error()}
		}
		h, _ := readHarness(root)
		id := c.worldID
		if h.WorldID != "" {
			id = h.WorldID
		}
		if id == "" {
			return nil, &ToolError{Code: "INVALID_ARGUMENT", Message: "--root mode requires --world-id or harness.yaml provenance"}
		}
		if h.WorldID == "" {
			h = defaultHarness(id)
		}
		return &WorldContext{ID: id, Root: root, RegistryRoot: root, Harness: h}, nil
	}
	if c.world == "" {
		return nil, &ToolError{Code: "INVALID_ARGUMENT", Message: "--world or --root is required"}
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return nil, &ToolError{Code: "REGISTRY_NOT_FOUND", Message: err.Error()}
	}
	item, ok := reg.Worlds[c.world]
	if !ok {
		return nil, &ToolError{Code: "WORLD_NOT_FOUND", Message: "world is not registered", Details: map[string]any{"world": c.world}}
	}
	root, err := normalizeRoot(item.Root, true)
	if err != nil {
		return nil, &ToolError{Code: "WORLD_NOT_FOUND", Message: err.Error(), Details: map[string]any{"root": item.Root}}
	}
	h, err := readHarness(root)
	if err != nil {
		return nil, &ToolError{Code: "WORLD_NOT_FOUND", Message: "harness.yaml not found or invalid", Details: err.Error()}
	}
	if h.WorldID != "" && h.WorldID != c.world {
		return nil, &ToolError{Code: "WORLD_NOT_FOUND", Message: "registry world id does not match harness.yaml", Details: map[string]any{"registry": c.world, "harness": h.WorldID}}
	}
	return &WorldContext{ID: c.world, Root: root, RegistryRoot: root, Harness: h}, nil
}

func normalizeRoot(path string, mustExist bool) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if mustExist {
		eval, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return "", err
		}
		return eval, nil
	}
	return filepath.Clean(abs), nil
}

func readHarness(root string) (Harness, error) {
	b, err := os.ReadFile(filepath.Join(root, "harness.yaml"))
	if err != nil {
		return Harness{}, err
	}
	var h Harness
	if err := yaml.Unmarshal(b, &h); err != nil {
		return Harness{}, err
	}
	if h.ContentDir == "" {
		h.ContentDir = "content"
	}
	if h.DraftDir == "" {
		h.DraftDir = "drafts"
	}
	if h.RunDir == "" {
		h.RunDir = "runs"
	}
	if h.InboxDir == "" {
		h.InboxDir = "runs/inbox"
	}
	if h.ArchiveDir == "" {
		h.ArchiveDir = "archive"
	}
	if h.GraphDir == "" {
		h.GraphDir = "graph"
	}
	return h, nil
}

func initWorld(root, worldID string) (*WorldContext, error) {
	if worldID == "" {
		return nil, fmt.Errorf("--world-id is required")
	}
	abs, err := normalizeRoot(root, false)
	if err != nil {
		return nil, err
	}
	h := defaultHarness(worldID)
	dirs := []string{
		"content",
		"drafts",
		"raw",
		"graph",
		"schema",
		"runs/inbox",
		"archive/accepted",
		"archive/rejected",
		"archive/deprecated",
	}
	for _, d := range contentDirs {
		dirs = append(dirs, filepath.Join("content", d))
	}
	for _, d := range draftDirs {
		dirs = append(dirs, filepath.Join("drafts", d))
	}
	for _, d := range dirs {
		if err := ensureDir(filepath.Join(abs, d)); err != nil {
			return nil, err
		}
	}
	b, err := yaml.Marshal(h)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(abs, "harness.yaml"), b, 0o644); err != nil {
		return nil, err
	}
	return &WorldContext{ID: worldID, Root: abs, RegistryRoot: abs, Harness: h}, nil
}
