package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mmconfig "github.com/bcrosbie/modeloman/internal/mm/config"
	mmcontext "github.com/bcrosbie/modeloman/internal/mm/context"
	"github.com/bcrosbie/modeloman/internal/mm/gitutil"
	"github.com/bcrosbie/modeloman/internal/mm/ui"
	"github.com/bcrosbie/modeloman/internal/mm/workflow"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func Run(args []string, commandName string) error {
	log.SetFlags(0)

	cfg, cfgPath, err := mmconfig.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	if len(args) < 1 {
		usage(commandName, cfgPath)
		return nil
	}

	switch args[0] {
	case "run":
		return runCommand(cfg, args[1:])
	case "tui":
		return ui.Run(cfg)
	case "add":
		return addCommand(args[1:])
	case "drop":
		return dropCommand(args[1:])
	case "list":
		return listCommand()
	case "clear":
		return clearCommand()
	default:
		usage(commandName, cfgPath)
		return nil
	}
}

func runCommand(cfg mmconfig.Config, args []string) error {
	backend := strings.TrimSpace(cfg.DefaultBackend)
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		backend = strings.TrimSpace(args[0])
		args = args[1:]
	}
	if backend == "" {
		return fmt.Errorf("backend is required (example: mm run codex --task bugfix)")
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	taskType := flags.String("task", "general-coding", "task type")
	skill := flags.String("skill", "", "skill name")
	var addList stringList
	flags.Var(&addList, "add", "additional path or glob for this run")
	budget := flags.Int("budget", 0, "optional token budget")
	dryRun := flags.Bool("dry-run", false, "render and log only")
	ptyMode := flags.Bool("pty", true, "run backend with PTY for interactive tools")
	promptText := ""
	flags.StringVar(&promptText, "p", "", "prompt text")
	flags.StringVar(&promptText, "prompt", "", "prompt text")
	flags.StringVar(&promptText, "objective", "", "objective prompt text")
	promptFile := flags.String("prompt-file", "", "path to prompt file, or - for stdin")
	if err := flags.Parse(args); err != nil {
		return err
	}
	objective, err := resolvePromptInput(promptText, *promptFile)
	if err != nil {
		return err
	}

	result, err := workflow.Run(context.Background(), cfg, workflow.RunParams{
		Backend:         backend,
		TaskType:        strings.TrimSpace(*taskType),
		Skill:           strings.TrimSpace(*skill),
		Objective:       objective,
		BudgetTokens:    *budget,
		DryRun:          *dryRun,
		UsePTY:          *ptyMode,
		ForwardInput:    true,
		AdditionalEntry: addList,
		OutputWriter:    os.Stdout,
	})
	if err != nil {
		return err
	}

	fmt.Printf("exit=%d duration=%s changed_files=%d run_id=%s\n",
		result.Runner.ExitCode,
		result.Runner.Duration.Round(time.Millisecond),
		len(result.DiffSummary.ChangedFiles),
		result.RunID,
	)

	rating, notes := askFeedback()
	if rating > 0 && strings.TrimSpace(result.RunID) != "" {
		_ = workflow.SendFeedback(context.Background(), cfg, result.RunID, rating, notes)
	}
	return nil
}

func resolvePromptInput(promptText, promptFile string) (string, error) {
	return resolvePromptInputWithReader(promptText, promptFile, os.Stdin, stdinHasData())
}

func resolvePromptInputWithReader(promptText, promptFile string, stdin io.Reader, hasStdin bool) (string, error) {
	inline := strings.TrimSpace(promptText)
	file := strings.TrimSpace(promptFile)
	if inline != "" && file != "" {
		return "", fmt.Errorf("prompt text and prompt file are mutually exclusive")
	}
	if file != "" {
		raw, err := readPromptSource(file, stdin)
		if err != nil {
			return "", err
		}
		prompt := strings.TrimSpace(raw)
		if prompt == "" {
			return "", fmt.Errorf("prompt file is empty")
		}
		return prompt, nil
	}
	if inline != "" {
		return inline, nil
	}
	if hasStdin {
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read prompt from stdin: %w", err)
		}
		prompt := strings.TrimSpace(string(raw))
		if prompt == "" {
			return "", fmt.Errorf("stdin prompt is empty")
		}
		return prompt, nil
	}
	return "", fmt.Errorf("prompt is required; use -p, --objective, --prompt-file, or pipe stdin")
}

func readPromptSource(path string, stdin io.Reader) (string, error) {
	if path == "-" {
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read prompt from stdin: %w", err)
		}
		return string(raw), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return string(raw), nil
}

func stdinHasData() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func addCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mm add PATH|GLOB ...")
	}
	repoRoot, err := gitutil.DetectRepoRoot()
	if err != nil {
		return err
	}
	cfg, err := mmcontext.Add(repoRoot, args)
	if err != nil {
		return err
	}
	fmt.Printf("saved %d context entries to %s\n", len(cfg.Entries), filepath.Join(repoRoot, ".modeloman/context.json"))
	return nil
}

func dropCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: mm drop PATH|GLOB ...")
	}
	repoRoot, err := gitutil.DetectRepoRoot()
	if err != nil {
		return err
	}
	cfg, err := mmcontext.Drop(repoRoot, args)
	if err != nil {
		return err
	}
	fmt.Printf("remaining context entries: %d\n", len(cfg.Entries))
	return nil
}

func listCommand() error {
	repoRoot, err := gitutil.DetectRepoRoot()
	if err != nil {
		return err
	}
	cfg, err := mmcontext.Load(repoRoot)
	if err != nil {
		return err
	}
	if len(cfg.Entries) == 0 {
		fmt.Println("no context entries set")
		return nil
	}
	for _, item := range cfg.Entries {
		fmt.Println(item)
	}
	return nil
}

func clearCommand() error {
	repoRoot, err := gitutil.DetectRepoRoot()
	if err != nil {
		return err
	}
	if err := mmcontext.Clear(repoRoot); err != nil {
		return err
	}
	fmt.Println("cleared repo context selection")
	return nil
}

func askLine(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func askFeedback() (int, string) {
	if !stdinIsTerminal() {
		return 0, ""
	}
	raw := askLine("Rate this run (1-5, Enter to skip): ")
	if strings.TrimSpace(raw) == "" {
		return 0, ""
	}
	rating, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || rating < 1 || rating > 5 {
		return 0, ""
	}
	notes := askLine("Feedback notes (optional): ")
	return rating, notes
}

func usage(commandName, configPath string) {
	fmt.Printf(`%s - ModeloMan workflow wrapper

Usage:
  %s run <backend> [--task TYPE] [--skill NAME] [--add PATH|GLOB ...] [--budget TOKENS] [--dry-run] [--pty=true] [-p "text" | --prompt-file PATH | < stdin]
  %s tui
  %s add PATH|GLOB ...
  %s drop PATH|GLOB ...
  %s list
  %s clear

Config file:
  %s
`, commandName, commandName, commandName, commandName, commandName, commandName, commandName, configPath)
}
