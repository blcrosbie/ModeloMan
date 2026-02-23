package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/bcrosbie/modeloman/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
}

const (
	defaultDBMaxOpenConns    = 25
	defaultDBMaxIdleConns    = 10
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBConnMaxIdleTime = 5 * time.Minute
	defaultDBPingTimeout     = 5 * time.Second
)

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, domain.InvalidArgument("DATABASE_URL is required when STORE_DRIVER=postgres")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, domain.Internal("failed to open postgres connection", err)
	}
	db.SetMaxOpenConns(defaultDBMaxOpenConns)
	db.SetMaxIdleConns(defaultDBMaxIdleConns)
	db.SetConnMaxLifetime(defaultDBConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultDBConnMaxIdleTime)

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Load() error {
	pingCtx, cancel := context.WithTimeout(context.Background(), defaultDBPingTimeout)
	defer cancel()
	if err := s.db.PingContext(pingCtx); err != nil {
		return domain.Internal("failed to connect to postgres", err)
	}
	return s.verifySchemaReady()
}

func (s *PostgresStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PostgresStore) verifySchemaReady() error {
	requiredTables := []string{
		"tasks",
		"notes",
		"changelog",
		"benchmarks",
		"agent_runs",
		"prompt_attempts",
		"run_events",
		"agent_api_keys",
		"idempotency_keys",
		"orchestration_policy",
		"policy_caps",
	}

	for _, tableName := range requiredTables {
		var exists bool
		if err := s.db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+tableName).Scan(&exists); err != nil {
			return domain.Internal("failed to verify database schema", err)
		}
		if !exists {
			return domain.FailedPrecondition(fmt.Sprintf("required table %q is missing; run database migrations before starting modeloman", tableName))
		}
	}

	var hasTimescaleExtension bool
	if err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_extension
			WHERE extname = 'timescaledb'
		)
	`).Scan(&hasTimescaleExtension); err != nil {
		return domain.Internal("failed to verify timescaledb extension", err)
	}
	if !hasTimescaleExtension {
		return domain.FailedPrecondition("timescaledb extension is not installed; run database migrations before starting modeloman")
	}

	return nil
}

func (s *PostgresStore) ExportState() (domain.State, error) {
	policy, err := s.GetPolicy()
	if err != nil {
		return domain.State{}, err
	}
	policyCaps, err := s.ListPolicyCaps()
	if err != nil {
		return domain.State{}, err
	}
	tasks, err := s.ListTasks()
	if err != nil {
		return domain.State{}, err
	}
	notes, err := s.ListNotes()
	if err != nil {
		return domain.State{}, err
	}
	changelog, err := s.ListChangelog()
	if err != nil {
		return domain.State{}, err
	}
	benchmarks, err := s.ListBenchmarks()
	if err != nil {
		return domain.State{}, err
	}
	runs, err := s.ListRuns()
	if err != nil {
		return domain.State{}, err
	}
	attempts, err := s.ListPromptAttempts("")
	if err != nil {
		return domain.State{}, err
	}
	runEvents, err := s.ListRunEvents("")
	if err != nil {
		return domain.State{}, err
	}

	return domain.State{
		Tasks:      tasks,
		Notes:      notes,
		Changelog:  changelog,
		Benchmarks: benchmarks,
		Runs:       runs,
		Attempts:   attempts,
		RunEvents:  runEvents,
		Policy:     policy,
		PolicyCaps: policyCaps,
	}, nil
}

func (s *PostgresStore) GetPolicy() (domain.OrchestrationPolicy, error) {
	row := s.db.QueryRow(`
		SELECT kill_switch, kill_switch_reason, max_cost_per_run_usd, max_attempts_per_run,
		       max_tokens_per_run, max_latency_per_attempt_ms, updated_at
		FROM orchestration_policy
		WHERE policy_id = 1
	`)

	policy := domain.DefaultPolicy()
	var updatedAt time.Time
	if err := row.Scan(
		&policy.KillSwitch,
		&policy.KillSwitchReason,
		&policy.MaxCostPerRunUSD,
		&policy.MaxAttemptsPerRun,
		&policy.MaxTokensPerRun,
		&policy.MaxLatencyPerAttemptMS,
		&updatedAt,
	); err != nil {
		return domain.OrchestrationPolicy{}, domain.Internal("failed to read orchestration policy", err)
	}
	policy.UpdatedAt = formatTime(updatedAt)
	return policy, nil
}

func (s *PostgresStore) SetPolicy(policy domain.OrchestrationPolicy) error {
	_, err := s.db.Exec(`
		UPDATE orchestration_policy
		SET kill_switch = $1,
		    kill_switch_reason = $2,
		    max_cost_per_run_usd = $3,
		    max_attempts_per_run = $4,
		    max_tokens_per_run = $5,
		    max_latency_per_attempt_ms = $6,
		    updated_at = NOW()
		WHERE policy_id = 1
	`, policy.KillSwitch, policy.KillSwitchReason, policy.MaxCostPerRunUSD, policy.MaxAttemptsPerRun, policy.MaxTokensPerRun, policy.MaxLatencyPerAttemptMS)
	if err != nil {
		return domain.Internal("failed to update orchestration policy", err)
	}
	return nil
}

func (s *PostgresStore) ListPolicyCaps() ([]domain.PolicyCap, error) {
	rows, err := s.db.Query(`
		SELECT id, name, provider_type, provider, model,
		       max_cost_per_run_usd, max_attempts_per_run, max_tokens_per_run,
		       max_cost_per_attempt_usd, max_tokens_per_attempt, max_latency_per_attempt_ms,
		       priority, dry_run, is_active, updated_at
		FROM policy_caps
		ORDER BY priority DESC, id ASC
	`)
	if err != nil {
		return nil, domain.Internal("failed to list policy caps", err)
	}
	defer rows.Close()

	items := []domain.PolicyCap{}
	for rows.Next() {
		var item domain.PolicyCap
		var updatedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.ProviderType,
			&item.Provider,
			&item.Model,
			&item.MaxCostPerRunUSD,
			&item.MaxAttemptsPerRun,
			&item.MaxTokensPerRun,
			&item.MaxCostPerAttemptUSD,
			&item.MaxTokensPerAttempt,
			&item.MaxLatencyPerAttemptMS,
			&item.Priority,
			&item.DryRun,
			&item.IsActive,
			&updatedAt,
		); err != nil {
			return nil, domain.Internal("failed to decode policy cap row", err)
		}
		item.UpdatedAt = formatTime(updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate policy cap rows", err)
	}
	return items, nil
}

func (s *PostgresStore) UpsertPolicyCap(cap domain.PolicyCap) error {
	_, err := s.db.Exec(`
		INSERT INTO policy_caps (
			id, name, provider_type, provider, model,
			max_cost_per_run_usd, max_attempts_per_run, max_tokens_per_run,
			max_cost_per_attempt_usd, max_tokens_per_attempt, max_latency_per_attempt_ms,
			priority, dry_run, is_active, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, NOW()
		)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    provider_type = EXCLUDED.provider_type,
		    provider = EXCLUDED.provider,
		    model = EXCLUDED.model,
		    max_cost_per_run_usd = EXCLUDED.max_cost_per_run_usd,
		    max_attempts_per_run = EXCLUDED.max_attempts_per_run,
		    max_tokens_per_run = EXCLUDED.max_tokens_per_run,
		    max_cost_per_attempt_usd = EXCLUDED.max_cost_per_attempt_usd,
		    max_tokens_per_attempt = EXCLUDED.max_tokens_per_attempt,
		    max_latency_per_attempt_ms = EXCLUDED.max_latency_per_attempt_ms,
		    priority = EXCLUDED.priority,
		    dry_run = EXCLUDED.dry_run,
		    is_active = EXCLUDED.is_active,
		    updated_at = NOW()
	`, cap.ID, cap.Name, cap.ProviderType, cap.Provider, cap.Model,
		cap.MaxCostPerRunUSD, cap.MaxAttemptsPerRun, cap.MaxTokensPerRun,
		cap.MaxCostPerAttemptUSD, cap.MaxTokensPerAttempt, cap.MaxLatencyPerAttemptMS,
		cap.Priority, cap.DryRun, cap.IsActive)
	if err != nil {
		return domain.Internal("failed to upsert policy cap", err)
	}
	return nil
}

func (s *PostgresStore) DeletePolicyCap(id string) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM policy_caps WHERE id = $1`, id)
	if err != nil {
		return false, domain.Internal("failed to delete policy cap", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, domain.Internal("failed to read policy cap delete result", err)
	}
	return affected > 0, nil
}

func (s *PostgresStore) ListTasks() ([]domain.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, title, details, status, tags, created_at, updated_at
		FROM tasks
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, domain.Internal("failed to list tasks", err)
	}
	defer rows.Close()

	items := []domain.Task{}
	for rows.Next() {
		var item domain.Task
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Details,
			&item.Status,
			&item.Tags,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, domain.Internal("failed to decode task row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		item.UpdatedAt = formatTime(updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate task rows", err)
	}
	return items, nil
}

func (s *PostgresStore) UpsertTask(task domain.Task) error {
	createdAt, err := parseTimestamp(task.CreatedAt)
	if err != nil {
		return domain.Internal("task created_at is invalid", err)
	}
	updatedAt, err := parseTimestamp(task.UpdatedAt)
	if err != nil {
		return domain.Internal("task updated_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO tasks (id, title, details, status, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
		    details = EXCLUDED.details,
		    status = EXCLUDED.status,
		    tags = EXCLUDED.tags,
		    updated_at = EXCLUDED.updated_at
	`, task.ID, task.Title, task.Details, task.Status, task.Tags, createdAt, updatedAt)
	if err != nil {
		return domain.Internal("failed to upsert task", err)
	}
	return nil
}

func (s *PostgresStore) DeleteTask(id string) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return false, domain.Internal("failed to delete task", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, domain.Internal("failed to read delete result", err)
	}
	return affected > 0, nil
}

func (s *PostgresStore) ListNotes() ([]domain.Note, error) {
	rows, err := s.db.Query(`
		SELECT id, title, body, tags, created_at
		FROM notes
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, domain.Internal("failed to list notes", err)
	}
	defer rows.Close()

	items := []domain.Note{}
	for rows.Next() {
		var item domain.Note
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.Tags, &createdAt); err != nil {
			return nil, domain.Internal("failed to decode note row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate note rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertNote(note domain.Note) error {
	createdAt, err := parseTimestamp(note.CreatedAt)
	if err != nil {
		return domain.Internal("note created_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO notes (id, title, body, tags, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, note.ID, note.Title, note.Body, note.Tags, createdAt)
	if err != nil {
		return domain.Internal("failed to insert note", err)
	}
	return nil
}

func (s *PostgresStore) ListChangelog() ([]domain.ChangelogEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, category, summary, details, actor, created_at
		FROM changelog
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, domain.Internal("failed to list changelog", err)
	}
	defer rows.Close()

	items := []domain.ChangelogEntry{}
	for rows.Next() {
		var item domain.ChangelogEntry
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Category, &item.Summary, &item.Details, &item.Actor, &createdAt); err != nil {
			return nil, domain.Internal("failed to decode changelog row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate changelog rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertChangelog(entry domain.ChangelogEntry) error {
	createdAt, err := parseTimestamp(entry.CreatedAt)
	if err != nil {
		return domain.Internal("changelog created_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO changelog (id, category, summary, details, actor, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, entry.ID, entry.Category, entry.Summary, entry.Details, entry.Actor, createdAt)
	if err != nil {
		return domain.Internal("failed to insert changelog", err)
	}
	return nil
}

func (s *PostgresStore) ListBenchmarks() ([]domain.Benchmark, error) {
	rows, err := s.db.Query(`
		SELECT id, workflow, provider_type, provider, model,
		       tokens_in, tokens_out, cost_usd, latency_ms, quality_score, notes, created_at
		FROM benchmarks
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, domain.Internal("failed to list benchmarks", err)
	}
	defer rows.Close()

	items := []domain.Benchmark{}
	for rows.Next() {
		var item domain.Benchmark
		var createdAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.Workflow,
			&item.ProviderType,
			&item.Provider,
			&item.Model,
			&item.TokensIn,
			&item.TokensOut,
			&item.CostUSD,
			&item.LatencyMS,
			&item.QualityScore,
			&item.Notes,
			&createdAt,
		); err != nil {
			return nil, domain.Internal("failed to decode benchmark row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate benchmark rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertBenchmark(benchmark domain.Benchmark) error {
	createdAt, err := parseTimestamp(benchmark.CreatedAt)
	if err != nil {
		return domain.Internal("benchmark created_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO benchmarks (
			id, workflow, provider_type, provider, model,
			tokens_in, tokens_out, cost_usd, latency_ms, quality_score, notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12
		)
	`, benchmark.ID, benchmark.Workflow, benchmark.ProviderType, benchmark.Provider, benchmark.Model,
		benchmark.TokensIn, benchmark.TokensOut, benchmark.CostUSD, benchmark.LatencyMS, benchmark.QualityScore, benchmark.Notes, createdAt)
	if err != nil {
		return domain.Internal("failed to insert benchmark", err)
	}
	return nil
}

func (s *PostgresStore) ListRuns() ([]domain.AgentRun, error) {
	return s.ListRunsFiltered(domain.RunFilter{})
}

func (s *PostgresStore) ListRunsFiltered(filter domain.RunFilter) ([]domain.AgentRun, error) {
	query := `
		SELECT id, task_id, workflow, agent_id, prompt_version, model_policy, status, max_retries,
		       total_attempts, success_attempts, failed_attempts, total_tokens_in, total_tokens_out,
		       total_cost_usd, duration_ms, last_error, started_at, finished_at
		FROM agent_runs
	`
	args := []any{}
	conditions := []string{}

	if strings.TrimSpace(filter.RunID) != "" {
		args = append(args, filter.RunID)
		conditions = append(conditions, fmt.Sprintf("id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.TaskID) != "" {
		args = append(args, filter.TaskID)
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Workflow) != "" {
		args = append(args, filter.Workflow)
		conditions = append(conditions, fmt.Sprintf("workflow = $%d", len(args)))
	}
	if strings.TrimSpace(filter.AgentID) != "" {
		args = append(args, filter.AgentID)
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.PromptVersion) != "" {
		args = append(args, filter.PromptVersion)
		conditions = append(conditions, fmt.Sprintf("prompt_version = $%d", len(args)))
	}
	if strings.TrimSpace(filter.StartedAfter) != "" {
		args = append(args, filter.StartedAfter)
		conditions = append(conditions, fmt.Sprintf("started_at >= $%d::timestamptz", len(args)))
	}
	if strings.TrimSpace(filter.StartedBefore) != "" {
		args = append(args, filter.StartedBefore)
		conditions = append(conditions, fmt.Sprintf("started_at <= $%d::timestamptz", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY started_at DESC, id DESC "
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d ", len(args))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, domain.Internal("failed to list runs", err)
	}
	defer rows.Close()

	items := []domain.AgentRun{}
	for rows.Next() {
		var item domain.AgentRun
		var startedAt time.Time
		var finishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.TaskID,
			&item.Workflow,
			&item.AgentID,
			&item.PromptVersion,
			&item.ModelPolicy,
			&item.Status,
			&item.MaxRetries,
			&item.TotalAttempts,
			&item.SuccessAttempts,
			&item.FailedAttempts,
			&item.TotalTokensIn,
			&item.TotalTokensOut,
			&item.TotalCostUSD,
			&item.DurationMS,
			&item.LastError,
			&startedAt,
			&finishedAt,
		); err != nil {
			return nil, domain.Internal("failed to decode run row", err)
		}
		item.StartedAt = formatTime(startedAt)
		if finishedAt.Valid {
			item.FinishedAt = formatTime(finishedAt.Time)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate run rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertRun(run domain.AgentRun) error {
	startedAt, err := parseTimestamp(run.StartedAt)
	if err != nil {
		return domain.Internal("run started_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO agent_runs (
			id, task_id, workflow, agent_id, prompt_version, model_policy, status, max_retries,
			total_attempts, success_attempts, failed_attempts, total_tokens_in, total_tokens_out,
			total_cost_usd, duration_ms, last_error, started_at, finished_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18
		)
	`, run.ID, run.TaskID, run.Workflow, run.AgentID, run.PromptVersion, run.ModelPolicy, run.Status, run.MaxRetries,
		run.TotalAttempts, run.SuccessAttempts, run.FailedAttempts, run.TotalTokensIn, run.TotalTokensOut,
		run.TotalCostUSD, run.DurationMS, run.LastError, startedAt, nullableTimestamp(run.FinishedAt))
	if err != nil {
		return domain.Internal("failed to insert run", err)
	}
	return nil
}

func (s *PostgresStore) UpdateRun(run domain.AgentRun) error {
	_, err := s.db.Exec(`
		UPDATE agent_runs
		SET task_id = $2,
		    workflow = $3,
		    agent_id = $4,
		    prompt_version = $5,
		    model_policy = $6,
		    status = $7,
		    max_retries = $8,
		    total_attempts = $9,
		    success_attempts = $10,
		    failed_attempts = $11,
		    total_tokens_in = $12,
		    total_tokens_out = $13,
		    total_cost_usd = $14,
		    duration_ms = $15,
		    last_error = $16,
		    finished_at = $17
		WHERE id = $1
	`, run.ID, run.TaskID, run.Workflow, run.AgentID, run.PromptVersion, run.ModelPolicy, run.Status, run.MaxRetries,
		run.TotalAttempts, run.SuccessAttempts, run.FailedAttempts, run.TotalTokensIn, run.TotalTokensOut,
		run.TotalCostUSD, run.DurationMS, run.LastError, nullableTimestamp(run.FinishedAt))
	if err != nil {
		return domain.Internal("failed to update run", err)
	}
	return nil
}

func (s *PostgresStore) ListPromptAttempts(runID string) ([]domain.PromptAttempt, error) {
	return s.ListPromptAttemptsFiltered(domain.AttemptFilter{RunID: runID})
}

func (s *PostgresStore) ListPromptAttemptsFiltered(filter domain.AttemptFilter) ([]domain.PromptAttempt, error) {
	query := `
		SELECT id, run_id, attempt_number, workflow, agent_id, provider_type, provider, model,
		       prompt_version, prompt_hash, outcome, error_type, error_message, tokens_in, tokens_out,
		       cost_usd, latency_ms, quality_score, created_at
		FROM prompt_attempts
	`
	args := []any{}
	conditions := []string{}
	if strings.TrimSpace(filter.RunID) != "" {
		args = append(args, filter.RunID)
		conditions = append(conditions, fmt.Sprintf("run_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Workflow) != "" {
		args = append(args, filter.Workflow)
		conditions = append(conditions, fmt.Sprintf("workflow = $%d", len(args)))
	}
	if strings.TrimSpace(filter.AgentID) != "" {
		args = append(args, filter.AgentID)
		conditions = append(conditions, fmt.Sprintf("agent_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Model) != "" {
		args = append(args, filter.Model)
		conditions = append(conditions, fmt.Sprintf("model = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Outcome) != "" {
		args = append(args, filter.Outcome)
		conditions = append(conditions, fmt.Sprintf("outcome = $%d", len(args)))
	}
	if strings.TrimSpace(filter.PromptVersion) != "" {
		args = append(args, filter.PromptVersion)
		conditions = append(conditions, fmt.Sprintf("prompt_version = $%d", len(args)))
	}
	if strings.TrimSpace(filter.CreatedAfter) != "" {
		args = append(args, filter.CreatedAfter)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d::timestamptz", len(args)))
	}
	if strings.TrimSpace(filter.CreatedBefore) != "" {
		args = append(args, filter.CreatedBefore)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d::timestamptz", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at DESC, id DESC `
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d ", len(args))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, domain.Internal("failed to list prompt attempts", err)
	}
	defer rows.Close()

	items := []domain.PromptAttempt{}
	for rows.Next() {
		var item domain.PromptAttempt
		var createdAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.RunID,
			&item.AttemptNumber,
			&item.Workflow,
			&item.AgentID,
			&item.ProviderType,
			&item.Provider,
			&item.Model,
			&item.PromptVersion,
			&item.PromptHash,
			&item.Outcome,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.TokensIn,
			&item.TokensOut,
			&item.CostUSD,
			&item.LatencyMS,
			&item.QualityScore,
			&createdAt,
		); err != nil {
			return nil, domain.Internal("failed to decode prompt attempt row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate prompt attempt rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertPromptAttempt(attempt domain.PromptAttempt) error {
	createdAt, err := parseTimestamp(attempt.CreatedAt)
	if err != nil {
		return domain.Internal("prompt attempt created_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO prompt_attempts (
			id, run_id, attempt_number, workflow, agent_id, provider_type, provider, model,
			prompt_version, prompt_hash, outcome, error_type, error_message, tokens_in, tokens_out,
			cost_usd, latency_ms, quality_score, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19
		)
	`, attempt.ID, attempt.RunID, attempt.AttemptNumber, attempt.Workflow, attempt.AgentID, attempt.ProviderType, attempt.Provider, attempt.Model,
		attempt.PromptVersion, attempt.PromptHash, attempt.Outcome, attempt.ErrorType, attempt.ErrorMessage, attempt.TokensIn, attempt.TokensOut,
		attempt.CostUSD, attempt.LatencyMS, attempt.QualityScore, createdAt)
	if err != nil {
		return domain.Internal("failed to insert prompt attempt", err)
	}
	return nil
}

func (s *PostgresStore) ListRunEvents(runID string) ([]domain.RunEvent, error) {
	return s.ListRunEventsFiltered(domain.EventFilter{RunID: runID})
}

func (s *PostgresStore) ListRunEventsFiltered(filter domain.EventFilter) ([]domain.RunEvent, error) {
	query := `
		SELECT id, run_id, event_type, level, message, data_json, created_at
		FROM run_events
	`
	args := []any{}
	conditions := []string{}
	if strings.TrimSpace(filter.RunID) != "" {
		args = append(args, filter.RunID)
		conditions = append(conditions, fmt.Sprintf("run_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.EventType) != "" {
		args = append(args, filter.EventType)
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Level) != "" {
		args = append(args, filter.Level)
		conditions = append(conditions, fmt.Sprintf("level = $%d", len(args)))
	}
	if strings.TrimSpace(filter.CreatedAfter) != "" {
		args = append(args, filter.CreatedAfter)
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d::timestamptz", len(args)))
	}
	if strings.TrimSpace(filter.CreatedBefore) != "" {
		args = append(args, filter.CreatedBefore)
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d::timestamptz", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at DESC, id DESC `
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d ", len(args))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, domain.Internal("failed to list run events", err)
	}
	defer rows.Close()

	items := []domain.RunEvent{}
	for rows.Next() {
		var item domain.RunEvent
		var createdAt time.Time
		if err := rows.Scan(
			&item.ID,
			&item.RunID,
			&item.EventType,
			&item.Level,
			&item.Message,
			&item.DataJSON,
			&createdAt,
		); err != nil {
			return nil, domain.Internal("failed to decode run event row", err)
		}
		item.CreatedAt = formatTime(createdAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Internal("failed to iterate run event rows", err)
	}
	return items, nil
}

func (s *PostgresStore) InsertRunEvent(event domain.RunEvent) error {
	createdAt, err := parseTimestamp(event.CreatedAt)
	if err != nil {
		return domain.Internal("run event created_at is invalid", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO run_events (id, run_id, event_type, level, message, data_json, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.ID, event.RunID, event.EventType, event.Level, event.Message, event.DataJSON, createdAt)
	if err != nil {
		return domain.Internal("failed to insert run event", err)
	}
	return nil
}

func (s *PostgresStore) AuthenticateAgentKey(rawKey string) (AgentPrincipal, bool, error) {
	hash := hashAPIKey(rawKey)
	if hash == "" {
		return AgentPrincipal{}, false, nil
	}

	row := s.db.QueryRow(`
		SELECT agent_id, key_id, scopes
		FROM agent_api_keys
		WHERE key_hash = $1
		  AND is_active = TRUE
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, hash)

	var principal AgentPrincipal
	if err := row.Scan(&principal.AgentID, &principal.KeyID, &principal.Scopes); err != nil {
		if err == sql.ErrNoRows {
			return AgentPrincipal{}, false, nil
		}
		return AgentPrincipal{}, false, domain.Internal("failed to validate api key", err)
	}

	if _, err := s.db.Exec(`UPDATE agent_api_keys SET last_used_at = NOW() WHERE key_id = $1`, principal.KeyID); err != nil {
		return AgentPrincipal{}, false, domain.Internal("failed to update api key last_used_at", err)
	}

	return principal, true, nil
}

func (s *PostgresStore) EnsureAgentKey(agentID, rawKey string) (string, bool, error) {
	cleanAgentID := strings.TrimSpace(agentID)
	if cleanAgentID == "" {
		return "", false, domain.InvalidArgument("agentID is required")
	}
	hash := hashAPIKey(rawKey)
	if hash == "" {
		return "", false, domain.InvalidArgument("raw agent key is required")
	}

	var existingKeyID string
	err := s.db.QueryRow(`SELECT key_id FROM agent_api_keys WHERE key_hash = $1`, hash).Scan(&existingKeyID)
	if err == nil {
		return existingKeyID, false, nil
	}
	if err != sql.ErrNoRows {
		return "", false, domain.Internal("failed to query existing api key", err)
	}

	keyID := newKeyID(cleanAgentID)
	_, err = s.db.Exec(`
		INSERT INTO agent_api_keys (
			agent_id, key_id, key_hash, is_active, created_at, last_used_at
		) VALUES ($1, $2, $3, TRUE, NOW(), NULL)
	`, cleanAgentID, keyID, hash)
	if err != nil {
		return "", false, domain.Internal("failed to insert api key", err)
	}
	return keyID, true, nil
}

func (s *PostgresStore) ReserveIdempotencyKey(method, idempotencyKey, requestHash string) (IdempotencyRecord, bool, error) {
	method = strings.TrimSpace(method)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	requestHash = strings.TrimSpace(requestHash)
	if method == "" || idempotencyKey == "" || requestHash == "" {
		return IdempotencyRecord{}, false, domain.InvalidArgument("method, idempotency_key, and request_hash are required")
	}

	result, err := s.db.Exec(`
		INSERT INTO idempotency_keys (method, idempotency_key, request_hash, response_json, created_at, completed_at)
		VALUES ($1, $2, $3, '', NOW(), NULL)
		ON CONFLICT (method, idempotency_key) DO NOTHING
	`, method, idempotencyKey, requestHash)
	if err != nil {
		return IdempotencyRecord{}, false, domain.Internal("failed to reserve idempotency key", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return IdempotencyRecord{}, false, domain.Internal("failed to read idempotency key reserve result", err)
	}
	if affected > 0 {
		return IdempotencyRecord{}, true, nil
	}

	var record IdempotencyRecord
	if err := s.db.QueryRow(`
		SELECT request_hash, response_json, completed_at IS NOT NULL
		FROM idempotency_keys
		WHERE method = $1 AND idempotency_key = $2
	`, method, idempotencyKey).Scan(&record.RequestHash, &record.ResponseJSON, &record.Completed); err != nil {
		if err == sql.ErrNoRows {
			return IdempotencyRecord{}, false, domain.NotFound("idempotency key was not found after reserve conflict")
		}
		return IdempotencyRecord{}, false, domain.Internal("failed to read existing idempotency key", err)
	}
	return record, false, nil
}

func (s *PostgresStore) CompleteIdempotencyKey(method, idempotencyKey, responseJSON string) error {
	method = strings.TrimSpace(method)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if method == "" || idempotencyKey == "" {
		return domain.InvalidArgument("method and idempotency_key are required")
	}

	result, err := s.db.Exec(`
		UPDATE idempotency_keys
		SET response_json = $3,
		    completed_at = NOW()
		WHERE method = $1
		  AND idempotency_key = $2
		  AND completed_at IS NULL
	`, method, idempotencyKey, responseJSON)
	if err != nil {
		return domain.Internal("failed to complete idempotency key", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.Internal("failed to read idempotency completion result", err)
	}
	if affected > 0 {
		return nil
	}

	var completed bool
	if err := s.db.QueryRow(`
		SELECT completed_at IS NOT NULL
		FROM idempotency_keys
		WHERE method = $1 AND idempotency_key = $2
	`, method, idempotencyKey).Scan(&completed); err != nil {
		if err == sql.ErrNoRows {
			return domain.NotFound("idempotency key not found")
		}
		return domain.Internal("failed to verify idempotency completion", err)
	}
	if completed {
		return nil
	}
	return domain.Internal("idempotency key completion did not apply", nil)
}

func (s *PostgresStore) ReleaseIdempotencyKey(method, idempotencyKey string) error {
	method = strings.TrimSpace(method)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if method == "" || idempotencyKey == "" {
		return nil
	}

	if _, err := s.db.Exec(`
		DELETE FROM idempotency_keys
		WHERE method = $1
		  AND idempotency_key = $2
		  AND completed_at IS NULL
	`, method, idempotencyKey); err != nil {
		return domain.Internal("failed to release idempotency key", err)
	}
	return nil
}

func (s *PostgresStore) ensureSchema() error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS timescaledb`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			tags TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			tags TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS changelog (
			id TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			summary TEXT NOT NULL,
			details TEXT NOT NULL DEFAULT '',
			actor TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS benchmarks (
			id TEXT NOT NULL,
			workflow TEXT NOT NULL,
			provider_type TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			tokens_in BIGINT NOT NULL DEFAULT 0,
			tokens_out BIGINT NOT NULL DEFAULT 0,
			cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			latency_ms BIGINT NOT NULL DEFAULT 0,
			quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			notes TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS agent_runs (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL DEFAULT '',
			workflow TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			prompt_version TEXT NOT NULL DEFAULT '',
			model_policy TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			max_retries BIGINT NOT NULL DEFAULT 0,
			total_attempts BIGINT NOT NULL DEFAULT 0,
			success_attempts BIGINT NOT NULL DEFAULT 0,
			failed_attempts BIGINT NOT NULL DEFAULT 0,
			total_tokens_in BIGINT NOT NULL DEFAULT 0,
			total_tokens_out BIGINT NOT NULL DEFAULT 0,
			total_cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			duration_ms BIGINT NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			started_at TIMESTAMPTZ NOT NULL,
			finished_at TIMESTAMPTZ NULL
		)`,
		`CREATE TABLE IF NOT EXISTS prompt_attempts (
			id TEXT NOT NULL,
			run_id TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
			attempt_number BIGINT NOT NULL,
			workflow TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			provider_type TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL,
			prompt_version TEXT NOT NULL DEFAULT '',
			prompt_hash TEXT NOT NULL DEFAULT '',
			outcome TEXT NOT NULL,
			error_type TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			tokens_in BIGINT NOT NULL DEFAULT 0,
			tokens_out BIGINT NOT NULL DEFAULT 0,
			cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			latency_ms BIGINT NOT NULL DEFAULT 0,
			quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS run_events (
			id TEXT NOT NULL,
			run_id TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
			event_type TEXT NOT NULL,
			level TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (id, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS agent_api_keys (
			agent_id TEXT NOT NULL,
			key_id TEXT PRIMARY KEY,
			key_hash TEXT NOT NULL UNIQUE,
			scopes TEXT[] NOT NULL DEFAULT ARRAY['tasks:write', 'telemetry:write', 'policy:write', 'admin:read']::TEXT[],
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_used_at TIMESTAMPTZ NULL,
			expires_at TIMESTAMPTZ NULL,
			revoked_at TIMESTAMPTZ NULL
		)`,
		`ALTER TABLE agent_api_keys ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT ARRAY['tasks:write', 'telemetry:write', 'policy:write', 'admin:read']::TEXT[]`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			method TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			response_json TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ NULL,
			PRIMARY KEY (method, idempotency_key)
		)`,
		`CREATE TABLE IF NOT EXISTS orchestration_policy (
			policy_id SMALLINT PRIMARY KEY,
			kill_switch BOOLEAN NOT NULL DEFAULT FALSE,
			kill_switch_reason TEXT NOT NULL DEFAULT '',
			max_cost_per_run_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			max_attempts_per_run BIGINT NOT NULL DEFAULT 0,
			max_tokens_per_run BIGINT NOT NULL DEFAULT 0,
			max_latency_per_attempt_ms BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO orchestration_policy (
			policy_id, kill_switch, kill_switch_reason, max_cost_per_run_usd, max_attempts_per_run, max_tokens_per_run, max_latency_per_attempt_ms, updated_at
		) VALUES (1, FALSE, '', 0, 0, 0, 0, NOW())
		ON CONFLICT (policy_id) DO NOTHING`,
		`CREATE TABLE IF NOT EXISTS policy_caps (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			provider_type TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			max_cost_per_run_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			max_attempts_per_run BIGINT NOT NULL DEFAULT 0,
			max_tokens_per_run BIGINT NOT NULL DEFAULT 0,
			max_cost_per_attempt_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
			max_tokens_per_attempt BIGINT NOT NULL DEFAULT 0,
			max_latency_per_attempt_ms BIGINT NOT NULL DEFAULT 0,
			priority BIGINT NOT NULL DEFAULT 0,
			dry_run BOOLEAN NOT NULL DEFAULT FALSE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`ALTER TABLE policy_caps ADD COLUMN IF NOT EXISTS dry_run BOOLEAN NOT NULL DEFAULT FALSE`,
		`SELECT create_hypertable('benchmarks', 'created_at', if_not_exists => TRUE, migrate_data => TRUE)`,
		`SELECT create_hypertable('prompt_attempts', 'created_at', if_not_exists => TRUE, migrate_data => TRUE)`,
		`SELECT create_hypertable('run_events', 'created_at', if_not_exists => TRUE, migrate_data => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks (updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_changelog_created_at ON changelog (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_benchmarks_created_at ON benchmarks (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_benchmarks_workflow_created_at ON benchmarks (workflow, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_started_at ON agent_runs (started_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_status_started_at ON agent_runs (status, started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_prompt_attempts_run_created_at ON prompt_attempts (run_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_prompt_attempts_outcome_created_at ON prompt_attempts (outcome, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_run_events_run_created_at ON run_events (run_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_api_keys_agent_id ON agent_api_keys (agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_api_keys_active ON agent_api_keys (is_active, revoked_at, expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at ON idempotency_keys (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_policy_caps_lookup ON policy_caps (provider_type, provider, model, is_active, priority DESC)`,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return domain.Internal("failed to start schema transaction", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return domain.Internal(fmt.Sprintf("failed to run schema statement: %s", statement), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Internal("failed to commit schema transaction", err)
	}
	return nil
}

func parseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTimestamp(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	parsed, err := parseTimestamp(clean)
	if err != nil {
		return nil
	}
	return parsed
}

func hashAPIKey(rawKey string) string {
	clean := strings.TrimSpace(rawKey)
	if clean == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(hash[:])
}

func newKeyID(agentID string) string {
	return fmt.Sprintf("ak_%s_%d", strings.ToLower(agentID), time.Now().UTC().UnixNano())
}
