package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type mockGMProvider struct{}

func (mockGMProvider) Generate(ctx context.Context, req GMRequest) (GMOutput, string, string, string, error) {
	_ = ctx
	if req.Job.JobType == "question_answer" {
		return mockQuestionOutput(req)
	}
	if req.Job.JobType == "prologue" {
		return mockPrologueOutput(req)
	}
	prev := req.Turns[len(req.Turns)-1]
	st := req.State
	input := firstNonEmpty(req.Job.Input.SelectedChoiceID, req.Job.Input.CustomText, "직접 행동")
	scene := GenerateLocalGMScene(prev, st, input, req.Job.Input.CustomInputMode)
	out := GMOutput{
		SchemaVersion: "story-gm-output.v1", StoryID: req.Job.StoryID,
		Turn:         GMOutputTurn{BranchID: "branch_main", TurnID: req.Job.TurnID, ParentTurnID: req.Job.ParentTurnID, InputID: req.Job.Input.ID, JobID: req.Job.ID, Source: "choice", SelectedChoiceID: req.Job.Input.SelectedChoiceID, CustomInputMode: req.Job.Input.CustomInputMode},
		SceneGoal:    "방금 선택의 즉시 후과를 정리하고 다음 행동의 기준을 세운다.",
		Conflict:     "현장 대응과 장부상 절차가 동시에 요구되어 시간이 압박된다.",
		TurningPoint: "동료가 추가 정보를 가져오면서 우선순위를 다시 정해야 한다.",
		Consequence:  "지금의 판단이 다음 턴의 위험도와 가용 자원을 바꾼다.",
		SceneTitle:   fmt.Sprintf("Turn %d의 여파", req.Job.TurnID), SceneBody: scene,
		CurrentSituation: GenerateCurrentSituation(st), RevealedFacts: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."},
		StatePatch: GMStatePatch{FactsAdd: []string{"이번 입력은 runtime story 이벤트로만 저장되며 canon에 반영되지 않았다."}, OpenThreadsAdd: []string{"방금 선택의 절차적 후속 근거 확보"}, RisksAdd: []string{"선택의 후속 조치가 지연되면 다음 장면의 대응 폭이 좁아질 수 있다."}, SummaryPatch: GenerateCurrentSituation(st)},
		Resolution: "accepted", Choices: GenerateNextChoices(req.Job.TurnID, st),
	}
	annotateChoiceIntent(out.Choices)
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}

func mockQuestionOutput(req GMRequest) (GMOutput, string, string, string, error) {
	if req.Job.Question == nil {
		return GMOutput{}, "", "mock", "mock-story-gm", errors.New("missing question")
	}
	answer := "Turn 기준으로 답하면, " + SummarizeQuestionAnswer(req.Job.Question.Question, req.State)
	out := GMOutput{SchemaVersion: "story-question-answer.v1", StoryID: req.Job.StoryID, Answer: answer}
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}

func mockPrologueOutput(req GMRequest) (GMOutput, string, string, string, error) {
	setup := req.Job.Setup
	if setup == nil {
		return GMOutput{}, "", "mock", "mock-story-gm", errors.New("missing setup")
	}
	name := firstNonEmpty(setup.CharacterName, "새 인물")
	traits := strings.TrimSpace(setup.Traits)
	location, scene, summary, facts, _, _, choices := luceraPrologueSeed(name, traits)
	out := GMOutput{
		SchemaVersion: "story-gm-output.v1", StoryID: req.Job.StoryID,
		Turn:             GMOutputTurn{BranchID: "branch_main", TurnID: 1, ParentTurnID: 0, InputID: "setup_" + req.Job.ID, JobID: req.Job.ID, Source: "setup"},
		SceneGoal:        "병동의 첫 위기를 정리하고 주인공의 자리와 책임을 드러낸다.",
		Conflict:         "새 환자와 기존 환자의 처치가 동시에 필요하지만 자원은 부족하다.",
		TurningPoint:     "긴급 장부 확인 결과 우선순위를 바꿔야 한다는 사실이 드러난다.",
		Consequence:      "어떤 선택을 하든 한쪽의 회복이 늦어지고 다음 사건의 부담이 남는다.",
		SceneTitle:       firstNonEmpty(setup.Style, "조사극") + "의 시작",
		SceneBody:        scene,
		CurrentSituation: summary,
		RevealedFacts:    facts,
		StatePatch:       GMStatePatch{LocationSet: location, ActiveCharactersSet: []string{name}, FactsAdd: facts, OpenThreadsAdd: []string{"새 환자와 기존 환자 중 누구를 먼저 살릴지 결정", "병동의 부족한 자원을 어떻게 배분할지 판단", "공공 수선과 행정 장부를 어떻게 맞출지 정리"}, RisksAdd: []string{"어느 쪽을 선택해도 다른 쪽의 상태가 악화될 수 있다.", "과로와 저안개 절차 때문에 판단 여력이 흔들릴 수 있다.", "장부상 허가가 늦어지면 회복 공공재 배분이 밀릴 수 있다."}, SummaryPatch: summary},
		Resolution:       "accepted",
		Choices:          choices,
	}
	annotateChoiceIntent(out.Choices)
	raw, _ := json.Marshal(out)
	return out, string(raw), "mock", "mock-story-gm", nil
}
