package service

import (
	"crypto/rand"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"github.com/bcrosbie/modeloman/internal/domain"
	"github.com/bcrosbie/modeloman/internal/store"
)

var (
	validTaskStatuses     = map[string]struct{}{"todo": {}, "in_progress": {}, "done": {}, "blocked": {}}
	validProviderTypes    = map[string]struct{}{"api": {}, "subscription": {}, "opensource": {}}
	validChangeCategories = map[string]struct{}{
		"platform": {},
		"policy":   {},
		"model":    {},
		"infra":    {},
		"ops":      {},
	}
)

type HubService struct {
	store    *store.FileStore
	dataFile string
}

func NewHubService(store *store.FileStore, dataFile string) *HubService {
	return &HubService{
		store:    store,
		dataFile: dataFile,
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

func (h *HubService) Health() map[string]any {
	return map[string]any{
		"status":    "ok",
		"data_file": h.dataFile,
		"time_utc":  time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (h *HubService) ExportState() domain.State {
	return h.store.Snapshot()
}

func (h *HubService) Summary() domain.Summary {
	state := h.store.Snapshot()
	summary := domain.Summary{}
	summary.Counts.Tasks = len(state.Tasks)
	summary.Counts.Notes = len(state.Notes)
	summary.Counts.Changelog = len(state.Changelog)
	summary.Counts.Benchmarks = len(state.Benchmarks)
	summary.Totals.ByProvider = map[string]struct {
		Count   int     `json:"count"`
		CostUSD float64 `json:"cost_usd"`
	}{
		"api":          {Count: 0, CostUSD: 0},
		"subscription": {Count: 0, CostUSD: 0},
		"opensource":   {Count: 0, CostUSD: 0},
	}

	for _, benchmark := range state.Benchmarks {
		summary.Totals.TokensIn += benchmark.TokensIn
		summary.Totals.TokensOut += benchmark.TokensOut
		summary.Totals.CostUSD += benchmark.CostUSD
		entry := summary.Totals.ByProvider[benchmark.ProviderType]
		entry.Count++
		entry.CostUSD += benchmark.CostUSD
		summary.Totals.ByProvider[benchmark.ProviderType] = entry
	}
	return summary
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

	if err := h.store.Mutate(func(state *domain.State) error {
		state.Tasks = append(state.Tasks, task)
		return nil
	}); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (h *HubService) UpdateTask(request UpdateTaskRequest) (domain.Task, error) {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domain.Task{}, domain.InvalidArgument("id is required")
	}

	var updated domain.Task
	err := h.store.Mutate(func(state *domain.State) error {
		for index := range state.Tasks {
			if state.Tasks[index].ID != id {
				continue
			}

			if title := strings.TrimSpace(request.Title); title != "" {
				state.Tasks[index].Title = title
			}
			if request.Details != "" {
				state.Tasks[index].Details = strings.TrimSpace(request.Details)
			}
			if request.Status != "" {
				status := strings.TrimSpace(request.Status)
				if _, ok := validTaskStatuses[status]; !ok {
					return domain.InvalidArgument("status must be one of: todo, in_progress, done, blocked")
				}
				state.Tasks[index].Status = status
			}
			if request.Tags != nil {
				state.Tasks[index].Tags = normalizeTags(request.Tags)
			}
			state.Tasks[index].UpdatedAt = timeNow()
			updated = state.Tasks[index]
			return nil
		}
		return domain.NotFound("task not found")
	})
	if err != nil {
		return domain.Task{}, err
	}
	return updated, nil
}

func (h *HubService) DeleteTask(request DeleteTaskRequest) error {
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return domain.InvalidArgument("id is required")
	}

	return h.store.Mutate(func(state *domain.State) error {
		for index, task := range state.Tasks {
			if task.ID != id {
				continue
			}
			state.Tasks = slices.Delete(state.Tasks, index, index+1)
			return nil
		}
		return domain.NotFound("task not found")
	})
}

func (h *HubService) ListTasks() []domain.Task {
	items := h.store.Snapshot().Tasks
	slices.SortFunc(items, func(a, b domain.Task) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.UpdatedAt, a.UpdatedAt)
	})
	return items
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

	if err := h.store.Mutate(func(state *domain.State) error {
		state.Notes = append(state.Notes, note)
		return nil
	}); err != nil {
		return domain.Note{}, err
	}
	return note, nil
}

func (h *HubService) ListNotes() []domain.Note {
	items := h.store.Snapshot().Notes
	slices.SortFunc(items, func(a, b domain.Note) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items
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

	if err := h.store.Mutate(func(state *domain.State) error {
		state.Changelog = append(state.Changelog, entry)
		return nil
	}); err != nil {
		return domain.ChangelogEntry{}, err
	}
	return entry, nil
}

func (h *HubService) ListChangelog() []domain.ChangelogEntry {
	items := h.store.Snapshot().Changelog
	slices.SortFunc(items, func(a, b domain.ChangelogEntry) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items
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

	if err := h.store.Mutate(func(state *domain.State) error {
		state.Benchmarks = append(state.Benchmarks, record)
		return nil
	}); err != nil {
		return domain.Benchmark{}, err
	}
	return record, nil
}

func (h *HubService) ListBenchmarks() []domain.Benchmark {
	items := h.store.Snapshot().Benchmarks
	slices.SortFunc(items, func(a, b domain.Benchmark) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		return strings.Compare(b.CreatedAt, a.CreatedAt)
	})
	return items
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
