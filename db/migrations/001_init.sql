CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS changelog (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    summary TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    actor TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS benchmarks (
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
);

CREATE TABLE IF NOT EXISTS agent_runs (
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
);

CREATE TABLE IF NOT EXISTS prompt_attempts (
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
);

CREATE TABLE IF NOT EXISTS run_events (
    id TEXT NOT NULL,
    run_id TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    level TEXT NOT NULL DEFAULT 'info',
    message TEXT NOT NULL DEFAULT '',
    data_json TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id, created_at)
);

CREATE TABLE IF NOT EXISTS agent_api_keys (
    agent_id TEXT NOT NULL,
    key_id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT ARRAY['tasks:write', 'telemetry:write', 'policy:write', 'admin:read']::TEXT[],
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL
);

ALTER TABLE agent_api_keys
ADD COLUMN IF NOT EXISTS scopes TEXT[] NOT NULL DEFAULT ARRAY['tasks:write', 'telemetry:write', 'policy:write', 'admin:read']::TEXT[];

CREATE TABLE IF NOT EXISTS idempotency_keys (
    method TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_json TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    PRIMARY KEY (method, idempotency_key)
);

CREATE TABLE IF NOT EXISTS orchestration_policy (
    policy_id SMALLINT PRIMARY KEY,
    kill_switch BOOLEAN NOT NULL DEFAULT FALSE,
    kill_switch_reason TEXT NOT NULL DEFAULT '',
    max_cost_per_run_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_attempts_per_run BIGINT NOT NULL DEFAULT 0,
    max_tokens_per_run BIGINT NOT NULL DEFAULT 0,
    max_latency_per_attempt_ms BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO orchestration_policy (
    policy_id,
    kill_switch,
    kill_switch_reason,
    max_cost_per_run_usd,
    max_attempts_per_run,
    max_tokens_per_run,
    max_latency_per_attempt_ms,
    updated_at
) VALUES (1, FALSE, '', 0, 0, 0, 0, NOW())
ON CONFLICT (policy_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS policy_caps (
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
);

ALTER TABLE policy_caps
ADD COLUMN IF NOT EXISTS dry_run BOOLEAN NOT NULL DEFAULT FALSE;

SELECT create_hypertable('benchmarks', 'created_at', if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('prompt_attempts', 'created_at', if_not_exists => TRUE, migrate_data => TRUE);
SELECT create_hypertable('run_events', 'created_at', if_not_exists => TRUE, migrate_data => TRUE);

CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks (updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_changelog_created_at ON changelog (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_benchmarks_created_at ON benchmarks (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_benchmarks_workflow_created_at ON benchmarks (workflow, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_started_at ON agent_runs (started_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_status_started_at ON agent_runs (status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_attempts_run_created_at ON prompt_attempts (run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_attempts_outcome_created_at ON prompt_attempts (outcome, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_run_events_run_created_at ON run_events (run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_api_keys_agent_id ON agent_api_keys (agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_api_keys_active ON agent_api_keys (is_active, revoked_at, expires_at);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at ON idempotency_keys (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_caps_lookup ON policy_caps (provider_type, provider, model, is_active, priority DESC);
