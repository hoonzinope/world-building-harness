package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	toolSchemaVersion = "world-tool.v1"
	docSchemaVersion  = "world-doc.v1"
)

type Envelope struct {
	SchemaVersion    string     `json:"schema_version"`
	OK               bool       `json:"ok"`
	CommandStatus    string     `json:"command_status"`
	Command          string     `json:"command"`
	WorldID          any        `json:"world_id"`
	RegistryRoot     any        `json:"registry_root"`
	Root             any        `json:"root"`
	RunID            any        `json:"run_id"`
	Data             any        `json:"data"`
	Error            *ToolError `json:"error,omitempty"`
	Issues           []Issue    `json:"issues"`
	AvailableActions []string   `json:"available_actions"`
}

type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type Issue struct {
	Code           string `json:"code"`
	Rule           string `json:"rule,omitempty"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	Path           string `json:"path,omitempty"`
	Field          string `json:"field,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

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

type Harness struct {
	SchemaVersion string `yaml:"schema_version" json:"schema_version"`
	WorldID       string `yaml:"world_id" json:"world_id"`
	WorldRoot     string `yaml:"world_root" json:"world_root"`
	ContentDir    string `yaml:"content_dir" json:"content_dir"`
	DraftDir      string `yaml:"draft_dir" json:"draft_dir"`
	RunDir        string `yaml:"run_dir" json:"run_dir"`
	InboxDir      string `yaml:"inbox_dir" json:"inbox_dir"`
	GraphDir      string `yaml:"graph_dir" json:"graph_dir"`
	ArchiveDir    string `yaml:"archive_dir" json:"archive_dir"`
}

type WorldContext struct {
	ID           string
	Root         string
	RegistryRoot string
	Harness      Harness
}

func okEnvelope(command string, ctx *WorldContext, runID any, data any, issues []Issue, actions []string) Envelope {
	var worldID, registryRoot, root any
	if ctx != nil {
		worldID = ctx.ID
		registryRoot = ctx.RegistryRoot
		root = ctx.Root
	}
	if issues == nil {
		issues = []Issue{}
	}
	if actions == nil {
		actions = []string{}
	}
	return Envelope{
		SchemaVersion:    toolSchemaVersion,
		OK:               true,
		CommandStatus:    "completed",
		Command:          command,
		WorldID:          worldID,
		RegistryRoot:     registryRoot,
		Root:             root,
		RunID:            runID,
		Data:             data,
		Issues:           issues,
		AvailableActions: actions,
	}
}

func blockedEnvelope(command string, ctx *WorldContext, runID any, blockReason string, data map[string]any, issues []Issue, actions []string) Envelope {
	env := okEnvelope(command, ctx, runID, data, issues, actions)
	env.CommandStatus = "blocked"
	if data == nil {
		data = map[string]any{}
		env.Data = data
	}
	data["block_reason"] = blockReason
	return env
}

func failEnvelope(command string, ctx *WorldContext, code, message string, details any) Envelope {
	var worldID, registryRoot, root any
	if ctx != nil {
		worldID = ctx.ID
		registryRoot = ctx.RegistryRoot
		root = ctx.Root
	}
	return Envelope{
		SchemaVersion:    toolSchemaVersion,
		OK:               false,
		CommandStatus:    "failed",
		Command:          command,
		WorldID:          worldID,
		RegistryRoot:     registryRoot,
		Root:             root,
		RunID:            nil,
		Data:             map[string]any{},
		Error:            &ToolError{Code: code, Message: message, Details: details},
		Issues:           []Issue{},
		AvailableActions: []string{},
	}
}

func emit(env Envelope) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
	if !env.OK {
		switch env.Error.Code {
		case "INVALID_ARGUMENT", "REGISTRY_NOT_FOUND", "WORLD_NOT_FOUND", "PATH_OUTSIDE_ROOT", "PATH_NOT_MARKDOWN", "PATH_SCOPE_DENIED", "INPUT_HASH_MISMATCH", "AUTH_CONTEXT_MISSING", "AUTH_CONTEXT_HASH_MISMATCH", "AUTH_CONTEXT_EXPIRED", "AUTH_CONTEXT_SCOPE_DENIED", "AUTH_CONTEXT_TEST_MODE_REQUIRED", "APPROVAL_ATTESTATION_HASH_MISMATCH", "APPROVAL_ATTESTATION_EXPIRED", "APPROVAL_ATTESTATION_BINDING_MISMATCH":
			return 2
		case "IO_ERROR", "LOCK_BUSY", "TRANSACTION_INCOMPLETE":
			return 3
		default:
			return 4
		}
	}
	return 0
}

func sha256Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func nowDate() string {
	return time.Now().UTC().Format("2006-01-02")
}

func runID() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())))
	return time.Now().UTC().Format("20060102-150405") + "-" + hex.EncodeToString(sum[:])[:8]
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeJSON(path string, v any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(v)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}

func writeFileAtomic(path string, b []byte, perm os.FileMode) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
