package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *storyStore) exportStoryBundle(storyID string, actor *authUser) (string, error) {
	m, err := s.readManifest(storyID)
	if err != nil {
		return "", err
	}
	if s.storyHasBlockingGMJob(m) {
		return "", errors.New("story has an active GM job")
	}
	st, err := s.readState(storyID)
	if err != nil {
		return "", err
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.exportRoot, 0o700); err != nil {
		return "", err
	}
	bundleID := time.Now().UTC().Format("20060102T150405Z") + "_" + randomID()
	dir := filepath.Join(s.exportRoot, storyID, bundleID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "source_manifest.json"), m); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "turn_hashes.json"), map[string]any{"story_id": storyID, "turns": turnHashes(turns)}); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, "summary.md"), []byte(firstNonEmpty(strings.TrimSpace(m.LatestSummary), "No summary yet")+"\n")); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, "storylet.md"), []byte(renderStoryletBundle(m, st, turns))); err != nil {
		return "", err
	}
	exportedAt := time.Now().UTC().Format(time.RFC3339)
	exportManifest := map[string]any{
		"story_id": storyID, "world_id": m.WorldID, "exported_at": exportedAt, "status": "draft_pending",
		"draft_target_suggestion": filepath.ToSlash(filepath.Join("drafts", "storylets", storyID+".md")),
		"source_files":            []string{"source_manifest.json", "turn_hashes.json", "storylet.md", "summary.md"},
		"turn_hashes_path":        "turn_hashes.json", "storylet_path": "storylet.md", "summary_path": "summary.md",
		"next_admin_cli_instructions": []string{"Copy the bundle into the admin writer workflow.", "Create the draft at the suggested target path.", "Mark the draft as ready before republishing or review."},
	}
	if actor != nil && strings.TrimSpace(actor.ID) != "" {
		exportManifest["exported_by"] = actor.ID
	}
	if err := writeJSONAtomic(filepath.Join(dir, "export_manifest.json"), exportManifest); err != nil {
		return "", err
	}
	actorID := ""
	if actor != nil {
		actorID = strings.TrimSpace(actor.ID)
	}
	if err := appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{"type": "story_export_handoff", "at": exportedAt, "story_id": storyID, "actor_id": actorID, "bundle_path": dir, "target_draft_suggestion": filepath.ToSlash(filepath.Join("drafts", "storylets", storyID+".md")), "status": "draft_pending"}); err != nil {
		return "", err
	}
	return dir, nil
}

func turnHashes(turns []storyTurn) []map[string]any {
	out := make([]map[string]any, 0, len(turns))
	for _, turn := range turns {
		b, _ := json.Marshal(turn)
		sum := sha256.Sum256(b)
		out = append(out, map[string]any{"turn_id": turn.TurnID, "branch_id": turn.BranchID, "source": turn.Source, "created_at": turn.CreatedAt, "hash": "sha256:" + hex.EncodeToString(sum[:]), "input_id": turn.InputID, "actor_id": turn.ActorID, "parent_turn": turn.ParentTurnID})
	}
	return out
}

func renderStoryletBundle(m storyManifest, st storyState, turns []storyTurn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", m.Title)
	fmt.Fprintf(&b, "- story_id: `%s`\n- status: `%s`\n- phase: `%s`\n- current_turn: `%d`\n- active_driver: `%s`\n", m.ID, m.Status, m.Phase, m.CurrentTurn, firstNonEmpty(m.ActiveDriverID, "open"))
	if m.SourceDraftPath != "" {
		fmt.Fprintf(&b, "- source_draft_path: `%s`\n", m.SourceDraftPath)
	}
	fmt.Fprintf(&b, "\n## Summary\n\n%s\n\n## State\n\n- Location: %s\n- Active characters: %s\n\n### Facts\n", firstNonEmpty(strings.TrimSpace(m.LatestSummary), "No summary yet"), firstNonEmpty(st.Location, "미정"), strings.Join(st.ActiveCharacters, ", "))
	for _, fact := range st.Facts {
		fmt.Fprintf(&b, "- %s\n", fact)
	}
	b.WriteString("\n### Open threads\n")
	for _, thread := range st.OpenThreads {
		fmt.Fprintf(&b, "- %s\n", thread)
	}
	b.WriteString("\n### Risks\n")
	for _, risk := range st.Risks {
		fmt.Fprintf(&b, "- %s\n", risk)
	}
	b.WriteString("\n## Turns\n")
	for _, turn := range turns {
		fmt.Fprintf(&b, "\n### Turn %d\n\n", turn.TurnID)
		fmt.Fprintf(&b, "- actor: `%s`\n- source: `%s`\n- input_id: `%s`\n\n", turn.ActorID, turn.Source, turn.InputID)
		if turn.SceneTitle != "" {
			fmt.Fprintf(&b, "**%s**\n\n", turn.SceneTitle)
		}
		if turn.SceneBody != "" {
			b.WriteString(strings.TrimSpace(turn.SceneBody) + "\n\n")
		}
		if turn.CurrentSituation != "" {
			fmt.Fprintf(&b, "_Current situation: %s_\n\n", turn.CurrentSituation)
		}
		if len(turn.RevealedFacts) > 0 {
			b.WriteString("Facts:\n")
			for _, fact := range turn.RevealedFacts {
				fmt.Fprintf(&b, "- %s\n", fact)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
