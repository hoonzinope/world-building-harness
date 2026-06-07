package cli

import (
	"flag"
	"io"
	"path/filepath"
)

func cmdDraftList(_ commonFlags, ctx *WorldContext, _ []string) int {
	docs, _ := listDocuments(ctx, "drafts")
	out := []map[string]any{}
	for _, doc := range docs {
		out = append(out, documentSummary(doc))
	}
	return emit(okEnvelope("draft.list", ctx, nil, map[string]any{"drafts": out}, nil, nil))
}

func draftFlag(args []string) string {
	fs := flag.NewFlagSet("draft-flag", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	draft := fs.String("draft", "", "draft")
	_ = fs.Parse(args)
	return *draft
}

func cmdDraftRead(_ commonFlags, ctx *WorldContext, args []string) int {
	draft := draftFlag(args)
	if draft == "" {
		return emit(failEnvelope("draft.read", ctx, "INVALID_ARGUMENT", "--draft is required", nil))
	}
	doc, err := readDocument(ctx, draft)
	if err != nil {
		return emit(failEnvelope("draft.read", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	return emit(okEnvelope("draft.read", ctx, nil, map[string]any{"draft_path": doc.Path, "draft_hash": sha256Bytes([]byte(doc.Raw)), "frontmatter": doc.Meta, "body": doc.Body}, nil, nil))
}

func cmdDraftValidate(_ commonFlags, ctx *WorldContext, args []string) int {
	draft := draftFlag(args)
	if draft == "" {
		return emit(failEnvelope("draft.validate", ctx, "INVALID_ARGUMENT", "--draft is required", nil))
	}
	doc, err := readDocument(ctx, draft)
	if err != nil {
		return emit(failEnvelope("draft.validate", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	rid, runDir, err := createRun(ctx, "draft.validate")
	if err != nil {
		return emit(failEnvelope("draft.validate", ctx, "IO_ERROR", err.Error(), nil))
	}
	status, issues := validateDocument(ctx, doc, false)
	data := map[string]any{"draft_path": doc.Path, "draft_hash": sha256Bytes([]byte(doc.Raw)), "validation_status": status}
	_ = writeJSON(filepath.Join(runDir, "validation.json"), map[string]any{"status": status, "issues": issues, "draft_path": doc.Path})
	return emit(okEnvelope("draft.validate", ctx, rid, data, issues, validationActions(status)))
}
