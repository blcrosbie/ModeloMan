import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import grpc from "@grpc/grpc-js";
import protoLoader from "@grpc/proto-loader";
import { HubError } from "./hub-service.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PROTO_PATH = join(__dirname, "..", "proto", "modeloman.proto");

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const loaded = grpc.loadPackageDefinition(packageDefinition);
const modelomanProto = loaded.modeloman.v1;

export function startRpcTransport(hub, rpcPort) {
  const server = new grpc.Server();

  server.addService(modelomanProto.ModelomanTransport.service, {
    Health: (_, callback) => {
      callback(null, asHealthResponse(hub.health()));
    },
    Summary: (_, callback) => {
      callback(null, { json_payload: JSON.stringify(hub.summary()) });
    },
    ExportState: (_, callback) => {
      callback(null, { json_payload: JSON.stringify(hub.exportState()) });
    },
    ListCollection: (call, callback) => {
      safely(callback, () => {
        const items = hub.listCollection(normalizeCollection(call.request.collection));
        return { json_items: items.map((item) => JSON.stringify(item)) };
      });
    },
    CreateCollectionItem: (call, callback) => {
      safelyAsync(callback, async () => {
        const item = await hub.createCollectionItem(
          normalizeCollection(call.request.collection),
          parseJson(call.request.json_payload)
        );
        return { json_payload: JSON.stringify(item) };
      });
    },
    UpdateTask: (call, callback) => {
      safelyAsync(callback, async () => {
        const item = await hub.updateTask(call.request.id, parseJson(call.request.json_payload));
        return { json_payload: JSON.stringify(item) };
      });
    },
    DeleteCollectionItem: (call, callback) => {
      safelyAsync(callback, async () => {
        await hub.deleteCollectionItem(normalizeCollection(call.request.collection), call.request.id);
        return { ok: true };
      });
    },
  });

  const bindAddress = `0.0.0.0:${rpcPort}`;
  server.bindAsync(bindAddress, grpc.ServerCredentials.createInsecure(), (error) => {
    if (error) {
      throw error;
    }
    server.start();
    console.log(`ModeloMan RPC transport listening on ${bindAddress}`);
  });

  return server;
}

function asHealthResponse(health) {
  return {
    status: health.status,
    data_file: health.dataFile,
    now: health.now,
  };
}

function parseJson(raw) {
  if (!raw) {
    return {};
  }
  try {
    return JSON.parse(raw);
  } catch {
    throw new HubError(400, "invalid_input", "json_payload must be valid JSON");
  }
}

function normalizeCollection(value) {
  return String(value || "").trim().toLowerCase();
}

function safely(callback, fn) {
  try {
    const data = fn();
    callback(null, data);
  } catch (error) {
    callback(toGrpcError(error));
  }
}

async function safelyAsync(callback, fn) {
  try {
    const data = await fn();
    callback(null, data);
  } catch (error) {
    callback(toGrpcError(error));
  }
}

function toGrpcError(error) {
  if (error instanceof HubError) {
    return {
      code: mapHttpStatusToGrpc(error.status),
      message: `${error.code}: ${error.message}`,
    };
  }
  return {
    code: grpc.status.INTERNAL,
    message: "internal: unexpected error",
  };
}

function mapHttpStatusToGrpc(status) {
  if (status === 400) {
    return grpc.status.INVALID_ARGUMENT;
  }
  if (status === 401) {
    return grpc.status.UNAUTHENTICATED;
  }
  if (status === 404) {
    return grpc.status.NOT_FOUND;
  }
  return grpc.status.INTERNAL;
}
