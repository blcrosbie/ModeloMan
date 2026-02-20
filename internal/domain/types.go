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

type State struct {
	Tasks      []Task           `json:"tasks"`
	Notes      []Note           `json:"notes"`
	Changelog  []ChangelogEntry `json:"changelog"`
	Benchmarks []Benchmark      `json:"benchmarks"`
}

type Summary struct {
	Counts struct {
		Tasks      int `json:"tasks"`
		Notes      int `json:"notes"`
		Changelog  int `json:"changelog"`
		Benchmarks int `json:"benchmarks"`
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

func EmptyState() State {
	return State{
		Tasks:      []Task{},
		Notes:      []Note{},
		Changelog:  []ChangelogEntry{},
		Benchmarks: []Benchmark{},
	}
}
