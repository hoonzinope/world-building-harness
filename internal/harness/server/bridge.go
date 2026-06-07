package server

import (
	"fmt"
	"os"

	"github.com/hoonzi/world-harness/internal/harness/core"
	"github.com/hoonzi/world-harness/internal/harness/world"
)

type WorldContext = world.Context
type Harness = world.Harness
type Document = world.Document

type Envelope = core.Envelope
type Issue = core.Issue

func firstNonEmpty(values ...string) string { return core.FirstNonEmpty(values...) }
func safeRel(root, rel string) (string, string, error) { return world.SafeRel(root, rel) }
func readDocument(ctx *WorldContext, rel string) (Document, error) { return world.ReadDocument(ctx, rel) }
func listDocuments(ctx *WorldContext, scope string) ([]Document, error) { return world.ListDocuments(ctx, scope) }
func searchDocuments(ctx *WorldContext, scope, query string) ([]map[string]any, error) {
	return world.SearchDocuments(ctx, scope, query)
}
func documentSummary(doc Document) map[string]any { return world.DocumentSummary(doc) }
func worldStatus(ctx *WorldContext) map[string]any { return world.Status(ctx) }
func parseMarkdown(rel string, b []byte) (Document, error) { return world.ParseMarkdown(rel, b) }
func readHarness(root string) (Harness, error) { return world.ReadHarness(root) }
func normalizeRoot(path string, mustExist bool) (string, error) { return world.NormalizeRoot(path, mustExist) }
func loadRegistry(flagValue string) (world.Registry, string, error) { return world.LoadRegistry(flagValue) }

func writeJSON(path string, v any) error { return core.WriteJSON(path, v) }
func writeFileAtomic(path string, b []byte, perm os.FileMode) error { return core.WriteFileAtomic(path, b, perm) }
func sha256Bytes(b []byte) string { return core.Sha256Bytes(b) }
func sha256File(path string) (string, error) { return core.Sha256File(path) }
func nowDate() string { return core.NowDate() }
func randomToken(n int) (string, error) { return core.RandomToken(n) }
func randomID() string { return core.RandomID() }

func failEnvelope(command string, ctx any, code, message string, details any) Envelope {
	return core.FailEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), code, message, details)
}
func okEnvelope(command string, ctx any, runID any, data any, issues []Issue, actions []string) Envelope {
	return core.OkEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), runID, data, issues, actions)
}
func blockedEnvelope(command string, ctx any, runID any, reason string, data map[string]any, issues []Issue, actions []string) Envelope {
	return core.BlockedEnvelope(command, worldID(ctx), registryRoot(ctx), rootPath(ctx), runID, reason, data, issues, actions)
}

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
