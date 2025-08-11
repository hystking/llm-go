# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and the project adheres to Semantic Versioning.

## v0.1.0 — Initial release
- Multi‑provider CLI: OpenAI (default), Anthropic, Gemini.
- Structured output for OpenAI via `--format` shorthand (strict JSON Schema).
- Input from arg, pipe, or file via stdin (`-`).
- Common flags: `--provider`, `--model`, `--instructions`, `--format`, `--only`, `--max-tokens`, `--base-url`, `--verbosity`, `--verbose`, `--reasoning-effort`, `--version`.
- Version metadata wired via `-ldflags`.

