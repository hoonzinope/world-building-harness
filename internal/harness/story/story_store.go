package story

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func openStoryStore(root, packsRoot string) (*Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &Store{root: root, packsRoot: packsRoot, exportRoot: filepath.Join(filepath.Dir(root), "exports")}, nil
}

func (s *Store) storyDir(id string) string { return filepath.Join(s.root, id) }

func (s *Store) listStories() ([]Manifest, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var out []Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := s.readManifest(e.Name())
		if err == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	out = dedupeStoryManifests(out)
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

func dedupeStoryManifests(stories []Manifest) []Manifest {
	bySource := map[string]int{}
	out := make([]Manifest, 0, len(stories))
	for _, m := range stories {
		sourcePath := strings.TrimSpace(m.SourceDraftPath)
		if sourcePath == "" {
			out = append(out, m)
			continue
		}
		if idx, ok := bySource[sourcePath]; ok {
			if betterStoryManifest(m, out[idx]) {
				out[idx] = m
			}
			continue
		}
		bySource[sourcePath] = len(out)
		out = append(out, m)
	}
	return out
}

func betterStoryManifest(candidate, current Manifest) bool {
	candidateDeleted := ManifestIsDeleted(candidate)
	currentDeleted := ManifestIsDeleted(current)
	if candidateDeleted != currentDeleted {
		return !candidateDeleted && currentDeleted
	}
	if candidate.UpdatedAt != current.UpdatedAt {
		return candidate.UpdatedAt > current.UpdatedAt
	}
	if candidate.CreatedAt != current.CreatedAt {
		return candidate.CreatedAt > current.CreatedAt
	}
	return candidate.ID > current.ID
}

func ManifestIsDeleted(m Manifest) bool {
	switch m.Status {
	case "deleted", "archived", "completed":
		return true
	default:
		return false
	}
}

func (s *Store) readManifest(id string) (Manifest, error) {
	var m Manifest
	err := readJSON(filepath.Join(s.storyDir(id), "manifest.json"), &m)
	return m, err
}

func (s *Store) readState(id string) (State, error) {
	var st State
	err := readJSON(filepath.Join(s.storyDir(id), "state.json"), &st)
	return st, err
}

func (s *Store) readTurns(id string) ([]Turn, error) {
	var turns []Turn
	err := readStoryJSONL(filepath.Join(s.storyDir(id), "turns.jsonl"), func(b []byte) error {
		var t Turn
		if err := json.Unmarshal(b, &t); err != nil {
			return err
		}
		turns = append(turns, t)
		return nil
	})
	return turns, err
}

func (s *Store) readQA(id string) ([]Question, error) {
	var qa []Question
	err := readStoryJSONL(filepath.Join(s.storyDir(id), "qa.jsonl"), func(b []byte) error {
		var q Question
		if err := json.Unmarshal(b, &q); err != nil {
			return err
		}
		qa = append(qa, q)
		return nil
	})
	return qa, err
}

func mustReadTurns(s *Store, storyID string) []Turn {
	turns, _ := s.readTurns(storyID)
	return turns
}

func (s *Store) createStory(m Manifest, st State, turns []Turn) error {
	dir := s.storyDir(m.ID)
	if err := os.MkdirAll(filepath.Join(dir, "jobs"), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "memory-cards"), 0o700); err != nil {
		return err
	}
	for _, t := range turns {
		if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "turn_committed", "at": t.CreatedAt, "turn": t}); err != nil {
			return err
		}
		if err := appendJSONL(filepath.Join(dir, "turns.jsonl"), t); err != nil {
			return err
		}
	}
	if err := writeJSONAtomic(filepath.Join(dir, "manifest.json"), m); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "summary.md"), []byte(m.LatestSummary+"\n"))
}

func (s *Store) writeStoryTurnsProjection(storyID string, turns []Turn) error {
	var b bytes.Buffer
	for _, turn := range turns {
		data, err := json.Marshal(turn)
		if err != nil {
			return err
		}
		if _, err := b.Write(data); err != nil {
			return err
		}
		if err := b.WriteByte('\n'); err != nil {
			return err
		}
	}
	return writeAtomic(filepath.Join(s.storyDir(storyID), "turns.jsonl"), b.Bytes())
}

func (s *Store) rewriteStorySummary(storyID, summary string) error {
	return writeAtomic(filepath.Join(s.storyDir(storyID), "summary.md"), []byte(firstNonEmpty(strings.TrimSpace(summary), "No summary yet")+"\n"))
}

func findStoryTurn(turns []Turn, turnID int) (int, bool) {
	for i := range turns {
		if turns[i].TurnID == turnID {
			return i, true
		}
	}
	return -1, false
}
