const endpoints = {
  health: "/api/health",
  summary: "/api/summary",
  tasks: "/api/tasks",
  notes: "/api/notes",
  changelog: "/api/changelog",
  benchmarks: "/api/benchmarks",
};

const healthEl = document.querySelector("#health");
const summaryEl = document.querySelector("#summary");
const template = document.querySelector("#list-item-template");

const resources = [
  {
    key: "tasks",
    form: document.querySelector("#task-form"),
    list: document.querySelector("#task-list"),
    format(item) {
      const labels = (item.labels || []).join(", ");
      return `${item.title}\n${item.details || ""}\nstatus: ${item.status}\n${labels ? `labels: ${labels}` : ""}\n${time(item.updatedAt)}`;
    },
    payload(formData) {
      return {
        title: formData.get("title"),
        details: formData.get("details"),
        status: formData.get("status"),
        labels: split(formData.get("labels")),
      };
    },
  },
  {
    key: "notes",
    form: document.querySelector("#note-form"),
    list: document.querySelector("#note-list"),
    format(item) {
      const tags = (item.tags || []).join(", ");
      return `${item.title}\n${item.body || ""}\n${tags ? `tags: ${tags}` : ""}\n${time(item.createdAt)}`;
    },
    payload(formData) {
      return {
        title: formData.get("title"),
        body: formData.get("body"),
        tags: split(formData.get("tags")),
      };
    },
  },
  {
    key: "changelog",
    form: document.querySelector("#changelog-form"),
    list: document.querySelector("#changelog-list"),
    format(item) {
      return `${item.summary}\n${item.details || ""}\nsource: ${item.source || "n/a"}\n${time(item.createdAt)}`;
    },
    payload(formData) {
      return {
        summary: formData.get("summary"),
        details: formData.get("details"),
        source: formData.get("source"),
      };
    },
  },
  {
    key: "benchmarks",
    form: document.querySelector("#benchmark-form"),
    list: document.querySelector("#benchmark-list"),
    format(item) {
      return `${item.taskType} | ${item.providerType} | ${item.model}\nprovider: ${item.provider || "n/a"}\ntokens: ${item.tokensIn} in / ${item.tokensOut} out\ncost: $${Number(item.costUsd || 0).toFixed(4)} | latency: ${item.latencyMs} ms\n${item.notes || ""}\n${time(item.createdAt)}`;
    },
    payload(formData) {
      return {
        taskType: formData.get("taskType"),
        providerType: formData.get("providerType"),
        provider: formData.get("provider"),
        model: formData.get("model"),
        tokensIn: numeric(formData.get("tokensIn")),
        tokensOut: numeric(formData.get("tokensOut")),
        costUsd: numeric(formData.get("costUsd")),
        latencyMs: numeric(formData.get("latencyMs")),
        notes: formData.get("notes"),
      };
    },
  },
];

await refreshAll();

for (const resource of resources) {
  resource.form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const formData = new FormData(resource.form);
    await request(endpoints[resource.key], {
      method: "POST",
      body: JSON.stringify(resource.payload(formData)),
    });
    resource.form.reset();
    await refreshAll();
  });
}

async function refreshAll() {
  const [health, summary] = await Promise.all([
    request(endpoints.health),
    request(endpoints.summary),
  ]);
  healthEl.textContent = `service: ${health.status} | ${new Date(health.now).toLocaleString()}`;
  renderSummary(summary);

  await Promise.all(resources.map(renderResource));
}

async function renderResource(resource) {
  const items = await request(endpoints[resource.key]);
  resource.list.textContent = "";

  for (const item of items) {
    const node = template.content.firstElementChild.cloneNode(true);
    node.querySelector(".item-main").textContent = resource.format(item);
    node.querySelector("button").addEventListener("click", async () => {
      await request(`${endpoints[resource.key]}/${item.id}`, { method: "DELETE" });
      await refreshAll();
    });
    resource.list.append(node);
  }
}

function renderSummary(summary) {
  const cards = [
    ["Tasks", summary.counts.tasks],
    ["Notes", summary.counts.notes],
    ["Changelog", summary.counts.changelog],
    ["Benchmarks", summary.counts.benchmarks],
    ["Total Tokens In", summary.totals.tokensIn],
    ["Total Tokens Out", summary.totals.tokensOut],
    ["Total API Cost (USD)", Number(summary.totals.costUsd).toFixed(4)],
    ["API Runs", summary.totals.byProviderType.api.count],
    ["Subscription Runs", summary.totals.byProviderType.subscription.count],
    ["Open Source Runs", summary.totals.byProviderType.opensource.count],
  ];

  summaryEl.textContent = "";
  for (const [label, value] of cards) {
    const card = document.createElement("div");
    card.className = "summary-tile";
    card.innerHTML = `<div class="summary-label">${label}</div><div class="summary-value">${value}</div>`;
    summaryEl.append(card);
  }
}

async function request(url, options = {}) {
  const response = await fetch(url, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });

  if (!response.ok) {
    const error = await safeJson(response);
    throw new Error(error.error || `Request failed: ${response.status}`);
  }
  return safeJson(response);
}

async function safeJson(response) {
  try {
    return await response.json();
  } catch {
    return {};
  }
}

function split(value) {
  return `${value || ""}`
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function numeric(value) {
  if (value === null || value === undefined || value === "") {
    return 0;
  }
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function time(value) {
  return value ? new Date(value).toLocaleString() : "";
}
