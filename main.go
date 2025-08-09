package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// parseFormat parses a format string like "key1:type,key2:type,..." into a JSON schema
// Supports array types with element specifications: "key:array[element_type]"
func parseFormat(format string) (map[string]interface{}, []string, error) {
	if format == "" {
		// Default format
		return map[string]interface{}{
			"message": map[string]interface{}{
				"type": "string",
			},
		}, []string{"message"}, nil
	}

	properties := make(map[string]interface{})
	var required []string

	pairs := strings.Split(format, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid format pair: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		typeStr := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, nil, fmt.Errorf("empty key in format pair: %s", pair)
		}

		// Check for array[element_type] format
		if strings.HasPrefix(typeStr, "array[") && strings.HasSuffix(typeStr, "]") {
			// Extract element type from array[element_type]
			elementType := typeStr[6 : len(typeStr)-1] // Remove "array[" and "]"
			elementType = strings.TrimSpace(elementType)
			
			if elementType == "" {
				return nil, nil, fmt.Errorf("empty element type in array specification: %s", typeStr)
			}

			properties[key] = map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": elementType,
				},
			}
		} else {
			properties[key] = map[string]interface{}{"type": typeStr}
		}
		
		required = append(required, key)
	}

	return properties, required, nil
}

func main() {
	var (
		model           string
		reasoningEffort string
		verbosity       string
		instructions    string
		format          string
	)

	var baseURL string

	var rootCmd = &cobra.Command{
		Use:   "llm [flags] [\"your message\"]",
		Short: "Send a message to the LLM API",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var message string
			var err error

			// Read from stdin if available
			stdinBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Println("failed to read from stdin:", err)
				os.Exit(1)
			}
			stdinMessage := string(stdinBytes)

			// Get message from command line argument if provided
			var argMessage string
			if len(args) == 1 {
				argMessage = args[0]
			}

			// Concatenate stdin and argument if both are provided
			if stdinMessage != "" && argMessage != "" {
				message = stdinMessage + argMessage
			} else if argMessage != "" {
				message = argMessage
			} else if stdinMessage != "" {
				message = stdinMessage
			} else {
				fmt.Println("no input provided via command line argument or stdin")
				os.Exit(1)
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

			// Parse Responses API shape
			var apiResp struct {
				OutputText string `json:"output_text"`
				Output     []struct {
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"output"`
			}
			if err := json.Unmarshal(respBody, &apiResp); err != nil {
				fmt.Println("failed to parse response:", err)
				os.Exit(1)
			}

			// Prefer the convenience field, then fall back to output[].content[].text
			textOut := apiResp.OutputText
			if textOut == "" {
				for _, item := range apiResp.Output {
					for _, c := range item.Content {
						if c.Type == "output_text" && c.Text != "" {
							textOut = c.Text
							break
						}
					}
					if textOut != "" {
						break
					}
				}
			}

			fmt.Printf(textOut)
		},
	}

	rootCmd.Flags().StringVar(&model, "model", "gpt-5-nano", "model name")
	// Empty by default to avoid sending to non-reasoning models. Set to low/medium/high to enable.
	rootCmd.Flags().StringVar(&reasoningEffort, "reasoning_effort", "low", "reasoning effort (low/medium/high)")
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

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
