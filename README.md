# ModeloMan

ModeloMan is a Go-first telemetry and control plane for AI coding runs.

The primary product is `mm`: a local wrapper around coding CLIs like Codex, Claude Code, Gemini CLI, and OpenCode. `mm` assembles repo context, records prompt attempts and run metadata, and sends that session data to the ModeloMan gRPC service for storage and later analysis.

The server in this repo exists to support that workflow:
- record runs, prompts, events, notes, and benchmarks
- enforce auth and policy over the logging surface
- expose gRPC contracts for wrapper clients and future analysis tooling

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
- `cmd/mm`: primary workflow wrapper for vendor coding CLIs + ModeloMan telemetry
- `cmd/modeloman`: alias entrypoint for the same wrapper
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
1. Start the server:
```bash
go run ./cmd/modeloman-server
```
2. Install the wrapper:
```bash
go install ./cmd/mm
# optional alias binary:
go install ./cmd/modeloman
```
3. Run the wrapper:
```bash
mm add internal/**/*.go cmd/mm/*.go
mm run codex -p "tighten the wrapper telemetry flow"
cat RESTART_PROMPT.md | mm run codex --add cmd/** --add internal/**
```
4. Run sample direct gRPC calls if needed:
```bash
go run ./cmd/modeloman-cli summary
go run ./cmd/modeloman-cli create-task --title "Set provider routing policy"
go run ./cmd/modeloman-cli list-tasks
```

### Wrapper CLI
Use `mm` as the canonical binary.

`modeloman` remains available as an alias for the same wrapper:
```bash
modeloman run codex --task bugfix -p "fix failing tests"
modeloman tui
```

## Environment Variables
- `GRPC_ADDR` (default `127.0.0.1:50051`)
- `HTTP_ADDR` (default `127.0.0.1:8080`, serves leaderboard webpage + JSON APIs)
- `STORE_DRIVER` (`postgres` or `file`, default `file`)
- `DATABASE_URL` (required when `STORE_DRIVER=postgres`)
- `DATA_FILE` (used when `STORE_DRIVER=file`, default `./data/modeloman.db.json`)
- `BOOTSTRAP_AGENT_ID` (optional, default `orchestrator`; used with bootstrap key)
- `BOOTSTRAP_AGENT_KEY` (optional; if set and postgres is enabled, inserts a per-agent API key)
- `ENABLE_REFLECTION` (default `false`; set `true` only in trusted dev/local environments)
- `AUTH_TOKEN` (optional legacy shared token; ignored unless legacy auth is explicitly enabled)
- `ALLOW_LEGACY_AUTH_TOKEN` (default `false`; must be `true` to allow `AUTH_TOKEN` fallback)

## Auth Model
`private_read` and `write` RPC methods require authentication.

Preferred auth:
- per-agent API key stored in Postgres (`x-modeloman-token` or `authorization: Bearer ...`)

Legacy fallback:
- shared `AUTH_TOKEN` is only accepted when both `AUTH_TOKEN` and `ALLOW_LEGACY_AUTH_TOKEN=true` are set.

Public read methods remain unauthenticated:
- `GetHealth`
- `GetTelemetrySummary`
- `GetLeaderboard`

Per-agent API keys are stored in `agent_api_keys` with hashed secrets (`SHA-256`) and audit fields (`created_at`, `last_used_at`, `revoked_at`, `expires_at`).
API keys also carry scopes (`tasks:write`, `telemetry:write`, `policy:write`, `admin:read`) enforced per RPC method.

Write RPCs support `idempotency_key` for retry-safe dedupe. Reusing the same key with the same method/payload returns the original response.

Policy controls are two-layer:
- global policy (`GetPolicy`/`SetPolicy`) for baseline budget + kill switch
- provider/model cap rules (`ListPolicyCaps`/`UpsertPolicyCap`/`DeletePolicyCap`) for targeted overrides

Policy caps support `dry_run=true` to log cap violations into `run_events` without blocking attempts.

See `docs/agent-api-keys.md` for bootstrap, rotation, and revoke examples.

## RPC Surface
Service: `modeloman.v1.ModeloManHub`

Public Read:
- `GetHealth`
- `GetTelemetrySummary`
- `GetLeaderboard`

Private Read (auth required):
- `GetSummary`
- `GetPolicy`
- `ExportState`
- `ListTasks`
- `ListNotes`
- `ListChangelog`
- `ListBenchmarks`
- `ListRuns`
- `ListPromptAttempts`
- `ListRunEvents`
- `ListPolicyCaps`

Write (auth + scope required):
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
- `docs/caddy-hardening.md`
- `docs/mm.md`
- `docs/postgres-migrations.md`
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
