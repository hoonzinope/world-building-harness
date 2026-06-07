package harness

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
	StoryID                     string                      `json:"story_id"`
	Status                      string                      `json:"status"`
	Phase                       string                      `json:"phase"`
	CurrentTurn                 int                         `json:"current_turn"`
	ActiveJobID                 string                      `json:"active_job_id,omitempty"`
	ActiveJobType               string                      `json:"active_job_type,omitempty"`
	ActiveJobStatus             string                      `json:"active_job_status,omitempty"`
	ActiveJobTurnID             int                         `json:"active_job_turn_id,omitempty"`
	LastCompletedJobID          string                      `json:"last_completed_job_id,omitempty"`
	LastCompletedJobType        string                      `json:"last_completed_job_type,omitempty"`
	LastCompletedJobTurnID      int                         `json:"last_completed_job_turn_id,omitempty"`
	LastCompletedJobStatus      string                      `json:"last_completed_job_status,omitempty"`
	LastCompletedJobCompletedAt string                      `json:"last_completed_job_completed_at,omitempty"`
	IsProcessing                bool                        `json:"is_processing"`
	CanDrive                    bool                        `json:"can_drive"`
	CanQuestion                 bool                        `json:"can_question"`
	StatusLabel                 string                      `json:"status_label"`
	ProgressMessage             string                      `json:"progress_message"`
	StepIndex                   int                         `json:"step_index"`
	StepLabel                   string                      `json:"step_label"`
	NextPollMS                  int                         `json:"next_poll_ms"`
	JobStartedAt                string                      `json:"job_started_at,omitempty"`
	JobCompletedAt              string                      `json:"job_completed_at,omitempty"`
	JobErrorCode                string                      `json:"job_error_code,omitempty"`
	JobErrorMessage             string                      `json:"job_error_message,omitempty"`
	PendingQuestions            []storyProgressQuestionView `json:"pending_questions,omitempty"`
	HasProgressMeta             bool                        `json:"-"`
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
		md:           goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(parser.WithAutoHeadingID()), goldmark.WithRendererOptions(html.WithUnsafe())),
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
