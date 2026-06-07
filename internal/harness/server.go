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
	MetaLine    string
	Summary     string
	Updated     string
	MetaLabels  []string
	Imported    bool
	IsMine      bool
	IsWatch     bool
	IsArchived  bool
	IsActive    bool
	CanDrive    bool
	DriverLabel string
	Permission  string
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
	HasProgressMeta  bool                        `json:"-"`
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
	return u != nil && m.Status == "active" && m.Phase == "waiting_for_choice" && (u.Role == "admin" || u.ID == m.ActiveDriverID)
}

func canQuestionStory(m storyManifest) bool {
	return (m.Status == "active" || m.Status == "paused") && m.Phase == "waiting_for_choice"
}

func friendlyDriverLabel(m storyManifest, u *authUser) string {
	if m.ActiveDriverID == "" {
		return "비어 있음"
	}
	if u != nil && u.ID == m.ActiveDriverID {
		return "나"
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
		return "참여 가능"
	case u != nil && canQuestionStory(m):
		return "질문 가능"
	default:
		return "읽기 전용"
	}
}

func storyLobbyMetaLabels(m storyManifest, u *authUser) []string {
	labels := make([]string, 0, 3)
	if m.SourceDraftPath != "" {
		labels = append(labels, "가져온 스토리")
	}
	if u != nil && (m.CreatedBy == u.ID || m.ActiveDriverID == u.ID) {
		labels = append(labels, "내 스토리")
	}
	if (m.Status == "active" || m.Status == "paused") && !canDriveStory(m, u) {
		labels = append(labels, "관전")
	}
	return labels
}

func storyLobbyMetaLine(m storyManifest, u *authUser) string {
	parts := append([]string{}, storyLobbyMetaLabels(m, u)...)
	parts = append(parts, fmt.Sprintf("턴 %d", m.CurrentTurn))
	parts = append(parts, "진행자 "+friendlyDriverLabel(m, u))
	return strings.Join(parts, " · ")
}

func storyLobbyPhaseLabel(phase string) string {
	if phase == "waiting_for_choice" {
		return "응답 대기"
	}
	return friendlyStoryPhaseLabel(phase)
}

func storyLobbyUpdatedAt(updatedAt string) string {
	return storyTimestampKST(updatedAt, "업데이트 시각 확인 불가")
}

func storyTimestampKST(timestamp, fallback string) string {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return fallback
	}
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return fallback
	}
	return parsed.In(time.FixedZone("KST", 9*60*60)).Format("2006.01.02 15:04")
}

func friendlyStoryStatusLabel(status string) string {
	switch status {
	case "active":
		return "진행 중"
	case "paused":
		return "일시 정지"
	case "completed":
		return "완료"
	case "archived":
		return "보관됨"
	case "deleted":
		return "삭제됨"
	case "setup":
		return "준비 중"
	default:
		return status
	}
}

func friendlyStoryPhaseLabel(phase string) string {
	switch phase {
	case "waiting_for_choice":
		return "입력 대기"
	case "gm_generating":
		return "GM 생성 중"
	case "validating_output":
		return "검증 중"
	case "applying_turn":
		return "반영 중"
	case "failed_waiting_retry":
		return "실패/복구 대기"
	default:
		return phase
	}
}

func friendlyStoryProgressStepLabel(step string) string {
	switch step {
	case "queued":
		return "대기열"
	case "generating":
		return "생성 중"
	case "applying":
		return "반영 중"
	case "ready":
		return "입력 가능"
	case "failed":
		return "실패"
	default:
		return step
	}
}

func friendlyStoryEventKindLabel(kind string) string {
	switch kind {
	case "setup":
		return "설정"
	case "choice":
		return "선택"
	case "custom":
		return "입력"
	case "question":
		return "질문"
	default:
		return kind
	}
}

func friendlyStatusLabel(m storyManifest) string {
	status := friendlyStoryStatusLabel(m.Status)
	phase := friendlyStoryPhaseLabel(m.Phase)
	if phase == "" {
		return status
	}
	if status == "" {
		return phase
	}
	return phase + " · " + status
}

func storyTurnTitle(turnID int, title, fallback string) string {
	title = strings.TrimSpace(title)
	if title == fmt.Sprintf("Turn %d", turnID) {
		return fallback
	}
	return title
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
	if s.pathRequiresAuth(path) {
		if s.auth == nil {
			http.NotFound(w, r)
			return
		}
		u, ok := s.requireAuth(w, r)
		if !ok {
			return
		}
		r = withUser(r, u)
	} else if u, ok := s.userFromRequest(r); ok {
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

func (s *webServer) pathRequiresAuth(path string) bool {
	switch path {
	case "/stories/new", "/stories/import/hector", "/admin/users", "/logout":
		return true
	}
	if strings.HasPrefix(path, "/stories/") {
		rest := strings.Trim(strings.TrimPrefix(path, "/stories/"), "/")
		if rest == "" {
			return false
		}
		parts := strings.Split(rest, "/")
		if len(parts) < 2 {
			return false
		}
		switch parts[1] {
		case "input", "question", "driver", "admin", "recover":
			return true
		}
	}
	return false
}

func (s *webServer) userFromRequest(r *http.Request) (*authUser, bool) {
	if s.auth == nil {
		return nil, false
	}
	c, err := r.Cookie(sessionCookieName)
	if err == nil && c.Value != "" {
		if u, err := s.auth.userForToken(c.Value); err == nil {
			return u, true
		}
	}
	return nil, false
}

func (s *webServer) requireAuth(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	if u, ok := s.userFromRequest(r); ok {
		return u, true
	}
	http.Redirect(w, r, s.base(r)+"/login", http.StatusSeeOther)
	return nil, false
}

func (s *webServer) requireLoggedInUser(w http.ResponseWriter, r *http.Request) (*authUser, bool) {
	if u := currentUser(r); u != nil {
		return u, true
	}
	if s.auth == nil {
		http.NotFound(w, r)
		return nil, false
	}
	return s.requireAuth(w, r)
}

func isAdminUser(u *authUser) bool {
	return u != nil && u.Role == "admin"
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
	if _, ok := s.requireLoggedInUser(w, r); !ok {
		return
	}
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
			MetaLine:    storyLobbyMetaLine(m, u),
			Summary:     m.LatestSummary,
			Updated:     storyLobbyUpdatedAt(m.UpdatedAt),
			MetaLabels:  storyLobbyMetaLabels(m, u),
			Imported:    m.SourceDraftPath != "",
			IsMine:      u != nil && (m.CreatedBy == u.ID || m.ActiveDriverID == u.ID),
			IsWatch:     (m.Status == "active" || m.Status == "paused") && !canDriveStory(m, u),
			IsArchived:  m.Status == "completed" || m.Status == "archived" || m.Status == "deleted",
			IsActive:    m.Status == "active",
			CanDrive:    canDriveStory(m, u),
			DriverLabel: friendlyDriverLabel(m, u),
			Permission:  friendlyPermissionLabel(m, u),
		}
		if !storyMatchesLobbyFilter(row, filter) {
			continue
		}
		rows = append(rows, row)
	}
	s.render(w, r, "스토리", storyLobbyTemplate, map[string]any{"Base": s.base(r), "User": u, "IsAnonymous": u == nil, "Stories": rows, "Filter": filter, "CSRFToken": mustCSRFToken(w, r)})
}

func (s *webServer) handleNewStory(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		s.render(w, r, "새 스토리", newStoryTemplate, map[string]any{"Base": s.base(r), "User": u, "CSRFToken": mustCSRFToken(w, r)})
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requireCSRF(w, r) {
		return
	}
	id, _, err := s.stories.importHector(u.ID)
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
	previousTurns := make([]storyTurn, 0, len(displayTurns))
	if hasTurns {
		latestTurnID = displayTurns[0].TurnID
		latestTurn = displayTurns[0]
		if len(displayTurns) > 1 {
			previousTurns = append(previousTurns, displayTurns[1:]...)
		}
	}
	progress := s.storyRoomProgressSnapshot(id, m, u)
	isProcessing := progress.IsProcessing
	canDrive := canDriveStory(m, u) && !isProcessing
	canClaim := u != nil && m.ActiveDriverID == "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canRelease := u != nil && (u.Role == "admin" || u.ID == m.ActiveDriverID) && m.ActiveDriverID != "" && m.Status == "active" && m.Phase == "waiting_for_choice" && !isProcessing
	canQuestion := u != nil && canQuestionStory(m) && !isProcessing
	canAdminMutate := isAdminUser(u) && hasTurns && !isProcessing
	driverLabel := friendlyDriverLabel(m, u)
	var failedJob *failedJobView
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		if job, err := s.stories.readJob(id, m.ActiveJobID); err == nil {
			failedJob = &failedJobView{Job: job, CanRecover: isAdminUser(u) || (u != nil && u.ID == job.ActorID), ActorLabel: job.ActorID}
		}
	}
	data := map[string]any{
		"Base":                s.base(r),
		"User":                u,
		"IsAnonymous":         u == nil,
		"Story":               m,
		"State":               st,
		"Turns":               displayTurns,
		"PreviousTurns":       previousTurns,
		"QA":                  qa,
		"CanDrive":            canDrive,
		"CanClaim":            canClaim,
		"CanRelease":          canRelease,
		"CanQuestion":         canQuestion,
		"IsAdmin":             isAdminUser(u),
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
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
	jobID, err := s.stories.submitQuestionJob(id, u, turnID, strings.TrimSpace(r.FormValue("idempotency_key")), question)
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
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
	if err := s.stories.updateDriver(id, u, r.FormValue("action")); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
}

func (s *webServer) handleStoryAdmin(w http.ResponseWriter, r *http.Request, id string) {
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
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
	switch r.FormValue("action") {
	case "update":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.adminUpdateStory(id, u.ID, r.FormValue("status"), r.FormValue("active_driver_id")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "edit_turn":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.editCurrentTurn(id, u.ID, r.FormValue("scene_body"), r.FormValue("current_situation")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "rollback_turn":
		if !isAdminUser(u) {
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
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.archiveStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "restore":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.restoreStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "delete":
		if !isAdminUser(u) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := s.stories.deleteStory(id, u.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, s.base(r)+"/stories/"+url.PathEscape(id), http.StatusSeeOther)
	case "export_bundle":
		if !isAdminUser(u) {
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
		if !isAdminUser(u) {
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
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
	u, ok := s.requireLoggedInUser(w, r)
	if !ok {
		return
	}
	if !isAdminUser(u) {
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
		CanQuestion: u != nil && canQuestionStory(m),
		StatusLabel: friendlyStatusLabel(m),
		StepIndex:   3,
		StepLabel:   "ready",
		NextPollMS:  0,
	}
	if u == nil {
		progress.ProgressMessage = "로그인하면 진행, 질문, 진행권, 관리 기능을 사용할 수 있습니다."
	} else {
		progress.ProgressMessage = "대기 중입니다. 새 입력을 제출할 수 있는 상태입니다."
	}
	if progress.CanQuestion && !progress.CanDrive {
		progress.ProgressMessage = "질문은 현재 턴에 대해 보낼 수 있습니다."
	}
	if m.Phase == "failed_waiting_retry" && m.ActiveJobID != "" {
		progress.HasProgressMeta = true
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
		progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
		return progress
	}
	if m.ActiveJobID != "" {
		progress.HasProgressMeta = true
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
				progress.ProgressMessage = "작업이 대기열에 들어갔습니다. 잠시만 기다려 주세요."
			case "running":
				progress.StepIndex = 1
				progress.StepLabel = "generating"
				progress.ProgressMessage = "GM이 장면을 생성 중입니다. 잠시만 기다려 주세요."
			case "validating", "applying":
				progress.StepIndex = 2
				progress.StepLabel = "applying"
				progress.ProgressMessage = "생성 결과를 반영하는 중입니다."
			case "failed":
				progress.StepIndex = 4
				progress.StepLabel = "failed"
				progress.ProgressMessage = "GM 작업이 실패했습니다. 복구 또는 취소가 필요합니다."
			default:
				progress.ProgressMessage = "GM 작업 상태를 확인 중입니다."
			}
			progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
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
		progress.ProgressMessage = "질문 답변을 준비 중입니다. 잠시만 기다려 주세요."
		progress.HasProgressMeta = progress.ActiveJobID != "" || progress.ActiveJobType != "" || progress.ActiveJobStatus != "" || progress.ActiveJobTurnID > 0 || len(progress.PendingQuestions) > 0
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
	data["AuthEnabled"] = s.auth != nil
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
		"sceneJournalTitle": func(turnID int, title string) string {
			return storyTurnTitle(turnID, title, "세션 기록")
		},
		"sceneIndexTitle": func(turnID int, title string) string {
			return storyTurnTitle(turnID, title, "")
		},
		"storyTurnTimestamp": func(timestamp string) string {
			return storyTimestampKST(timestamp, "시각 확인 불가")
		},
		"friendlyStoryStatusLabel":       friendlyStoryStatusLabel,
		"friendlyStoryPhaseLabel":        friendlyStoryPhaseLabel,
		"storyLobbyPhaseLabel":           storyLobbyPhaseLabel,
		"friendlyStoryProgressStepLabel": friendlyStoryProgressStepLabel,
		"friendlyStoryEventKindLabel":    friendlyStoryEventKindLabel,
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
:root { --paper:#eef2ef; --ink:#1f2523; --muted:#66716d; --line:#cdd7d2; --accent:#b43f34; --deep:#173b37; --wash:#dde7e1; --panel:#ffffff; --ok:#24684b; --warn:#9a6400; --info:#315f99; --shadow:0 16px 42px rgba(17,27,24,.12); }
* { box-sizing:border-box; }
html { scroll-behavior:smooth; }
body { margin:0; background:linear-gradient(180deg,#f4f7f6 0%,#e3e9e5 100%); color:var(--ink); font-family: ui-serif, Georgia, "Apple SD Gothic Neo", "Noto Serif KR", serif; line-height:1.65; }
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
button, .button { border:1px solid var(--deep); background:var(--deep); color:white; border-radius:6px; padding:11px 15px; min-height:44px; font:600 14px ui-sans-serif, system-ui, sans-serif; cursor:pointer; text-decoration:none; display:inline-flex; align-items:center; justify-content:center; white-space:normal; text-align:center; line-height:1.25; max-width:100%; }
button:focus-visible, .button:focus-visible, input:focus-visible, select:focus-visible, textarea:focus-visible, .link-button:focus-visible, a:focus-visible { outline:3px solid rgba(184,51,45,.45); outline-offset:2px; }
button:disabled { opacity:.45; cursor:not-allowed; }
button.secondary, .button.secondary { background:transparent; color:var(--deep); }
button.danger { border-color:var(--accent); background:var(--accent); }
.filter-bar { display:flex; gap:8px; flex-wrap:wrap; margin:12px 0 18px; }
.filter-link { display:inline-flex; align-items:center; border:1px solid var(--line); border-radius:999px; padding:6px 12px; background:rgba(255,255,255,.38); text-decoration:none; color:var(--deep); font:600 13px ui-sans-serif, system-ui, sans-serif; min-height:36px; }
.filter-link[aria-current="page"], .filter-link[aria-selected="true"] { background:var(--deep); color:#fff; border-color:var(--deep); }
.status-line { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
.story-summary { max-width:44ch; }
.toolbar { display:flex; gap:10px; flex-wrap:wrap; align-items:center; margin:18px 0 22px; }
.table { width:100%; border-collapse:collapse; font-family:ui-sans-serif, system-ui, sans-serif; font-size:14px; }
.table th, .table td { text-align:left; border-bottom:1px solid var(--line); padding:10px 8px; vertical-align:top; }
.story-lobby-list { display:grid; gap:12px; margin-top:4px; }
.story-card { display:grid; gap:14px; border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.46); padding:16px; box-shadow:0 10px 22px rgba(17,27,24,.04); }
.story-card-head { display:grid; gap:12px; }
.story-card-heading { display:grid; gap:8px; min-width:0; }
.story-card-title { margin:0; font-size:22px; line-height:1.2; word-break:keep-all; overflow-wrap:anywhere; }
.story-card-meta { display:flex; gap:8px; flex-wrap:wrap; align-items:center; font-family:ui-sans-serif, system-ui, sans-serif; color:var(--muted); font-size:13px; }
.story-card-meta .meta { font-size:13px; }
.story-card-badges { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.story-card-summary { border-left:3px solid var(--accent); background:rgba(255,255,255,.55); border-radius:4px; padding:14px 14px 14px 13px; font-size:16px; line-height:1.75; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.story-card-foot { display:flex; gap:12px; flex-wrap:wrap; align-items:center; justify-content:space-between; }
.story-card-updated { font-size:12px; }
.story-card-actions { display:flex; justify-content:flex-end; flex:0 0 auto; min-width:120px; }
.story-card-actions .button { min-width:120px; }
.badge { display:inline-flex; border:1px solid var(--line); border-radius:999px; padding:2px 8px; font:12px ui-sans-serif, system-ui, sans-serif; color:var(--muted); background:rgba(255,255,255,.35); }
.story-room-shell { display:grid; gap:18px; }
.story-room-shell,
.story-room-header > *,
.story-room-grid > *,
.current-turn-column > *,
.turn-sidebar > *,
.dossier-stack,
.dossier-panel,
.story-composer,
.story-composer-panel,
.story-progress,
.turn-timeline a,
.previous-turn,
.choice-card,
.choice-card-archived { min-width:0; }
.story-room-header { display:grid; grid-template-columns:minmax(0, 1fr) auto; gap:14px; align-items:end; padding-bottom:16px; border-bottom:1px solid var(--line); }
.story-room-headline { display:grid; gap:10px; }
.story-room-headline h1 { margin-bottom:0; }
.story-room-meta { display:flex; gap:8px; flex-wrap:wrap; font-family:ui-sans-serif, system-ui, sans-serif; }
.driver-actions { display:flex; gap:8px; flex-wrap:wrap; justify-content:flex-end; }
.story-room-grid { display:grid; grid-template-columns:240px minmax(0, 1fr) 320px; grid-template-areas:"timeline current dossier"; gap:24px; align-items:start; }
.current-turn-column { grid-area:current; display:grid; gap:14px; min-width:0; }
.turn-sidebar { grid-area:timeline; display:grid; gap:12px; min-width:0; position:sticky; top:18px; align-self:start; }
.dossier-stack { grid-area:dossier; display:grid; gap:12px; position:sticky; top:18px; }
.current-turn-panel,
.turn-timeline-panel,
.previous-turns-panel,
.dossier-panel { display:grid; gap:12px; }
.dossier-panel { display:grid; gap:10px; }
.dossier-panel form { display:grid; gap:8px; }
.dossier-panel label { line-height:1.35; }
.admin-action-grid { display:grid; grid-template-columns:repeat(2, minmax(0, 1fr)); gap:8px; }
.admin-action-grid form { width:100%; }
.admin-action-grid button { width:100%; }
.dossier-panel.panel,
.story-composer-panel { margin-bottom:0; }
.turn-timeline { display:grid; gap:8px; font-family:ui-sans-serif, system-ui, sans-serif; }
.turn-timeline-link { display:grid; gap:3px; border:1px solid var(--line); border-left:3px solid var(--deep); border-radius:6px; padding:9px 10px; background:rgba(255,255,255,.72); color:var(--ink); text-decoration:none; box-shadow:none; word-break:keep-all; overflow-wrap:anywhere; }
.turn-timeline-link:hover { border-color:rgba(23,59,55,.35); background:rgba(255,255,255,.92); }
.turn-timeline-link[aria-current="true"] { border-color:rgba(49,95,153,.45); background:rgba(49,95,153,.08); border-left-color:var(--info); }
.turn-timeline-turn { font-size:11px; color:var(--muted); }
.turn-timeline-title { font-size:13px; font-weight:600; line-height:1.35; display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; }
.previous-turns { display:grid; gap:10px; }
.previous-turn { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:0; scroll-margin-top:18px; }
.previous-turn summary { min-height:60px; cursor:pointer; display:flex; align-items:flex-start; justify-content:space-between; gap:12px; list-style:none; padding:16px 16px 14px; font:700 17px ui-sans-serif, system-ui, sans-serif; }
.previous-turn summary::-webkit-details-marker { display:none; }
.previous-turn summary::after { content:"열기"; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; border:1px solid var(--line); border-radius:999px; padding:4px 8px; flex:0 0 auto; }
.previous-turn[open] summary::after { content:"접기"; }
.previous-turn-head { display:grid; gap:6px; min-width:0; }
.previous-turn-label { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; }
.previous-turn-turn { font-size:14px; color:var(--deep); }
.previous-turn-title { font-size:18px; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.previous-turn-meta { display:flex; gap:8px; flex-wrap:wrap; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; word-break:keep-all; overflow-wrap:anywhere; }
.previous-turn-body { display:grid; gap:14px; padding:0 16px 18px; }
.current-turn-panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); box-shadow:0 14px 30px rgba(17,27,24,.08); padding:18px; scroll-margin-top:18px; }
.current-turn-header { display:grid; gap:8px; padding-bottom:12px; border-bottom:1px solid rgba(17,27,24,.08); }
.current-turn-label { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; }
.current-turn-turn { font-size:14px; color:var(--deep); }
.current-turn-title { font-size:22px; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.current-turn-meta { display:flex; gap:8px; flex-wrap:wrap; color:var(--muted); font:12px ui-sans-serif, system-ui, sans-serif; word-break:keep-all; overflow-wrap:anywhere; }
.current-turn-body { display:grid; gap:16px; padding-top:14px; }
.current-turn-flow { display:grid; gap:14px; border-left:4px solid var(--info); background:rgba(49,95,153,.05); border-radius:6px; padding:14px 14px 14px 16px; }
.turn-section { display:grid; gap:8px; padding-top:12px; border-top:1px solid rgba(17,27,24,.08); }
.turn-section:first-child { padding-top:0; border-top:0; }
.scene { white-space:pre-wrap; font-size:18px; line-height:1.86; max-width:72ch; margin:0; text-wrap:pretty; word-break:keep-all; overflow-wrap:anywhere; }
.choice-list { display:grid; gap:10px; }
.choice-card { display:grid; grid-template-columns:48px minmax(0, 1fr); text-align:left; justify-content:flex-start; background:var(--panel); color:var(--ink); border-color:var(--line); white-space:normal; align-items:flex-start; gap:12px; width:100%; padding:14px; }
.choice-card:disabled { opacity:.65; }
.choice-card-letter { width:48px; min-width:48px; height:48px; border-radius:6px; display:inline-flex; align-items:center; justify-content:center; background:rgba(23,59,55,.08); color:var(--deep); font:700 16px ui-sans-serif, system-ui, sans-serif; flex:0 0 48px; }
.choice-card-copy { display:grid; gap:4px; min-width:0; }
.choice-card-copy strong { display:block; font-size:15px; line-height:1.55; word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-risk { font:12px ui-sans-serif, system-ui, sans-serif; color:var(--accent); word-break:keep-all; overflow-wrap:anywhere; }
.choice-card-archived { display:grid; grid-template-columns:48px minmax(0, 1fr); gap:12px; border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:14px; }
.choice-card-archived .choice-card-copy { padding-top:2px; }
.choice-card-archived .choice-card-letter { background:rgba(49,95,153,.08); color:var(--info); }
.story-composer { scroll-margin-top:18px; display:grid; gap:12px; }
.story-composer-panel { display:grid; gap:14px; }
.story-choice-submit-panel { display:grid; gap:12px; }
.story-custom-input-panel { display:grid; gap:12px; }
.story-choice-submit-head,
.story-composer-panel-head { display:flex; gap:8px; flex-wrap:wrap; align-items:baseline; justify-content:space-between; }
.story-choice-submit-head strong,
.story-composer-panel-head strong { font:700 15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); }
.story-input-divider { height:1px; background:rgba(17,27,24,.08); }
.story-composer-actions { margin-top:0; }
.story-choice-submit-panel .choice-list { gap:8px; }
.story-choice-submit-panel .choice-card { background:rgba(255,255,255,.72); border-color:var(--line); }
.story-custom-input-panel textarea { min-height:120px; }
.mode-tabs { display:grid; grid-template-columns:repeat(4, minmax(0, 1fr)); gap:8px; }
.mode-tabs label { min-height:48px; display:block; cursor:pointer; position:relative; }
.mode-tabs label:focus-within { outline:3px solid rgba(184,51,45,.35); outline-offset:2px; }
.mode-tabs input { position:absolute; inset:0; opacity:0; margin:0; }
.mode-tabs span { position:relative; z-index:1; display:flex; align-items:center; justify-content:center; min-height:48px; width:100%; padding:8px 12px; border:1px solid var(--line); border-radius:6px; background:var(--panel); font:600 14px ui-sans-serif, system-ui, sans-serif; color:var(--muted); text-align:center; word-break:keep-all; box-sizing:border-box; }
.mode-tabs input:checked + span { border-color:rgba(49,95,153,.45); box-shadow:inset 0 0 0 1px rgba(49,95,153,.08); color:var(--deep); background:rgba(49,95,153,.06); }
.mode-tabs input:focus-visible + span { outline:3px solid rgba(184,51,45,.35); outline-offset:2px; }
.mobile-action-dock { display:none; }
.form-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(220px,1fr)); gap:12px; }
.panel { border:1px solid var(--line); border-radius:6px; background:var(--panel); padding:16px; margin-bottom:14px; }
.status-panel { border-left:4px solid var(--info); }
.story-progress { display:grid; gap:12px; margin-bottom:0; }
.progress-loader { display:flex; align-items:center; gap:10px; padding-bottom:10px; border-bottom:1px solid rgba(17,27,24,.08); }
.progress-loader-dot { width:12px; height:12px; border-radius:999px; border:2px solid var(--info); background:transparent; flex:0 0 auto; }
.story-progress:not([aria-busy="true"]) .progress-loader-copy { display:none; }
.story-progress[aria-busy="true"] .progress-loader-dot { background:var(--warn); border-color:var(--warn); box-shadow:0 0 0 0 rgba(154,100,0,.26); animation:loaderPulse 1.5s ease-in-out infinite; }
@keyframes loaderPulse { 0% { box-shadow:0 0 0 0 rgba(154,100,0,.26); } 70% { box-shadow:0 0 0 8px rgba(154,100,0,0); } 100% { box-shadow:0 0 0 0 rgba(154,100,0,0); } }
.progress-loader-copy { display:flex; gap:10px; align-items:center; flex-wrap:wrap; min-width:0; }
.progress-loader-copy strong { font:700 16px ui-sans-serif, system-ui, sans-serif; text-transform:lowercase; }
.story-progress[data-step-label="ready"] .progress-loader-copy strong { color:var(--ok); }
.story-progress[data-step-label="queued"] .progress-loader-copy strong { color:var(--info); }
.story-progress[data-step-label="generating"] .progress-loader-copy strong { color:var(--warn); }
.story-progress[data-step-label="applying"] .progress-loader-copy strong { color:var(--deep); }
.story-progress[data-step-label="failed"] .progress-loader-copy strong { color:var(--accent); }
.story-progress-message { margin:0; font:15px ui-sans-serif, system-ui, sans-serif; color:var(--ink); word-break:keep-all; overflow-wrap:anywhere; }
.story-progress-meta { margin:0; word-break:keep-all; overflow-wrap:anywhere; }
.story-progress-actions { margin-top:0; }
.story-progress [data-story-refresh] { width:auto; }
[hidden] { display:none !important; }
.input-panel textarea:disabled,
.story-composer-panel textarea:disabled,
.story-composer-panel input:disabled,
.story-composer-panel button:disabled,
.story-progress button:disabled,
.story-progress input:disabled { opacity:.58; cursor:not-allowed; }
.input-panel textarea:disabled { background:rgba(255,255,255,.6); color:var(--muted); }
.story-room-shell [aria-busy="true"] .story-progress { border-left-color:var(--warn); }
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
  .story-room-header{grid-template-columns:1fr; align-items:start;}
  .story-room-grid{grid-template-columns:1fr; grid-template-areas:"current" "timeline" "dossier";}
  .turn-sidebar,
  .dossier-stack{position:static;}
  .driver-actions{justify-content:flex-start;}
  .current-turn-body{display:flex; flex-direction:column;}
  .current-turn-flow{order:1;}
  .current-turn-body .scene{order:2;}
  .story-choice-submit-head,
  .story-composer-panel-head{align-items:flex-start;}
  .story-choice-submit-head span,
  .story-composer-panel-head span{width:100%;}
  .story-choice-submit-panel .choice-list,
  .story-custom-input-panel,
  .story-composer-actions{width:100%;}
  .story-choice-submit-panel .choice-card,
  .story-custom-input-panel textarea,
  .story-composer-actions > *{width:100%;}
  .scene{font-size:17px; line-height:1.72;}
  .panel{padding:14px;}
  .toolbar > *, .driver-actions > *{flex:1 1 auto; min-width:0;}
  button, .button{width:100%; min-height:48px;}
  .table, .table tbody, .table tr, .table td{display:block; width:100%;}
  .table thead{display:none;}
  .table tr{border:1px solid var(--line); border-radius:6px; background:rgba(255,255,255,.35); margin:0 0 12px; padding:10px;}
  .table td{border:0; padding:6px 4px;}
  .story-lobby-list{gap:10px;}
  .story-card{padding:14px;}
  .story-card-title{font-size:19px;}
  .story-card-summary{font-size:15px; line-height:1.7;}
  .story-card-foot{align-items:stretch;}
  .story-card-actions{width:100%; justify-content:stretch;}
  .story-card-actions .button{width:100%;}
  .mobile-action-dock{position:fixed; left:0; right:0; bottom:0; z-index:10; display:grid; grid-template-columns:1fr 1fr; gap:8px; padding:10px 12px calc(14px + env(safe-area-inset-bottom)); background:rgba(255,255,255,.94); border-top:1px solid var(--line); box-shadow:var(--shadow); backdrop-filter:blur(12px);}
  .mobile-action-dock a{min-height:48px;}
}
@media (max-width:960px){ .story-room-grid{grid-template-columns:1fr; grid-template-areas:"current" "timeline" "dossier";} .turn-sidebar, .dossier-stack{position:static;} .mode-tabs{grid-template-columns:repeat(2, minmax(0, 1fr));} .story-choice-submit-panel .choice-list{grid-template-columns:1fr;} .story-choice-submit-panel .choice-card{grid-template-columns:48px minmax(0, 1fr);} .story-custom-input-panel textarea{min-height:110px;} .table{font-size:13px;} .turn-timeline-link{max-width:100%;} }
</style>
</head>
<body>
<main class="shell">
<div class="top"><a class="brand" href="{{.Base}}/">World Harness</a><div class="nav">{{if .StoryEnabled}}<a href="{{.Base}}/stories">스토리</a>{{end}}<a href="{{.Base}}/packs/lumen-federation/">세계관</a>{{with .User}}{{if eq .Role "admin"}}<a href="{{$.Base}}/admin/users">Admin</a>{{end}}<form class="nav-form" method="post" action="{{$.Base}}/logout"><input type="hidden" name="csrf_token" value="{{$.CSRFToken}}"><button class="link-button" type="submit">Logout</button></form>{{else}}{{if .AuthEnabled}}<a href="{{.Base}}/login">로그인</a>{{end}}<span>{{$.PageTitle}}</span>{{end}}</div></div>
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
<h1>스토리</h1>
<p class="lede">세계관 문서를 읽고, 실시간 스토리 룸에서 장면 단위로 진행합니다.</p>
<div class="toolbar story-lobby-actions">
  {{if .User}}<a class="button story-lobby-primary-action" href="{{.Base}}/stories/new">새 스토리</a>{{else if .AuthEnabled}}<a class="button story-lobby-primary-action" href="{{.Base}}/login">로그인</a>{{end}}
  <a class="button secondary story-lobby-refresh-action" href="{{.Base}}/stories">새로고침</a>
</div>
{{if .IsAnonymous}}<p class="muted">로그인하지 않아도 스토리 목록과 세계관은 읽을 수 있습니다. 새 스토리 생성과 진행은 로그인 후 가능합니다.</p>{{end}}
<div class="filter-bar" role="tablist" aria-label="스토리 필터">
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "all"}}true{{else}}false{{end}}" href="{{.Base}}/stories" {{if eq .Filter "all"}}aria-current="page"{{end}}>전체</a>
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "active"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=active" {{if eq .Filter "active"}}aria-current="page"{{end}}>진행 중</a>
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "mine"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=mine" {{if eq .Filter "mine"}}aria-current="page"{{end}}>내 스토리</a>
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "watch"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=watch" {{if eq .Filter "watch"}}aria-current="page"{{end}}>관전</a>
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "archived"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=archived" {{if eq .Filter "archived"}}aria-current="page"{{end}}>보관됨</a>
  <a class="filter-link" role="tab" aria-selected="{{if eq .Filter "imported"}}true{{else}}false{{end}}" href="{{.Base}}/stories?filter=imported" {{if eq .Filter "imported"}}aria-current="page"{{end}}>가져온 스토리</a>
</div>
<div class="story-lobby-list" role="list" aria-label="스토리 세션 목록">
  {{range .Stories}}
    <article class="story-card" role="listitem">
      <div class="story-card-head">
        <div class="story-card-heading">
          <h2 class="story-card-title">{{.Title}}</h2>
          <div class="story-card-meta">
            <span class="meta">{{.MetaLine}}</span>
          </div>
        </div>
        <div class="story-card-badges">
          <span class="badge">{{friendlyStoryStatusLabel .Status}}</span>
          <span class="badge">{{storyLobbyPhaseLabel .Phase}}</span>
          <span class="badge">{{.Permission}}</span>
        </div>
      </div>
      <div class="story-card-summary">{{.Summary}}</div>
      <div class="story-card-foot">
        <div class="story-card-updated muted">업데이트 {{.Updated}}</div>
        <div class="story-card-actions"><a class="button" href="{{storyURL $.Base .ID}}">입장하기</a></div>
      </div>
    </article>
  {{else}}
    <div class="panel empty-state story-lobby-empty">아직 story room이 없습니다.</div>
  {{end}}
</div>
{{end}}`

const newStoryTemplate = `{{define "content"}}
<h1>새 스토리</h1>
<form method="post" class="panel">
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
  <div class="form-grid">
    <div><label class="muted">세계관</label><input value="lumen-federation" disabled></div>
    <div><label class="muted">제목</label><input name="title" placeholder="새 스토리"></div>
    <div><label class="muted">스타일</label><select name="style"><option value="조사극">조사극</option><option value="생존극">생존극</option><option value="행정/법정극">행정/법정극</option><option value="앙상블">앙상블</option><option value="자유">자유</option></select></div>
    <div><label class="muted">캐릭터 이름</label><input name="character_name" placeholder="캐릭터 이름"></div>
  </div>
  <label class="muted">특징 / 취향</label>
  <textarea name="traits" placeholder="캐릭터 특징, 보고 싶은 장면 압력, 피하고 싶은 톤"></textarea>
  <div class="toolbar"><button>프롤로그 생성</button><a class="button secondary" href="{{.Base}}/stories">취소</a></div>
</form>
{{end}}`

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
