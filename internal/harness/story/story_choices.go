package story

import "strings"

func annotateChoiceIntent(choices []Choice) {
	for i := range choices {
		if strings.TrimSpace(choices[i].Intent) == "" {
			choices[i].Intent = choiceIntentForText(choices[i].Text)
		}
		if strings.TrimSpace(choices[i].RiskHint) == "" {
			choices[i].RiskHint = "선택 후속 결과가 즉시 반영된다."
		}
	}
}

func choiceIntentForText(text string) string {
	switch {
	case strings.Contains(text, "확인"):
		return "현장 상태를 먼저 확인해 위험을 줄인다."
	case strings.Contains(text, "이어간다"):
		return "기존 처치를 이어가며 약속을 지킨다."
	case strings.Contains(text, "설명"):
		return "설명을 통해 시간을 벌고 충돌을 완화한다."
	case strings.Contains(text, "계산"):
		return "기록과 근거를 다시 계산해 우선순위를 정한다."
	default:
		return "지금의 상황에 맞춰 다음 행동의 근거를 만든다."
	}
}
