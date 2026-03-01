# `mm` CLI (ModeloMan Workflow Wrapper)

`mm` is the primary operator-facing tool in ModeloMan.

It wraps vendor coding CLIs (Codex, Claude Code, Gemini CLI, OpenCode, etc.), assembles deterministic repo context, and logs each run to ModeloManHub over gRPC.

`modeloman` is an alias entrypoint for the same wrapper, but `mm` is the canonical binary name for docs, install, and server rollout.

It provides:
- per-repo context sets
- deterministic context bundle packing
- strict prompt template assembly
- backend subprocess execution
- telemetry logging to ModeloManHub over gRPC

## Install

```bash
go build -o mm ./cmd/mm
go build -o modeloman ./cmd/modeloman
```

or install directly:
```bash
go install ./cmd/mm
# optional alias binary:
go install ./cmd/modeloman
```

## Config

Path:
- `~/.config/modeloman/mm.yaml`

Example:

```yaml
grpc_addr: "grpc.modeloman.com:443"
grpc_insecure: false
token_env_var: "MODELOMAN_TOKEN"
default_backend: "codex"
redaction: true
max_context_bytes: 350000
max_transcript_bytes: 200000
allow_raw_transcript: false
custom_redaction_regex:
  - "(?i)my_internal_secret_[a-z0-9]+"
```

Token source:
- set env var from `token_env_var` (default `MODELOMAN_TOKEN`)
- legacy fallback env var accepted: `MODEL0MAN_TOKEN`

## Commands

```bash
mm add PATH|GLOB ...
mm drop PATH|GLOB ...
mm list
mm clear
mm run <backend> [--task TYPE] [--skill NAME] [--add PATH|GLOB ...] [--budget TOKENS] [--dry-run] [--pty=true] [-p "text" | --prompt-file PATH | < stdin]
mm tui
```

Alias:

```bash
modeloman run codex -p "same wrapper, alternate name"
```

Examples:

```bash
mm add internal/**/*.go cmd/mm/*.go
mm list
mm run codex --task bugfix --skill grpc-hardening --budget 12000 -p "Add max gRPC message size limits"
mm run claude --add README.md --prompt-file RESTART_PROMPT.md
cat RESTART_PROMPT.md | mm run codex --add cmd/** --add internal/**
mm tui
```

## Deliverable A Behavior

- Context set persisted at `.modeloman/context.json` in the git repo root.
- Context bundle contains:
  - repo root, branch, commit, dirty status
  - selected files
  - tree outline
  - git status + staged/unstaged diff
  - optional symbol grep hits extracted from objective text
- Prompt is wrapped into fixed template sections:
  - Objective
  - Constraints
  - Context
  - Deliverables
  - Test plan
  - Definition of Done
- Telemetry:
  - `StartRun`
  - `RecordPromptAttempt` (single attempt for MVP)
  - `RecordRunEvent` (start metadata, diff summary, feedback)
  - `FinishRun`
- Safety defaults:
  - redaction enabled by default
  - no raw token persistence
  - transcript storage not enabled unless future config change

## Deliverable B (PTY + Transcript)

- `mm run` defaults to PTY mode (`--pty=true`) on supported OSes.
- Structured runner events are logged:
  - `backend_started`
  - `prompt_injected`
  - `backend_ended`
- Transcript capture is capped by `max_transcript_bytes`.
- Transcript event payload always stores redacted transcript.
- Raw transcript is only included when `allow_raw_transcript: true`.

## Deliverable C (Bubble Tea TUI)

- Launch with:
```bash
mm tui
# or
modeloman tui
```

- Screens:
  - Home: choose backend/task/skill/budget and objective text.
  - Context Picker: fuzzy filter repo files, toggle selection, persist context.
  - Preview: context stats + prompt preview.
  - Run: live backend output stream + runner events + timer.
  - Post-run: diff summary, changed files, rating + notes, prompt coach suggestions.

- TUI persistence:
  - `.modeloman/context.json` for context entries.
  - `.modeloman/ui_state.json` for last backend/task/skill/budget/objective and recent selected files.

- True passthrough in Run screen:
  - Press `i` to toggle passthrough ON/OFF.
  - When ON, keys are forwarded to backend PTY.
  - Press `ctrl+g` to leave passthrough mode and return to TUI controls.
