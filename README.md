# llmx

Minimal CLI for sending a prompt to an LLM API and printing the response. Optional JSON schema enforcement via a simple `--format` shorthand.

## Install
- Requires Go 1.24.x
- Build: `make build`
- Install: `make install` (ensure `~/bin` is in `PATH`)
- Or: `go build -o llmx .`

## Setup
- OpenAI: `export OPENAI_API_KEY=...`  (default provider)
- Anthropic: `export ANTHROPIC_API_KEY=...`  (when `--provider anthropic`)

## Usage
- `llmx [flags] "your message"`
- `echo "text" | llmx`
- `llmx -`  (force stdin)

## Common flags
- `--provider openai|anthropic` (default: openai)
- `--model string`
- `--instructions string`
- `--format string`  e.g. `"name:string,age:integer,tags:string[]"`
- `--only key`       print only a key from structured JSON output
- `--max-tokens int`
- `--base-url string`
- `--verbosity low|medium|high` (default: low)
- `--reasoning_effort minimal|low|medium|high` (default: minimal)
- `--version`

## Format shorthand
- key:type pairs, comma-separated: `name:string,age:integer`
- Arrays: `key:type[]` (`tags:string[]`)
- Omitted type defaults to string (`name`)

## Examples
- Basic: `llmx "Hello"`
- Stdin: `echo "Hello" | llmx`
- Structured JSON: `llmx --format "name:string,age:integer" "Alice is 14."`
- Only a field: `llmx --format "command:string,explanation:string" --only command "Turn this into a shell command: list go files"`
- Anthropic: `llmx --provider anthropic "Hello"`

## Development
- Tests: `make test`
- Clean: `make clean`
- Build with version info:
  ```sh
  go build -o llmx \
    -ldflags "-X llmx/pkg/version.Version=v0.1.0 \
               -X llmx/pkg/version.Commit=$(git rev-parse --short HEAD) \
               -X llmx/pkg/version.Date=$(date -u +%Y-%m-%d)" .
  ```
