package store

import "github.com/bcrosbie/modeloman/internal/domain"

// HubStore is the persistence contract used by the service layer.
type HubStore interface {
	Load() error
	Close() error

	ExportState() (domain.State, error)
	GetPolicy() (domain.OrchestrationPolicy, error)
	SetPolicy(domain.OrchestrationPolicy) error
	ListPolicyCaps() ([]domain.PolicyCap, error)
	UpsertPolicyCap(domain.PolicyCap) error
	DeletePolicyCap(id string) (bool, error)

	ListTasks() ([]domain.Task, error)
	UpsertTask(domain.Task) error
	DeleteTask(id string) (bool, error)

	ListNotes() ([]domain.Note, error)
	InsertNote(domain.Note) error

	ListChangelog() ([]domain.ChangelogEntry, error)
	InsertChangelog(domain.ChangelogEntry) error

	ListBenchmarks() ([]domain.Benchmark, error)
	InsertBenchmark(domain.Benchmark) error

	ListRunsFiltered(filter domain.RunFilter) ([]domain.AgentRun, error)
	ListRuns() ([]domain.AgentRun, error)
	InsertRun(domain.AgentRun) error
	UpdateRun(domain.AgentRun) error

	ListPromptAttemptsFiltered(filter domain.AttemptFilter) ([]domain.PromptAttempt, error)
	ListPromptAttempts(runID string) ([]domain.PromptAttempt, error)
	InsertPromptAttempt(domain.PromptAttempt) error

	ListRunEventsFiltered(filter domain.EventFilter) ([]domain.RunEvent, error)
	ListRunEvents(runID string) ([]domain.RunEvent, error)
	InsertRunEvent(domain.RunEvent) error
}

type AgentPrincipal struct {
	AgentID string
	KeyID   string
}

// AgentKeyAuthenticator validates write API keys and returns the caller principal.
type AgentKeyAuthenticator interface {
	AuthenticateAgentKey(rawKey string) (AgentPrincipal, bool, error)
	EnsureAgentKey(agentID, rawKey string) (keyID string, created bool, err error)
}
