# grpcurl Cookbook

Assumes server is running on `localhost:50051`.

## List Services
```bash
grpcurl -plaintext localhost:50051 list
```

## List Methods
```bash
grpcurl -plaintext localhost:50051 list modeloman.v1.ModeloManHub
```

## Health
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/GetHealth
```

## Create Task
```bash
grpcurl -plaintext -d '{"title":"Define provider fallback chain","status":"todo","tags":["routing","policy"]}' \
  localhost:50051 modeloman.v1.ModeloManHub/CreateTask
```

## List Tasks
```bash
grpcurl -plaintext -d '{}' localhost:50051 modeloman.v1.ModeloManHub/ListTasks
```

## Record Benchmark
```bash
grpcurl -plaintext -d '{"workflow":"draft-generation","provider_type":"api","provider":"openai","model":"gpt-5-mini","tokens_in":1200,"tokens_out":300,"cost_usd":0.08,"latency_ms":900}' \
  localhost:50051 modeloman.v1.ModeloManHub/RecordBenchmark
```

## Authenticated Write Example
```bash
grpcurl -plaintext -H "x-modeloman-token: your-token" \
  -d '{"summary":"Enabled auth token for write methods","category":"policy"}' \
  localhost:50051 modeloman.v1.ModeloManHub/AppendChangelog
```
