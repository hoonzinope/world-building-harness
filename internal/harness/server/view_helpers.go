package server

import (
	"fmt"
	"strings"
	"time"
)

func canDriveStory(m storyManifest, u *authUser) bool {
	return u != nil && m.Status == "active" && m.Phase == "waiting_for_choice" && (u.Role == "admin" || u.ID == m.ActiveDriverID)
}

func canQuestionStory(m storyManifest) bool {
	return (m.Status == "active" || m.Status == "paused") && m.Phase == "waiting_for_choice"
}

func friendlyDriverLabel(m storyManifest, u *authUser) string {
	if m.ActiveDriverID == "" {
		return "비어 있음"
	}
	if u != nil && u.ID == m.ActiveDriverID {
		return "나"
	}
	if u != nil && u.Role == "admin" {
		return m.ActiveDriverID
	}
	return m.ActiveDriverID
}

func friendlyUserLabel(u *authUser, fallback string) string {
	if u == nil {
		return fallback
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.Username != "" {
		return u.Username
	}
	return fallback
}

func friendlyPermissionLabel(m storyManifest, u *authUser) string {
	switch {
	case m.Status == "completed" || m.Status == "archived" || m.Status == "deleted":
		return "종료"
	case canDriveStory(m, u):
		return "참여 가능"
	case u != nil && canQuestionStory(m):
		return "질문 가능"
	default:
		return "읽기 전용"
	}
}

func storyLobbyMetaLabels(m storyManifest, u *authUser) []string {
	labels := make([]string, 0, 3)
	if m.SourceDraftPath != "" {
		labels = append(labels, "가져온 스토리")
	}
	if u != nil && (m.CreatedBy == u.ID || m.ActiveDriverID == u.ID) {
		labels = append(labels, "내 스토리")
	}
	if (m.Status == "active" || m.Status == "paused") && !canDriveStory(m, u) {
		labels = append(labels, "관전")
	}
	return labels
}

func storyLobbyMetaLine(m storyManifest, u *authUser) string {
	parts := append([]string{}, storyLobbyMetaLabels(m, u)...)
	parts = append(parts, fmt.Sprintf("턴 %d", m.CurrentTurn))
	parts = append(parts, "진행자 "+friendlyDriverLabel(m, u))
	return strings.Join(parts, " · ")
}

func storyLobbyPhaseLabel(phase string) string {
	if phase == "waiting_for_choice" {
		return "응답 대기"
	}
	return friendlyStoryPhaseLabel(phase)
}

func storyLobbyUpdatedAt(updatedAt string) string {
	return storyTimestampKST(updatedAt, "업데이트 시각 확인 불가")
}

func storyTimestampKST(timestamp, fallback string) string {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return fallback
	}
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return fallback
	}
	return parsed.In(time.FixedZone("KST", 9*60*60)).Format("2006.01.02 15:04")
}

func friendlyStoryStatusLabel(status string) string {
	switch status {
	case "active":
		return "진행 중"
	case "paused":
		return "일시 정지"
	case "completed":
		return "완료"
	case "archived":
		return "보관됨"
	case "deleted":
		return "삭제됨"
	case "setup":
		return "준비 중"
	default:
		return status
	}
}

func friendlyStoryPhaseLabel(phase string) string {
	switch phase {
	case "waiting_for_choice":
		return "입력 대기"
	case "gm_generating":
		return "GM 생성 중"
	case "validating_output":
		return "검증 중"
	case "applying_turn":
		return "반영 중"
	case "failed_waiting_retry":
		return "실패/복구 대기"
	default:
		return phase
	}
}

func friendlyStoryProgressStepLabel(step string) string {
	switch step {
	case "queued":
		return "대기열"
	case "generating":
		return "생성 중"
	case "applying":
		return "반영 중"
	case "ready":
		return "입력 가능"
	case "failed":
		return "실패"
	default:
		return step
	}
}

func friendlyStoryEventKindLabel(kind string) string {
	switch kind {
	case "setup":
		return "설정"
	case "choice":
		return "선택"
	case "custom":
		return "입력"
	case "question":
		return "질문"
	default:
		return kind
	}
}

func friendlyStatusLabel(m storyManifest) string {
	status := friendlyStoryStatusLabel(m.Status)
	phase := friendlyStoryPhaseLabel(m.Phase)
	if phase == "" {
		return status
	}
	if status == "" {
		return phase
	}
	return phase + " · " + status
}

func storyTurnTitle(turnID int, title, fallback string) string {
	title = strings.TrimSpace(title)
	if title == fmt.Sprintf("Turn %d", turnID) {
		return fallback
	}
	return title
}

func storyMatchesLobbyFilter(row lobbyStoryRow, filter string) bool {
	switch filter {
	case "", "all":
		return true
	case "active":
		return row.IsActive
	case "mine":
		return row.IsMine
	case "watch":
		return row.IsWatch
	case "archived":
		return row.IsArchived
	case "imported":
		return row.Imported
	default:
		return true
	}
}
