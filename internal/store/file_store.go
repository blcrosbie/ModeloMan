package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/bcrosbie/modeloman/internal/domain"
)

type FileStore struct {
	path  string
	mu    sync.RWMutex
	state domain.State
}

func NewFileStore(path string) *FileStore {
	return &FileStore{
		path:  path,
		state: domain.EmptyState(),
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
	return state
}

func cloneState(in domain.State) domain.State {
	raw, _ := json.Marshal(in)
	var out domain.State
	_ = json.Unmarshal(raw, &out)
	return withDefaults(out)
}
