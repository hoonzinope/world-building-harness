package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
