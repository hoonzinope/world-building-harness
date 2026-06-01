package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type webServer struct {
	packsRoot string
	registry  string
	basePath  string
	md        goldmark.Markdown
}

func runServe(args []string) int {
	fs := flagSet("serve")
	addr := fs.String("addr", envDefault("WORLD_HARNESS_ADDR", ":8097"), "listen address")
	packsRoot := fs.String("packs-root", envDefault("WORLD_HARNESS_PACKS_ROOT", "packs"), "packs root")
	registry := fs.String("registry", os.Getenv("WORLD_TOOL_REGISTRY"), "registry")
	basePath := fs.String("base-path", envDefault("WORLD_HARNESS_BASE_PATH", ""), "base path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	s := &webServer{
		packsRoot: *packsRoot,
		registry:  *registry,
		basePath:  strings.TrimRight(*basePath, "/"),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	server := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stderr, "world-harness serving %s from %s\n", *addr, *packsRoot)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		_ = server.Shutdown(context.Background())
		return 1
	}
	return 0
}

func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (s *webServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if r.URL.Path == "/health" || r.URL.Path == s.base(r)+"/health" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
		return
	}
	path := r.URL.Path
	base := s.base(r)
	if base != "" && strings.HasPrefix(path, base+"/") {
		path = strings.TrimPrefix(path, base)
	}
	switch {
	case path == "/" || path == "":
		s.renderIndex(w, r)
	case strings.HasPrefix(path, "/packs/"):
		s.renderPackRoute(w, r, strings.TrimPrefix(path, "/packs/"))
	case path == "/api/packs":
		s.renderPackAPI(w)
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) base(r *http.Request) string {
	if forwarded := strings.TrimRight(r.Header.Get("X-Forwarded-Prefix"), "/"); forwarded != "" {
		return forwarded
	}
	return s.basePath
}

func (s *webServer) renderIndex(w http.ResponseWriter, r *http.Request) {
	packs := s.packs()
	data := map[string]any{"Title": "World Harness", "Base": s.base(r), "Packs": packs}
	s.render(w, "World Harness", indexTemplate, data)
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
	data := map[string]any{
		"Title":   ctx.ID,
		"Base":    s.base(r),
		"Pack":    ctx.ID,
		"Summary": worldStatus(ctx)["summary"],
		"Groups":  groups,
		"Types":   keys,
		"Query":   query,
	}
	s.render(w, ctx.ID, packTemplate, data)
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
	data := map[string]any{
		"Title":       doc.Title(),
		"Base":        s.base(r),
		"Pack":        ctx.ID,
		"Doc":         documentSummary(doc),
		"Frontmatter": doc.Meta,
		"BodyHTML":    template.HTML(htmlBuf.String()),
	}
	s.render(w, doc.Title(), docTemplate, data)
}

var markdownLinkPattern = regexp.MustCompile(`\]\(([^)#][^)]+?\.md)(#[^)]+)?\)`)

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
		out = append(out, map[string]any{
			"id":      ctx.ID,
			"title":   packTitle(ctx),
			"root":    ctx.Root,
			"summary": status["summary"],
		})
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

func (s *webServer) render(w http.ResponseWriter, title, body string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data["PageTitle"] = title
	t, err := template.New("page").Funcs(template.FuncMap{
		"docURL": func(base, pack, path string) string {
			return fmt.Sprintf("%s/packs/%s/doc?path=%s", base, url.PathEscape(pack), url.QueryEscape(path))
		},
		"packURL": func(base, pack string) string {
			return fmt.Sprintf("%s/packs/%s/", base, url.PathEscape(pack))
		},
	}).Parse(layoutTemplate + body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const layoutTemplate = `<!doctype html>
<html lang="ko">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.PageTitle}}</title>
<style>
:root { --paper:#f7f4ed; --ink:#1f2321; --muted:#68706b; --line:#d7d0c3; --accent:#b8332d; --deep:#0f332e; --wash:#e9e2d4; }
* { box-sizing:border-box; }
body { margin:0; background:var(--paper); color:var(--ink); font-family: ui-serif, Georgia, "Apple SD Gothic Neo", "Noto Serif KR", serif; line-height:1.65; }
a { color:var(--deep); text-decoration-thickness:1px; text-underline-offset:3px; }
.shell { max-width:1180px; margin:0 auto; padding:28px 20px 72px; }
.top { display:flex; align-items:flex-end; justify-content:space-between; gap:20px; border-bottom:1px solid var(--line); padding-bottom:18px; margin-bottom:24px; }
.brand { font-size:13px; letter-spacing:.08em; text-transform:uppercase; color:var(--muted); font-family: ui-sans-serif, system-ui, sans-serif; }
.crumb { font-family: ui-sans-serif, system-ui, sans-serif; font-size:14px; color:var(--muted); }
h1 { font-size:clamp(34px, 6vw, 72px); line-height:1; margin:0 0 18px; letter-spacing:0; }
h2 { font-size:24px; margin:34px 0 12px; border-top:1px solid var(--line); padding-top:18px; }
.lede { max-width:760px; font-size:19px; color:#323833; }
.grid { display:grid; grid-template-columns:repeat(auto-fit, minmax(230px, 1fr)); gap:12px; }
.card { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.36); padding:16px; min-height:118px; }
.card strong { display:block; font-size:19px; margin-bottom:8px; }
.meta { font-family: ui-sans-serif, system-ui, sans-serif; color:var(--muted); font-size:13px; }
.doc-list { columns:2 320px; column-gap:28px; }
.doc-link { break-inside:avoid; display:block; padding:8px 0; border-bottom:1px solid rgba(31,35,33,.08); }
.doc-link span { display:block; color:var(--muted); font-family:ui-sans-serif, system-ui, sans-serif; font-size:12px; }
.reader { display:grid; grid-template-columns:minmax(0, 1fr) 280px; gap:42px; align-items:start; }
.prose { max-width:780px; }
.prose h1 { font-size:44px; margin-top:0; }
.prose h2 { font-size:24px; }
.prose p, .prose li { font-size:18px; }
.side { position:sticky; top:16px; border-left:3px solid var(--accent); padding-left:16px; font-family:ui-sans-serif, system-ui, sans-serif; color:var(--muted); }
.search { display:flex; gap:8px; max-width:520px; margin:18px 0 24px; }
.search input { flex:1; border:1px solid var(--line); border-radius:6px; padding:10px 12px; background:#fffaf0; font:inherit; }
.search button { border:1px solid var(--deep); background:var(--deep); color:white; border-radius:6px; padding:10px 14px; }
@media (max-width:820px){ .top{align-items:flex-start; flex-direction:column;} .reader{grid-template-columns:1fr;} .side{position:static;} h1{font-size:42px;} }
</style>
</head>
<body>
<main class="shell">
<div class="top"><a class="brand" href="{{.Base}}/">World Harness</a><div class="crumb">{{.PageTitle}}</div></div>
{{template "content" .}}
</main>
</body>
</html>`

const indexTemplate = `{{define "content"}}
<h1>World Packs</h1>
<p class="lede">읽기 전용 위키는 pack 단위로 분리되고, 변경은 Telegram/Codex와 world-tool draft workflow를 통해 들어갑니다.</p>
<div class="grid">
{{range .Packs}}
  <a class="card" href="{{packURL $.Base .id}}"><strong>{{.title}}</strong><span class="meta">{{.id}}</span><br><span class="meta">{{index .summary "content_documents"}} canon docs · {{index .summary "active_drafts"}} drafts</span></a>
{{else}}
  <div class="card"><strong>No packs</strong><span class="meta">Create packs/&lt;id&gt;/harness.yaml first.</span></div>
{{end}}
</div>
{{end}}`

const packTemplate = `{{define "content"}}
<h1>{{.Title}}</h1>
<p class="lede">{{index .Summary "content_documents"}} canon documents, {{index .Summary "active_drafts"}} active drafts.</p>
<form class="search" method="get"><input name="q" value="{{.Query}}" placeholder="문서 검색"><button>Search</button></form>
{{range .Types}}
  <h2>{{.}}</h2>
  <div class="doc-list">
  {{range index $.Groups .}}
    <a class="doc-link" href="{{docURL $.Base $.Pack .path}}">{{.title}}<span>{{.path}}</span></a>
  {{end}}
  </div>
{{end}}
{{end}}`

const docTemplate = `{{define "content"}}
<div class="reader">
  <article class="prose">{{.BodyHTML}}</article>
  <aside class="side">
    <div><strong>{{index .Doc "title"}}</strong></div>
    <div>{{index .Doc "type"}} · {{index .Doc "status"}}</div>
    <div>{{index .Doc "path"}}</div>
    <p><a href="{{packURL .Base .Pack}}">Back to pack</a></p>
  </aside>
</div>
{{end}}`
