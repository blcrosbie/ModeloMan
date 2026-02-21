package domain

type Task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Details   string   `json:"details"`
	Status    string   `json:"status"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type Note struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
}

type ChangelogEntry struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Summary   string `json:"summary"`
	Details   string `json:"details"`
	Actor     string `json:"actor"`
	CreatedAt string `json:"created_at"`
}

type Benchmark struct {
	ID           string  `json:"id"`
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
	CreatedAt    string  `json:"created_at"`
}

type AgentRun struct {
	ID              string  `json:"id"`
	TaskID          string  `json:"task_id"`
	Workflow        string  `json:"workflow"`
	AgentID         string  `json:"agent_id"`
	PromptVersion   string  `json:"prompt_version"`
	ModelPolicy     string  `json:"model_policy"`
	Status          string  `json:"status"`
	MaxRetries      int64   `json:"max_retries"`
	TotalAttempts   int64   `json:"total_attempts"`
	SuccessAttempts int64   `json:"success_attempts"`
	FailedAttempts  int64   `json:"failed_attempts"`
	TotalTokensIn   int64   `json:"total_tokens_in"`
	TotalTokensOut  int64   `json:"total_tokens_out"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	DurationMS      int64   `json:"duration_ms"`
	LastError       string  `json:"last_error"`
	StartedAt       string  `json:"started_at"`
	FinishedAt      string  `json:"finished_at"`
}

type PromptAttempt struct {
	ID            string  `json:"id"`
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
	CreatedAt     string  `json:"created_at"`
}

type RunEvent struct {
	ID        string `json:"id"`
	RunID     string `json:"run_id"`
	EventType string `json:"event_type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	DataJSON  string `json:"data_json"`
	CreatedAt string `json:"created_at"`
}

type OrchestrationPolicy struct {
	KillSwitch             bool    `json:"kill_switch"`
	KillSwitchReason       string  `json:"kill_switch_reason"`
	MaxCostPerRunUSD       float64 `json:"max_cost_per_run_usd"`
	MaxAttemptsPerRun      int64   `json:"max_attempts_per_run"`
	MaxTokensPerRun        int64   `json:"max_tokens_per_run"`
	MaxLatencyPerAttemptMS int64   `json:"max_latency_per_attempt_ms"`
	UpdatedAt              string  `json:"updated_at"`
}

type PolicyCap struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	ProviderType           string  `json:"provider_type"`
	Provider               string  `json:"provider"`
	Model                  string  `json:"model"`
	MaxCostPerRunUSD       float64 `json:"max_cost_per_run_usd"`
	MaxAttemptsPerRun      int64   `json:"max_attempts_per_run"`
	MaxTokensPerRun        int64   `json:"max_tokens_per_run"`
	MaxCostPerAttemptUSD   float64 `json:"max_cost_per_attempt_usd"`
	MaxTokensPerAttempt    int64   `json:"max_tokens_per_attempt"`
	MaxLatencyPerAttemptMS int64   `json:"max_latency_per_attempt_ms"`
	Priority               int64   `json:"priority"`
	DryRun                 bool    `json:"dry_run"`
	IsActive               bool    `json:"is_active"`
	UpdatedAt              string  `json:"updated_at"`
}

type RunFilter struct {
	RunID         string
	TaskID        string
	Workflow      string
	AgentID       string
	Status        string
	PromptVersion string
	StartedAfter  string
	StartedBefore string
	Limit         int64
}

type AttemptFilter struct {
	RunID         string
	Workflow      string
	AgentID       string
	Model         string
	Outcome       string
	PromptVersion string
	CreatedAfter  string
	CreatedBefore string
	Limit         int64
}

type EventFilter struct {
	RunID         string
	EventType     string
	Level         string
	CreatedAfter  string
	CreatedBefore string
	Limit         int64
}

type LeaderboardEntry struct {
	Workflow         string  `json:"workflow"`
	PromptVersion    string  `json:"prompt_version"`
	Model            string  `json:"model"`
	Attempts         int64   `json:"attempts"`
	SuccessAttempts  int64   `json:"success_attempts"`
	FailedAttempts   int64   `json:"failed_attempts"`
	SuccessRate      float64 `json:"success_rate"`
	AverageCostUSD   float64 `json:"average_cost_usd"`
	AverageLatencyMS float64 `json:"average_latency_ms"`
	Score            float64 `json:"score"`
}

type State struct {
	Tasks      []Task              `json:"tasks"`
	Notes      []Note              `json:"notes"`
	Changelog  []ChangelogEntry    `json:"changelog"`
	Benchmarks []Benchmark         `json:"benchmarks"`
	Runs       []AgentRun          `json:"runs"`
	Attempts   []PromptAttempt     `json:"attempts"`
	RunEvents  []RunEvent          `json:"run_events"`
	Policy     OrchestrationPolicy `json:"policy"`
	PolicyCaps []PolicyCap         `json:"policy_caps"`
}

type Summary struct {
	Counts struct {
		Tasks      int `json:"tasks"`
		Notes      int `json:"notes"`
		Changelog  int `json:"changelog"`
		Benchmarks int `json:"benchmarks"`
		Runs       int `json:"runs"`
		Attempts   int `json:"attempts"`
		RunEvents  int `json:"run_events"`
	} `json:"counts"`
	Totals struct {
		TokensIn   int64   `json:"tokens_in"`
		TokensOut  int64   `json:"tokens_out"`
		CostUSD    float64 `json:"cost_usd"`
		ByProvider map[string]struct {
			Count   int     `json:"count"`
			CostUSD float64 `json:"cost_usd"`
		} `json:"by_provider"`
	} `json:"totals"`
}

type TelemetrySummary struct {
	Counts struct {
		Runs            int64 `json:"runs"`
		RunningRuns     int64 `json:"running_runs"`
		CompletedRuns   int64 `json:"completed_runs"`
		FailedRuns      int64 `json:"failed_runs"`
		CancelledRuns   int64 `json:"cancelled_runs"`
		Attempts        int64 `json:"attempts"`
		SuccessAttempts int64 `json:"success_attempts"`
		FailedAttempts  int64 `json:"failed_attempts"`
		Retries         int64 `json:"retries"`
		Events          int64 `json:"events"`
	} `json:"counts"`
	Totals struct {
		TokensIn  int64   `json:"tokens_in"`
		TokensOut int64   `json:"tokens_out"`
		CostUSD   float64 `json:"cost_usd"`
		LatencyMS int64   `json:"latency_ms"`
	} `json:"totals"`
	Averages struct {
		AttemptLatencyMS float64 `json:"attempt_latency_ms"`
		CostPerAttempt   float64 `json:"cost_per_attempt"`
		SuccessRate      float64 `json:"success_rate"`
	} `json:"averages"`
}

func EmptyState() State {
	return State{
		Tasks:      []Task{},
		Notes:      []Note{},
		Changelog:  []ChangelogEntry{},
		Benchmarks: []Benchmark{},
		Runs:       []AgentRun{},
		Attempts:   []PromptAttempt{},
		RunEvents:  []RunEvent{},
		Policy:     DefaultPolicy(),
		PolicyCaps: []PolicyCap{},
	}
}

func DefaultPolicy() OrchestrationPolicy {
	return OrchestrationPolicy{
		KillSwitch:             false,
		KillSwitchReason:       "",
		MaxCostPerRunUSD:       0,
		MaxAttemptsPerRun:      0,
		MaxTokensPerRun:        0,
		MaxLatencyPerAttemptMS: 0,
	}
}
