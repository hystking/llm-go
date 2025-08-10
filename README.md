# llmx

A tiny CLI that sends a message to an LLM Responses API and prints the returned text. It can optionally enforce a JSON Schema using a compact --format shorthand.

## Install
- Prerequisite: Go 1.24.x
- Build: `make build`
- Install to `~/bin`: `make install` (ensure `~/bin` is in your PATH)
- Or: `go build -o llmx .`

## Configure
- Set your API key: `export OPENAI_API_KEY=your_key`

## Usage
- `llmx [flags] "your message"`
- `llmx -` (force reading from stdin)
- `echo "text" | llmx` (reads piped stdin)
- Output: prints the API’s `output_text` (or the first `output[].content[]` item with `type="output_text"`)

## Flags
- `--model` (default: gpt-5-nano)
- `--reasoning_effort` minimal|low|medium|high (default: minimal)
- `--verbosity` low|medium|high (default: low)
- `--base-url` (default: https://api.openai.com/v1)
- `--instructions` string
- `--format` string (shorthand for a JSON Schema)
- `--only` key (print only the specified top-level key from structured JSON output)
- `--version` print version and exit

## Output formatting (--format)
- Shorthand: key:type pairs separated by commas
  - Example: name:string,age:integer,active:boolean
- Arrays: key:element_type[] (type[] style)
  - Example: tags:string[], scores:number[]
- Omitted type defaults to string
  - Example: name,age:integer => name is string
- If omitted, no schema is enforced and free-form text is returned
- All fields are required and additionalProperties=false

## Examples
- Basic: `llmx "Hello"`
- Pipe stdin: `echo "Hello" | llmx`
- Force stdin: `printf "Hello" | llmx -`
- With instructions: `llmx --instructions "Be brief." "Explain recursion"`
- Structured JSON: `llmx --format "name:string,age:integer" "Alice is a 14-year-old who is good at dancing."`
- Only a key: `llmx --format "name:string,age:integer" --only name "Alice is a 14-year-old..."`
- Arrays: `llmx --format "tags:string[]" "Give three tags for golang"`
- Custom base URL: `llmx --base-url http://localhost:8080/v1 "ping"`

## Advanced Examples

### Git commit message generation
```bash
# Generate commit message from staged changes
git diff --staged | llmx \
  --format "commit_message:string" \
  --instructions "Follow conventional commits format. Type should be feat/fix/docs/style/refactor/test/chore. Generate a git commit message for the following changes:"\
  --only "commit_message"
```

### Log file analysis
```bash
# Analyze error logs
cat error.log | llmx "Summarize the main issues and suggest solutions.\n" --format "issues:string[],suggest_solutions:string[]"
```

### Natural language to commands
```bash
# Convert description to shell command
llmx "How do I find all .go files modified in the last 7 days?" --format "command,explanation" --only command
```

## Development
- Run tests: `make test` or `go test -v ./...`
- Clean: `make clean`
  
### Build with version metadata (optional)
```bash
go build -o llmx \
  -ldflags "-X llmx/pkg/version.Version=v0.1.0 \
            -X llmx/pkg/version.Commit=$(git rev-parse --short HEAD) \
            -X llmx/pkg/version.Date=$(date -u +%Y-%m-%d)" .
```
