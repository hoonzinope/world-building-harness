package server

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func readJSONL(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) { return nil }
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 8*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" { continue }
		if err := fn([]byte(sc.Text())); err != nil { return err }
	}
	return sc.Err()
}

func appendJSONL(path string, v any) error { return storyAppendJSONL(path, v) }
func writeJSONAtomic(path string, v any) error { return storyWriteJSONAtomic(path, v) }
func writeAtomic(path string, b []byte) error { return storyWriteAtomic(path, b) }
func fsyncDir(path string) error { return storyFsyncDir(path) }

// local copies to preserve old root helpers for callers that still use them.
func storyAppendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return err }
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil { return err }
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil { return err }
	if _, err := f.Write(append(b, '\n')); err != nil { return err }
	if err := f.Sync(); err != nil { return err }
	return storyFsyncDir(filepath.Dir(path))
}

func storyWriteJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil { return err }
	return storyWriteAtomic(path, append(b, '\n'))
}

func storyWriteAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return err }
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil { return err }
	f, err := os.Open(tmp)
	if err != nil { return err }
	if err := f.Sync(); err != nil { _ = f.Close(); return err }
	if err := f.Close(); err != nil { return err }
	if err := os.Rename(tmp, path); err != nil { return err }
	return storyFsyncDir(filepath.Dir(path))
}

func storyFsyncDir(path string) error {
	d, err := os.Open(path)
	if err != nil { return err }
	defer d.Close()
	return d.Sync()
}
