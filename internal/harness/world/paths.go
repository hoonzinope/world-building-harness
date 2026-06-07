package world

import (
	"fmt"
	"path/filepath"
	"strings"
)

func SafeRel(root, rel string) (string, string, error) {
	if rel == "" {
		return "", "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", "", fmt.Errorf("path traversal is not allowed")
	}
	abs := filepath.Join(root, clean)
	rootEval, err := filepath.EvalSymlinks(root)
	if err != nil {
		rootEval = root
	}
	parent := filepath.Dir(abs)
	if evalParent, err := filepath.EvalSymlinks(parent); err == nil {
		if !pathInside(rootEval, evalParent) {
			return "", "", fmt.Errorf("path escapes world root")
		}
	}
	if evalAbs, err := filepath.EvalSymlinks(abs); err == nil {
		if !pathInside(rootEval, evalAbs) {
			return "", "", fmt.Errorf("path escapes world root")
		}
	}
	return abs, filepath.ToSlash(clean), nil
}

func pathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func RequireMarkdownPath(rel string) error {
	if !strings.HasSuffix(strings.ToLower(rel), ".md") {
		return fmt.Errorf("path is not markdown")
	}
	return nil
}

func DraftPath(docType, id string) (string, error) {
	dir, ok := draftDirs[docType]
	if !ok {
		return "", fmt.Errorf("unsupported document type %q", docType)
	}
	return filepath.ToSlash(filepath.Join("drafts", dir, id+".md")), nil
}

func ContentPath(docType, id string) (string, error) {
	dir, ok := contentDirs[docType]
	if !ok {
		return "", fmt.Errorf("unsupported content document type %q", docType)
	}
	return filepath.ToSlash(filepath.Join("content", dir, id+".md")), nil
}
