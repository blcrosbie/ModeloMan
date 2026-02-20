import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { createRpcClient } from "./rpc-client.js";

const rpc = createRpcClient(process.env.RPC_TARGET ?? "127.0.0.1:50051");

const server = new McpServer({
  name: "modeloman-mcp",
  version: "0.2.0",
});

server.tool("modeloman_summary", "Returns aggregate orchestration metrics.", async () => {
  const summary = await rpc.summary();
  return {
    content: [{ type: "text", text: JSON.stringify(summary, null, 2) }],
  };
});

server.tool(
  "modeloman_list_collection",
  "Lists tasks, notes, changelog, or benchmarks from ModeloMan.",
  {
    collection: z.enum(["tasks", "notes", "changelog", "benchmarks"]),
  },
  async ({ collection }) => {
    const items = await rpc.listCollection(collection);
    return {
      content: [{ type: "text", text: JSON.stringify(items, null, 2) }],
    };
  }
);

server.tool(
  "modeloman_create_task",
  "Creates a new orchestration task in ModeloMan.",
  {
    title: z.string().min(1),
    details: z.string().optional(),
    status: z.enum(["todo", "in_progress", "done"]).optional(),
    labels: z.array(z.string()).optional(),
  },
  async ({ title, details, status, labels }) => {
    const item = await rpc.createCollectionItem("tasks", {
      title,
      details: details ?? "",
      status: status ?? "todo",
      labels: labels ?? [],
    });
    return {
      content: [{ type: "text", text: JSON.stringify(item, null, 2) }],
    };
  }
);

server.tool(
  "modeloman_add_note",
  "Adds a note from research, RSS scans, or model release findings.",
  {
    title: z.string().min(1),
    body: z.string().optional(),
    tags: z.array(z.string()).optional(),
  },
  async ({ title, body, tags }) => {
    const item = await rpc.createCollectionItem("notes", {
      title,
      body: body ?? "",
      tags: tags ?? [],
    });
    return {
      content: [{ type: "text", text: JSON.stringify(item, null, 2) }],
    };
  }
);

server.tool(
  "modeloman_add_benchmark",
  "Adds a benchmark datapoint for token, cost, and latency tracking.",
  {
    taskType: z.string().min(1),
    providerType: z.enum(["api", "subscription", "opensource"]),
    provider: z.string().optional(),
    model: z.string().min(1),
    tokensIn: z.number().nonnegative().optional(),
    tokensOut: z.number().nonnegative().optional(),
    costUsd: z.number().nonnegative().optional(),
    latencyMs: z.number().nonnegative().optional(),
    notes: z.string().optional(),
  },
  async ({ taskType, providerType, provider, model, tokensIn, tokensOut, costUsd, latencyMs, notes }) => {
    const item = await rpc.createCollectionItem("benchmarks", {
      taskType,
      providerType,
      provider: provider ?? "",
      model,
      tokensIn: tokensIn ?? 0,
      tokensOut: tokensOut ?? 0,
      costUsd: costUsd ?? 0,
      latencyMs: latencyMs ?? 0,
      notes: notes ?? "",
    });
    return {
      content: [{ type: "text", text: JSON.stringify(item, null, 2) }],
    };
  }
);

const transport = new StdioServerTransport();
await server.connect(transport);

process.on("SIGINT", () => {
  rpc.close();
  process.exit(0);
});

process.on("SIGTERM", () => {
  rpc.close();
  process.exit(0);
});
