package cli

import (
	"flag"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hoonzi/world-harness/internal/harness/auth"
	"github.com/hoonzi/world-harness/internal/harness/core"
	"github.com/hoonzi/world-harness/internal/harness/story"
	"github.com/hoonzi/world-harness/internal/harness/world"
)

type commonFlags struct {
	world    string
	root     string
	worldID  string
	registry string
	json     bool
	runID    string
	dryRun   bool
	verbose  bool
}

type WorldContext = world.Context
type Harness = world.Harness
type Document = world.Document
type Registry = world.Registry
type RegistryWorld = world.RegistryWorld
type storyTurn = story.Turn
type storyQuestion = story.Question

type Envelope = core.Envelope
type Issue = core.Issue
type ToolError = core.ToolError

const docSchemaVersion = core.DocSchemaVersion

func okEnvelope(command string, ctx any, runID any, data any, issues []Issue, actions []string) Envelope {
	return core.OkEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), runID, data, issues, actions)
}

func blockedEnvelope(command string, ctx any, runID any, reason string, data map[string]any, issues []Issue, actions []string) Envelope {
	return core.BlockedEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), runID, reason, data, issues, actions)
}

func failEnvelope(command string, ctx any, code, message string, details any) Envelope {
	return core.FailEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), code, message, details)
}

func emit(env Envelope) int { return core.Emit(env) }
func sha256Bytes(b []byte) string { return core.Sha256Bytes(b) }
func sha256File(path string) (string, error) { return core.Sha256File(path) }
func nowDate() string { return core.NowDate() }
func writeJSON(path string, v any) error { return core.WriteJSON(path, v) }
func writeFileAtomic(path string, b []byte, perm os.FileMode) error { return core.WriteFileAtomic(path, b, perm) }
func firstNonEmpty(values ...string) string { return core.FirstNonEmpty(values...) }
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
func ensureDir(path string) error { return core.EnsureDir(path) }
func documentSummary(doc Document) map[string]any { return world.DocumentSummary(doc) }
func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
func containsString(in []string, want string) bool {
	for _, v := range in {
		if v == want {
			return true
		}
	}
	return false
}
func headingTitle(body string) string { return world.HeadingTitle(body) }

type authStore struct{ *auth.Store }

func openAuthStore(path string) (*authStore, error) {
	s, err := auth.OpenStore(path)
	if err != nil {
		return nil, err
	}
	return &authStore{Store: s}, nil
}

type storyStore struct{ *story.Store }

func openStoryStore(root, packsRoot string) (*storyStore, error) {
	s, err := story.OpenStore(root, packsRoot)
	if err != nil {
		return nil, err
	}
	return &storyStore{Store: s}, nil
}

func (s *storyStore) recoverStory(storyID string) (story.RecoveryReport, error) {
	return s.Store.RecoverStory(storyID)
}

func (s *authStore) resetPasswordByUsername(username, password string) error {
	return s.Store.ResetPasswordByUsername(username, password)
}
func (s *authStore) revokeUserSessions(id string) error { return s.Store.RevokeUserSessions(id) }

func resolveWorld(c commonFlags, _ string) (*WorldContext, *ToolError) {
	return world.Resolve(world.CommonFlags{World: c.world, Root: c.root, WorldID: c.worldID, Registry: c.registry})
}

func initWorld(root, worldID string) (*WorldContext, error) { return world.Init(root, worldID) }
func worldStatus(ctx *WorldContext) map[string]any { return world.Status(ctx) }
func safeRel(root, rel string) (string, string, error) { return world.SafeRel(root, rel) }
func readDocument(ctx *WorldContext, rel string) (Document, error) { return world.ReadDocument(ctx, rel) }
func listDocuments(ctx *WorldContext, scope string) ([]Document, error) { return world.ListDocuments(ctx, scope) }
func searchDocuments(ctx *WorldContext, scope, query string) ([]map[string]any, error) {
	return world.SearchDocuments(ctx, scope, query)
}
func validateDocument(ctx *WorldContext, doc Document, acceptMode bool) (string, []Issue) {
	return world.ValidateDocument(ctx, doc, acceptMode)
}
func draftPath(docType, id string) (string, error) { return world.DraftPath(docType, id) }
func contentPath(docType, id string) (string, error) { return world.ContentPath(docType, id) }
func createRun(ctx *WorldContext, command string) (string, string, error) { return world.CreateRun(ctx, command) }
func acquireWorldLock(ctx *WorldContext, command string) (func(), error) { return world.AcquireWorldLock(ctx, command) }
func unresolvedRecovery(ctx *WorldContext) (string, bool) { return world.UnresolvedRecovery(ctx) }
func loadRegistry(flagValue string) (Registry, string, error) { return world.LoadRegistry(flagValue) }
func registryWorldList(reg Registry) []map[string]any { return world.RegistryWorldList(reg) }
func normalizeRoot(path string, mustExist bool) (string, error) { return world.NormalizeRoot(path, mustExist) }
func parseMarkdown(rel string, b []byte) (Document, error) { return world.ParseMarkdown(rel, b) }
func buildMarkdown(meta map[string]any, body string) ([]byte, error) { return world.BuildMarkdown(meta, body) }
func metaString(meta map[string]any, key string) string { return world.MetaString(meta, key) }
func metaStringList(meta map[string]any, key string) []string { return world.MetaStringList(meta, key) }
func findContentByID(ctx *WorldContext, id string) (Document, bool) { return world.FindContentByID(ctx, id) }
func idExists(ctx *WorldContext, id string, includeDrafts bool) bool { return world.IDExists(ctx, id, includeDrafts) }
func validationStatus(issues []Issue) string { return world.ValidationStatus(issues) }
func validationActions(status string) []string { return world.ValidationActions(status) }
func SaveRegistry(flagValue string, reg Registry) (string, error) { return world.SaveRegistry(flagValue, reg) }

func worldID(ctx any) any {
	if c, ok := ctx.(*WorldContext); ok && c != nil {
		return c.ID
	}
	return nil
}

func registryRoot(ctx any) any {
	if c, ok := ctx.(*WorldContext); ok && c != nil {
		return c.RegistryRoot
	}
	return nil
}

func rootPath(ctx any) any {
	if c, ok := ctx.(*WorldContext); ok && c != nil {
		return c.Root
	}
	return nil
}

func init() { fmt.Sprint("") }
