package harness

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

func (s *webServer) render(w http.ResponseWriter, r *http.Request, title, body string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, ok := data["CSRFToken"]; !ok {
		data["CSRFToken"] = mustCSRFToken(w, r)
	}
	data["PageTitle"] = title
	data["StoryEnabled"] = s.storyEnabled
	data["AuthEnabled"] = s.auth != nil
	data["BaseStyles"] = template.CSS(ui.BaseStyles)
	t, err := template.New("page").Funcs(templateFuncMap()).Parse(ui.LayoutTemplate + body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"docURL": func(base, pack, path string) string {
			return fmt.Sprintf("%s/packs/%s/doc?path=%s", base, url.PathEscape(pack), url.QueryEscape(path))
		},
		"packURL": func(base, pack string) string {
			return fmt.Sprintf("%s/packs/%s/", base, url.PathEscape(pack))
		},
		"storyURL": func(base, id string) string {
			return fmt.Sprintf("%s/stories/%s", base, url.PathEscape(id))
		},
		"sceneJournalTitle": func(turnID int, title string) string {
			return storyTurnTitle(turnID, title, "세션 기록")
		},
		"sceneIndexTitle": func(turnID int, title string) string {
			return storyTurnTitle(turnID, title, "")
		},
		"storyTurnTimestamp": func(timestamp string) string {
			return storyTimestampKST(timestamp, "시각 확인 불가")
		},
		"friendlyStoryStatusLabel":       friendlyStoryStatusLabel,
		"friendlyStoryPhaseLabel":        friendlyStoryPhaseLabel,
		"storyLobbyPhaseLabel":           storyLobbyPhaseLabel,
		"friendlyStoryProgressStepLabel": friendlyStoryProgressStepLabel,
		"friendlyStoryEventKindLabel":    friendlyStoryEventKindLabel,
		"idem": func() string {
			return mustTurnIdempotencyKey()
		},
		"eq": func(a, b any) bool {
			return fmt.Sprint(a) == fmt.Sprint(b)
		},
		"not": func(v bool) bool { return !v },
		"nl2br": func(s string) template.HTML {
			return template.HTML(template.HTMLEscapeString(s))
		},
	}
}
