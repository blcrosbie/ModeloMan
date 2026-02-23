# ModeloMan Production Hardening Runbook

This file is the execution checklist for hardening ModeloMan as a security-sensitive orchestration control-plane.

It is tailored to intended usage:
- You run many semi-trusted/untrusted agents.
- You want strong budget and policy controls.
- You want rich telemetry and prompt-performance analytics.
- You expose a public marketing/UI site, but control-plane writes must be protected.

## 0. Target Operating Model (Intended Use)

Use this split in production:
- `modeloman.com` (public): leaderboard + marketing UI only.
- `grpc.modeloman.com` (restricted): gRPC control-plane for orchestrator/agents.
- Postgres/Timescale: private only, never internet-exposed.

Control-plane principle:
- Read-only marketing data can be public.
- Orchestration state, policy, and raw telemetry are private/admin-scoped.

## 1. Agent Loop Protocol

When running an autonomous upgrade loop, follow this exact cycle:
1. Pick the first unchecked P0 item.
2. Implement only that item.
3. Run validation commands for that item.
4. Mark checkbox complete and add date + short note.
5. Open/commit PR with item ID in title.
6. Move to next unchecked item.

Global validation after each item:
```bash
go test ./...
```

## 2. Current Strong Points (Do Not Regress)

- Per-agent API keys with hashed storage and revocation/expiry checks (`internal/store/postgres_store.go`).
- Write-method auth guard in interceptor chain (`internal/transport/grpc/interceptors.go`).
- Kill switch and budget/policy caps (including `dry_run`) in service logic (`internal/service/hub_service.go`).
- Timescale hypertables + indexes for telemetry (`internal/store/postgres_store.go`).
- Loopback port mapping in compose (`docker-compose.yml`).

## 3. P0 Immediate Hardening (Do First)

- [x] **P0-01 Disable gRPC reflection in production** *(2026-02-21: added `ENABLE_REFLECTION` default false; reflection registration now opt-in)*
  - Why: reflection leaks full RPC surface and method names to attackers.
  - Files: `cmd/modeloman-server/main.go`, `internal/config/config.go`
  - Change:
    - Add `ENABLE_REFLECTION` env flag, default `false`.
    - Only call `reflection.Register(server)` when enabled.
  - Verify:
    - `grpcurl grpc.modeloman.com:443 list` fails in prod.
    - Works only in local/dev when enabled.

- [x] **P0-02 Enforce loopback listener defaults** *(2026-02-21: default listeners set to `127.0.0.1` for gRPC/HTTP in config)*
  - Why: default `:50051`/`:8080` binds all interfaces if misconfigured.
  - Files: `internal/config/config.go`, deployment env
  - Change:
    - Default `GRPC_ADDR=127.0.0.1:50051`
    - Default `HTTP_ADDR=127.0.0.1:8080`
  - Verify:
    - `ss -lntp` (or `netstat`) shows listeners only on `127.0.0.1`.

- [x] **P0-03 Add gRPC message size and concurrency guards** *(2026-02-21: server now sets recv=1MiB, send=2MiB, max streams=256)*
  - Why: `google.protobuf.Struct` can be abused for payload amplification.
  - Files: `cmd/modeloman-server/main.go`
  - Change:
    - Add `grpc.MaxRecvMsgSize`, `grpc.MaxSendMsgSize`, `grpc.MaxConcurrentStreams`.
    - Suggested start: recv 1 MiB, send 2 MiB, streams 256.
  - Verify:
    - oversized request returns `ResourceExhausted`.

- [x] **P0-04 Require auth for sensitive read methods** *(2026-02-21: added `public_read`/`private_read` classes; auth now required for private reads and writes)*
  - Why: unauthenticated `ExportState` and broad list methods can leak internals.
  - Files: `internal/rpccontract/methods.go`, `internal/transport/grpc/interceptors.go`
  - Change:
    - Introduce method classes:
      - `public_read` (health + marketing-safe reads)
      - `private_read` (telemetry internals, export, policy caps)
      - `write`
    - Require auth for `private_read` and `write`.
  - Verify:
    - unauth call to `ExportState` returns `Unauthenticated`.

- [x] **P0-05 Add per-key scopes (RBAC-lite)** *(2026-02-21: `agent_api_keys.scopes` added and enforced via method-to-scope checks in interceptor)*
  - Why: any valid key currently has full write power (including policy mutation).
  - Files: `internal/store/postgres_store.go`, `internal/store/store.go`, `internal/transport/grpc/interceptors.go`
  - Change:
    - Add scope column (`TEXT[]` or `JSONB`) to `agent_api_keys`.
    - Return scopes in auth principal.
    - Enforce method-to-scope mapping (example: `policy:write`, `telemetry:write`, `tasks:write`, `admin:read`).
  - Verify:
    - non-admin key cannot call `SetPolicy` / `DeletePolicyCap`.

- [x] **P0-06 Disable legacy shared AUTH_TOKEN by default** *(2026-02-21: added `ALLOW_LEGACY_AUTH_TOKEN` default false; legacy token ignored unless enabled)*
  - Why: shared token is high-blast-radius and hard to rotate safely.
  - Files: `internal/transport/grpc/interceptors.go`, `README.md`
  - Change:
    - Add `ALLOW_LEGACY_AUTH_TOKEN=false` default.
    - Reject legacy token unless explicitly enabled.
  - Verify:
    - legacy token fails unless override enabled.

- [x] **P0-07 Add constant-time comparison for token checks** *(2026-02-21: legacy fallback now uses SHA-256 + `subtle.ConstantTimeCompare`)*
  - Why: direct string equality can leak timing signal over many trials.
  - Files: `internal/transport/grpc/interceptors.go`
  - Change:
    - Use `crypto/subtle.ConstantTimeCompare` for legacy fallback.
  - Verify:
    - unit tests pass; behavior unchanged functionally.

- [x] **P0-08 Add per-key/IP rate limiting interceptor** *(2026-02-21: token-bucket interceptor added with key-id buckets for authenticated calls and IP buckets for unauthenticated calls)*
  - Why: unauth/public reads + write endpoints are DoS and brute-force targets.
  - Files: `internal/transport/grpc/interceptors.go`, `cmd/modeloman-server/main.go`
  - Change:
    - Token bucket keyed by `key_id` (authenticated) and remote IP (unauth).
  - Verify:
    - burst above limit yields `ResourceExhausted`.

- [x] **P0-09 Harden Caddy gRPC vhost behavior** *(2026-02-22: added hardened `Caddyfile` with TLS1.2+/ALPN h2, gRPC matcher, and non-gRPC `404`; verification commands documented)*
  - Why: prevent protocol confusion and non-gRPC traffic abuse.
  - Files: infra `Caddyfile` (server repo)
  - Change:
    - Enforce TLS 1.2+, ALPN `h2`, gRPC matcher, reject non-gRPC with `404`.
  - Verify:
    - ALPN negotiation is `h2`.
    - plain HTTP to `grpc.modeloman.com` is denied.

## 4. P1 Near-Term Hardening (Next)

- [x] **P1-01 Add DB pool limits + timeouts** *(2026-02-22: configured pool sizing and connection lifetime/idle/ping timeouts in Postgres store)*
  - Files: `internal/store/postgres_store.go`
  - Change: configure `SetMaxOpenConns`, `SetMaxIdleConns`, lifetime/idle timers.

- [x] **P1-02 Stop runtime schema/extension management in app startup** *(2026-02-22: startup now verifies schema/extension presence only; DDL moved to `db/migrations/001_init.sql` + migration docs)*
  - Why: app role should not need DDL/superuser-level capabilities.
  - Files: `internal/store/postgres_store.go`, migration tooling docs
  - Change: move DDL and `CREATE EXTENSION` out to migration job.

- [x] **P1-03 Add Timescale retention + compression policies** *(2026-02-22: added migration `db/migrations/002_timescale_policies.sql` with compression + retention policies for `prompt_attempts`, `run_events`, `benchmarks`)*
  - Files: SQL migration(s)
  - Change:
    - retention for `prompt_attempts`, `run_events`, `benchmarks`.
    - compression for older chunks.

- [x] **P1-04 Add idempotency keys for write RPCs** *(2026-02-22: added `idempotency_key` contract support, `idempotency_keys` dedupe table/index, and write RPC idempotency interceptor with stored response replay)*
  - Why: retries/replays can double-write attempts and events.
  - Files: proto/contract + service + store
  - Change: add `idempotency_key` and dedupe table/index.

- [ ] **P1-05 Eliminate full scans on hot paths**
  - Why: `FinishRun` / `RecordRunEvent` use broad list patterns now.
  - Files: `internal/service/hub_service.go`, `internal/store/postgres_store.go`
  - Change: add targeted `GetRunByID` query path.

- [ ] **P1-06 Add immutable audit sink**
  - Files: logging pipeline config + app log adapters
  - Change: ship key auth events + policy changes to append-only external store.

## 5. P2 Structural Hardening (After Stabilization)

- [ ] **P2-01 Migrate critical Struct RPCs to typed protobuf messages**
  - Prioritize: `SetPolicy`, `UpsertPolicyCap`, `RecordPromptAttempt`, `RecordRunEvent`.

- [ ] **P2-02 Add replay-resistant request signing for high-risk writes**
  - Timestamp + nonce + HMAC over canonical request components.

- [ ] **P2-03 Add mTLS for private agent networks**
  - Especially for cross-server agent traffic.

- [ ] **P2-04 Add provider-side cost reconciliation**
  - Treat agent-reported `cost_usd/tokens` as provisional until reconciled.

## 6. Deployment Baseline (Compose + Caddy)

- [ ] DB credentials must come from secrets, not static defaults.
- [ ] `DATABASE_URL` should use TLS when DB is off-host.
- [ ] Set container runtime hardening:
  - non-root user
  - read-only root filesystem
  - `cap_drop: [ALL]`
  - `no-new-privileges:true`

## 7. Acceptance Criteria (MVP Secure State)

Your intended-use baseline is met when all are true:
- Reflection off in prod.
- Private read + write RPCs require auth.
- Per-key scopes enforced for policy/admin operations.
- Legacy shared token disabled by default.
- gRPC max message + rate limits active.
- Loopback-only local listeners validated.
- Caddy gRPC route enforces TLS/h2 and rejects non-gRPC.
- DB pool/timeouts configured.
- Timescale retention/compression policies active.

## 8. Suggested First 3 PRs

1. `PR-1`: P0-01, P0-02, P0-03, P0-07  
2. `PR-2`: P0-04, P0-05, P0-06  
3. `PR-3`: P0-08, P0-09 + deployment config hardening

---

## Progress Log

- [x] `2026-02-21` P0-01: reflection is disabled by default and enabled only via `ENABLE_REFLECTION=true`.
- [x] `2026-02-21` P0-02: `GRPC_ADDR`/`HTTP_ADDR` defaults now bind to loopback (`127.0.0.1`).
- [x] `2026-02-21` P0-03: gRPC message and concurrency limits were added (`MaxRecvMsgSize`, `MaxSendMsgSize`, `MaxConcurrentStreams`).
- [x] `2026-02-21` P0-04: method access classes added; private reads and writes now require authentication.
- [x] `2026-02-21` P0-05: per-key scopes were added (`tasks:write`, `telemetry:write`, `policy:write`, `admin:read`) and checked per RPC.
- [x] `2026-02-21` P0-06: shared `AUTH_TOKEN` fallback is disabled by default and must be explicitly enabled.
- [x] `2026-02-21` P0-07: legacy token checks now use constant-time comparison.
- [x] `2026-02-21` P0-08: unary rate limiting added (authenticated by `key_id`, unauthenticated by remote IP).
- [x] `2026-02-22` P0-09: added hardened `Caddyfile` and deployment verification checklist (`docs/caddy-hardening.md`).
- [x] `2026-02-22` P1-01: added Postgres pool limits and startup ping timeout guards.
- [x] `2026-02-22` P1-02: runtime DDL/extension creation removed from startup path; migration script/documentation added.
- [x] `2026-02-22` P1-03: Timescale compression and retention policies added via migration SQL.
- [x] `2026-02-22` P1-04: idempotency key dedupe added for all write RPCs to prevent retry/replay double-writes.
