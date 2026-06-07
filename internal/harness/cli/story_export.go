package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func cmdStoryExportDraft(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("story.export-draft.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bundleFlag := fs.String("bundle", "", "bundle path")
	targetDraft := fs.String("target-draft", "", "target draft path")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if strings.TrimSpace(*bundleFlag) == "" {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", "--bundle is required", nil))
	}
	bundlePath, err := filepath.Abs(*bundleFlag)
	if err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	info, err := os.Stat(bundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", "bundle path does not exist", nil))
		}
		return emit(failEnvelope("story.export-draft", ctx, "IO_ERROR", err.Error(), nil))
	}
	if !info.IsDir() {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", "bundle path must be a directory", nil))
	}
	storyletPath := filepath.Join(bundlePath, "storylet.md")
	storyletBytes, err := os.ReadFile(storyletPath)
	if err != nil {
		if os.IsNotExist(err) {
			return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", "bundle is missing storylet.md", nil))
		}
		return emit(failEnvelope("story.export-draft", ctx, "IO_ERROR", err.Error(), nil))
	}
	storyletDoc, err := parseMarkdown("storylet.md", storyletBytes)
	if err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	exportManifest, _ := readOptionalJSONMap(filepath.Join(bundlePath, "export_manifest.json"))
	sourceManifest, _ := readOptionalJSONMap(filepath.Join(bundlePath, "source_manifest.json"))
	storyID := firstNonEmpty(
		jsonMapString(exportManifest, "story_id"),
		jsonMapString(sourceManifest, "id"),
		storyletDoc.ID(),
	)
	if strings.TrimSpace(storyID) == "" {
		return emit(failEnvelope("story.export-draft", ctx, "INVALID_ARGUMENT", "bundle does not identify a story_id", nil))
	}
	title := firstNonEmpty(
		jsonMapString(sourceManifest, "title"),
		headingTitle(storyletDoc.Body),
		metaString(storyletDoc.Meta, "title"),
		storyID,
	)
	targetRel := strings.TrimSpace(*targetDraft)
	if targetRel == "" {
		targetRel = jsonMapString(exportManifest, "draft_target_suggestion")
	}
	if targetRel == "" {
		targetRel = filepath.ToSlash(filepath.Join("drafts", "storylets", storyID+".md"))
	}
	targetAbs, targetRel, err := resolveStoryletDraftTarget(ctx.Root, targetRel)
	if err != nil {
		code := "PATH_SCOPE_DENIED"
		if strings.Contains(err.Error(), "PATH_NOT_MARKDOWN") {
			code = "PATH_NOT_MARKDOWN"
		}
		return emit(failEnvelope("story.export-draft", ctx, code, err.Error(), nil))
	}
	rid, runDir, err := createRun(ctx, "story.export-draft")
	if err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "IO_ERROR", err.Error(), nil))
	}
	meta := map[string]any{}
	for k, v := range storyletDoc.Meta {
		meta[k] = v
	}
	meta["schema_version"] = docSchemaVersion
	meta["id"] = storyID
	meta["type"] = "storylet"
	meta["status"] = "draft"
	meta["title"] = title
	meta["tags"] = []string{}
	meta["created_at"] = nowDate()
	meta["updated_at"] = nowDate()
	meta["related"] = []string{}
	meta["relationships"] = []any{}
	meta["source_run_id"] = rid
	meta["change_type"] = "create"
	meta["target_id"] = nil
	meta["retcon_reason"] = nil
	out, err := buildMarkdown(meta, storyletDoc.Body)
	if err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "INTERNAL_ERROR", err.Error(), nil))
	}
	if err := writeFileAtomic(targetAbs, out, 0o644); err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "IO_ERROR", err.Error(), nil))
	}
	draftHash := sha256Bytes(out)
	summary := map[string]any{
		"bundle_path": bundlePath,
		"draft_path":  targetRel,
		"draft_hash":  draftHash,
		"story_id":    storyID,
		"status":      "draft_created",
	}
	if err := writeJSON(filepath.Join(runDir, "summary.json"), summary); err != nil {
		return emit(failEnvelope("story.export-draft", ctx, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("story.export-draft", ctx, rid, summary, nil, []string{"world_read_draft", "world_validate_draft", "world_diff_draft"}))
}

func cmdStoryRecover(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("story.recover.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	storyID := fs.String("story-id", "", "story id")
	storiesRoot := fs.String("stories-root", "", "stories root")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("story.recover", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if strings.TrimSpace(*storyID) == "" {
		return emit(failEnvelope("story.recover", ctx, "INVALID_ARGUMENT", "--story-id is required", nil))
	}
	root, err := resolveStoriesRoot(ctx, *storiesRoot)
	if err != nil {
		return emit(failEnvelope("story.recover", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	store, err := openStoryStore(root, ctx.Root)
	if err != nil {
		return emit(failEnvelope("story.recover", ctx, "IO_ERROR", err.Error(), nil))
	}
	report, err := store.recoverStory(strings.TrimSpace(*storyID))
	if err != nil {
		return emit(failEnvelope("story.recover", ctx, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("story.recover", ctx, nil, map[string]any{
		"story_id":        report.StoryID,
		"recovery_status": report.RecoveryStatus,
		"checked_files":   report.CheckedFiles,
		"repaired_items":  report.RepairedItems,
		"lock_removed":    report.LockRemoved,
	}, nil, nil))
}

func resolveStoriesRoot(ctx *WorldContext, explicit string) (string, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return normalizeRoot(trimmed, true)
	}
	if ctx == nil {
		return "", fmt.Errorf("--stories-root is required")
	}
	candidates := []string{
		filepath.Join(ctx.Root, "data", "stories"),
		filepath.Join(ctx.Root, "stories"),
	}
	var existing []string
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			existing = append(existing, candidate)
		}
	}
	switch len(existing) {
	case 1:
		return normalizeRoot(existing[0], true)
	case 0:
		return "", fmt.Errorf("--stories-root is required")
	default:
		return "", fmt.Errorf("--stories-root is ambiguous; pass it explicitly")
	}
}

func localScopeFlag(args []string, def string) string {
	fs := flag.NewFlagSet("scope", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scope := fs.String("scope", def, "scope")
	_ = fs.Parse(args)
	return *scope
}

func readOptionalJSONMap(path string) (map[string]any, error) {
	var out map[string]any
	if err := readJSON(path, &out); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func jsonMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func resolveStoryletDraftTarget(root, targetRel string) (string, string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRel)))
	if clean == "." || clean == "" {
		return "", "", fmt.Errorf("PATH_SCOPE_DENIED: target draft must be inside drafts/storylets/")
	}
	if filepath.IsAbs(clean) {
		return "", "", fmt.Errorf("PATH_SCOPE_DENIED: target draft must be inside drafts/storylets/")
	}
	if !strings.HasSuffix(strings.ToLower(clean), ".md") {
		return "", "", fmt.Errorf("PATH_NOT_MARKDOWN: target draft must end in .md")
	}
	slash := filepath.ToSlash(clean)
	if !strings.HasPrefix(slash, "drafts/storylets/") {
		return "", "", fmt.Errorf("PATH_SCOPE_DENIED: target draft must be inside drafts/storylets/")
	}
	abs, rel, err := safeRel(root, slash)
	if err != nil {
		return "", "", fmt.Errorf("PATH_SCOPE_DENIED: %w", err)
	}
	return abs, rel, nil
}
