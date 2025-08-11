# llmx

One CLI for OpenAI/Anthropic/Gemini. Simple prompts in, useful text out — and structured JSON when you want it via a friendly `--format` shorthand (OpenAI).

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

If a required key is missing, the CLI prints a clear error (no secrets leaked).

## Usage
- Minimal OpenAI: `llmx "Hello"`
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
- `--format string`  e.g. `"name:string,age:integer,tags:string[]"`
- `--only key`  print only a top‑level key from structured JSON
- `--max-tokens int`
- `--base-url string`  override API base URL (useful for gateways)
- `--verbosity low|medium|high` (default: low)
- `--verbose` enable debug logs to stderr (secrets are redacted)
- `--reasoning-effort minimal|low|medium|high` (default: minimal)
- `--version`

## Structured Output (OpenAI)
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

Note: Anthropic currently ignores `--format` (text‑only). Gemini support varies by model.

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

## Notes
- Security: API keys are never printed; debug logs redact secrets. Be cautious with `--base-url` if using third‑party gateways.
- Telemetry: none by default.

## Why this CLI?
- Multi‑provider: OpenAI, Anthropic, Gemini behind one interface.
- Structured output: simple `--format` shorthand → strict JSON (OpenAI).
- Simple flags: predictable, small surface area; works with pipes and files.
