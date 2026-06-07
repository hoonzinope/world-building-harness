package harness

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

func registryPath(flagValue string, forWrite bool) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}
	if env := os.Getenv("WORLD_TOOL_REGISTRY"); env != "" {
		return filepath.Abs(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(home, ".opencrabs", "worlds.yaml"),
		filepath.Join(home, ".config", "world-tool", "worlds.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if forWrite {
		return candidates[0], nil
	}
	return candidates[0], nil
}

func loadRegistry(flagValue string) (Registry, string, error) {
	p, err := registryPath(flagValue, false)
	if err != nil {
		return Registry{}, "", err
	}
	reg := Registry{Worlds: map[string]RegistryWorld{}}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return reg, p, nil
	}
	if err != nil {
		return Registry{}, p, err
	}
	if err := yaml.Unmarshal(b, &reg); err != nil {
		return Registry{}, p, err
	}
	if reg.Worlds == nil {
		reg.Worlds = map[string]RegistryWorld{}
	}
	return reg, p, nil
}

func saveRegistry(flagValue string, reg Registry) (string, error) {
	p, err := registryPath(flagValue, true)
	if err != nil {
		return "", err
	}
	if reg.Worlds == nil {
		reg.Worlds = map[string]RegistryWorld{}
	}
	b, err := yaml.Marshal(reg)
	if err != nil {
		return "", err
	}
	if err := ensureDir(filepath.Dir(p)); err != nil {
		return "", err
	}
	return p, writeFileAtomic(p, b, 0o644)
}

func registryWorldList(reg Registry) []map[string]any {
	keys := make([]string, 0, len(reg.Worlds))
	for id := range reg.Worlds {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, id := range keys {
		item := reg.Worlds[id]
		out = append(out, map[string]any{
			"id":         id,
			"world_id":   id,
			"title":      item.Title,
			"root":       item.Root,
			"is_default": id == reg.Default,
		})
	}
	return out
}
