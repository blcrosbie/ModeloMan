package workflow

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	mmconfig "github.com/bcrosbie/modeloman/internal/mm/config"
	mmcontext "github.com/bcrosbie/modeloman/internal/mm/context"
)

func TestRunDryRunBuildsBundleFromStoredAndAdditionalContext(t *testing.T) {
	repoRoot := initSmokeRepo(t, map[string]string{
		"README.md":       "# ModeloMan\n",
		"internal/app.go": "package internal\n\nfunc App() string { return \"ok\" }\n",
	})

	if _, err := mmcontext.Add(repoRoot, []string{"README.md"}); err != nil {
		t.Fatalf("add stored context: %v", err)
	}

	cfg := smokeConfig(t)
	result, err := Run(context.Background(), cfg, RunParams{
		Backend:         "codex",
		TaskType:        "revamp",
		Objective:       "Refresh the wrapper docs and inspect App behavior",
		DryRun:          true,
		RepoRoot:        repoRoot,
		AdditionalEntry: []string{"internal"},
		OutputWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("workflow run: %v", err)
	}

	if result.Runner.ExitCode != 0 {
		t.Fatalf("unexpected exit code %d", result.Runner.ExitCode)
	}
	if result.Status != "completed" || result.Outcome != "success" {
		t.Fatalf("unexpected status/outcome %q/%q", result.Status, result.Outcome)
	}
	if !contains(result.Bundle.SelectedFiles, "README.md") {
		t.Fatalf("expected README.md in bundle: %v", result.Bundle.SelectedFiles)
	}
	if !contains(result.Bundle.SelectedFiles, "internal/app.go") {
		t.Fatalf("expected internal/app.go in bundle: %v", result.Bundle.SelectedFiles)
	}
	if !strings.Contains(result.Prompt, "Objective:") {
		t.Fatalf("expected prompt template sections, got %q", result.Prompt)
	}
	if !strings.Contains(result.Bundle.Rendered, "## Selected File Contents") {
		t.Fatalf("expected rendered bundle contents")
	}
}

func TestRunDryRunReportsExistingRepoDiff(t *testing.T) {
	repoRoot := initSmokeRepo(t, map[string]string{
		"README.md": "# ModeloMan\n",
	})

	readmePath := filepath.Join(repoRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("# ModeloMan\n\nUpdated.\n"), 0o644); err != nil {
		t.Fatalf("update readme: %v", err)
	}

	cfg := smokeConfig(t)
	result, err := Run(context.Background(), cfg, RunParams{
		Backend:         "codex",
		TaskType:        "docs",
		Objective:       "Summarize README changes",
		DryRun:          true,
		RepoRoot:        repoRoot,
		AdditionalEntry: []string{"README.md"},
		OutputWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("workflow run: %v", err)
	}

	if !contains(result.DiffSummary.ChangedFiles, "README.md") {
		t.Fatalf("expected README.md in diff summary: %v", result.DiffSummary.ChangedFiles)
	}
	if result.Bundle.RepoMeta.Dirty != true {
		t.Fatalf("expected repo to be dirty")
	}
}

func initSmokeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoRoot := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "config", "user.email", "smoke@example.com")
	runGit(t, repoRoot, "config", "user.name", "Smoke Test")
	runGit(t, repoRoot, "add", ".")
	runGit(t, repoRoot, "commit", "-m", "initial")
	return repoRoot
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func smokeConfig(t *testing.T) mmconfig.Config {
	t.Helper()
	t.Setenv("MODELOMAN_TOKEN", "")
	t.Setenv("MODEL0MAN_TOKEN", "")

	cfg := mmconfig.Default()
	cfg.TokenEnvVar = "MM_TEST_TOKEN"
	return cfg
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
