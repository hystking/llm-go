package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"llmx/pkg/config"
	"llmx/pkg/parser"
	"llmx/pkg/provider"
	"llmx/pkg/version"

	"github.com/spf13/cobra"
)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
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
	profileName     string
	configPath      string
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

		// Build default config path on CLI side
		cfgPath := configPath
		if strings.TrimSpace(cfgPath) == "" {
			if dir, err := os.UserConfigDir(); err == nil {
				cfgPath = filepath.Join(dir, "llmx", "config.json")
			}
		}
		// Load profile (non-fatal if missing or path absent)
		prof, _ := config.Load(cfgPath, profileName)

		// Select provider (profile -> CLI)
		prov, err := provider.New(firstNonEmpty(providerName, prof.Provider))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Build properties parsing CLI/profile format if provided
		var properties map[string]interface{}
		if fs := firstNonEmpty(format, prof.Format); strings.TrimSpace(fs) != "" {
			props, err := parser.ParseFormat(fs)
			if err != nil {
				fmt.Printf("failed to parse format: %v\n", err)
				os.Exit(1)
			}
			properties = props
		}

		// Merge defaults from provider with profile and CLI options
		def := prov.DefaultOptions()
		payload, err := prov.BuildAPIPayload(provider.Options{
			Model:           firstNonEmpty(model, prof.Model, def.Model),
			Instructions:    firstNonEmpty(instructions, prof.Instructions),
			Message:         message,
			Verbosity:       firstNonEmpty(verbosity, prof.Verbosity),
			ReasoningEffort: firstNonEmpty(reasoningEffort, prof.ReasoningEffort),
			Properties:      properties,
			MaxTokens:       firstNonZero(maxTokens, prof.MaxTokens, def.MaxTokens),
		})

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Build request (API key resolved in provider if omitted here)
		req, err := prov.BuildAPIRequest(
			payload,
			firstNonEmpty(baseURL, prof.BaseURL),
			provider.RequestOptions{},
		)
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

		// If only is specified (CLI/profile), print only the specified key
		if only := firstNonEmpty(onlyKey, prof.Only); strings.TrimSpace(only) != "" {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(textOut), &obj); err != nil {
				fmt.Println("--only requires structured JSON output; failed to parse JSON:", err)
				os.Exit(1)
			}
			val, ok := obj[only]
			if !ok {
				fmt.Printf("key not found: %s\n", only)
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
	rootCmd.Flags().StringVar(&reasoningEffort, "reasoning_effort", "minimal", "reasoning effort (minimal/low/medium/high)")
	rootCmd.Flags().StringVar(&verbosity, "verbosity", "low", "verbosity (low/medium/high)")
	rootCmd.Flags().StringVar(&baseURL, "base-url", "", "override base URL (provider default if empty)")
	rootCmd.Flags().StringVar(&providerName, "provider", "openai", "LLM provider name (e.g., openai)")
	rootCmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "max output tokens (override; provider default if 0)")
	rootCmd.Flags().StringVar(&profileName, "profile", "", "profile name from config (falls back to default_profile)")
	rootCmd.Flags().StringVar(&configPath, "config", "", "path to config file (defaults to ~/.config/llmx/config.json)")
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
