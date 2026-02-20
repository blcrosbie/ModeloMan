# ModeloMan

Barebones orchestration control hub for:
- Prompting best practices
- Skill building/review
- Task + note tracking
- Changelog capture
- Model benchmark/cost logging

This repo is intentionally lightweight so it can evolve into:
- A public showcase repo
- A CLI/agent-connected control plane
- A deploy-anywhere Docker service (local + server)

## V2 Included (RPC + MCP Upgrade)
- Shared hub logic in `src/hub-service.js` (single source of truth)
- Internal gRPC transport for subagent communication:
  - schema: `proto/modeloman.proto`
  - server: `src/rpc-transport.js`
  - client helper: `src/rpc-client.js`
- MCP bridge server for LLM tool exposure:
  - `src/mcp-bridge.js`
- REST API kept for UI/public compatibility
- Ingestion endpoint for orchestrators (`n8n`, `LiteLLM`):
  - `POST /api/ingest/events`
- JSON data persistence mounted at `./data/modeloman-db.json`
- Dockerfile + docker-compose for portable deployment

## Quick Start (Local)
1. Install Node 20+
2. Install dependencies:
   ```bash
   npm install
   ```
3. Run:
   ```bash
   npm start
   ```
4. Open `http://localhost:3000`

Optional MCP bridge (run in a separate terminal):
```bash
npm run start:mcp
```

Example MCP client config is in `mcp/modeloman.mcp.json`.

## Quick Start (Docker)
```bash
docker compose up --build
```

Then open `http://localhost:3000`.

## API Surface
- `GET /api/health`
- `GET /api/summary`
- `GET /api/export`
- `POST /api/ingest/events`
- `GET|POST /api/tasks`
- `PATCH|DELETE /api/tasks/:id`
- `GET|POST /api/notes`
- `DELETE /api/notes/:id`
- `GET|POST /api/changelog`
- `DELETE /api/changelog/:id`
- `GET|POST /api/benchmarks`
- `DELETE /api/benchmarks/:id`

## Data Model (high level)
- `tasks`: execution-oriented work items with status and labels
- `notes`: freeform learnings, findings, and references
- `changelog`: meaningful updates and operational decisions
- `benchmarks`: task-level model usage snapshots
  - provider type: `api`, `subscription`, `opensource`
  - token in/out, cost, latency, model/provider metadata

## Environment Variables
- `PORT` (default: `3000`)
- `RPC_PORT` (default: `50051`)
- `RPC_TARGET` (for MCP/client, default: `127.0.0.1:50051`)
- `DATA_FILE` (default: `./data/modeloman-db.json`)
- `INGEST_KEY` (optional, recommended for orchestrator ingestion)

## RPC Surface (Internal)
- `Health`
- `Summary`
- `ExportState`
- `ListCollection`
- `CreateCollectionItem`
- `UpdateTask`
- `DeleteCollectionItem`

See `docs/rpc-implementation-map.md` for the full architecture map and rollout.

## Suggested Next Steps
1. Move persistent storage to PostgreSQL for concurrent multi-agent writes.
2. Add TLS and service auth for gRPC transport in production.
3. Add LiteLLM routing policy snapshots into `changelog`.
4. Add n8n workflows for RSS/HF/GitHub signal ingestion.
