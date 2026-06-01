package harness

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Run(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "usage: world-tool <resource> <action> [flags]")
		return 0
	}
	if args[0] == "serve" {
		return runServe(args[1:])
	}
	if args[0] == "telegram" {
		return runTelegram(args[1:])
	}
	if len(args) < 2 {
		return emit(failEnvelope("unknown", nil, "INVALID_ARGUMENT", "resource and action are required", nil))
	}
	resource, action := args[0], args[1]
	rest := args[2:]
	switch resource + "." + action {
	case "world.init":
		return cmdWorldInit(rest)
	case "world.status":
		return withWorld("world.status", rest, cmdWorldStatus)
	case "world.list":
		return cmdWorldList(rest)
	case "registry.list":
		return cmdRegistryList(rest)
	case "registry.add":
		return cmdRegistryAdd(rest)
	case "registry.remove":
		return cmdRegistryRemove(rest)
	case "registry.default":
		return cmdRegistryDefault(rest)
	case "input.stage":
		return withWorld("input.stage", rest, cmdInputStage)
	case "doc.list":
		return withWorld("doc.list", rest, cmdDocList)
	case "doc.read":
		return withWorld("doc.read", rest, cmdDocRead)
	case "doc.search":
		return withWorld("doc.search", rest, cmdDocSearch)
	case "draft.create":
		return withWorld("draft.create", rest, cmdDraftCreate)
	case "draft.update":
		return withWorld("draft.update", rest, cmdDraftUpdate)
	case "draft.list":
		return withWorld("draft.list", rest, cmdDraftList)
	case "draft.read":
		return withWorld("draft.read", rest, cmdDraftRead)
	case "draft.validate":
		return withWorld("draft.validate", rest, cmdDraftValidate)
	case "draft.diff":
		return withWorld("draft.diff", rest, cmdDraftDiff)
	case "draft.accept":
		return withWorld("draft.accept", rest, cmdDraftAccept)
	case "draft.reject":
		return withWorld("draft.reject", rest, cmdDraftReject)
	case "content.validate":
		return withWorld("content.validate", rest, cmdContentValidate)
	case "run.list":
		return withWorld("run.list", rest, cmdRunList)
	case "run.get":
		return withWorld("run.get", rest, cmdRunGet)
	case "run.recover":
		return withWorld("run.recover", rest, cmdRunRecover)
	case "content.migrate":
		return withWorld("content.migrate", rest, cmdContentMigrate)
	case "approval.attest":
		return withWorld("approval.attest", rest, cmdApprovalAttest)
	default:
		return emit(failEnvelope(resource+"."+action, nil, "INVALID_ARGUMENT", "unknown command", nil))
	}
}

type worldCommand func(commonFlags, *WorldContext, []string) int

var writeCommands = map[string]bool{
	"input.stage":      true,
	"draft.create":     true,
	"draft.update":     true,
	"draft.validate":   true,
	"draft.diff":       true,
	"draft.accept":     true,
	"draft.reject":     true,
	"content.validate": true,
	"content.migrate":  true,
	"approval.attest":  true,
	"run.recover":      true,
}

func withWorld(command string, args []string, fn worldCommand) int {
	c, remaining, err := parseCommon(command, args)
	if err != nil {
		return emit(failEnvelope(command, nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	ctx, terr := resolveWorld(c, command)
	if terr != nil {
		return emit(failEnvelope(command, nil, terr.Code, terr.Message, terr.Details))
	}
	if writeCommands[command] {
		if command != "run.recover" {
			if rid, ok := unresolvedRecovery(ctx); ok {
				data := map[string]any{"block_reason": "TRANSACTION_INCOMPLETE", "recovery_run_id": rid}
				return emit(blockedEnvelope(command, ctx, nil, "TRANSACTION_INCOMPLETE", data, []Issue{{Code: "TRANSACTION_INCOMPLETE", Severity: "conflict", Message: "unresolved recovery blocks write commands"}}, []string{"world_list_runs", "world_get_run", "world_get_run_artifact", "world_recover_run"}))
			}
		}
		unlock, err := acquireWorldLock(ctx, command)
		if err != nil {
			code := "IO_ERROR"
			if strings.HasPrefix(err.Error(), "LOCK_BUSY:") {
				code = "LOCK_BUSY"
			}
			return emit(failEnvelope(command, ctx, code, err.Error(), nil))
		}
		defer unlock()
	}
	return fn(c, ctx, remaining)
}

func parseCommon(name string, args []string) (commonFlags, []string, error) {
	c := commonFlags{}
	remaining := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, value, hasValue := strings.Cut(arg, "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			i++
			return args[i], nil
		}
		switch key {
		case "--world":
			v, err := takeValue()
			if err != nil {
				return c, nil, err
			}
			c.world = v
		case "--root":
			v, err := takeValue()
			if err != nil {
				return c, nil, err
			}
			c.root = v
		case "--world-id":
			v, err := takeValue()
			if err != nil {
				return c, nil, err
			}
			c.worldID = v
		case "--registry":
			v, err := takeValue()
			if err != nil {
				return c, nil, err
			}
			c.registry = v
		case "--run-id":
			v, err := takeValue()
			if err != nil {
				return c, nil, err
			}
			c.runID = v
		case "--json":
			c.json = true
		case "--dry-run":
			c.dryRun = true
		case "--verbose":
			c.verbose = true
		default:
			remaining = append(remaining, arg)
		}
	}
	if c.dryRun && name != "content.migrate" {
		return c, nil, fmt.Errorf("--dry-run is only supported by content migrate")
	}
	return c, remaining, nil
}

func addCommon(fs *flag.FlagSet, c *commonFlags) {
	fs.StringVar(&c.world, "world", "", "world id")
	fs.StringVar(&c.root, "root", "", "world root")
	fs.StringVar(&c.worldID, "world-id", "", "root mode world id")
	fs.StringVar(&c.registry, "registry", "", "registry file")
	fs.BoolVar(&c.json, "json", false, "json output")
	fs.StringVar(&c.runID, "run-id", "", "run id")
	fs.BoolVar(&c.dryRun, "dry-run", false, "dry run")
	fs.BoolVar(&c.verbose, "verbose", false, "verbose")
}

func cmdWorldInit(args []string) int {
	c, remaining, err := parseCommon("world.init", args)
	if err != nil {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if len(remaining) != 0 {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", "unexpected arguments", remaining))
	}
	if c.root == "" || c.worldID == "" {
		return emit(failEnvelope("world.init", nil, "INVALID_ARGUMENT", "--root and --world-id are required", nil))
	}
	ctx, err := initWorld(c.root, c.worldID)
	if err != nil {
		return emit(failEnvelope("world.init", nil, "IO_ERROR", err.Error(), nil))
	}
	return emit(okEnvelope("world.init", ctx, nil, map[string]any{"world_id": ctx.ID, "root": ctx.Root}, nil, nil))
}

func cmdWorldStatus(_ commonFlags, ctx *WorldContext, _ []string) int {
	return emit(okEnvelope("world.status", ctx, nil, worldStatus(ctx), nil, nil))
}

func cmdWorldList(args []string) int {
	c, remaining, err := parseCommon("world.list", args)
	if err != nil {
		return emit(failEnvelope("world.list", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world != "" || c.root != "" || len(remaining) != 0 {
		return emit(failEnvelope("world.list", nil, "INVALID_ARGUMENT", "world list does not accept --world, --root, or positional arguments", nil))
	}
	reg, regPath, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("world.list", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	return emit(okEnvelope("world.list", nil, nil, map[string]any{"registry_path": regPath, "worlds": registryWorldList(reg)}, nil, nil))
}

func cmdRegistryList(args []string) int {
	c, remaining, err := parseCommon("registry.list", args)
	if err != nil {
		return emit(failEnvelope("registry.list", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if len(remaining) != 0 {
		return emit(failEnvelope("registry.list", nil, "INVALID_ARGUMENT", "unexpected arguments", remaining))
	}
	reg, regPath, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.list", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	return emit(okEnvelope("registry.list", nil, nil, map[string]any{"registry_path": regPath, "default": reg.Default, "worlds": registryWorldList(reg)}, nil, nil))
}

func cmdRegistryAdd(args []string) int {
	c, remaining, err := parseCommon("registry.add", args)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	fs := flag.NewFlagSet("registry.add.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	title := fs.String("title", "", "title")
	root := fs.String("root", c.root, "root")
	if err := fs.Parse(remaining); err != nil {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || *root == "" || *title == "" {
		return emit(failEnvelope("registry.add", nil, "INVALID_ARGUMENT", "--world, --root, and --title are required", nil))
	}
	abs, err := normalizeRoot(*root, true)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "WORLD_NOT_FOUND", err.Error(), nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	reg.Worlds[c.world] = RegistryWorld{Title: *title, Root: abs}
	if reg.Default == "" {
		reg.Default = c.world
	}
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.add", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.add", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath, "registry_root": abs}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}

func cmdRegistryRemove(args []string) int {
	c, remaining, err := parseCommon("registry.remove", args)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || len(remaining) != 0 {
		return emit(failEnvelope("registry.remove", nil, "INVALID_ARGUMENT", "--world is required", nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	delete(reg.Worlds, c.world)
	if reg.Default == c.world {
		reg.Default = ""
	}
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.remove", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.remove", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}

func cmdRegistryDefault(args []string) int {
	c, remaining, err := parseCommon("registry.default", args)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if c.world == "" || len(remaining) != 0 {
		return emit(failEnvelope("registry.default", nil, "INVALID_ARGUMENT", "--world is required", nil))
	}
	reg, _, err := loadRegistry(c.registry)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "REGISTRY_NOT_FOUND", err.Error(), nil))
	}
	if _, ok := reg.Worlds[c.world]; !ok {
		return emit(failEnvelope("registry.default", nil, "WORLD_NOT_FOUND", "world is not registered", nil))
	}
	reg.Default = c.world
	regPath, err := saveRegistry(c.registry, reg)
	if err != nil {
		return emit(failEnvelope("registry.default", nil, "IO_ERROR", err.Error(), nil))
	}
	env := okEnvelope("registry.default", nil, nil, map[string]any{"world_id": c.world, "registry_path": regPath}, nil, nil)
	env.WorldID = c.world
	return emit(env)
}

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

func localScopeFlag(args []string, def string) string {
	fs := flag.NewFlagSet("scope", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scope := fs.String("scope", def, "scope")
	_ = fs.Parse(args)
	return *scope
}

func readStagedFile(ctx *WorldContext, rel, expectedHash string) ([]byte, error) {
	if !strings.HasPrefix(filepath.ToSlash(filepath.Clean(rel)), "runs/inbox/") {
		return nil, fmt.Errorf("PATH_SCOPE_DENIED: staged files must be under runs/inbox")
	}
	abs, _, err := safeRel(ctx.Root, rel)
	if err != nil {
		return nil, fmt.Errorf("PATH_OUTSIDE_ROOT: %w", err)
	}
	got, err := sha256File(abs)
	if err != nil {
		return nil, fmt.Errorf("IO_ERROR: %w", err)
	}
	if expectedHash == "" || got != expectedHash {
		return nil, fmt.Errorf("INPUT_HASH_MISMATCH: expected %s got %s", expectedHash, got)
	}
	return os.ReadFile(abs)
}

func fileReadError(command string, ctx *WorldContext, err error) Envelope {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "INPUT_HASH_MISMATCH"):
		return failEnvelope(command, ctx, "INPUT_HASH_MISMATCH", msg, nil)
	case strings.Contains(msg, "PATH_SCOPE_DENIED"):
		return failEnvelope(command, ctx, "PATH_SCOPE_DENIED", msg, nil)
	case strings.Contains(msg, "PATH_OUTSIDE_ROOT"):
		return failEnvelope(command, ctx, "PATH_OUTSIDE_ROOT", msg, nil)
	default:
		return failEnvelope(command, ctx, "IO_ERROR", msg, nil)
	}
}

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

func baseDraftMeta(id, docType, title, sourceRunID, changeType, targetID, retcon string) map[string]any {
	meta := map[string]any{
		"schema_version": docSchemaVersion,
		"id":             id,
		"type":           docType,
		"status":         "draft",
		"title":          title,
		"tags":           []string{},
		"created_at":     nowDate(),
		"updated_at":     nowDate(),
		"related":        []string{},
		"relationships":  []any{},
		"source_run_id":  sourceRunID,
		"change_type":    changeType,
	}
	if targetID != "" {
		meta["target_id"] = targetID
	} else {
		meta["target_id"] = nil
	}
	if retcon != "" {
		meta["retcon_reason"] = retcon
	} else {
		meta["retcon_reason"] = nil
	}
	return meta
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

func cmdDraftList(_ commonFlags, ctx *WorldContext, _ []string) int {
	docs, _ := listDocuments(ctx, "drafts")
	out := []map[string]any{}
	for _, doc := range docs {
		out = append(out, documentSummary(doc))
	}
	return emit(okEnvelope("draft.list", ctx, nil, map[string]any{"drafts": out}, nil, nil))
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

func draftFlag(args []string) string {
	fs := flag.NewFlagSet("draft-flag", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	draft := fs.String("draft", "", "draft")
	_ = fs.Parse(args)
	return *draft
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

func nullableHash(hash string) any {
	if hash == "" || hash == "none" {
		return nil
	}
	return hash
}

func targetForDraft(ctx *WorldContext, doc Document) (string, bool, string, string) {
	changeType := metaString(doc.Meta, "change_type")
	switch changeType {
	case "create":
		if doc.Type() == "storylet" {
			return "", false, "", "STORYLET_NOT_CANON_TARGET"
		}
		target, err := contentPath(doc.Type(), doc.ID())
		if err != nil {
			return "", false, "", "TARGET_PATH_CONFLICT"
		}
		abs, _, _ := safeRel(ctx.Root, target)
		if _, err := os.Stat(abs); err == nil {
			return target, true, "", "TARGET_PATH_CONFLICT"
		}
		return target, false, "", ""
	case "update", "deprecate":
		target, ok := findContentByID(ctx, metaString(doc.Meta, "target_id"))
		if !ok {
			return "", false, "", "MISSING_TARGET"
		}
		return target.Path, true, sha256Bytes([]byte(target.Raw)), ""
	default:
		return "", false, "", "INVALID_ARGUMENT"
	}
}

func patchText(targetPath, targetHash, draftHash string) string {
	return fmt.Sprintf("--- %s %s\n+++ draft %s\n", targetPath, nullableHash(targetHash), draftHash)
}

func hasIssueCode(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func cmdApprovalAttest(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("approval.attest.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	diffRunID := fs.String("diff-run-id", "", "diff run id")
	draftHash := fs.String("draft-hash", "", "draft hash")
	targetBaseHash := fs.String("target-base-hash", "", "target base hash")
	patchHash := fs.String("patch-hash", "", "patch hash")
	approverID := fs.String("approver-id", "", "approver id")
	channel := fs.String("approval-channel", "", "channel")
	downstream := fs.String("downstream-action", "", "downstream")
	actor := fs.String("authenticated-actor", "", "actor")
	authFile := fs.String("auth-context-file", "", "auth context file")
	authHash := fs.String("auth-context-hash", "", "auth context hash")
	reasonHash := fs.String("reason-hash", "", "reason hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("approval.attest", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	required := []*string{diffRunID, draftHash, targetBaseHash, patchHash, approverID, channel, downstream, actor, authFile, authHash, reasonHash}
	for _, ptr := range required {
		if *ptr == "" {
			return emit(failEnvelope("approval.attest", ctx, "INVALID_ARGUMENT", "all approval attest flags are required", nil))
		}
	}
	auth, errEnv := verifyAuthContext(ctx, *authFile, *authHash, *downstream, *actor, *channel)
	if errEnv != nil {
		return emit(*errEnv)
	}
	rid, _, err := createRun(ctx, "approval.attest")
	if err != nil {
		return emit(failEnvelope("approval.attest", ctx, "IO_ERROR", err.Error(), nil))
	}
	targetHash := *targetBaseHash
	var normalizedTarget any = targetHash
	if targetHash == "none" {
		normalizedTarget = nil
	}
	payload := map[string]any{
		"world_id":            ctx.ID,
		"authenticated_actor": *actor,
		"approver_id":         *approverID,
		"approval_channel":    *channel,
		"issuer":              auth["issuer"],
		"audience":            auth["audience"],
		"scope_verification":  map[string]any{"allowed_actions": auth["allowed_actions"]},
		"downstream_action":   *downstream,
		"diff_run_id":         *diffRunID,
		"draft_hash":          *draftHash,
		"target_base_hash":    normalizedTarget,
		"patch_hash":          *patchHash,
		"reason_hash":         *reasonHash,
		"issued_at":           auth["issued_at"],
		"expires_at":          auth["expires_at"],
		"session_id":          auth["session_id"],
		"created_at":          time.Now().UTC().Format(time.RFC3339),
	}
	rel := filepath.ToSlash(filepath.Join("runs", "inbox", rid+"-approval-attestation.json"))
	abs, _, _ := safeRel(ctx.Root, rel)
	b, _ := json.MarshalIndent(payload, "", "  ")
	if err := writeFileAtomic(abs, append(b, '\n'), 0o600); err != nil {
		return emit(failEnvelope("approval.attest", ctx, "IO_ERROR", err.Error(), nil))
	}
	hash := sha256Bytes(append(b, '\n'))
	data := map[string]any{
		"approval_attestation_file": rel,
		"approval_attestation_hash": hash,
		"world_id":                  ctx.ID,
		"issuer":                    auth["issuer"],
		"audience":                  auth["audience"],
		"scope_verification":        payload["scope_verification"],
		"downstream_action":         *downstream,
		"reason_hash":               *reasonHash,
		"authenticated_actor":       *actor,
		"approver_id":               *approverID,
		"approval_channel":          *channel,
		"issued_at":                 auth["issued_at"],
		"expires_at":                auth["expires_at"],
		"diff_run_id":               *diffRunID,
		"draft_hash":                *draftHash,
		"target_base_hash":          normalizedTarget,
		"patch_hash":                *patchHash,
	}
	action := "world_accept_draft"
	if *downstream == "world_force_accept_draft" {
		action = "world_force_accept_draft"
	}
	return emit(okEnvelope("approval.attest", ctx, rid, data, nil, []string{action}))
}

func verifyAuthContext(ctx *WorldContext, path, expectedHash, downstream, actor, channel string) (map[string]any, *Envelope) {
	got, err := sha256File(path)
	if err != nil {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_MISSING", err.Error(), nil)
		return nil, &env
	}
	if got != expectedHash {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_HASH_MISMATCH", "auth context hash mismatch", map[string]any{"expected": expectedHash, "got": got})
		return nil, &env
	}
	b, err := os.ReadFile(path)
	if err != nil {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_MISSING", err.Error(), nil)
		return nil, &env
	}
	var auth map[string]any
	if err := json.Unmarshal(b, &auth); err != nil {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_MISSING", err.Error(), nil)
		return nil, &env
	}
	if auth["fixture_mode"] == true && os.Getenv("WORLD_TOOL_TEST_AUTH_CONTEXT") != "1" {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_TEST_MODE_REQUIRED", "fixture auth context requires WORLD_TOOL_TEST_AUTH_CONTEXT=1", nil)
		return nil, &env
	}
	if fmt.Sprint(auth["world_id"]) != ctx.ID || fmt.Sprint(auth["authenticated_actor"]) != actor || fmt.Sprint(auth["approval_channel"]) != channel || fmt.Sprint(auth["downstream_action"]) != downstream {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_SCOPE_DENIED", "auth context binding mismatch", nil)
		return nil, &env
	}
	if !allowedAction(auth["allowed_actions"], "world_create_approval_attestation") || !allowedAction(auth["allowed_actions"], downstream) {
		env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_SCOPE_DENIED", "auth context does not allow requested action", nil)
		return nil, &env
	}
	if exp := fmt.Sprint(auth["expires_at"]); exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil && time.Now().After(t) {
			env := failEnvelope("approval.attest", ctx, "AUTH_CONTEXT_EXPIRED", "auth context expired", nil)
			return nil, &env
		}
	}
	return auth, nil
}

func allowedAction(v any, action string) bool {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			if fmt.Sprint(item) == action {
				return true
			}
		}
	case []string:
		for _, item := range t {
			if item == action {
				return true
			}
		}
	}
	return false
}

func cmdDraftAccept(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("draft.accept.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	draft := fs.String("draft", "", "draft")
	diffRunID := fs.String("diff-run-id", "", "diff run")
	draftHash := fs.String("draft-hash", "", "draft hash")
	targetBaseHash := fs.String("target-base-hash", "", "target base")
	patchHash := fs.String("patch-hash", "", "patch")
	force := fs.Bool("force", false, "force")
	approverID := fs.String("approver-id", "", "approver")
	channel := fs.String("approval-channel", "", "channel")
	attFile := fs.String("approval-attestation-file", "", "attestation")
	attHash := fs.String("approval-attestation-hash", "", "attestation hash")
	actor := fs.String("authenticated-actor", "", "actor")
	reasonFile := fs.String("reason-file", "", "reason")
	reasonHash := fs.String("reason-hash", "", "reason hash")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("draft.accept", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *draft == "" || *diffRunID == "" || *draftHash == "" || *targetBaseHash == "" || *patchHash == "" || *approverID == "" || *channel == "" || *attFile == "" || *attHash == "" || *actor == "" || *reasonFile == "" || *reasonHash == "" {
		return emit(failEnvelope("draft.accept", ctx, "INVALID_ARGUMENT", "accept requires draft, diff binding, reason, actor, and approval attestation flags", nil))
	}
	reasonBytes, err := readStagedFile(ctx, *reasonFile, *reasonHash)
	if err != nil {
		return emit(fileReadError("draft.accept", ctx, err))
	}
	attPayload, errEnv := verifyAttestation(ctx, *attFile, *attHash, downstreamAction(*force), *actor, *channel, *approverID, *diffRunID, *draftHash, *targetBaseHash, *patchHash, *reasonHash)
	if errEnv != nil {
		return emit(*errEnv)
	}
	doc, err := readDocument(ctx, *draft)
	if err != nil {
		return emit(failEnvelope("draft.accept", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil))
	}
	targetPath, targetExists, targetHash, blocked := targetForDraft(ctx, doc)
	if blocked != "" {
		data := map[string]any{"draft_path": doc.Path, "validation_status": "conflict"}
		return emit(blockedEnvelope("draft.accept", ctx, nil, blocked, data, []Issue{{Code: blocked, Severity: "conflict", Message: "target is not available", Path: doc.Path}}, validationActions("conflict")))
	}
	currentDraftHash := sha256Bytes([]byte(doc.Raw))
	normalizedTarget := *targetBaseHash
	if normalizedTarget == "none" {
		normalizedTarget = ""
	}
	if currentDraftHash != *draftHash || targetHash != normalizedTarget || sha256Bytes([]byte(patchText(targetPath, targetHash, currentDraftHash))) != *patchHash {
		data := map[string]any{"draft_path": doc.Path, "validation_status": "conflict", "block_reason": "DIFF_BINDING_MISMATCH"}
		return emit(blockedEnvelope("draft.accept", ctx, nil, "DIFF_BINDING_MISMATCH", data, []Issue{{Code: "DIFF_BINDING_MISMATCH", Severity: "conflict", Message: "diff binding does not match current files", Path: doc.Path}}, []string{"world_diff_draft", "world_validate_draft", "world_update_draft"}))
	}
	status, issues := validateDocument(ctx, doc, true)
	if hasIssueCode(issues, "MISSING_TARGET") {
		data := map[string]any{"draft_path": doc.Path, "validation_status": "conflict"}
		return emit(blockedEnvelope("draft.accept", ctx, nil, "MISSING_TARGET", data, issues, validationActions("conflict")))
	}
	if (status == "error" || status == "conflict") && !*force {
		data := map[string]any{"draft_path": doc.Path, "validation_status": status}
		return emit(blockedEnvelope("draft.accept", ctx, nil, "VALIDATION_BLOCKED", data, issues, validationActions(status)))
	}
	rid, runDir, err := createRun(ctx, "draft.accept")
	if err != nil {
		return emit(failEnvelope("draft.accept", ctx, "IO_ERROR", err.Error(), nil))
	}
	_ = writeJSON(filepath.Join(runDir, "result.json"), map[string]any{"status": "pending", "draft_path": doc.Path})
	contentBytes, err := acceptedContent(ctx, doc, targetPath, targetExists, rid)
	if err != nil {
		return emit(failEnvelope("draft.accept", ctx, "INTERNAL_ERROR", err.Error(), nil))
	}
	targetAbs, _, _ := safeRel(ctx.Root, targetPath)
	if err := writeFileAtomic(targetAbs, contentBytes, 0o644); err != nil {
		return emit(failEnvelope("draft.accept", ctx, "IO_ERROR", err.Error(), nil))
	}
	archiveRel := filepath.ToSlash(filepath.Join("archive", "accepted", rid+"-"+filepath.Base(doc.Path)))
	archiveAbs, _, _ := safeRel(ctx.Root, archiveRel)
	draftAbs, _, _ := safeRel(ctx.Root, doc.Path)
	if err := ensureDir(filepath.Dir(archiveAbs)); err != nil {
		return emit(failEnvelope("draft.accept", ctx, "IO_ERROR", err.Error(), nil))
	}
	if err := os.Rename(draftAbs, archiveAbs); err != nil {
		recovery := createAcceptRecovery(ctx, rid, doc.Path, targetPath, archiveRel, sha256Bytes(contentBytes), err.Error())
		env := failEnvelope("draft.accept", ctx, "TRANSACTION_INCOMPLETE", err.Error(), map[string]any{"recovery": recovery})
		env.RunID = rid
		env.AvailableActions = []string{"world_list_runs", "world_get_run", "world_get_run_artifact", "world_recover_run"}
		return emit(env)
	}
	approval := map[string]any{
		"approver_id":                *approverID,
		"approval_channel":           *channel,
		"authenticated_actor":        *actor,
		"issuer":                     attPayload["issuer"],
		"audience":                   attPayload["audience"],
		"scope_verification":         attPayload["scope_verification"],
		"issued_at":                  attPayload["issued_at"],
		"expires_at":                 attPayload["expires_at"],
		"attestation_validated_at":   time.Now().UTC().Format(time.RFC3339),
		"approval_attestation_file":  *attFile,
		"approval_attestation_hash":  *attHash,
		"reason_file":                *reasonFile,
		"reason_hash":                *reasonHash,
		"downstream_action":          downstreamAction(*force),
		"redacted_reason_size_bytes": len(reasonBytes),
	}
	data := map[string]any{
		"draft_path":        doc.Path,
		"archive_path":      archiveRel,
		"content_path":      targetPath,
		"content_hash":      sha256Bytes(contentBytes),
		"validation_status": status,
		"approval":          approval,
	}
	_ = writeJSON(filepath.Join(runDir, "result.json"), data)
	return emit(okEnvelope("draft.accept", ctx, rid, data, issues, nil))
}

func createAcceptRecovery(ctx *WorldContext, rid, draftPath, contentPathValue, archivePath, contentHash, cause string) map[string]any {
	recoveryRel := filepath.ToSlash(filepath.Join("runs", rid, "recovery.json"))
	recovery := map[string]any{
		"recovery_run_id": rid,
		"recovery_path":   recoveryRel,
		"recovery_status": "unresolved",
		"resolved":        false,
		"command":         "draft.accept",
		"cause":           cause,
		"completed_steps": []string{"content_written"},
		"remaining_steps": []string{"archive_draft", "write_completed_result"},
		"draft_path":      draftPath,
		"content_path":    contentPathValue,
		"archive_path":    archivePath,
		"content_hash":    contentHash,
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	}
	_ = writeJSON(filepath.Join(ctx.Root, recoveryRel), recovery)
	return recovery
}

func downstreamAction(force bool) string {
	if force {
		return "world_force_accept_draft"
	}
	return "world_accept_draft"
}

func verifyAttestation(ctx *WorldContext, rel, expectedHash, downstream, actor, channel, approver, diffRunID, draftHash, targetBaseHash, patchHash, reasonHash string) (map[string]any, *Envelope) {
	if !strings.HasPrefix(filepath.ToSlash(filepath.Clean(rel)), "runs/inbox/") || !strings.HasSuffix(rel, "-approval-attestation.json") {
		env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_BINDING_MISMATCH", "approval attestation must be in runs/inbox and generated by approval attest", nil)
		return nil, &env
	}
	abs, _, err := safeRel(ctx.Root, rel)
	if err != nil {
		env := failEnvelope("draft.accept", ctx, "PATH_OUTSIDE_ROOT", err.Error(), nil)
		return nil, &env
	}
	got, err := sha256File(abs)
	if err != nil {
		env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_HASH_MISMATCH", err.Error(), nil)
		return nil, &env
	}
	if got != expectedHash {
		env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_HASH_MISMATCH", "attestation hash mismatch", nil)
		return nil, &env
	}
	b, _ := os.ReadFile(abs)
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_BINDING_MISMATCH", err.Error(), nil)
		return nil, &env
	}
	normalizedTarget := fmt.Sprint(payload["target_base_hash"])
	if payload["target_base_hash"] == nil {
		normalizedTarget = "none"
	}
	checks := map[string]string{
		"world_id":            ctx.ID,
		"authenticated_actor": actor,
		"approver_id":         approver,
		"approval_channel":    channel,
		"downstream_action":   downstream,
		"diff_run_id":         diffRunID,
		"draft_hash":          draftHash,
		"target_base_hash":    targetBaseHash,
		"patch_hash":          patchHash,
		"reason_hash":         reasonHash,
	}
	for key, expected := range checks {
		got := fmt.Sprint(payload[key])
		if key == "target_base_hash" {
			got = normalizedTarget
		}
		if got != expected {
			env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_BINDING_MISMATCH", "attestation binding mismatch", map[string]any{"field": key})
			return nil, &env
		}
	}
	if exp := fmt.Sprint(payload["expires_at"]); exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil && time.Now().After(t) {
			env := failEnvelope("draft.accept", ctx, "APPROVAL_ATTESTATION_EXPIRED", "approval attestation expired", nil)
			return nil, &env
		}
	}
	return payload, nil
}

func acceptedContent(ctx *WorldContext, doc Document, targetPath string, targetExists bool, rid string) ([]byte, error) {
	meta := map[string]any{}
	for k, v := range doc.Meta {
		meta[k] = v
	}
	changeType := metaString(doc.Meta, "change_type")
	if changeType == "deprecate" && targetExists {
		target, err := readDocument(ctx, targetPath)
		if err != nil {
			return nil, err
		}
		for k, v := range target.Meta {
			meta[k] = v
		}
		meta["status"] = "deprecated"
		meta["updated_at"] = nowDate()
		meta["source_run_id"] = rid
		meta["deprecated_by_run_id"] = rid
		meta["deprecation_reason"] = metaString(doc.Meta, "retcon_reason")
		return buildMarkdown(meta, target.Body)
	}
	meta["status"] = "canon"
	meta["updated_at"] = nowDate()
	meta["source_run_id"] = rid
	delete(meta, "change_type")
	delete(meta, "target_id")
	delete(meta, "retcon_reason")
	return buildMarkdown(meta, doc.Body)
}

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

func cmdRunList(_ commonFlags, ctx *WorldContext, _ []string) int {
	runsRoot := filepath.Join(ctx.Root, "runs")
	entries, _ := os.ReadDir(runsRoot)
	runs := []map[string]any{}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "inbox" {
			continue
		}
		item := map[string]any{"run_id": e.Name()}
		if b, err := os.ReadFile(filepath.Join(runsRoot, e.Name(), "run.json")); err == nil {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				item["manifest"] = m
			}
		}
		runs = append(runs, item)
	}
	sort.Slice(runs, func(i, j int) bool { return fmt.Sprint(runs[i]["run_id"]) > fmt.Sprint(runs[j]["run_id"]) })
	return emit(okEnvelope("run.list", ctx, nil, map[string]any{"runs": runs}, nil, nil))
}

func cmdRunGet(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("run.get.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rid := fs.String("run-id", "", "run id")
	artifact := fs.String("artifact", "", "artifact")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("run.get", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *rid == "" {
		return emit(failEnvelope("run.get", ctx, "INVALID_ARGUMENT", "--run-id is required", nil))
	}
	runDir := filepath.Join(ctx.Root, "runs", filepath.Base(*rid))
	if *artifact != "" {
		allowed := map[string]bool{"run.json": true, "summary.json": true, "result.json": true, "validation.json": true, "diff.patch": true, "recovery.json": true}
		name := filepath.Base(*artifact)
		if !allowed[name] {
			return emit(failEnvelope("run.get", ctx, "PATH_SCOPE_DENIED", "artifact is not allowlisted", nil))
		}
		b, err := os.ReadFile(filepath.Join(runDir, name))
		if err != nil {
			return emit(failEnvelope("run.get", ctx, "IO_ERROR", err.Error(), nil))
		}
		return emit(okEnvelope("run.get", ctx, nil, map[string]any{"run_id": *rid, "artifact_name": name, "artifact_hash": sha256Bytes(b), "media_type": mediaType(name), "size_bytes": len(b), "redacted": true, "content": string(b)}, nil, nil))
	}
	manifest := map[string]any{"run_id": *rid}
	if b, err := os.ReadFile(filepath.Join(runDir, "run.json")); err == nil {
		_ = json.Unmarshal(b, &manifest)
	}
	return emit(okEnvelope("run.get", ctx, nil, map[string]any{"manifest": manifest, "status_summary": map[string]any{"run_id": *rid, "redacted": true}}, nil, nil))
}

func mediaType(name string) string {
	if strings.HasSuffix(name, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(name, ".patch") {
		return "text/x-patch"
	}
	return "text/plain"
}

func cmdRunRecover(_ commonFlags, ctx *WorldContext, args []string) int {
	fs := flag.NewFlagSet("run.recover.local", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rid := fs.String("run-id", "", "run id")
	if err := fs.Parse(args); err != nil {
		return emit(failEnvelope("run.recover", ctx, "INVALID_ARGUMENT", err.Error(), nil))
	}
	if *rid == "" {
		return emit(failEnvelope("run.recover", ctx, "INVALID_ARGUMENT", "--run-id is required", nil))
	}
	recoveryPath := filepath.Join(ctx.Root, "runs", filepath.Base(*rid), "recovery.json")
	recoveryRel := filepath.ToSlash(filepath.Join("runs", filepath.Base(*rid), "recovery.json"))
	b, err := os.ReadFile(recoveryPath)
	if os.IsNotExist(err) {
		data := map[string]any{"recovery_run_id": *rid, "recovery_path": recoveryRel, "recovery_status": "not_required", "recovered_at": time.Now().UTC().Format(time.RFC3339)}
		return emit(okEnvelope("run.recover", ctx, nil, data, nil, nil))
	}
	if err != nil {
		return emit(failEnvelope("run.recover", ctx, "IO_ERROR", err.Error(), nil))
	}
	var recovery map[string]any
	if err := json.Unmarshal(b, &recovery); err != nil {
		return emit(failEnvelope("run.recover", ctx, "IO_ERROR", err.Error(), nil))
	}
	if recovery["resolved"] == true || fmt.Sprint(recovery["recovery_status"]) == "resolved" {
		data := map[string]any{"recovery_run_id": *rid, "recovery_path": recoveryRel, "recovery_status": "resolved", "recovered_at": recovery["recovered_at"]}
		return emit(okEnvelope("run.recover", ctx, nil, data, nil, nil))
	}
	if fmt.Sprint(recovery["command"]) == "draft.accept" {
		draftPath := fmt.Sprint(recovery["draft_path"])
		archivePath := fmt.Sprint(recovery["archive_path"])
		if draftPath != "" && archivePath != "" {
			draftAbs, _, draftErr := safeRel(ctx.Root, draftPath)
			archiveAbs, _, archiveErr := safeRel(ctx.Root, archivePath)
			if draftErr != nil || archiveErr != nil {
				return emit(failEnvelope("run.recover", ctx, "PATH_OUTSIDE_ROOT", "recovery paths are invalid", map[string]any{"draft_error": fmt.Sprint(draftErr), "archive_error": fmt.Sprint(archiveErr)}))
			}
			if _, err := os.Stat(archiveAbs); os.IsNotExist(err) {
				if _, err := os.Stat(draftAbs); err == nil {
					if err := ensureDir(filepath.Dir(archiveAbs)); err != nil {
						return emit(failEnvelope("run.recover", ctx, "IO_ERROR", err.Error(), nil))
					}
					if err := os.Rename(draftAbs, archiveAbs); err != nil {
						return emit(failEnvelope("run.recover", ctx, "IO_ERROR", err.Error(), nil))
					}
				}
			}
		}
	}
	recovery["resolved"] = true
	recovery["recovery_status"] = "resolved"
	recovery["recovered_at"] = time.Now().UTC().Format(time.RFC3339)
	_ = writeJSON(recoveryPath, recovery)
	data := map[string]any{"recovery_run_id": *rid, "recovery_path": recoveryRel, "recovery_status": "resolved", "recovered_at": recovery["recovered_at"]}
	return emit(okEnvelope("run.recover", ctx, nil, data, nil, nil))
}

func cmdContentMigrate(_ commonFlags, ctx *WorldContext, _ []string) int {
	rid, runDir, err := createRun(ctx, "content.migrate")
	if err != nil {
		return emit(failEnvelope("content.migrate", ctx, "IO_ERROR", err.Error(), nil))
	}
	data := map[string]any{
		"migration_run_id":       rid,
		"migration_report_path":  filepath.ToSlash(filepath.Join("runs", rid, "migration.md")),
		"migration_actions_path": filepath.ToSlash(filepath.Join("runs", rid, "migration-actions.jsonl")),
		"candidates":             0,
		"blockers":               0,
		"partial_apply":          false,
	}
	_ = writeFileAtomic(filepath.Join(runDir, "migration.md"), []byte("# Migration Report\n\nNo automatic content mutation is performed.\n"), 0o644)
	_ = writeFileAtomic(filepath.Join(runDir, "migration-actions.jsonl"), []byte(""), 0o644)
	return emit(okEnvelope("content.migrate", ctx, rid, data, nil, nil))
}
