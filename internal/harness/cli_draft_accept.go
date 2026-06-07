package harness

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
