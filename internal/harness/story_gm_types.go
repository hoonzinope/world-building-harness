package harness

import (
	"context"
	"strings"
)

type gmJob struct {
	ID                   string         `json:"id"`
	StoryID              string         `json:"story_id"`
	JobType              string         `json:"job_type"`
	Status               string         `json:"status"`
	Attempt              int            `json:"attempt"`
	ActorID              string         `json:"actor_id"`
	ActorRole            string         `json:"actor_role"`
	Input                *storyInput    `json:"input,omitempty"`
	Setup                *storySetup    `json:"setup,omitempty"`
	Question             *storyQuestion `json:"question,omitempty"`
	TurnID               int            `json:"turn_id"`
	ParentTurnID         int            `json:"parent_turn_id"`
	ContextHash          string         `json:"context_hash"`
	ErrorCode            string         `json:"error_code,omitempty"`
	ErrorMessage         string         `json:"error_message,omitempty"`
	Provider             string         `json:"provider,omitempty"`
	Model                string         `json:"model,omitempty"`
	CreatedAt            string         `json:"created_at"`
	StartedAt            string         `json:"started_at,omitempty"`
	CompletedAt          string         `json:"completed_at,omitempty"`
	RawOutputPath        string         `json:"raw_output_path,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	ExclusiveProgression bool           `json:"exclusive_progression"`
}

type storyInput struct {
	ID               string `json:"id"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
	CustomText       string `json:"custom_text,omitempty"`
}

type storySetup struct {
	Title         string `json:"title"`
	Style         string `json:"style"`
	CharacterName string `json:"character_name"`
	Traits        string `json:"traits"`
}

type gmRequest struct {
	Job          gmJob         `json:"job"`
	Manifest     storyManifest `json:"manifest"`
	State        storyState    `json:"state"`
	Turns        []storyTurn   `json:"recent_turns"`
	WorldContext string        `json:"world_context,omitempty"`
}

type gmOutput struct {
	SchemaVersion    string              `json:"schema_version"`
	StoryID          string              `json:"story_id"`
	Turn             gmOutputTurn        `json:"turn"`
	SceneTitle       string              `json:"scene_title"`
	SceneBody        string              `json:"scene_body"`
	Answer           string              `json:"answer,omitempty"`
	CurrentSituation string              `json:"current_situation"`
	RevealedFacts    []string            `json:"revealed_facts"`
	StatePatch       gmStatePatch        `json:"state_patch"`
	Resolution       string              `json:"resolution"`
	Choices          []storyChoice       `json:"choices"`
	GMNotes          map[string][]string `json:"gm_notes,omitempty"`
}

type gmOutputTurn struct {
	BranchID         string `json:"branch_id"`
	TurnID           int    `json:"turn_id"`
	ParentTurnID     int    `json:"parent_turn_id"`
	InputID          string `json:"input_id"`
	JobID            string `json:"job_id"`
	Source           string `json:"source"`
	SelectedChoiceID string `json:"selected_choice_id,omitempty"`
	CustomInputMode  string `json:"custom_input_mode,omitempty"`
}

type gmStatePatch struct {
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

type gmProvider interface {
	Generate(context.Context, gmRequest) (gmOutput, string, string, string, error)
}

func newGMProvider(name string) gmProvider {
	switch strings.TrimSpace(name) {
	case "codex_cli":
		return codexCLIProvider{}
	default:
		return mockGMProvider{}
	}
}
