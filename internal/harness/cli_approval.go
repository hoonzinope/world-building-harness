package harness

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

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
