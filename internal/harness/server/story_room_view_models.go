package server

import "strings"

const (
	storyRoomDossierItemLimit = 6
	storyRoomCharacterLimit   = 8
	storyRoomTurnExcerptLimit = 180
)

type storyRoomStoryView struct {
	ID          string
	Title       string
	Status      string
	Phase       string
	CurrentTurn int
}

type storyRoomPlayerDossierView struct {
	Location                  string
	ActiveCharacters          []string
	ActiveCharactersRemaining int
	Facts                     []string
	FactsRemaining            int
	HiddenFactCount           int
	OpenThreads               []string
	OpenThreadsRemaining      int
	HiddenThreadCount         int
	Risks                     []string
	RisksRemaining            int
	HiddenRiskCount           int
}

type storyRoomTurnView struct {
	TurnID                 int
	SceneTitle             string
	SceneBody              string
	CurrentSituation       string
	RevealedFacts          []string
	RevealedFactsRemaining int
	Choices                []storyChoice
	CreatedAt              string
	Source                 string
}

type storyRoomTurnPreview struct {
	TurnID           int
	SceneTitle       string
	Title            string
	CreatedAt        string
	Timestamp        string
	Source           string
	SourceLabel      string
	CurrentSituation string
	Excerpt          string

	// Legacy fields keep the current templates rendering while carrying preview data only.
	SceneBody     string
	RevealedFacts []string
	Choices       []storyChoice
}

type storyRoomFailedJobNotice struct {
	CanRecover bool
	ActorLabel string
}

type storyRoomAdminData struct {
	Story         storyManifest
	State         storyState
	Turns         []storyTurn
	PreviousTurns []storyTurn
	LatestTurn    any
	QA            []storyQuestion
	FailedJob     *failedJobView
}

func storyRoomStoryViewFromManifest(m storyManifest) storyRoomStoryView {
	return storyRoomStoryView{
		ID:          m.ID,
		Title:       m.Title,
		Status:      m.Status,
		Phase:       m.Phase,
		CurrentTurn: m.CurrentTurn,
	}
}

func storyRoomPlayerDossierViewFromState(st storyState) storyRoomPlayerDossierView {
	activeCharacters, activeRemaining := storyRoomVisibleItems(st.ActiveCharacters, storyRoomCharacterLimit)
	facts, factsRemaining := storyRoomVisibleItems(st.Facts, storyRoomDossierItemLimit)
	openThreads, openThreadsRemaining := storyRoomVisibleItems(st.OpenThreads, storyRoomDossierItemLimit)
	risks, risksRemaining := storyRoomVisibleItems(st.Risks, storyRoomDossierItemLimit)
	return storyRoomPlayerDossierView{
		Location:                  strings.TrimSpace(st.Location),
		ActiveCharacters:          activeCharacters,
		ActiveCharactersRemaining: activeRemaining,
		Facts:                     facts,
		FactsRemaining:            factsRemaining,
		HiddenFactCount:           factsRemaining,
		OpenThreads:               openThreads,
		OpenThreadsRemaining:      openThreadsRemaining,
		HiddenThreadCount:         openThreadsRemaining,
		Risks:                     risks,
		RisksRemaining:            risksRemaining,
		HiddenRiskCount:           risksRemaining,
	}
}

func storyRoomVisibleTurnViewFromTurn(t storyTurn) storyRoomTurnView {
	revealedFacts, revealedRemaining := storyRoomVisibleItems(t.RevealedFacts, storyRoomDossierItemLimit)
	return storyRoomTurnView{
		TurnID:                 t.TurnID,
		SceneTitle:             strings.TrimSpace(t.SceneTitle),
		SceneBody:              strings.TrimSpace(t.SceneBody),
		CurrentSituation:       strings.TrimSpace(t.CurrentSituation),
		RevealedFacts:          revealedFacts,
		RevealedFactsRemaining: revealedRemaining,
		Choices:                append([]storyChoice(nil), t.Choices...),
		CreatedAt:              t.CreatedAt,
		Source:                 t.Source,
	}
}

func storyRoomVisibleLatestTurn(latestTurn any) any {
	switch t := latestTurn.(type) {
	case nil:
		return nil
	case storyTurn:
		return storyRoomVisibleTurnViewFromTurn(t)
	case *storyTurn:
		if t == nil {
			return nil
		}
		return storyRoomVisibleTurnViewFromTurn(*t)
	default:
		return t
	}
}

func storyRoomTurnPreviews(turns []storyTurn) []storyRoomTurnPreview {
	out := make([]storyRoomTurnPreview, 0, len(turns))
	for _, turn := range turns {
		out = append(out, storyRoomTurnPreviewFromTurn(turn))
	}
	return out
}

func storyRoomTurnPreviewFromTurn(turn storyTurn) storyRoomTurnPreview {
	title := storyTurnTitle(turn.TurnID, turn.SceneTitle, "세션 기록")
	excerpt := storyRoomExcerpt(turn.SceneBody, storyRoomTurnExcerptLimit)
	return storyRoomTurnPreview{
		TurnID:           turn.TurnID,
		SceneTitle:       strings.TrimSpace(turn.SceneTitle),
		Title:            title,
		CreatedAt:        turn.CreatedAt,
		Timestamp:        storyTimestampKST(turn.CreatedAt, "시각 확인 불가"),
		Source:           turn.Source,
		SourceLabel:      friendlyStoryEventKindLabel(turn.Source),
		CurrentSituation: strings.TrimSpace(turn.CurrentSituation),
		Excerpt:          excerpt,
		SceneBody:        excerpt,
	}
}

func storyRoomFailedJobNoticeFromFailedJob(failedJob *failedJobView) *storyRoomFailedJobNotice {
	if failedJob == nil {
		return nil
	}
	return &storyRoomFailedJobNotice{
		CanRecover: failedJob.CanRecover,
		ActorLabel: failedJob.ActorLabel,
	}
}

func storyRoomAdminDataFor(isAdmin bool, m storyManifest, st storyState, displayTurns, previousTurns []storyTurn, qa []storyQuestion, latestTurn any, failedJob *failedJobView) any {
	if !isAdmin {
		return nil
	}
	return &storyRoomAdminData{
		Story:         m,
		State:         st,
		Turns:         append([]storyTurn(nil), displayTurns...),
		PreviousTurns: append([]storyTurn(nil), previousTurns...),
		LatestTurn:    latestTurn,
		QA:            append([]storyQuestion(nil), qa...),
		FailedJob:     failedJob,
	}
}

func storyRoomVisibleItems(items []string, limit int) ([]string, int) {
	if limit <= 0 {
		return nil, 0
	}
	visible := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || storyRoomLooksInternal(item) {
			continue
		}
		visible = append(visible, item)
	}
	if len(visible) <= limit {
		return visible, 0
	}
	return visible[:limit], len(visible) - limit
}

func storyRoomLooksInternal(item string) bool {
	normalized := strings.ToLower(strings.TrimSpace(item))
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"runtime story",
		"runtime_story",
		"canon",
		"job",
		"input",
		"schema",
		"gm",
		"source",
		"internal",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func storyRoomExcerpt(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}
