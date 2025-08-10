package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"llmx/pkg/parser"
	"llmx/pkg/version"

	"github.com/spf13/cobra"
)

var (
	model           string
	reasoningEffort string
	verbosity       string
	instructions    string
	format          string
	baseURL         string
	onlyKey         string
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

		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("OPENAI_API_KEY is not set")
			os.Exit(1)
		}

		// Build payload per Responses API
		// When --format is omitted, do NOT enforce a schema.
		textPayload := map[string]interface{}{
			"verbosity": verbosity,
		}

		if strings.TrimSpace(format) != "" {
			// Parse format string to generate JSON schema
			properties, required, err := parser.ParseFormat(format)
			if err != nil {
				fmt.Printf("failed to parse format: %v\n", err)
				os.Exit(1)
			}

			textPayload["format"] = map[string]interface{}{
				"type":   "json_schema",
				"name":   "response",
				"strict": true,
				"schema": map[string]interface{}{
					"type":                 "object",
					"properties":           properties,
					"required":             required,
					"additionalProperties": false,
				},
			}
		}

		payload := map[string]interface{}{
			"model":        model,
			"instructions": instructions,
			"input":        message,
			"store":        false,
			"text":         textPayload,
			"reasoning": map[string]interface{}{
				"effort": reasoningEffort,
			},
		}

		body, err := json.Marshal(payload)
		if err != nil {
			fmt.Println("failed to encode payload:", err)
			os.Exit(1)
		}

		req, err := http.NewRequest("POST", baseURL+"/responses", bytes.NewReader(body))
		if err != nil {
			fmt.Println("failed to create request:", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

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

		// Parse API response to extract text output
		textOut, err := parser.ParseAPIResponse(respBody)
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

	rootCmd.Flags().StringVar(&model, "model", "gpt-5-nano", "model name")
	rootCmd.Flags().StringVar(&reasoningEffort, "reasoning_effort", "minimal", "reasoning effort (minimal/low/medium/high)")
	rootCmd.Flags().StringVar(&verbosity, "verbosity", "low", "verbosity (low/medium/high)")
	rootCmd.Flags().StringVar(&baseURL, "base-url", "https://api.openai.com/v1", "base URL for the LLM API (e.g. https://api.openai.com/v1)")
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
