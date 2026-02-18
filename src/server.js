import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import express from "express";
import { DataStore } from "./store.js";
import { createHubService, HubError } from "./hub-service.js";
import { startRpcTransport } from "./rpc-transport.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const app = express();

const PORT = Number.parseInt(process.env.PORT ?? "3000", 10);
const RPC_PORT = Number.parseInt(process.env.RPC_PORT ?? "50051", 10);
const DATA_FILE = process.env.DATA_FILE ?? join(process.cwd(), "data", "modeloman-db.json");
const INGEST_KEY = process.env.INGEST_KEY ?? "";

const store = new DataStore(DATA_FILE);
await store.init();
const hub = createHubService(store, DATA_FILE);
const rpcServer = startRpcTransport(hub, RPC_PORT);

app.use(express.json({ limit: "1mb" }));
app.use((_, response, next) => {
  response.setHeader("X-ModeloMan", "hub-v2");
  next();
});

app.get("/api/health", (_, response) => {
  response.json(hub.health());
});

app.get("/api/export", (_, response) => {
  response.json(hub.exportState());
});

app.get("/api/summary", (_, response) => {
  response.json(hub.summary());
});

app.get("/api/tasks", (_, response) => {
  response.json(hub.listCollection("tasks"));
});

app.post("/api/tasks", asyncRoute(async (request, response) => {
  const task = await hub.createCollectionItem("tasks", request.body ?? {});
  response.status(201).json(task);
}));

app.patch("/api/tasks/:id", asyncRoute(async (request, response) => {
  const updated = await hub.updateTask(request.params.id, request.body ?? {});
  response.json(updated);
}));

app.delete("/api/tasks/:id", asyncRoute(async (request, response) => {
  await hub.deleteCollectionItem("tasks", request.params.id);
  response.status(204).send();
}));

app.get("/api/notes", (_, response) => {
  response.json(hub.listCollection("notes"));
});

app.post("/api/notes", asyncRoute(async (request, response) => {
  const note = await hub.createCollectionItem("notes", request.body ?? {});
  response.status(201).json(note);
}));

app.delete("/api/notes/:id", asyncRoute(async (request, response) => {
  await hub.deleteCollectionItem("notes", request.params.id);
  response.status(204).send();
}));

app.get("/api/changelog", (_, response) => {
  response.json(hub.listCollection("changelog"));
});

app.post("/api/changelog", asyncRoute(async (request, response) => {
  const entry = await hub.createCollectionItem("changelog", request.body ?? {});
  response.status(201).json(entry);
}));

app.delete("/api/changelog/:id", asyncRoute(async (request, response) => {
  await hub.deleteCollectionItem("changelog", request.params.id);
  response.status(204).send();
}));

app.get("/api/benchmarks", (_, response) => {
  response.json(hub.listCollection("benchmarks"));
});

app.post("/api/benchmarks", asyncRoute(async (request, response) => {
  const benchmark = await hub.createCollectionItem("benchmarks", request.body ?? {});
  response.status(201).json(benchmark);
}));

app.delete("/api/benchmarks/:id", asyncRoute(async (request, response) => {
  await hub.deleteCollectionItem("benchmarks", request.params.id);
  response.status(204).send();
}));

app.post("/api/ingest/events", asyncRoute(async (request, response) => {
  const ingestKey = request.headers["x-modeloman-ingest-key"] || request.body?.ingestKey;
  const item = await hub.ingestEvent(
    { ...request.body, ingestKey: typeof ingestKey === "string" ? ingestKey : "" },
    INGEST_KEY
  );
  response.status(201).json(item);
}));

const publicDir = join(__dirname, "..", "public");
app.use(express.static(publicDir));

app.use("/api", (_, response) => {
  response.status(404).json({ error: "not found" });
});

app.get("*", (_, response) => {
  response.sendFile(join(publicDir, "index.html"));
});

app.listen(PORT, () => {
  console.log(`ModeloMan HTTP listening on http://localhost:${PORT}`);
});

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);

function shutdown() {
  try {
    rpcServer.forceShutdown();
  } finally {
    process.exit(0);
  }
}

function asyncRoute(handler) {
  return async (request, response) => {
    try {
      await handler(request, response);
    } catch (error) {
      handleError(error, response);
    }
  };
}

function handleError(error, response) {
  if (error instanceof HubError) {
    response.status(error.status).json({ error: error.message, code: error.code });
    return;
  }

  response.status(500).json({ error: "internal server error", code: "internal" });
}
