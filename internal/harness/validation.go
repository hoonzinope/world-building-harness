package harness

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

var allowedTypes = map[string]bool{
	"character": true, "nation": true, "organization": true, "place": true,
	"event": true, "timeline": true, "magic": true, "glossary": true, "storylet": true,
}

var allowedStatus = map[string]bool{
	"draft": true, "canon": true, "deprecated": true, "rejected": true,
	"public-review": true,
}

var relationshipTypes = map[string]struct {
	domain map[string]bool
	rng    map[string]bool
}{
	"member_of":       {set("character", "organization"), set("organization", "nation")},
	"affiliated_with": {set("character", "organization"), set("organization", "nation")},
	"located_in":      {set("place", "organization"), set("place", "nation")},
	"capital_of":      {set("place"), set("nation")},
	"headquarters_of": {set("place"), set("organization")},
	"rules":           {set("character", "organization"), set("nation", "place")},
	"parent_of":       {set("character"), set("character")},
	"child_of":        {set("character"), set("character")},
	"sibling_of":      {set("character"), set("character")},
	"ally_of":         {set("character", "nation", "organization"), set("character", "nation", "organization")},
	"rival_of":        {set("character", "nation", "organization"), set("character", "nation", "organization")},
	"predecessor_of":  {set("nation", "organization", "event"), set("nation", "organization", "event")},
	"successor_of":    {set("nation", "organization", "event"), set("nation", "organization", "event")},
	"participates_in": {set("character", "organization", "nation"), set("event")},
	"occurred_at":     {set("event"), set("place")},
	"uses_magic":      {set("character", "organization", "nation"), set("magic")},
}

func set(values ...string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[v] = true
	}
	return out
}

func validateDocument(ctx *WorldContext, doc Document, acceptMode bool) (string, []Issue) {
	issues := []Issue{}
	required := []string{"schema_version", "id", "type", "status", "title", "created_at", "updated_at"}
	for _, key := range required {
		if strings.TrimSpace(metaString(doc.Meta, key)) == "" {
			severity := "error"
			if key == "schema_version" && strings.HasPrefix(doc.Path, "content/") {
				severity = "warning"
			}
			issues = append(issues, Issue{Code: "MISSING_FIELD", Rule: "VR-002", Severity: severity, Message: "required frontmatter field is missing", Path: doc.Path, Field: key})
		}
	}
	id := doc.ID()
	docType := doc.Type()
	status := doc.Status()
	if id != "" && !idPattern.MatchString(id) {
		issues = append(issues, Issue{Code: "INVALID_ID", Rule: "VR-102", Severity: "error", Message: "id must use lowercase letters, numbers, and underscores", Path: doc.Path, Field: "id"})
	}
	if docType != "" && !allowedTypes[docType] {
		issues = append(issues, Issue{Code: "INVALID_TYPE", Rule: "VR-003", Severity: "error", Message: "unsupported document type", Path: doc.Path, Field: "type"})
	}
	if status != "" && !allowedStatus[status] {
		issues = append(issues, Issue{Code: "INVALID_STATUS", Rule: "VR-004", Severity: "error", Message: "unsupported document status", Path: doc.Path, Field: "status"})
	}
	if strings.HasPrefix(doc.Path, "drafts/") {
		expected, ok := draftDirs[docType]
		if ok && !strings.HasPrefix(doc.Path, filepath.ToSlash(filepath.Join("drafts", expected))+"/") {
			issues = append(issues, Issue{Code: "PATH_TYPE_MISMATCH", Rule: "VR-006", Severity: "error", Message: "draft path does not match document type", Path: doc.Path, Field: "type"})
		}
		changeType := metaString(doc.Meta, "change_type")
		switch changeType {
		case "create":
			if metaString(doc.Meta, "target_id") != "" || metaString(doc.Meta, "retcon_reason") != "" {
				issues = append(issues, Issue{Code: "INVALID_ARGUMENT", Rule: "VR-002", Severity: "error", Message: "create draft must not define target_id or retcon_reason", Path: doc.Path})
			}
			if id != "" && idExists(ctx, id, false) {
				issues = append(issues, Issue{Code: "ID_CONFLICT", Rule: "VR-101", Severity: "conflict", Message: "id already exists in content", Path: doc.Path, Field: "id"})
			}
		case "update", "deprecate":
			targetID := metaString(doc.Meta, "target_id")
			if targetID == "" {
				issues = append(issues, Issue{Code: "MISSING_TARGET", Rule: "VR-002", Severity: "error", Message: "update/deprecate draft requires target_id", Path: doc.Path, Field: "target_id"})
			}
			if metaString(doc.Meta, "retcon_reason") == "" {
				issues = append(issues, Issue{Code: "MISSING_FIELD", Rule: "VR-503", Severity: "error", Message: "update/deprecate draft requires retcon_reason", Path: doc.Path, Field: "retcon_reason"})
			}
			if id != "" && targetID != "" && id != targetID {
				issues = append(issues, Issue{Code: "INVALID_ARGUMENT", Rule: "VR-002", Severity: "error", Message: "draft id must match target_id", Path: doc.Path, Field: "target_id"})
			}
			if targetID != "" {
				if _, ok := findContentByID(ctx, targetID); !ok {
					issues = append(issues, Issue{Code: "MISSING_TARGET", Rule: "VR-002", Severity: "conflict", Message: "target canon document is missing", Path: doc.Path, Field: "target_id"})
				}
			}
		default:
			issues = append(issues, Issue{Code: "INVALID_ARGUMENT", Rule: "VR-002", Severity: "error", Message: "draft requires change_type create/update/deprecate", Path: doc.Path, Field: "change_type"})
		}
	}
	if docType == "storylet" && strings.HasPrefix(doc.Path, "content/") {
		issues = append(issues, Issue{Code: "STORYLET_NOT_CANON_TARGET", Rule: "VR-507", Severity: "error", Message: "storylet cannot be canon content", Path: doc.Path, Field: "type"})
	}
	validateRelated(ctx, doc, acceptMode, &issues)
	validateRelationships(ctx, doc, acceptMode, &issues)
	validateTimeline(doc, &issues)
	return validationStatus(issues), issues
}

func validateRelated(ctx *WorldContext, doc Document, acceptMode bool, issues *[]Issue) {
	for _, id := range metaStringList(doc.Meta, "related") {
		if id == "" {
			continue
		}
		if _, ok := findContentByID(ctx, id); ok {
			continue
		}
		severity := "warning"
		if acceptMode {
			severity = "conflict"
		}
		*issues = append(*issues, Issue{Code: "MISSING_TARGET", Rule: "VR-301", Severity: severity, Message: "related id is not canon content", Path: doc.Path, Field: "related"})
	}
}

func validateRelationships(ctx *WorldContext, doc Document, acceptMode bool, issues *[]Issue) {
	items, ok := doc.Meta["relationships"].([]any)
	if !ok {
		if doc.Meta["relationships"] != nil {
			*issues = append(*issues, Issue{Code: "INVALID_RELATIONSHIP", Rule: "VR-302", Severity: "error", Message: "relationships must be an array", Path: doc.Path, Field: "relationships"})
		}
		return
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			*issues = append(*issues, Issue{Code: "INVALID_RELATIONSHIP", Rule: "VR-302", Severity: "error", Message: "relationship item must be an object", Path: doc.Path, Field: "relationships"})
			continue
		}
		relType := fmt.Sprint(m["type"])
		target := fmt.Sprint(m["target"])
		meta, known := relationshipTypes[relType]
		if !known {
			*issues = append(*issues, Issue{Code: "UNKNOWN_RELATIONSHIP_TYPE", Rule: "VR-304", Severity: "conflict", Message: "unknown relationship type", Path: doc.Path, Field: "relationships.type"})
			continue
		}
		if !meta.domain[doc.Type()] {
			*issues = append(*issues, Issue{Code: "RELATIONSHIP_DOMAIN_RANGE", Rule: "VR-305", Severity: "conflict", Message: "relationship source type is outside domain", Path: doc.Path, Field: "relationships.type"})
		}
		targetDoc, found := findContentByID(ctx, target)
		if !found {
			severity := "warning"
			if acceptMode {
				severity = "conflict"
			}
			*issues = append(*issues, Issue{Code: "MISSING_TARGET", Rule: "VR-303", Severity: severity, Message: "relationship target is not canon content", Path: doc.Path, Field: "relationships.target"})
			continue
		}
		if !meta.rng[targetDoc.Type()] {
			*issues = append(*issues, Issue{Code: "RELATIONSHIP_DOMAIN_RANGE", Rule: "VR-305", Severity: "conflict", Message: "relationship target type is outside range", Path: doc.Path, Field: "relationships.target"})
		}
	}
}

func validateTimeline(doc Document, issues *[]Issue) {
	if doc.Type() == "character" {
		birth, bok := metaInt(doc.Meta, "birth_year")
		death, dok := metaInt(doc.Meta, "death_year")
		if bok && dok && birth > death {
			*issues = append(*issues, Issue{Code: "TIMELINE_CONFLICT", Rule: "VR-201", Severity: "conflict", Message: "birth_year is after death_year", Path: doc.Path})
		}
	}
	if doc.Type() == "event" {
		start, sok := metaInt(doc.Meta, "start_year")
		end, eok := metaInt(doc.Meta, "end_year")
		if sok && eok && start > end {
			*issues = append(*issues, Issue{Code: "TIMELINE_CONFLICT", Rule: "VR-202", Severity: "conflict", Message: "start_year is after end_year", Path: doc.Path})
		}
	}
}

func metaInt(meta map[string]any, key string) (int, bool) {
	v, ok := meta[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		if strings.TrimSpace(t) == "" {
			return 0, false
		}
		n, err := strconv.Atoi(t)
		return n, err == nil
	default:
		return 0, false
	}
}

func validationStatus(issues []Issue) string {
	rank := map[string]int{"pass": 0, "warning": 1, "conflict": 2, "error": 3}
	status := "pass"
	for _, issue := range issues {
		if rank[issue.Severity] > rank[status] {
			status = issue.Severity
		}
	}
	return status
}

func validationActions(status string) []string {
	switch status {
	case "pass", "warning":
		return []string{"world_diff_draft", "world_update_draft", "world_reject_draft"}
	case "conflict":
		return []string{"world_update_draft", "world_reject_draft", "world_diff_draft", "world_validate_draft"}
	default:
		return []string{"world_update_draft", "world_reject_draft", "world_validate_draft"}
	}
}
