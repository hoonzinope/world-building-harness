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
	packsRoot    string
	registry     string
	basePath     string
	authRequired bool
	storyEnabled bool
	auth         *authStore
	stories      *storyStore
	md           goldmark.Markdown
}

func runServe(args []string) int {
	fs := flagSet("serve")
	addr := fs.String("addr", envDefault("WORLD_HARNESS_ADDR", ":8097"), "listen address")
	packsRoot := fs.String("packs-root", envDefault("WORLD_HARNESS_PACKS_ROOT", "packs"), "packs root")
	registry := fs.String("registry", os.Getenv("WORLD_TOOL_REGISTRY"), "registry")
	basePath := fs.String("base-path", envDefault("WORLD_HARNESS_BASE_PATH", ""), "base path")
	dataRoot := fs.String("data-root", envDefault("WORLD_HARNESS_DATA_ROOT", "/app/data"), "runtime data root")
	authRequired := fs.Bool("auth-required", envBool("WORLD_HARNESS_AUTH_REQUIRED", false), "require login for web routes")
	storyEnabled := fs.Bool("story-enabled", envBool("WORLD_HARNESS_STORY_ENABLED", false), "enable private story UI routes")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	var auth *authStore
	var stories *storyStore
	var err error
	if *authRequired || *storyEnabled {
		auth, err = openAuthStore(filepath.Join(*dataRoot, "auth.sqlite"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if *storyEnabled {
		stories, err = openStoryStore(filepath.Join(*dataRoot, "stories"), *packsRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		adminID, err := auth.firstActiveAdminID()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := stories.ensureSeedStories(adminID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &webServer{
		packsRoot:    *packsRoot,
		registry:     *registry,
		basePath:     strings.TrimRight(*basePath, "/"),
		authRequired: *authRequired,
		storyEnabled: *storyEnabled,
		auth:         auth,
		stories:      stories,
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	if stories != nil {
		stories.startGMWorker(ctx, newGMProvider(envDefault("WORLD_HARNESS_GM_PROVIDER", "mock")))
	}
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

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (s *webServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Cache-Control", "no-store")
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
	if path == "/login" && s.auth != nil {
		s.handleLogin(w, r)
		return
	}
	var u *authUser
	if s.authRequired || strings.HasPrefix(path, "/stories") || strings.HasPrefix(path, "/admin") || path == "/logout" {
		if s.auth == nil {
			http.NotFound(w, r)
			return
		}
		var ok bool
		u, ok = s.requireAuth(w, r)
		if !ok {
			return
		}
		r = withUser(r, u)
	}
	switch {
	case path == "/" || path == "":
		if s.storyEnabled {
			http.Redirect(w, r, s.base(r)+"/stories", http.StatusSeeOther)
			return
		}
		s.renderIndex(w, r)
	case path == "/logout":
		s.handleLogout(w, r)
	case path == "/stories":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.renderStoryLobby(w, r)
	case path == "/stories/new":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleNewStory(w, r)
	case path == "/stories/import/hector":
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleImportHector(w, r)
	case strings.HasPrefix(path, "/stories/"):
		if !s.storyEnabled {
			http.NotFound(w, r)
			return
		}
		s.handleStoryRoute(w, r, strings.TrimPrefix(path, "/stories/"))
	case path == "/admin/users":
		s.handleAdminUsers(w, r)
	case strings.HasPrefix(path, "/packs/"):
		s.renderPackRoute(w, r, strings.TrimPrefix(path, "/packs/"))
	case path == "/api/packs":
		s.renderPackAPI(w)
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) requireAuth(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err == nil && c.Value != "" {
		if u, err := s.auth.userForToken(c.Value); err == nil {
			return u, true
		}
	}
	http.Redirect(w, r, s.base(r)+"/login", http.StatusSeeOther)
	return nil, false
}

func (s *webServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.render(w, "Login", loginTemplate, map[string]any{"Base": s.base(r)})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, err := s.auth.authenticate(strings.TrimSpace(r.FormValue("username")), r.FormValue("password"))
	if err != nil {
		s.render(w, "Login", loginTemplate, map[string]any{"Base": s.base(r), "Error": "로그인 정보를 확인할 수 없습니다."})
		return
	}
	token, expires, err := s.auth.createSession(u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, token, expires, r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https")
	http.Redirect(w, r, s.base(r)+"/stories", http.StatusSeeOther)
}

func (s *webServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.auth.revokeToken(c.Value)
	}
	clearSessionCookie(w)
	http.Redirect(w, r, s.base(r)+"/login", http.StatusSeeOther)
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

func (s *webServer) renderStoryLobby(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	stories, err := s.stories.listStories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := []map[string]any{}
	for _, m := range stories {
		canDrive := u.Role == "admin" || (m.ActiveDriverID == u.ID && m.Status == "active" && m.Phase == "waiting_for_choice")
		rows = append(rows, map[string]any{"id": m.ID, "title": m.Title, "status": m.Status, "phase": m.Phase, "turn": m.CurrentTurn, "active_driver": m.ActiveDriverID, "summary": m.LatestSummary, "updated": m.UpdatedAt, "can_drive": canDrive, "imported": m.SourceDraftPath != ""})
	}
	s.render(w, "Stories", storyLobbyTemplate, map[string]any{"Base": s.base(r), "User": u, "Stories": rows})
}

func (s *webServer) handleNewStory(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if r.Method == http.MethodGet {
		s.render(w, "New Story", newStoryTemplate, map[string]any{"Base": s.base(r), "User": u})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id, err := s.stories.createStoryWithPrologueJob(u.ID, r.FormValue("title"), r.FormValue("style"), r.FormValue("character_name"), r.FormValue("traits"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleImportHector(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, _, err := s.stories.importHector(currentUser(r).ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleStoryRoute(w http.ResponseWriter, r *http.Request, rest string) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "":
		s.renderStoryRoom(w, r, id)
	case "input":
		s.handleStoryInput(w, r, id)
	case "question":
		s.handleStoryQuestion(w, r, id)
	case "driver":
		s.handleStoryDriver(w, r, id)
	case "admin":
		s.handleStoryAdmin(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *webServer) renderStoryRoom(w http.ResponseWriter, r *http.Request, id string) {
	u := currentUser(r)
	m, err := s.stories.readManifest(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	st, _ := s.stories.readState(id)
	turns, _ := s.stories.readTurns(id)
	qa, _ := s.stories.readQA(id)
	latestTurnID := 0
	if len(turns) > 0 {
		latestTurnID = turns[len(turns)-1].TurnID
	}
	canDrive := u.Role == "admin" || (u.ID == m.ActiveDriverID && m.Status == "active" && m.Phase == "waiting_for_choice")
	canClaim := m.ActiveDriverID == "" && m.Status == "active" && m.Phase == "waiting_for_choice"
	canRelease := (u.Role == "admin" || u.ID == m.ActiveDriverID) && m.ActiveDriverID != "" && m.Status == "active" && m.Phase == "waiting_for_choice"
	canQuestion := (m.Status == "active" || m.Status == "paused") && m.Phase == "waiting_for_choice"
	driverLabel := m.ActiveDriverID
	if driverLabel == "" {
		driverLabel = "open"
	}
	isProcessing := m.Phase == "gm_generating" || m.Phase == "validating_output" || m.Phase == "applying_turn"
	data := map[string]any{"Base": s.base(r), "User": u, "Story": m, "State": st, "Turns": turns, "QA": qa, "CanDrive": canDrive, "CanClaim": canClaim, "CanRelease": canRelease, "CanQuestion": canQuestion, "IsAdmin": u.Role == "admin", "LatestTurnID": latestTurnID, "DriverLabel": driverLabel, "IsProcessing": isProcessing}
	s.render(w, m.Title, storyRoomTemplate, data)
}

func (s *webServer) handleStoryInput(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_, err := s.stories.submitStoryInput(id, currentUser(r), r.FormValue("choice_id"), r.FormValue("mode"), strings.TrimSpace(r.FormValue("custom_text")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#input-panel", http.StatusSeeOther)
}

func (s *webServer) handleStoryQuestion(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.stories.askQuestion(id, currentUser(r), strings.TrimSpace(r.FormValue("question"))); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#qa", http.StatusSeeOther)
}

func (s *webServer) handleStoryDriver(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.stories.updateDriver(id, currentUser(r), r.FormValue("action")); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleStoryAdmin(w http.ResponseWriter, r *http.Request, id string) {
	u := currentUser(r)
	if u.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.stories.adminUpdateStory(id, r.FormValue("status"), r.FormValue("active_driver_id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch r.FormValue("action") {
		case "create":
			if err := s.auth.createUser(r.FormValue("username"), r.FormValue("display_name"), r.FormValue("role"), r.FormValue("password")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "update":
			if err := s.auth.updateUser(r.FormValue("id"), r.FormValue("role"), r.FormValue("status")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "reset":
			if err := s.auth.resetPassword(r.FormValue("id"), r.FormValue("password")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "revoke":
			if err := s.auth.revokeUserSessions(r.FormValue("id")); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		http.Redirect(w, r, s.base(r)+"/admin/users", http.StatusSeeOther)
		return
	}
	users, err := s.auth.listUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "Admin Users", adminUsersTemplate, map[string]any{"Base": s.base(r), "User": u, "Users": users})
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
	data["StoryEnabled"] = s.storyEnabled
	t, err := template.New("page").Funcs(template.FuncMap{
		"docURL": func(base, pack, path string) string {
			return fmt.Sprintf("%s/packs/%s/doc?path=%s", base, url.PathEscape(pack), url.QueryEscape(path))
		},
		"packURL": func(base, pack string) string {
			return fmt.Sprintf("%s/packs/%s/", base, url.PathEscape(pack))
		},
		"storyURL": func(base, id string) string {
			return fmt.Sprintf("%s/stories/%s", base, url.PathEscape(id))
		},
		"eq":  func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) },
		"not": func(v bool) bool { return !v },
		"nl2br": func(s string) template.HTML {
			return template.HTML(template.HTMLEscapeString(s))
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
:root { --paper:#f7f4ed; --ink:#1f2321; --muted:#68706b; --line:#d7d0c3; --accent:#b8332d; --deep:#0f332e; --wash:#e9e2d4; --panel:#fffaf0; --ok:#1e6b4f; --warn:#8a4f00; --shadow:0 16px 42px rgba(31,35,33,.12); }
* { box-sizing:border-box; }
html { scroll-behavior:smooth; }
body { margin:0; background:linear-gradient(180deg,#f8f4ea 0%,#eee7d8 100%); color:var(--ink); font-family: ui-serif, Georgia, "Apple SD Gothic Neo", "Noto Serif KR", serif; line-height:1.65; }
a { color:var(--deep); text-decoration-thickness:1px; text-underline-offset:3px; }
.shell { max-width:1180px; margin:0 auto; padding:28px 20px 88px; }
.top { display:flex; align-items:flex-end; justify-content:space-between; gap:20px; border-bottom:1px solid var(--line); padding-bottom:18px; margin-bottom:24px; }
.brand { font-size:13px; letter-spacing:.08em; text-transform:uppercase; color:var(--muted); font-family: ui-sans-serif, system-ui, sans-serif; }
.crumb, .nav { font-family: ui-sans-serif, system-ui, sans-serif; font-size:14px; color:var(--muted); display:flex; gap:12px; flex-wrap:wrap; justify-content:flex-end; }
.nav a { color:var(--deep); }
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
.search input, input, select, textarea { border:1px solid var(--line); border-radius:6px; padding:12px 13px; background:var(--panel); font:inherit; width:100%; min-height:44px; }
textarea { min-height:110px; resize:vertical; }
button, .button { border:1px solid var(--deep); background:var(--deep); color:white; border-radius:6px; padding:11px 15px; min-height:44px; font:600 14px ui-sans-serif, system-ui, sans-serif; cursor:pointer; text-decoration:none; display:inline-flex; align-items:center; justify-content:center; }
button:disabled { opacity:.45; cursor:not-allowed; }
button.secondary, .button.secondary { background:transparent; color:var(--deep); }
button.danger { border-color:var(--accent); background:var(--accent); }
.toolbar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin:18px 0 22px; }
.table { width:100%; border-collapse:collapse; font-family:ui-sans-serif, system-ui, sans-serif; font-size:14px; }
.table th, .table td { text-align:left; border-bottom:1px solid var(--line); padding:10px 8px; vertical-align:top; }
.badge { display:inline-flex; border:1px solid var(--line); border-radius:999px; padding:2px 8px; font:12px ui-sans-serif, system-ui, sans-serif; color:var(--muted); background:rgba(255,255,255,.35); }
.story-header { display:grid; grid-template-columns:minmax(0,1fr) auto; gap:16px; align-items:end; border-bottom:1px solid var(--line); padding-bottom:18px; margin-bottom:20px; }
.story-header h1 { margin-bottom:0; }
.driver-actions { display:flex; gap:8px; flex-wrap:wrap; justify-content:flex-end; }
.story-layout { display:grid; grid-template-columns:minmax(0, 1fr) 340px; gap:34px; align-items:start; }
.story-layout article { background:linear-gradient(90deg, rgba(84,58,34,.08), transparent 22px), var(--panel); border:1px solid #cfc4b2; border-radius:8px; box-shadow:0 22px 70px rgba(43,34,24,.14); padding:30px clamp(18px,4vw,54px); position:relative; }
.story-layout article::before { content:""; position:absolute; left:18px; top:18px; bottom:18px; width:1px; background:rgba(94,67,42,.18); }
.story-layout aside { position:sticky; top:18px; }
.turn-nav { display:flex; gap:8px; flex-wrap:wrap; margin:8px 0 20px; font-family:ui-sans-serif, system-ui, sans-serif; }
.turn-nav a { min-height:36px; display:inline-flex; align-items:center; border:1px solid var(--line); border-radius:999px; padding:5px 10px; background:rgba(255,250,240,.72); text-decoration:none; box-shadow:0 4px 14px rgba(31,35,33,.05); }
.turn { border-top:1px solid var(--line); padding-top:22px; margin-top:18px; scroll-margin-top:20px; }
.turn h2 { border:0; padding:0; margin:0 0 8px; }
.turn-card { border-top:1px solid rgba(94,67,42,.18); margin-top:20px; padding-top:0; scroll-margin-top:18px; }
.turn-card:first-child { border-top:0; margin-top:0; }
.turn-card summary { min-height:58px; cursor:pointer; display:flex; align-items:center; justify-content:space-between; gap:12px; list-style:none; padding:18px 0; font:700 18px ui-sans-serif, system-ui, sans-serif; }
.turn-card summary::-webkit-details-marker { display:none; }
.turn-card summary::after { content:"열기"; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; border:1px solid var(--line); border-radius:999px; padding:3px 8px; }
.turn-card[open] summary::after { content:"접기"; }
.turn-title { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; }
.scene { white-space:pre-wrap; font-size:19px; line-height:1.92; max-width:72ch; margin:18px auto 24px; text-wrap:pretty; }
.choice-list { display:grid; gap:10px; margin:12px 0; }
.choice { text-align:left; justify-content:flex-start; background:var(--panel); color:var(--ink); border-color:var(--line); }
.choice strong { margin-right:8px; color:var(--accent); }
.archived-choice { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.25); padding:10px 12px; }
.input-panel { scroll-margin-top:18px; }
.mobile-action-dock { display:none; }
.form-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:12px; }
.panel { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.35); padding:16px; margin-bottom:14px; }
.story-layout article > .panel, .turn-card .panel { background:rgba(255,255,255,.46); }
.status-panel { border-left:4px solid var(--accent); background:rgba(255,250,240,.72); }
.panel h2, .panel h3 { margin-top:0; border:0; padding-top:0; font-family:ui-sans-serif, system-ui, sans-serif; }
.panel ul { padding-left:20px; }
.muted { color:var(--muted); font-family:ui-sans-serif, system-ui, sans-serif; font-size:13px; }
.error { color:var(--accent); font-family:ui-sans-serif, system-ui, sans-serif; }
@media (max-width:820px){
  .shell{padding:16px 14px 112px;}
  .top{align-items:flex-start; flex-direction:column; gap:10px; margin-bottom:18px;}
  .nav{justify-content:flex-start;}
  .reader{grid-template-columns:1fr;}
  .side{position:static;}
  h1{font-size:38px;}
  .story-header{grid-template-columns:1fr; align-items:start;}
  .story-layout article{padding:18px 16px; border-radius:6px;}
  .story-layout article::before{display:none;}
  .story-layout aside{position:static;}
  .driver-actions{justify-content:flex-start;}
  .scene{font-size:17px; line-height:1.72;}
  .panel{padding:14px;}
  .toolbar > *, .driver-actions > *{flex:1 1 auto;}
  button, .button{width:100%; min-height:48px;}
  .table, .table tbody, .table tr, .table td{display:block; width:100%;}
  .table thead{display:none;}
  .table tr{border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.35); margin:0 0 12px; padding:10px;}
  .table td{border:0; padding:6px 4px;}
  .mobile-action-dock{position:fixed; left:0; right:0; bottom:0; z-index:10; display:grid; grid-template-columns:1fr 1fr; gap:8px; padding:10px 12px calc(10px + env(safe-area-inset-bottom)); background:rgba(247,244,237,.94); border-top:1px solid var(--line); box-shadow:var(--shadow); backdrop-filter:blur(12px);}
  .mobile-action-dock a{min-height:48px;}
}
@media (max-width:960px){ .story-layout{grid-template-columns:1fr;} .table{font-size:13px;} }
</style>
</head>
<body>
<main class="shell">
<div class="top"><a class="brand" href="{{.Base}}/">World Harness</a><div class="nav">{{if .StoryEnabled}}<a href="{{.Base}}/stories">스토리</a>{{end}}<a href="{{.Base}}/packs/lumen-federation/">세계관</a>{{with .User}}{{if eq .Role "admin"}}<a href="{{$.Base}}/admin/users">Admin</a>{{end}}<a href="{{$.Base}}/logout">Logout</a>{{else}}<span>{{$.PageTitle}}</span>{{end}}</div></div>
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

const loginTemplate = `{{define "content"}}
<h1>World Harness</h1>
<p class="lede">Private story runtime</p>
{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
<form method="post" class="panel" style="max-width:420px">
  <label class="muted">Username</label>
  <input name="username" autocomplete="username" required autofocus>
  <label class="muted">Password</label>
  <input name="password" type="password" autocomplete="current-password" required>
  <div class="toolbar"><button>로그인</button></div>
</form>
{{end}}`

const storyLobbyTemplate = `{{define "content"}}
<h1>Stories</h1>
<p class="lede">세계관 문서를 읽고, runtime story room에서 장면 단위로 진행합니다.</p>
<div class="toolbar">
  <a class="button" href="{{.Base}}/stories/new">새 스토리</a>
  <a class="button secondary" href="{{.Base}}/stories">새로고침</a>
</div>
<table class="table">
  <thead><tr><th>제목</th><th>상태</th><th>Turn</th><th>진행자</th><th>현재 상황</th><th>업데이트</th><th>권한</th><th></th></tr></thead>
  <tbody>
  {{range .Stories}}
    <tr>
      <td><strong>{{.title}}</strong></td>
      <td><span class="badge">{{.status}}</span><br><span class="muted">{{.phase}}</span></td>
      <td>{{.turn}}</td>
      <td>{{if .active_driver}}{{.active_driver}}{{else}}open{{end}}</td>
      <td>{{.summary}}</td>
      <td class="muted">{{.updated}}</td>
      <td>{{if .can_drive}}진행 가능{{else}}읽기/질문{{end}}</td>
      <td><a class="button secondary" href="{{storyURL $.Base .id}}">입장</a></td>
    </tr>
  {{else}}
    <tr><td colspan="8" class="muted">아직 story room이 없습니다.</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}`

const newStoryTemplate = `{{define "content"}}
<h1>New Story</h1>
<form method="post" class="panel">
  <div class="form-grid">
    <div><label class="muted">World</label><input value="lumen-federation" disabled></div>
    <div><label class="muted">Title</label><input name="title" placeholder="새 스토리"></div>
    <div><label class="muted">Style</label><select name="style"><option>조사극</option><option>생존극</option><option>행정/법정극</option><option>앙상블</option><option>자유</option></select></div>
    <div><label class="muted">Character Name</label><input name="character_name" placeholder="캐릭터 이름"></div>
  </div>
  <label class="muted">특징 / 취향</label>
  <textarea name="traits" placeholder="캐릭터 특징, 보고 싶은 장면 압력, 피하고 싶은 톤"></textarea>
  <div class="toolbar"><button>프롤로그 생성</button><a class="button secondary" href="{{.Base}}/stories">취소</a></div>
</form>
{{end}}`

const storyRoomTemplate = `{{define "content"}}
{{if .IsProcessing}}<meta http-equiv="refresh" content="3">{{end}}
<div class="story-header">
  <div>
    <h1>{{.Story.Title}}</h1>
    <div class="toolbar">
      <span class="badge">{{.Story.Status}}</span><span class="badge">{{.Story.Phase}}</span><span class="badge">Turn {{.Story.CurrentTurn}}</span><span class="badge">driver {{.DriverLabel}}</span>
    </div>
  </div>
  <div class="driver-actions">
    {{if .CanClaim}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver"><input type="hidden" name="action" value="claim"><button>진행권 받기</button></form>{{end}}
    {{if .CanRelease}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver"><input type="hidden" name="action" value="release"><button class="secondary">open으로 나가기</button></form>{{end}}
  </div>
</div>
{{if .IsProcessing}}<div class="panel status-panel"><strong>GM 생성 중</strong><p>요청 이벤트가 접수되었습니다. Codex/GM worker가 장면을 생성하는 동안 추가 진행 입력은 잠시 막힙니다.</p><p class="muted">active job: {{.Story.ActiveJobID}} · phase: {{.Story.Phase}}</p></div>{{end}}
{{if eq .Story.Phase "failed_waiting_retry"}}<div class="panel status-panel"><strong>GM 생성 실패</strong><p>현재 job이 실패 상태입니다. 새 진행 입력은 실패 job 처리 후 가능합니다.</p><p class="muted">active job: {{.Story.ActiveJobID}}</p></div>{{end}}
<nav class="turn-nav" aria-label="turn list">
  {{range .Turns}}<a href="#turn-{{.TurnID}}">Turn {{.TurnID}}</a>{{end}}
</nav>
<div class="story-layout">
  <article>
    {{range .Turns}}
      <details class="turn-card" id="turn-{{.TurnID}}" {{if eq .TurnID $.LatestTurnID}}open{{end}}>
        <summary><span class="turn-title"><span>Turn {{.TurnID}}</span><span class="muted">{{.SceneTitle}}</span></span></summary>
        <div class="muted">{{.CreatedAt}} · {{.Source}}</div>
        <div class="scene">{{.SceneBody}}</div>
        <div class="panel"><strong>현재 상황</strong><p>{{.CurrentSituation}}</p></div>
        {{if .RevealedFacts}}<div class="panel"><strong>확인된 정보</strong><ul>{{range .RevealedFacts}}<li>{{.}}</li>{{end}}</ul></div>{{end}}
        {{$turnID := .TurnID}}{{if .Choices}}<div class="panel"><strong>{{if eq .TurnID $.LatestTurnID}}다음 갈림길{{else}}기록된 선택지{{end}}</strong><div class="choice-list">{{range .Choices}}{{if eq $turnID $.LatestTurnID}}<form method="post" action="{{$.Base}}/stories/{{$.Story.ID}}/input"><input type="hidden" name="choice_id" value="{{.ID}}"><button class="choice" {{if not $.CanDrive}}disabled{{end}}><strong>{{.ID}}</strong>{{.Text}}</button>{{if .RiskHint}}<div class="muted">{{.RiskHint}}</div>{{end}}</form>{{else}}<div class="archived-choice"><strong>{{.ID}}</strong> {{.Text}}{{if .RiskHint}}<div class="muted">{{.RiskHint}}</div>{{end}}</div>{{end}}{{end}}</div></div>{{end}}
      </details>
    {{end}}
    <section class="turn input-panel" id="input-panel">
      <h2>직접 입력</h2>
      {{if .CanDrive}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/input" class="panel">
        <div class="form-grid"><div><label class="muted">Mode</label><select name="mode"><option value="action">행동</option><option value="dialogue">대사</option><option value="narration">서술 보정</option></select></div></div>
        <textarea name="custom_text" placeholder="플레이어 캐릭터가 시도하는 행동/대사/서술 요청"></textarea>
        <div class="toolbar"><button>진행</button></div>
      </form>
      {{else}}{{if .IsProcessing}}<p class="muted">GM 생성 중입니다. 완료되면 새 턴이 자동으로 표시됩니다.</p>{{else}}{{if .CanClaim}}<p class="muted">현재 진행권이 open 상태입니다. 진행권을 받은 뒤 입력할 수 있습니다.</p>{{else}}<p class="muted">현재 {{.DriverLabel}}가 진행 중입니다. 진행 입력은 비활성화되어 있습니다.</p>{{end}}{{end}}{{end}}
      <h2 id="qa">질문</h2>
      {{if .CanQuestion}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/question" class="panel">
        <textarea name="question" placeholder="현재 상황, 인물, 단서, 설정, 선택지 의미를 묻는 비진행 질문"></textarea>
        <div class="toolbar"><button class="secondary">질문 제출</button></div>
      </form>
      {{else}}{{if .IsProcessing}}<p class="muted">GM 생성 중에는 질문 제출도 잠시 막습니다.</p>{{else}}<p class="muted">completed/archived room에서는 새 질문을 받지 않습니다.</p>{{end}}{{end}}
      {{range .QA}}<div class="panel"><div class="muted">{{.CreatedAt}} · Turn {{.TurnID}}</div><strong>Q. {{.Question}}</strong><p>A. {{.Answer}}</p></div>{{end}}
    </section>
  </article>
  <aside>
    <div class="panel"><h3>현재 상태</h3><p><strong>{{.State.Location}}</strong></p><p class="muted">인물: {{range .State.ActiveCharacters}}<span class="badge">{{.}}</span> {{end}}</p></div>
    <div class="panel"><h3>확인된 정보</h3><ul>{{range .State.Facts}}<li>{{.}}</li>{{end}}</ul></div>
    <div class="panel"><h3>열린 실마리</h3><ul>{{range .State.OpenThreads}}<li>{{.}}</li>{{end}}</ul></div>
    <div class="panel"><h3>위험</h3><ul>{{range .State.Risks}}<li>{{.}}</li>{{end}}</ul></div>
    {{if .IsAdmin}}<div class="panel"><h3>Admin</h3><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><label class="muted">Status</label><select name="status"><option value="">변경 없음</option><option>active</option><option>paused</option><option>completed</option><option>archived</option></select><label class="muted">Active driver user id</label><input name="active_driver_id" placeholder="{{.DriverLabel}}"><div class="toolbar"><button>적용</button></div></form><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="active_driver_id" value="__open__"><button class="secondary">open으로 변경</button></form></div>{{end}}
  </aside>
</div>
<div class="mobile-action-dock"><a class="button secondary" href="#turn-{{.LatestTurnID}}">최신 턴</a><a class="button" href="#input-panel">입력</a></div>
{{end}}`

const adminUsersTemplate = `{{define "content"}}
<h1>Admin Users</h1>
<div class="panel">
  <h2>Create user</h2>
  <form method="post" class="form-grid">
    <input type="hidden" name="action" value="create">
    <input name="username" placeholder="username" required>
    <input name="display_name" placeholder="display name">
    <select name="role"><option>friend</option><option>admin</option></select>
    <input name="password" type="password" placeholder="temporary password" required>
    <button>create</button>
  </form>
</div>
<table class="table">
<thead><tr><th>username</th><th>display</th><th>role/status</th><th>last login</th><th>sessions</th><th>actions</th></tr></thead>
<tbody>{{range .Users}}<tr>
  <td>{{.username}}<br><span class="muted">{{.id}}</span></td>
  <td>{{.display_name}}</td>
  <td><span class="badge">{{.role}}</span> <span class="badge">{{.status}}</span></td>
  <td class="muted">{{.last_login_at}}</td>
  <td>{{.active_sessions}}</td>
  <td>
    <form method="post" class="toolbar"><input type="hidden" name="action" value="update"><input type="hidden" name="id" value="{{.id}}"><select name="role"><option {{if eq .role "friend"}}selected{{end}}>friend</option><option {{if eq .role "admin"}}selected{{end}}>admin</option></select><select name="status"><option {{if eq .status "active"}}selected{{end}}>active</option><option {{if eq .status "disabled"}}selected{{end}}>disabled</option></select><button class="secondary">update</button></form>
    <form method="post" class="toolbar"><input type="hidden" name="action" value="reset"><input type="hidden" name="id" value="{{.id}}"><input name="password" type="password" placeholder="new password"><button class="secondary">reset password</button></form>
    <form method="post" class="toolbar"><input type="hidden" name="action" value="revoke"><input type="hidden" name="id" value="{{.id}}"><button class="danger">revoke sessions</button></form>
  </td>
</tr>{{end}}</tbody>
</table>
{{end}}`
