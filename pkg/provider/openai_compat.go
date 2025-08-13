package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// OpenAICompatProvider implements Provider for OpenAI-compatible Chat Completions API.
type OpenAICompatProvider struct{}

func (p *OpenAICompatProvider) DefaultOptions() Options {
	return Options{
		Model: "gpt-4o-mini",
	}
}

func (p *OpenAICompatProvider) BuildAPIPayload(opts Options) (map[string]interface{}, error) {
	// Build messages: optional system with instructions (+ schema hint), then user message
	messages := make([]map[string]interface{}, 0, 2)

	// Merge instruction with strict JSON hint if properties exist.
	if sys := buildStrictJSONSystem(opts.Properties, opts.Instructions); strings.TrimSpace(sys) != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": sys,
		})
	}

	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": opts.Message,
	})

	payload := map[string]interface{}{
		"model":    opts.Model,
		"messages": messages,
		// Keep default n=1; no streaming
	}

	if opts.MaxTokens > 0 {
		// Use widely supported field for compatibility
		payload["max_tokens"] = opts.MaxTokens
	}

	return payload, nil
}

func (p *OpenAICompatProvider) BuildAPIRequest(payload map[string]interface{}, baseURL string, reqOpts RequestOptions) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	req, err := http.NewRequest("POST", strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
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
		return nil, MissingAPIKeyError{Provider: "openai-compat", EnvVar: "OPENAI_API_KEY"}
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

func (p *OpenAICompatProvider) ParseAPIResponse(respBody []byte) (string, error) {
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return apiResp.Choices[0].Message.Content, nil
}
