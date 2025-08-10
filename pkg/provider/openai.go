package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// OpenAIProvider implements Provider for OpenAI Responses API.
type OpenAIProvider struct{}

func (p *OpenAIProvider) DefaultOptions() Options {
	return Options{
		Model: "gpt-5-nano",
	}
}

func (p *OpenAIProvider) BuildAPIPayload(opts Options) (map[string]interface{}, error) {
	textPayload := map[string]interface{}{
		"verbosity": opts.Verbosity,
	}

	if len(opts.Properties) > 0 {
		// Build JSON schema from provided properties and mark all as required
		// Collect keys as required
		required := make([]string, 0, len(opts.Properties))
		for k := range opts.Properties {
			required = append(required, k)
		}
		textPayload["format"] = map[string]interface{}{
			"type":   "json_schema",
			"name":   "response",
			"strict": true,
			"schema": map[string]interface{}{
				"type":                 "object",
				"properties":           opts.Properties,
				"required":             required,
				"additionalProperties": false,
			},
		}
	}

	payload := map[string]interface{}{
		"model":        opts.Model,
		"instructions": opts.Instructions,
		"input":        opts.Message,
		"store":        false,
		"text":         textPayload,
		"reasoning": map[string]interface{}{
			"effort": opts.ReasoningEffort,
		},
	}

	if opts.MaxTokens > 0 {
		payload["max_output_tokens"] = opts.MaxTokens
	}

	return payload, nil
}

func (p *OpenAIProvider) BuildAPIRequest(payload map[string]interface{}, baseURL string, reqOpts RequestOptions) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	req, err := http.NewRequest("POST", strings.TrimRight(baseURL, "/")+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	apiKey := reqOpts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	for k, v := range reqOpts.ExtraHeaders {
		if k == "" || v == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	return req, nil
}

func (p *OpenAIProvider) ParseAPIResponse(respBody []byte) (string, error) {
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
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

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

	return textOut, nil
}
