package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func mustCSRFToken(w http.ResponseWriter, r *http.Request) string {
	token, err := ensureCSRFToken(w, r)
	if err != nil {
		return ""
	}
	return token
}

func isJSONStoryTaskRequest(r *http.Request) bool {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	return strings.HasPrefix(contentType, "application/json")
}

func parseStoryTaskRequest(r *http.Request) error {
	if isJSONStoryTaskRequest(r) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return err
		}
		values := url.Values{}
		for key, raw := range payload {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("json field %q must be a string", key)
			}
			values.Set(key, str)
		}
		if r.URL != nil {
			for key, vals := range r.URL.Query() {
				for _, v := range vals {
					values.Add(key, v)
				}
			}
		}
		r.PostForm = values
		r.Form = values
		return nil
	}
	return r.ParseForm()
}

func mustTurnIdempotencyKey() string {
	token, err := randomToken(18)
	if err != nil {
		return randomID()
	}
	return token
}

func parseFormInt(v string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(v))
	return n
}

func queryCSV(v string) []string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func firstNonZero(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}
