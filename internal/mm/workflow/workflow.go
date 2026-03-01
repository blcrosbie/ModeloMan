package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	mmconfig "github.com/bcrosbie/modeloman/internal/mm/config"
	mmcontext "github.com/bcrosbie/modeloman/internal/mm/context"
	"github.com/bcrosbie/modeloman/internal/mm/gitutil"
	"github.com/bcrosbie/modeloman/internal/mm/prompt"
	"github.com/bcrosbie/modeloman/internal/mm/redact"
	"github.com/bcrosbie/modeloman/internal/mm/runner"
	"github.com/bcrosbie/modeloman/internal/mm/telemetry"
)

type RunParams struct {
	Backend         string
	TaskType        string
	Skill           string
	Objective       string
	BudgetTokens    int
	DryRun          bool
	UsePTY          bool
	ForwardInput    bool
	InputReader     io.Reader
	AdditionalEntry []string
	RepoRoot        string
	OutputWriter    io.Writer
	OnOutput        func(string)
	OnRunnerEvent   func(runner.Event)
}

type RunResult struct {
	RunID         string
	RepoRoot      string
	Bundle        mmcontext.Bundle
	Prompt        string
	PromptHash    string
	Runner        runner.Result
	DiffSummary   gitutil.DiffSummary
	Status        string
	Outcome       string
	LastError     string
	AgentID       string
	SelectedEntry []string
}

func Run(ctx context.Context, cfg mmconfig.Config, params RunParams) (RunResult, error) {
	backend := strings.TrimSpace(params.Backend)
	if backend == "" {
		backend = strings.TrimSpace(cfg.DefaultBackend)
	}
	if backend == "" {
		return RunResult{}, fmt.Errorf("backend is required")
	}
	taskType := strings.TrimSpace(params.TaskType)
	if taskType == "" {
		taskType = "general-coding"
	}
	objective := strings.TrimSpace(params.Objective)
	if objective == "" {
		return RunResult{}, fmt.Errorf("objective is required")
	}

	repoRoot := strings.TrimSpace(params.RepoRoot)
	if repoRoot == "" {
		var err error
		repoRoot, err = gitutil.DetectRepoRoot()
		if err != nil {
			return RunResult{}, err
		}
	}

	storedCtx, err := mmcontext.Load(repoRoot)
	if err != nil {
		return RunResult{}, err
	}
	entries := mergeEntries(storedCtx.Entries, params.AdditionalEntry)

	bundle, err := mmcontext.BuildBundle(mmcontext.BuildOptions{
		RepoRoot:    repoRoot,
		Entries:     entries,
		Prompt:      objective,
		MaxBytes:    cfg.MaxContextBytes,
		TokenBudget: params.BudgetTokens,
	})
	if err != nil {
		return RunResult{}, err
	}

	snippet := loadSkillSnippet(repoRoot, params.Skill)
	houseRules := strings.Join([]string{
		"- Do not leak secrets in logs or summaries.",
		"- Keep the change set minimal and verifiable.",
		"- Prioritize compile/test pass and explicit next steps.",
	}, "\n")
	finalPrompt := prompt.Build(prompt.TemplateInput{
		Objective:      objective,
		TaskType:       taskType,
		SkillName:      params.Skill,
		SkillSnippet:   snippet,
		ContextDigest:  bundle.Hash,
		Backend:        backend,
		BudgetTokens:   params.BudgetTokens,
		AdditionalHint: houseRules,
	})

	redactor := redact.New(cfg.RedactionEnabled, cfg.CustomRedactRegexes)
	safeBundle := redactor.Apply(bundle.Rendered)
	safePrompt := redactor.Apply(finalPrompt)
	promptHash := digestString(safePrompt)

	token := mmconfig.ResolveToken(cfg)
	var client *telemetry.Client
	if strings.TrimSpace(token) != "" {
		client, err = telemetry.New(cfg, token)
		if err != nil {
			log.Printf("telemetry disabled: %v", err)
		}
	}
	if client != nil {
		defer client.Close()
	}

	runID := ""
	agentID := localAgentID()
	if client != nil {
		startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		runID, err = client.StartRun(startCtx, telemetry.StartRunInput{
			Workflow:      taskType,
			AgentID:       agentID,
			PromptVersion: strings.TrimSpace(params.Skill),
			ModelPolicy:   backend,
		})
		cancel()
		if err != nil {
			log.Printf("start run failed: %v", err)
		} else {
			_ = client.RecordRunEvent(context.Background(), telemetry.EventInput{
				RunID:     runID,
				EventType: "mm_run_started",
				Level:     "info",
				Message:   "mm wrapper started backend session",
				Data: map[string]any{
					"backend":          backend,
					"task_type":        taskType,
					"budget_tokens":    params.BudgetTokens,
					"repo_root":        bundle.RepoMeta.Root,
					"branch":           bundle.RepoMeta.Branch,
					"commit":           bundle.RepoMeta.Commit,
					"dirty":            bundle.RepoMeta.Dirty,
					"context_hash":     bundle.Hash,
					"prompt_hash":      promptHash,
					"selected_entries": entries,
					"selected_files":   bundle.SelectedFiles,
				},
			})
		}
	}

	runResult := runner.Result{
		ExitCode:  0,
		StartedAt: time.Now().UTC(),
		EndedAt:   time.Now().UTC(),
		Duration:  0,
		Events:    []runner.Event{},
	}
	if !params.DryRun {
		runResult = runner.Run(ctx, runner.Options{
			Backend:            backend,
			RepoDir:            repoRoot,
			Prompt:             finalPrompt + "\n\nContext bundle:\n" + safeBundle,
			UsePTY:             params.UsePTY,
			CaptureTranscript:  true,
			MaxTranscriptBytes: cfg.MaxTranscriptBytes,
			ForwardInput:       params.ForwardInput,
			InputReader:        params.InputReader,
			OutputWriter:       params.OutputWriter,
			OnOutput:           params.OnOutput,
			OnEvent:            params.OnRunnerEvent,
		})
	}

	diffSummary, diffErr := gitutil.SummarizeDiff(repoRoot)
	if diffErr != nil {
		log.Printf("diff summary warning: %v", diffErr)
	}

	outcome := "success"
	status := "completed"
	lastErr := ""
	if runResult.Err != nil || runResult.ExitCode != 0 {
		outcome = "failed"
		status = "failed"
		if runResult.Err != nil {
			lastErr = runResult.Err.Error()
		} else {
			lastErr = fmt.Sprintf("backend exited with status %d", runResult.ExitCode)
		}
	}

	if client != nil && strings.TrimSpace(runID) != "" {
		for _, event := range runResult.Events {
			_ = client.RecordRunEvent(context.Background(), telemetry.EventInput{
				RunID:     runID,
				EventType: "mm_runner_event",
				Level:     "info",
				Message:   event.Message,
				Data: map[string]any{
					"type": event.Type,
					"at":   event.At,
					"data": event.Data,
				},
			})
		}

		if transcript := strings.TrimSpace(runResult.Transcript); transcript != "" {
			data := map[string]any{
				"redacted_transcript": redactor.Apply(transcript),
				"bytes":               len(transcript),
				"truncated":           runResult.TranscriptTruncated,
			}
			if cfg.AllowRawTranscript {
				data["raw_transcript"] = transcript
			}
			_ = client.RecordRunEvent(context.Background(), telemetry.EventInput{
				RunID:     runID,
				EventType: "mm_transcript",
				Level:     "info",
				Message:   "captured backend transcript",
				Data:      data,
			})
		}

		_ = client.RecordPromptAttempt(context.Background(), telemetry.AttemptInput{
			RunID:         runID,
			AttemptNumber: 1,
			Workflow:      taskType,
			AgentID:       agentID,
			Model:         backend,
			PromptVersion: strings.TrimSpace(params.Skill),
			PromptHash:    promptHash,
			Outcome:       outcome,
			ErrorMessage:  redactor.Apply(lastErr),
			LatencyMS:     runResult.Duration.Milliseconds(),
		})
		_ = client.RecordRunEvent(context.Background(), telemetry.EventInput{
			RunID:     runID,
			EventType: "mm_run_diff_summary",
			Level:     "info",
			Message:   "post-run repository diff summary",
			Data: map[string]any{
				"changed_files": diffSummary.ChangedFiles,
				"added_lines":   diffSummary.AddedLines,
				"deleted_lines": diffSummary.DeletedLines,
			},
		})
		_ = client.FinishRun(context.Background(), telemetry.FinishRunInput{
			RunID:     runID,
			Status:    status,
			LastError: redactor.Apply(lastErr),
		})
	}

	return RunResult{
		RunID:         runID,
		RepoRoot:      repoRoot,
		Bundle:        bundle,
		Prompt:        finalPrompt,
		PromptHash:    promptHash,
		Runner:        runResult,
		DiffSummary:   diffSummary,
		Status:        status,
		Outcome:       outcome,
		LastError:     lastErr,
		AgentID:       agentID,
		SelectedEntry: entries,
	}, nil
}

func SendFeedback(ctx context.Context, cfg mmconfig.Config, runID string, rating int, notes string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" || rating <= 0 {
		return nil
	}
	token := mmconfig.ResolveToken(cfg)
	if strings.TrimSpace(token) == "" {
		return nil
	}
	client, err := telemetry.New(cfg, token)
	if err != nil {
		return err
	}
	defer client.Close()

	redactor := redact.New(cfg.RedactionEnabled, cfg.CustomRedactRegexes)
	return client.RecordRunEvent(ctx, telemetry.EventInput{
		RunID:     runID,
		EventType: "mm_feedback",
		Level:     "info",
		Message:   "post-run user feedback",
		Data: map[string]any{
			"rating": rating,
			"notes":  redactor.Apply(notes),
		},
	})
}

func loadSkillSnippet(repoRoot, skill string) string {
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return ""
	}
	paths := []string{
		filepath.Join(repoRoot, ".modeloman", "skills", skill+".md"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "modeloman", "skills", skill+".md"))
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err == nil {
			return string(raw)
		}
	}
	return ""
}

func localAgentID() string {
	host, _ := os.Hostname()
	currentUser, _ := user.Current()
	u := "unknown-user"
	if currentUser != nil && currentUser.Username != "" {
		u = currentUser.Username
	}
	if host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("mm@%s:%s", host, u)
}

func digestString(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func mergeEntries(base, extra []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range append(append([]string{}, base...), extra...) {
		clean := strings.TrimSpace(item)
		if clean == "" {
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
