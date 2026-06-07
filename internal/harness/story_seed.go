package harness

import (
	"fmt"
	"strings"
	"time"
)

func (s *storyStore) createDemoStory(actorID, title, style, characterName, traits string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := "story_" + randomID()
	name := firstNonEmpty(strings.TrimSpace(characterName), "새 인물")
	style = firstNonEmpty(strings.TrimSpace(style), "조사극")
	traits = strings.TrimSpace(traits)
	location, scene, summary, facts, openThreads, risks, choices := luceraPrologueSeed(name, traits)
	m := storyManifest{ID: id, Title: firstNonEmpty(strings.TrimSpace(title), name+"의 이야기"), WorldID: "lumen-federation", Status: "active", Phase: "waiting_for_choice", CurrentTurn: 1, ActiveDriverID: actorID, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now, LatestSummary: summary}
	st := storyState{Location: location, ActiveCharacters: []string{name}, Facts: facts, OpenThreads: openThreads, Risks: risks, Flags: []string{"runtime_story_created"}}
	turn := storyTurn{TurnID: 1, BranchID: "branch_main", ActorID: actorID, InputID: "setup_" + randomID(), Source: "setup", SceneTitle: style + "의 시작", SceneBody: scene, CurrentSituation: m.LatestSummary, RevealedFacts: st.Facts, Choices: choices, CreatedAt: now, CustomText: traits}
	return id, s.createStory(m, st, []storyTurn{turn})
}

func luceraPrologueSeed(name, traits string) (string, string, string, []string, []string, []string, []storyChoice) {
	subject := name + "는"
	location := "루세라 야간 진료동"
	scene := fmt.Sprintf("루멘 연방의 의료 도시 루세라에서는 불빛이 생명을 살리지만, 같은 빛이 괴물의 길을 그어 버리기도 한다. 그래서 이 병동은 늘 저안개 차단막을 낮추고, 낮은 절차로 움직이며, 공공 수선과 회복 공공재 배분을 함께 계산해야 한다.\n\n%s 간호기록판을 팔에 끼운 채 잠깐 멈춰 섰다. 접수대 위에는 오늘의 환자 목록과 공공 수선 요청서, 그리고 행정 장부의 빈 칸이 겹쳐 놓여 있었다. 방금 들어온 환자 셋의 기록은 서로 다른 증상을 말하고 있었지만, 병동의 빈 침상 수는 같은 답만 내놓았다. 더 받을 수 없다.\n\n그때 접수대 쪽에서 누군가 %s의 이름을 불렀다. 새 환자 하나가 쓰러졌고, 동시에 이미 누워 있던 아이의 보호자가 약속된 처치를 왜 미루냐고 묻기 시작했다. 둘 다 기다릴 수 없지만, %s의 손은 하나뿐이다.\n\n병동 안에는 배분표를 다시 맞추는 사람도, 공공 수선 반에게 문의하는 사람도, 장부상 허가를 확인하는 사람도 있었다. 루세라에서 회복은 늘 누군가의 자원을 다시 계산하는 일과 붙어 다녔다.", subject, name, name)
	if traits != "" {
		scene += "\n\n초기 설정 메모: " + traits
	}
	summary := fmt.Sprintf("루멘 연방의 루세라 야간 진료동에서 공공 수선과 회복 공공재, 그리고 행정 장부가 서로를 밀어내고 있다.")
	facts := []string{"루멘 연방의 루세라는 병원, 약, 수면을 맡는 의료 도시다.", "병동 불빛은 회복 공공재이지만 저안개 차단과 낮은 절차로 다뤄야 한다.", "공공 수선과 자원 배분은 보험과 행정 장부의 허가를 거친다.", fmt.Sprintf("주인공은 %s이다.", name), fmt.Sprintf("%s 루세라의 간호사다.", subject), fmt.Sprintf("초기 배경은 %s이다.", location), "아직 canon이 아닌 runtime story 상태다."}
	if traits != "" {
		facts = append(facts, "초기 설정: "+traits)
	}
	openThreads := []string{"새 환자와 기존 환자 중 누구를 먼저 살릴지 결정", "병동의 부족한 자원을 어떻게 배분할지 판단", "공공 수선과 행정 장부를 어떻게 맞출지 정리"}
	risks := []string{"어느 쪽을 선택해도 다른 쪽의 상태가 악화될 수 있다.", "과로와 저안개 절차 때문에 판단 여력이 흔들릴 수 있다.", "장부상 허가가 늦어지면 회복 공공재 배분이 밀릴 수 있다."}
	choices := []storyChoice{{ID: "A", Text: "새로 쓰러진 환자의 상태를 직접 확인한다.", RiskHint: "즉시 위험을 볼 수 있지만 기존 처치가 더 밀린다."}, {ID: "B", Text: "기존 아이 환자의 처치를 먼저 이어간다.", RiskHint: "약속된 처치를 지키지만 새 환자를 놓칠 수 있다."}, {ID: "C", Text: "보호자에게 짧게 설명하고 동료를 호출한다.", RiskHint: "시간을 벌 수 있지만 항의가 커질 수 있다."}, {ID: "D", Text: "기록판과 펜으로 우선순위를 빠르게 다시 계산한다.", RiskHint: "근거는 남지만 현장 반응이 늦어진다."}}
	return location, scene, summary, facts, openThreads, risks, choices
}

func luceraWorldContextSeed() string {
	return "루멘 연방의 루세라는 병원·약·수면의 도시다. 병동 불빛은 회복 공공재이지만 저안개 차단, 낮은 절차, 공공 수선, 자원 배분, 그리고 보험과 행정 장부의 마찰 속에서만 유지된다."
}
