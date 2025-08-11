# Repository Guidelines

## Project Structure & Modules
- `main.go`: Entrypoint that invokes `cmd.Execute()`.
- `cmd/`: Cobra CLI (`root.go`) defining flags, I/O, and execution flow.
- `pkg/provider/`: Provider abstraction and implementations (`openai.go`, `anthropic.go`).
- `pkg/parser/`: `--format` shorthand parser and tests.
- `pkg/version/`: Build-time version metadata.
- `Makefile`: Common build/test/install targets.
- `README.md`: Usage, examples, and install notes.

## Build, Test, and Dev Commands
- `make build`: Compile `llmx` binary in the project root.
- `make install`: Move binary to `~/bin/` (ensure it’s in `PATH`).
- `make test` or `go test -v ./...`: Run all tests.
- `go build -o llmx .`: Direct build with optional `-ldflags` for version.
- Run locally: `./llmx --provider openai "Hello"` (set API keys below).

## Coding Style & Conventions
- Language: Go. Use `go fmt ./...` before pushing; keep imports tidy.
- Structure: prefer small packages (`cmd`, `pkg/*`), table-driven tests, clear error messages.
- Naming: exported identifiers use Go conventions (CamelCase); files `snake_case.go`.
- Flags: add to `cmd/root.go`; keep names kebab-case (e.g., `--max-tokens`).

## Testing Guidelines
- Framework: standard `testing` package; tests live beside code as `*_test.go`.
- Patterns: table-driven tests (see `pkg/parser/format_test.go`).
- Run subsets: `go test ./pkg/parser -run TestParseFormat`.
- Aim to cover new branches when changing parser/provider logic.

## Commit & Pull Requests
- Commits: follow Conventional Commits (`feat:`, `fix:`, `test:`, `style:`, optional scopes like `feat(provider,cmd): ...`).
- PRs should include: clear description, linked issue (if any), rationale, and CLI examples.
- Requirements: tests passing (`make test`), update `README.md` when flags/behavior change, note env impacts.

## Security & Configuration
- Providers: `--provider openai|anthropic`.
- Env vars: `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` must be set.
- Optional overrides: `--base-url` for self-hosted gateways; validate before committing examples.

## Architecture Notes
- CLI builds a provider-agnostic `Options`, providers map to API payloads and parse responses.
- `--format` builds a strict JSON Schema for OpenAI; Anthropic currently ignores schema (text-only).

---

# AI Agent Collaboration Guide

These guidelines align AI coding agents (e.g., Codex CLI, Claude) with this repository’s practices. `CLAUDE.md` symlinks to this file to keep a single source of truth.

## Operating Principles
- Keep changes minimal and focused on the user’s request.
- Prefer root-cause fixes over superficial patches.
- Avoid unrelated refactors; call them out separately if discovered.
- Match existing style and structure; run `go fmt ./...` after edits.
- Update docs when flags/behavior change; include concise rationale.

## Workflow Expectations
- Preambles: before running grouped commands, state what you’ll do in 1–2 short sentences.
- Plans: for non-trivial tasks, keep a live plan with concise steps using the `update_plan` tool; always one `in_progress` step.
- Task execution: proceed end-to-end until the request is resolved; don’t stop mid-flight.
- Progress updates: for longer work, share short updates (one sentence) before/after substantive actions.
- Finalization: summarize what changed, where, and any next steps; keep it brief and scannable.

## Communication Style
- Be concise, direct, and friendly; prioritize actionable guidance.
- Use present tense and active voice (e.g., “Add test”, “Update flag”).
- Structure answers with short sections and bullets when helpful.
- Formatting rules for answers:
  - Section headers: 1–3 words, Title Case, wrapped with `**` (rendered by CLI).
  - Bullets: `- ` followed by a bold keyword and brief description.
  - Monospace: wrap commands, paths, env vars, and identifiers in backticks.
  - Avoid deep nesting; keep lists short and scannable.

## Editing Code
- Always use the `apply_patch` tool to add/update/delete files.
- Keep diffs surgical; do not rename/move files unless required by the task.
- Follow repository structure (`cmd/`, `pkg/*`); new logic belongs in the most specific package.
- Respect Go conventions: exported identifiers use CamelCase; files are `snake_case.go`.
- Do not add license headers unless explicitly requested.
- Avoid one-letter variable names and inline code comments unless asked.

## Testing & Validation
- Use `make test` or `go test -v ./...` to run tests.
- Start with focused tests for changed code, then broaden scope.
- Add tests only where the codebase patterns indicate a logical place (e.g., alongside parser/provider logic); avoid introducing a new testing framework.
- Do not fix unrelated failing tests; mention them succinctly if they block.
- Format code precisely; when formatter config exists, prefer running it on the smallest scope practical.

## Providers, Secrets, and Config
- Respect provider flags: `--provider openai|anthropic` and `--base-url` overrides.
- Never hardcode API keys; use env vars `OPENAI_API_KEY` or `ANTHROPIC_API_KEY`.
- When documenting examples, note required env vars and any safety implications.

## Sandbox & Approvals
- Prefer operating within the provided sandbox and approval settings.
- Request escalation only when necessary (e.g., network installs, destructive actions), and clearly explain why.
- For read-only environments, prepare patches and instructions without performing writes.

## When To Ask For Clarification
- Ambiguous scope (e.g., “optimize performance” without target metrics or files).
- Breaking changes to CLI flags or output formats.
- Large refactors, renames, or provider behavior changes.

## Quick Checklist (Before Finishing)
- Code builds: `make build` or `go build -o llmx .`.
- Tests pass: `make test` or targeted `go test` runs for changed packages.
- Formatting: `go fmt ./...` and imports are tidy.
- Docs: updated `README.md` and this file if behavior or workflow changed.
- Summary: concise change log, file paths, and any follow-up suggestions.
