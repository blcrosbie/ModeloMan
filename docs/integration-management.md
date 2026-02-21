# Integration Management

This backend is gRPC-only. External systems should integrate through RPC clients or adapters, not HTTP REST.

## Internal Agent/Subagent Integration
Use direct gRPC calls against `modeloman.v1.ModeloManHub`.

Recommended pattern:
1. Shared client wrapper in each agent runtime.
2. Write calls only from orchestrator-authorized paths.
3. Read calls available for context retrieval.

## MCP Layer
Keep MCP as a separate process that calls ModeloMan via gRPC.

Pattern:
1. MCP tool receives LLM request.
2. MCP tool maps request to gRPC method.
3. gRPC response is transformed to tool output.

This keeps LLM context tooling decoupled from control-plane state.

## LiteLLM / Routing Orchestrator
Use orchestration middleware to:
1. choose provider/model policy
2. execute completion
3. emit benchmark writes to ModeloMan through `RecordBenchmark`
4. track execution lifecycle with:
   - `StartRun` (once per orchestration run)
   - `RecordPromptAttempt` (once per model attempt/retry)
   - `RecordRunEvent` (state transitions/errors/tool signals)
   - `FinishRun` (terminal state + aggregate finalize)
5. enforce central guardrails by reading/updating policy:
   - `GetPolicy`
   - `SetPolicy` (kill switch + budget ceilings)
   - `UpsertPolicyCap` for provider/model-specific overrides

Recommended payload fields:
- `workflow`
- `provider_type`
- `provider`
- `model`
- `tokens_in`
- `tokens_out`
- `cost_usd`
- `latency_ms`

## n8n / Signal Pipelines
n8n can remain a visual workflow engine while ModeloMan remains source of truth.

Example pipelines:
1. RSS + model-drop scanner -> `CreateNote`
2. policy decision workflow -> `AppendChangelog`
3. periodic budget report -> `GetSummary` + alerting
4. retry/failure trend report -> `GetTelemetrySummary` + `ListPromptAttempts`
5. prompt ranking report -> `GetLeaderboard`

## gRPC CLI Reference
Local helper client:
- `go run ./cmd/modeloman-cli health`
- `go run ./cmd/modeloman-cli summary`
- `go run ./cmd/modeloman-cli telemetry-summary`
- `go run ./cmd/modeloman-cli create-task --title "..."`

For third-party clients, use reflection-enabled tools like `grpcurl`.
