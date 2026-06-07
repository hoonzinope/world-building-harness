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
