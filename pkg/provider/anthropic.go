package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// AnthropicProvider implements Provider for Anthropic Messages API.
type AnthropicProvider struct{}

func (p *AnthropicProvider) BuildAPIPayload(opts Options) (map[string]interface{}, error) {
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = "claude-3-5-haiku-latest"
	}
	// Minimal mapping: one-turn user message and optional system prompt.
	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": 8_192, // sensible default if user didn't specify
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": opts.Message,
			},
		},
	}

	if strings.TrimSpace(opts.Instructions) != "" {
		// Map instructions to Anthropic's top-level system prompt.
		payload["system"] = opts.Instructions
	}

	// Note: opts.Properties is ignored for Anthropic in this minimal implementation.
	// Advanced: could translate to tools or structured output guidance via system prompt.

	return payload, nil
}

func (p *AnthropicProvider) BuildAPIRequest(payload map[string]interface{}, baseURL string, reqOpts RequestOptions) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	// Choose default baseURL for Anthropic if not provided.
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	req, err := http.NewRequest("POST", strings.TrimRight(baseURL, "/")+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	apiKey := reqOpts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	req.Header.Set("x-api-key", apiKey)

	for k, v := range reqOpts.ExtraHeaders {
		if k == "" || v == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	return req, nil
}

func (p *AnthropicProvider) ParseAPIResponse(respBody []byte) (string, error) {
	// Aggregate all text content blocks.
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	var b strings.Builder
	for _, c := range apiResp.Content {
		if c.Type == "text" && c.Text != "" {
			b.WriteString(c.Text)
		}
	}
	return b.String(), nil
}
