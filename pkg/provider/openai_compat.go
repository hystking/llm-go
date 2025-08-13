package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
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

	// If properties exist, craft a concise schema hint to nudge strict JSON output
	var schemaHint string
	if len(opts.Properties) > 0 {
		keys := make([]string, 0, len(opts.Properties))
		for k := range opts.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("RETURN ONLY A STRICT JSON OBJECT. NO PROSE, NO MARKDOWN.\\n")
		b.WriteString("Fields (all required): ")
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			// best-effort type label
			t := "string"
			if m, ok := opts.Properties[k].(map[string]interface{}); ok {
				if tt, ok := m["type"].(string); ok {
					if strings.EqualFold(tt, "array") {
						if it, ok := m["items"].(map[string]interface{}); ok {
							if itype, ok := it["type"].(string); ok {
								t = "array<" + itype + ">"
							}
						}
					} else {
						t = tt
					}
				}
			}
			b.WriteString(k + ": " + t)
		}
		schemaHint = b.String()
	}

	sys := strings.TrimSpace(opts.Instructions)
	if schemaHint != "" {
		if sys != "" {
			sys = sys + "\n\n" + schemaHint
		} else {
			sys = schemaHint
		}
	}
	if sys != "" {
		// Use system role for wide compatibility
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
