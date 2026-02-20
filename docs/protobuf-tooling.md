# Protobuf Tooling

This repo is contract-first and already ships a running gRPC server without generated stubs.

For typed client/server stubs, install tooling:
- `buf`
- `protoc-gen-go`
- `protoc-gen-go-grpc`

Then run:
```bash
buf lint
buf generate
```

Outputs are configured in `buf.gen.yaml` under `gen/go`.

## Why this is optional today
The runtime currently uses protobuf well-known types (`Struct`, `ListValue`, `Empty`) and manual service registration, so local development is not blocked by missing `protoc`/`buf`.

Typed stubs are the next step for stricter schema evolution.
