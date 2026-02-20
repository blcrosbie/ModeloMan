import { randomUUID } from "node:crypto";

const TASK_STATUSES = new Set(["todo", "in_progress", "done"]);
const PROVIDER_TYPES = new Set(["api", "subscription", "opensource"]);
const COLLECTIONS = new Set(["tasks", "notes", "changelog", "benchmarks"]);

export class HubError extends Error {
  constructor(status, code, message) {
    super(message);
    this.name = "HubError";
    this.status = status;
    this.code = code;
  }
}

export function createHubService(store, dataFile) {
  return {
    health() {
      return {
        status: "ok",
        dataFile,
        now: new Date().toISOString(),
      };
    },

    exportState() {
      return store.snapshot();
    },

    summary() {
      const data = store.snapshot();
      const totals = {
        tokensIn: 0,
        tokensOut: 0,
        costUsd: 0,
        byProviderType: {
          api: { count: 0, costUsd: 0 },
          subscription: { count: 0, costUsd: 0 },
          opensource: { count: 0, costUsd: 0 },
        },
      };

      for (const item of data.benchmarks) {
        totals.tokensIn += numberValue(item.tokensIn);
        totals.tokensOut += numberValue(item.tokensOut);
        totals.costUsd += numberValue(item.costUsd);

        if (totals.byProviderType[item.providerType]) {
          totals.byProviderType[item.providerType].count += 1;
          totals.byProviderType[item.providerType].costUsd += numberValue(item.costUsd);
        }
      }

      return {
        counts: {
          tasks: data.tasks.length,
          notes: data.notes.length,
          changelog: data.changelog.length,
          benchmarks: data.benchmarks.length,
        },
        totals,
      };
    },

    listCollection(collection) {
      validateCollection(collection);
      if (collection === "tasks") {
        return sortByDate(store.list("tasks"), "updatedAt");
      }
      return sortByDate(store.list(collection), "createdAt");
    },

    async createCollectionItem(collection, payload = {}) {
      validateCollection(collection);
      const now = new Date().toISOString();

      if (collection === "tasks") {
        const title = textValue(payload.title);
        if (!title) {
          throw new HubError(400, "invalid_input", "title is required");
        }

        const status = payload.status ?? "todo";
        if (!TASK_STATUSES.has(status)) {
          throw new HubError(400, "invalid_input", "status must be todo, in_progress, or done");
        }

        const task = {
          id: randomUUID(),
          title,
          details: textValue(payload.details),
          status,
          labels: stringArray(payload.labels),
          createdAt: now,
          updatedAt: now,
        };
        await store.insert("tasks", task);
        return task;
      }

      if (collection === "notes") {
        const title = textValue(payload.title);
        if (!title) {
          throw new HubError(400, "invalid_input", "title is required");
        }

        const note = {
          id: randomUUID(),
          title,
          body: textValue(payload.body),
          tags: stringArray(payload.tags),
          createdAt: now,
        };
        await store.insert("notes", note);
        return note;
      }

      if (collection === "changelog") {
        const summary = textValue(payload.summary);
        if (!summary) {
          throw new HubError(400, "invalid_input", "summary is required");
        }

        const entry = {
          id: randomUUID(),
          summary,
          details: textValue(payload.details),
          source: textValue(payload.source),
          createdAt: now,
        };
        await store.insert("changelog", entry);
        return entry;
      }

      const taskType = textValue(payload.taskType);
      const providerType = textValue(payload.providerType);
      const model = textValue(payload.model);
      if (!taskType || !providerType || !model) {
        throw new HubError(400, "invalid_input", "taskType, providerType, and model are required");
      }
      if (!PROVIDER_TYPES.has(providerType)) {
        throw new HubError(
          400,
          "invalid_input",
          "providerType must be api, subscription, or opensource"
        );
      }

      const benchmark = {
        id: randomUUID(),
        taskType,
        providerType,
        provider: textValue(payload.provider),
        model,
        tokensIn: numberValue(payload.tokensIn),
        tokensOut: numberValue(payload.tokensOut),
        costUsd: numberValue(payload.costUsd),
        latencyMs: numberValue(payload.latencyMs),
        notes: textValue(payload.notes),
        createdAt: now,
      };
      await store.insert("benchmarks", benchmark);
      return benchmark;
    },

    async updateTask(id, payload = {}) {
      const status = payload.status;
      if (status !== undefined && !TASK_STATUSES.has(status)) {
        throw new HubError(400, "invalid_input", "status must be todo, in_progress, or done");
      }

      const updated = await store.update("tasks", id, (task) => ({
        ...task,
        title: payload.title !== undefined ? textValue(payload.title) : task.title,
        details: payload.details !== undefined ? textValue(payload.details) : task.details,
        status: status ?? task.status,
        labels: payload.labels !== undefined ? stringArray(payload.labels) : task.labels,
        updatedAt: new Date().toISOString(),
      }));

      if (!updated) {
        throw new HubError(404, "not_found", "task not found");
      }
      return updated;
    },

    async deleteCollectionItem(collection, id) {
      validateCollection(collection);
      const removed = await store.remove(collection, id);
      if (!removed) {
        throw new HubError(404, "not_found", `${collection.slice(0, -1)} not found`);
      }
      return { ok: true };
    },

    async ingestEvent(payload = {}, expectedKey) {
      if (expectedKey && payload.ingestKey !== expectedKey) {
        throw new HubError(401, "unauthorized", "invalid ingest key");
      }

      const kind = textValue(payload.kind);
      const title = textValue(payload.title);
      const body = textValue(payload.body);
      if (!kind || !title) {
        throw new HubError(400, "invalid_input", "kind and title are required");
      }

      if (kind === "note") {
        return this.createCollectionItem("notes", {
          title,
          body,
          tags: stringArray(payload.tags),
        });
      }
      if (kind === "changelog") {
        return this.createCollectionItem("changelog", {
          summary: title,
          details: body,
          source: textValue(payload.source || "orchestrator"),
        });
      }
      if (kind === "benchmark") {
        return this.createCollectionItem("benchmarks", {
          taskType: textValue(payload.taskType),
          providerType: textValue(payload.providerType),
          provider: textValue(payload.provider),
          model: textValue(payload.model),
          tokensIn: numberValue(payload.tokensIn),
          tokensOut: numberValue(payload.tokensOut),
          costUsd: numberValue(payload.costUsd),
          latencyMs: numberValue(payload.latencyMs),
          notes: body,
        });
      }
      throw new HubError(400, "invalid_input", "kind must be note, changelog, or benchmark");
    },
  };
}

function validateCollection(collection) {
  if (!COLLECTIONS.has(collection)) {
    throw new HubError(400, "invalid_input", `unknown collection: ${collection}`);
  }
}

function numberValue(value) {
  if (value === null || value === undefined || value === "") {
    return 0;
  }
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function textValue(value) {
  if (typeof value !== "string") {
    return "";
  }
  return value.trim();
}

function stringArray(value) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => (typeof item === "string" ? item.trim() : ""))
    .filter((item) => item.length > 0);
}

function sortByDate(items, key) {
  return [...items].sort((a, b) => Date.parse(b[key] ?? "") - Date.parse(a[key] ?? ""));
}
