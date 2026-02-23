package context

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/bcrosbie/modeloman/internal/mm/gitutil"
)

type BuildOptions struct {
	RepoRoot      string
	Entries       []string
	Prompt        string
	MaxBytes      int
	TokenBudget   int
	MaxTreeLines  int
	MaxFileBytes  int
	MaxGrepHits   int
	GitDiffBudget int
}

type Bundle struct {
	RepoMeta       gitutil.RepoMeta `json:"repo_meta"`
	SelectedFiles  []string         `json:"selected_files"`
	TreeOutline    []string         `json:"tree_outline"`
	GitStatus      string           `json:"git_status"`
	GitDiff        string           `json:"git_diff"`
	SymbolHits     []string         `json:"symbol_hits"`
	Rendered       string           `json:"rendered"`
	RenderedBytes  int              `json:"rendered_bytes"`
	EstimatedToken int              `json:"estimated_tokens"`
	Hash           string           `json:"hash"`
}

var ignoredDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"dist":         {},
	"vendor":       {},
	"bin":          {},
	".next":        {},
	".idea":        {},
	".vscode":      {},
	"target":       {},
}

func BuildBundle(opts BuildOptions) (Bundle, error) {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return Bundle{}, errors.New("repo root is required")
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 350000
	}
	if opts.TokenBudget > 0 {
		tokenBytes := opts.TokenBudget * 4
		if tokenBytes > 0 && tokenBytes < opts.MaxBytes {
			opts.MaxBytes = tokenBytes
		}
	}
	if opts.MaxTreeLines <= 0 {
		opts.MaxTreeLines = 300
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = 64000
	}
	if opts.MaxGrepHits <= 0 {
		opts.MaxGrepHits = 20
	}
	if opts.GitDiffBudget <= 0 {
		opts.GitDiffBudget = opts.MaxBytes / 3
	}

	meta, err := gitutil.Metadata(opts.RepoRoot)
	if err != nil {
		return Bundle{}, err
	}
	selected, err := ResolveEntries(opts.RepoRoot, opts.Entries)
	if err != nil {
		return Bundle{}, err
	}
	status, err := gitutil.StatusPorcelain(opts.RepoRoot)
	if err != nil {
		return Bundle{}, err
	}
	diff, err := gitutil.CombinedDiff(opts.RepoRoot, opts.GitDiffBudget)
	if err != nil {
		return Bundle{}, err
	}
	tree, err := buildTreeOutline(opts.RepoRoot, opts.MaxTreeLines)
	if err != nil {
		return Bundle{}, err
	}
	symbolHits := grepPromptSymbols(opts.RepoRoot, opts.Prompt, opts.MaxGrepHits)

	rendered := renderBundle(meta, selected, tree, status, diff, symbolHits, opts.MaxBytes, opts.MaxFileBytes, opts.RepoRoot)
	hash := sha256.Sum256([]byte(rendered))

	return Bundle{
		RepoMeta:       meta,
		SelectedFiles:  selected,
		TreeOutline:    tree,
		GitStatus:      status,
		GitDiff:        diff,
		SymbolHits:     symbolHits,
		Rendered:       rendered,
		RenderedBytes:  len(rendered),
		EstimatedToken: len(rendered) / 4,
		Hash:           hex.EncodeToString(hash[:]),
	}, nil
}

func ResolveEntries(repoRoot string, entries []string) ([]string, error) {
	normalized := normalizeEntries(repoRoot, entries)
	found := map[string]struct{}{}

	for _, entry := range normalized {
		if strings.Contains(entry, "**") {
			if err := resolveDoubleStar(repoRoot, entry, found); err != nil {
				return nil, err
			}
			continue
		}
		if hasGlob(entry) {
			matches, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(entry)))
			if err != nil {
				return nil, fmt.Errorf("glob %q: %w", entry, err)
			}
			for _, match := range matches {
				if err := addPath(repoRoot, match, found); err != nil {
					return nil, err
				}
			}
			continue
		}
		path := filepath.Join(repoRoot, filepath.FromSlash(entry))
		if err := addPath(repoRoot, path, found); err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(found))
	for file := range found {
		out = append(out, file)
	}
	sort.Strings(out)
	return out, nil
}

func addPath(repoRoot, absPath string, found map[string]struct{}) error {
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(absPath, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if _, skip := ignoredDirs[entry.Name()]; skip {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			found[filepath.ToSlash(rel)] = struct{}{}
			return nil
		})
	}
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return err
	}
	found[filepath.ToSlash(rel)] = struct{}{}
	return nil
}

func resolveDoubleStar(repoRoot, pattern string, found map[string]struct{}) error {
	re, err := doublestarRegex(pattern)
	if err != nil {
		return err
	}
	return filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, skip := ignoredDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if re.MatchString(rel) {
			found[rel] = struct{}{}
		}
		return nil
	})
}

func doublestarRegex(pattern string) (*regexp.Regexp, error) {
	escaped := regexp.QuoteMeta(filepath.ToSlash(pattern))
	escaped = strings.ReplaceAll(escaped, `\*\*`, `@@DOUBLESTAR@@`)
	escaped = strings.ReplaceAll(escaped, `\*`, `[^/]*`)
	escaped = strings.ReplaceAll(escaped, `\?`, `[^/]`)
	escaped = strings.ReplaceAll(escaped, `@@DOUBLESTAR@@`, `.*`)
	return regexp.Compile("^" + escaped + "$")
}

func buildTreeOutline(repoRoot string, maxLines int) ([]string, error) {
	lines := make([]string, 0, maxLines)
	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == repoRoot {
			return nil
		}
		if entry.IsDir() {
			if _, skip := ignoredDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		lines = append(lines, rel)
		if len(lines) >= maxLines {
			return errors.New("tree-truncated")
		}
		return nil
	})
	if err != nil && err.Error() != "tree-truncated" {
		return nil, err
	}
	sort.Strings(lines)
	return lines, nil
}

func renderBundle(
	meta gitutil.RepoMeta,
	selected, tree []string,
	gitStatus, gitDiff string,
	symbolHits []string,
	maxBytes int,
	maxFileBytes int,
	repoRoot string,
) string {
	var buf bytes.Buffer
	remaining := maxBytes

	appendLimited := func(text string) bool {
		if remaining <= 0 {
			return false
		}
		if len(text) <= remaining {
			buf.WriteString(text)
			remaining -= len(text)
			return true
		}
		buf.WriteString(text[:remaining])
		buf.WriteString("\n...[bundle-truncated]\n")
		remaining = 0
		return false
	}

	appendLimited("## Repo Metadata\n")
	appendLimited("root: " + meta.Root + "\n")
	appendLimited("branch: " + meta.Branch + "\n")
	appendLimited("commit: " + meta.Commit + "\n")
	appendLimited("dirty: " + strconv.FormatBool(meta.Dirty) + "\n\n")

	appendLimited("## Selected Files\n")
	for _, item := range selected {
		if !appendLimited("- " + item + "\n") {
			break
		}
	}
	appendLimited("\n## Tree Outline\n")
	for _, item := range tree {
		if !appendLimited("- " + item + "\n") {
			break
		}
	}

	appendLimited("\n## Git Status (Porcelain)\n")
	appendLimited(gitStatus + "\n")

	appendLimited("\n## Git Diff (Staged + Unstaged)\n")
	appendLimited(gitDiff + "\n")

	if len(symbolHits) > 0 {
		appendLimited("\n## Prompt Symbol Grep Hits\n")
		for _, hit := range symbolHits {
			if !appendLimited("- " + hit + "\n") {
				break
			}
		}
	}

	appendLimited("\n## Selected File Contents\n")
	for _, rel := range selected {
		if remaining <= 0 {
			break
		}
		abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
		content, truncated := readFileSnippet(abs, maxFileBytes)
		header := "### " + rel + "\n"
		if truncated {
			header += "(truncated)\n"
		}
		if !appendLimited(header) {
			break
		}
		if !appendLimited(content + "\n") {
			break
		}
	}

	return buf.String()
}

func readFileSnippet(path string, maxBytes int) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "[read error: " + err.Error() + "]", false
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return "[binary file omitted]", false
	}
	if maxBytes > 0 && len(raw) > maxBytes {
		return string(raw[:maxBytes]), true
	}
	return string(raw), false
}

func grepPromptSymbols(repoRoot, prompt string, maxHits int) []string {
	symbols := extractSymbols(prompt, 6)
	if len(symbols) == 0 || maxHits <= 0 {
		return nil
	}
	if _, err := exec.LookPath("rg"); err != nil {
		return nil
	}

	hits := make([]string, 0, maxHits)
	for _, symbol := range symbols {
		cmd := exec.Command("rg", "--no-heading", "-n", "-m", "2", symbol, ".")
		cmd.Dir = repoRoot
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			hits = append(hits, symbol+": "+line)
			if len(hits) >= maxHits {
				sort.Strings(hits)
				return hits
			}
		}
	}
	sort.Strings(hits)
	return hits
}

func extractSymbols(prompt string, max int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, max)
	parts := strings.FieldsFunc(prompt, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
	for _, part := range parts {
		if len(part) < 4 {
			continue
		}
		if !isIdentifierPrefix(part[0]) {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
		if len(out) >= max {
			break
		}
	}
	return out
}

func isIdentifierPrefix(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}
