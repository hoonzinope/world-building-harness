package story

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type hermesAPIProvider struct{}

type hermesChatRequest struct {
	Model    string              `json:"model"`
	Messages []hermesChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type hermesChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type hermesChatResponse struct {
	Choices []struct {
		Message hermesChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

func (hermesAPIProvider) Generate(ctx context.Context, req GMRequest) (GMOutput, string, string, string, error) {
	baseURL := strings.TrimRight(firstNonEmpty(os.Getenv("WORLD_HARNESS_HERMES_API_BASE_URL"), "http://127.0.0.1:8642/v1"), "/")
	model := firstNonEmpty(strings.TrimSpace(os.Getenv("WORLD_HARNESS_HERMES_MODEL")), "hermes-agent")
	prompt := buildCodexGMPrompt(req, "embedded context")
	body, _ := json.Marshal(hermesChatRequest{
		Model: model,
		Messages: []hermesChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return GMOutput{}, "", "hermes_api", model, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(os.Getenv("WORLD_HARNESS_HERMES_API_KEY")); key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return GMOutput{}, "", "hermes_api", model, err
	}
	defer resp.Body.Close()
	var parsed hermesChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return GMOutput{}, "", "hermes_api", model, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := resp.Status
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			msg = parsed.Error.Message
		}
		return GMOutput{}, "", "hermes_api", model, fmt.Errorf("hermes api error: %s", msg)
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return GMOutput{}, "", "hermes_api", model, errors.New(parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return GMOutput{}, "", "hermes_api", model, errors.New("hermes api returned no choices")
	}
	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	out, jsonText, err := parseGMOutputJSON(raw)
	if err != nil {
		return GMOutput{}, raw, "hermes_api", model, err
	}
	return out, jsonText, "hermes_api", model, nil
}
