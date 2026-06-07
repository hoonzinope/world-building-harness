package story

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (s *Store) acquireLock(storyID, reason, actorID string) (func(), error) {
	dir := s.storyDir(storyID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "lock.json")
	now := time.Now().UTC()
	if b, err := os.ReadFile(path); err == nil {
		var existing map[string]any
		_ = json.Unmarshal(b, &existing)
		if at, _ := time.Parse(time.RFC3339, fmt.Sprint(existing["acquired_at"])); !at.IsZero() && now.Sub(at) > StoryLockTimeout {
			_ = os.Remove(path)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, errors.New("story is locked")
		}
		return nil, err
	}
	lock := map[string]any{"story_id": storyID, "reason": reason, "actor_id": actorID, "acquired_at": now.Format(time.RFC3339)}
	b, _ := json.MarshalIndent(lock, "", "  ")
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	fsyncDir(dir)
	return func() { _ = os.Remove(path); fsyncDir(dir) }, nil
}
