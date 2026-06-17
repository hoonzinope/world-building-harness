package story

import (
	"encoding/json"
	"errors"
	"strings"
)

func parseGMOutputJSON(raw string) (GMOutput, string, error) {
	jsonText, err := extractJSONText(raw)
	if err != nil {
		return GMOutput{}, strings.TrimSpace(raw), err
	}
	var out GMOutput
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		return GMOutput{}, jsonText, err
	}
	return out, jsonText, nil
}

func extractJSONText(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("empty provider output")
	}
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
			s = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	if json.Valid([]byte(s)) {
		return s, nil
	}
	var sawObject bool
outer:
	for start := 0; start < len(s); start++ {
		if s[start] != '{' {
			continue
		}
		sawObject = true
		depth := 0
		inString := false
		escaped := false
		for i := start; i < len(s); i++ {
			c := s[i]
			if inString {
				if escaped {
					escaped = false
					continue
				}
				switch c {
				case '\\':
					escaped = true
				case '"':
					inString = false
				}
				continue
			}
			switch c {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(s[start : i+1])
					if json.Valid([]byte(candidate)) {
						return candidate, nil
					}
					continue outer
				}
			}
		}
	}
	if sawObject {
		return "", errors.New("provider output contains invalid or incomplete JSON object")
	}
	return "", errors.New("provider output does not contain JSON object")
}
