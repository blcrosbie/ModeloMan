package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/bcrosbie/modeloman/internal/domain"
	"github.com/bcrosbie/modeloman/internal/store"
)

var (
	validTaskStatuses     = map[string]struct{}{"todo": {}, "in_progress": {}, "done": {}, "blocked": {}}
	validProviderTypes    = map[string]struct{}{"api": {}, "subscription": {}, "opensource": {}}
	validRunStatuses      = map[string]struct{}{"running": {}, "completed": {}, "failed": {}, "cancelled": {}}
	validAttemptOutcomes  = map[string]struct{}{"success": {}, "failed": {}, "timeout": {}, "retryable_error": {}, "tool_error": {}}
	validEventLevels      = map[string]struct{}{"info": {}, "warn": {}, "error": {}}
	validChangeCategories = map[string]struct{}{
		"platform": {},
		"policy":   {},
		"model":    {},
		"infra":    {},
		"ops":      {},
	}
)

type HubService struct {
	store      store.HubStore
	dataSource string
}

func NewHubService(store store.HubStore, dataSource string) *HubService {
	return &HubService{
		store:      store,
		dataSource: dataSource,
	}
}

type CreateTaskRequest struct {
	Title   string   `json:"title"`
	Details string   `json:"details"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
}

type UpdateTaskRequest struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Details string   `json:"details"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
}

type DeleteTaskRequest struct {
	ID string `json:"id"`
}

type CreateNoteRequest struct {
	Title string   `json:"title"`
	Body  string   `json:"body"`
	Tags  []string `json:"tags"`
}

type AppendChangelogRequest struct {
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Details  string `json:"details"`
	Actor    string `json:"actor"`
}

type RecordBenchmarkRequest struct {
	Workflow     string  `json:"workflow"`
	ProviderType string  `json:"provider_type"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	TokensIn     int64   `json:"tokens_in"`
	TokensOut    int64   `json:"tokens_out"`
	CostUSD      float64 `json:"cost_usd"`
	LatencyMS    int64   `json:"latency_ms"`
	QualityScore float64 `json:"quality_score"`
	Notes        string  `json:"notes"`
}

type StartRunRequest struct {
	TaskID        string `json:"task_id"`
	Workflow      string `json:"workflow"`
	AgentID       string `json:"agent_id"`
	PromptVersion string `json:"prompt_version"`
	ModelPolicy   string `json:"model_policy"`
	MaxRetries    int64  `json:"max_retries"`
}

type FinishRunRequest struct {
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	LastError string `json:"last_error"`
}

type RecordPromptAttemptRequest struct {
	RunID         string  `json:"run_id"`
	AttemptNumber int64   `json:"attempt_number"`
	Workflow      string  `json:"workflow"`
	AgentID       string  `json:"agent_id"`
	ProviderType  string  `json:"provider_type"`
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	PromptVersion string  `json:"prompt_version"`
	PromptHash    string  `json:"prompt_hash"`
	Outcome       string  `json:"outcome"`
	ErrorType     string  `json:"error_type"`
	ErrorMessage  string  `json:"error_message"`
	TokensIn      int64   `json:"tokens_in"`
	TokensOut     int64   `json:"tokens_out"`
	CostUSD       float64 `json:"cost_usd"`
	LatencyMS     int64   `json:"latency_ms"`
	QualityScore  float64 `json:"quality_score"`
}

type RecordRunEventRequest struct {
	RunID     string `json:"run_id"`
	EventType string `json:"event_type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	DataJSON  string `json:"data_json"`
}

type SetPolicyRequest struct {
	KillSwitch             *bool    `json:"kill_switch"`
	KillSwitchReason       *string  `json:"kill_switch_reason"`
	MaxCostPerRunUSD       *float64 `json:"max_cost_per_run_usd"`
	MaxAttemptsPerRun      *int64   `json:"max_attempts_per_run"`
	MaxTokensPerRun        *int64   `json:"max_tokens_per_run"`
	MaxLatencyPerAttemptMS *int64   `json:"max_latency_per_attempt_ms"`
}

type UpsertPolicyCapRequest struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	ProviderType           string   `json:"provider_type"`
	Provider               string   `json:"provider"`
	Model                  string   `json:"model"`
	MaxCostPerRunUSD       *float64 `json:"max_cost_per_run_usd"`
	MaxAttemptsPerRun      *int64   `json:"max_attempts_per_run"`
	MaxTokensPerRun        *int64   `json:"max_tokens_per_run"`
	MaxCostPerAttemptUSD   *float64 `json:"max_cost_per_attempt_usd"`
	MaxTokensPerAttempt    *int64   `json:"max_tokens_per_attempt"`
	MaxLatencyPerAttemptMS *int64   `json:"max_latency_per_attempt_ms"`
	Priority               *int64   `json:"priority"`
	DryRun                 *bool    `json:"dry_run"`
	IsActive               *bool    `json:"is_active"`
}

type DeletePolicyCapRequest struct {
	ID string `json:"id"`
}

type ListRunsRequest struct {
	RunID         string `json:"run_id"`
	TaskID        string `json:"task_id"`
	Workflow      string `json:"workflow"`
	AgentID       string `json:"agent_id"`
	Status        string `json:"status"`
	PromptVersion string `json:"prompt_version"`
	StartedAfter  string `json:"started_after"`
	StartedBefore string `json:"started_before"`
	Limit         int64  `json:"limit"`
}

type ListPromptAttemptsRequest struct {
	RunID         string `json:"run_id"`
	Workflow      string `json:"workflow"`
	AgentID       string `json:"agent_id"`
	Model         string `json:"model"`
	Outcome       string `json:"outcome"`
	PromptVersion string `json:"prompt_version"`
	CreatedAfter  string `json:"created_after"`
	CreatedBefore string `json:"created_before"`
	Limit         int64  `json:"limit"`
}

type ListRunEventsRequest struct {
	RunID         string `json:"run_id"`
	EventType     string `json:"event_type"`
	Level         string `json:"level"`
	CreatedAfter  string `json:"created_after"`
	CreatedBefore string `json:"created_before"`
	Limit         int64  `json:"limit"`
}

type LeaderboardRequest struct {
	Workflow      string `json:"workflow"`
	Model         string `json:"model"`
	PromptVersion string `json:"prompt_version"`
	WindowDays    int64  `json:"window_days"`
	Limit         int64  `json:"limit"`
}

type effectiveLimits struct {
	MaxCostPerRunUSD       float64
	MaxAttemptsPerRun      int64
	MaxTokensPerRun        int64
	MaxLatencyPerAttemptMS int64
	MaxCostPerAttemptUSD   float64
	MaxTokensPerAttempt    int64
	Source                 string
}

func (h *HubService) Health() map[string]any {
	return map[string]any{
		"status":      "ok",
		"data_source": h.dataSource,
		"time_utc":    time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (h *HubService) ExportState() (domain.State, error) {
	return h.store.ExportState()
}

func (h *HubService) GetPolicy() (domain.OrchestrationPolicy, error) {
	return h.store.GetPolicy()
}

func (h *HubService) SetPolicy(request SetPolicyRequest) (domain.OrchestrationPolicy, error) {
	policy, err := h.store.GetPolicy()
	if err != nil {
		return domain.OrchestrationPolicy{}, err
	}

	if request.KillSwitch != nil {
		policy.KillSwitch = *request.KillSwitch
	}
	if request.KillSwitchReason != nil {
		policy.KillSwitchReason = strings.TrimSpace(*request.KillSwitchReason)
	}
	if request.MaxCostPerRunUSD != nil {
		if *request.MaxCostPerRunUSD < 0 {
			return domain.OrchestrationPolicy{}, domain.InvalidArgument("max_cost_per_run_usd must be non-negative")
		}
		policy.MaxCostPerRunUSD = *request.MaxCostPerRunUSD
	}
	if request.MaxAttemptsPerRun != nil {
		if *request.MaxAttemptsPerRun < 0 {
			return domain.OrchestrationPolicy{}, domain.InvalidArgument("max_attempts_per_run must be non-negative")
		}
		policy.MaxAttemptsPerRun = *request.MaxAttemptsPerRun
	}
	if request.MaxTokensPerRun != nil {
		if *request.MaxTokensPerRun < 0 {
			return domain.OrchestrationPolicy{}, domain.InvalidArgument("max_tokens_per_run must be non-negative")
		}
		policy.MaxTokensPerRun = *request.MaxTokensPerRun
	}
	if request.MaxLatencyPerAttemptMS != nil {
		if *request.MaxLatencyPerAttemptMS < 0 {
			return domain.OrchestrationPolicy{}, domain.InvalidArgument("max_latency_per_attempt_ms must be non-negative")
		}
		policy.MaxLatencyPerAttemptMS = *request.MaxLatencyPerAttemptMS
	}

	policy.UpdatedAt = timeNow()
	if err := h.store.SetPolicy(policy); err != nil {
		return domain.OrchestrationPolicy{}, err
	}
	return h.store.GetPolicy()
}

func (h *HubService) ListPolicyCaps() ([]domain.PolicyCap, error) {
	items, err := h.store.ListPolicyCaps()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.PolicyCap) int {
		if a.Priority == b.Priority {
			return strings.Compare(a.ID, b.ID)
		}
		if a.Priority > b.Priority {
			return -1
		}
		return 1
	})
	return items, nil
}

func (h *HubService) UpsertPolicyCap(request UpsertPolicyCapRequest) (domain.PolicyCap, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		id = newID("cap")
	}
	providerType := strings.TrimSpace(request.ProviderType)
	if providerType != "" {
		if _, ok := validProviderTypes[providerType]; !ok {
			return domain.PolicyCap{}, domain.InvalidArgument("provider_type must be one of: api, subscription, opensource")
		}
	}

	current := domain.PolicyCap{
		ID:           id,
		Name:         strings.TrimSpace(request.Name),
		ProviderType: providerType,
		Provider:     strings.TrimSpace(request.Provider),
		Model:        strings.TrimSpace(request.Model),
		DryRun:       false,
		IsActive:     true,
		UpdatedAt:    timeNow(),
	}

	existing, err := h.store.ListPolicyCaps()
	if err != nil {
		return domain.PolicyCap{}, err
	}
	for _, item := range existing {
		if item.ID == id {
			current = item
			break
		}
	}

	if request.Name != "" {
		current.Name = strings.TrimSpace(request.Name)
	}
	if request.ProviderType != "" {
		current.ProviderType = providerType
	}
	if request.Provider != "" {
		current.Provider = strings.TrimSpace(request.Provider)
	}
	if request.Model != "" {
		current.Model = strings.TrimSpace(request.Model)
	}
	if request.MaxCostPerRunUSD != nil {
		if *request.MaxCostPerRunUSD < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_cost_per_run_usd must be non-negative")
		}
		current.MaxCostPerRunUSD = *request.MaxCostPerRunUSD
	}
	if request.MaxAttemptsPerRun != nil {
		if *request.MaxAttemptsPerRun < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_attempts_per_run must be non-negative")
		}
		current.MaxAttemptsPerRun = *request.MaxAttemptsPerRun
	}
	if request.MaxTokensPerRun != nil {
		if *request.MaxTokensPerRun < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_tokens_per_run must be non-negative")
		}
		current.MaxTokensPerRun = *request.MaxTokensPerRun
	}
	if request.MaxCostPerAttemptUSD != nil {
		if *request.MaxCostPerAttemptUSD < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_cost_per_attempt_usd must be non-negative")
		}
		current.MaxCostPerAttemptUSD = *request.MaxCostPerAttemptUSD
	}
	if request.MaxTokensPerAttempt != nil {
		if *request.MaxTokensPerAttempt < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_tokens_per_attempt must be non-negative")
		}
		current.MaxTokensPerAttempt = *request.MaxTokensPerAttempt
	}
	if request.MaxLatencyPerAttemptMS != nil {
		if *request.MaxLatencyPerAttemptMS < 0 {
			return domain.PolicyCap{}, domain.InvalidArgument("max_latency_per_attempt_ms must be non-negative")
		}
		current.MaxLatencyPerAttemptMS = *request.MaxLatencyPerAttemptMS
	}
	if request.Priority != nil {
		current.Priority = *request.Priority
	}
	if request.DryRun != nil {
		current.DryRun = *request.DryRun
	}
	if request.IsActive != nil {
		current.IsActive = *request.IsActive
	}
	current.UpdatedAt = timeNow()

	if err := h.store.UpsertPolicyCap(current); err != nil {
		return domain.PolicyCap{}, err
	}
	return current, nil
}

func (h *HubService) DeletePolicyCap(request DeletePolicyCapRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domain.InvalidArgument("id is required")
	}
	deleted, err := h.store.DeletePolicyCap(id)
	if err != nil {
		return err
	}
	if !deleted {
		return domain.NotFound("policy cap not found")
	}
	return nil
}

func (h *HubService) Summary() (domain.Summary, error) {
	benchmarks, err := h.store.ListBenchmarks()
	if err != nil {
		return domain.Summary{}, err
	}

	summary := domain.Summary{}
	tasks, err := h.store.ListTasks()
	if err != nil {
		return domain.Summary{}, err
	}
	notes, err := h.store.ListNotes()
	if err != nil {
		return domain.Summary{}, err
	}
	changelog, err := h.store.ListChangelog()
	if err != nil {
		return domain.Summary{}, err
	}

	summary.Counts.Tasks = len(tasks)
	summary.Counts.Notes = len(notes)
	summary.Counts.Changelog = len(changelog)
	summary.Counts.Benchmarks = len(benchmarks)
	runs, err := h.store.ListRuns()
	if err != nil {
		return domain.Summary{}, err
	}
	attempts, err := h.store.ListPromptAttempts("")
	if err != nil {
		return domain.Summary{}, err
	}
	events, err := h.store.ListRunEvents("")
	if err != nil {
		return domain.Summary{}, err
	}
	summary.Counts.Runs = len(runs)
	summary.Counts.Attempts = len(attempts)
	summary.Counts.RunEvents = len(events)
	summary.Totals.ByProvider = map[string]struct {
		Count   int     `json:"count"`
		CostUSD float64 `json:"cost_usd"`
	}{
		"api":          {Count: 0, CostUSD: 0},
		"subscription": {Count: 0, CostUSD: 0},
		"opensource":   {Count: 0, CostUSD: 0},
	}

	for _, benchmark := range benchmarks {
		summary.Totals.TokensIn += benchmark.TokensIn
		summary.Totals.TokensOut += benchmark.TokensOut
		summary.Totals.CostUSD += benchmark.CostUSD
		entry := summary.Totals.ByProvider[benchmark.ProviderType]
		entry.Count++
		entry.CostUSD += benchmark.CostUSD
		summary.Totals.ByProvider[benchmark.ProviderType] = entry
	}
	return summary, nil
}

func (h *HubService) CreateTask(request CreateTaskRequest) (domain.Task, error) {
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return domain.Task{}, domain.InvalidArgument("title is required")
	}

	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "todo"
	}
	if _, ok := validTaskStatuses[status]; !ok {
		return domain.Task{}, domain.InvalidArgument("status must be one of: todo, in_progress, done, blocked")
	}

	task := domain.Task{
		ID:        newID("task"),
		Title:     title,
		Details:   strings.TrimSpace(request.Details),
		Status:    status,
		Tags:      normalizeTags(request.Tags),
		CreatedAt: timeNow(),
		UpdatedAt: timeNow(),
	}

	if err := h.store.UpsertTask(task); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (h *HubService) UpdateTask(request UpdateTaskRequest) (domain.Task, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domain.Task{}, domain.InvalidArgument("id is required")
	}

	items, err := h.store.ListTasks()
	if err != nil {
		return domain.Task{}, err
	}

	for i := range items {
		if items[i].ID != id {
			continue
		}

		if title := strings.TrimSpace(request.Title); title != "" {
			items[i].Title = title
		}
		if request.Details != "" {
			items[i].Details = strings.TrimSpace(request.Details)
		}
		if request.Status != "" {
			status := strings.TrimSpace(request.Status)
			if _, ok := validTaskStatuses[status]; !ok {
				return domain.Task{}, domain.InvalidArgument("status must be one of: todo, in_progress, done, blocked")
			}
			items[i].Status = status
		}
		if request.Tags != nil {
			items[i].Tags = normalizeTags(request.Tags)
		}
		items[i].UpdatedAt = timeNow()
		if err := h.store.UpsertTask(items[i]); err != nil {
			return domain.Task{}, err
		}
		return items[i], nil
	}

	return domain.Task{}, domain.NotFound("task not found")
}

func (h *HubService) DeleteTask(request DeleteTaskRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domain.InvalidArgument("id is required")
	}

	deleted, err := h.store.DeleteTask(id)
	if err != nil {
		return err
	}
	if !deleted {
		return domain.NotFound("task not found")
	}
	return nil
}

func (h *HubService) ListTasks() ([]domain.Task, error) {
	items, err := h.store.ListTasks()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.Task) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	return items, nil
}

func (h *HubService) CreateNote(request CreateNoteRequest) (domain.Note, error) {
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return domain.Note{}, domain.InvalidArgument("title is required")
	}

	note := domain.Note{
		ID:        newID("note"),
		Title:     title,
		Body:      strings.TrimSpace(request.Body),
		Tags:      normalizeTags(request.Tags),
		CreatedAt: timeNow(),
	}

	if err := h.store.InsertNote(note); err != nil {
		return domain.Note{}, err
	}
	return note, nil
}

func (h *HubService) ListNotes() ([]domain.Note, error) {
	items, err := h.store.ListNotes()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.Note) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items, nil
}

func (h *HubService) AppendChangelog(request AppendChangelogRequest) (domain.ChangelogEntry, error) {
	summary := strings.TrimSpace(request.Summary)
	if summary == "" {
		return domain.ChangelogEntry{}, domain.InvalidArgument("summary is required")
	}

	category := strings.TrimSpace(request.Category)
	if category == "" {
		category = "ops"
	}
	if _, ok := validChangeCategories[category]; !ok {
		return domain.ChangelogEntry{}, domain.InvalidArgument("category must be one of: platform, policy, model, infra, ops")
	}

	entry := domain.ChangelogEntry{
		ID:        newID("chg"),
		Category:  category,
		Summary:   summary,
		Details:   strings.TrimSpace(request.Details),
		Actor:     strings.TrimSpace(request.Actor),
		CreatedAt: timeNow(),
	}

	if err := h.store.InsertChangelog(entry); err != nil {
		return domain.ChangelogEntry{}, err
	}
	return entry, nil
}

func (h *HubService) ListChangelog() ([]domain.ChangelogEntry, error) {
	items, err := h.store.ListChangelog()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.ChangelogEntry) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items, nil
}

func (h *HubService) RecordBenchmark(request RecordBenchmarkRequest) (domain.Benchmark, error) {
	workflow := strings.TrimSpace(request.Workflow)
	providerType := strings.TrimSpace(request.ProviderType)
	model := strings.TrimSpace(request.Model)

	if workflow == "" || providerType == "" || model == "" {
		return domain.Benchmark{}, domain.InvalidArgument("workflow, provider_type, and model are required")
	}
	if _, ok := validProviderTypes[providerType]; !ok {
		return domain.Benchmark{}, domain.InvalidArgument("provider_type must be one of: api, subscription, opensource")
	}
	if request.TokensIn < 0 || request.TokensOut < 0 || request.CostUSD < 0 || request.LatencyMS < 0 {
		return domain.Benchmark{}, domain.InvalidArgument("tokens, cost, and latency must be non-negative")
	}

	record := domain.Benchmark{
		ID:           newID("bm"),
		Workflow:     workflow,
		ProviderType: providerType,
		Provider:     strings.TrimSpace(request.Provider),
		Model:        model,
		TokensIn:     request.TokensIn,
		TokensOut:    request.TokensOut,
		CostUSD:      request.CostUSD,
		LatencyMS:    request.LatencyMS,
		QualityScore: request.QualityScore,
		Notes:        strings.TrimSpace(request.Notes),
		CreatedAt:    timeNow(),
	}

	if err := h.store.InsertBenchmark(record); err != nil {
		return domain.Benchmark{}, err
	}
	return record, nil
}

func (h *HubService) ListBenchmarks() ([]domain.Benchmark, error) {
	items, err := h.store.ListBenchmarks()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.Benchmark) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items, nil
}

func (h *HubService) StartRun(request StartRunRequest) (domain.AgentRun, error) {
	workflow := strings.TrimSpace(request.Workflow)
	agentID := strings.TrimSpace(request.AgentID)
	if workflow == "" || agentID == "" {
		return domain.AgentRun{}, domain.InvalidArgument("workflow and agent_id are required")
	}
	if request.MaxRetries < 0 {
		return domain.AgentRun{}, domain.InvalidArgument("max_retries must be non-negative")
	}
	policy, err := h.store.GetPolicy()
	if err != nil {
		return domain.AgentRun{}, err
	}
	if policy.KillSwitch {
		reason := strings.TrimSpace(policy.KillSwitchReason)
		if reason == "" {
			reason = "kill switch is enabled"
		}
		return domain.AgentRun{}, domain.FailedPrecondition(reason)
	}

	run := domain.AgentRun{
		ID:            newID("run"),
		TaskID:        strings.TrimSpace(request.TaskID),
		Workflow:      workflow,
		AgentID:       agentID,
		PromptVersion: strings.TrimSpace(request.PromptVersion),
		ModelPolicy:   strings.TrimSpace(request.ModelPolicy),
		Status:        "running",
		MaxRetries:    request.MaxRetries,
		StartedAt:     timeNow(),
	}
	if err := h.store.InsertRun(run); err != nil {
		return domain.AgentRun{}, err
	}
	return run, nil
}

func (h *HubService) FinishRun(request FinishRunRequest) (domain.AgentRun, error) {
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		return domain.AgentRun{}, domain.InvalidArgument("run_id is required")
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "completed"
	}
	if _, ok := validRunStatuses[status]; !ok || status == "running" {
		return domain.AgentRun{}, domain.InvalidArgument("status must be one of: completed, failed, cancelled")
	}

	runs, err := h.store.ListRuns()
	if err != nil {
		return domain.AgentRun{}, err
	}

	for _, run := range runs {
		if run.ID != runID {
			continue
		}

		now := timeNow()
		run.Status = status
		run.FinishedAt = now
		run.LastError = strings.TrimSpace(request.LastError)

		startedAt, err := time.Parse(time.RFC3339Nano, run.StartedAt)
		if err == nil {
			run.DurationMS = time.Since(startedAt).Milliseconds()
		}

		attempts, err := h.store.ListPromptAttempts(runID)
		if err != nil {
			return domain.AgentRun{}, err
		}
		for _, attempt := range attempts {
			run.TotalAttempts++
			run.TotalTokensIn += attempt.TokensIn
			run.TotalTokensOut += attempt.TokensOut
			run.TotalCostUSD += attempt.CostUSD
			if attempt.Outcome == "success" {
				run.SuccessAttempts++
			} else {
				run.FailedAttempts++
			}
		}

		if err := h.store.UpdateRun(run); err != nil {
			return domain.AgentRun{}, err
		}
		return run, nil
	}

	return domain.AgentRun{}, domain.NotFound("run not found")
}

func (h *HubService) RecordPromptAttempt(request RecordPromptAttemptRequest) (domain.PromptAttempt, error) {
	runID := strings.TrimSpace(request.RunID)
	outcome := strings.TrimSpace(request.Outcome)
	model := strings.TrimSpace(request.Model)
	providerType := strings.TrimSpace(request.ProviderType)
	if providerType == "" {
		providerType = "api"
	}
	provider := strings.TrimSpace(request.Provider)
	if runID == "" || outcome == "" || model == "" {
		return domain.PromptAttempt{}, domain.InvalidArgument("run_id, outcome, and model are required")
	}
	if request.AttemptNumber <= 0 {
		return domain.PromptAttempt{}, domain.InvalidArgument("attempt_number must be greater than 0")
	}
	if _, ok := validAttemptOutcomes[outcome]; !ok {
		return domain.PromptAttempt{}, domain.InvalidArgument("outcome must be one of: success, failed, timeout, retryable_error, tool_error")
	}
	if request.TokensIn < 0 || request.TokensOut < 0 || request.CostUSD < 0 || request.LatencyMS < 0 {
		return domain.PromptAttempt{}, domain.InvalidArgument("tokens, cost, and latency must be non-negative")
	}
	policy, err := h.store.GetPolicy()
	if err != nil {
		return domain.PromptAttempt{}, err
	}
	if policy.KillSwitch {
		reason := strings.TrimSpace(policy.KillSwitchReason)
		if reason == "" {
			reason = "kill switch is enabled"
		}
		return domain.PromptAttempt{}, domain.FailedPrecondition(reason)
	}
	caps, err := h.store.ListPolicyCaps()
	if err != nil {
		return domain.PromptAttempt{}, err
	}
	selectedCap, hasCap := selectPolicyCap(caps, providerType, provider, model)
	limits := resolveEffectiveLimits(policy, selectedCap, hasCap)

	runs, err := h.store.ListRunsFiltered(domain.RunFilter{RunID: runID, Limit: 1})
	if err != nil {
		return domain.PromptAttempt{}, err
	}
	if len(runs) == 0 {
		return domain.PromptAttempt{}, domain.NotFound("run not found")
	}
	if runs[0].Status != "running" {
		return domain.PromptAttempt{}, domain.FailedPrecondition("run is not in running state")
	}

	capOverridesAttemptLatency := hasCap && selectedCap.MaxLatencyPerAttemptMS > 0
	capOverridesRunCost := hasCap && selectedCap.MaxCostPerRunUSD > 0
	capOverridesRunAttempts := hasCap && selectedCap.MaxAttemptsPerRun > 0
	capOverridesRunTokens := hasCap && selectedCap.MaxTokensPerRun > 0
	attemptTokens := request.TokensIn + request.TokensOut
	if limits.MaxLatencyPerAttemptMS > 0 && request.LatencyMS > limits.MaxLatencyPerAttemptMS {
		if capOverridesAttemptLatency && selectedCap.DryRun {
			h.logPolicyCapDryRunViolation(runID, selectedCap, "attempt latency exceeds cap limit")
		} else {
			return domain.PromptAttempt{}, domain.ResourceExhausted("attempt latency exceeds policy cap (" + limits.Source + ")")
		}
	}
	if limits.MaxCostPerAttemptUSD > 0 && request.CostUSD > limits.MaxCostPerAttemptUSD {
		if hasCap && selectedCap.DryRun {
			h.logPolicyCapDryRunViolation(runID, selectedCap, "attempt cost exceeds cap limit")
		} else {
			return domain.PromptAttempt{}, domain.ResourceExhausted("attempt cost exceeds policy cap (" + limits.Source + ")")
		}
	}
	if limits.MaxTokensPerAttempt > 0 && attemptTokens > limits.MaxTokensPerAttempt {
		if hasCap && selectedCap.DryRun {
			h.logPolicyCapDryRunViolation(runID, selectedCap, "attempt tokens exceed cap limit")
		} else {
			return domain.PromptAttempt{}, domain.ResourceExhausted("attempt tokens exceed policy cap (" + limits.Source + ")")
		}
	}
	existingAttempts, err := h.store.ListPromptAttemptsFiltered(domain.AttemptFilter{RunID: runID})
	if err != nil {
		return domain.PromptAttempt{}, err
	}
	if limits.MaxAttemptsPerRun > 0 && int64(len(existingAttempts))+1 > limits.MaxAttemptsPerRun {
		if capOverridesRunAttempts && selectedCap.DryRun {
			h.logPolicyCapDryRunViolation(runID, selectedCap, "run exceeds max attempts cap")
		} else {
			return domain.PromptAttempt{}, domain.ResourceExhausted("run exceeds max attempts cap (" + limits.Source + ")")
		}
	}
	if limits.MaxCostPerRunUSD > 0 || limits.MaxTokensPerRun > 0 {
		var totalCost float64
		var totalTokens int64
		for _, item := range existingAttempts {
			totalCost += item.CostUSD
			totalTokens += item.TokensIn + item.TokensOut
		}
		totalCost += request.CostUSD
		totalTokens += request.TokensIn + request.TokensOut

		if limits.MaxCostPerRunUSD > 0 && totalCost > limits.MaxCostPerRunUSD {
			if capOverridesRunCost && selectedCap.DryRun {
				h.logPolicyCapDryRunViolation(runID, selectedCap, "run exceeds max cost cap")
			} else {
				return domain.PromptAttempt{}, domain.ResourceExhausted("run exceeds max cost cap (" + limits.Source + ")")
			}
		}
		if limits.MaxTokensPerRun > 0 && totalTokens > limits.MaxTokensPerRun {
			if capOverridesRunTokens && selectedCap.DryRun {
				h.logPolicyCapDryRunViolation(runID, selectedCap, "run exceeds max tokens cap")
			} else {
				return domain.PromptAttempt{}, domain.ResourceExhausted("run exceeds max tokens cap (" + limits.Source + ")")
			}
		}
	}

	attempt := domain.PromptAttempt{
		ID:            newID("pat"),
		RunID:         runID,
		AttemptNumber: request.AttemptNumber,
		Workflow:      strings.TrimSpace(request.Workflow),
		AgentID:       strings.TrimSpace(request.AgentID),
		ProviderType:  providerType,
		Provider:      provider,
		Model:         model,
		PromptVersion: strings.TrimSpace(request.PromptVersion),
		PromptHash:    strings.TrimSpace(request.PromptHash),
		Outcome:       outcome,
		ErrorType:     strings.TrimSpace(request.ErrorType),
		ErrorMessage:  strings.TrimSpace(request.ErrorMessage),
		TokensIn:      request.TokensIn,
		TokensOut:     request.TokensOut,
		CostUSD:       request.CostUSD,
		LatencyMS:     request.LatencyMS,
		QualityScore:  request.QualityScore,
		CreatedAt:     timeNow(),
	}

	if err := h.store.InsertPromptAttempt(attempt); err != nil {
		return domain.PromptAttempt{}, err
	}
	return attempt, nil
}

func (h *HubService) RecordRunEvent(request RecordRunEventRequest) (domain.RunEvent, error) {
	runID := strings.TrimSpace(request.RunID)
	eventType := strings.TrimSpace(request.EventType)
	if runID == "" || eventType == "" {
		return domain.RunEvent{}, domain.InvalidArgument("run_id and event_type are required")
	}

	level := strings.TrimSpace(request.Level)
	if level == "" {
		level = "info"
	}
	if _, ok := validEventLevels[level]; !ok {
		return domain.RunEvent{}, domain.InvalidArgument("level must be one of: info, warn, error")
	}

	runs, err := h.store.ListRuns()
	if err != nil {
		return domain.RunEvent{}, err
	}
	runExists := false
	for _, run := range runs {
		if run.ID == runID {
			runExists = true
			break
		}
	}
	if !runExists {
		return domain.RunEvent{}, domain.NotFound("run not found")
	}

	event := domain.RunEvent{
		ID:        newID("evt"),
		RunID:     runID,
		EventType: eventType,
		Level:     level,
		Message:   strings.TrimSpace(request.Message),
		DataJSON:  strings.TrimSpace(request.DataJSON),
		CreatedAt: timeNow(),
	}
	if err := h.store.InsertRunEvent(event); err != nil {
		return domain.RunEvent{}, err
	}
	return event, nil
}

func (h *HubService) ListRuns(request ListRunsRequest) ([]domain.AgentRun, error) {
	if request.Limit < 0 {
		return nil, domain.InvalidArgument("limit must be non-negative")
	}
	if request.StartedAfter != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.StartedAfter)); err != nil {
			return nil, domain.InvalidArgument("started_after must be RFC3339 timestamp")
		}
	}
	if request.StartedBefore != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.StartedBefore)); err != nil {
			return nil, domain.InvalidArgument("started_before must be RFC3339 timestamp")
		}
	}
	filter := domain.RunFilter{
		RunID:         strings.TrimSpace(request.RunID),
		TaskID:        strings.TrimSpace(request.TaskID),
		Workflow:      strings.TrimSpace(request.Workflow),
		AgentID:       strings.TrimSpace(request.AgentID),
		Status:        strings.TrimSpace(request.Status),
		PromptVersion: strings.TrimSpace(request.PromptVersion),
		StartedAfter:  strings.TrimSpace(request.StartedAfter),
		StartedBefore: strings.TrimSpace(request.StartedBefore),
		Limit:         request.Limit,
	}
	items, err := h.store.ListRunsFiltered(filter)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.AgentRun) int {
		if a.StartedAt == b.StartedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.StartedAt, a.StartedAt)
	})
	return items, nil
}

func (h *HubService) ListPromptAttempts(request ListPromptAttemptsRequest) ([]domain.PromptAttempt, error) {
	if request.Limit < 0 {
		return nil, domain.InvalidArgument("limit must be non-negative")
	}
	if request.CreatedAfter != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.CreatedAfter)); err != nil {
			return nil, domain.InvalidArgument("created_after must be RFC3339 timestamp")
		}
	}
	if request.CreatedBefore != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.CreatedBefore)); err != nil {
			return nil, domain.InvalidArgument("created_before must be RFC3339 timestamp")
		}
	}
	filter := domain.AttemptFilter{
		RunID:         strings.TrimSpace(request.RunID),
		Workflow:      strings.TrimSpace(request.Workflow),
		AgentID:       strings.TrimSpace(request.AgentID),
		Model:         strings.TrimSpace(request.Model),
		Outcome:       strings.TrimSpace(request.Outcome),
		PromptVersion: strings.TrimSpace(request.PromptVersion),
		CreatedAfter:  strings.TrimSpace(request.CreatedAfter),
		CreatedBefore: strings.TrimSpace(request.CreatedBefore),
		Limit:         request.Limit,
	}
	items, err := h.store.ListPromptAttemptsFiltered(filter)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.PromptAttempt) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items, nil
}

func (h *HubService) ListRunEvents(request ListRunEventsRequest) ([]domain.RunEvent, error) {
	if request.Limit < 0 {
		return nil, domain.InvalidArgument("limit must be non-negative")
	}
	if request.CreatedAfter != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.CreatedAfter)); err != nil {
			return nil, domain.InvalidArgument("created_after must be RFC3339 timestamp")
		}
	}
	if request.CreatedBefore != "" {
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.CreatedBefore)); err != nil {
			return nil, domain.InvalidArgument("created_before must be RFC3339 timestamp")
		}
	}
	filter := domain.EventFilter{
		RunID:         strings.TrimSpace(request.RunID),
		EventType:     strings.TrimSpace(request.EventType),
		Level:         strings.TrimSpace(request.Level),
		CreatedAfter:  strings.TrimSpace(request.CreatedAfter),
		CreatedBefore: strings.TrimSpace(request.CreatedBefore),
		Limit:         request.Limit,
	}
	items, err := h.store.ListRunEventsFiltered(filter)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b domain.RunEvent) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items, nil
}

func (h *HubService) TelemetrySummary() (domain.TelemetrySummary, error) {
	summary := domain.TelemetrySummary{}

	runs, err := h.store.ListRuns()
	if err != nil {
		return summary, err
	}
	attempts, err := h.store.ListPromptAttempts("")
	if err != nil {
		return summary, err
	}
	events, err := h.store.ListRunEvents("")
	if err != nil {
		return summary, err
	}

	summary.Counts.Runs = int64(len(runs))
	summary.Counts.Events = int64(len(events))

	for _, run := range runs {
		switch run.Status {
		case "running":
			summary.Counts.RunningRuns++
		case "completed":
			summary.Counts.CompletedRuns++
		case "failed":
			summary.Counts.FailedRuns++
		case "cancelled":
			summary.Counts.CancelledRuns++
		}
	}

	for _, attempt := range attempts {
		summary.Counts.Attempts++
		summary.Totals.TokensIn += attempt.TokensIn
		summary.Totals.TokensOut += attempt.TokensOut
		summary.Totals.CostUSD += attempt.CostUSD
		summary.Totals.LatencyMS += attempt.LatencyMS

		if attempt.Outcome == "success" {
			summary.Counts.SuccessAttempts++
		} else {
			summary.Counts.FailedAttempts++
		}
		if attempt.AttemptNumber > 1 {
			summary.Counts.Retries++
		}
	}

	if summary.Counts.Attempts > 0 {
		summary.Averages.AttemptLatencyMS = float64(summary.Totals.LatencyMS) / float64(summary.Counts.Attempts)
		summary.Averages.CostPerAttempt = summary.Totals.CostUSD / float64(summary.Counts.Attempts)
		summary.Averages.SuccessRate = float64(summary.Counts.SuccessAttempts) / float64(summary.Counts.Attempts)
	}

	return summary, nil
}

func (h *HubService) Leaderboard(request LeaderboardRequest) ([]domain.LeaderboardEntry, error) {
	if request.Limit < 0 {
		return nil, domain.InvalidArgument("limit must be non-negative")
	}
	if request.WindowDays < 0 {
		return nil, domain.InvalidArgument("window_days must be non-negative")
	}

	filter := domain.AttemptFilter{
		Workflow:      strings.TrimSpace(request.Workflow),
		Model:         strings.TrimSpace(request.Model),
		PromptVersion: strings.TrimSpace(request.PromptVersion),
	}
	if request.WindowDays > 0 {
		filter.CreatedAfter = time.Now().UTC().Add(-time.Duration(request.WindowDays) * 24 * time.Hour).Format(time.RFC3339Nano)
	}
	attempts, err := h.store.ListPromptAttemptsFiltered(filter)
	if err != nil {
		return nil, err
	}

	type aggregate struct {
		workflow      string
		promptVersion string
		model         string
		attempts      int64
		successes     int64
		failures      int64
		totalCost     float64
		totalLatency  int64
	}
	grouped := map[string]*aggregate{}
	for _, item := range attempts {
		key := strings.Join([]string{item.Workflow, item.PromptVersion, item.Model}, "|")
		entry, ok := grouped[key]
		if !ok {
			entry = &aggregate{
				workflow:      item.Workflow,
				promptVersion: item.PromptVersion,
				model:         item.Model,
			}
			grouped[key] = entry
		}
		entry.attempts++
		entry.totalCost += item.CostUSD
		entry.totalLatency += item.LatencyMS
		if item.Outcome == "success" {
			entry.successes++
		} else {
			entry.failures++
		}
	}

	out := make([]domain.LeaderboardEntry, 0, len(grouped))
	for _, item := range grouped {
		if item.attempts == 0 {
			continue
		}
		successRate := float64(item.successes) / float64(item.attempts)
		avgCost := item.totalCost / float64(item.attempts)
		avgLatency := float64(item.totalLatency) / float64(item.attempts)
		score := (successRate * 100.0) - (avgCost * 100.0) - (avgLatency / 1000.0)

		out = append(out, domain.LeaderboardEntry{
			Workflow:         item.workflow,
			PromptVersion:    item.promptVersion,
			Model:            item.model,
			Attempts:         item.attempts,
			SuccessAttempts:  item.successes,
			FailedAttempts:   item.failures,
			SuccessRate:      successRate,
			AverageCostUSD:   avgCost,
			AverageLatencyMS: avgLatency,
			Score:            score,
		})
	}

	slices.SortFunc(out, func(a, b domain.LeaderboardEntry) int {
		if a.Score == b.Score {
			if a.SuccessRate == b.SuccessRate {
				return strings.Compare(a.PromptVersion, b.PromptVersion)
			}
			if a.SuccessRate > b.SuccessRate {
				return -1
			}
			return 1
		}
		if a.Score > b.Score {
			return -1
		}
		return 1
	})

	if request.Limit > 0 && int64(len(out)) > request.Limit {
		out = out[:request.Limit]
	}

	return out, nil
}

func resolveEffectiveLimits(policy domain.OrchestrationPolicy, cap domain.PolicyCap, hasCap bool) effectiveLimits {
	out := effectiveLimits{
		MaxCostPerRunUSD:       policy.MaxCostPerRunUSD,
		MaxAttemptsPerRun:      policy.MaxAttemptsPerRun,
		MaxTokensPerRun:        policy.MaxTokensPerRun,
		MaxLatencyPerAttemptMS: policy.MaxLatencyPerAttemptMS,
		MaxCostPerAttemptUSD:   0,
		MaxTokensPerAttempt:    0,
		Source:                 "global-policy",
	}
	if !hasCap {
		return out
	}
	out.Source = "policy-cap:" + cap.ID
	if cap.MaxCostPerRunUSD > 0 {
		out.MaxCostPerRunUSD = cap.MaxCostPerRunUSD
	}
	if cap.MaxAttemptsPerRun > 0 {
		out.MaxAttemptsPerRun = cap.MaxAttemptsPerRun
	}
	if cap.MaxTokensPerRun > 0 {
		out.MaxTokensPerRun = cap.MaxTokensPerRun
	}
	if cap.MaxLatencyPerAttemptMS > 0 {
		out.MaxLatencyPerAttemptMS = cap.MaxLatencyPerAttemptMS
	}
	if cap.MaxCostPerAttemptUSD > 0 {
		out.MaxCostPerAttemptUSD = cap.MaxCostPerAttemptUSD
	}
	if cap.MaxTokensPerAttempt > 0 {
		out.MaxTokensPerAttempt = cap.MaxTokensPerAttempt
	}
	return out
}

func selectPolicyCap(caps []domain.PolicyCap, providerType, provider, model string) (domain.PolicyCap, bool) {
	var selected domain.PolicyCap
	found := false
	bestSpecificity := int64(-1)
	bestPriority := int64(-1 << 62)
	for _, cap := range caps {
		if !cap.IsActive {
			continue
		}
		if cap.ProviderType != "" && cap.ProviderType != providerType {
			continue
		}
		if cap.Provider != "" && cap.Provider != provider {
			continue
		}
		if cap.Model != "" && cap.Model != model {
			continue
		}

		specificity := int64(0)
		if cap.ProviderType != "" {
			specificity++
		}
		if cap.Provider != "" {
			specificity++
		}
		if cap.Model != "" {
			specificity++
		}

		if !found || specificity > bestSpecificity || (specificity == bestSpecificity && cap.Priority > bestPriority) {
			selected = cap
			found = true
			bestSpecificity = specificity
			bestPriority = cap.Priority
		}
	}
	return selected, found
}

func (h *HubService) logPolicyCapDryRunViolation(runID string, cap domain.PolicyCap, message string) {
	payload := map[string]any{
		"cap_id":        cap.ID,
		"cap_name":      cap.Name,
		"provider_type": cap.ProviderType,
		"provider":      cap.Provider,
		"model":         cap.Model,
		"priority":      cap.Priority,
		"dry_run":       cap.DryRun,
	}
	serialized, _ := json.Marshal(payload)
	_ = h.store.InsertRunEvent(domain.RunEvent{
		ID:        newID("evt"),
		RunID:     runID,
		EventType: "policy_cap_violation_dry_run",
		Level:     "warn",
		Message:   message,
		DataJSON:  string(serialized),
		CreatedAt: timeNow(),
	})
}

func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}

	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		clean := strings.ToLower(strings.TrimSpace(tag))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func timeNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func newID(prefix string) string {
	var raw [8]byte
	_, _ = rand.Read(raw[:])
	return prefix + "_" + time.Now().UTC().Format("20060102T150405.000000000") + "_" + hex.EncodeToString(raw[:])
}
