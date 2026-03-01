package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bcrosbie/modeloman/internal/domain"
)

type FileStore struct {
	path        string
	mu          sync.RWMutex
	state       domain.State
	idempotency map[string]IdempotencyRecord
}

func NewFileStore(path string) *FileStore {
	return &FileStore{
		path:        path,
		state:       domain.EmptyState(),
		idempotency: map[string]IdempotencyRecord{},
	}
}

func (s *FileStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return domain.Internal("failed to create data directory", err)
	}

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.state = domain.EmptyState()
			return s.persistLocked()
		}
		return domain.Internal("failed to read data file", err)
	}

	var parsed domain.State
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.Internal("failed to parse data file", err)
	}

	s.state = withDefaults(parsed)
	if s.idempotency == nil {
		s.idempotency = map[string]IdempotencyRecord{}
	}
	return nil
}

func (s *FileStore) Close() error {
	return nil
}

func (s *FileStore) Snapshot() domain.State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneState(s.state)
}

func (s *FileStore) Mutate(mutate func(*domain.State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneState(s.state)
	if err := mutate(&next); err != nil {
		return err
	}

	s.state = withDefaults(next)
	return s.persistLocked()
}

func (s *FileStore) persistLocked() error {
	serialized, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return domain.Internal("failed to serialize state", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, append(serialized, '\n'), 0o600); err != nil {
		return domain.Internal("failed to write temporary state file", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return domain.Internal("failed to atomically persist state file", err)
	}
	return nil
}

func withDefaults(state domain.State) domain.State {
	if state.Tasks == nil {
		state.Tasks = []domain.Task{}
	}
	if state.Notes == nil {
		state.Notes = []domain.Note{}
	}
	if state.Changelog == nil {
		state.Changelog = []domain.ChangelogEntry{}
	}
	if state.Benchmarks == nil {
		state.Benchmarks = []domain.Benchmark{}
	}
	if state.Runs == nil {
		state.Runs = []domain.AgentRun{}
	}
	if state.Attempts == nil {
		state.Attempts = []domain.PromptAttempt{}
	}
	if state.RunEvents == nil {
		state.RunEvents = []domain.RunEvent{}
	}
	if state.Policy.UpdatedAt == "" && !state.Policy.KillSwitch {
		state.Policy = domain.DefaultPolicy()
	}
	if state.PolicyCaps == nil {
		state.PolicyCaps = []domain.PolicyCap{}
	}
	return state
}

func cloneState(in domain.State) domain.State {
	raw, _ := json.Marshal(in)
	var out domain.State
	_ = json.Unmarshal(raw, &out)
	return withDefaults(out)
}

func (s *FileStore) ExportState() (domain.State, error) {
	return s.Snapshot(), nil
}

func (s *FileStore) GetPolicy() (domain.OrchestrationPolicy, error) {
	return s.Snapshot().Policy, nil
}

func (s *FileStore) SetPolicy(policy domain.OrchestrationPolicy) error {
	return s.Mutate(func(state *domain.State) error {
		state.Policy = policy
		return nil
	})
}

func (s *FileStore) ListPolicyCaps() ([]domain.PolicyCap, error) {
	return s.Snapshot().PolicyCaps, nil
}

func (s *FileStore) UpsertPolicyCap(cap domain.PolicyCap) error {
	return s.Mutate(func(state *domain.State) error {
		for i := range state.PolicyCaps {
			if state.PolicyCaps[i].ID != cap.ID {
				continue
			}
			state.PolicyCaps[i] = cap
			return nil
		}
		state.PolicyCaps = append(state.PolicyCaps, cap)
		return nil
	})
}

func (s *FileStore) DeletePolicyCap(id string) (bool, error) {
	deleted := false
	err := s.Mutate(func(state *domain.State) error {
		for index, item := range state.PolicyCaps {
			if item.ID != id {
				continue
			}
			state.PolicyCaps = slices.Delete(state.PolicyCaps, index, index+1)
			deleted = true
			return nil
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
}

func (s *FileStore) ListTasks() ([]domain.Task, error) {
	return s.Snapshot().Tasks, nil
}

func (s *FileStore) UpsertTask(task domain.Task) error {
	return s.Mutate(func(state *domain.State) error {
		for i := range state.Tasks {
			if state.Tasks[i].ID == task.ID {
				state.Tasks[i] = task
				return nil
			}
		}
		state.Tasks = append(state.Tasks, task)
		return nil
	})
}

func (s *FileStore) DeleteTask(id string) (bool, error) {
	deleted := false
	err := s.Mutate(func(state *domain.State) error {
		for index, task := range state.Tasks {
			if task.ID != id {
				continue
			}
			state.Tasks = slices.Delete(state.Tasks, index, index+1)
			deleted = true
			return nil
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
}

func (s *FileStore) ListNotes() ([]domain.Note, error) {
	return s.Snapshot().Notes, nil
}

func (s *FileStore) InsertNote(note domain.Note) error {
	return s.Mutate(func(state *domain.State) error {
		state.Notes = append(state.Notes, note)
		return nil
	})
}

func (s *FileStore) ListChangelog() ([]domain.ChangelogEntry, error) {
	return s.Snapshot().Changelog, nil
}

func (s *FileStore) InsertChangelog(entry domain.ChangelogEntry) error {
	return s.Mutate(func(state *domain.State) error {
		state.Changelog = append(state.Changelog, entry)
		return nil
	})
}

func (s *FileStore) ListBenchmarks() ([]domain.Benchmark, error) {
	return s.Snapshot().Benchmarks, nil
}

func (s *FileStore) InsertBenchmark(benchmark domain.Benchmark) error {
	return s.Mutate(func(state *domain.State) error {
		state.Benchmarks = append(state.Benchmarks, benchmark)
		return nil
	})
}

func (s *FileStore) ListRuns() ([]domain.AgentRun, error) {
	return s.Snapshot().Runs, nil
}

func (s *FileStore) ListRunsFiltered(filter domain.RunFilter) ([]domain.AgentRun, error) {
	items := s.Snapshot().Runs
	out := make([]domain.AgentRun, 0, len(items))
	for _, item := range items {
		if filter.RunID != "" && item.ID != filter.RunID {
			continue
		}
		if filter.TaskID != "" && item.TaskID != filter.TaskID {
			continue
		}
		if filter.Workflow != "" && item.Workflow != filter.Workflow {
			continue
		}
		if filter.AgentID != "" && item.AgentID != filter.AgentID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.PromptVersion != "" && item.PromptVersion != filter.PromptVersion {
			continue
		}
		if filter.StartedAfter != "" && item.StartedAt <= filter.StartedAfter {
			continue
		}
		if filter.StartedBefore != "" && item.StartedAt >= filter.StartedBefore {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && int64(len(out)) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *FileStore) InsertRun(run domain.AgentRun) error {
	return s.Mutate(func(state *domain.State) error {
		state.Runs = append(state.Runs, run)
		return nil
	})
}

func (s *FileStore) UpdateRun(run domain.AgentRun) error {
	return s.Mutate(func(state *domain.State) error {
		for i := range state.Runs {
			if state.Runs[i].ID != run.ID {
				continue
			}
			state.Runs[i] = run
			return nil
		}
		state.Runs = append(state.Runs, run)
		return nil
	})
}

func (s *FileStore) ListPromptAttempts(runID string) ([]domain.PromptAttempt, error) {
	return s.ListPromptAttemptsFiltered(domain.AttemptFilter{RunID: runID})
}

func (s *FileStore) ListPromptAttemptsFiltered(filter domain.AttemptFilter) ([]domain.PromptAttempt, error) {
	items := s.Snapshot().Attempts
	out := make([]domain.PromptAttempt, 0, len(items))
	for _, item := range items {
		if filter.RunID != "" && item.RunID != filter.RunID {
			continue
		}
		if filter.Workflow != "" && item.Workflow != filter.Workflow {
			continue
		}
		if filter.AgentID != "" && item.AgentID != filter.AgentID {
			continue
		}
		if filter.Model != "" && item.Model != filter.Model {
			continue
		}
		if filter.Outcome != "" && item.Outcome != filter.Outcome {
			continue
		}
		if filter.PromptVersion != "" && item.PromptVersion != filter.PromptVersion {
			continue
		}
		if filter.CreatedAfter != "" && item.CreatedAt <= filter.CreatedAfter {
			continue
		}
		if filter.CreatedBefore != "" && item.CreatedAt >= filter.CreatedBefore {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && int64(len(out)) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *FileStore) InsertPromptAttempt(attempt domain.PromptAttempt) error {
	return s.Mutate(func(state *domain.State) error {
		state.Attempts = append(state.Attempts, attempt)
		return nil
	})
}

func (s *FileStore) ListRunEvents(runID string) ([]domain.RunEvent, error) {
	return s.ListRunEventsFiltered(domain.EventFilter{RunID: runID})
}

func (s *FileStore) ListRunEventsFiltered(filter domain.EventFilter) ([]domain.RunEvent, error) {
	items := s.Snapshot().RunEvents
	out := make([]domain.RunEvent, 0, len(items))
	for _, item := range items {
		if filter.RunID != "" && item.RunID != filter.RunID {
			continue
		}
		if filter.EventType != "" && item.EventType != filter.EventType {
			continue
		}
		if filter.Level != "" && item.Level != filter.Level {
			continue
		}
		if filter.CreatedAfter != "" && item.CreatedAt <= filter.CreatedAfter {
			continue
		}
		if filter.CreatedBefore != "" && item.CreatedAt >= filter.CreatedBefore {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && int64(len(out)) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *FileStore) InsertRunEvent(event domain.RunEvent) error {
	return s.Mutate(func(state *domain.State) error {
		state.RunEvents = append(state.RunEvents, event)
		return nil
	})
}

func (s *FileStore) ReserveIdempotencyKey(method, idempotencyKey, requestHash string) (IdempotencyRecord, bool, error) {
	method = normalizeIdempotencyToken(method)
	idempotencyKey = normalizeIdempotencyToken(idempotencyKey)
	if method == "" || idempotencyKey == "" || requestHash == "" {
		return IdempotencyRecord{}, false, domain.InvalidArgument("method, idempotency_key, and request_hash are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fileIdempotencyRecordKey(method, idempotencyKey)
	record, exists := s.idempotency[key]
	if exists {
		return record, false, nil
	}

	s.idempotency[key] = IdempotencyRecord{
		RequestHash: requestHash,
		Completed:   false,
	}
	return IdempotencyRecord{}, true, nil
}

func (s *FileStore) CompleteIdempotencyKey(method, idempotencyKey, responseJSON string) error {
	method = normalizeIdempotencyToken(method)
	idempotencyKey = normalizeIdempotencyToken(idempotencyKey)
	if method == "" || idempotencyKey == "" {
		return domain.InvalidArgument("method and idempotency_key are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fileIdempotencyRecordKey(method, idempotencyKey)
	record, exists := s.idempotency[key]
	if !exists {
		return domain.NotFound("idempotency key not found")
	}
	record.ResponseJSON = responseJSON
	record.Completed = true
	s.idempotency[key] = record
	return nil
}

func (s *FileStore) ReleaseIdempotencyKey(method, idempotencyKey string) error {
	method = normalizeIdempotencyToken(method)
	idempotencyKey = normalizeIdempotencyToken(idempotencyKey)
	if method == "" || idempotencyKey == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fileIdempotencyRecordKey(method, idempotencyKey)
	record, exists := s.idempotency[key]
	if !exists {
		return nil
	}
	if record.Completed {
		return nil
	}
	delete(s.idempotency, key)
	return nil
}

func fileIdempotencyRecordKey(method, idempotencyKey string) string {
	return method + "::" + idempotencyKey
}

func normalizeIdempotencyToken(value string) string {
	return strings.TrimSpace(value)
}
