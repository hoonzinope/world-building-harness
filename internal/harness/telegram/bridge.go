package telegram

import (
	"os"

	"github.com/hoonzi/world-harness/internal/harness/core"
	"github.com/hoonzi/world-harness/internal/harness/world"
)

type WorldContext = world.Context
type Document = world.Document

func safeRel(root, rel string) (string, string, error) { return world.SafeRel(root, rel) }
func searchDocuments(ctx *WorldContext, scope, query string) ([]map[string]any, error) {
	return world.SearchDocuments(ctx, scope, query)
}
func worldStatus(ctx *WorldContext) map[string]any { return world.Status(ctx) }
func createRun(ctx *WorldContext, command string) (string, string, error) {
	return world.CreateRun(ctx, command)
}
func writeJSON(path string, v any) error { return core.WriteJSON(path, v) }
func writeFileAtomic(path string, b []byte, perm os.FileMode) error {
	return core.WriteFileAtomic(path, b, perm)
}
func sha256Bytes(b []byte) string { return core.Sha256Bytes(b) }
func readDocument(ctx *WorldContext, rel string) (Document, error) {
	return world.ReadDocument(ctx, rel)
}
func parseMarkdown(rel string, b []byte) (Document, error) { return world.ParseMarkdown(rel, b) }
func buildMarkdown(meta map[string]any, body string) ([]byte, error) {
	return world.BuildMarkdown(meta, body)
}
func firstNonEmpty(values ...string) string { return core.FirstNonEmpty(values...) }
func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func normalizeRoot(path string, mustExist bool) (string, error) {
	return world.NormalizeRoot(path, mustExist)
}
func Packs(root string) ([]map[string]any, error)           { return world.Packs(root) }
func PackContext(root, pack string) (*WorldContext, error)  { return world.PackContext(root, pack) }
func draftPath(docType, id string) (string, error)          { return world.DraftPath(docType, id) }
func initWorld(root, worldID string) (*WorldContext, error) { return world.Init(root, worldID) }
