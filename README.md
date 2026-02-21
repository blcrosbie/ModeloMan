# ModeloMan

Go-first, gRPC-only orchestration control hub for:
- agent/subagent coordination metadata
- changelog and operations journaling
- benchmark telemetry (tokens, cost, latency, provider mix)
- documentation-driven protobuf contracts

## Stack
- Language: Go (`go1.25+`)
- Transport: gRPC (`google.golang.org/grpc`)
- Serialization: Protobuf (`Struct`/`ListValue` contract phase)
- Persistence:
  - PostgreSQL (primary runtime path)
  - TimescaleDB hypertable for benchmark telemetry
  - file-backed JSON store fallback (`STORE_DRIVER=file`)
- Runtime: single binary (`cmd/modeloman-server`)

## Project Layout
- `cmd/modeloman-server`: production server entrypoint
- `cmd/modeloman-cli`: local integration CLI (gRPC client)
- `internal/service`: business logic, validation, domain workflows
- `internal/store`: PostgreSQL/Timescale + file-store persistence adapters
- `internal/transport/grpc`: server registration + interceptors
- `internal/rpccontract`: canonical full RPC method names
- `proto/modeloman/v1/hub.proto`: protobuf contract
- `docs/`: architecture, protobuf payload schemas, error handling, integration ops
- `CHANGELOG.md`: human-facing change ledger

## Quick Start
1. Start server:
```bash
go run ./cmd/modeloman-server
```
2. Run sample calls:
```bash
go run ./cmd/modeloman-cli summary
go run ./cmd/modeloman-cli create-task --title "Set provider routing policy"
go run ./cmd/modeloman-cli list-tasks
```

## Environment Variables
- `GRPC_ADDR` (default `:50051`)
- `HTTP_ADDR` (default `:8080`, serves leaderboard webpage + JSON APIs)
- `STORE_DRIVER` (`postgres` or `file`, default `file`)
- `DATABASE_URL` (required when `STORE_DRIVER=postgres`)
- `DATA_FILE` (used when `STORE_DRIVER=file`, default `./data/modeloman.db.json`)
- `BOOTSTRAP_AGENT_ID` (optional, default `orchestrator`; used with bootstrap key)
- `BOOTSTRAP_AGENT_KEY` (optional; if set and postgres is enabled, inserts a per-agent API key)
- `AUTH_TOKEN` (optional legacy fallback token for write RPCs)

## Auth Model
Write methods require authentication when either agent-key auth or `AUTH_TOKEN` is configured:
- preferred: per-agent API key stored in Postgres (`x-modeloman-token` or `authorization: Bearer ...`)
- fallback: shared `AUTH_TOKEN` (if configured)
- read methods remain unauthenticated

Per-agent API keys are stored in `agent_api_keys` with hashed secrets (`SHA-256`) and audit fields (`created_at`, `last_used_at`, `revoked_at`, `expires_at`).

Policy controls are two-layer:
- global policy (`GetPolicy`/`SetPolicy`) for baseline budget + kill switch
- provider/model cap rules (`ListPolicyCaps`/`UpsertPolicyCap`/`DeletePolicyCap`) for targeted overrides

Policy caps support `dry_run=true` to log cap violations into `run_events` without blocking attempts.

See `docs/agent-api-keys.md` for bootstrap, rotation, and revoke examples.

## RPC Surface
Service: `modeloman.v1.ModeloManHub`

Read:
- `GetHealth`
- `GetSummary`
- `GetTelemetrySummary`
- `GetPolicy`
- `ExportState`
- `ListTasks`
- `ListNotes`
- `ListChangelog`
- `ListBenchmarks`
- `ListRuns`
- `ListPromptAttempts`
- `ListRunEvents`
- `GetLeaderboard`
- `ListPolicyCaps`

Write:
- `CreateTask`
- `UpdateTask`
- `DeleteTask`
- `CreateNote`
- `AppendChangelog`
- `RecordBenchmark`
- `StartRun`
- `FinishRun`
- `RecordPromptAttempt`
- `RecordRunEvent`
- `SetPolicy`
- `UpsertPolicyCap`
- `DeletePolicyCap`

## Error Handling
- Domain errors are normalized to gRPC status codes in unary interceptor.
- Panic recovery interceptor converts panics to `Internal`.
- Logs include method, latency, and final gRPC status code.

See `docs/error-handling.md`.

## Documentation Map
- `docs/architecture.md`
- `docs/agent-api-keys.md`
- `docs/protobuf-contract.md`
- `docs/error-handling.md`
- `docs/integration-management.md`
- `docs/changelog-policy.md`
- `docs/protobuf-tooling.md`
- `docs/grpcurl-cookbook.md`

## Docker
Set bootstrap key before startup:
```bash
export BOOTSTRAP_AGENT_KEY="replace-with-a-strong-agent-key"
```

```bash
docker compose up --build
```

`docker-compose.yml` boots:
- `timescaledb` (`timescale/timescaledb`)
- `modeloman` gRPC server (port `50051`) + leaderboard web UI (port `8080`)

The default compose path runs with:
- `STORE_DRIVER=postgres`
- `DATABASE_URL=postgres://modeloman:modeloman@timescaledb:5432/modeloman?sslmode=disable`
- `BOOTSTRAP_AGENT_ID=orchestrator`
- `BOOTSTRAP_AGENT_KEY=<required>`

Open leaderboard UI:
```bash
http://localhost:8080
```

Example authenticated write:
```bash
grpcurl -plaintext -H "x-modeloman-token: ${BOOTSTRAP_AGENT_KEY}" \
  -d '{"summary":"agent key auth enabled","category":"security"}' \
  localhost:50051 modeloman.v1.ModeloManHub/AppendChangelog
```
