# llmx

A tiny CLI that sends a message to an LLM Responses API and prints the returned text. It can optionally enforce a JSON Schema using a compact --format shorthand.

## Install
- Prerequisite: Go 1.24.x
- Build: make build
- Install to /usr/local/bin: sudo make install
- Or: go build -o llmx main.go parser.go

## Configure
- Set your API key: export OPENAI_API_KEY=your_key
- Default base URL: https://api.openai.com/v1

## Usage
- `llmx [flags] "your message"`
- `echo "text" | llmx`
- Input rules:
  - If an argument is provided: message = argument
  - Else: message = stdin (errors if empty)
- Output: prints the API’s output_text (or the first output[].content[] item with type="output_text") without a trailing newline

## Flags
- `--model` (default: gpt-5-nano)
- `--reasoning_effort` low|medium|high (default: low)
- `--verbosity` low|medium|high (default: low)
- `--base-url` (default: https://api.openai.com/v1)
- `--instructions` string
- `--format` string (shorthand for a JSON Schema)

## Output formatting (--format)
- Shorthand: key:type pairs separated by commas
  - Example: name:string,age:integer,active:boolean
- Arrays: key:array[element_type]
  - Example: tags:array[string], scores:array[number]
- If omitted, a default schema with a single required field message:string is used
- All fields are required and additionalProperties=false
- Limitations: no nesting; element_type is required for arrays; invalid pairs (e.g., :string, tags:array[], name:string:string) cause an error

## Examples
- Basic: `llmx "Hello"`
- Pipe stdin: `echo "Hello" | llmx`
- With instructions: `llmx --instructions "Be brief." "Explain recursion"`
- Structured JSON: `llmx --format "name:string,age:integer" "Alice is a 14-year-old who is good at dancing."`
- Arrays: `llmx --format "tags:array[string]" "Give three tags for golang"`
- Custom base URL: `llmx --base-url http://localhost:8080/v1 "ping"`

## Advanced Examples

### Git commit message generation
```bash
# Generate commit message from staged changes
git diff --staged | llmx \
  --format "commit_message:string" \
  --instructions "Follow conventional commits format. Type should be feat/fix/docs/style/refactor/test/chore. Generate a git commit message for the following changes:" | \
  jq -r .commit_message
```

### Log file analysis
```bash
# Analyze error logs
cat error.log | llmx "Summarize the main issues and suggest solutions.\n" --format "issues:array[string],suggest_solutions:array[string]"
```

### Natural language to commands
```bash
# Convert description to shell command
llmx "How do I find all .go files modified in the last 7 days?" --format "command:string,explanation:string"
```

## Behavior and errors
- Non-2xx responses: prints the body and exits with code 1
- Missing OPENAI_API_KEY or invalid --format: exits with code 1
- Response parsing expects output_text or output[].content[].type == "output_text"

## Development
- Run tests: `make test` or `go test -v ./...`
- Clean: `make clean`
