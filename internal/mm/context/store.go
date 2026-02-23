package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	contextRelPath = ".modeloman/context.json"
)

type RepoContext struct {
	Version   int      `json:"version"`
	Entries   []string `json:"entries"`
	UpdatedAt string   `json:"updated_at"`
}

func Load(repoRoot string) (RepoContext, error) {
	path := filePath(repoRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RepoContext{Version: 1, Entries: []string{}, UpdatedAt: ""}, nil
		}
		return RepoContext{}, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg RepoContext
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return RepoContext{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Entries == nil {
		cfg.Entries = []string{}
	}
	sort.Strings(cfg.Entries)
	return cfg, nil
}

func Save(repoRoot string, cfg RepoContext) error {
	cfg.Version = 1
	cfg.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	cfg.Entries = normalizeEntries(repoRoot, cfg.Entries)

	path := filePath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir context dir: %w", err)
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode context: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write context: %w", err)
	}
	return nil
}

func Add(repoRoot string, entries []string) (RepoContext, error) {
	cfg, err := Load(repoRoot)
	if err != nil {
		return RepoContext{}, err
	}
	cfg.Entries = append(cfg.Entries, entries...)
	cfg.Entries = normalizeEntries(repoRoot, cfg.Entries)
	if err := Save(repoRoot, cfg); err != nil {
		return RepoContext{}, err
	}
	return cfg, nil
}

func Drop(repoRoot string, entries []string) (RepoContext, error) {
	cfg, err := Load(repoRoot)
	if err != nil {
		return RepoContext{}, err
	}
	toDrop := map[string]struct{}{}
	for _, item := range normalizeEntries(repoRoot, entries) {
		toDrop[item] = struct{}{}
	}
	next := make([]string, 0, len(cfg.Entries))
	for _, item := range cfg.Entries {
		if _, remove := toDrop[item]; remove {
			continue
		}
		next = append(next, item)
	}
	cfg.Entries = normalizeEntries(repoRoot, next)
	if err := Save(repoRoot, cfg); err != nil {
		return RepoContext{}, err
	}
	return cfg, nil
}

func Clear(repoRoot string) error {
	path := filePath(repoRoot)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear context file: %w", err)
	}
	return nil
}

func normalizeEntries(repoRoot string, entries []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		clean := strings.TrimSpace(entry)
		if clean == "" {
			continue
		}
		clean = strings.TrimPrefix(clean, "./")
		if !hasGlob(clean) && filepath.IsAbs(clean) {
			rel, err := filepath.Rel(repoRoot, clean)
			if err == nil && !strings.HasPrefix(rel, "..") {
				clean = rel
			}
		}
		clean = filepath.ToSlash(filepath.Clean(clean))
		if clean == "." {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

func filePath(repoRoot string) string {
	return filepath.Join(repoRoot, contextRelPath)
}

func hasGlob(value string) bool {
	return strings.ContainsAny(value, "*?[")
}
