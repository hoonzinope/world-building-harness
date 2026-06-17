package story

import "testing"

func TestExtractJSONTextFromFencedOutput(t *testing.T) {
	got, err := extractJSONText("```json\n{\"schema_version\":\"story-question-answer.v1\",\"story_id\":\"story_1\",\"answer\":\"ok\"}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"schema_version":"story-question-answer.v1","story_id":"story_1","answer":"ok"}` {
		t.Fatalf("json = %q", got)
	}
}

func TestExtractJSONTextFromPrefixedOutput(t *testing.T) {
	got, err := extractJSONText("final:\n{\"scene_body\":\"brace in string } is fine\",\"choices\":[{\"id\":\"A\"}]}\nthanks")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"scene_body":"brace in string } is fine","choices":[{"id":"A"}]}` {
		t.Fatalf("json = %q", got)
	}
}

func TestExtractJSONTextSkipsStrayLeadingBrace(t *testing.T) {
	got, err := extractJSONText("note {not-json} before payload\n{\"scene_body\":\"ok\",\"choices\":[{\"id\":\"A\"}]}")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"scene_body":"ok","choices":[{"id":"A"}]}` {
		t.Fatalf("json = %q", got)
	}
}
