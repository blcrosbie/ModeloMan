# ModeloMan

Go-first, gRPC-only orchestration control hub for:
- agent/subagent coordination metadata
- changelog and operations journaling
- benchmark telemetry (tokens, cost, latency, provider mix)
- documentation-driven protobuf contracts

This repo intentionally avoids Node/npm and REST.

## Stack
- Language: Go (`go1.25+`)
- Transport: gRPC (`google.golang.org/grpc`)
- Serialization: Protobuf (`Struct`/`ListValue` contract phase)
- Persistence: local JSON state store (`data/modeloman.db.json`)
- Runtime: single binary (`cmd/modeloman-server`)

## Project Layout
- `cmd/modeloman-server`: production server entrypoint
- `cmd/modeloman-cli`: local integration CLI (gRPC client)
- `internal/service`: business logic, validation, domain workflows
- `internal/store`: atomic JSON persistence layer
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
- `DATA_FILE` (default `./data/modeloman.db.json`)
- `AUTH_TOKEN` (optional; if set, write RPCs require token metadata)

## Auth Model
If `AUTH_TOKEN` is set:
- write methods require either:
  - metadata `x-modeloman-token: <token>`
  - or `authorization: Bearer <token>`
- read methods remain unauthenticated

## RPC Surface
Service: `modeloman.v1.ModeloManHub`

Read:
- `GetHealth`
- `GetSummary`
- `ExportState`
- `ListTasks`
- `ListNotes`
- `ListChangelog`
- `ListBenchmarks`

Write:
- `CreateTask`
- `UpdateTask`
- `DeleteTask`
- `CreateNote`
- `AppendChangelog`
- `RecordBenchmark`

## Error Handling
- Domain errors are normalized to gRPC status codes in unary interceptor.
- Panic recovery interceptor converts panics to `Internal`.
- Logs include method, latency, and final gRPC status code.

See `docs/error-handling.md`.

## Documentation Map
- `docs/architecture.md`
- `docs/protobuf-contract.md`
- `docs/error-handling.md`
- `docs/integration-management.md`
- `docs/changelog-policy.md`
- `docs/protobuf-tooling.md`
- `docs/grpcurl-cookbook.md`

## Docker
```bash
docker compose up --build
```

The service listens on gRPC port `50051`.
