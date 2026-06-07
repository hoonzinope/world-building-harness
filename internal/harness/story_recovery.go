package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type storyRecoveryReport struct {
	StoryID        string   `json:"story_id"`
	RecoveryStatus string   `json:"recovery_status"`
	CheckedFiles   []string `json:"checked_files"`
	RepairedItems  []string `json:"repaired_items"`
	LockRemoved    bool     `json:"lock_removed"`
}

func (s *storyStore) recoverStory(storyID string) (storyRecoveryReport, error) {
	dir := s.storyDir(storyID)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return storyRecoveryReport{}, errors.New("story not found")
		}
		return storyRecoveryReport{}, err
	}
	if !info.IsDir() {
		return storyRecoveryReport{}, errors.New("story path is not a directory")
	}
	lockRemoved := false
	lockPath := filepath.Join(dir, "lock.json")
	if b, err := os.ReadFile(lockPath); err == nil {
		var lock map[string]any
		if json.Unmarshal(b, &lock) == nil {
			if at, err := time.Parse(time.RFC3339, fmt.Sprint(lock["acquired_at"])); err == nil && !at.IsZero() {
				if time.Since(at) > storyLockTimeout {
					if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
						return storyRecoveryReport{}, err
					}
					_ = fsyncDir(dir)
					lockRemoved = true
				} else {
					return storyRecoveryReport{}, errors.New("story is locked")
				}
			}
		}
	}
	unlock, err := s.acquireLock(storyID, "store_recover", "admin")
	if err != nil {
		return storyRecoveryReport{}, err
	}
	defer unlock()
	checked := []string{"events.jsonl", "turns.jsonl", "qa.jsonl"}
	repaired := []string{}
	eventsPath := filepath.Join(dir, "events.jsonl")
	eventsBefore, err := readFileIfExists(eventsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if err := readStoryJSONL(eventsPath, func(b []byte) error { var v map[string]any; return json.Unmarshal(b, &v) }); err != nil {
		return storyRecoveryReport{}, err
	}
	eventsAfter, err := readFileIfExists(eventsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(eventsBefore, eventsAfter) {
		repaired = append(repaired, "events.jsonl")
	}
	turnsPath := filepath.Join(dir, "turns.jsonl")
	turnsBefore, err := readFileIfExists(turnsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if _, err := s.readTurns(storyID); err != nil {
		return storyRecoveryReport{}, err
	}
	turnsAfter, err := readFileIfExists(turnsPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(turnsBefore, turnsAfter) {
		repaired = append(repaired, "turns.jsonl")
	}
	qaPath := filepath.Join(dir, "qa.jsonl")
	qaBefore, err := readFileIfExists(qaPath)
	if err != nil {
		return storyRecoveryReport{}, err
	}
	if _, err := s.readQA(storyID); err != nil {
		return storyRecoveryReport{}, err
	}
	qaAfter, err := readFileIfExists(qaPath)
	if err != nil {
		return storyRecoveryReport{}, err
	} else if !bytes.Equal(qaBefore, qaAfter) {
		repaired = append(repaired, "qa.jsonl")
	}
	status := "checked"
	if len(repaired) > 0 || lockRemoved {
		status = "recovered"
	}
	if err := appendJSONL(eventsPath, map[string]any{"type": "story_recovered", "at": time.Now().UTC().Format(time.RFC3339), "story_id": storyID, "checked_files": checked, "repaired_items": repaired, "lock_removed": lockRemoved, "recovery_status": status}); err != nil {
		return storyRecoveryReport{}, err
	}
	return storyRecoveryReport{StoryID: storyID, RecoveryStatus: status, CheckedFiles: checked, RepairedItems: repaired, LockRemoved: lockRemoved}, nil
}
