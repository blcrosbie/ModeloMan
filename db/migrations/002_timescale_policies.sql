-- Retention/compression policy baseline for telemetry hypertables.
-- Tune windows per your compliance and analytics requirements.

ALTER TABLE benchmarks
SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'workflow,provider_type,provider,model'
);

ALTER TABLE prompt_attempts
SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'run_id,workflow,agent_id,model,outcome'
);

ALTER TABLE run_events
SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'run_id,event_type,level'
);

SELECT add_compression_policy('benchmarks', INTERVAL '14 days', if_not_exists => TRUE);
SELECT add_compression_policy('prompt_attempts', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_compression_policy('run_events', INTERVAL '7 days', if_not_exists => TRUE);

SELECT add_retention_policy('benchmarks', INTERVAL '180 days', if_not_exists => TRUE);
SELECT add_retention_policy('prompt_attempts', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('run_events', INTERVAL '90 days', if_not_exists => TRUE);
