package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hoonzi/world-harness/internal/harness/ui"
)

var markdownLinkPattern = regexp.MustCompile(`\]\(([^)#][^)]+?\.md)(#[^)]+)?\)`)

func (s *webServer) base(r *http.Request) string {
	if forwarded := strings.TrimRight(r.Header.Get("X-Forwarded-Prefix"), "/"); forwarded != "" {
		return forwarded
	}
	return s.basePath
}

func (s *webServer) renderIndex(w http.ResponseWriter, r *http.Request) {
	packs := s.packs()
	data := map[string]any{"Title": "World Harness", "Base": s.base(r), "Packs": packs}
	s.render(w, r, "World Harness", ui.IndexTemplate, data)
}

func (s *webServer) renderPackRoute(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.SplitN(rest, "/", 2)
	pack := parts[0]
	if pack == "" {
		http.NotFound(w, r)
		return
	}
	ctx, err := s.packContext(pack)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if len(parts) == 1 || parts[1] == "" {
		s.renderPack(w, r, ctx, r.URL.Query().Get("q"))
		return
	}
	if parts[1] == "doc" {
		s.renderDoc(w, r, ctx)
		return
	}
	http.NotFound(w, r)
}

func (s *webServer) renderPack(w http.ResponseWriter, r *http.Request, ctx *WorldContext, query string) {
	docs, _ := listDocuments(ctx, "content")
	groups := map[string][]map[string]any{}
	for _, doc := range docs {
		if query != "" {
			hay := strings.ToLower(doc.Title() + " " + doc.Path + " " + doc.Body)
			if !strings.Contains(hay, strings.ToLower(query)) {
				continue
			}
		}
		groups[firstNonEmpty(doc.Type(), "misc")] = append(groups[firstNonEmpty(doc.Type(), "misc")], documentSummary(doc))
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	data := map[string]any{"Title": ctx.ID, "Base": s.base(r), "Pack": ctx.ID, "Summary": worldStatus(ctx)["summary"], "Groups": groups, "Types": keys, "Query": query}
	s.render(w, r, ctx.ID, ui.PackTemplate, data)
}

func (s *webServer) renderDoc(w http.ResponseWriter, r *http.Request, ctx *WorldContext) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	doc, err := readDocument(ctx, rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var htmlBuf bytes.Buffer
	if err := s.md.Convert([]byte(rewriteMarkdownLinks(doc.Body, doc.Path, ctx.ID, s.base(r))), &htmlBuf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": doc.Title(), "Base": s.base(r), "Pack": ctx.ID, "Doc": documentSummary(doc), "Frontmatter": doc.Meta, "BodyHTML": template.HTML(htmlBuf.String())}
	s.render(w, r, doc.Title(), ui.DocTemplate, data)
}

func rewriteMarkdownLinks(body, currentPath, pack, base string) string {
	currentDir := filepath.Dir(currentPath)
	return markdownLinkPattern.ReplaceAllStringFunc(body, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		target := parts[1]
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			return match
		}
		clean := filepath.ToSlash(filepath.Clean(filepath.Join(currentDir, target)))
		u := fmt.Sprintf("%s/packs/%s/doc?path=%s", base, url.PathEscape(pack), url.QueryEscape(clean))
		if len(parts) > 2 && parts[2] != "" {
			u += parts[2]
		}
		return "](" + u + ")"
	})
}

func (s *webServer) renderPackAPI(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"packs": s.packs()})
}

func (s *webServer) packContext(pack string) (*WorldContext, error) {
	root := filepath.Join(s.packsRoot, pack)
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	h, err := readHarness(root)
	if err != nil {
		return nil, err
	}
	id := firstNonEmpty(h.WorldID, pack)
	return &WorldContext{ID: id, Root: root, RegistryRoot: root, Harness: h}, nil
}

func (s *webServer) packs() []map[string]any {
	entries, err := os.ReadDir(s.packsRoot)
	if err != nil {
		return nil
	}
	out := []map[string]any{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ctx, err := s.packContext(e.Name())
		if err != nil {
			continue
		}
		status := worldStatus(ctx)
		out = append(out, map[string]any{"id": ctx.ID, "title": packTitle(ctx), "root": ctx.Root, "summary": status["summary"]})
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]["id"]) < fmt.Sprint(out[j]["id"]) })
	return out
}

func packTitle(ctx *WorldContext) string {
	idx := filepath.Join(ctx.Root, "content", "index.md")
	if b, err := os.ReadFile(idx); err == nil {
		if doc, err := parseMarkdown("content/index.md", b); err == nil {
			return doc.Title()
		}
	}
	return ctx.ID
}
