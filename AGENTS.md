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
