package world

import (
	"os"
	"path/filepath"
)

type Registry struct {
	Default string                   `yaml:"default,omitempty" json:"default,omitempty"`
	Worlds  map[string]RegistryWorld `yaml:"worlds" json:"worlds"`
}

type RegistryWorld struct {
	Title string `yaml:"title" json:"title"`
	Root  string `yaml:"root" json:"root"`
}

type Harness struct {
	SchemaVersion string `yaml:"schema_version" json:"schema_version"`
	WorldID       string `yaml:"world_id" json:"world_id"`
	WorldRoot     string `yaml:"world_root" json:"world_root"`
	ContentDir    string `yaml:"content_dir" json:"content_dir"`
	DraftDir      string `yaml:"draft_dir" json:"draft_dir"`
	RunDir        string `yaml:"run_dir" json:"run_dir"`
	InboxDir      string `yaml:"inbox_dir" json:"inbox_dir"`
	GraphDir      string `yaml:"graph_dir" json:"graph_dir"`
	ArchiveDir    string `yaml:"archive_dir" json:"archive_dir"`
}

var contentDirs = map[string]string{
	"character":    "characters",
	"nation":       "nations",
	"organization": "organizations",
	"place":        "places",
	"event":        "events",
	"timeline":     "timeline",
	"magic":        "magic",
	"glossary":     "glossary",
}

var draftDirs = map[string]string{
	"character":    "characters",
	"nation":       "nations",
	"organization": "organizations",
	"place":        "places",
	"event":        "events",
	"timeline":     "timeline",
	"magic":        "magic",
	"glossary":     "glossary",
	"storylet":     "storylets",
}

func DefaultHarness(worldID string) Harness {
	return Harness{
		SchemaVersion: "world-harness.v1",
		WorldID:       worldID,
		WorldRoot:     ".",
		ContentDir:    "content",
		DraftDir:      "drafts",
		RunDir:        "runs",
		InboxDir:      "runs/inbox",
		GraphDir:      "graph",
		ArchiveDir:    "archive",
	}
}

type Context struct {
	ID           string
	Root         string
	RegistryRoot string
	Harness      Harness
}

type CommonFlags struct {
	World    string
	Root     string
	WorldID  string
	Registry string
}

type Pack struct {
	ID     string
	Title  string
	Root   string
	Status map[string]any
}

func PackTitle(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	idx := filepath.Join(ctx.Root, "content", "index.md")
	if b, err := os.ReadFile(idx); err == nil {
		if doc, err := ParseMarkdown("content/index.md", b); err == nil {
			return doc.Title()
		}
	}
	return ctx.ID
}
