package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/server"
	"github.com/hoonzi/world-harness/internal/harness/telegram"
)

func Run(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "usage: world-tool <resource> <action> [flags]")
		return 0
	}
	if args[0] == "serve" {
		return server.Run(args[1:])
	}
	if args[0] == "telegram" {
		return telegram.Run(args[1:])
	}
	if args[0] == "admin" {
		return runAdmin(args[1:])
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
	case "story.export-draft":
		return withWorld("story.export-draft", rest, cmdStoryExportDraft)
	case "story.recover":
		return withWorld("story.recover", rest, cmdStoryRecover)
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
	"input.stage":        true,
	"draft.create":       true,
	"draft.update":       true,
	"draft.validate":     true,
	"draft.diff":         true,
	"draft.accept":       true,
	"draft.reject":       true,
	"story.export-draft": true,
	"story.recover":      true,
	"content.validate":   true,
	"content.migrate":    true,
	"approval.attest":    true,
	"run.recover":        true,
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
