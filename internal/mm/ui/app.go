package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	mmconfig "github.com/bcrosbie/modeloman/internal/mm/config"
	mmcontext "github.com/bcrosbie/modeloman/internal/mm/context"
	"github.com/bcrosbie/modeloman/internal/mm/gitutil"
	"github.com/bcrosbie/modeloman/internal/mm/prompt"
	"github.com/bcrosbie/modeloman/internal/mm/runner"
	"github.com/bcrosbie/modeloman/internal/mm/workflow"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenHome screen = iota
	screenContext
	screenPreview
	screenRun
	screenPost
)

type filesLoadedMsg struct {
	files []string
	err   error
}

type runOutputMsg string
type runEventMsg runner.Event

type runCompleteMsg struct {
	result workflow.RunResult
	err    error
}

type tickMsg time.Time

type feedbackSavedMsg struct {
	err error
}

type model struct {
	cfg        mmconfig.Config
	repoRoot   string
	ctxStore   mmcontext.RepoContext
	uiState    mmcontext.UIState
	screen     screen
	width      int
	height     int
	statusLine string

	backends []string
	backend  int

	taskInput      textinput.Model
	skillInput     textinput.Model
	budgetInput    textinput.Model
	objectiveInput textarea.Model
	homeFocus      int

	filterInput textinput.Model
	allFiles    []string
	filtered    []string
	selected    map[string]struct{}
	cursor      int
	filesReady  bool

	previewBundle mmcontext.Bundle
	previewPrompt string

	runOutput      strings.Builder
	runStartedAt   time.Time
	runResult      workflow.RunResult
	runErr         error
	runDone        bool
	runInProgress  bool
	runPassthrough bool
	runCtxCancel   context.CancelFunc
	runInputWriter *io.PipeWriter
	runOutputCh    chan string
	runEventCh     chan runner.Event
	runDoneCh      chan runCompleteMsg
	lastEventLines []string

	ratingInput textinput.Model
	notesInput  textarea.Model
	postFocus   int
	coach       coachOutput
}

type coachOutput struct {
	improvements []string
	questions    []string
	snippet      string
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func Run(cfg mmconfig.Config) error {
	repoRoot, err := gitutil.DetectRepoRoot()
	if err != nil {
		return err
	}
	ctxStore, err := mmcontext.Load(repoRoot)
	if err != nil {
		return err
	}
	uiState, err := mmcontext.LoadUIState(repoRoot)
	if err != nil {
		return err
	}

	taskInput := textinput.New()
	taskInput.Placeholder = "task type"
	taskInput.SetValue(defaultString(uiState.TaskType, "general-coding"))
	taskInput.Prompt = "Task: "

	skillInput := textinput.New()
	skillInput.Placeholder = "skill name"
	skillInput.SetValue(uiState.Skill)
	skillInput.Prompt = "Skill: "

	budgetInput := textinput.New()
	budgetInput.Placeholder = "token budget (optional)"
	if uiState.Budget > 0 {
		budgetInput.SetValue(strconv.Itoa(uiState.Budget))
	}
	budgetInput.Prompt = "Budget: "

	objectiveInput := textarea.New()
	objectiveInput.Placeholder = "Describe objective..."
	objectiveInput.SetValue(uiState.Objective)
	objectiveInput.Prompt = ""
	objectiveInput.SetHeight(6)
	objectiveInput.SetWidth(96)

	filterInput := textinput.New()
	filterInput.Prompt = "Filter files: "
	filterInput.Placeholder = "type to fuzzy filter"
	filterInput.Focus()

	ratingInput := textinput.New()
	ratingInput.Prompt = "Rating (1-5): "
	ratingInput.Placeholder = "1-5"
	ratingInput.Width = 6

	notesInput := textarea.New()
	notesInput.Prompt = ""
	notesInput.Placeholder = "Notes..."
	notesInput.SetHeight(4)
	notesInput.SetWidth(96)

	m := model{
		cfg:            cfg,
		repoRoot:       repoRoot,
		ctxStore:       ctxStore,
		uiState:        uiState,
		screen:         screenHome,
		backends:       []string{"codex", "claude", "gemini", "opencode"},
		taskInput:      taskInput,
		skillInput:     skillInput,
		budgetInput:    budgetInput,
		objectiveInput: objectiveInput,
		filterInput:    filterInput,
		selected:       map[string]struct{}{},
		ratingInput:    ratingInput,
		notesInput:     notesInput,
		statusLine:     "Tab through fields. Enter for context picker.",
	}
	for i, backend := range m.backends {
		if backend == uiState.Backend {
			m.backend = i
			break
		}
	}

	for _, entry := range ctxStore.Entries {
		m.selected[entry] = struct{}{}
	}
	for _, item := range uiState.LastFiles {
		m.selected[item] = struct{}{}
	}
	m.applyHomeFocus()
	m.applyPostFocus()

	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		loadFilesCmd(m.repoRoot),
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.objectiveInput.SetWidth(maxInt(40, typed.Width-8))
		m.notesInput.SetWidth(maxInt(40, typed.Width-8))
		return m, nil
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" {
			if m.runInProgress && m.runCtxCancel != nil {
				m.runCtxCancel()
			}
			if m.runInputWriter != nil {
				_ = m.runInputWriter.Close()
				m.runInputWriter = nil
			}
			return m, tea.Quit
		}
	case filesLoadedMsg:
		if typed.err != nil {
			m.statusLine = "file scan error: " + typed.err.Error()
			return m, nil
		}
		m.filesReady = true
		m.allFiles = typed.files
		m.filtered = applyFilter(typed.files, m.filterInput.Value())
		if m.cursor >= len(m.filtered) {
			m.cursor = maxInt(0, len(m.filtered)-1)
		}
		return m, nil
	case runOutputMsg:
		if m.runOutput.Len() < m.cfg.MaxTranscriptBytes {
			chunk := string(typed)
			if m.runOutput.Len()+len(chunk) > m.cfg.MaxTranscriptBytes {
				chunk = chunk[:m.cfg.MaxTranscriptBytes-m.runOutput.Len()]
			}
			m.runOutput.WriteString(chunk)
		}
		return m, waitRunOutputCmd(m.runOutputCh)
	case runEventMsg:
		event := runner.Event(typed)
		m.lastEventLines = append(m.lastEventLines, fmt.Sprintf("%s %s", event.Type, event.Message))
		if len(m.lastEventLines) > 8 {
			m.lastEventLines = m.lastEventLines[len(m.lastEventLines)-8:]
		}
		return m, waitRunEventCmd(m.runEventCh)
	case runCompleteMsg:
		m.runDone = true
		m.runInProgress = false
		m.runPassthrough = false
		m.runResult = typed.result
		m.runErr = typed.err
		if m.runInputWriter != nil {
			_ = m.runInputWriter.Close()
			m.runInputWriter = nil
		}
		m.screen = screenPost
		if m.runErr != nil {
			m.statusLine = "run failed: " + m.runErr.Error()
		} else {
			m.statusLine = "run complete"
		}
		m.coach = buildCoach(typed.result)
		m.applyPostFocus()
		return m, nil
	case tickMsg:
		if m.runInProgress {
			return m, tickCmd()
		}
		return m, nil
	case feedbackSavedMsg:
		if typed.err != nil {
			m.statusLine = "feedback failed: " + typed.err.Error()
		} else {
			m.statusLine = "feedback saved"
		}
		return m, nil
	}

	switch m.screen {
	case screenHome:
		return m.updateHome(msg)
	case screenContext:
		return m.updateContext(msg)
	case screenPreview:
		return m.updatePreview(msg)
	case screenRun:
		return m.updateRun(msg)
	case screenPost:
		return m.updatePost(msg)
	default:
		return m, nil
	}
}

func (m model) View() string {
	var body string
	switch m.screen {
	case screenHome:
		body = m.viewHome()
	case screenContext:
		body = m.viewContext()
	case screenPreview:
		body = m.viewPreview()
	case screenRun:
		body = m.viewRun()
	case screenPost:
		body = m.viewPost()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("ModeloMan TUI"),
		mutedStyle.Render("Repo: "+m.repoRoot),
		"",
		body,
		"",
		mutedStyle.Render(m.statusLine),
	)
}

func (m model) updateHome(msg tea.Msg) (model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "tab":
			m.homeFocus = (m.homeFocus + 1) % 4
			m.applyHomeFocus()
			return m, nil
		case "shift+tab":
			m.homeFocus = (m.homeFocus + 3) % 4
			m.applyHomeFocus()
			return m, nil
		case "[":
			m.backend = (m.backend + len(m.backends) - 1) % len(m.backends)
			return m, nil
		case "]":
			m.backend = (m.backend + 1) % len(m.backends)
			return m, nil
		case "enter":
			m.persistHomeState()
			m.screen = screenContext
			m.statusLine = "Context picker: / filter, space toggle, enter preview"
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.homeFocus {
	case 0:
		m.taskInput, cmd = m.taskInput.Update(msg)
	case 1:
		m.skillInput, cmd = m.skillInput.Update(msg)
	case 2:
		m.budgetInput, cmd = m.budgetInput.Update(msg)
	case 3:
		m.objectiveInput, cmd = m.objectiveInput.Update(msg)
	}
	return m, cmd
}

func (m model) updateContext(msg tea.Msg) (model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case " ":
			if len(m.filtered) > 0 {
				file := m.filtered[m.cursor]
				if _, ok := m.selected[file]; ok {
					delete(m.selected, file)
				} else {
					m.selected[file] = struct{}{}
				}
				m.persistSelections()
			}
			return m, nil
		case "ctrl+s":
			m.persistSelections()
			m.statusLine = "saved context selection"
			return m, nil
		case "enter":
			m.persistSelections()
			bundle, preview, err := m.buildPreview()
			if err != nil {
				m.statusLine = "preview error: " + err.Error()
				return m, nil
			}
			m.previewBundle = bundle
			m.previewPrompt = preview
			m.screen = screenPreview
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filtered = applyFilter(m.allFiles, m.filterInput.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = maxInt(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m model) updatePreview(msg tea.Msg) (model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "esc":
			m.screen = screenContext
			return m, nil
		case "r":
			bundle, preview, err := m.buildPreview()
			if err != nil {
				m.statusLine = "run prep failed: " + err.Error()
				return m, nil
			}
			m.previewBundle = bundle
			m.previewPrompt = preview
			return m.startRun()
		}
	}
	return m, nil
}

func (m model) updateRun(msg tea.Msg) (model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if !m.runInProgress {
			return m, nil
		}

		if typed.String() == "i" {
			m.runPassthrough = !m.runPassthrough
			if m.runPassthrough {
				m.statusLine = "passthrough ON (keys go to backend). ctrl+g exits passthrough."
			} else {
				m.statusLine = "passthrough OFF. q cancels run."
			}
			return m, nil
		}

		if m.runPassthrough {
			if typed.String() == "ctrl+g" {
				m.runPassthrough = false
				m.statusLine = "passthrough OFF. q cancels run."
				return m, nil
			}
			if m.runInputWriter != nil {
				if payload, ok := keyToBytes(typed); ok {
					_, _ = m.runInputWriter.Write(payload)
				}
			}
			return m, nil
		}

		if typed.String() == "q" {
			if m.runCtxCancel != nil {
				m.runCtxCancel()
				m.statusLine = "cancellation requested"
			}
			if m.runInputWriter != nil {
				_ = m.runInputWriter.Close()
				m.runInputWriter = nil
			}
		}
	}
	return m, nil
}

func (m model) updatePost(msg tea.Msg) (model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "esc":
			m.screen = screenHome
			return m, nil
		case "tab":
			m.postFocus = (m.postFocus + 1) % 2
			m.applyPostFocus()
			return m, nil
		case "shift+tab":
			m.postFocus = (m.postFocus + 1) % 2
			m.applyPostFocus()
			return m, nil
		case "enter":
			rating, _ := strconv.Atoi(strings.TrimSpace(m.ratingInput.Value()))
			if rating < 1 || rating > 5 || strings.TrimSpace(m.runResult.RunID) == "" {
				m.statusLine = "rating must be 1-5 and run_id present"
				return m, nil
			}
			notes := m.notesInput.Value()
			return m, saveFeedbackCmd(m.cfg, m.runResult.RunID, rating, notes)
		}
	}
	var cmd tea.Cmd
	if m.postFocus == 0 {
		m.ratingInput, cmd = m.ratingInput.Update(msg)
	} else {
		m.notesInput, cmd = m.notesInput.Update(msg)
	}
	return m, cmd
}

func (m model) viewHome() string {
	lines := []string{
		sectionStyle.Render("Home"),
		"Backend: " + m.backends[m.backend] + "  [ / ] to cycle",
		focusPrefix(m.homeFocus == 0) + m.taskInput.View(),
		focusPrefix(m.homeFocus == 1) + m.skillInput.View(),
		focusPrefix(m.homeFocus == 2) + m.budgetInput.View(),
		focusPrefix(m.homeFocus == 3) + "Objective:",
		m.objectiveInput.View(),
		"",
		mutedStyle.Render("Enter: Context Picker | Tab: next field | Ctrl+C: quit"),
	}
	return strings.Join(lines, "\n")
}

func (m model) viewContext() string {
	lines := []string{
		sectionStyle.Render("Context Picker"),
		m.filterInput.View(),
		fmt.Sprintf("Files: %d filtered / %d total | Selected: %d", len(m.filtered), len(m.allFiles), len(m.selected)),
		"",
	}
	start := maxInt(0, m.cursor-10)
	end := minInt(len(m.filtered), start+20)
	for i := start; i < end; i++ {
		item := m.filtered[i]
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		mark := "[ ]"
		if _, ok := m.selected[item]; ok {
			mark = "[x]"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", cursor, mark, item))
	}
	lines = append(lines, "", mutedStyle.Render("space: toggle | enter: preview | ctrl+s: save | esc: back"))
	return strings.Join(lines, "\n")
}

func (m model) viewPreview() string {
	promptPreview := m.previewPrompt
	if len(promptPreview) > 1200 {
		promptPreview = promptPreview[:1200] + "\n...[truncated]"
	}
	lines := []string{
		sectionStyle.Render("Preview"),
		fmt.Sprintf("Selected files: %d", len(m.previewBundle.SelectedFiles)),
		fmt.Sprintf("Context bytes: %d", m.previewBundle.RenderedBytes),
		fmt.Sprintf("Estimated tokens: %d", m.previewBundle.EstimatedToken),
		fmt.Sprintf("Context hash: %s", m.previewBundle.Hash),
		"",
		sectionStyle.Render("Prompt"),
		promptPreview,
		"",
		mutedStyle.Render("r: run | esc: back"),
	}
	return strings.Join(lines, "\n")
}

func (m model) viewRun() string {
	elapsed := time.Since(m.runStartedAt).Round(time.Second)
	output := m.runOutput.String()
	if len(output) > 5000 {
		output = output[len(output)-5000:]
	}
	lines := []string{
		sectionStyle.Render("Run"),
		fmt.Sprintf("Elapsed: %s | In progress: %v | RunID: %s", elapsed, m.runInProgress, m.runResult.RunID),
		mutedStyle.Render("i: toggle passthrough | ctrl+g: exit passthrough | q: cancel"),
		mutedStyle.Render(fmt.Sprintf("Passthrough: %v", m.runPassthrough)),
		"",
		sectionStyle.Render("Output (tail)"),
		output,
		"",
		sectionStyle.Render("Runner Events"),
		strings.Join(m.lastEventLines, "\n"),
	}
	return strings.Join(lines, "\n")
}

func (m model) viewPost() string {
	status := okStyle.Render(m.runResult.Status)
	if m.runResult.Status != "completed" {
		status = errStyle.Render(m.runResult.Status)
	}
	lines := []string{
		sectionStyle.Render("Post-run"),
		fmt.Sprintf("RunID: %s", m.runResult.RunID),
		fmt.Sprintf("Status: %s", status),
		fmt.Sprintf("Exit code: %d", m.runResult.Runner.ExitCode),
		fmt.Sprintf("Duration: %s", m.runResult.Runner.Duration.Round(time.Millisecond)),
		fmt.Sprintf("Changed files: %d (+%d/-%d)",
			len(m.runResult.DiffSummary.ChangedFiles),
			m.runResult.DiffSummary.AddedLines,
			m.runResult.DiffSummary.DeletedLines,
		),
		"",
		sectionStyle.Render("Changed Files"),
		strings.Join(m.runResult.DiffSummary.ChangedFiles, "\n"),
		"",
		sectionStyle.Render("Rating + Notes"),
		focusPrefix(m.postFocus == 0) + m.ratingInput.View(),
		focusPrefix(m.postFocus == 1) + "Notes:",
		m.notesInput.View(),
		"",
		sectionStyle.Render("Prompt Coach"),
		"Improvements:",
		" - " + strings.Join(m.coach.improvements, "\n - "),
		"Questions:",
		" - " + strings.Join(m.coach.questions, "\n - "),
		"Suggested Skill Snippet:\n" + m.coach.snippet,
		"",
		mutedStyle.Render("tab: next field | enter: submit feedback | esc: home"),
	}
	return strings.Join(lines, "\n")
}

func (m model) startRun() (model, tea.Cmd) {
	m.screen = screenRun
	m.runOutput.Reset()
	m.lastEventLines = nil
	m.runDone = false
	m.runInProgress = true
	m.runPassthrough = false
	m.runStartedAt = time.Now().UTC()
	m.statusLine = "run started"

	runCtx, cancel := context.WithCancel(context.Background())
	m.runCtxCancel = cancel
	m.runOutputCh = make(chan string, 128)
	m.runEventCh = make(chan runner.Event, 128)
	m.runDoneCh = make(chan runCompleteMsg, 1)
	inputReader, inputWriter := io.Pipe()
	m.runInputWriter = inputWriter

	backend := m.backends[m.backend]
	taskType := strings.TrimSpace(m.taskInput.Value())
	budget, _ := strconv.Atoi(strings.TrimSpace(m.budgetInput.Value()))
	objective := strings.TrimSpace(m.objectiveInput.Value())
	skill := strings.TrimSpace(m.skillInput.Value())
	selectedEntries := m.selectedEntries()
	m.persistSelections()
	m.persistHomeState()

	go func() {
		result, err := workflow.Run(runCtx, m.cfg, workflow.RunParams{
			Backend:         backend,
			TaskType:        taskType,
			Skill:           skill,
			Objective:       objective,
			BudgetTokens:    budget,
			DryRun:          false,
			UsePTY:          true,
			ForwardInput:    true,
			InputReader:     inputReader,
			AdditionalEntry: selectedEntries,
			RepoRoot:        m.repoRoot,
			OutputWriter:    io.Discard,
			OnOutput: func(chunk string) {
				select {
				case m.runOutputCh <- chunk:
				default:
				}
			},
			OnRunnerEvent: func(event runner.Event) {
				select {
				case m.runEventCh <- event:
				default:
				}
			},
		})
		m.runDoneCh <- runCompleteMsg{result: result, err: err}
		close(m.runOutputCh)
		close(m.runEventCh)
		close(m.runDoneCh)
	}()

	return m, tea.Batch(
		waitRunOutputCmd(m.runOutputCh),
		waitRunEventCmd(m.runEventCh),
		waitRunDoneCmd(m.runDoneCh),
		tickCmd(),
	)
}

func (m *model) applyHomeFocus() {
	m.taskInput.Blur()
	m.skillInput.Blur()
	m.budgetInput.Blur()
	m.objectiveInput.Blur()
	switch m.homeFocus {
	case 0:
		m.taskInput.Focus()
	case 1:
		m.skillInput.Focus()
	case 2:
		m.budgetInput.Focus()
	case 3:
		m.objectiveInput.Focus()
	}
}

func (m *model) applyPostFocus() {
	m.ratingInput.Blur()
	m.notesInput.Blur()
	if m.postFocus == 0 {
		m.ratingInput.Focus()
	} else {
		m.notesInput.Focus()
	}
}

func (m *model) persistSelections() {
	entries := m.selectedEntries()
	_ = mmcontext.Save(m.repoRoot, mmcontext.RepoContext{
		Version: 1,
		Entries: entries,
	})
	m.uiState.LastFiles = entries
	_ = mmcontext.SaveUIState(m.repoRoot, m.uiState)
}

func (m *model) persistHomeState() {
	m.uiState.Backend = m.backends[m.backend]
	m.uiState.TaskType = strings.TrimSpace(m.taskInput.Value())
	m.uiState.Skill = strings.TrimSpace(m.skillInput.Value())
	m.uiState.Objective = strings.TrimSpace(m.objectiveInput.Value())
	m.uiState.Budget, _ = strconv.Atoi(strings.TrimSpace(m.budgetInput.Value()))
	m.uiState.LastScreen = fmt.Sprintf("%d", m.screen)
	_ = mmcontext.SaveUIState(m.repoRoot, m.uiState)
}

func (m model) selectedEntries() []string {
	out := make([]string, 0, len(m.selected))
	for item := range m.selected {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func (m model) buildPreview() (mmcontext.Bundle, string, error) {
	budget, _ := strconv.Atoi(strings.TrimSpace(m.budgetInput.Value()))
	bundle, err := mmcontext.BuildBundle(mmcontext.BuildOptions{
		RepoRoot:    m.repoRoot,
		Entries:     m.selectedEntries(),
		Prompt:      strings.TrimSpace(m.objectiveInput.Value()),
		MaxBytes:    m.cfg.MaxContextBytes,
		TokenBudget: budget,
	})
	if err != nil {
		return mmcontext.Bundle{}, "", err
	}
	template := prompt.Build(prompt.TemplateInput{
		Objective:      strings.TrimSpace(m.objectiveInput.Value()),
		TaskType:       strings.TrimSpace(m.taskInput.Value()),
		SkillName:      strings.TrimSpace(m.skillInput.Value()),
		ContextDigest:  bundle.Hash,
		Backend:        m.backends[m.backend],
		BudgetTokens:   budget,
		AdditionalHint: "- Keep changes minimal.\n- Verify with tests.",
	})
	return bundle, template, nil
}

func loadFilesCmd(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		files, err := scanRepoFiles(repoRoot)
		return filesLoadedMsg{files: files, err: err}
	}
}

func waitRunOutputCmd(ch <-chan string) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		item, ok := <-ch
		if !ok {
			return nil
		}
		return runOutputMsg(item)
	}
}

func waitRunEventCmd(ch <-chan runner.Event) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		item, ok := <-ch
		if !ok {
			return nil
		}
		return runEventMsg(item)
	}
}

func waitRunDoneCmd(ch <-chan runCompleteMsg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		item, ok := <-ch
		if !ok {
			return nil
		}
		return item
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func saveFeedbackCmd(cfg mmconfig.Config, runID string, rating int, notes string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := workflow.SendFeedback(ctx, cfg, runID, rating, notes)
		return feedbackSavedMsg{err: err}
	}
}

func applyFilter(files []string, query string) []string {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return append([]string{}, files...)
	}
	out := make([]string, 0, len(files))
	for _, file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, query) || fuzzyContains(lower, query) {
			out = append(out, file)
		}
	}
	return out
}

func scanRepoFiles(repoRoot string) ([]string, error) {
	files := make([]string, 0, 8192)
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "vendor", "bin", ".next", "target", ".idea", ".vscode":
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func fuzzyContains(value, query string) bool {
	if query == "" {
		return true
	}
	j := 0
	for i := 0; i < len(value) && j < len(query); i++ {
		if value[i] == query[j] {
			j++
		}
	}
	return j == len(query)
}

func keyToBytes(msg tea.KeyMsg) ([]byte, bool) {
	switch msg.Type {
	case tea.KeyRunes:
		return []byte(string(msg.Runes)), true
	case tea.KeySpace:
		return []byte(" "), true
	case tea.KeyEnter:
		return []byte("\n"), true
	case tea.KeyTab:
		return []byte("\t"), true
	case tea.KeyBackspace, tea.KeyDelete:
		return []byte{0x7f}, true
	case tea.KeyUp:
		return []byte("\x1b[A"), true
	case tea.KeyDown:
		return []byte("\x1b[B"), true
	case tea.KeyRight:
		return []byte("\x1b[C"), true
	case tea.KeyLeft:
		return []byte("\x1b[D"), true
	case tea.KeyEsc:
		return []byte{0x1b}, true
	}

	switch msg.String() {
	case "ctrl+c":
		return []byte{0x03}, true
	case "ctrl+d":
		return []byte{0x04}, true
	case "ctrl+z":
		return []byte{0x1a}, true
	}
	return nil, false
}

func buildCoach(result workflow.RunResult) coachOutput {
	improvements := []string{}
	lowerPrompt := strings.ToLower(result.Prompt)
	if !strings.Contains(lowerPrompt, "acceptance") && !strings.Contains(lowerPrompt, "definition of done") {
		improvements = append(improvements, "Add explicit acceptance criteria to reduce retries.")
	}
	if !strings.Contains(lowerPrompt, "test") {
		improvements = append(improvements, "Include concrete test commands and expected outcomes.")
	}
	if result.Bundle.RenderedBytes > 250000 {
		improvements = append(improvements, "Context bundle is heavy; narrow to likely touched files.")
	}
	if len(improvements) < 3 {
		improvements = append(improvements, "Use stronger action verbs and exact outputs to reduce ambiguity.")
	}
	if len(improvements) < 3 {
		improvements = append(improvements, "Pin constraints for safety, budget, and completion criteria.")
	}

	questions := []string{
		"Which artifact proves completion for this run?",
		"What is the smallest test that would fail before this change and pass after?",
	}
	snippet := "## Skill Notes\n- Goal:\n- Guardrails:\n- Required checks:\n- Expected deliverables:\n"
	return coachOutput{
		improvements: improvements[:3],
		questions:    questions,
		snippet:      snippet,
	}
}

func focusPrefix(active bool) string {
	if active {
		return "> "
	}
	return "  "
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
