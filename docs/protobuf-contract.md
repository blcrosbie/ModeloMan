# Protobuf Contract

Source: `proto/modeloman/v1/hub.proto`

Service: `modeloman.v1.ModeloManHub`

## Why Struct/ListValue
Current phase uses `google.protobuf.Struct` and `google.protobuf.ListValue` so the project remains fully gRPC/protobuf without requiring local `protoc` during bootstrap.

This is a transitional contract strategy. Once `buf/protoc` is available, replace each Struct payload with typed messages while preserving method names.

## Payload Schemas

All write RPC request payloads support:

```json
{
  "idempotency_key": "string (optional but recommended for retry-safe writes)"
}
```

Behavior:
- Reusing the same `idempotency_key` with the same write method and same payload returns the original response.
- Reusing the same key with a different payload returns a conflict error.

`CreateTask` request:
```json
{
  "title": "string (required)",
  "details": "string (optional)",
  "status": "todo|in_progress|done|blocked (optional, default todo)",
  "tags": ["string", "..."]
}
```

`UpdateTask` request:
```json
{
  "id": "string (required)",
  "title": "string (optional)",
  "details": "string (optional)",
  "status": "todo|in_progress|done|blocked (optional)",
  "tags": ["string", "..."] 
}
```

`DeleteTask` request:
```json
{
  "id": "string (required)"
}
```

`CreateNote` request:
```json
{
  "title": "string (required)",
  "body": "string (optional)",
  "tags": ["string", "..."]
}
```

`AppendChangelog` request:
```json
{
  "category": "platform|policy|model|infra|ops (optional, default ops)",
  "summary": "string (required)",
  "details": "string (optional)",
  "actor": "string (optional)"
}
```

`RecordBenchmark` request:
```json
{
  "workflow": "string (required)",
  "provider_type": "api|subscription|opensource (required)",
  "provider": "string (optional)",
  "model": "string (required)",
  "tokens_in": "int64 (optional, default 0)",
  "tokens_out": "int64 (optional, default 0)",
  "cost_usd": "float64 (optional, default 0)",
  "latency_ms": "int64 (optional, default 0)",
  "quality_score": "float64 (optional, default 0)",
  "notes": "string (optional)"
}
```

`StartRun` request:
```json
{
  "task_id": "string (optional)",
  "workflow": "string (required)",
  "agent_id": "string (required)",
  "prompt_version": "string (optional)",
  "model_policy": "string (optional)",
  "max_retries": "int64 (optional, default 0)"
}
```

`FinishRun` request:
```json
{
  "run_id": "string (required)",
  "status": "completed|failed|cancelled (optional, default completed)",
  "last_error": "string (optional)"
}
```

`RecordPromptAttempt` request:
```json
{
  "run_id": "string (required)",
  "attempt_number": "int64 (required, >=1)",
  "workflow": "string (optional)",
  "agent_id": "string (optional)",
  "provider_type": "string (optional, default api)",
  "provider": "string (optional)",
  "model": "string (required)",
  "prompt_version": "string (optional)",
  "prompt_hash": "string (optional)",
  "outcome": "success|failed|timeout|retryable_error|tool_error (required)",
  "error_type": "string (optional)",
  "error_message": "string (optional)",
  "tokens_in": "int64 (optional, default 0)",
  "tokens_out": "int64 (optional, default 0)",
  "cost_usd": "float64 (optional, default 0)",
  "latency_ms": "int64 (optional, default 0)",
  "quality_score": "float64 (optional, default 0)"
}
```

`RecordRunEvent` request:
```json
{
  "run_id": "string (required)",
  "event_type": "string (required)",
  "level": "info|warn|error (optional, default info)",
  "message": "string (optional)",
  "data_json": "string (optional; serialized JSON payload)"
}
```

`ListPromptAttempts` request:
```json
{
  "run_id": "string (optional filter)"
}
```

`ListRunEvents` request:
```json
{
  "run_id": "string (optional filter)",
  "event_type": "string (optional filter)",
  "level": "info|warn|error (optional filter)",
  "created_after": "RFC3339 timestamp (optional filter)",
  "created_before": "RFC3339 timestamp (optional filter)",
  "limit": "int64 (optional)"
}
```

`ListRuns` request:
```json
{
  "run_id": "string (optional filter)",
  "task_id": "string (optional filter)",
  "workflow": "string (optional filter)",
  "agent_id": "string (optional filter)",
  "status": "running|completed|failed|cancelled (optional filter)",
  "prompt_version": "string (optional filter)",
  "started_after": "RFC3339 timestamp (optional filter)",
  "started_before": "RFC3339 timestamp (optional filter)",
  "limit": "int64 (optional)"
}
```

`SetPolicy` request:
```json
{
  "kill_switch": "bool (optional)",
  "kill_switch_reason": "string (optional)",
  "max_cost_per_run_usd": "float64 (optional, 0=unlimited)",
  "max_attempts_per_run": "int64 (optional, 0=unlimited)",
  "max_tokens_per_run": "int64 (optional, 0=unlimited)",
  "max_latency_per_attempt_ms": "int64 (optional, 0=unlimited)"
}
```

`GetLeaderboard` request:
```json
{
  "workflow": "string (optional filter)",
  "model": "string (optional filter)",
  "prompt_version": "string (optional filter)",
  "window_days": "int64 (optional lookback)",
  "limit": "int64 (optional, default 20)"
}
```

`UpsertPolicyCap` request:
```json
{
  "id": "string (optional; generated if omitted)",
  "name": "string (optional)",
  "provider_type": "api|subscription|opensource (optional; empty matches any)",
  "provider": "string (optional; empty matches any)",
  "model": "string (optional; empty matches any)",
  "max_cost_per_run_usd": "float64 (optional, 0 means inherit global)",
  "max_attempts_per_run": "int64 (optional, 0 means inherit global)",
  "max_tokens_per_run": "int64 (optional, 0 means inherit global)",
  "max_cost_per_attempt_usd": "float64 (optional, 0 means unset)",
  "max_tokens_per_attempt": "int64 (optional, 0 means unset)",
  "max_latency_per_attempt_ms": "int64 (optional, 0 means inherit global)",
  "priority": "int64 (optional; higher wins on same specificity)",
  "dry_run": "bool (optional; true logs violations without blocking)",
  "is_active": "bool (optional; default true)"
}
```

`DeletePolicyCap` request:
```json
{
  "id": "string (required)"
}
```

Response objects use normalized domain JSON:
- tasks: `id,title,details,status,tags,created_at,updated_at`
- notes: `id,title,body,tags,created_at`
- changelog: `id,category,summary,details,actor,created_at`
- benchmarks: `id,workflow,provider_type,provider,model,tokens_in,tokens_out,cost_usd,latency_ms,quality_score,notes,created_at`
- runs: `id,task_id,workflow,agent_id,prompt_version,model_policy,status,max_retries,total_attempts,success_attempts,failed_attempts,total_tokens_in,total_tokens_out,total_cost_usd,duration_ms,last_error,started_at,finished_at`
- prompt attempts: `id,run_id,attempt_number,workflow,agent_id,provider_type,provider,model,prompt_version,prompt_hash,outcome,error_type,error_message,tokens_in,tokens_out,cost_usd,latency_ms,quality_score,created_at`
- run events: `id,run_id,event_type,level,message,data_json,created_at`
- telemetry summary: `counts,totals,averages`
- orchestration policy: `kill_switch,kill_switch_reason,max_cost_per_run_usd,max_attempts_per_run,max_tokens_per_run,max_latency_per_attempt_ms,updated_at`
- policy cap: `id,name,provider_type,provider,model,max_cost_per_run_usd,max_attempts_per_run,max_tokens_per_run,max_cost_per_attempt_usd,max_tokens_per_attempt,max_latency_per_attempt_ms,priority,dry_run,is_active,updated_at`
- leaderboard entry: `workflow,prompt_version,model,attempts,success_attempts,failed_attempts,success_rate,average_cost_usd,average_latency_ms,score`

## Backward-Compatible Upgrade Plan
1. Introduce typed messages alongside Struct methods.
2. Publish deprecation window for Struct methods.
3. Migrate clients and orchestrators.
4. Remove Struct methods after adoption threshold.
