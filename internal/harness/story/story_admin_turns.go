package story

import (
	"errors"
	"path/filepath"
	"time"
)

func (s *Store) editCurrentTurn(storyID, actorID, sceneBody, currentSituation string) error {
	unlock, err := s.acquireLock(storyID, "admin_turn_edit", actorID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return err
	}
	idx, ok := findStoryTurn(turns, m.CurrentTurn)
	if !ok {
		return errors.New("current turn not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	before := turns[idx]
	turns[idx].SceneBody = sceneBody
	turns[idx].CurrentSituation = currentSituation
	m.CurrentTurn = turns[idx].TurnID
	m.LatestSummary = currentSituation
	m.UpdatedAt = now
	if err := s.writeStoryTurnsProjection(storyID, turns); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m); err != nil {
		return err
	}
	if err := s.rewriteStorySummary(storyID, currentSituation); err != nil {
		return err
	}
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":                "turn_edited_by_admin",
		"at":                  now,
		"story_id":            storyID,
		"actor_id":            actorID,
		"turn_id":             before.TurnID,
		"previous_scene_body": before.SceneBody,
		"previous_situation":  before.CurrentSituation,
		"turn":                turns[idx],
	})
}

func (s *Store) rollbackStoryToTurn(storyID, actorID string, targetTurnID int) error {
	unlock, err := s.acquireLock(storyID, "admin_turn_rollback", actorID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if s.storyHasBlockingGMJob(m) {
		return errors.New("story has an active GM job")
	}
	turns, err := s.readTurns(storyID)
	if err != nil {
		return err
	}
	idx, ok := findStoryTurn(turns, targetTurnID)
	if !ok {
		return errors.New("selected turn not found")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	kept := append([]Turn(nil), turns[:idx+1]...)
	fromTurnID := m.CurrentTurn
	if fromTurnID == 0 && len(turns) > 0 {
		fromTurnID = turns[len(turns)-1].TurnID
	}
	m.CurrentTurn = targetTurnID
	m.LatestSummary = kept[len(kept)-1].CurrentSituation
	m.UpdatedAt = now
	if err := s.writeStoryTurnsProjection(storyID, kept); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m); err != nil {
		return err
	}
	if err := s.rewriteStorySummary(storyID, m.LatestSummary); err != nil {
		return err
	}
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":            "story_rolled_back_by_admin",
		"at":              now,
		"story_id":        storyID,
		"actor_id":        actorID,
		"from_turn_id":    fromTurnID,
		"to_turn_id":      targetTurnID,
		"kept_turn_count": len(kept),
		"removed_turn_ids": func() []int {
			if len(turns) <= len(kept) {
				return nil
			}
			removed := make([]int, 0, len(turns)-len(kept))
			for _, turn := range turns[len(kept):] {
				removed = append(removed, turn.TurnID)
			}
			return removed
		}(),
	})
}
