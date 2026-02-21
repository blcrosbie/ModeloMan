# Architecture

## Goal
ModeloMan acts as a control-plane backend for AI orchestration strategy, benchmarking, and change governance.

## Core Principles
1. gRPC-only transport.
2. Go-only runtime.
3. Contract-first protobuf with documented payload schema.
4. Explicit error semantics mapped to gRPC status codes.
5. Durable changelog discipline.

## Runtime Components
1. `cmd/modeloman-server`
- boots config
- loads pluggable state store
- exposes gRPC service + optional read-only HTTP leaderboard/dashboard
- registers health + reflection
- handles graceful shutdown

2. `internal/service`
- validation and domain rules
- deterministic state mutations
- summary aggregations for budget analytics

3. `internal/store`
- pluggable persistence adapters (`STORE_DRIVER`)
- PostgreSQL canonical store for tasks/notes/changelog
- TimescaleDB hypertable for benchmark time-series telemetry
- file-store fallback for local bootstrap

4. `internal/transport/grpc`
- manual service registration
- unary interceptors:
  - panic recovery
  - auth guard for write RPCs (per-agent API keys + optional legacy shared token)
  - logging
  - domain-error mapping

5. `internal/transport/http`
- read-only dashboard + leaderboard view for marketing/demo
- JSON endpoints for leaderboard and telemetry summary

## Evolution Path
1. Move Struct payloads to typed protobuf messages.
2. Add mTLS and per-client auth scopes.
3. Add stream RPCs for live orchestration telemetry.
