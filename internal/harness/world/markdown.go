package world

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/core"
	"gopkg.in/yaml.v3"
)

type Document struct {
	Path string         `json:"path"`
	Meta map[string]any `json:"frontmatter"`
	Body string         `json:"body"`
	Raw  string         `json:"raw"`
}

func (d Document) ID() string     { return MetaString(d.Meta, "id") }
func (d Document) Type() string   { return MetaString(d.Meta, "type") }
func (d Document) Status() string { return MetaString(d.Meta, "status") }
func (d Document) Title() string { return core.FirstNonEmpty(MetaString(d.Meta, "title"), headingTitle(d.Body), filepath.Base(d.Path)) }

func ParseMarkdown(rel string, b []byte) (Document, error) {
	raw := string(b)
	doc := Document{Path: filepath.ToSlash(rel), Meta: map[string]any{}, Raw: raw}
	if strings.HasPrefix(raw, "---\n") {
		rest := raw[4:]
		idx := strings.Index(rest, "\n---")
		if idx >= 0 {
			fm := rest[:idx]
			body := rest[idx+4:]
			body = strings.TrimPrefix(body, "\n")
			if err := yaml.Unmarshal([]byte(fm), &doc.Meta); err != nil {
				return doc, err
			}
			doc.Body = body
			return doc, nil
		}
	}
	doc.Body = raw
	return doc, nil
}

func BuildMarkdown(meta map[string]any, body string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("---\n")
	b, err := yaml.Marshal(meta)
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimSpace(body))
	buf.WriteString("\n")
	return buf.Bytes(), nil
}

func ReadDocument(ctx *Context, rel string) (Document, error) {
	if err := RequireMarkdownPath(rel); err != nil {
		return Document{}, err
	}
	abs, clean, err := SafeRel(ctx.Root, rel)
	if err != nil {
		return Document{}, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return Document{}, err
	}
	return ParseMarkdown(clean, b)
}

func MetaString(meta map[string]any, key string) string {
	v, ok := meta[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func MetaStringList(meta map[string]any, key string) []string {
	v, ok := meta[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func headingTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func HeadingTitle(body string) string { return headingTitle(body) }

func ListDocuments(ctx *Context, scope string) ([]Document, error) {
	roots := []string{}
	switch scope {
	case "", "active":
		roots = []string{"content", "drafts"}
	case "content":
		roots = []string{"content"}
	case "drafts":
		roots = []string{"drafts"}
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
	docs := []Document{}
	for _, prefix := range roots {
		base := filepath.Join(ctx.Root, prefix)
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "archive" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			rel, err := filepath.Rel(ctx.Root, path)
			if err != nil {
				return nil
			}
			doc, err := ReadDocument(ctx, filepath.ToSlash(rel))
			if err == nil {
				docs = append(docs, doc)
			}
			return nil
		})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, nil
}

func DocumentSummary(doc Document) map[string]any {
	return map[string]any{
		"path":   doc.Path,
		"id":     doc.ID(),
		"type":   doc.Type(),
		"status": doc.Status(),
		"title":  doc.Title(),
		"tags":   MetaStringList(doc.Meta, "tags"),
	}
}

func SearchDocuments(ctx *Context, scope, query string) ([]map[string]any, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	docs, err := ListDocuments(ctx, scope)
	if err != nil {
		return nil, err
	}
	results := []map[string]any{}
	for _, doc := range docs {
		hay := strings.ToLower(doc.Path + "\n" + doc.ID() + "\n" + doc.Title() + "\n" + strings.Join(MetaStringList(doc.Meta, "tags"), " ") + "\n" + doc.Body)
		if query == "" || strings.Contains(hay, query) {
			summary := DocumentSummary(doc)
			summary["snippet"] = snippet(doc.Body, query)
			results = append(results, summary)
		}
	}
	return results, nil
}

func snippet(body, query string) string {
	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	if body == "" {
		return ""
	}
	if query != "" {
		lower := strings.ToLower(body)
		if idx := strings.Index(lower, strings.ToLower(query)); idx >= 0 {
			start := idx - 90
			if start < 0 {
				start = 0
			}
			end := idx + len(query) + 160
			if end > len(body) {
				end = len(body)
			}
			return strings.TrimSpace(body[start:end])
		}
	}
	lines := strings.Split(body, "\n")
	out := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
		if len(strings.Join(out, " ")) > 220 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(out, " "))
}

func FindContentByID(ctx *Context, id string) (Document, bool) {
	docs, _ := ListDocuments(ctx, "content")
	for _, doc := range docs {
		if doc.ID() == id {
			return doc, true
		}
	}
	return Document{}, false
}

func IDExists(ctx *Context, id string, includeDrafts bool) bool {
	if _, ok := FindContentByID(ctx, id); ok {
		return true
	}
	if includeDrafts {
		docs, _ := ListDocuments(ctx, "drafts")
		for _, doc := range docs {
			if doc.ID() == id {
				return true
			}
		}
	}
	return false
}
