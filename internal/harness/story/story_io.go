package story

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
func readFileIfExists(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

func readStoryJSONL(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var offset, lastGood int64
	var lineNo int
	var lastLine []byte
	var needsRepair bool
	sc := bufio.NewReader(f)
	for {
		line, err := sc.ReadBytes('\n')
		if len(line) == 0 && err == io.EOF {
			break
		}
		lineNo++
		offset += int64(len(line))
		trimmed := bytes.TrimRight(line, "\r\n")
		if len(bytes.TrimSpace(trimmed)) == 0 {
			if err == io.EOF {
				break
			}
			lastGood = offset
			continue
		}
		if fnErr := fn(trimmed); fnErr != nil {
			if err == io.EOF {
				needsRepair = true
				lastLine = append([]byte(nil), trimmed...)
				break
			}
			_ = f.Close()
			return fmt.Errorf("malformed JSONL line %d in %s: %w", lineNo, path, fnErr)
		}
		lastGood = offset
		if err == io.EOF {
			break
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if !needsRepair {
		return nil
	}
	if err := truncateStoryJSONL(path, lastGood); err != nil {
		return err
	}
	if err := appendStoryRecoveryEvent(path, lastGood, lastLine); err != nil {
		return err
	}
	return nil
}

func truncateStoryJSONL(path string, offset int64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := f.Truncate(offset); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func appendStoryRecoveryEvent(path string, truncatedTo int64, repairedLine []byte) error {
	eventPath := path
	if filepath.Base(path) != "events.jsonl" {
		eventPath = filepath.Join(filepath.Dir(path), "events.jsonl")
	}
	return appendJSONL(eventPath, map[string]any{"type": "story_recovered", "at": time.Now().UTC().Format(time.RFC3339), "recovered_path": filepath.Base(path), "truncated_to": truncatedTo, "repaired_tail": string(repairedLine)})
}

func readJSONL(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 8*1024*1024)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		if err := fn([]byte(sc.Text())); err != nil {
			return err
		}
	}
	return sc.Err()
}

func appendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(b, '\n'))
}
func writeAtomic(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp." + randomID()
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func fsyncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func appendUnique(in []string, vals ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range append(in, vals...) {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
