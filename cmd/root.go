package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	instructions    string
	format          string
	baseURL         string
	onlyKey         string
	providerName    string
	maxTokens       int
)

var rootCmd = &cobra.Command{
	Use:   "llmx [flags] [\"your message\"|-]",
	Short: "Send a message to the LLM API",
	Args:  cobra.MaximumNArgs(1),
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
			fmt.Println(err)
			os.Exit(1)
		}

		// Build properties on CLI side if --format is specified
		var properties map[string]interface{}
		if strings.TrimSpace(format) != "" {
			props, err := parser.ParseFormat(format)
			if err != nil {
				fmt.Printf("failed to parse format: %v\n", err)
				os.Exit(1)
			}
			properties = props
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

		// Build request (API key resolved in provider if omitted here)
		req, err := prov.BuildAPIRequest(payload, baseURL, provider.RequestOptions{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("request failed:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("failed to read response:", err)
			os.Exit(1)
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

		// If --only is specified, attempt to parse structured JSON and print only that key
		if strings.TrimSpace(onlyKey) != "" {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(textOut), &obj); err != nil {
				fmt.Println("--only requires structured JSON output; failed to parse JSON:", err)
				os.Exit(1)
			}
			val, ok := obj[onlyKey]
			if !ok {
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
	rootCmd.Version = version.Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.Flags().StringVar(&model, "model", "", "model name (provider default if empty)")
	rootCmd.Flags().StringVar(&reasoningEffort, "reasoning-effort", "minimal", "reasoning effort (minimal/low/medium/high)")
	rootCmd.Flags().StringVar(&verbosity, "verbosity", "low", "verbosity (low/medium/high)")
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
		"",
		"output format specification (e.g., \"name:string,age:integer,active:boolean\")",
	)
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
