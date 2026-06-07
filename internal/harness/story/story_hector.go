package story

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) ensureSeedStories(actorID string) error {
	id, _, err := s.importHector(actorID)
	if err != nil {
		return err
	}
	return s.refreshHectorHistory(id, actorID)
}

func (s *Store) importHector(actorID string) (string, bool, error) {
	sourceRel := HectorSourceRel()
	b, err := os.ReadFile(filepath.Join(s.packsRoot, sourceRel))
	if err != nil {
		return "", false, err
	}
	hash := HectorSourceHash(b)
	parsed, err := ParseHectorDraft(string(b))
	if err != nil {
		return "", false, err
	}
	existing, _ := s.listStories()
	for _, m := range existing {
		if m.SourceDraftPath == filepath.ToSlash(sourceRel) {
			return s.updateHectorImportedStory(m, parsed, hash, true)
		}
	}
	id := "story_hector_first_residual_check"
	if _, err := os.Stat(s.storyDir(id)); err == nil {
		id = id + "_" + randomID()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := newHectorSeedManifest(id, parsed, actorID, sourceRel, hash, now)
	st := newHectorSeedState()
	turns := parsed.Turns
	if len(turns) == 0 {
		turns = []Turn{newHectorFallbackTurn(parsed, actorID, now)}
	}
	for i := range turns {
		turns[i].ActorID, turns[i].CreatedAt = actorID, now
	}
	if err := s.createStory(m, st, turns); err != nil {
		return "", false, err
	}
	return id, false, nil
}

func (s *Store) refreshHectorHistory(storyID, actorID string) error {
	b, err := os.ReadFile(filepath.Join(s.packsRoot, HectorSourceRel()))
	if err != nil {
		return err
	}
	currentHash := HectorSourceHash(b)
	parsed, err := s.parseHectorHistory()
	if err != nil || len(parsed.Turns) == 0 {
		return err
	}
	existing, _ := s.readTurns(storyID)
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.SourceHash == currentHash && hectorTurnsMatch(existing, parsed.Turns) {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	existingTurn := maxStoryTurn(existing)
	nextTurn := maxInt(m.CurrentTurn, maxInt(existingTurn, parsed.TurnID))
	m.CurrentTurn, m.UpdatedAt, m.SourceHash = nextTurn, now, currentHash
	if nextTurn <= parsed.TurnID {
		m.LatestSummary = hectorCurrentSituation()
	}
	if m.ActiveDriverID == "" {
		m.ActiveDriverID = actorID
	}
	if existingTurn > parsed.TurnID || m.CurrentTurn > parsed.TurnID {
		return writeJSONAtomic(filepath.Join(s.storyDir(storyID), "manifest.json"), m)
	}
	st, _ := s.readState(storyID)
	for i := range parsed.Turns {
		parsed.Turns[i].ActorID, parsed.Turns[i].CreatedAt = actorID, now
	}
	return s.replaceStory(m, st, parsed.Turns)
}

func maxInt(values ...int) int {
	max := 0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func maxStoryTurn(turns []Turn) int {
	max := 0
	for _, turn := range turns {
		if turn.TurnID > max {
			max = turn.TurnID
		}
	}
	return max
}

func (s *Store) parseHectorHistory() (HectorParsed, error) {
	sourcePath := filepath.Join(s.packsRoot, HectorSourceRel())
	paths, _ := filepath.Glob(filepath.Join(s.packsRoot, "lumen-federation", "runs", "inbox", "*-body.md"))
	paths = append(paths, sourcePath)
	sort.Strings(paths)
	byTurn := map[int]Turn{}
	title := "헥터: 첫 잔명 대조"
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		p, err := ParseHectorDraft(string(b))
		if err != nil {
			continue
		}
		if p.Title != "" {
			title = p.Title
		}
		for _, t := range p.Turns {
			byTurn[t.TurnID] = t
		}
	}
	if len(byTurn) == 0 {
		return HectorParsed{}, errors.New("no hector history turns found")
	}
	ids := make([]int, 0, len(byTurn))
	for id := range byTurn {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	turns := make([]Turn, 0, len(ids))
	for _, id := range ids {
		t := byTurn[id]
		t.BranchID = "branch_main"
		t.ParentTurnID = id - 1
		if id == 1 {
			t.ParentTurnID = 0
		}
		t.InputID = fmt.Sprintf("import_hector_turn_%d", id)
		t.Source = "import"
		turns = append(turns, t)
	}
	latest := turns[len(turns)-1]
	return HectorParsed{Title: title, TurnID: latest.TurnID, SceneBody: latest.SceneBody, Facts: latest.RevealedFacts, Choices: latest.Choices, Turns: turns}, nil
}

func hectorTurnsMatch(existing, parsed []Turn) bool {
	if len(existing) != len(parsed) {
		return false
	}
	for i := range existing {
		if !hectorTurnMatch(existing[i], parsed[i]) {
			return false
		}
	}
	return true
}
func hectorTurnMatch(existing, parsed Turn) bool {
	if existing.TurnID != parsed.TurnID || existing.BranchID != parsed.BranchID || existing.ParentTurnID != parsed.ParentTurnID || existing.InputID != parsed.InputID || existing.Source != parsed.Source || existing.SelectedChoiceID != parsed.SelectedChoiceID || existing.CustomInputMode != parsed.CustomInputMode || existing.CustomText != parsed.CustomText || existing.SceneTitle != parsed.SceneTitle || existing.SceneBody != parsed.SceneBody || existing.CurrentSituation != parsed.CurrentSituation {
		return false
	}
	if strings.Join(existing.RevealedFacts, "\n") != strings.Join(parsed.RevealedFacts, "\n") {
		return false
	}
	if len(existing.Choices) != len(parsed.Choices) {
		return false
	}
	for i := range existing.Choices {
		if existing.Choices[i] != parsed.Choices[i] {
			return false
		}
	}
	return true
}

func (s *Store) replaceStory(m Manifest, st State, turns []Turn) error {
	dir := s.storyDir(m.ID)
	for _, name := range []string{"events.jsonl", "turns.jsonl", "manifest.json", "state.json", "summary.md"} {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return s.createStory(m, st, turns)
}

func HectorSourceRel() string {
	return filepath.Join("lumen-federation", "drafts", "storylets", "hector_first_residual_check.md")
}

func HectorSourceHash(b []byte) string {
	hashBytes := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(hashBytes[:])
}

func newHectorSeedManifest(id string, parsed HectorParsed, actorID, sourceRel, hash, now string) Manifest {
	return Manifest{
		ID:              id,
		Title:           firstNonEmpty(parsed.Title, "헥터: 첫 잔명 대조"),
		WorldID:         "lumen-federation",
		Status:          "active",
		Phase:           "waiting_for_choice",
		CurrentTurn:     parsed.TurnID,
		ActiveDriverID:  actorID,
		SourceDraftPath: filepath.ToSlash(sourceRel),
		SourceHash:      hash,
		CreatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		LatestSummary:   hectorCurrentSituation(),
	}
}

func newHectorSeedState() State {
	return State{
		Location: "베이르 제3정정실 / 마라 베온 대기실 연결 채널",
		ActiveCharacters: []string{
			"헥터",
			"라우",
			"마라 베온",
			"아델 카이",
			"V-13 베른",
		},
		Facts: []string{
			"마라 베온은 의료 진정 상태로 안정화 중이다.",
			"대체 우안 동기화율은 하락하기 시작했다.",
			"라우는 생존 보호 명령 범위를 표본 안정자 내부 반응 조정까지 확장했다.",
			"감리단은 이 확장이 자기 고유 권한 침해라고 주장한다.",
			"17-B 기록군에는 23개의 사망 보상권, 9개의 신체 권리 분쟁, 4개의 생존 주장 잔류 건이 묶여 있다.",
		},
		OpenThreads: []string{
			"17-B 기록군 전체를 열 추가 연결점 확보",
			"감리단 권한 충돌 대응",
			"마라 베온의 생존 주장과 원 신체 권리 주장 확정",
			"17B-STB-EYE-03의 표본 안정자 지위 반박",
		},
		Risks: []string{
			"감리단이 범위 확장 철회를 요구하고 있다.",
			"기록군 전체 재심사가 다수의 보상권과 신체 권리 분쟁을 흔들 수 있다.",
			"마라의 안정 상태는 의료 조치에 의존하고 있어 장기 안전이 확정되지 않았다.",
		},
		Flags: []string{
			"mara_medically_stabilized",
			"survival_protection_extended",
			"inspectorate_conflict_open",
			"record_group_17b_revealed",
		},
	}
}

func newHectorFallbackTurn(parsed HectorParsed, actorID, now string) Turn {
	return Turn{
		TurnID:           parsed.TurnID,
		BranchID:         "branch_main",
		ParentTurnID:     18,
		ActorID:          actorID,
		InputID:          "import_hector_turn_19",
		Source:           "import",
		SelectedChoiceID: "B",
		SceneTitle:       "첫 잔명 대조",
		SceneBody:        parsed.SceneBody,
		CurrentSituation: hectorCurrentSituation(),
		RevealedFacts:    parsed.Facts,
		Choices:          parsed.Choices,
		CreatedAt:        now,
	}
}

func (s *Store) updateHectorImportedStory(m Manifest, parsed HectorParsed, hash string, imported bool) (string, bool, error) {
	_ = imported
	updated := false
	title := firstNonEmpty(parsed.Title, "헥터: 첫 잔명 대조")
	nextTurn := maxInt(m.CurrentTurn, parsed.TurnID)
	nextSummary := m.LatestSummary
	if nextTurn <= parsed.TurnID {
		nextSummary = hectorCurrentSituation()
	}
	if m.SourceHash != hash || m.Title != title || m.CurrentTurn != nextTurn || m.LatestSummary != nextSummary {
		now := time.Now().UTC().Format(time.RFC3339)
		m.Title = title
		m.SourceHash = hash
		m.UpdatedAt = now
		m.CurrentTurn = nextTurn
		m.LatestSummary = nextSummary
		updated = true
	}
	if updated {
		if err := writeJSONAtomic(filepath.Join(s.storyDir(m.ID), "manifest.json"), m); err != nil {
			return "", false, err
		}
	}
	return m.ID, true, nil
}

func hectorCurrentSituation() string {
	return "마라 베온은 안정화 중이고, 감리단 권한 충돌이 열린 상태다. 17-B 기록군 전체를 열 추가 연결점이 필요하다."
}
