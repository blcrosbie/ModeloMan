# ModeloMan RPC Implementation Map

## Layered Stack
1. Transport Layer (RPC)
- Protocol: gRPC (`proto/modeloman.proto`)
- Purpose: subagent-to-subagent calls for high-throughput internal traffic
- Entry points: `src/rpc-transport.js`, `src/rpc-client.js`
- Port: `50051` (default, configurable via `RPC_PORT`)

2. Context Layer (MCP)
- Protocol: MCP over stdio
- Purpose: expose ModeloMan as tools to LLM runtimes
- Entry point: `src/mcp-bridge.js`
- Tools:
  - `modeloman_summary`
  - `modeloman_list_collection`
  - `modeloman_create_task`
  - `modeloman_add_note`
  - `modeloman_add_benchmark`

3. Orchestrator Layer (n8n/LiteLLM)
- Purpose: route model/provider calls and ingest external intelligence
- Ingestion endpoint: `POST /api/ingest/events`
- Auth: `INGEST_KEY` via `x-modeloman-ingest-key` header

## Data Flow
1. Subagent A sends internal call to gRPC transport.
2. gRPC service executes shared hub logic (`src/hub-service.js`).
3. State is persisted via `src/store.js`.
4. MCP bridge calls the same transport using `src/rpc-client.js`.
5. LLM assistants consume MCP tools for retrieval and write-back actions.
6. n8n/LiteLLM pipelines can ingest model/news/benchmark events through `/api/ingest/events`.

## Why This Split
- gRPC handles internal throughput and low-latency binary transport.
- MCP gives model-agnostic tool exposure for Claude/GPT/Gemini/open-source clients.
- Orchestrator stays policy-focused (key routing, budget logic, fallback chains) without owning business state.

## n8n Pipeline Starters
1. RSS + Model Drop Monitor
- Sources: OpenAI/Anthropic/Google blogs, HF trending, GitHub releases.
- Transform: classify event (`note` vs `changelog`), extract summary.
- Sink: `POST /api/ingest/events`.

2. Benchmark Collector
- Source: LiteLLM response metadata.
- Transform: map token/cost/latency to benchmark payload.
- Sink: `POST /api/ingest/events` with `kind=benchmark`.

3. Budget Watchdog
- Source: scheduled fetch from `/api/summary`.
- Rules: threshold alerts for daily/weekly token burn.
- Action: create `task` or `changelog` entry for policy updates.

## Example Ingest Payloads

```json
{
  "kind": "note",
  "title": "HF model drop: new 32B instruct variant",
  "body": "Strong coding eval gains, check quantized local run.",
  "tags": ["huggingface", "model-drop", "triage"]
}
```

```json
{
  "kind": "benchmark",
  "title": "weekly eval run",
  "body": "LiteLLM routed to gpt-5-mini for draft stage.",
  "taskType": "draft-generation",
  "providerType": "api",
  "provider": "openai",
  "model": "gpt-5-mini",
  "tokensIn": 12400,
  "tokensOut": 2600,
  "costUsd": 0.37,
  "latencyMs": 1880
}
```

## Rollout Plan
1. Adopt gRPC for internal subagent integrations.
2. Move LLM-facing tool calls to MCP bridge.
3. Add orchestrator policies:
- model routing matrix by task type
- token/cost ceilings by environment
- fallback chain: API -> subscription -> open-source local
4. Replace JSON file store with PostgreSQL once ingestion volume increases.
