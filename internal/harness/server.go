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
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type webServer struct {
	packsRoot    string
	dataRoot     string
	registry     string
	basePath     string
	authRequired bool
	storyEnabled bool
	auth         *authStore
	stories      *storyStore
	md           goldmark.Markdown
}

type lobbyStoryRow struct {
	ID          string
	Title       string
	Status      string
	Phase       string
	Turn        int
	Summary     string
	Updated     string
	Imported    bool
	IsMine      bool
	IsWatch     bool
	IsArchived  bool
	IsActive    bool
	CanDrive    bool
	DriverLabel string
	Permission  string
	StatusLabel string
}

type failedJobView struct {
	Job        gmJob
	CanRecover bool
	ActorLabel string
}

type storyProgressQuestionView struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	TurnID    int    `json:"turn_id"`
	Question  string `json:"question"`
	CreatedAt string `json:"created_at"`
}

type storyProgressView struct {
	StoryID          string                      `json:"story_id"`
	Status           string                      `json:"status"`
	Phase            string                      `json:"phase"`
	CurrentTurn      int                         `json:"current_turn"`
	ActiveJobID      string                      `json:"active_job_id,omitempty"`
	ActiveJobType    string                      `json:"active_job_type,omitempty"`
	ActiveJobStatus  string                      `json:"active_job_status,omitempty"`
	ActiveJobTurnID  int                         `json:"active_job_turn_id,omitempty"`
	IsProcessing     bool                        `json:"is_processing"`
	CanDrive         bool                        `json:"can_drive"`
	CanQuestion      bool                        `json:"can_question"`
	StatusLabel      string                      `json:"status_label"`
	ProgressMessage  string                      `json:"progress_message"`
	StepIndex        int                         `json:"step_index"`
	StepLabel        string                      `json:"step_label"`
	NextPollMS       int                         `json:"next_poll_ms"`
	JobStartedAt     string                      `json:"job_started_at,omitempty"`
	JobCompletedAt   string                      `json:"job_completed_at,omitempty"`
	JobErrorCode     string                      `json:"job_error_code,omitempty"`
	JobErrorMessage  string                      `json:"job_error_message,omitempty"`
	PendingQuestions []storyProgressQuestionView `json:"pending_questions,omitempty"`
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
		dataRoot:     *dataRoot,
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

func (s *webServer) addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; form-action 'self'; connect-src 'self'")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
}

func canDriveStory(m storyManifest, u *authUser) bool {
	return m.Status == "active" && m.Phase == "waiting_for_choice" && (u.Role == "admin" || u.ID == m.ActiveDriverID)
}

func canQuestionStory(m storyManifest) bool {
	return (m.Status == "active" || m.Status == "paused") && m.Phase == "waiting_for_choice"
}

func friendlyDriverLabel(m storyManifest, u *authUser) string {
	if m.ActiveDriverID == "" {
		return "open"
	}
	if u != nil && u.ID == m.ActiveDriverID {
		return "you"
	}
	if u != nil && u.Role == "admin" {
		return m.ActiveDriverID
	}
	return m.ActiveDriverID
}

func friendlyUserLabel(u *authUser, fallback string) string {
	if u == nil {
		return fallback
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.Username != "" {
		return u.Username
	}
	return fallback
}

func friendlyPermissionLabel(m storyManifest, u *authUser) string {
	switch {
	case m.Status == "completed" || m.Status == "archived" || m.Status == "deleted":
		return "종료"
	case canDriveStory(m, u):
		return "진행 가능"
	case canQuestionStory(m):
		return "질문 가능"
	case m.Status == "active" || m.Status == "paused":
		return "관전 가능"
	default:
		return "읽기 전용"
	}
}

func friendlyStatusLabel(m storyManifest) string {
	label := m.Status
	if m.Phase != "" {
		label += " / " + m.Phase
	}
	return label
}

func storyMatchesLobbyFilter(row lobbyStoryRow, filter string) bool {
	switch filter {
	case "", "all":
		return true
	case "active":
		return row.IsActive
	case "mine":
		return row.IsMine
	case "watch":
		return row.IsWatch
	case "archived":
		return row.IsArchived
	case "imported":
		return row.Imported
	default:
		return true
	}
}

func mustCSRFToken(w http.ResponseWriter, r *http.Request) string {
	token, err := ensureCSRFToken(w, r)
	if err != nil {
		return ""
	}
	return token
}

func (s *webServer) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if !requireCSRF(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func (s *webServer) requireStoryTaskCSRF(w http.ResponseWriter, r *http.Request) bool {
	if requireCSRF(r) {
		return true
	}
	if isJSONStoryTaskRequest(r) {
		formToken := strings.TrimSpace(r.FormValue("csrf_token"))
		if formToken != "" {
			headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if headerToken == "" || headerToken == formToken {
				return true
			}
		}
	}
	if wantsJSONResponse(r) {
		writeJSONResponse(w, http.StatusForbidden, map[string]any{"error": "csrf token mismatch"})
		return false
	}
	http.Error(w, "forbidden", http.StatusForbidden)
	return false
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

func (s *webServer) handle(w http.ResponseWriter, r *http.Request) {
	s.addSecurityHeaders(w)
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
	if path == "/assets/story-room.js" {
		s.handleStoryRoomAsset(w, r)
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
		s.render(w, r, "Login", loginTemplate, map[string]any{"Base": s.base(r), "CSRFToken": mustCSRFToken(w, r)})
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
	if !s.requireCSRF(w, r) {
		return
	}
	u, err := s.auth.authenticate(strings.TrimSpace(r.FormValue("username")), r.FormValue("password"))
	if err != nil {
		s.render(w, r, "Login", loginTemplate, map[string]any{"Base": s.base(r), "Error": "로그인 정보를 확인할 수 없습니다.", "CSRFToken": mustCSRFToken(w, r)})
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	s.render(w, r, "World Harness", indexTemplate, data)
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
	s.render(w, r, ctx.ID, packTemplate, data)
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
	s.render(w, r, doc.Title(), docTemplate, data)
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
	filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter")))
	if filter == "" {
		filter = "all"
	}
	rows := []lobbyStoryRow{}
	for _, m := range stories {
		row := lobbyStoryRow{
			ID:          m.ID,
			Title:       m.Title,
			Status:      m.Status,
			Phase:       m.Phase,
			Turn:        m.CurrentTurn,
			Summary:     m.LatestSummary,
			Updated:     m.UpdatedAt,
			Imported:    m.SourceDraftPath != "",
			IsMine:      m.CreatedBy == u.ID || m.ActiveDriverID == u.ID,
			IsWatch:     (m.Status == "active" || m.Status == "paused") && !canDriveStory(m, u),
			IsArchived:  m.Status == "completed" || m.Status == "archived" || m.Status == "deleted",
			IsActive:    m.Status == "active",
			CanDrive:    canDriveStory(m, u),
			DriverLabel: friendlyDriverLabel(m, u),
			Permission:  friendlyPermissionLabel(m, u),
			StatusLabel: friendlyStatusLabel(m),
		}
		if !storyMatchesLobbyFilter(row, filter) {
			continue
		}
		rows = append(rows, row)
	}
	s.render(w, r, "Stories", storyLobbyTemplate, map[string]any{"Base": s.base(r), "User": u, "Stories": rows, "Filter": filter, "CSRFToken": mustCSRFToken(w, r)})
}

func (s *webServer) handleNewStory(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if r.Method == http.MethodGet {
		s.render(w, r, "New Story", newStoryTemplate, map[string]any{"Base": s.base(r), "User": u, "CSRFToken": mustCSRFToken(w, r)})
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
	if !s.requireCSRF(w, r) {
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
	if !s.requireCSRF(w, r) {
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
	case "status":
		s.handleStoryStatus(w, r, id)
	case "driver":
		s.handleStoryDriver(w, r, id)
	case "admin":
		s.handleStoryAdmin(w, r, id)
	case "recover":
		s.handleStoryRecovery(w, r, id)
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
	hasTurns := len(turns) > 0
	displayTurns := append([]storyTurn(nil), turns...)
	sort.SliceStable(displayTurns, func(i, j int) bool {
		return displayTurns[i].TurnID > displayTurns[j].TurnID
	})
	latestTurnID := 0
	var latestTurn any
	if hasTurns {
		latestTurnID = displayTurns[0].TurnID
		latestTurn = displayTurns[0]
	}
	progress := s.storyRoomProgressSnapshot(id, m, u)
	isProcessing := progress.IsProcessing
	canDrive := canDriveStory(m, u) && !isProcessing
	canClaim := m.ActiveDriverID == "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canRelease := (u.Role == "admin" || u.ID == m.ActiveDriverID) && m.ActiveDriverID != "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canQuestion := canQuestionStory(m) && !isProcessing
	canAdminMutate := u.Role == "admin" && hasTurns && !isProcessing
	driverLabel := friendlyDriverLabel(m, u)
	var failedJob *failedJobView
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			failedJob = &failedJobView{Job: job, CanRecover: u.Role == "admin" || u.ID == job.ActorID, ActorLabel: job.ActorID}
		}
	}
	data := map[string]any{
		"Base":                s.base(r),
		"User":                u,
		"Story":               m,
		"State":               st,
		"Turns":               displayTurns,
		"QA":                  qa,
		"CanDrive":            canDrive,
		"CanClaim":            canClaim,
		"CanRelease":          canRelease,
		"CanQuestion":         canQuestion,
		"IsAdmin":             u.Role == "admin",
		"CanAdminMutate":      canAdminMutate,
		"LatestTurnID":        latestTurnID,
		"LatestTurn":          latestTurn,
		"HasTurns":            hasTurns,
		"DriverLabel":         driverLabel,
		"IsProcessing":        isProcessing,
		"Progress":            progress,
		"StatusURL":           s.base(r) + "/stories/" + url.PathEscape(id) + "/status",
		"FailedJob":           failedJob,
		"ExportedBundle":      strings.TrimSpace(r.URL.Query().Get("exported")),
		"ExportedStatus":      strings.TrimSpace(r.URL.Query().Get("export_status")),
		"ExportDraftTarget":   strings.TrimSpace(r.URL.Query().Get("export_draft_target")),
		"RecoveryStatus":      strings.TrimSpace(r.URL.Query().Get("recovery_status")),
		"RecoveryMessage":     strings.TrimSpace(r.URL.Query().Get("recovery_message")),
		"RecoveryChecked":     queryCSV(r.URL.Query().Get("recovery_checked")),
		"RecoveryRepaired":    queryCSV(r.URL.Query().Get("recovery_repaired")),
		"RecoveryLockRemoved": strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("recovery_lock_removed")), "true"),
		"CSRFToken":           mustCSRFToken(w, r),
	}
	s.render(w, r, m.Title, storyRoomTemplate, data)
}

func (s *webServer) handleStoryInput(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := parseStoryTaskRequest(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireStoryTaskCSRF(w, r) {
		return
	}
	u := currentUser(r)
	mode := strings.TrimSpace(r.FormValue("mode"))
	turnID := parseFormInt(r.FormValue("turn_id"))
	idem := strings.TrimSpace(r.FormValue("idempotency_key"))
	var (
		jobID string
		err   error
	)
	if mode == "question" {
		jobID, err = s.stories.submitQuestionJob(id, u, turnID, idem, strings.TrimSpace(r.FormValue("custom_text")))
	} else {
		jobID, err = s.stories.submitStoryInput(id, u, turnID, idem, r.FormValue("choice_id"), mode, strings.TrimSpace(r.FormValue("custom_text")))
	}
	if err != nil {
		s.writeStoryTaskError(w, r, err.Error(), http.StatusForbidden)
		return
	}
	if wantsJSONResponse(r) {
		m, readErr := s.stories.readManifest(id)
		if readErr != nil {
			s.writeStoryTaskError(w, r, readErr.Error(), http.StatusInternalServerError)
			return
		}
		progress := s.storyRoomProgressSnapshot(id, m, u)
		jobType := "story_turn"
		if mode == "question" {
			jobType = "question_answer"
		}
		s.writeStoryTaskAccepted(w, r, id, jobType, turnID, jobID, progress)
		return
	}
	fragment := "#input-panel"
	if mode == "question" {
		fragment = "#qa"
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+fragment, http.StatusSeeOther)
}

func (s *webServer) handleStoryQuestion(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := parseStoryTaskRequest(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireStoryTaskCSRF(w, r) {
		return
	}
	turnID := parseFormInt(r.FormValue("turn_id"))
	if turnID == 0 {
		if m, err := s.stories.readManifest(id); err == nil {
			turnID = m.CurrentTurn
		}
	}
	question := firstNonEmpty(strings.TrimSpace(r.FormValue("question")), strings.TrimSpace(r.FormValue("custom_text")))
	jobID, err := s.stories.submitQuestionJob(id, currentUser(r), turnID, strings.TrimSpace(r.FormValue("idempotency_key")), question)
	if err != nil {
		s.writeStoryTaskError(w, r, err.Error(), http.StatusForbidden)
		return
	}
	if wantsJSONResponse(r) {
		u := currentUser(r)
		m, readErr := s.stories.readManifest(id)
		if readErr != nil {
			s.writeStoryTaskError(w, r, readErr.Error(), http.StatusInternalServerError)
			return
		}
		progress := s.storyRoomProgressSnapshot(id, m, u)
		s.writeStoryTaskAccepted(w, r, id, "question_answer", turnID, jobID, progress)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#qa", http.StatusSeeOther)
}

func (s *webServer) handleStoryStatus(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	u := currentUser(r)
	m, err := s.stories.readManifest(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	progress := s.storyRoomProgressSnapshot(id, m, u)
	writeJSONResponse(w, http.StatusOK, progress)
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
	if !s.requireCSRF(w, r) {
		return
	}
	if err := s.stories.updateDriver(id, currentUser(r), r.FormValue("action")); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleStoryAdmin(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	u := currentUser(r)
	switch r.FormValue("action") {
	case "update":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.adminUpdateStory(id, u.ID, r.FormValue("status"), r.FormValue("active_driver_id")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "edit_turn":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.editCurrentTurn(id, u.ID, r.FormValue("scene_body"), r.FormValue("current_situation")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "rollback_turn":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		turnID := parseFormInt(r.FormValue("turn_id"))
		if turnID <= 0 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.stories.rollbackStoryToTurn(id, u.ID, turnID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "archive":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.archiveStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "restore":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.restoreStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "delete":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.deleteStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "export_bundle":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		bundlePath, err := s.stories.exportStoryBundle(id, u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		draftTarget := filepath.ToSlash(filepath.Join("drafts", "storylets", id+".md"))
		redirectURL := s.base(r) + "/stories/" + url.PathEscape(id) + "?exported=" + url.QueryEscape(bundlePath) + "&export_status=" + url.QueryEscape("draft_pending") + "&export_draft_target=" + url.QueryEscape(draftTarget)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	case "recover_store":
		if u.Role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		report, err := s.stories.recoverStory(id)
		if err != nil {
			redirectURL := s.base(r) + "/stories/" + url.PathEscape(id) + "?recovery_status=" + url.QueryEscape("failed") + "&recovery_message=" + url.QueryEscape(err.Error())
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}
		values := url.Values{}
		values.Set("recovery_status", report.RecoveryStatus)
		values.Set("recovery_checked", strings.Join(report.CheckedFiles, ","))
		values.Set("recovery_repaired", strings.Join(report.RepairedItems, ","))
		values.Set("recovery_lock_removed", fmt.Sprint(report.LockRemoved))
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"?"+values.Encode(), http.StatusSeeOther)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
	}
}

func (s *webServer) handleStoryRecovery(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	u := currentUser(r)
	switch r.FormValue("action") {
	case "resume":
		if _, err := s.stories.resumeFailedJob(id, u); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#input-panel", http.StatusSeeOther)
	case "cancel":
		if err := s.stories.cancelFailedJob(id, u); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id)+"#input-panel", http.StatusSeeOther)
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
	}
}

func (s *webServer) handleStoryRoomAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(storyRoomAssetJS))
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
		if !s.requireCSRF(w, r) {
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
	s.render(w, r, "Admin Users", adminUsersTemplate, map[string]any{"Base": s.base(r), "User": u, "Users": users, "CSRFToken": mustCSRFToken(w, r)})
}

func wantsJSONResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	if strings.Contains(accept, "application/json") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Requested-With")), "XMLHttpRequest")
}

func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *webServer) writeStoryTaskError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if wantsJSONResponse(r) {
		writeJSONResponse(w, status, map[string]any{"error": message})
		return
	}
	http.Error(w, message, status)
}

func (s *webServer) writeStoryTaskAccepted(w http.ResponseWriter, r *http.Request, storyID, jobType string, turnID int, jobID string, progress storyProgressView) {
	if jobID == "" {
		http.Error(w, "missing job id", http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusAccepted, map[string]any{
		"story_id":          storyID,
		"job_id":            jobID,
		"job_type":          jobType,
		"turn_id":           firstNonZero(progress.ActiveJobTurnID, turnID),
		"status_url":        s.base(r) + "/stories/" + url.PathEscape(storyID) + "/status",
		"next_poll_ms":      progress.NextPollMS,
		"status_label":      progress.StatusLabel,
		"progress_message":  progress.ProgressMessage,
		"step_index":        progress.StepIndex,
		"step_label":        progress.StepLabel,
		"is_processing":     progress.IsProcessing,
		"active_job_id":     progress.ActiveJobID,
		"active_job_type":   progress.ActiveJobType,
		"active_job_status": progress.ActiveJobStatus,
		"current_turn":      progress.CurrentTurn,
	})
}

func (s *webServer) storyRoomProgressSnapshot(id string, m storyManifest, u *authUser) storyProgressView {
	progress := storyProgressView{
		StoryID:     id,
		Status:      m.Status,
		Phase:       m.Phase,
		CurrentTurn: m.CurrentTurn,
		CanDrive:    canDriveStory(m, u),
		CanQuestion: canQuestionStory(m),
		StatusLabel: friendlyStatusLabel(m),
		StepIndex:   3,
		StepLabel:   "ready",
		NextPollMS:  0,
	}
	progress.ProgressMessage = "대기 중입니다. 새 입력을 제출할 수 있는 상태입니다."
	if progress.CanQuestion && !progress.CanDrive {
		progress.ProgressMessage = "질문은 현재 턴에 대해 보낼 수 있습니다."
	}
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			progress.ActiveJobID = job.ID
			progress.ActiveJobType = job.JobType
			progress.ActiveJobStatus = job.Status
			progress.ActiveJobTurnID = job.TurnID
			progress.JobStartedAt = job.StartedAt
			progress.JobCompletedAt = job.CompletedAt
			progress.JobErrorCode = job.ErrorCode
			progress.JobErrorMessage = job.ErrorMessage
			progress.ProgressMessage = "GM 작업이 실패했습니다. 복구 또는 취소가 필요합니다."
			progress.StepIndex = 4
			progress.StepLabel = "failed"
		}
		return progress
	}
	if m.ActiveJobID != "" {
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			progress.ActiveJobID = job.ID
			progress.ActiveJobType = job.JobType
			progress.ActiveJobStatus = job.Status
			progress.ActiveJobTurnID = job.TurnID
			progress.JobStartedAt = job.StartedAt
			progress.JobCompletedAt = job.CompletedAt
			progress.JobErrorCode = job.ErrorCode
			progress.JobErrorMessage = job.ErrorMessage
			progress.IsProcessing = job.Status == "queued" || job.Status == "running" || job.Status == "validating" || job.Status == "applying"
			progress.NextPollMS = 2500
			switch job.Status {
			case "queued":
				progress.StepIndex = 0
				progress.StepLabel = "queued"
				progress.ProgressMessage = fmt.Sprintf("작업이 대기열에 들어갔습니다. active job %s · 보통 10초-2분, Codex provider는 더 걸릴 수 있음", job.ID)
			case "running":
				progress.StepIndex = 1
				progress.StepLabel = "generating"
				progress.ProgressMessage = fmt.Sprintf("GM이 장면을 생성 중입니다. active job %s · 보통 10초-2분, Codex provider는 더 걸릴 수 있음", job.ID)
			case "validating", "applying":
				progress.StepIndex = 2
				progress.StepLabel = "applying"
				progress.ProgressMessage = fmt.Sprintf("생성 결과를 반영하는 중입니다. active job %s · phase %s", job.ID, job.Status)
			case "failed":
				progress.StepIndex = 4
				progress.StepLabel = "failed"
				progress.ProgressMessage = fmt.Sprintf("GM 작업이 실패했습니다. active job %s", job.ID)
			default:
				progress.ProgressMessage = fmt.Sprintf("GM 작업 상태를 확인 중입니다. active job %s · phase %s", job.ID, job.Status)
			}
			return progress
		}
	}
	pending := s.storyProgressPendingQuestions(id)
	if len(pending) > 0 {
		progress.PendingQuestions = pending
		progress.ActiveJobID = pending[0].JobID
		progress.ActiveJobType = "question_answer"
		progress.ActiveJobStatus = pending[0].Status
		progress.ActiveJobTurnID = pending[0].TurnID
		progress.IsProcessing = true
		progress.NextPollMS = 2500
		switch pending[0].Status {
		case "queued":
			progress.StepIndex = 0
			progress.StepLabel = "queued"
		case "running":
			progress.StepIndex = 1
			progress.StepLabel = "generating"
		default:
			progress.StepIndex = 1
			progress.StepLabel = "generating"
		}
		progress.ProgressMessage = fmt.Sprintf("질문 답변을 준비 중입니다. active job %s · 보통 10초-2분, Codex provider는 더 걸릴 수 있음", pending[0].JobID)
		return progress
	}
	return progress
}

func (s *webServer) storyProgressPendingQuestions(id string) []storyProgressQuestionView {
	jobs, err := s.stories.listJobs(id)
	if err != nil {
		return nil
	}
	out := make([]storyProgressQuestionView, 0, len(jobs))
	for _, job := range jobs {
		if job.JobType != "question_answer" || (job.Status != "queued" && job.Status != "running") {
			continue
		}
		text := ""
		if job.Question != nil {
			text = job.Question.Question
		}
		out = append(out, storyProgressQuestionView{
			JobID:     job.ID,
			Status:    job.Status,
			TurnID:    job.TurnID,
			Question:  text,
			CreatedAt: job.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].JobID < out[j].JobID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
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

func (s *webServer) render(w http.ResponseWriter, r *http.Request, title, body string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, ok := data["CSRFToken"]; !ok {
		data["CSRFToken"] = mustCSRFToken(w, r)
	}
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
		"idem": func() string {
			return mustTurnIdempotencyKey()
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
.shell { max-width:1180px; margin:0 auto; padding:28px 20px 124px; }
.top { display:flex; align-items:flex-end; justify-content:space-between; gap:20px; border-bottom:1px solid var(--line); padding-bottom:18px; margin-bottom:24px; }
.brand { font-size:13px; letter-spacing:.08em; text-transform:uppercase; color:var(--muted); font-family: ui-sans-serif, system-ui, sans-serif; }
.crumb, .nav { font-family: ui-sans-serif, system-ui, sans-serif; font-size:14px; color:var(--muted); display:flex; gap:12px; flex-wrap:wrap; justify-content:flex-end; }
.nav a { color:var(--deep); }
.nav-form { display:inline-flex; margin:0; }
.link-button { border:0; background:none; color:var(--deep); padding:0; min-height:auto; font:inherit; text-decoration:underline; text-underline-offset:3px; border-radius:0; }
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
.search input, input, select, textarea { border:1px solid var(--line); border-radius:6px; padding:12px 13px; background:var(--panel); font:inherit; width:100%; min-height:44px; max-width:100%; }
textarea { min-height:110px; resize:vertical; }
button, .button { border:1px solid var(--deep); background:var(--deep); color:white; border-radius:6px; padding:11px 15px; min-height:44px; font:600 14px ui-sans-serif, system-ui, sans-serif; cursor:pointer; text-decoration:none; display:inline-flex; align-items:center; justify-content:center; white-space:nowrap; }
button:focus-visible, .button:focus-visible, input:focus-visible, select:focus-visible, textarea:focus-visible, .link-button:focus-visible, a:focus-visible { outline:3px solid rgba(184,51,45,.45); outline-offset:2px; }
button:disabled { opacity:.45; cursor:not-allowed; }
button.secondary, .button.secondary { background:transparent; color:var(--deep); }
button.danger { border-color:var(--accent); background:var(--accent); }
.filter-bar { display:flex; gap:8px; flex-wrap:wrap; margin:12px 0 18px; }
.filter-link { display:inline-flex; align-items:center; border:1px solid var(--line); border-radius:999px; padding:6px 12px; background:rgba(255,255,255,.38); text-decoration:none; color:var(--deep); font:600 13px ui-sans-serif, system-ui, sans-serif; min-height:36px; }
.filter-link[aria-current="page"] { background:var(--deep); color:#fff; border-color:var(--deep); }
.status-line { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
.story-summary { max-width:44ch; }
.toolbar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin:18px 0 22px; }
.table { width:100%; border-collapse:collapse; font-family:ui-sans-serif, system-ui, sans-serif; font-size:14px; }
.table th, .table td { text-align:left; border-bottom:1px solid var(--line); padding:10px 8px; vertical-align:top; }
.story-lobby-table { table-layout:auto; }
.story-lobby-table th, .story-lobby-table td { word-break:keep-all; }
.story-lobby-table .story-lobby-status,
.story-lobby-table .story-lobby-turn,
.story-lobby-table .story-lobby-driver,
.story-lobby-table .story-lobby-updated,
.story-lobby-table .story-lobby-permission,
.story-lobby-table .story-lobby-action { white-space:nowrap; }
.story-lobby-table .story-lobby-summary { min-width:18ch; }
.story-lobby-table .story-lobby-action { width:1%; }
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
.choice { text-align:left; justify-content:flex-start; background:var(--panel); color:var(--ink); border-color:var(--line); white-space:normal; align-items:flex-start; gap:4px; }
.choice strong { margin-right:8px; color:var(--accent); }
.archived-choice { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.25); padding:10px 12px; }
.input-panel { scroll-margin-top:18px; display:grid; gap:12px; }
.mobile-action-dock { display:none; }
.form-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:12px; }
.panel { border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.35); padding:16px; margin-bottom:14px; }
.story-layout article > .panel, .turn-card .panel { background:rgba(255,255,255,.46); }
.status-panel { border-left:4px solid var(--accent); background:rgba(255,250,240,.72); }
.story-progress { display:grid; gap:10px; margin-bottom:0; }
.story-progress-head { display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; }
.story-progress-head strong { font:700 16px ui-sans-serif, system-ui, sans-serif; }
.story-progress-steps { list-style:none; margin:0; padding:0; display:flex; flex-wrap:wrap; gap:6px; }
.story-progress-steps li { min-height:28px; display:inline-flex; align-items:center; border:1px solid var(--line); border-radius:999px; padding:3px 10px; background:rgba(255,255,255,.5); font:12px ui-sans-serif, system-ui, sans-serif; color:var(--muted); text-transform:lowercase; }
.story-progress-steps li.is-active { border-color:rgba(184,51,45,.35); color:var(--ink); background:rgba(184,51,45,.08); }
.story-progress-message { margin:0; font:15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); }
.story-progress-meta { margin:0; word-break:break-word; }
.story-progress-actions { margin-top:0; }
.story-progress [data-story-refresh] { width:auto; }
.input-panel textarea:disabled,
.story-layout .panel textarea:disabled,
.story-layout .panel select:disabled,
.story-layout .panel button:disabled { opacity:.58; cursor:not-allowed; }
.input-panel textarea:disabled { background:rgba(255,255,255,.6); color:var(--muted); }
.story-layout [aria-busy="true"] .story-progress { border-left-color:var(--warn); }
.panel h2, .panel h3 { margin-top:0; border:0; padding-top:0; font-family:ui-sans-serif, system-ui, sans-serif; }
.panel ul { padding-left:20px; }
.muted { color:var(--muted); font-family:ui-sans-serif, system-ui, sans-serif; font-size:13px; }
.error { color:var(--accent); font-family:ui-sans-serif, system-ui, sans-serif; }
.empty-state { padding:18px; border:1px dashed var(--line); border-radius:6px; background:rgba(255,255,255,.26); }
.failed-job-meta { display:grid; gap:6px; }
@media (max-width:820px){
  .shell{padding:16px 14px 176px;}
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
  .toolbar > *, .driver-actions > *{flex:1 1 auto; min-width:0;}
  button, .button{width:100%; min-height:48px;}
  .table, .table tbody, .table tr, .table td{display:block; width:100%;}
  .table thead{display:none;}
  .table tr{border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.35); margin:0 0 12px; padding:10px;}
  .table td{border:0; padding:6px 4px;}
  .story-lobby-table th, .story-lobby-table td{white-space:normal; word-break:keep-all;}
  .story-lobby-table .story-lobby-action{width:auto;}
  .mobile-action-dock{position:fixed; left:0; right:0; bottom:0; z-index:10; display:grid; grid-template-columns:1fr 1fr; gap:8px; padding:10px 12px calc(14px + env(safe-area-inset-bottom)); background:rgba(247,244,237,.94); border-top:1px solid var(--line); box-shadow:var(--shadow); backdrop-filter:blur(12px);}
  .mobile-action-dock a{min-height:48px;}
}
@media (max-width:960px){ .story-layout{grid-template-columns:1fr;} .table{font-size:13px;} }
</style>
</head>
<body>
<main class="shell">
<div class="top"><a class="brand" href="{{.Base}}/">World Harness</a><div class="nav">{{if .StoryEnabled}}<a href="{{.Base}}/stories">스토리</a>{{end}}<a href="{{.Base}}/packs/lumen-federation/">세계관</a>{{with .User}}{{if eq .Role "admin"}}<a href="{{$.Base}}/admin/users">Admin</a>{{end}}<form class="nav-form" method="post" action="{{$.Base}}/logout"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><button class="link-button" type="submit">Logout</button></form>{{else}}<span>{{$.PageTitle}}</span>{{end}}</div></div>
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
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
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
  <form class="nav-form" method="post" action="{{.Base}}/stories/import/hector"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><button class="secondary" type="submit">헥터 import</button></form>
</div>
<div class="filter-bar" role="tablist" aria-label="story filters">
  <a class="filter-link" href="{{.Base}}/stories" {{if eq .Filter "all"}}aria-current="page"{{end}}>all</a>
  <a class="filter-link" href="{{.Base}}/stories?filter=active" {{if eq .Filter "active"}}aria-current="page"{{end}}>active</a>
  <a class="filter-link" href="{{.Base}}/stories?filter=mine" {{if eq .Filter "mine"}}aria-current="page"{{end}}>mine</a>
  <a class="filter-link" href="{{.Base}}/stories?filter=watch" {{if eq .Filter "watch"}}aria-current="page"{{end}}>watch</a>
  <a class="filter-link" href="{{.Base}}/stories?filter=archived" {{if eq .Filter "archived"}}aria-current="page"{{end}}>archived</a>
  <a class="filter-link" href="{{.Base}}/stories?filter=imported" {{if eq .Filter "imported"}}aria-current="page"{{end}}>imported</a>
</div>
<table class="table story-lobby-table">
  <thead><tr><th class="story-lobby-title">제목</th><th class="story-lobby-status">상태</th><th class="story-lobby-turn">Turn</th><th class="story-lobby-driver">진행자</th><th class="story-lobby-summary">현재 상황</th><th class="story-lobby-updated">업데이트</th><th class="story-lobby-permission">권한</th><th class="story-lobby-action"></th></tr></thead>
  <tbody>
  {{range .Stories}}
    <tr>
      <td class="story-lobby-title"><strong>{{.Title}}</strong><div class="muted">{{.ID}}{{if .Imported}} · imported{{end}}</div></td>
      <td class="story-lobby-status"><div class="status-line"><span class="badge">{{.Status}}</span><span class="badge">{{.Phase}}</span></div><div class="muted">{{.StatusLabel}}</div></td>
      <td class="story-lobby-turn">{{.Turn}}</td>
      <td class="story-lobby-driver">{{.DriverLabel}}</td>
      <td class="story-lobby-summary"><div class="story-summary">{{.Summary}}</div></td>
      <td class="story-lobby-updated muted">{{.Updated}}</td>
      <td class="story-lobby-permission">{{.Permission}}</td>
      <td class="story-lobby-action"><a class="button secondary" href="{{storyURL $.Base .ID}}">입장</a></td>
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
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
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
<div id="story-room" data-story-room data-story-id="{{.Story.ID}}" data-status-url="{{.StatusURL}}" data-current-turn="{{.Story.CurrentTurn}}" data-initial-processing="{{if .IsProcessing}}true{{else}}false{{end}}">
<div class="story-header">
  <div>
    <h1>{{.Story.Title}}</h1>
    <div class="toolbar">
      <span class="badge">{{.Story.Status}}</span><span class="badge">{{.Story.Phase}}</span><span class="badge">Turn {{.Story.CurrentTurn}}</span><span class="badge">driver {{.DriverLabel}}</span>
    </div>
  </div>
  <div class="driver-actions">
    {{if .CanClaim}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="claim"><button>진행권 받기</button></form>{{end}}
    {{if .CanRelease}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/driver"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="release"><button class="secondary">open으로 나가기</button></form>{{end}}
  </div>
</div>
{{if .IsProcessing}}<div class="panel status-panel"><strong>GM 생성 중</strong><p>요청 이벤트가 접수되었습니다. Codex/GM worker가 장면을 생성하는 동안 추가 진행 입력은 잠시 막힙니다.</p><p class="muted">active job: {{.Story.ActiveJobID}} · phase: {{.Story.Phase}}</p></div>{{end}}
{{if .FailedJob}}{{if .FailedJob.CanRecover}}<div class="panel status-panel"><strong>GM 생성 실패</strong><p>현재 job이 실패 상태입니다. 복구를 진행하거나 취소할 수 있습니다.</p><p class="muted">active job: {{.Story.ActiveJobID}} · actor: {{.FailedJob.ActorLabel}}</p><div class="toolbar"><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/recover"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="resume"><button>resume</button></form><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/recover"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="cancel"><button class="secondary">cancel</button></form></div></div>{{else}}<div class="panel status-panel"><strong>GM 생성 실패</strong><p>현재 job이 실패 상태입니다. 새 진행 입력은 실패 job 처리 후 가능합니다.</p><p class="muted">active job: {{.Story.ActiveJobID}}</p></div>{{end}}{{end}}
{{if .ExportedBundle}}<div class="panel status-panel"><strong>Export handoff</strong><p>Bundle exported to <code>{{.ExportedBundle}}</code>.</p><p class="muted">Draft creation is pending/manual via the admin writer path. An admin can now create the draft with story export-draft through the writer path.</p><p class="muted">Target draft: <code>{{.ExportDraftTarget}}</code> · status: <span class="badge">{{if .ExportedStatus}}{{.ExportedStatus}}{{else}}draft_pending{{end}}</span></p></div>{{end}}
{{if .RecoveryStatus}}<div class="panel status-panel"><strong>Store recovery</strong><p>Recovery status: <span class="badge">{{.RecoveryStatus}}</span></p>{{if .RecoveryMessage}}<p>{{.RecoveryMessage}}</p>{{end}}<p class="muted">Checked files: {{range $i, $v := .RecoveryChecked}}{{if $i}}, {{end}}<code>{{$v}}</code>{{end}}</p>{{if .RecoveryRepaired}}<p class="muted">Repaired items: {{range $i, $v := .RecoveryRepaired}}{{if $i}}, {{end}}<code>{{$v}}</code>{{end}}</p>{{else}}<p class="muted">No file tails needed repair.</p>{{end}}{{if .RecoveryLockRemoved}}<p class="muted">Stale lock.json was removed.</p>{{end}}</div>{{end}}
{{if .HasTurns}}<nav class="turn-nav" aria-label="turn list">
  {{range .Turns}}<a href="#turn-{{.TurnID}}">Turn {{.TurnID}}</a>{{end}}
</nav>{{end}}
<div class="story-layout">
  <article>
    {{range .Turns}}
      <details class="turn-card" id="turn-{{.TurnID}}" {{if eq .TurnID $.LatestTurnID}}open{{end}}>
        <summary><span class="turn-title"><span>Turn {{.TurnID}}</span><span class="muted">{{.SceneTitle}}</span></span></summary>
        <div class="muted">{{.CreatedAt}} · {{.Source}}</div>
        <div class="scene">{{.SceneBody}}</div>
        <div class="panel"><strong>현재 상황</strong><p>{{.CurrentSituation}}</p></div>
        {{if .RevealedFacts}}<div class="panel"><strong>확인된 정보</strong><ul>{{range .RevealedFacts}}<li>{{.}}</li>{{end}}</ul></div>{{end}}
        {{$turnID := .TurnID}}{{if .Choices}}<div class="panel"><strong>{{if eq .TurnID $.LatestTurnID}}다음 갈림길{{else}}기록된 선택지{{end}}</strong><div class="choice-list">{{range .Choices}}{{if eq $turnID $.LatestTurnID}}<form method="post" action="{{$.Base}}/stories/{{$.Story.ID}}/input" data-story-submit data-story-submit-kind="choice"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><input type="hidden" name="turn_id" value="{{$.LatestTurnID}}"><input type="hidden" name="idempotency_key" value="{{idem}}"><input type="hidden" name="choice_id" value="{{.ID}}"><button class="choice" type="submit" {{if not $.CanDrive}}disabled{{end}}><strong>{{.ID}}</strong>{{.Text}}</button>{{if .RiskHint}}<div class="muted">{{.RiskHint}}</div>{{end}}</form>{{else}}<div class="archived-choice"><strong>{{.ID}}</strong> {{.Text}}{{if .RiskHint}}<div class="muted">{{.RiskHint}}</div>{{end}}</div>{{end}}{{end}}</div></div>{{end}}
      </details>
    {{end}}
    <section class="turn input-panel" id="input-panel" aria-busy="{{if .Progress.IsProcessing}}true{{else}}false{{end}}" data-story-input-panel>
      <h2>직접 입력</h2>
      <div class="panel status-panel story-progress" id="story-progress" role="status" aria-live="polite" aria-atomic="true" data-story-progress data-status-url="{{.StatusURL}}" data-step-index="{{.Progress.StepIndex}}" data-step-label="{{.Progress.StepLabel}}" data-active-job-id="{{.Progress.ActiveJobID}}" data-active-job-status="{{.Progress.ActiveJobStatus}}" data-active-job-type="{{.Progress.ActiveJobType}}" data-next-poll-ms="{{.Progress.NextPollMS}}">
        <div class="story-progress-head">
          <strong data-story-progress-label>{{.Progress.StepLabel}}</strong>
          <span class="badge" data-story-progress-status>{{.Progress.StatusLabel}}</span>
        </div>
        <ol class="story-progress-steps" aria-hidden="true">
          <li data-story-step="queued">queued</li>
          <li data-story-step="generating">generating</li>
          <li data-story-step="applying">applying</li>
          <li data-story-step="ready">ready</li>
          <li data-story-step="failed">failed</li>
        </ol>
        <p class="story-progress-message" data-story-progress-message>{{.Progress.ProgressMessage}}</p>
        <p class="muted story-progress-meta" data-story-progress-meta>active job: <code data-story-progress-job-id>{{.Progress.ActiveJobID}}</code>{{if .Progress.ActiveJobType}} · type: <span data-story-progress-job-type>{{.Progress.ActiveJobType}}</span>{{end}}{{if .Progress.ActiveJobStatus}} · status: <span data-story-progress-job-status>{{.Progress.ActiveJobStatus}}</span>{{end}}{{if gt .Progress.ActiveJobTurnID 0}} · turn <span data-story-progress-turn>{{.Progress.ActiveJobTurnID}}</span>{{end}}{{if .Progress.PendingQuestions}} · queued questions: <span data-story-progress-pending-count>{{len .Progress.PendingQuestions}}</span>{{end}}</p>
        <div class="toolbar story-progress-actions"><button type="button" class="secondary" hidden data-story-refresh>새 내용 표시</button></div>
      </div>
      {{if .CanDrive}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/input" class="panel" data-story-submit data-story-submit-kind="input">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <input type="hidden" name="turn_id" value="{{.LatestTurnID}}">
        <input type="hidden" name="idempotency_key" value="{{idem}}">
        <div class="form-grid"><div><label class="muted">Mode</label><select name="mode"><option value="action">행동</option><option value="dialogue">대사</option><option value="narration">서술 보정</option><option value="question">질문</option></select></div></div>
        <textarea name="custom_text" data-story-custom-textarea placeholder="플레이어 캐릭터가 시도하는 행동/대사/서술/질문"></textarea>
        <div class="toolbar"><button type="submit">제출</button></div>
      </form>
      {{else}}{{if .IsProcessing}}<p class="muted">GM 생성 중입니다. 완료되면 새 내용 표시 버튼으로 최신 턴을 갱신할 수 있습니다.</p>{{else}}{{if .CanClaim}}<p class="muted">현재 진행권이 open 상태입니다. 진행권을 받은 뒤 입력할 수 있습니다.</p>{{else}}<p class="muted">현재 {{.DriverLabel}}가 진행 중입니다. 진행 입력은 비활성화되어 있습니다.</p>{{end}}{{end}}{{end}}
      <h2 id="qa">질문</h2>
      {{if .CanDrive}}<p class="muted">질문은 직접 입력에서 question 모드를 선택해 제출할 수 있습니다.</p>{{else}}{{if .CanQuestion}}
      <form method="post" action="{{.Base}}/stories/{{.Story.ID}}/question" class="panel" data-story-submit data-story-submit-kind="question">
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
        <input type="hidden" name="turn_id" value="{{.LatestTurnID}}">
        <input type="hidden" name="idempotency_key" value="{{idem}}">
        <textarea name="question" data-story-question-textarea placeholder="현재 상황, 인물, 단서, 설정, 선택지 의미를 묻는 비진행 질문"></textarea>
        <div class="toolbar"><button class="secondary" type="submit">질문 제출</button></div>
      </form>
      {{else}}{{if .IsProcessing}}<p class="muted">GM 생성 중에는 질문 제출도 잠시 막습니다.</p>{{else}}<p class="muted">completed/archived/deleted room에서는 새 질문을 받지 않습니다.</p>{{end}}{{end}}{{end}}
      {{range .QA}}<div class="panel"><div class="muted">{{.CreatedAt}} · Turn {{.TurnID}}</div><strong>Q. {{.Question}}</strong><p>A. {{.Answer}}</p></div>{{end}}
    </section>
  </article>
  <aside>
    <div class="panel"><h3>현재 상태</h3><p><strong>{{.State.Location}}</strong></p><p class="muted">인물: {{range .State.ActiveCharacters}}<span class="badge">{{.}}</span> {{end}}</p></div>
    <div class="panel"><h3>확인된 정보</h3><ul>{{range .State.Facts}}<li>{{.}}</li>{{end}}</ul></div>
    <div class="panel"><h3>열린 실마리</h3><ul>{{range .State.OpenThreads}}<li>{{.}}</li>{{end}}</ul></div>
    <div class="panel"><h3>위험</h3><ul>{{range .State.Risks}}<li>{{.}}</li>{{end}}</ul></div>
    {{if .IsAdmin}}<div class="panel"><h3>Admin</h3><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="update"><label class="muted">Status</label><select name="status"><option value="">변경 없음</option><option>active</option><option>paused</option><option>completed</option><option>archived</option></select><label class="muted">Active driver user id</label><input name="active_driver_id" placeholder="{{.DriverLabel}}"><div class="toolbar"><button>적용</button></div></form><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="update"><input type="hidden" name="active_driver_id" value="__open__"><button class="secondary">open으로 변경</button></form>{{if .CanAdminMutate}}{{with .LatestTurn}}<form method="post" action="{{$.Base}}/stories/{{$.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><input type="hidden" name="action" value="edit_turn"><label class="muted">Edit current turn {{$.LatestTurnID}}</label><label class="muted">Scene body</label><textarea name="scene_body">{{.SceneBody}}</textarea><label class="muted">Current situation</label><textarea name="current_situation">{{.CurrentSituation}}</textarea><div class="toolbar"><button class="secondary">save turn edit</button></div></form>{{end}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="rollback_turn"><label class="muted">Rollback to turn</label><select name="turn_id">{{range .Turns}}<option value="{{.TurnID}}" {{if eq .TurnID $.LatestTurnID}}selected{{end}}>Turn {{.TurnID}}</option>{{end}}</select><div class="toolbar"><button class="secondary">rollback</button></div></form>{{else if .IsProcessing}}<p class="muted">GM 생성 중에는 편집과 롤백을 막습니다.</p>{{end}}<div class="toolbar">{{if or (eq .Story.Status "archived") (eq .Story.Status "deleted")}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="restore"><button>restore</button></form>{{else}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="archive"><button>archive</button></form>{{end}}{{if ne .Story.Status "deleted"}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="delete"><button class="secondary">delete</button></form>{{end}}<form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="export_bundle"><button class="secondary">export bundle</button></form><form method="post" action="{{.Base}}/stories/{{.Story.ID}}/admin"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><input type="hidden" name="action" value="recover_store"><button class="secondary">recover store</button></form></div></div>{{end}}
  </aside>
</div>
{{if .HasTurns}}<div class="mobile-action-dock"><a class="button secondary" href="#turn-{{.LatestTurnID}}">최신 턴</a><a class="button" href="#input-panel">입력</a></div>{{else}}<div class="mobile-action-dock"><a class="button" href="#input-panel">입력</a></div>{{end}}
<script defer src="{{.Base}}/assets/story-room.js"></script>
</div>
{{end}}`

// storyRoomAssetJS is the only story-room client implementation.
// It is served from /assets/story-room.js and loaded by the template via a same-origin script tag.
const storyRoomAssetJS = `(() => {
  const root = document.querySelector('[data-story-room]');
  if (!root) return;
  const progress = root.querySelector('[data-story-progress]');
  if (!progress) return;
  const refreshButton = progress.querySelector('[data-story-refresh]');
  const statusLabel = progress.querySelector('[data-story-progress-label]');
  const statusBadge = progress.querySelector('[data-story-progress-status]');
  const messageNode = progress.querySelector('[data-story-progress-message]');
  const jobIdNode = progress.querySelector('[data-story-progress-job-id]');
  const jobTypeNode = progress.querySelector('[data-story-progress-job-type]');
  const jobStatusNode = progress.querySelector('[data-story-progress-job-status]');
  const turnNode = progress.querySelector('[data-story-progress-turn]');
  const pendingNode = progress.querySelector('[data-story-progress-pending-count]');
  const stepNodes = Array.from(progress.querySelectorAll('[data-story-step]'));
  const forms = Array.from(root.querySelectorAll('form[data-story-submit]'));
  const inputPanel = root.querySelector('[data-story-input-panel]');
  const storyTurn = Number(root.dataset.currentTurn || 0);
  const initialControlState = new WeakMap();
  let pollTimer = null;
  let activeTask = null;

  function captureInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    if (!initialControlState.has(control)) {
      initialControlState.set(control, {
        disabled: control.disabled,
        ariaDisabled: control.getAttribute('aria-disabled'),
      });
    }
  }

  function restoreInitialControlState(control) {
    if (!control || control.type === 'hidden') return;
    const initial = initialControlState.get(control);
    if (!initial) return;
    control.disabled = initial.disabled;
    if (initial.disabled) {
      control.setAttribute('aria-disabled', initial.ariaDisabled ?? 'true');
    } else if (initial.ariaDisabled === null) {
      control.removeAttribute('aria-disabled');
    } else {
      control.setAttribute('aria-disabled', initial.ariaDisabled);
    }
    if (control.tagName === 'BUTTON' && control.dataset.storyOriginalHtml) {
      control.innerHTML = control.dataset.storyOriginalHtml;
      delete control.dataset.storyOriginalHtml;
    }
  }

  forms.forEach((form) => {
    form.querySelectorAll('button, input, select, textarea').forEach(captureInitialControlState);
  });

  function setStep(stepLabel) {
    stepNodes.forEach((node) => {
      const active = node.dataset.storyStep === stepLabel;
      node.toggleAttribute('aria-current', active);
      node.classList.toggle('is-active', active);
    });
  }

  function setBusy(busy) {
    root.setAttribute('aria-busy', busy ? 'true' : 'false');
    progress.setAttribute('aria-busy', busy ? 'true' : 'false');
    if (inputPanel) {
      inputPanel.setAttribute('aria-busy', busy ? 'true' : 'false');
    }
    forms.forEach((form) => {
      form.querySelectorAll('button, input, select, textarea').forEach((control) => {
        if (control.type === 'hidden') return;
        captureInitialControlState(control);
        if (busy) {
          control.disabled = true;
          control.setAttribute('aria-disabled', 'true');
          if (control.tagName === 'BUTTON' && form.dataset.storySubmitKind !== 'choice') {
            if (!control.dataset.storyOriginalHtml) control.dataset.storyOriginalHtml = control.innerHTML;
            control.innerHTML = '처리 중...';
          }
        } else {
          restoreInitialControlState(control);
        }
      });
    });
  }

  function showRefresh(visible) {
    if (!refreshButton) return;
    refreshButton.hidden = !visible;
  }

  async function readErrorMessage(response) {
    const contentType = (response.headers.get('content-type') || '').toLowerCase();
    if (contentType.includes('application/json')) {
      const payload = await response.json().catch(() => null);
      if (payload && payload.error) {
        return payload.error;
      }
      return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다';
    }
    const text = await response.text().catch(() => '');
    const snippet = text.trim().replace(/\s+/g, ' ').slice(0, 160);
    return 'HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : '');
  }

  function renderStatus(payload) {
    progress.dataset.stepIndex = String(payload.step_index ?? 3);
    progress.dataset.stepLabel = payload.step_label || 'ready';
    progress.dataset.activeJobId = payload.active_job_id || '';
    progress.dataset.activeJobStatus = payload.active_job_status || '';
    progress.dataset.activeJobType = payload.active_job_type || '';
    progress.dataset.nextPollMs = String(payload.next_poll_ms || 0);
    if (statusLabel) statusLabel.textContent = payload.step_label || 'ready';
    if (statusBadge) statusBadge.textContent = payload.status_label || '';
    if (messageNode) messageNode.textContent = payload.progress_message || '';
    if (jobIdNode) jobIdNode.textContent = payload.active_job_id || '';
    if (jobTypeNode) jobTypeNode.textContent = payload.active_job_type || '';
    if (jobStatusNode) jobStatusNode.textContent = payload.active_job_status || '';
    if (turnNode) turnNode.textContent = payload.active_job_turn_id ? String(payload.active_job_turn_id) : '';
    if (pendingNode) pendingNode.textContent = payload.pending_questions ? String(payload.pending_questions.length) : '';
    setStep(payload.step_label || (payload.is_processing ? 'generating' : 'ready'));
    setBusy(Boolean(payload.is_processing));
  }

  async function pollStatus() {
    if (!activeTask || !activeTask.status_url) return;
    try {
      const response = await fetch(activeTask.status_url, {
        headers: {
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        credentials: 'include',
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response));
      }
      const payload = await response.json();
      renderStatus(payload);
      const nextPoll = Number(payload.next_poll_ms || activeTask.next_poll_ms || 2500);
      if (payload.is_processing) {
        pollTimer = window.setTimeout(pollStatus, Math.max(1000, nextPoll));
        return;
      }
      activeTask = null;
      showRefresh(true);
      if (payload.current_turn > storyTurn || payload.active_job_status === 'completed' || payload.step_label === 'ready') {
        if (messageNode) messageNode.textContent = payload.progress_message || '새 내용이 준비되었습니다. 새 내용 표시를 눌러 갱신하세요.';
      }
    } catch (error) {
      if (messageNode) messageNode.textContent = '상태를 다시 불러오지 못했습니다. 잠시 후 다시 시도해 주세요.';
      pollTimer = window.setTimeout(pollStatus, 2500);
    }
  }

  async function submitForm(form) {
    if (pollTimer) {
      window.clearTimeout(pollTimer);
      pollTimer = null;
    }
    const data = new FormData(form);
    const requestPayload = Object.fromEntries(data.entries());
    const actionURL = new URL(form.action, window.location.href);
    const requestURL = actionURL.origin === window.location.origin ? actionURL.pathname + actionURL.search : form.action;
    setBusy(true);
    if (messageNode) messageNode.textContent = '제출을 보냈습니다. 서버 응답을 기다립니다.';
    showRefresh(false);
    try {
      const response = await fetch(requestURL, {
        method: (form.method || 'post').toUpperCase(),
        body: JSON.stringify(requestPayload),
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          'X-CSRF-Token': form.querySelector('input[name="csrf_token"]')?.value || '',
        },
      });
      const contentType = (response.headers.get('content-type') || '').toLowerCase();
      const rawBody = await response.text();
      let responsePayload = null;
      if (contentType.includes('application/json')) {
        try {
          responsePayload = JSON.parse(rawBody);
        } catch (parseError) {
          responsePayload = null;
        }
      }
      if (!response.ok || !responsePayload) {
        if (responsePayload && responsePayload.error) {
          throw new Error(responsePayload.error);
        }
        const snippet = rawBody.trim().replace(/\s+/g, ' ').slice(0, 160);
        throw new Error('HTTP ' + response.status + ' - 제출 응답을 JSON으로 받지 못했습니다' + (snippet ? ': ' + snippet : ''));
      }
      activeTask = {
        status_url: responsePayload.status_url,
        next_poll_ms: responsePayload.next_poll_ms || 2500,
        turn_id: responsePayload.turn_id || 0,
        job_id: responsePayload.job_id || '',
        job_type: responsePayload.job_type || '',
      };
      renderStatus(responsePayload);
      pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
    } catch (error) {
      setBusy(false);
      showRefresh(false);
      if (messageNode) messageNode.textContent = error.message || '제출 처리에 실패했습니다.';
    }
  }

  root.addEventListener('submit', (event) => {
    const form = event.target.closest('form[data-story-submit]');
    if (!form) return;
    event.preventDefault();
    submitForm(form);
  });

  if (refreshButton) {
    refreshButton.addEventListener('click', () => window.location.reload());
  }

  setStep(progress.dataset.stepLabel || (root.dataset.initialProcessing === 'true' ? 'generating' : 'ready'));
  setBusy(root.dataset.initialProcessing === 'true');
  if (root.dataset.initialProcessing === 'true') {
    activeTask = {
      status_url: progress.dataset.statusUrl || root.dataset.statusUrl,
      next_poll_ms: Number(progress.dataset.nextPollMs || 2500),
    };
    pollTimer = window.setTimeout(pollStatus, Math.max(1000, activeTask.next_poll_ms || 2500));
  }
})();`

const adminUsersTemplate = `{{define "content"}}
<h1>Admin Users</h1>
<div class="panel">
  <h2>Create user</h2>
  <form method="post" class="form-grid">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
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
    <form method="post" class="toolbar"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><input type="hidden" name="action" value="update"><input type="hidden" name="id" value="{{.id}}"><select name="role"><option {{if eq .role "friend"}}selected{{end}}>friend</option><option {{if eq .role "admin"}}selected{{end}}>admin</option></select><select name="status"><option {{if eq .status "active"}}selected{{end}}>active</option><option {{if eq .status "disabled"}}selected{{end}}>disabled</option></select><button class="secondary">update</button></form>
    <form method="post" class="toolbar"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><input type="hidden" name="action" value="reset"><input type="hidden" name="id" value="{{.id}}"><input name="password" type="password" placeholder="new password"><button class="secondary">reset password</button></form>
    <form method="post" class="toolbar"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><input type="hidden" name="action" value="revoke"><input type="hidden" name="id" value="{{.id}}"><button class="danger">revoke sessions</button></form>
  </td>
</tr>{{end}}</tbody>
</table>
{{end}}`
