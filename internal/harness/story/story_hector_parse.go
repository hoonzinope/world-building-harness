package story

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type HectorParsed struct {
	Title     string
	TurnID    int
	SceneBody string
	Facts     []string
	Choices   []Choice
	Turns     []Turn
}

func ParseHectorDraft(body string) (HectorParsed, error) {
	var p HectorParsed
	if strings.HasPrefix(body, "---") {
		if end := strings.Index(body[3:], "---"); end >= 0 {
			fm := body[3 : 3+end]
			for _, line := range strings.Split(fm, "\n") {
				if strings.HasPrefix(line, "title:") {
					p.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "title:")), "'\"")
				}
			}
		}
	}
	re := regexp.MustCompile(`(?m)^## Turn ([0-9]+)\s*$`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return p, errors.New("no turn heading")
	}
	last := matches[len(matches)-1]
	n, _ := strconv.Atoi(body[last[2]:last[3]])
	p.TurnID = n
	start := last[1]
	section := body[start:]
	p.SceneBody = sectionBetween(section, "### 판정", "### 확인된 정보")
	facts := sectionBetween(section, "### 확인된 정보", "### 다음 갈림길")
	for _, line := range strings.Split(facts, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			p.Facts = append(p.Facts, line)
		}
	}
	choices := sectionBetween(section, "### 다음 갈림길", "")
	p.Choices = extractChoices(choices)
	p.Turns = parseHectorTurns(body)
	return p, nil
}

func parseHectorTurns(body string) []Turn {
	re := regexp.MustCompile(`(?m)^## Turn ([0-9]+)\s*$`)
	matches := re.FindAllStringSubmatchIndex(body, -1)
	var turns []Turn
	for i, m := range matches {
		id, _ := strconv.Atoi(body[m[2]:m[3]])
		start := m[1]
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		turns = append(turns, parseHectorTurnSection(id, strings.TrimSpace(body[start:end])))
	}
	return turns
}

func parseHectorTurnSection(id int, section string) Turn {
	facts := extractListSection(section, "### 확인된 정보", "### 다음 갈림길")
	choices := extractChoices(firstNonEmpty(sectionBetween(section, "### 다음 갈림길", ""), sectionBetween(section, "### 선택지", "### 선택")))
	scene := strings.TrimSpace(section)
	if block := sectionBetween(section, "### 확인된 정보", "### 다음 갈림길"); block != "" {
		scene = strings.Replace(scene, "### 확인된 정보\n"+block, "", 1)
	}
	if block := sectionBetween(section, "### 다음 갈림길", ""); block != "" {
		scene = strings.Replace(scene, "### 다음 갈림길\n"+block, "", 1)
	}
	situation := firstNonEmpty(sectionBetween(section, "### 현재 결과", ""), sectionBetween(section, "### 상황", "### 선택지"), hectorCurrentSituation())
	return Turn{TurnID: id, SceneTitle: fmt.Sprintf("Turn %d", id), SceneBody: strings.TrimSpace(scene), CurrentSituation: situation, RevealedFacts: facts, Choices: choices}
}

func extractListSection(body, startHeading, endHeading string) []string {
	block := sectionBetween(body, startHeading, endHeading)
	var out []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func extractChoices(block string) []Choice {
	var out []Choice
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		if len(line) > 3 && line[1] == '.' {
			id := line[:1]
			if id >= "A" && id <= "D" {
				out = append(out, Choice{ID: id, Text: strings.TrimSpace(line[2:])})
			}
		}
	}
	return out
}

func sectionBetween(body, startHeading, endHeading string) string {
	start := strings.Index(body, startHeading)
	if start < 0 {
		return ""
	}
	start += len(startHeading)
	rest := strings.TrimSpace(body[start:])
	if endHeading != "" {
		if end := strings.Index(rest, endHeading); end >= 0 {
			rest = rest[:end]
		}
	}
	return strings.TrimSpace(rest)
}
