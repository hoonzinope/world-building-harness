package story

import (
	"context"
	"strings"
)

type GMJob struct {
	ID                   string    `json:"id"`
	StoryID              string    `json:"story_id"`
	JobType              string    `json:"job_type"`
	Status               string    `json:"status"`
	Attempt              int       `json:"attempt"`
	ActorID              string    `json:"actor_id"`
	ActorRole            string    `json:"actor_role"`
	Input                *Input    `json:"input,omitempty"`
	Setup                *Setup    `json:"setup,omitempty"`
	Question             *Question `json:"question,omitempty"`
	TurnID               int       `json:"turn_id"`
	ParentTurnID         int       `json:"parent_turn_id"`
	ContextHash          string    `json:"context_hash"`
	ErrorCode            string    `json:"error_code,omitempty"`
	ErrorMessage         string    `json:"error_message,omitempty"`
	Provider             string    `json:"provider,omitempty"`
	Model                string    `json:"model,omitempty"`
	CreatedAt            string    `json:"created_at"`
	StartedAt            string    `json:"started_at,omitempty"`
	CompletedAt          string    `json:"completed_at,omitempty"`
	RawOutputPath        string    `json:"raw_output_path,omitempty"`
	IdempotencyKey       string    `json:"idempotency_key,omitempty"`
	ExclusiveProgression bool      `json:"exclusive_progression"`
}

type Input struct {
	ID               string `json:"id"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
	CustomText       string `json:"custom_text,omitempty"`
}

type Setup struct {
	Title         string `json:"title"`
	Style         string `json:"style"`
	CharacterName string `json:"character_name"`
	Traits        string `json:"traits"`
}

type GMRequest struct {
	Job          GMJob    `json:"job"`
	Manifest     Manifest `json:"manifest"`
	State        State    `json:"state"`
	Turns        []Turn   `json:"recent_turns"`
	WorldContext string   `json:"world_context,omitempty"`
}

type GMOutput struct {
	SchemaVersion    string              `json:"schema_version"`
	StoryID          string              `json:"story_id"`
	Turn             GMOutputTurn        `json:"turn"`
	SceneGoal        string              `json:"scene_goal,omitempty"`
	Conflict         string              `json:"conflict,omitempty"`
	TurningPoint     string              `json:"turning_point,omitempty"`
	Consequence      string              `json:"consequence,omitempty"`
	SceneTitle       string              `json:"scene_title"`
	SceneBody        string              `json:"scene_body"`
	Answer           string              `json:"answer,omitempty"`
	CurrentSituation string              `json:"current_situation"`
	RevealedFacts    []string            `json:"revealed_facts"`
	StatePatch       GMStatePatch        `json:"state_patch"`
	Resolution       string              `json:"resolution"`
	Choices          []Choice            `json:"choices"`
	GMNotes          map[string][]string `json:"gm_notes,omitempty"`
}

type GMOutputTurn struct {
	BranchID         string `json:"branch_id"`
	TurnID           int    `json:"turn_id"`
	ParentTurnID     int    `json:"parent_turn_id"`
	InputID          string `json:"input_id"`
	JobID            string `json:"job_id"`
	Source           string `json:"source"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
}

type GMStatePatch struct {
	LocationSet         string   `json:"location_set,omitempty"`
	ActiveCharactersSet []string `json:"active_characters_set,omitempty"`
	FactsAdd            []string `json:"facts_add,omitempty"`
	FactsRemove         []string `json:"facts_remove,omitempty"`
	OpenThreadsAdd      []string `json:"open_threads_add,omitempty"`
	OpenThreadsResolve  []string `json:"open_threads_resolve,omitempty"`
	RisksAdd            []string `json:"risks_add,omitempty"`
	RisksRemove         []string `json:"risks_remove,omitempty"`
	FlagsAdd            []string `json:"flags_add,omitempty"`
	FlagsRemove         []string `json:"flags_remove,omitempty"`
	SummaryPatch        string   `json:"summary_patch,omitempty"`
}

type GMProvider interface {
	Generate(context.Context, GMRequest) (GMOutput, string, string, string, error)
}

func newGMProvider(name string) GMProvider {
	switch strings.TrimSpace(name) {
	case "codex_cli":
		return codexCLIProvider{}
	case "hermes_api":
		return hermesAPIProvider{}
	default:
		return mockGMProvider{}
	}
}
