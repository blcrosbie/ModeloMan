package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bcrosbie/modeloman/internal/service"
)

func NewServer(addr string, hub *service.HubService) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(leaderboardPageHTML))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/api/telemetry-summary", func(w http.ResponseWriter, _ *http.Request) {
		summary, err := hub.TelemetrySummary()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, summary)
	})
	mux.HandleFunc("/api/policy", func(w http.ResponseWriter, _ *http.Request) {
		policy, err := hub.GetPolicy()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, policy)
	})
	mux.HandleFunc("/api/policy-caps", func(w http.ResponseWriter, _ *http.Request) {
		items, err := hub.ListPolicyCaps()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("/api/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit := int64(20)
		if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be non-negative int64"})
				return
			}
			limit = parsed
		}
		windowDays := int64(0)
		if raw := strings.TrimSpace(query.Get("window_days")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "window_days must be non-negative int64"})
				return
			}
			windowDays = parsed
		}

		items, err := hub.Leaderboard(service.LeaderboardRequest{
			Workflow:      strings.TrimSpace(query.Get("workflow")),
			Model:         strings.TrimSpace(query.Get("model")),
			PromptVersion: strings.TrimSpace(query.Get("prompt_version")),
			WindowDays:    windowDays,
			Limit:         limit,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("http json encode error: %v", err)
	}
}

const leaderboardPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ModeloMan Leaderboard</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;600;700&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #08161f;
      --bg2: #102534;
      --card: rgba(12, 28, 39, 0.78);
      --line: #2a4b63;
      --text: #e5f4ff;
      --muted: #9bbacf;
      --accent: #54f2b2;
      --accent2: #4db6ff;
      --warn: #ffca63;
      --danger: #ff6b7d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--text);
      background:
        radial-gradient(800px 500px at 10% -20%, rgba(77, 182, 255, 0.3), transparent 70%),
        radial-gradient(900px 540px at 100% 0%, rgba(84, 242, 178, 0.2), transparent 65%),
        linear-gradient(130deg, var(--bg), var(--bg2));
      font-family: "Space Grotesk", "Segoe UI", sans-serif;
      min-height: 100vh;
    }
    .shell {
      max-width: 1120px;
      margin: 0 auto;
      padding: 28px 18px 40px;
    }
    .headline {
      display: flex;
      justify-content: space-between;
      align-items: end;
      gap: 14px;
      margin-bottom: 18px;
    }
    h1 {
      margin: 0;
      letter-spacing: 0.04em;
      font-weight: 700;
      font-size: clamp(1.5rem, 2vw, 2.1rem);
    }
    .tag {
      color: var(--muted);
      font-family: "JetBrains Mono", monospace;
      font-size: 12px;
    }
    .cards {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
      margin-bottom: 14px;
    }
    .card {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      backdrop-filter: blur(8px);
    }
    .k {
      font-family: "JetBrains Mono", monospace;
      font-size: 11px;
      color: var(--muted);
      margin-bottom: 8px;
      text-transform: uppercase;
      letter-spacing: 0.06em;
    }
    .v {
      font-size: 1.3rem;
      font-weight: 700;
    }
    .filters {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
      margin-bottom: 14px;
    }
    input, button {
      width: 100%;
      border-radius: 10px;
      border: 1px solid var(--line);
      background: rgba(8, 23, 33, 0.86);
      color: var(--text);
      padding: 10px 11px;
      font: inherit;
    }
    button {
      border-color: #3f6f91;
      background: linear-gradient(90deg, rgba(77, 182, 255, 0.22), rgba(84, 242, 178, 0.2));
      cursor: pointer;
      font-weight: 600;
    }
    .table-wrap {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 12px;
      overflow: auto;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      min-width: 860px;
    }
    th, td {
      padding: 10px 11px;
      text-align: left;
      border-bottom: 1px solid rgba(42, 75, 99, 0.55);
      font-size: 14px;
    }
    th {
      font-size: 11px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.07em;
    }
    .mono { font-family: "JetBrains Mono", monospace; }
    .ok { color: var(--accent); }
    .bad { color: var(--danger); }
    .warn { color: var(--warn); }
    @media (max-width: 920px) {
      .cards { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    }
  </style>
</head>
<body>
  <main class="shell">
    <section class="headline">
      <div>
        <h1>ModeloMan Prompt Leaderboard</h1>
        <div class="tag">Read-only telemetry view for ranking prompt versions by quality, cost, and latency.</div>
      </div>
      <button id="refreshBtn">Refresh</button>
    </section>

    <section class="cards">
      <article class="card"><div class="k">Runs</div><div id="runs" class="v">-</div></article>
      <article class="card"><div class="k">Attempts</div><div id="attempts" class="v">-</div></article>
      <article class="card"><div class="k">Success Rate</div><div id="successRate" class="v">-</div></article>
      <article class="card"><div class="k">Cost / Attempt</div><div id="costPerAttempt" class="v">-</div></article>
    </section>

    <section class="filters">
      <input id="workflow" placeholder="workflow filter" />
      <input id="model" placeholder="model filter" />
      <input id="windowDays" type="number" min="0" placeholder="window days (0 all)" />
      <input id="limit" type="number" min="1" placeholder="limit (default 20)" />
    </section>

    <section class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Workflow</th>
            <th>Prompt Version</th>
            <th>Model</th>
            <th>Attempts</th>
            <th>Success Rate</th>
            <th>Avg Cost</th>
            <th>Avg Latency</th>
            <th>Score</th>
          </tr>
        </thead>
        <tbody id="rows"></tbody>
      </table>
    </section>
  </main>
  <script>
    async function fetchJSON(url) {
      const res = await fetch(url);
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    }
    function pct(v) { return (v * 100).toFixed(1) + "%"; }
    function usd(v) { return "$" + Number(v || 0).toFixed(4); }
    function ms(v) { return Number(v || 0).toFixed(1) + " ms"; }

    async function refresh() {
      const workflow = document.getElementById("workflow").value.trim();
      const model = document.getElementById("model").value.trim();
      const windowDays = document.getElementById("windowDays").value.trim();
      const limit = document.getElementById("limit").value.trim();

      const summary = await fetchJSON("/api/telemetry-summary");
      document.getElementById("runs").textContent = summary.counts.runs;
      document.getElementById("attempts").textContent = summary.counts.attempts;
      document.getElementById("successRate").textContent = pct(summary.averages.success_rate || 0);
      document.getElementById("costPerAttempt").textContent = usd(summary.averages.cost_per_attempt || 0);

      const params = new URLSearchParams();
      if (workflow) params.set("workflow", workflow);
      if (model) params.set("model", model);
      if (windowDays) params.set("window_days", windowDays);
      if (limit) params.set("limit", limit);

      const items = await fetchJSON("/api/leaderboard?" + params.toString());
      const rows = document.getElementById("rows");
      rows.innerHTML = "";
      items.forEach((item, i) => {
        const tr = document.createElement("tr");
        const scoreCls = item.score >= 70 ? "ok" : item.score >= 45 ? "warn" : "bad";
        tr.innerHTML =
          '<td class="mono">' + (i + 1) + '</td>' +
          '<td>' + (item.workflow || "-") + '</td>' +
          '<td class="mono">' + (item.prompt_version || "-") + '</td>' +
          '<td class="mono">' + (item.model || "-") + '</td>' +
          '<td class="mono">' + (item.attempts || 0) + '</td>' +
          '<td class="mono">' + pct(item.success_rate || 0) + '</td>' +
          '<td class="mono">' + usd(item.average_cost_usd || 0) + '</td>' +
          '<td class="mono">' + ms(item.average_latency_ms || 0) + '</td>' +
          '<td class="mono ' + scoreCls + '">' + Number(item.score || 0).toFixed(2) + '</td>';
        rows.appendChild(tr);
      });
    }

    document.getElementById("refreshBtn").addEventListener("click", () => refresh().catch(console.error));
    ["workflow","model","windowDays","limit"].forEach((id) => {
      document.getElementById(id).addEventListener("change", () => refresh().catch(console.error));
    });
    refresh().catch(console.error);
  </script>
</body>
</html>`
