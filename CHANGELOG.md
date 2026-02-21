# Changelog

All notable changes to this repository are documented in this file.

## 2026-02-18

### Added
- Full repository reset from Node/REST prototype to Go-only, gRPC-only backend.
- gRPC service `modeloman.v1.ModeloManHub` with read/write orchestration methods.
- Domain-layer validation and normalized error model.
- Unary interceptors for panic recovery, auth, logging, and gRPC status mapping.
- File-backed atomic JSON state store for tasks, notes, changelog, and benchmarks.
- CLI client for local integration testing over gRPC.
- Protobuf contract and architecture/error/integration documentation set.
- Docker and compose runtime for gRPC deployment.
