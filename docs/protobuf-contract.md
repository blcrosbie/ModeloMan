# Protobuf Contract

Source: `proto/modeloman/v1/hub.proto`

Service: `modeloman.v1.ModeloManHub`

## Why Struct/ListValue
Current phase uses `google.protobuf.Struct` and `google.protobuf.ListValue` so the project remains fully gRPC/protobuf without requiring local `protoc` during bootstrap.

This is a transitional contract strategy. Once `buf/protoc` is available, replace each Struct payload with typed messages while preserving method names.

## Payload Schemas

`CreateTask` request:
```json
{
  "title": "string (required)",
  "details": "string (optional)",
  "status": "todo|in_progress|done|blocked (optional, default todo)",
  "tags": ["string", "..."]
}
```

`UpdateTask` request:
```json
{
  "id": "string (required)",
  "title": "string (optional)",
  "details": "string (optional)",
  "status": "todo|in_progress|done|blocked (optional)",
  "tags": ["string", "..."] 
}
```

`DeleteTask` request:
```json
{
  "id": "string (required)"
}
```

`CreateNote` request:
```json
{
  "title": "string (required)",
  "body": "string (optional)",
  "tags": ["string", "..."]
}
```

`AppendChangelog` request:
```json
{
  "category": "platform|policy|model|infra|ops (optional, default ops)",
  "summary": "string (required)",
  "details": "string (optional)",
  "actor": "string (optional)"
}
```

`RecordBenchmark` request:
```json
{
  "workflow": "string (required)",
  "provider_type": "api|subscription|opensource (required)",
  "provider": "string (optional)",
  "model": "string (required)",
  "tokens_in": "int64 (optional, default 0)",
  "tokens_out": "int64 (optional, default 0)",
  "cost_usd": "float64 (optional, default 0)",
  "latency_ms": "int64 (optional, default 0)",
  "quality_score": "float64 (optional, default 0)",
  "notes": "string (optional)"
}
```

Response objects use normalized domain JSON:
- tasks: `id,title,details,status,tags,created_at,updated_at`
- notes: `id,title,body,tags,created_at`
- changelog: `id,category,summary,details,actor,created_at`
- benchmarks: `id,workflow,provider_type,provider,model,tokens_in,tokens_out,cost_usd,latency_ms,quality_score,notes,created_at`

## Backward-Compatible Upgrade Plan
1. Introduce typed messages alongside Struct methods.
2. Publish deprecation window for Struct methods.
3. Migrate clients and orchestrators.
4. Remove Struct methods after adoption threshold.
