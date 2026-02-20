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
- loads file-backed state
- exposes gRPC service
- registers health + reflection
- handles graceful shutdown

2. `internal/service`
- validation and domain rules
- deterministic state mutations
- summary aggregations for budget analytics

3. `internal/store`
- in-process mutex protection
- atomic file persistence (`.tmp` + rename)
- durable JSON state

4. `internal/transport/grpc`
- manual service registration
- unary interceptors:
  - panic recovery
  - auth guard for write RPCs
  - logging
  - domain-error mapping

## Evolution Path
1. Replace file store with PostgreSQL.
2. Move Struct payloads to typed protobuf messages.
3. Add mTLS and per-client auth scopes.
4. Add stream RPCs for live orchestration telemetry.
