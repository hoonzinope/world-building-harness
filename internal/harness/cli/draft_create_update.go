package cli

import (
	"flag"
	"io"
	"path/filepath"
	"strings"
)

func cmdDraftCreate(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("draft.create.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	changeType := fs.String("change-type", "", "change type")
	docType := fs.String("type", "", "type")
	id := fs.String("id", "", "id")
	targetID := fs.String("target-id", "", "target id")
	titleFile := fs.String("title-file", "", "title file")
	titleHash := fs.String("title-hash", "", "title hash")
	bodyFile := fs.String("body-file", "", "body file")
	bodyHash := fs.String("body-hash", "", "body hash")
	retconFile := fs.String("retcon-reason-file", "", "retcon reason file")
	retconHash := fs.String("retcon-reason-hash", "", "retcon reason hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *changeType == "" || *titleFile == "" || *bodyFile == "" {
		return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", "--change-type, --title-file, and --body-file are required", nil))
	}
	titleBytes, err := readStagedFile(ctx, *titleFile, *titleHash)
	if err != nil {
		return emit(fileReadError("draft.create", ctx, err))
	}
	bodyBytes, err := readStagedFile(ctx, *bodyFile, *bodyHash)
	if err != nil {
		return emit(fileReadError("draft.create", ctx, err))
	}
	title := strings.TrimSpace(string(titleBytes))
	body := strings.TrimSpace(string(bodyBytes))
	var retcon string
	switch *changeType {
	case "create":
		if *docType == "" || *id == "" || *targetID != "" {
			return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", "create requires --type and --id and forbids --target-id", nil))
		}
		if idExists(ctx, *id, true) {
			data := map[string]any{"id": *id, "validation_status": "conflict"}
			return emit(blockedEnvelope("draft.create", ctx, nil, "ID_CONFLICT", data, []Issue{{Code: "ID_CONFLICT", Rule: "VR-101", Severity: "conflict", Message: "id already exists", Field: "id"}}, nil))
		}
	case "update", "deprecate":
		if *targetID == "" || *id != "" || *docType != "" || *retconFile == "" {
			return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", "update/deprecate requires --target-id and --retcon-reason-file and forbids --id/--type", nil))
		}
		target, ok := findContentByID(ctx, *targetID)
		if !ok {
			data := map[string]any{"target_id": *targetID, "validation_status": "conflict"}
			return emit(blockedEnvelope("draft.create", ctx, nil, "MISSING_TARGET", data, []Issue{{Code: "MISSING_TARGET", Rule: "VR-002", Severity: "conflict", Message: "target canon document is missing", Field: "target_id"}}, nil))
		}
		*id = *targetID
		*docType = target.Type()
		retconBytes, err := readStagedFile(ctx, *retconFile, *retconHash)
		if err != nil {
			return emit(fileReadError("draft.create", ctx, err))
		}
		retcon = strings.TrimSpace(string(retconBytes))
	default:
		return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", "unsupported change type", nil))
	}
	path, err := draftPath(*docType, *id)
	if err != nil {
		return emit(failEnvelope("draft.create", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	rid, runDir, err := createRun(ctx, "draft.create")
	if err != nil {
		return emit(failEnvelope("draft.create", ctx, "IO_ERROR", err.Error(), nil))
	}
	meta := baseDraftMeta(*id, *docType, title, rid, *changeType, *targetID, retcon)
	if !strings.HasPrefix(body, "# ") {
		body = "# " + title + "\n\n" + body
	}
	out, err := buildMarkdown(meta, body)
	if err != nil {
		return emit(failEnvelope("draft.create", ctx, "INTERNAL_ERROR", err.Error(), nil))
	}
	abs, _, err := safeRel(ctx.Root, path)
	if err != nil {
		return emit(failEnvelope("draft.create", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	if err := writeFileAtomic(abs, out, 0o644); err != nil {
		return emit(failEnvelope("draft.create", ctx, "IO_ERROR", err.Error(), nil))
	}
	dhash := sha256Bytes(out)
	_ = writeJSON(filepath.Join(runDir, "summary.json"), map[string]any{"draft_path": path, "draft_hash": dhash})
	return emit(okEnvelope("draft.create", ctx, rid, map[string]any{"id": *id, "draft_path": path, "draft_hash": dhash}, nil, []string{"world_read_draft", "world_validate_draft", "world_update_draft", "world_reject_draft"}))
}

func cmdDraftUpdate(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("draft.update.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	draft := fs.String("draft", "", "draft")
	bodyFile := fs.String("body-file", "", "body file")
	bodyHash := fs.String("body-hash", "", "body hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("draft.update", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *draft == "" || *bodyFile == "" {
		return emit(failEnvelope("draft.update", ctx, "INVALID_ARGUMENT", "--draft and --body-file are required", nil))
	}
	doc, err := readDocument(ctx, *draft)
	if err != nil {
		return emit(failEnvelope("draft.update", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	b, err := readStagedFile(ctx, *bodyFile, *bodyHash)
	if err != nil {
		return emit(fileReadError("draft.update", ctx, err))
	}
	rid, _, err := createRun(ctx, "draft.update")
	if err != nil {
		return emit(failEnvelope("draft.update", ctx, "IO_ERROR", err.Error(), nil))
	}
	doc.Meta["updated_at"] = nowDate()
	doc.Meta["source_run_id"] = rid
	out, err := buildMarkdown(doc.Meta, string(b))
	if err != nil {
		return emit(failEnvelope("draft.update", ctx, "INTERNAL_ERROR", err.Error(), nil))
	}
	abs, _, _ := safeRel(ctx.Root, *draft)
	if err := writeFileAtomic(abs, out, 0o644); err != nil {
		return emit(failEnvelope("draft.update", ctx, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("draft.update", ctx, rid, map[string]any{"draft_path": doc.Path, "draft_hash": sha256Bytes(out)}, nil, []string{"world_read_draft", "world_validate_draft", "world_reject_draft"}))
}
