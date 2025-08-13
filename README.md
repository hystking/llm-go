# llmx

A fast, schema-first CLI for calling multiple LLM providers (OpenAI, OpenAI-Compatible Chat, Anthropic, Gemini) with structured JSON output by default.

- Multi-provider: OpenAI (Responses API), OpenAI-Compatible Chat (Chat Completions), Anthropic (Messages API), Gemini (GenerateContent)
- JSON-first: build strict schemas from a compact `--format` shorthand
- Simple I/O: message from arg, pipe, or file (`-`)
- Dev-friendly: verbose debugging with redaction, consistent flags across providers


## Quick Start

1) Set an API key for your chosen provider:

- OpenAI: `export OPENAI_API_KEY=sk-...`
- Anthropic: `export ANTHROPIC_API_KEY=...`
- Gemini: `export GEMINI_API_KEY=...`

2) Call a model (OpenAI by default):

```
llmx "Summarize this tool in one line"
```

3) Choose a provider:

```
llmx --provider anthropic "Hello"
llmx --provider gemini "Hello"
```

4) Read input from stdin:

```
echo "Turn this into a shell one-liner: list Go files" | llmx
llmx - < prompt.txt
```

5) Structured output (default). The default schema is `message:string,error:string`:

```
llmx "What is 2+2?"
# => {"message":"4","error":""}
```

6) Print only one key from the structured output:

```
llmx --format "command:string,explanation:string" --only command \
  "Turn this into a shell command: list Go files"
# => find . -type f -name "*.go"
```


## Installation

- Binaries: see GitHub Releases (built via GoReleaser) for Linux, macOS, Windows (amd64, arm64).
- From source (Go 1.24+):

```
git clone <this-repo>
cd llmx
make build           # builds ./llmx with version metadata
make install         # moves llmx to ~/bin/ (ensure it's in PATH)
```

Run tests:

```
make test
```


## CLI Overview

- `llmx [flags] ["your message"|-]`
- If `-` is given or stdin is piped, llmx reads the message from stdin. Otherwise it uses the single argument as the message. If neither is provided and stdin is a TTY, help is shown.

Common flags:

- `--provider` string: `openai` (default) | `openai-compat` | `anthropic` | `gemini`
- `--model` string: model name; defaults per provider
- `--instructions` string: system/instructions text
- `--format` string: output schema shorthand (default `"message,error"`)
- `--only` string: print only the specified top-level key
- `--error-key` string: name of the error field (default `error`)
- `--max-tokens` int: provider-specific max output tokens (0 = provider default)
- `--verbosity` string: `low` (default) | `medium` | `high` (OpenAI only)
- `--reasoning-effort` string: `minimal` (default) | `low` | `medium` | `high` (OpenAI only)
- `--base-url` string: override provider base URL (full URL)
- `--verbose`: print request/response debug info to stderr (secrets redacted)
- `--version`: print version (tag/commit/date)

Exit behavior:

- Non-2xx HTTP: prints status/body and exits non-zero.
- Structured output is required: response text must be valid JSON. If JSON parsing fails, llmx exits non-zero.
- Error gating: if `--error-key` is present in the JSON and is a non-empty string (not `"null"`), llmx prints it to stderr and exits non-zero.


## Structured Output (Schema Shorthand)

llmx builds provider-specific JSON constraints from a compact `--format` shorthand:

- Grammar: `key[:type]` pairs, comma-separated. Example: `name:string,age:integer,active:boolean`.
- Arrays: `type[]` (e.g., `tags:string[]`, `scores:number[]`). Nested arrays (`[][]`) are not allowed.
- Whitespace around keys/types is ignored; duplicate keys: last one wins.
- All keys are considered required.

Examples:

- `--format "message,error"` (default) => both are strings
- `--format "name:string,age:integer,active:boolean"`
- `--format "tags:string[]"`

Provider mapping:

- OpenAI (Responses API): strict `json_schema` with `required` for all keys.
- OpenAI-Compatible Chat (Chat Completions): adds a strict-JSON system hint and, when possible, sets `response_format={type:"json_schema", json_schema:{...}}`.
- Gemini (GenerateContent): `generationConfig.responseMimeType=application/json` + `responseSchema` with uppercased types (`STRING`, `INTEGER`, `NUMBER`, `BOOLEAN`, `ARRAY`).
- Anthropic (Messages API): a precise system instruction is injected that asks for strict JSON only; Anthropic does not enforce JSON schema natively.

Error gating with `--error-key` (default `error`): if present and non-empty, llmx exits non-zero. Change with `--error-key <name>` and add that key to your `--format`.


## Providers

OpenAI

- API: `POST https://api.openai.com/v1/responses`
- Auth: `Authorization: Bearer $OPENAI_API_KEY`
- Defaults: `model=gpt-5-nano`
- Mapping:
  - `input` = message
  - `instructions` = instructions
  - `text.verbosity` = `--verbosity`
  - `reasoning.effort` = `--reasoning-effort`
  - `max_output_tokens` = `--max-tokens` (if > 0)
  - JSON schema when `--format` is provided (default is provided).

OpenAI-Compatible Chat

- API: `POST https://api.openai.com/v1/chat/completions`
- Auth: `Authorization: Bearer $OPENAI_API_KEY`
- Defaults: `model=gpt-4o-mini`
- Mapping:
  - `messages=[{role:system, content: instructions (+ strict JSON hint)}, {role:user, content: message}]`
  - `response_format={type:json_schema, json_schema:{name: "response", schema:{...}}}` when `--format` is provided
  - `max_tokens` = `--max-tokens` (if > 0)

Anthropic

- API: `POST https://api.anthropic.com/v1/messages`
- Auth: `x-api-key: $ANTHROPIC_API_KEY`, `anthropic-version: 2023-06-01`
- Defaults: `model=claude-3-5-haiku-latest`, `max_tokens` derived from model family; override with `--max-tokens`.
- Mapping:
  - `messages=[{role:user, content: message}]`
  - `system` = instructions (+ strict JSON guidance when `--format` is set)
  - `max_tokens` = `--max-tokens` or default per model

Gemini

- API: `POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key=$GEMINI_API_KEY`
- Auth: query string `key=$GEMINI_API_KEY`
- Defaults: `model=gemini-2.0-flash`
- Mapping:
  - `contents=[{parts:[{text: message}]}]`
  - `systemInstruction.parts[0].text` = instructions (optional)
  - `generationConfig.maxOutputTokens` = `--max-tokens` (if > 0)
  - JSON mode when `--format` is provided (default is provided): `responseMimeType=application/json` + `responseSchema`.

Base URLs

- Override with `--base-url` (full URL, including scheme and host). Defaults:
  - OpenAI: `https://api.openai.com/v1`
  - Anthropic: `https://api.anthropic.com/v1`
  - Gemini: `https://generativelanguage.googleapis.com`


## Debugging and Logging

- `--verbose` prints:
  - Provider payload (JSON)
  - Request method/URL (API keys redacted), headers (secrets redacted)
  - Response status and raw body (truncated at 64 KiB)
- Logs are written to stderr; standard output is reserved for model output (or the `--only` selection).


## Development

- Build: `make build`
- Install: `make install` (to `~/bin/`)
- Test: `make test`
- Lint: `make lint` (golangci-lint)
- Release: `make release` (GoReleaser)

Version metadata is embedded via `-ldflags` (tag/commit/date); `llmx --version` prints it.

Project layout:

- `main.go`: entrypoint
- `cmd/`: Cobra CLI (`root.go`)
- `pkg/provider/`: provider interface and implementations
- `pkg/parser/`: `--format` shorthand parser
- `pkg/version/`: build-time version metadata


## Extending (Adding a Provider)

- Implement `pkg/provider.Provider` (four methods):
  - `DefaultOptions() Options`
  - `BuildAPIPayload(Options) (map[string]interface{}, error)`
  - `BuildAPIRequest(payload, baseURL, RequestOptions)`
  - `ParseAPIResponse([]byte) (string, error)`
- Register it in `provider.New(name)` switch.
- Add tests mirroring existing providers.


## Notes and Guarantees

- Non-streaming: responses are printed after the request completes.
- Structured JSON is required: the CLI parses the model output as JSON and exits non-zero on parse failure.
- Secrets are never printed in logs; API keys are redacted.


## Environment Variables

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GEMINI_API_KEY`

Set one per the provider you use. You can also pass API keys via gateways using `--base-url` (ensure compatible auth semantics).


## Changelog

See CHANGELOG.md.


## Contributing

See AGENTS.md for repository practices, coding style, and testing guidelines. PRs with tests and concise rationale are welcome.


## License

MIT License (see LICENSE).
