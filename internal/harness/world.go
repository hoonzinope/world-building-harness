package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	Default string                   `yaml:"default,omitempty" json:"default,omitempty"`
	Worlds  map[string]RegistryWorld `yaml:"worlds" json:"worlds"`
}

type RegistryWorld struct {
	Title string `yaml:"title" json:"title"`
	Root  string `yaml:"root" json:"root"`
}

var contentDirs = map[string]string{
	"character":    "characters",
	"nation":       "nations",
	"organization": "organizations",
	"place":        "places",
	"event":        "events",
	"timeline":     "timeline",
	"magic":        "magic",
	"glossary":     "glossary",
}

var draftDirs = map[string]string{
	"character":    "characters",
	"nation":       "nations",
	"organization": "organizations",
	"place":        "places",
	"event":        "events",
	"timeline":     "timeline",
	"magic":        "magic",
	"glossary":     "glossary",
	"storylet":     "storylets",
}

func defaultHarness(worldID string) Harness {
	return Harness{
		SchemaVersion: "world-harness.v1",
		WorldID:       worldID,
		WorldRoot:     ".",
		ContentDir:    "content",
		DraftDir:      "drafts",
		RunDir:        "runs",
		InboxDir:      "runs/inbox",
		GraphDir:      "graph",
		ArchiveDir:    "archive",
	}
}

func registryPath(flagValue string, forWrite bool) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}
	if env := os.Getenv("WORLD_TOOL_REGISTRY"); env != "" {
		return filepath.Abs(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(home, ".opencrabs", "worlds.yaml"),
		filepath.Join(home, ".config", "world-tool", "worlds.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if forWrite {
		return candidates[0], nil
	}
	return candidates[0], nil
}

func loadRegistry(flagValue string) (Registry, string, error) {
	p, err := registryPath(flagValue, false)
	if err != nil {
		return Registry{}, "", err
	}
	reg := Registry{Worlds: map[string]RegistryWorld{}}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return reg, p, nil
	}
	if err != nil {
		return Registry{}, p, err
	}
	if err := yaml.Unmarshal(b, &reg); err != nil {
		return Registry{}, p, err
	}
	if reg.Worlds == nil {
		reg.Worlds = map[string]RegistryWorld{}
	}
	return reg, p, nil
}

func saveRegistry(flagValue string, reg Registry) (string, error) {
	p, err := registryPath(flagValue, true)
	if err != nil {
		return "", err
	}
	if reg.Worlds == nil {
		reg.Worlds = map[string]RegistryWorld{}
	}
	b, err := yaml.Marshal(reg)
	if err != nil {
		return "", err
	}
	if err := ensureDir(filepath.Dir(p)); err != nil {
		return "", err
	}
	return p, writeFileAtomic(p, b, 0o644)
}

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

func safeRel(root, rel string) (string, string, error) {
	if rel == "" {
		return "", "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", "", fmt.Errorf("path traversal is not allowed")
	}
	abs := filepath.Join(root, clean)
	rootEval, err := filepath.EvalSymlinks(root)
	if err != nil {
		rootEval = root
	}
	parent := filepath.Dir(abs)
	if evalParent, err := filepath.EvalSymlinks(parent); err == nil {
		if !pathInside(rootEval, evalParent) {
			return "", "", fmt.Errorf("path escapes world root")
		}
	}
	if evalAbs, err := filepath.EvalSymlinks(abs); err == nil {
		if !pathInside(rootEval, evalAbs) {
			return "", "", fmt.Errorf("path escapes world root")
		}
	}
	return abs, filepath.ToSlash(clean), nil
}

func pathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func acquireWorldLock(ctx *WorldContext, command string) (func(), error) {
	lockPath := filepath.Join(ctx.Root, "runs", ".lock")
	if err := ensureDir(filepath.Dir(lockPath)); err != nil {
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

func unresolvedRecovery(ctx *WorldContext) (string, bool) {
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

func requireMarkdownPath(rel string) error {
	if !strings.HasSuffix(strings.ToLower(rel), ".md") {
		return fmt.Errorf("path is not markdown")
	}
	return nil
}

func draftPath(docType, id string) (string, error) {
	dir, ok := draftDirs[docType]
	if !ok {
		return "", fmt.Errorf("unsupported document type %q", docType)
	}
	return filepath.ToSlash(filepath.Join("drafts", dir, id+".md")), nil
}

func contentPath(docType, id string) (string, error) {
	dir, ok := contentDirs[docType]
	if !ok {
		return "", fmt.Errorf("unsupported content document type %q", docType)
	}
	return filepath.ToSlash(filepath.Join("content", dir, id+".md")), nil
}

func createRun(ctx *WorldContext, command string) (string, string, error) {
	id := runID()
	dir := filepath.Join(ctx.Root, "runs", id)
	if err := ensureDir(dir); err != nil {
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
	if err := writeJSON(filepath.Join(dir, "run.json"), manifest); err != nil {
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

func worldStatus(ctx *WorldContext) map[string]any {
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

func registryWorldList(reg Registry) []map[string]any {
	keys := make([]string, 0, len(reg.Worlds))
	for id := range reg.Worlds {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, id := range keys {
		item := reg.Worlds[id]
		out = append(out, map[string]any{
			"id":         id,
			"world_id":   id,
			"title":      item.Title,
			"root":       item.Root,
			"is_default": id == reg.Default,
		})
	}
	return out
}
