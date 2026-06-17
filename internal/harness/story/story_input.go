package story

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

func (s *Store) appendChoice(storyID string, u *Actor, choiceID, customMode, customText string) error {
	unlock, err := s.acquireLock(storyID, "turn_input", u.ID)
	if err != nil {
		return err
	}
	defer unlock()
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	if m.Status != "active" || m.Phase != "waiting_for_choice" {
		return errors.New("story is not waiting for input")
	}
	if u.Role != "admin" && (m.ActiveDriverID == "" || m.ActiveDriverID != u.ID) {
		return errors.New("only active driver can progress this story")
	}
	turns, err := s.readTurns(storyID)
	if err != nil || len(turns) == 0 {
		return errors.New("story has no turns")
	}
	prev := turns[len(turns)-1]
	if choiceID != "" {
		found := false
		for _, c := range prev.Choices {
			if c.ID == choiceID {
				found = true
			}
		}
		if !found {
			return errors.New("invalid choice")
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	inputLabel := firstNonEmpty(choiceID, customText)
	st, _ := s.readState(storyID)
	scene := GenerateLocalGMScene(prev, st, inputLabel, customMode)
	nextChoices := GenerateNextChoices(prev.TurnID+1, st)
	situation := GenerateCurrentSituation(st)
	t := Turn{TurnID: prev.TurnID + 1, BranchID: "branch_main", ParentTurnID: prev.TurnID, ActorID: u.ID, InputID: "input_" + randomID(), Source: "choice", SelectedChoiceID: choiceID, CustomInputMode: customMode, CustomText: customText, SceneGoal: "지금까지의 단서와 압박을 종합해 다음 선택의 기준을 정한다.", Conflict: "시간과 불확실성으로 판단 우선순위가 동시에 흔들린다.", TurningPoint: "방금 선택이 다음 장면의 위험 경로를 가르는 분기점이 된다.", Consequence: "선택의 추적이 다음 회차의 사실·문장·위험 평가를 바꾼다.", SceneTitle: fmt.Sprintf("Turn %d의 여파", prev.TurnID+1), SceneBody: scene, CurrentSituation: situation, RevealedFacts: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."}, Choices: nextChoices, CreatedAt: now}
	dir := s.storyDir(storyID)
	if err := appendJSONL(filepath.Join(dir, "events.jsonl"), map[string]any{"type": "turn_committed", "at": now, "turn": t}); err != nil {
		return err
	}
	if err := appendJSONL(filepath.Join(dir, "turns.jsonl"), t); err != nil {
		return err
	}
	st.Facts = appendUnique(st.Facts, t.RevealedFacts...)
	st.OpenThreads = appendUnique(st.OpenThreads, "방금 선택의 절차적 후속 근거 확보")
	m.CurrentTurn = t.TurnID
	m.UpdatedAt = now
	m.LatestSummary = t.CurrentSituation
	if err := writeJSONAtomic(filepath.Join(dir, "state.json"), st); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, "manifest.json"), m)
}

func (s *Store) askQuestion(storyID string, u *Actor, question string) error {
	m, err := s.readManifest(storyID)
	if err != nil {
		return err
	}
	_, err = s.submitQuestionJob(storyID, u, m.CurrentTurn, "", question)
	return err
}
