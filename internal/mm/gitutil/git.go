package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type RepoMeta struct {
	Root   string `json:"root"`
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type DiffSummary struct {
	ChangedFiles []string `json:"changed_files"`
	AddedLines   int      `json:"added_lines"`
	DeletedLines int      `json:"deleted_lines"`
}

func DetectRepoRoot() (string, error) {
	out, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("detect repo root: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func Metadata(repoRoot string) (RepoMeta, error) {
	commit, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return RepoMeta{}, fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	branch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return RepoMeta{}, fmt.Errorf("git branch: %w", err)
	}
	status, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return RepoMeta{}, fmt.Errorf("git status: %w", err)
	}
	return RepoMeta{
		Root:   repoRoot,
		Commit: strings.TrimSpace(commit),
		Branch: strings.TrimSpace(branch),
		Dirty:  strings.TrimSpace(status) != "",
	}, nil
}

func StatusPorcelain(repoRoot string) (string, error) {
	out, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("git status --porcelain: %w", err)
	}
	return out, nil
}

func CombinedDiff(repoRoot string, maxBytes int) (string, error) {
	unstaged, err := runGit(repoRoot, "diff", "--no-color")
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	staged, err := runGit(repoRoot, "diff", "--cached", "--no-color")
	if err != nil {
		return "", fmt.Errorf("git diff --cached: %w", err)
	}

	combined := "=== git diff (unstaged) ===\n" + unstaged + "\n=== git diff (staged) ===\n" + staged
	if maxBytes > 0 && len(combined) > maxBytes {
		return combined[:maxBytes] + "\n...[truncated]", nil
	}
	return combined, nil
}

func SummarizeDiff(repoRoot string) (DiffSummary, error) {
	unstaged, err := runGit(repoRoot, "diff", "--numstat")
	if err != nil {
		return DiffSummary{}, fmt.Errorf("git diff --numstat: %w", err)
	}
	staged, err := runGit(repoRoot, "diff", "--cached", "--numstat")
	if err != nil {
		return DiffSummary{}, fmt.Errorf("git diff --cached --numstat: %w", err)
	}

	files := map[string]struct{}{}
	var added, deleted int
	for _, chunk := range []string{unstaged, staged} {
		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			add, addErr := strconv.Atoi(parts[0])
			del, delErr := strconv.Atoi(parts[1])
			if addErr == nil {
				added += add
			}
			if delErr == nil {
				deleted += del
			}
			files[parts[2]] = struct{}{}
		}
	}

	outFiles := make([]string, 0, len(files))
	for file := range files {
		outFiles = append(outFiles, file)
	}
	sort.Strings(outFiles)

	return DiffSummary{
		ChangedFiles: outFiles,
		AddedLines:   added,
		DeletedLines: deleted,
	}, nil
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(repoRoot) != "" {
		cmd.Dir = repoRoot
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
