package story

import "time"

type Store struct {
	root       string
	packsRoot  string
	exportRoot string
}

const StoryLockTimeout = 10 * time.Minute

type Manifest struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	WorldID         string `json:"world_id"`
	Status          string `json:"status"`
	Phase           string `json:"phase"`
	CurrentTurn     int    `json:"current_turn"`
	ActiveDriverID  string `json:"active_driver_id"`
	ActiveJobID     string `json:"active_job_id,omitempty"`
	SourceDraftPath string `json:"source_draft_path,omitempty"`
	SourceHash      string `json:"source_hash,omitempty"`
	CreatedBy       string `json:"created_by"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LatestSummary   string `json:"latest_summary"`
}

type State struct {
	Location         string   `json:"location"`
	ActiveCharacters []string `json:"active_characters"`
	Facts            []string `json:"facts"`
	OpenThreads      []string `json:"open_threads"`
	Risks            []string `json:"risks"`
	Flags            []string `json:"flags"`
}

type Choice struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Intent   string `json:"intent,omitempty"`
	RiskHint string `json:"risk_hint,omitempty"`
}

type Turn struct {
	TurnID           int           `json:"turn_id"`
	BranchID         string        `json:"branch_id"`
	ParentTurnID     int           `json:"parent_turn_id"`
	ActorID          string        `json:"actor_id"`
	InputID          string        `json:"input_id"`
	Source           string        `json:"source"`
	SelectedChoiceID string        `json:"selected_choice_id,omitempty"`
	CustomInputMode  string        `json:"custom_input_mode,omitempty"`
	CustomText       string        `json:"custom_text,omitempty"`
	SceneTitle       string        `json:"scene_title"`
	SceneBody        string        `json:"scene_body"`
	CurrentSituation string        `json:"current_situation"`
	RevealedFacts    []string      `json:"revealed_facts"`
	Choices          []Choice `json:"choices"`
	CreatedAt        string        `json:"created_at"`
}

type Question struct {
	ID        string `json:"id"`
	ActorID   string `json:"actor_id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	TurnID    int    `json:"turn_id"`
	CreatedAt string `json:"created_at"`
}
