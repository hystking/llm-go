package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var (
		model           string
		reasoningEffort string
		verbosity       string
		instructions    string
		format          string
		prompt          string
	)

	var baseURL string

	var rootCmd = &cobra.Command{
		Use:   "llm [flags] [\"your message\"]",
		Short: "Send a message to the LLM API",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var message string

			if prompt != "" {
				// When --prompt is specified, use it and optionally combine with stdin
				stdinBytes, err := io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Println("failed to read from stdin:", err)
					os.Exit(1)
				}
				stdinMessage := string(stdinBytes)

				if stdinMessage != "" {
					message = prompt + stdinMessage
				} else {
					message = prompt
				}
			} else {
				// Get message from command line argument if provided
				if len(args) == 1 {
					message = args[0]
				} else {
					// Read from stdin only if no command line argument
					stdinBytes, err := io.ReadAll(os.Stdin)
					if err != nil {
						fmt.Println("failed to read from stdin:", err)
						os.Exit(1)
					}
					stdinMessage := string(stdinBytes)

					if stdinMessage != "" {
						message = stdinMessage
					} else {
						fmt.Println("no input provided via command line argument or stdin")
						os.Exit(1)
					}
				}
			}

			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				fmt.Println("OPENAI_API_KEY is not set")
				os.Exit(1)
			}

			// Parse format string to generate JSON schema
			properties, required, err := parseFormat(format)
			if err != nil {
				fmt.Printf("failed to parse format: %v\n", err)
				os.Exit(1)
			}

			// Build payload per Responses API
			payload := map[string]interface{}{
				"model":        model,
				"instructions": instructions,
				"input":        message,
				"store":        false,
				"text": map[string]interface{}{
					"format": map[string]interface{}{
						"type":   "json_schema",
						"name":   "response",
						"strict": true,
						"schema": map[string]interface{}{
							"type":                 "object",
							"properties":           properties,
							"required":             required,
							"additionalProperties": false,
						},
					},
					"verbosity": verbosity,
				},
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
			textOut, err := parseAPIResponse(respBody)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Printf("%s", textOut)
		},
	}

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
		&prompt,
		"prompt",
		"",
		"prompt text that can be combined with stdin input",
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
