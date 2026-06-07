package harness

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func cmdInputStage(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("input.stage.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	kind := fs.String("kind", "", "kind")
	stdin := fs.Bool("stdin", false, "read stdin")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("input.stage", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *kind == "" || !*stdin {
		return emit(failEnvelope("input.stage", ctx, "INVALID_ARGUMENT", "--kind and --stdin are required", nil))
	}
	allowed := map[string]bool{"query": true, "title": true, "body": true, "reason": true, "retcon_reason": true, "note": true}
	if !allowed[*kind] {
		return emit(failEnvelope("input.stage", ctx, "INVALID_ARGUMENT", "unsupported input kind", nil))
	}
	b, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20+1))
	if err != nil {
		return emit(failEnvelope("input.stage", ctx, "IO_ERROR", err.Error(), nil))
	}
	if len(b) > 1<<20 {
		return emit(failEnvelope("input.stage", ctx, "INVALID_ARGUMENT", "input is larger than 1 MiB", nil))
	}
	rid, _, err := createRun(ctx, "input.stage")
	if err != nil {
		return emit(failEnvelope("input.stage", ctx, "IO_ERROR", err.Error(), nil))
	}
	ext := ".txt"
	if *kind == "body" || *kind == "note" {
		ext = ".md"
	}
	rel := filepath.ToSlash(filepath.Join("runs", "inbox", rid+"-"+*kind+ext))
	abs, _, err := safeRel(ctx.Root, rel)
	if err != nil {
		return emit(failEnvelope("input.stage", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	if err := writeFileAtomic(abs, b, 0o600); err != nil {
		return emit(failEnvelope("input.stage", ctx, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("input.stage", ctx, rid, map[string]any{"kind": *kind, "input_path": rel, "input_hash": sha256Bytes(b)}, nil, nil))
}

func cmdDocList(_ commonFlags, ctx *WorldContext, args []string) int {
	scope := localScopeFlag(args, "active")
	docs, err := listDocuments(ctx, scope)
	if err != nil {
		return emit(failEnvelope("doc.list", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	out := []map[string]any{}
	for _, doc := range docs {
		out = append(out, documentSummary(doc))
	}
	return emit(okEnvelope("doc.list", ctx, nil, map[string]any{"documents": out}, nil, nil))
}

func cmdDocRead(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("doc.read.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "path")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("doc.read", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *path == "" {
		return emit(failEnvelope("doc.read", ctx, "INVALID_ARGUMENT", "--path is required", nil))
	}
	if !strings.HasPrefix(*path, "content/") && !strings.HasPrefix(*path, "drafts/") {
		return emit(failEnvelope("doc.read", ctx, "PATH_SCOPE_DENIED", "doc read only allows content/ or drafts/ markdown", nil))
	}
	doc, err := readDocument(ctx, *path)
	if err != nil {
		return emit(failEnvelope("doc.read", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	return emit(okEnvelope("doc.read", ctx, nil, map[string]any{"path": doc.Path, "document": documentSummary(doc), "body": doc.Body, "frontmatter": doc.Meta}, nil, nil))
}

func cmdDocSearch(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("doc.search.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scope := fs.String("scope", "active", "scope")
	query := fs.String("query", "", "query")
	queryFile := fs.String("query-file", "", "query file")
	queryHash := fs.String("query-hash", "", "query hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("doc.search", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	q := *query
	if *queryFile != "" {
		b, err := readStagedFile(ctx, *queryFile, *queryHash)
		if err != nil {
			return emit(fileReadError("doc.search", ctx, err))
		}
		q = string(b)
	}
	results, err := searchDocuments(ctx, *scope, q)
	if err != nil {
		return emit(failEnvelope("doc.search", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	return emit(okEnvelope("doc.search", ctx, nil, map[string]any{"query_hash": sha256Bytes([]byte(q)), "results": results}, nil, nil))
}

func cmdContentValidate(_ commonFlags, ctx *WorldContext, _ []string) int {
	rid, runDir, err := createRun(ctx, "content.validate")
	if err != nil {
		return emit(failEnvelope("content.validate", ctx, "IO_ERROR", err.Error(), nil))
	}
	docs, _ := listDocuments(ctx, "content")
	allIssues := []Issue{}
	for _, doc := range docs {
		_, issues := validateDocument(ctx, doc, false)
		allIssues = append(allIssues, issues...)
	}
	status := validationStatus(allIssues)
	data := map[string]any{"validation_status": status, "document_count": len(docs), "blockers": issueCount(allIssues, "error") + issueCount(allIssues, "conflict"), "findings": len(allIssues)}
	_ = writeJSON(filepath.Join(runDir, "validation.json"), map[string]any{"status": status, "issues": allIssues})
	return emit(okEnvelope("content.validate", ctx, rid, data, allIssues, nil))
}
