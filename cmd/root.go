package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	netpkg "net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"llmx/pkg/parser"
	"llmx/pkg/provider"
	"llmx/pkg/version"

	"github.com/spf13/cobra"
)

func ifEmpty(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func ifZero(val, fallback int) int {
	if val == 0 {
		return fallback
	}
	return val
}

var (
	model           string
	reasoningEffort string
	verbosity       string
	verbose         bool
	instructions    string
	format          string
	errorKey        string
	baseURL         string
	onlyKey         string
	providerName    string
	maxTokens       int
)

var rootCmd = &cobra.Command{
	Use:   "llmx [flags] [\"your message\"|-]",
	Short: "Send a message to the LLM API",
	Example: strings.TrimSpace(`
  # Minimal usage (OpenAI by default)
  llmx "Hello"

  # Anthropic / Gemini
  llmx --provider anthropic "Hello"
  llmx --provider gemini "Hello"

  # Read from stdin (pipe or file)
  echo "Hello" | llmx
  llmx - < prompt.txt

  # Structured JSON (OpenAI). Only print one key
  llmx --format "name:string,age:integer" "Alice is 14."
  llmx --format "command:string,explanation:string" --only command "Turn this into a shell command: list go files"
    `),
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var message string

		// Decide message source with a single read path
		shouldReadStdin := false
		if len(args) == 1 {
			if args[0] == "-" {
				// Force reading from stdin even on TTY
				shouldReadStdin = true
			} else {
				message = args[0]
			}
		} else { // no arg
			// If no arg, check whether stdin has piped input
			if fi, _ := os.Stdin.Stat(); fi.Mode()&os.ModeCharDevice == 0 {
				shouldReadStdin = true
			} else {
				// No piped input; show help like `llmx -h`
				_ = cmd.Help()
				return
			}
		}

		if shouldReadStdin {
			stdinBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Println("failed to read from stdin:", err)
				os.Exit(1)
			}
			message = string(stdinBytes)
		}

		// Select provider
		prov, err := provider.New(providerName)
		if err != nil {
			// Unknown provider: print supported list for clarity
			var up provider.ErrUnknownProvider
			if errors.As(err, &up) {
				fmt.Printf("unknown provider: %s\nSupported providers: openai, openai-compat, anthropic, gemini\n", providerName)
			} else {
				fmt.Println(err)
			}
			os.Exit(1)
		}

		// Always build properties (format).
		properties, err := parser.ParseFormat(format)
		if err != nil {
			fmt.Printf("failed to parse format: %v\n", err)
			os.Exit(1)
		}
		// If a custom --error-key is provided, require that the schema includes it.
		if strings.TrimSpace(errorKey) != "" && errorKey != "error" {
			if _, hasCustom := properties[errorKey]; !hasCustom {
				fmt.Printf("--error-key %q not found in --format schema. Include it in --format.\n", errorKey)
				os.Exit(1)
			}
		}

		// If --only is specified, validate that the key exists in the schema.
		if strings.TrimSpace(onlyKey) != "" {
			if _, hasOnly := properties[onlyKey]; !hasOnly {
				fmt.Printf("--only %q not found in --format schema. Include it in --format.\n", onlyKey)
				os.Exit(1)
			}
		}

		// Merge defaults from provider with CLI options
		def := prov.DefaultOptions()

		// Build provider payload
		payload, err := prov.BuildAPIPayload(
			provider.Options{
				Model:           ifEmpty(model, def.Model),
				Instructions:    instructions,
				Message:         message,
				Verbosity:       verbosity,
				ReasoningEffort: reasoningEffort,
				Properties:      properties,
				MaxTokens:       ifZero(maxTokens, def.MaxTokens),
			},
		)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if verbose {
			// Print payload intended for the provider
			if b, err := json.MarshalIndent(payload, "", "  "); err == nil {
				fmt.Fprintln(os.Stderr, "[llmx] Request payload:")
				fmt.Fprintln(os.Stderr, string(b))
			}
		}

		// Validate custom base URL early for friendlier errors
		if strings.TrimSpace(baseURL) != "" {
			if u, err := url.Parse(baseURL); err != nil || u.Scheme == "" || u.Host == "" {
				fmt.Printf("invalid --base-url: %q\nUse a full URL like https://api.example.com\n", baseURL)
				os.Exit(1)
			}
		}

		// Build request (API key resolved in provider if omitted here)
		req, err := prov.BuildAPIRequest(payload, baseURL, provider.RequestOptions{})
		if err != nil {
			// Friendly guidance for missing API keys using typed errors
			var mk provider.MissingAPIKeyError
			if errors.Is(err, provider.ErrMissingAPIKey) && errors.As(err, &mk) {
				env := strings.TrimSpace(mk.EnvVar)
				if env == "" {
					env = "API_KEY"
				}
				fmt.Printf("%s not found. Set one of:\n  bash/zsh: export %s=sk-...\n  fish:    set -x %s sk-...\n", env, env, env)
				os.Exit(1)
			}
			fmt.Println(err)
			os.Exit(1)
		}

		if verbose {
			// Redact secrets in URL and headers
			safeURL := req.URL.String()
			if u, err := url.Parse(safeURL); err == nil {
				q := u.Query()
				if q.Has("key") {
					q.Set("key", "***")
					u.RawQuery = q.Encode()
				}
				safeURL = u.String()
			}
			fmt.Fprintf(os.Stderr, "[llmx] Request: %s %s\n", req.Method, safeURL)
			fmt.Fprintln(os.Stderr, "[llmx] Headers:")
			for k, v := range req.Header {
				if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "x-api-key") || strings.EqualFold(k, "X-API-Key") {
					fmt.Fprintf(os.Stderr, "  %s: ***\n", k)
					continue
				}
				if len(v) > 0 {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", k, v[0])
				}
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// Add a bit more context for common network failures
			if ue, ok := err.(*url.Error); ok {
				if _, ok := ue.Err.(*netpkg.OpError); ok || strings.Contains(strings.ToLower(ue.Error()), "no such host") {
					fmt.Printf("network error: %v\nCheck connectivity and --base-url (if set).\n", err)
					os.Exit(1)
				}
			}
			fmt.Println("request failed:", err)
			os.Exit(1)
		}
		defer func() {
			// Explicitly ignore close error to satisfy errcheck
			_ = resp.Body.Close()
		}()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("failed to read response:", err)
			os.Exit(1)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "[llmx] Response status: %d\n", resp.StatusCode)
			// Print raw body (truncated if very large)
			const maxDump = 64 * 1024
			dump := respBody
			if len(dump) > maxDump {
				dump = dump[:maxDump]
			}
			fmt.Fprintln(os.Stderr, "[llmx] Raw response:")
			fmt.Fprintln(os.Stderr, string(dump))
			if len(respBody) > maxDump {
				fmt.Fprintln(os.Stderr, "[llmx] (truncated)")
			}
		}

		// Non-2xx handling
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Printf("request failed with status %d:\n%s\n", resp.StatusCode, string(respBody))
			os.Exit(1)
		}

		// Parse API response to extract text output (provider-specific)
		textOut, err := prov.ParseAPIResponse(respBody)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Best-effort JSON decode once; reuse for error/only handling.
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(textOut), &obj); err == nil {
			// If the structured JSON contains a non-empty "error", exit non-zero.
			if ev, ok := obj[errorKey]; ok {
				es, ok := ev.(string)
				es = strings.TrimSpace(es)
				if ok && es != "" && es != "null" {
					fmt.Fprintln(os.Stderr, es)
					os.Exit(1)
				}
			}
		} else {
			os.Exit(1)
		}

		// If --only is specified, attempt to parse structured JSON and print only that key
		if strings.TrimSpace(onlyKey) != "" {
			val, hasOnly := obj[onlyKey]
			if !hasOnly {
				fmt.Printf("key not found: %s\n", onlyKey)
				os.Exit(1)
			}
			switch v := val.(type) {
			case string:
				textOut = v
			case float64, bool, nil:
				b, _ := json.Marshal(v)
				textOut = string(b)
			default:
				// objects/arrays: print compact JSON
				b, err := json.Marshal(v)
				if err != nil {
					fmt.Println("failed to encode value:", err)
					os.Exit(1)
				}
				textOut = string(b)
			}
		}

		// Ensure output ends with a single newline
		if !strings.HasSuffix(textOut, "\n") {
			textOut += "\n"
		}
		fmt.Print(textOut)
	},
}

func init() {
	// Version info and template
	rootCmd.Version = version.String()
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.Flags().StringVar(&model, "model", "", "model name (provider default if empty)")
	rootCmd.Flags().StringVar(&reasoningEffort, "reasoning-effort", "minimal", "reasoning effort (minimal/low/medium/high)")
	rootCmd.Flags().StringVar(&verbosity, "verbosity", "low", "verbosity (low/medium/high)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "enable verbose debug logging to stderr")
	rootCmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL (provider default if empty)")
	rootCmd.Flags().StringVar(&providerName, "provider", "openai", "LLM provider name (e.g., openai)")
	rootCmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "max output tokens (override; provider default if 0)")
	rootCmd.Flags().StringVar(
		&instructions,
		"instructions",
		"",
		"instructions to guide the model",
	)
	rootCmd.Flags().StringVar(
		&format,
		"format",
		"message,error",
		"output format specification (default: \"message,error\"; e.g., \"name:string,age:integer,active:boolean\"). The error field name can be changed via --error-key",
	)
	rootCmd.Flags().StringVar(&errorKey, "error-key", "error", "name of the error field in structured JSON (non-empty triggers non-zero exit)")
	rootCmd.Flags().StringVar(
		&onlyKey,
		"only",
		"",
		"print only the specified top-level key from structured JSON output",
	)
}

func Execute() error {
	return rootCmd.Execute()
}
