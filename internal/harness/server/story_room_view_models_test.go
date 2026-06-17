package server

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestStoryRoomTemplateDataBuildsFilteredPlayerDossier(t *testing.T) {
	st := storyState{
		Location:         "  루세라 야간 진료동  ",
		ActiveCharacters: []string{"르네", "GM overseer", "루세라 기록관"},
		Facts: append(
			numberedStoryRoomItems("공개 사실", storyRoomDossierItemLimit+2),
			"runtime story 내부 메모",
			"아직 canon으로 승격되지 않은 항목",
			"job payload 상태",
		),
		OpenThreads: append(
			numberedStoryRoomItems("공개 실마리", storyRoomDossierItemLimit+1),
			"source draft 검토",
		),
		Risks: []string{
			"공개 위험 1",
			"schema validation failure",
			"public input queue detail",
			"공개 위험 2",
		},
		Flags: []string{"runtime_story_created"},
	}
	latestTurn := storyTurn{
		TurnID:        3,
		SceneTitle:    "최신 턴",
		SceneBody:     "최신 장면 본문",
		RevealedFacts: []string{"플레이어에게 보여도 되는 사실", "GM job 내부 처리"},
		CreatedAt:     "2026-06-07T00:00:00Z",
		Source:        "choice",
	}

	data := testStoryRoomTemplateData(st, nil, nil, false, latestTurn)
	dossier, ok := data["PlayerDossier"].(storyRoomPlayerDossierView)
	if !ok {
		t.Fatalf("PlayerDossier type = %T", data["PlayerDossier"])
	}
	stateDossier, ok := data["State"].(storyRoomPlayerDossierView)
	if !ok {
		t.Fatalf("State should be visible dossier data, got %T", data["State"])
	}
	if fmt.Sprintf("%#v", dossier) != fmt.Sprintf("%#v", stateDossier) {
		t.Fatalf("State compatibility view diverged from PlayerDossier: %#v vs %#v", stateDossier, dossier)
	}
	if dossier.Location != "루세라 야간 진료동" {
		t.Fatalf("location = %q", dossier.Location)
	}
	assertNoInternalStoryRoomItems(t, "facts", dossier.Facts)
	assertNoInternalStoryRoomItems(t, "open threads", dossier.OpenThreads)
	assertNoInternalStoryRoomItems(t, "risks", dossier.Risks)
	assertNoInternalStoryRoomItems(t, "active characters", dossier.ActiveCharacters)
	if len(dossier.Facts) != storyRoomDossierItemLimit || dossier.FactsRemaining != 2 {
		t.Fatalf("facts limit/remain = %d/%d, facts=%#v", len(dossier.Facts), dossier.FactsRemaining, dossier.Facts)
	}
	if len(dossier.OpenThreads) != storyRoomDossierItemLimit || dossier.OpenThreadsRemaining != 1 {
		t.Fatalf("open thread limit/remain = %d/%d, threads=%#v", len(dossier.OpenThreads), dossier.OpenThreadsRemaining, dossier.OpenThreads)
	}
	if len(dossier.Risks) != 2 || dossier.RisksRemaining != 0 {
		t.Fatalf("risks limit/remain = %d/%d, risks=%#v", len(dossier.Risks), dossier.RisksRemaining, dossier.Risks)
	}
	if data["AdminData"] != nil {
		t.Fatalf("non-admin template data should not include raw admin data: %#v", data["AdminData"])
	}
	visibleTurn, ok := data["LatestTurn"].(storyRoomTurnView)
	if !ok {
		t.Fatalf("LatestTurn type = %T", data["LatestTurn"])
	}
	if len(visibleTurn.RevealedFacts) != 1 || visibleTurn.RevealedFacts[0] != "플레이어에게 보여도 되는 사실" {
		t.Fatalf("revealed facts were not filtered: %#v", visibleTurn.RevealedFacts)
	}

	adminData := testStoryRoomTemplateData(st, nil, nil, true, latestTurn)
	admin, ok := adminData["AdminData"].(*storyRoomAdminData)
	if !ok || admin == nil {
		t.Fatalf("admin template data missing raw AdminData: %#v", adminData["AdminData"])
	}
	if !strings.Contains(strings.Join(admin.State.Facts, "\n"), "runtime story 내부 메모") {
		t.Fatalf("admin raw state should retain internal facts: %#v", admin.State.Facts)
	}
}

func TestStoryRoomPreviousTurnPreviewsDoNotIncludeFullSceneBody(t *testing.T) {
	sentinel := "FULL_BODY_SENTINEL"
	fullBody := strings.Repeat("이전 장면의 상세 본문 ", 40) + sentinel
	previousTurn := storyTurn{
		TurnID:           2,
		SceneTitle:       "이전 턴",
		SceneBody:        fullBody,
		CurrentSituation: "다음 행동을 고르는 중",
		CreatedAt:        "2026-06-07T00:02:00Z",
		Source:           "choice",
		RevealedFacts:    []string{"이전 턴의 긴 사실"},
		Choices:          []storyChoice{{ID: "A", Text: "선택지"}},
	}

	data := testStoryRoomTemplateData(storyState{}, []storyTurn{previousTurn}, []storyTurn{previousTurn}, false, nil)
	previews, ok := data["PreviousTurnPreviews"].([]storyRoomTurnPreview)
	if !ok {
		t.Fatalf("PreviousTurnPreviews type = %T", data["PreviousTurnPreviews"])
	}
	legacyPreviews, ok := data["PreviousTurns"].([]storyRoomTurnPreview)
	if !ok {
		t.Fatalf("PreviousTurns should be preview data, got %T", data["PreviousTurns"])
	}
	for name, list := range map[string][]storyRoomTurnPreview{
		"PreviousTurnPreviews": previews,
		"PreviousTurns":        legacyPreviews,
	} {
		if len(list) != 1 {
			t.Fatalf("%s len = %d", name, len(list))
		}
		preview := list[0]
		if preview.TurnID != previousTurn.TurnID || preview.CreatedAt != previousTurn.CreatedAt || preview.Source != previousTurn.Source {
			t.Fatalf("%s identity/meta = %#v", name, preview)
		}
		if preview.CurrentSituation != previousTurn.CurrentSituation {
			t.Fatalf("%s current situation = %q", name, preview.CurrentSituation)
		}
		if preview.Excerpt == "" || preview.SceneBody != preview.Excerpt {
			t.Fatalf("%s excerpt/legacy scene body mismatch: %#v", name, preview)
		}
		if preview.Excerpt == fullBody || preview.SceneBody == fullBody || strings.Contains(fmt.Sprintf("%#v", preview), sentinel) {
			t.Fatalf("%s includes full previous SceneBody: %#v", name, preview)
		}
		if len(preview.RevealedFacts) != 0 || len(preview.Choices) != 0 {
			t.Fatalf("%s should not carry full turn details: %#v", name, preview)
		}
	}

	adminData := testStoryRoomTemplateData(storyState{}, []storyTurn{previousTurn}, []storyTurn{previousTurn}, true, nil)
	admin, ok := adminData["AdminData"].(*storyRoomAdminData)
	if !ok || admin == nil {
		t.Fatalf("missing admin data: %#v", adminData["AdminData"])
	}
	if len(admin.PreviousTurns) != 1 || !strings.Contains(admin.PreviousTurns[0].SceneBody, sentinel) {
		t.Fatalf("admin raw previous turn was not retained separately: %#v", admin.PreviousTurns)
	}
}

func testStoryRoomTemplateData(st storyState, displayTurns, previousTurns []storyTurn, isAdmin bool, latestTurn any) map[string]any {
	m := storyManifest{
		ID:          "story_test",
		Title:       "테스트 이야기",
		Status:      "active",
		Phase:       "waiting_for_choice",
		CurrentTurn: 3,
	}
	return storyRoomTemplateData(
		"/base",
		m.ID,
		&authUser{ID: "user_test", Role: "friend"},
		m,
		st,
		displayTurns,
		previousTurns,
		nil,
		false,
		false,
		false,
		false,
		isAdmin,
		false,
		3,
		latestTurn,
		len(displayTurns) > 0,
		"driver",
		false,
		storyProgressView{},
		nil,
		url.Values{},
		"csrf",
	)
}

func numberedStoryRoomItems(prefix string, count int) []string {
	items := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, fmt.Sprintf("%s %d", prefix, i))
	}
	return items
}

func assertNoInternalStoryRoomItems(t *testing.T, label string, items []string) {
	t.Helper()
	for _, item := range items {
		if storyRoomLooksInternal(item) {
			t.Fatalf("%s contains internal item %q in %#v", label, item, items)
		}
	}
}
