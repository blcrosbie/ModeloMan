You are Codex. Implement a Go CLI tool named `mm` inside this repository (ModeloMan). Its job is to wrap AI coding CLIs (codex/claude/gemini/opencode) and log each session to the existing ModeloMan gRPC server.

Primary objective: improve the human’s prompting skill by recording every run, prompt, and outcome. We are collecting maximum useful metadata now; analysis/coaching comes later.

Constraints:
- Go only. No Node. No REST.
- Must compile and run on Ubuntu 24.04.
- Must work in any git repo: detect repo root, branch, commit SHA, dirty status.
- Must support a persistent per-repo “context set” stored at: <repo_root>/.modeloman/context.json
- Must support `mm add/drop/list/clear` to manage the context set (paths and globs).
- Must support `mm run <backend>` to:
  1) collect prompt (from -p flag or stdin)
  2) resolve context globs to a stable ordered file list (ignore .git, node_modules, dist, vendor, bin)
  3) build a deterministic “context bundle” summary: file list + truncated file contents up to max_context_bytes + git diff (staged+unstaged)
  4) start a session log in ModeloMan via gRPC and store the prompt attempt
  5) run the backend tool locally (initially non-interactive: pass the final prompt to stdin)
  6) on exit, capture changed files + diff summary
  7) ask user for rating 1–5 and a short note
  8) finish the session log via gRPC

Logging target:
- Use the existing gRPC endpoint: grpc.modeloman.com:443 with TLS.
- Authenticate using metadata header: x-modeloman-token read from env var MODELOMAN_TOKEN.
- Use the proto in proto/modeloman/v1/hub.proto.
- If there is no dedicated RPC for session logging yet, use AppendChangelog to store a JSON object containing:
  - backend, task_type, repo metadata, prompt hash, context hash, rating, notes, elapsed_ms, changed_files, diff_stats
- Prefer using StartRun/FinishRun/RecordPromptAttempt RPCs if they exist in the proto, otherwise fall back to AppendChangelog.

Deliverables:
1) Add cmd/mm/main.go with cobra or urfave/cli (pick one). Provide subcommands: add, drop, list, clear, run.
2) Add internal packages for config, gitmeta, contextset, bundler, runner, modeloman client.
3) Provide a minimal config loader that reads ~/.config/modeloman/mm.yaml (grpc addr, capture limits) but works with defaults if missing.
4) Include clear README docs for mm usage.

Hard requirements:
- Deterministic bundle building (stable ordering).
- Safe by default: redact obvious secrets in bundle (API keys, Bearer tokens, AWS keys, private key blocks) with simple regex.
- Timeouts and good error messages.
- Do NOT ask questions; choose sensible defaults and implement.

Now implement the code with buildable Go modules and include exact commands to run and test locally.