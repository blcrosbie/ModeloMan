import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname } from "node:path";

const DEFAULT_STATE = Object.freeze({
  tasks: [],
  notes: [],
  changelog: [],
  benchmarks: [],
});

export class DataStore {
  constructor(dataFile) {
    this.dataFile = dataFile;
    this.state = null;
  }

  async init() {
    await mkdir(dirname(this.dataFile), { recursive: true });

    try {
      const raw = await readFile(this.dataFile, "utf8");
      this.state = this.#withDefaults(JSON.parse(raw));
    } catch (error) {
      this.state = this.#withDefaults({});
      await this.#persist();
    }
  }

  list(collection) {
    return [...this.#collection(collection)];
  }

  snapshot() {
    return JSON.parse(JSON.stringify(this.state));
  }

  async insert(collection, item) {
    const list = this.#collection(collection);
    list.push(item);
    await this.#persist();
    return item;
  }

  async update(collection, id, updater) {
    const list = this.#collection(collection);
    const index = list.findIndex((item) => item.id === id);
    if (index === -1) {
      return null;
    }

    const updated = updater(list[index]);
    list[index] = updated;
    await this.#persist();
    return updated;
  }

  async remove(collection, id) {
    const list = this.#collection(collection);
    const index = list.findIndex((item) => item.id === id);
    if (index === -1) {
      return false;
    }

    list.splice(index, 1);
    await this.#persist();
    return true;
  }

  #collection(collection) {
    if (!this.state || !Array.isArray(this.state[collection])) {
      throw new Error(`Unknown collection: ${collection}`);
    }

    return this.state[collection];
  }

  #withDefaults(parsed) {
    return {
      tasks: Array.isArray(parsed.tasks) ? parsed.tasks : [],
      notes: Array.isArray(parsed.notes) ? parsed.notes : [],
      changelog: Array.isArray(parsed.changelog) ? parsed.changelog : [],
      benchmarks: Array.isArray(parsed.benchmarks) ? parsed.benchmarks : [],
    };
  }

  async #persist() {
    await writeFile(this.dataFile, `${JSON.stringify(this.state, null, 2)}\n`, "utf8");
  }
}

export { DEFAULT_STATE };
