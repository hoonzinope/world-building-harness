package harness

import (
	"path/filepath"
)

func issueCount(issues []Issue, severity string) int {
	n := 0
	for _, issue := range issues {
		if issue.Severity == severity {
			n++
		}
	}
	return n
}

func cmdDraftDiff(_ commonFlags, ctx *WorldContext, args []string) int {
	draft := draftFlag(args)
	if draft == "" {
		return emit(failEnvelope("draft.diff", ctx, "INVALID_ARGUMENT", "--draft is required", nil))
	}
	doc, err := readDocument(ctx, draft)
	if err != nil {
		return emit(failEnvelope("draft.diff", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	targetPath, targetExists, targetHash, blocked := targetForDraft(ctx, doc)
	if blocked != "" {
		data := map[string]any{"draft_path": doc.Path, "validation_status": "conflict"}
		return emit(blockedEnvelope("draft.diff", ctx, nil, blocked, data, []Issue{{Code: blocked, Severity: "conflict", Message: "target is not available", Path: doc.Path}}, validationActions("conflict")))
	}
	status, issues := validateDocument(ctx, doc, true)
	if hasIssueCode(issues, "MISSING_TARGET") {
		data := map[string]any{"draft_path": doc.Path, "validation_status": "conflict"}
		return emit(blockedEnvelope("draft.diff", ctx, nil, "MISSING_TARGET", data, issues, validationActions("conflict")))
	}
	rid, runDir, err := createRun(ctx, "draft.diff")
	if err != nil {
		return emit(failEnvelope("draft.diff", ctx, "IO_ERROR", err.Error(), nil))
	}
	draftHash := sha256Bytes([]byte(doc.Raw))
	patch := patchText(targetPath, targetHash, draftHash)
	patchHash := sha256Bytes([]byte(patch))
	if err := writeFileAtomic(filepath.Join(runDir, "diff.patch"), []byte(patch), 0o644); err != nil {
		return emit(failEnvelope("draft.diff", ctx, "IO_ERROR", err.Error(), nil))
	}
	data := map[string]any{
		"diff_run_id":       rid,
		"draft_path":        doc.Path,
		"draft_hash":        draftHash,
		"target_exists":     targetExists,
		"target_path":       targetPath,
		"target_base_hash":  nullableHash(targetHash),
		"patch_hash":        patchHash,
		"validation_status": status,
	}
	_ = writeJSON(filepath.Join(runDir, "summary.json"), data)
	return emit(okEnvelope("draft.diff", ctx, rid, data, issues, []string{"world_stage_input"}))
}
