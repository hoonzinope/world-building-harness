package harness

import (
	"errors"
	"path/filepath"
	"time"
)

func (s *storyStore) appendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return appendJSONL(filepath.Join(s.storyDir(storyID), "events.jsonl"), map[string]any{
		"type":        eventType,
		"at":          now,
		"story_id":    storyID,
		"actor_id":    actorID,
		"from_status": fromStatus,
		"to_status":   toStatus,
	})
}

func (s *storyStore) changeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType string) error {
	unlock, err := s.acquireLock(storyID, "admin_update", "admin")
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
	if m.Status == nextStatus {
		return nil
	}
	if err := s.appendStoryLifecycleEvent(storyID, actorID, eventType, m.Status, nextStatus); err != nil {
		return err
	}
	m.Status = nextStatus
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}

func (s *storyStore) archiveStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "archived", "story_archived")
}
func (s *storyStore) restoreStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "active", "story_restored")
}
func (s *storyStore) deleteStory(storyID, actorID string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, "deleted", "story_deleted")
}

func (s *storyStore) adminUpdateStory(storyID, actorID, status, activeDriver string) error {
	unlock, err := s.acquireLock(storyID, "admin_update", actorID)
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
	if status != "" && status != m.Status {
		if err := s.appendStoryLifecycleEvent(storyID, actorID, "story_status_changed", m.Status, status); err != nil {
			return err
		}
		m.Status = status
	}
	if activeDriver == "__open__" {
		m.ActiveDriverID = ""
	} else if activeDriver != "" {
		m.ActiveDriverID = activeDriver
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}

func (s *storyStore) updateDriver(storyID string, u *authUser, action string) error {
	unlock, err := s.acquireLock(storyID, "driver_"+action, u.ID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return errors.New("driver can only change while story is waiting")
	}
	switch action {
	case "release":
		if u.Role != "admin" && m.ActiveDriverID != u.ID {
			return errors.New("only active driver can open this story")
		}
		m.ActiveDriverID = ""
	case "claim":
		if m.ActiveDriverID != "" && u.Role != "admin" {
			return errors.New("story already has an active driver")
		}
		m.ActiveDriverID = u.ID
	default:
		return errors.New("unknown driver action")
	}
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
}
