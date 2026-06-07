package world

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonzi/world-harness/internal/harness/core"
	"gopkg.in/yaml.v3"
)

func Resolve(c CommonFlags) (*Context, *core.ToolError) {
	if c.World != "" && c.Root != "" {
		return nil, &core.ToolError{Code: "INVALID_ARGUMENT", Message: "--world and --root are mutually exclusive"}
	}
	if c.Root != "" {
		root, err := NormalizeRoot(c.Root, true)
		if err != nil {
			return nil, &core.ToolError{Code: "WORLD_NOT_FOUND", Message: err.Error()}
		}
		h, _ := ReadHarness(root)
		id := c.WorldID
		if h.WorldID != "" {
			id = h.WorldID
		}
		if id == "" {
			return nil, &core.ToolError{Code: "INVALID_ARGUMENT", Message: "--root mode requires --world-id or harness.yaml provenance"}
		}
		if h.WorldID == "" {
			h = DefaultHarness(id)
		}
		return &Context{ID: id, Root: root, RegistryRoot: root, Harness: h}, nil
	}
	if c.World == "" {
		return nil, &core.ToolError{Code: "INVALID_ARGUMENT", Message: "--world or --root is required"}
	}
	reg, _, err := LoadRegistry(c.Registry)
	if err != nil {
		return nil, &core.ToolError{Code: "REGISTRY_NOT_FOUND", Message: err.Error()}
	}
	item, ok := reg.Worlds[c.World]
	if !ok {
		return nil, &core.ToolError{Code: "WORLD_NOT_FOUND", Message: "world is not registered", Details: map[string]any{"world": c.World}}
	}
	root, err := NormalizeRoot(item.Root, true)
	if err != nil {
		return nil, &core.ToolError{Code: "WORLD_NOT_FOUND", Message: err.Error(), Details: map[string]any{"root": item.Root}}
	}
	h, err := ReadHarness(root)
	if err != nil {
		return nil, &core.ToolError{Code: "WORLD_NOT_FOUND", Message: "harness.yaml not found or invalid", Details: err.Error()}
	}
	if h.WorldID != "" && h.WorldID != c.World {
		return nil, &core.ToolError{Code: "WORLD_NOT_FOUND", Message: "registry world id does not match harness.yaml", Details: map[string]any{"registry": c.World, "harness": h.WorldID}}
	}
	return &Context{ID: c.World, Root: root, RegistryRoot: root, Harness: h}, nil
}

func NormalizeRoot(path string, mustExist bool) (string, error) {
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

func ReadHarness(root string) (Harness, error) {
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

func Init(root, worldID string) (*Context, error) {
	if worldID == "" {
		return nil, fmt.Errorf("--world-id is required")
	}
	abs, err := NormalizeRoot(root, false)
	if err != nil {
		return nil, err
	}
	h := DefaultHarness(worldID)
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
		if err := core.EnsureDir(filepath.Join(abs, d)); err != nil {
			return nil, err
		}
	}
	b, err := yaml.Marshal(h)
	if err != nil {
		return nil, err
	}
	if err := core.WriteFileAtomic(filepath.Join(abs, "harness.yaml"), b, 0o644); err != nil {
		return nil, err
	}
	return &Context{ID: worldID, Root: abs, RegistryRoot: abs, Harness: h}, nil
}
