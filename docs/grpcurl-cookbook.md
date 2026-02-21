# grpcurl Cookbook

Assumes server is running on `localhost:50051`.

## List Services
```bash
grpcurl -plaintext localhost:50051 list
```

## List Methods
```bash
grpcurl -plaintext localhost:50051 list modeloman.v1.ModeloManHub
```

## Health
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/GetHealth
```

## Create Task
```bash
grpcurl -plaintext -d '{"title":"Define provider fallback chain","status":"todo","tags":["routing","policy"]}' \
  localhost:50051 modeloman.v1.ModeloManHub/CreateTask
```

## List Tasks
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/ListTasks
```

## List Runs (Filtered)
```bash
grpcurl -plaintext -d '{"workflow":"mvp-build","status":"failed","limit":25}' \
  localhost:50051 modeloman.v1.ModeloManHub/ListRuns
```

## Record Benchmark
```bash
grpcurl -plaintext -d '{"workflow":"draft-generation","provider_type":"api","provider":"openai","model":"gpt-5-mini","tokens_in":1200,"tokens_out":300,"cost_usd":0.08,"latency_ms":900}' \
  localhost:50051 modeloman.v1.ModeloManHub/RecordBenchmark
```

## Start Run
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"workflow":"mvp-build","agent_id":"planner-1","task_id":"task_abc","prompt_version":"v3","model_policy":"cheap-first","max_retries":4}' \
  localhost:50051 modeloman.v1.ModeloManHub/StartRun
```

## Record Prompt Attempt
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"run_id":"run_...","attempt_number":2,"workflow":"mvp-build","agent_id":"planner-1","provider_type":"api","provider":"openai","model":"gpt-5-mini","prompt_version":"v3","outcome":"failed","error_type":"validation","error_message":"missing field","tokens_in":1800,"tokens_out":400,"cost_usd":0.11,"latency_ms":1200}' \
  localhost:50051 modeloman.v1.ModeloManHub/RecordPromptAttempt
```

## Finish Run
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"run_id":"run_...","status":"failed","last_error":"max retries reached"}' \
  localhost:50051 modeloman.v1.ModeloManHub/FinishRun
```

## Telemetry Summary
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/GetTelemetrySummary
```

## List Attempts For One Run
```bash
grpcurl -plaintext -d '{"run_id":"run_..."}' \
  localhost:50051 modeloman.v1.ModeloManHub/ListPromptAttempts
```

## Get Policy
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/GetPolicy
```

## Set Policy (Kill Switch + Budget Guard)
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"kill_switch":false,"max_cost_per_run_usd":2.5,"max_attempts_per_run":8,"max_tokens_per_run":50000}' \
  localhost:50051 modeloman.v1.ModeloManHub/SetPolicy
```

## Prompt Leaderboard
```bash
grpcurl -plaintext -d '{"workflow":"mvp-build","window_days":14,"limit":20}' \
  localhost:50051 modeloman.v1.ModeloManHub/GetLeaderboard
```

## List Policy Caps
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/ListPolicyCaps
```

## Upsert Policy Cap (Expensive Model)
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"name":"expensive-model-tight-cap","provider_type":"api","provider":"openai","model":"gpt-5","max_cost_per_run_usd":6,"max_cost_per_attempt_usd":1.2,"max_attempts_per_run":6,"priority":50,"dry_run":true,"is_active":true}' \
  localhost:50051 modeloman.v1.ModeloManHub/UpsertPolicyCap
```

## Upsert Policy Cap (OSS Looser)
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"name":"oss-looser-cap","provider_type":"opensource","max_tokens_per_run":250000,"max_attempts_per_run":20,"priority":10,"is_active":true}' \
  localhost:50051 modeloman.v1.ModeloManHub/UpsertPolicyCap
```

## Authenticated Write Example (Agent API Key)
```bash
grpcurl -plaintext -H "x-modeloman-token: your-agent-key" \
  -d '{"summary":"Enabled agent API key auth for write methods","category":"policy"}' \
  localhost:50051 modeloman.v1.ModeloManHub/AppendChangelog
```

`authorization: Bearer your-agent-key` is also supported.
