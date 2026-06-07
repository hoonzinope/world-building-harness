package harness

import (
	"fmt"
	"strings"
)

func generateLocalGMScene(prev storyTurn, st storyState, inputLabel, mode string) string {
	if inputLabel == "" {
		inputLabel = "직접 행동"
	}
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	location := firstNonEmpty(st.Location, "현재 장면")
	prefix := fmt.Sprintf("%s의 선택이 장면에 기록된다", name)
	if mode == "dialogue" {
		prefix = fmt.Sprintf("%s의 말이 주변의 반응을 바꾼다", name)
	} else if mode == "narration" {
		prefix = "장면의 초점이 한 단계 좁아진다"
	}
	anchor := "아직 확정된 단서는 많지 않다."
	if len(st.Facts) > 0 {
		anchor = st.Facts[0]
	}
	thread := "다음에 무엇을 붙잡을지 정해야 한다."
	if len(st.OpenThreads) > 0 {
		thread = st.OpenThreads[0]
	}
	risk := "상황이 쉽게 안정되지 않는다."
	if len(st.Risks) > 0 {
		risk = st.Risks[0]
	}
	return fmt.Sprintf("%s.\n\n입력: %s\n\n%s에서 반응은 즉시 결론으로 이어지지 않는다. 먼저 %s의 현재 상태와 방금 입력이 서로 맞물리는 지점이 드러난다. 확인된 기준은 '%s'이고, 열린 문제는 '%s'이다.\n\n이 진행은 아직 실제 GM worker가 아니라 서버 내장 MVP 판정으로 생성된 runtime 장면이다. 선택은 append-only 이벤트로 보존되며, admin이 export하지 않는 한 world pack의 canon이나 draft를 바꾸지 않는다.\n\n그럼에도 장면 안에서는 압력이 이동했다. %s 이제 플레이어는 이전 Turn %d에서 남은 단서를 기준으로 다음 선택 앞에 선다.", prefix, inputLabel, location, name, anchor, thread, risk, prev.TurnID)
}

func generateNextChoices(turn int, st storyState) []storyChoice {
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	return []storyChoice{{ID: "A", Text: "눈앞의 기록과 사실만 차분히 정리한다.", RiskHint: "안전하지만 느리다."}, {ID: "B", Text: name + "가 가장 불편해하는 지점을 직접 말한다.", RiskHint: "빠르지만 충돌이 커진다."}, {ID: "C", Text: "주변 인물의 반응을 살펴 추가 단서를 확보한다.", RiskHint: "장면 압박이 관계 쪽으로 옮겨간다."}, {ID: "D", Text: fmt.Sprintf("Turn %d의 기록을 정리하고 잠시 관망한다.", turn), RiskHint: "정보는 보존하지만 주도권을 잃을 수 있다."}}
}

func generateCurrentSituation(st storyState) string {
	name := "인물"
	if len(st.ActiveCharacters) > 0 && strings.TrimSpace(st.ActiveCharacters[0]) != "" {
		name = strings.TrimSpace(st.ActiveCharacters[0])
	}
	location := firstNonEmpty(st.Location, "현재 장면")
	return fmt.Sprintf("%s에서 %s의 선택 결과가 다음 압력으로 이어졌다.", location, name)
}

func summarizeQuestionAnswer(question string, st storyState) string {
	if strings.Contains(question, "마라") {
		return "마라 베온은 의료 진정 상태로 안정화 중이며, 대체 우안 동기화율은 하락하기 시작한 것으로 기록되어 있다. 장면 기준으로 장기 안전은 아직 확정되지 않았다."
	}
	if strings.Contains(question, "17-B") || strings.Contains(question, "기록군") {
		return "17-B 기록군은 사망 보상권, 신체 권리 분쟁, 생존 주장 잔류 건이 묶인 불안정군이다. 현재 인물들이 확인한 범위에서는 이 기록군을 열 추가 연결점이 필요하다."
	}
	if len(st.Facts) > 0 {
		return "확인된 정보는 '" + st.Facts[0] + "' 등으로 제한된다. 그 밖의 전개는 현재 장면 기준으로는 확인되지 않았다."
	}
	return "현재 장면에서 확인된 정보가 충분하지 않다. 추론과 canon 사실은 구분해서 다뤄야 한다."
}
