# llm (Go)

A tiny Go CLI that calls the OpenAI Responses API and prints a structured JSON answer. You describe the output shape on the command line via a simple format string, and the tool enforces that shape using a JSON Schema.

Great for:
- Scripting structured LLM results (pipe to jq, etc.)
- Quick extract/transform tasks
- Prototyping Responses API usage


## Features
- One-file CLI, easy to build and use
- Uses Responses API text format with JSON Schema to force structure
- All declared fields are required; no additional properties allowed
- Simple format string: "key:type,key:type,..."
- Optional instructions and reasoning effort
- Configurable base URL (default https://api.openai.com/v1)


## Requirements
- Go (see .tool-versions: 1.24.6 or newer compatible Go)
- An API key in OPENAI_API_KEY
- A provider that implements the OpenAI Responses API at /v1/responses (default is api.openai.com)


## Install
Clone and build:

```bash
# Clone your repo, then:
make build  # or: go build -o llm main.go
```

The resulting binary is ./llm (ignored by .gitignore).

Optionally move it into your PATH:

```bash
mv llm /usr/local/bin/
```


## Quick start
```bash
export OPENAI_API_KEY=sk-...

# Basic. Default schema = { message: string }
./llm "Say hi in one short sentence"
# => {"message":"Hello there!"}
```

Pipe to jq:

```bash
./llm "Give me a one-liner" | jq -r .message
```


## Structured output with --format
You can require specific keys and types by passing a simple format string:

```bash
./llm \
  --format "name:string,age:integer,active:boolean" \
  "Extract this user's info from the text: Alice is 42 and still employed."
# => {"name":"Alice","age":42,"active":true}
```

Notes on format:
- Use comma-separated pairs: key:type
- All provided keys become required
- additionalProperties=false is enforced (model should not add extra keys)
- Keys cannot contain commas or colons
- Type strings are placed directly into the JSON Schema. Common types include:
  - string, integer, number, boolean, array, object
  - For array/object, no nested item/property schema is generated (the CLI currently supports only a flat object with primitive type constraints)

If you omit --format, the tool uses a default schema:

```json
{
  "message": { "type": "string" }
}
```


## Instructions and reasoning
You can add instruction text and tweak reasoning effort:

```bash
./llm \
  --instructions "Be concise and factual." \
  --reasoning_effort high \
  --format "message:string" \
  "Summarize why the sky is blue in one sentence"
```

- reasoning_effort: low | medium | high (some models may ignore this; use a reasoning-capable model if needed)


## Models and base URL
- --model lets you select any model that supports Responses API; default is "gpt-5-nano" (override to a model you have access to)
- --base-url defaults to https://api.openai.com/v1

Examples:

```bash
# Use a different model
./llm --model some-supported-model "Brief greeting"

# Target a compatible proxy or self-hosted service that implements /v1/responses
./llm --base-url http://localhost:8000/v1 "Local test"
```

If your target does not implement the /responses endpoint (for example, it only supports chat/completions), you will see a 404 or similar error.


## CLI usage
```
llm [flags] "your message"

Flags:
  --model string              Model name (default "gpt-5-nano")
  --reasoning_effort string   Reasoning effort: low|medium|high (default "low")
  --verbosity string          Verbosity: low|medium|high (currently unused; default "low")
  --base-url string           Base URL for the LLM API (default "https://api.openai.com/v1")
  --instructions string       Instructions to guide the model
  --format string             Output format like: "name:string,age:integer,active:boolean"
  -h, --help                  Help for llm
```

Environment:
- OPENAI_API_KEY must be set


## How it works
The CLI sends a POST to {baseURL}/responses with a payload like:

```json
{
  "model": "...",
  "instructions": "...",
  "input": "...",
  "store": false,
  "text": {
    "format": {
      "type": "json_schema",
      "name": "response",
      "strict": true,
      "schema": {
        "type": "object",
        "properties": { "...": {"type":"..."} },
        "required": ["..."],
        "additionalProperties": false
      }
    }
  },
  "reasoning": { "effort": "low|medium|high" }
}
```

The program reads output_text from the API response. If missing, it falls back to the first output[].content[] item with type == "output_text". The printed text is expected to be the JSON that matches your schema.

Tip: The CLI does not append a newline; add one yourself or pipe to tools like jq.


## Troubleshooting
- 401 Unauthorized: Ensure OPENAI_API_KEY is set and valid
- 404 Not Found: Your base URL likely does not implement /v1/responses
- 400 Bad Request: Invalid model or format; verify types and that the target supports the Responses API
- Schema mismatch: If the model returns extra/missing keys, check your prompt and schema. The request enforces additionalProperties=false and marks all keys required.


## Project structure
- main.go: CLI implementation using cobra
- go.mod: module and dependencies
- Makefile: build helpers
- .tool-versions: Go version hint
- .gitignore: ignores the built binary (llm)


## Limitations and notes
- No streaming; responses are printed after completion
- Simple, flat schema only (no nested objects/items)
- Keys cannot contain commas or colons (used as separators)
- The verbosity flag is currently parsed but not sent to the API
- The default model value may not be available to your account—override it as needed


## Contributing
Issues and PRs are welcome. Ideas:
- Streaming mode
- Nested schema support
- More helpful validation for --format
- Append newline by default (with a flag to disable)
- Verbosity handling


## License
No license file was found in this repository. Until a LICENSE is added, all rights are reserved by the author.
