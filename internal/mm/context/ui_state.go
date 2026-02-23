package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const uiStateRelPath = ".modeloman/ui_state.json"

type UIState struct {
	Version    int      `json:"version"`
	Backend    string   `json:"backend"`
	TaskType   string   `json:"task_type"`
	Skill      string   `json:"skill"`
	Budget     int      `json:"budget"`
	Objective  string   `json:"objective"`
	LastScreen string   `json:"last_screen"`
	LastFiles  []string `json:"last_files"`
	UpdatedAt  string   `json:"updated_at"`
}

func LoadUIState(repoRoot string) (UIState, error) {
	path := filepath.Join(repoRoot, uiStateRelPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return UIState{Version: 1, LastFiles: []string{}}, nil
		}
		return UIState{}, fmt.Errorf("read ui state: %w", err)
	}
	var state UIState
	if err := json.Unmarshal(raw, &state); err != nil {
		return UIState{}, fmt.Errorf("decode ui state: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.LastFiles == nil {
		state.LastFiles = []string{}
	}
	return state, nil
}

func SaveUIState(repoRoot string, state UIState) error {
	state.Version = 1
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	path := filepath.Join(repoRoot, uiStateRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir ui state dir: %w", err)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ui state: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write ui state: %w", err)
	}
	return nil
}
