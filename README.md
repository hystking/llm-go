# llmx

One CLI for OpenAI/Anthropic/Gemini. Simple prompts in, useful JSON out — always structured via a friendly `--format` shorthand. By default, output is a JSON object with two string fields: `message` and `error`. If the configured error field is non-empty, the CLI prints it to stderr and exits with a non-zero status.

## Quickstart
- Requires Go 1.24.x
- Build: `make build`
- Install to `~/bin`: `make install` (ensure `~/bin` is in `PATH`)
- Direct build: `go build -o llmx .`
- Or from Releases: download a binary, `chmod +x llmx`, then place it in your `PATH`.

Tip: If publishing the module, you can also support `go install <module>@latest`.

## API Keys
- OpenAI (default provider): `OPENAI_API_KEY`
- Anthropic: `ANTHROPIC_API_KEY`
- Gemini: `GEMINI_API_KEY`

Examples:
- bash/zsh: `export OPENAI_API_KEY=sk-...`
- fish: `set -x OPENAI_API_KEY sk-...`

The default provider is OpenAI. If you omit `--provider` and `OPENAI_API_KEY` is not set, the CLI prints a clear error with setup examples. Secrets are never shown.

## Usage
- Minimal OpenAI (JSON by default): `llmx "Hello"`
- Minimal Anthropic: `llmx --provider anthropic "Hello"`
- Minimal Gemini: `llmx --provider gemini "Hello"`

Input options:
- Arg: `llmx "your message"`
- Pipe: `echo "text" | llmx`
- File via stdin: `llmx - < prompt.txt`

## Common flags
- `--provider openai|anthropic|gemini` (default: openai)
- `--model string`
- `--instructions string`
- `--format string`  default `"message,error"`; e.g., `"name:string,age:integer,tags:string[]"`
- `--error-key string` name of the error field in structured JSON (default: `error`). When set, your `--format` schema must include this key.
- `--only key`  print only the specified top-level key from structured JSON output (requires JSON output, e.g., via `--format`)
- `--max-tokens int`
- `--base-url string`  override API base URL (useful for gateways)
- `--verbosity low|medium|high` (default: low)
- `--verbose` enable debug logs to stderr (secrets are redacted)
- `--reasoning-effort minimal|low|medium|high` (default: minimal)
- `--version`

## Structured Output
- Shorthand: key:type pairs, comma‑separated: `name:string,age:integer`
- Arrays: `key:type[]` (`tags:string[]`)
- Omitted type defaults to string (`name`)

Example:
```
$ llmx --format "name:string,age:integer" "Alice is 14."
{"name":"Alice","age":14}

$ llmx --format "command:string,explanation:string" --only command \
  "Turn this into a shell command: list go files"
"find . -type f -name '*.go'"
```

Notes:
- OpenAI: strict JSON schema is enforced.
- Anthropic: the CLI adds a system prompt to return only strict JSON for the requested keys; compliance depends on the model.
- Gemini: uses JSON mode with a response schema when supported by the model.

## Examples
- Basic: `llmx "Hello"`
- Anthropic: `llmx --provider anthropic "Hello"`
- Gemini: `llmx --provider gemini "Hello"`
- Pipe: `echo "Hello" | llmx`
- File: `llmx - < prompt.txt`
- Structured JSON: `llmx --format "name:string,age:integer" "Alice is 14."`
- Only a field: `llmx --format "command:string,explanation:string" --only command "Turn this into a shell command: list go files"`

## Development
- Tests: `make test`
- Clean: `make clean`
- Build with version info:
```
go build -o llmx \
  -ldflags "-X llmx/pkg/version.Version=v0.1.0 \
             -X llmx/pkg/version.Commit=$(git rev-parse --short HEAD) \
             -X llmx/pkg/version.Date=$(date -u +%Y-%m-%d)" .
```

## Releases (GitHub Releases)
- Automated (recommended):
  - Update `CHANGELOG.md`.
  - Create a tag: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
  - GitHub Actions builds binaries for Linux/macOS/Windows (amd64/arm64) via GoReleaser and creates a draft Release with assets.
  - Review the draft, add notes if needed, and publish.

- Manual (local):
  - Ensure GoReleaser is installed and `GH_TOKEN`/`GITHUB_TOKEN` is set.
  - Run `make release` or `goreleaser release --clean` from a tagged commit (e.g., `vX.Y.Z`).

Notes:
- Version/commit/date are embedded via `-ldflags` from the tag and Git metadata.
- The workflow file lives at `.github/workflows/release.yml`; GoReleaser config is `.goreleaser.yml`.
- To auto‑publish instead of draft, set `release.draft: false` in `.goreleaser.yml`.

## Notes
- Security: API keys are never printed; even with `--verbose`, secrets are redacted.
- Base URL: when you set `--base-url`, your prompts and outputs may be sent to a third‑party gateway. Use only trusted endpoints and review their privacy/logging policies. The URL must be a full `https://` URL with a host; invalid URLs are validated early with a friendly error.
- Telemetry: none by default.

## Why this CLI?
- Multi‑provider: OpenAI, Anthropic, Gemini behind one interface.
- Structured output: simple `--format` shorthand → strict JSON (OpenAI).
- Simple flags: predictable, small surface area; works with pipes and files.
