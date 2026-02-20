import grpc from "@grpc/grpc-js";
import protoLoader from "@grpc/proto-loader";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

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

export function createRpcClient(target = process.env.RPC_TARGET ?? "127.0.0.1:50051") {
  const client = new modelomanProto.ModelomanTransport(
    target,
    grpc.credentials.createInsecure()
  );

  return {
    health: () => unary(client, "Health", {}),
    summary: async () => parseObject((await unary(client, "Summary", {})).json_payload),
    exportState: async () => parseObject((await unary(client, "ExportState", {})).json_payload),
    listCollection: async (collection) => {
      const response = await unary(client, "ListCollection", { collection });
      return response.json_items.map((item) => parseObject(item));
    },
    createCollectionItem: async (collection, payload) => {
      const response = await unary(client, "CreateCollectionItem", {
        collection,
        json_payload: JSON.stringify(payload ?? {}),
      });
      return parseObject(response.json_payload);
    },
    updateTask: async (id, payload) => {
      const response = await unary(client, "UpdateTask", {
        id,
        json_payload: JSON.stringify(payload ?? {}),
      });
      return parseObject(response.json_payload);
    },
    deleteCollectionItem: (collection, id) =>
      unary(client, "DeleteCollectionItem", { collection, id }),
    close: () => client.close(),
  };
}

function unary(client, method, request) {
  return new Promise((resolve, reject) => {
    client[method](request, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(response);
    });
  });
}

function parseObject(raw) {
  if (!raw) {
    return {};
  }
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}
