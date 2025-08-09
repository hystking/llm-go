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

Array types with element specifications:

```bash
./llm \
  --format "dependencies:array[string],complexity:string,suggestions:array[string]" \
  "Analyze this Go project and list its dependencies, complexity, and suggestions"
# => {"dependencies":["github.com/spf13/cobra"],"complexity":"low","suggestions":["add tests","improve error handling"]}
```

Notes on format:
- Use comma-separated pairs: key:type
- All provided keys become required
- additionalProperties=false is enforced (model should not add extra keys)
- Keys cannot contain commas or colons
- Type strings are placed directly into the JSON Schema. Common types include:
  - string, integer, number, boolean, array, object
- Arrays with specific element types are supported using `array[element_type]` syntax:
  - `dependencies:array[string]` creates an array of strings
  - `scores:array[number]` creates an array of numbers
  - `flags:array[boolean]` creates an array of booleans

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

## Response API Overview

The OpenAI Responses API provides a streamlined way to generate structured outputs from language models. This API is designed for creating model responses with support for text, images, structured JSON, tool calling, and multi-turn conversations.

### Endpoint
- **POST** `https://api.openai.com/v1/responses`

### Authentication
- **Header**: `Authorization: Bearer $OPENAI_API_KEY`
- **Content-Type**: `application/json`

### Core Parameters

#### Required
- **model** (string): Model ID to use for generation
  - Examples: `"gpt-4o"`, `"o3"`, `"gpt-4.1"`
  - Different models have varying capabilities and performance characteristics

#### Input Options (at least one required)
- **input** (string or array): Primary input to the model
  - Can be text, image URLs, or file references
  - Supports multiple input types in a single request
- **prompt** (object): Reference to a prompt template with variables
  - Enables reusable prompt templates with parameter substitution

#### System Configuration
- **instructions** (string): System message to guide model behavior
  - Inserted into model context, not carried over in multi-turn conversations
- **text** (object): Configuration for text output format
  - **format.type**: `"text"` (plain text) or `"json_schema"` (structured)
  - **format.schema**: JSON Schema for structured outputs
  - **format.strict**: Boolean for strict schema adherence

### Conversation & State Management
- **previous_response_id** (string): ID of previous response for multi-turn conversations
- **store** (boolean): Whether to store response for later retrieval (default: true)
  - Set to false for stateless usage or zero data retention

### Model Behavior Controls
- **temperature** (number, 0-2): Sampling randomness (default: 1)
  - Higher values = more random, lower = more deterministic
- **top_p** (number, 0-1): Nucleus sampling probability mass (default: 1)
  - Alternative to temperature sampling
- **max_output_tokens** (integer): Upper bound for generated tokens
  - Includes both visible output and reasoning tokens
- **verbosity** (string): Response verbosity level
  - Options: `"low"`, `"medium"`, `"high"`
- **truncation** (string): Context window handling
  - `"auto"`: Automatic truncation if context exceeds limit
  - `"disabled"`: Fail with 400 error if context too large

### Tool Integration
- **tools** (array): Available tools for the model
  - **Built-in tools**: Web search, file search, code interpreter
  - **Custom functions**: User-defined functions with typed parameters
- **tool_choice** (string or object): Tool selection strategy
  - `"auto"`: Model chooses when to use tools
  - `"required"`: Force tool usage
  - Specific tool selection possible
- **parallel_tool_calls** (boolean): Allow concurrent tool execution (default: true)
- **max_tool_calls** (integer): Maximum total tool calls per response

### Reasoning Models (o-series)
- **reasoning** (object): Configuration for reasoning-capable models
  - **effort** (string): `"low"`, `"medium"`, `"high"`
  - Controls depth of reasoning process

### Advanced Features
- **background** (boolean): Run response asynchronously
- **stream** (boolean): Enable server-sent events streaming
- **stream_options** (object): Streaming configuration
- **include** (array): Additional data to include in response
  - `"code_interpreter_call.outputs"`: Python execution results
  - `"file_search_call.results"`: Search results details
  - `"message.output_text.logprobs"`: Token log probabilities
  - `"reasoning.encrypted_content"`: Encrypted reasoning tokens

### Quality & Safety
- **safety_identifier** (string): User identifier for policy violation detection
- **service_tier** (string): Processing tier selection
  - `"auto"`: Use project default
  - `"default"`: Standard processing
  - `"flex"` or `"priority"`: Enhanced processing tiers
- **metadata** (map): Key-value pairs for request tracking
  - Up to 16 pairs, keys ≤64 chars, values ≤512 chars

### Response Structure

The API returns a Response object with:

```json
{
  "id": "resp_...",
  "object": "response",
  "created_at": 1234567890,
  "status": "completed|in_progress|failed",
  "model": "gpt-4o-2024-08-06",
  "output": [
    {
      "type": "message",
      "role": "assistant", 
      "content": [
        {
          "type": "output_text",
          "text": "Generated response text",
          "annotations": []
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 36,
    "output_tokens": 87,
    "total_tokens": 123,
    "input_tokens_details": { "cached_tokens": 0 },
    "output_tokens_details": { "reasoning_tokens": 0 }
  },
  "reasoning": {
    "effort": "low",
    "summary": "Brief reasoning explanation"
  }
}
```

### Error Handling
- **401 Unauthorized**: Invalid API key
- **400 Bad Request**: Invalid parameters or model limitations
- **404 Not Found**: Endpoint not implemented by provider
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Provider-side issues

### Use Cases
- **Structured Data Extraction**: Use JSON Schema format for consistent outputs
- **Multi-turn Conversations**: Chain responses with `previous_response_id`
- **Tool-augmented Generation**: Combine LLM reasoning with external data/APIs
- **Background Processing**: Handle long-running generations asynchronously
- **Streaming Applications**: Real-time response generation with SSE

Tip: The CLI does not append a newline; add one yourself or pipe to tools like jq.


## Advanced Usage Examples

### Generate commit messages from git diff
Use structured output to generate commit messages automatically:

```bash
# Generate a structured commit message
git diff --staged | ./llm \
  --format "commit_message:string" \
  --instructions "Write a concise conventional commit message." \
  "Generate a commit message for these changes:\n" | jq -r .commit_message

# Output: {"type":"feat","scope":"auth","description":"add user authentication system","body":"Implement JWT-based authentication with login and logout endpoints"}

# Use in a script to create commits
COMMIT_MSG=$(git diff --staged | ./llm \
  --format "commit_message:string" \
  --instructions "Follow conventional commits format. Type should be feat/fix/docs/style/refactor/test/chore." \
  "Generate a commit message for these changes:\n" | jq -r .commit_message)
git commit -m "$COMMIT_MSG"
```

### Convert natural language to shell commands
Transform plain English requests into executable commands:

```bash
# Generate a command with explanation
./llm \
  --format "command:string,explanation:string,warning:string" \
  "I want to find all JavaScript files larger than 1MB in the current directory and subdirectories"

# Output: {"command":"find . -name '*.js' -size +1M","explanation":"Searches current directory recursively for .js files larger than 1MB","warning":"This command will traverse all subdirectories"}

# Interactive command generation and execution
CMD_INFO=$(./llm \
  --format "command:string,safe:boolean,explanation:string" \
  "Delete all log files older than 7 days in /var/log")
echo "Command: $(echo $CMD_INFO | jq -r .command)"
echo "Safe: $(echo $CMD_INFO | jq -r .safe)"
echo "Explanation: $(echo $CMD_INFO | jq -r .explanation)"
read -p "Execute? (y/n) " -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
  eval $(echo $CMD_INFO | jq -r .command)
fi
```

### Analyze and explain file contents
Extract structured information from files:

```bash
# Analyze a configuration file
cat config.yaml | ./llm \
  --format "purpose:string,potential_issues:array[string],security_level:string" \
  --instructions "Analyze this configuration file and identify its purpose, key settings, and any potential issues."

# Summarize source code
cat main.go | ./llm \
  --format "language:string,main_functionality:string,dependencies:array[string],complexity:string,suggestions:array[string]" \
  --instructions "Analyze this source code file and provide a structured summary"

# Document API endpoints from code
cat api.py | ./llm \
  --format "endpoints:array[string],authentication:string,data_formats:array[string],error_handling:string" \
  --instructions "Extract API documentation details from this code."
```


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
