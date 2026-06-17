package story

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type codexCLIProvider struct{}

func (codexCLIProvider) Generate(ctx context.Context, req GMRequest) (GMOutput, string, string, string, error) {
	work := filepath.Join(os.TempDir(), "world-harness-gm-"+req.Job.ID)
	if err := os.MkdirAll(work, 0o700); err != nil {
		return GMOutput{}, "", "", "", err
	}
	contextPath := filepath.Join(work, "context.json")
	b, _ := json.MarshalIndent(req, "", "  ")
	if err := os.WriteFile(contextPath, b, 0o600); err != nil {
		return GMOutput{}, "", "", "", err
	}
	prompt := buildCodexGMPrompt(req, contextPath)
	outputPath := filepath.Join(work, "output.json")
	cmd := exec.CommandContext(ctx, "codex", "exec", "-C", work, "--add-dir", filepath.Join(os.Getenv("WORLD_HARNESS_PACKS_ROOT"), "lumen-federation"), "--sandbox", "read-only", "--skip-git-repo-check", "--ephemeral", "--output-last-message", outputPath, prompt)
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"CODEX_HOME=" + firstNonEmpty(os.Getenv("CODEX_HOME"), filepath.Join(os.Getenv("HOME"), ".codex")),
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"WORLD_HARNESS_PACKS_ROOT=" + os.Getenv("WORLD_HARNESS_PACKS_ROOT"),
		"PATH=" + os.Getenv("PATH"),
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return GMOutput{}, stdout.String(), "codex_cli", "", fmt.Errorf("%w: %s", err, tail(stderr.String(), 1200))
	}
	rawBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return GMOutput{}, stdout.String(), "codex_cli", "", err
	}
	raw := strings.TrimSpace(string(rawBytes))
	var out GMOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return GMOutput{}, raw, "codex_cli", "", err
	}
	return out, raw, "codex_cli", "codex-cli", nil
}

func buildCodexGMPrompt(req GMRequest, contextPath string) string {
	contextJSON, _ := json.MarshalIndent(req, "", "  ")
	if req.Job.JobType == "question_answer" {
		return fmt.Sprintf(`You are the question-answer GM worker for world-harness Story Web UI.

Use the embedded context below as authoritative. A copy also exists at %s. Answer the user's non-progressing story question using only current story state, recent turns, the world context seed, and read-only world documents under the added pack directory if needed.

World context seed:
%s

Embedded context JSON:
%s

Do not advance the story. Do not change choices, state, summary, canon, or files.

Keep Lucera / Lumen Federation specifics visible when they are already established in the world seed: public recovery, civic repair, low-intensity procedures, resource allocation, and admin/ledger friction.

Return exactly one JSON object. Do not use Markdown fences. Do not include explanations outside JSON.

Required schema:
{
  "schema_version": "story-question-answer.v1",
  "story_id": %q,
  "answer": "Korean answer, concise but useful"
}`, contextPath, req.WorldContext, string(contextJSON), req.Job.StoryID)
	}
	inputID := ""
	source := outputSourceForJob(req.Job)
	if req.Job.Input != nil {
		inputID = req.Job.Input.ID
	} else {
		inputID = "setup_" + req.Job.ID
	}
	inputSummary := summarizeGMInput(req)
	return fmt.Sprintf(`You are the GM worker for world-harness Story Web UI.

Use the embedded context below as authoritative. A copy also exists at %s, but do not say context is missing when the embedded context is present.

World context seed:
%s

Use the world context seed as a required story anchor, not optional flavor. For prologues and subsequent turns, keep Lucera / Lumen Federation specifics visible when the story lives there: public recovery, civic repair, low-intensity procedures, low-fog operations, resource allocation, and admin/ledger friction.

Current player input:
%s

Embedded context JSON:
%s

Return exactly one JSON object. Do not use Markdown fences. Do not include explanations outside JSON.

Required JSON shape. Keep scene_goal, conflict, turning_point, consequence, scene_title, scene_body, current_situation, revealed_facts, state_patch, resolution, and choices as top-level fields, not inside turn:
{
  "schema_version": "story-gm-output.v1",
  "story_id": %q,
  "turn": {
    "branch_id": "branch_main",
    "turn_id": %d,
    "parent_turn_id": %d,
    "input_id": %q,
    "job_id": %q,
    "source": %q
  },
  "scene_goal": "장면의 즉시 목표",
  "conflict": "장면에서 막히는 핵심 갈등",
  "turning_point": "전환점이 되는 사건",
  "consequence": "이 장면의 비용이나 후과",
  "scene_title": "Korean title",
  "scene_body": "Korean literary prose",
  "current_situation": "Korean current situation",
  "revealed_facts": ["Korean fact"],
  "state_patch": {
    "location_set": "",
    "active_characters_set": [],
    "facts_add": [],
    "facts_remove": [],
    "open_threads_add": [],
    "open_threads_resolve": [],
    "risks_add": [],
    "risks_remove": [],
    "flags_add": [],
    "flags_remove": [],
    "summary_patch": ""
  },
  "resolution": "accepted",
  "choices": [
    {"id": "A", "text": "Korean choice", "intent": "Korean intent", "risk_hint": "Korean risk"}
  ]
}

Rules:
- Continue directly from the latest recent_turns entry.
- If selected_choice_id is set, resolve it against the latest turn choices and depict that choice.
- Never write that prior context is unavailable if recent_turns is non-empty.
- Do not expose job_id, input_id, schema details, or implementation metadata as revealed facts.
- Make the scene goal, conflict, turning point, and consequence explicit in separate fields and keep them aligned with the prose.
- Ensure every choice has a distinct intent and a meaningful risk hint; do not leave them generic or empty.
- Use the state patch to carry forward concrete state updates, not only prose summary changes.
- Validate your final answer as complete JSON before returning it.
- The output should be interactive literary Korean prose, 1500-3000 Korean characters when possible. Do not change canon or files.`, contextPath, req.WorldContext, inputSummary, string(contextJSON), req.Job.StoryID, req.Job.TurnID, req.Job.ParentTurnID, inputID, req.Job.ID, source)
}

func outputSourceForJob(job GMJob) string {
	if job.JobType == "prologue" {
		return "setup"
	}
	if job.Input != nil && job.Input.SelectedChoiceID != "" {
		return "choice"
	}
	return "custom"
}

func StoryWorldContextSeedForRequest(m Manifest, st State, job GMJob) string {
	if m.WorldID == "lumen-federation" || job.JobType == "prologue" || strings.Contains(strings.ToLower(st.Location), "루세라") {
		return luceraWorldContextSeed()
	}
	return ""
}

func summarizeGMInput(req GMRequest) string {
	if req.Job.Input == nil {
		if req.Job.Setup != nil {
			return fmt.Sprintf("프롤로그 생성: 이름=%s, 스타일=%s, 특징=%s", req.Job.Setup.CharacterName, req.Job.Setup.Style, req.Job.Setup.Traits)
		}
		return "입력 없음"
	}
	if req.Job.Input.SelectedChoiceID != "" {
		chosen := ""
		if len(req.Turns) > 0 {
			for _, c := range req.Turns[len(req.Turns)-1].Choices {
				if c.ID == req.Job.Input.SelectedChoiceID {
					chosen = c.Text
					break
				}
			}
		}
		return fmt.Sprintf("선택지 %s: %s", req.Job.Input.SelectedChoiceID, chosen)
	}
	return fmt.Sprintf("%s: %s", req.Job.Input.CustomInputMode, req.Job.Input.CustomText)
}
