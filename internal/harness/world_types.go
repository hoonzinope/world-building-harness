package harness

type Registry struct {
	Default string                   `yaml:"default,omitempty" json:"default,omitempty"`
	Worlds  map[string]RegistryWorld `yaml:"worlds" json:"worlds"`
}

type RegistryWorld struct {
	Title string `yaml:"title" json:"title"`
	Root  string `yaml:"root" json:"root"`
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

func defaultHarness(worldID string) Harness {
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
