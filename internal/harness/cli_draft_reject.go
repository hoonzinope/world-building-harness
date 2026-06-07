package harness

import (
	"flag"
	"io"
	"os"
	"path/filepath"
)

func cmdDraftReject(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("draft.reject.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	draft := fs.String("draft", "", "draft")
	reasonFile := fs.String("reason-file", "", "reason")
	reasonHash := fs.String("reason-hash", "", "reason hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("draft.reject", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *draft == "" || *reasonFile == "" {
		return emit(failEnvelope("draft.reject", ctx, "INVALID_ARGUMENT", "--draft and --reason-file are required", nil))
	}
	reasonBytes, err := readStagedFile(ctx, *reasonFile, *reasonHash)
	if err != nil {
		return emit(fileReadError("draft.reject", ctx, err))
	}
	rid, runDir, err := createRun(ctx, "draft.reject")
	if err != nil {
		return emit(failEnvelope("draft.reject", ctx, "IO_ERROR", err.Error(), nil))
	}
	doc, err := readDocument(ctx, *draft)
	if err != nil {
		return emit(failEnvelope("draft.reject", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	archiveRel := filepath.ToSlash(filepath.Join("archive", "rejected", rid+"-"+filepath.Base(doc.Path)))
	archiveAbs, _, _ := safeRel(ctx.Root, archiveRel)
	draftAbs, _, _ := safeRel(ctx.Root, doc.Path)
	_ = ensureDir(filepath.Dir(archiveAbs))
	if err := os.Rename(draftAbs, archiveAbs); err != nil {
		return emit(failEnvelope("draft.reject", ctx, "IO_ERROR", err.Error(), nil))
	}
	data := map[string]any{"draft_path": doc.Path, "archive_path": archiveRel, "reason_hash": *reasonHash, "redacted_reason_size_bytes": len(reasonBytes)}
	_ = writeJSON(filepath.Join(runDir, "result.json"), data)
	return emit(okEnvelope("draft.reject", ctx, rid, data, nil, nil))
}
