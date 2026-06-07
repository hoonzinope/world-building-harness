package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func validateGMOutput(job gmJob, out gmOutput) error {
	if out.SchemaVersion != "story-gm-output.v1" {
		return errors.New("schema_version mismatch")
	}
	expectedInputID := out.Turn.InputID
	if job.Input != nil {
		expectedInputID = job.Input.ID
	}
	if out.StoryID != job.StoryID || out.Turn.TurnID != job.TurnID || out.Turn.ParentTurnID != job.ParentTurnID || out.Turn.InputID != expectedInputID || out.Turn.JobID != job.ID {
		return errors.New("lineage mismatch")
	}
	if strings.TrimSpace(out.SceneBody) == "" || strings.TrimSpace(out.SceneTitle) == "" || strings.TrimSpace(out.CurrentSituation) == "" {
		return errors.New("empty required scene field")
	}
	if len(out.SceneBody) < 200 {
		return errors.New("scene_body too short")
	}
	blocked := []string{"맥락을 확인할 수 없", "이전 장면의 구체적 내용이 확인되지", "input_id", "job_id", "작업 ID", "입력 ID"}
	joined := out.SceneBody + "\n" + out.CurrentSituation + "\n" + strings.Join(out.RevealedFacts, "\n")
	for _, term := range blocked {
		if strings.Contains(joined, term) {
			return fmt.Errorf("output contains invalid fallback/meta text: %s", term)
		}
	}
	if len(out.Choices) < 3 || len(out.Choices) > 4 {
		return errors.New("choices must be 3 or 4")
	}
	seen := map[string]bool{}
	for _, c := range out.Choices {
		if c.ID == "" || c.Text == "" || seen[c.ID] {
			return errors.New("invalid choice")
		}
		seen[c.ID] = true
	}
	return nil
}

func applyGMStatePatch(st storyState, p gmStatePatch, revealed []string) storyState {
	if p.LocationSet != "" {
		st.Location = p.LocationSet
	}
	if len(p.ActiveCharactersSet) > 0 {
		st.ActiveCharacters = p.ActiveCharactersSet
	}
	st.Facts = removeStrings(appendUnique(st.Facts, revealed...), p.FactsRemove...)
	st.Facts = appendUnique(st.Facts, p.FactsAdd...)
	st.OpenThreads = removeStrings(st.OpenThreads, p.OpenThreadsResolve...)
	st.OpenThreads = appendUnique(st.OpenThreads, p.OpenThreadsAdd...)
	st.Risks = removeStrings(st.Risks, p.RisksRemove...)
	st.Risks = appendUnique(st.Risks, p.RisksAdd...)
	st.Flags = removeStrings(st.Flags, p.FlagsRemove...)
	st.Flags = appendUnique(st.Flags, p.FlagsAdd...)
	return st
}

func removeStrings(in []string, vals ...string) []string {
	remove := map[string]bool{}
	for _, v := range vals {
		remove[strings.TrimSpace(v)] = true
	}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !remove[strings.TrimSpace(v)] {
			out = append(out, v)
		}
	}
	return out
}

func rewriteQuestionProjection(path string, updated storyQuestion) error {
	var qs []storyQuestion
	_ = readJSONL(path, func(b []byte) error {
		var q storyQuestion
		if err := json.Unmarshal(b, &q); err != nil {
			return err
		}
		if q.ID == updated.ID {
			q = updated
		}
		qs = append(qs, q)
		return nil
	})
	found := false
	for i := range qs {
		if qs[i].ID == updated.ID {
			qs[i] = updated
			found = true
		}
	}
	if !found {
		qs = append(qs, updated)
	}
	var b strings.Builder
	for _, q := range qs {
		line, _ := json.Marshal(q)
		b.Write(line)
		b.WriteByte('\n')
	}
	return writeAtomic(path, []byte(b.String()))
}
