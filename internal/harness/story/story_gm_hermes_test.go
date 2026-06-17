package story

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHermesAPIProviderGenerateParsesChatCompletion(t *testing.T) {
	out := GMOutput{
		SchemaVersion: "story-question-answer.v1",
		StoryID:       "story_1",
		Answer:        "테스트 답변",
	}
	content, _ := json.Marshal(out)
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var req hermesChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "hermes-agent-test" || len(req.Messages) != 1 || req.Messages[0].Role != "user" {
			t.Fatalf("request = %#v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": string(content)}}},
		})
	}))
	defer server.Close()
	t.Setenv("WORLD_HARNESS_HERMES_API_BASE_URL", server.URL+"/v1")
	t.Setenv("WORLD_HARNESS_HERMES_API_KEY", "test-key")
	t.Setenv("WORLD_HARNESS_HERMES_MODEL", "hermes-agent-test")
	got, raw, provider, model, err := hermesAPIProvider{}.Generate(context.Background(), GMRequest{
		Job: GMJob{ID: "job_1", StoryID: "story_1", JobType: "question_answer", Question: &Question{Question: "질문"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Answer != out.Answer || provider != "hermes_api" || model != "hermes-agent-test" {
		t.Fatalf("got=%#v raw=%s provider=%s model=%s", got, raw, provider, model)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("auth header = %q", gotAuth)
	}
}

func TestNewGMProviderHermesAPI(t *testing.T) {
	if _, ok := newGMProvider("hermes_api").(hermesAPIProvider); !ok {
		t.Fatalf("expected hermes api provider")
	}
	if _, ok := newGMProvider(" hermes_api ").(hermesAPIProvider); !ok {
		t.Fatalf("expected trimmed hermes api provider")
	}
}

func TestHermesAPIProviderReportsErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "bad key"}})
	}))
	defer server.Close()
	t.Setenv("WORLD_HARNESS_HERMES_API_BASE_URL", server.URL+"/v1")
	t.Setenv("WORLD_HARNESS_HERMES_MODEL", "")
	_, _, provider, model, err := hermesAPIProvider{}.Generate(context.Background(), GMRequest{
		Job: GMJob{ID: "job_1", StoryID: "story_1", JobType: "question_answer", Question: &Question{Question: "질문"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if provider != "hermes_api" || model != "hermes-agent" {
		t.Fatalf("provider=%s model=%s", provider, model)
	}
}
