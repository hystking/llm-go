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

// AnthropicProvider implements Provider for Anthropic Messages API.
type AnthropicProvider struct{}

func (p *AnthropicProvider) DefaultOptions() Options {
	model := "claude-3-5-haiku-latest"
	return Options{
		Model:     model,
		MaxTokens: anthropicDefaultMaxTokens(model),
	}
}

func (p *AnthropicProvider) BuildAPIPayload(opts Options) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"model":      opts.Model,
		"max_tokens": opts.MaxTokens,
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

	// If properties are provided, guide Anthropic to return strict JSON.
	if len(opts.Properties) > 0 {
		// Build a concise schema hint for the system prompt.
		keys := make([]string, 0, len(opts.Properties))
		for k := range opts.Properties {
			keys = append(keys, k)
		}
		// Keep deterministic order for tests/logs
		sort.Strings(keys)

		var b strings.Builder
		b.WriteString("Return only a strict JSON object with keys ")
		b.WriteString(strings.Join(keys, ", "))
		b.WriteString(". No prose, no explanations, no markdown. ")
		b.WriteString("All keys are required. Types: ")
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			// Best-effort type description from shorthand
			t := "string"
			if m, ok := opts.Properties[k].(map[string]interface{}); ok {
				if tt, ok := m["type"].(string); ok {
					if strings.EqualFold(tt, "array") {
						if it, ok := m["items"].(map[string]interface{}); ok {
							if itype, ok := it["type"].(string); ok {
								t = fmt.Sprintf("array<%s>", itype)
							}
						}
					} else {
						t = tt
					}
				}
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(t)
		}

		sys := b.String()
		if s, ok := payload["system"].(string); ok && strings.TrimSpace(s) != "" {
			payload["system"] = s + "\n\n" + sys
		} else {
			payload["system"] = sys
		}
	}

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
		return nil, MissingAPIKeyError{Provider: "anthropic", EnvVar: "ANTHROPIC_API_KEY"}
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

// anthropicDefaultMaxTokens returns a default max_tokens per model family
// based on Anthropic's Models overview page.
func anthropicDefaultMaxTokens(model string) int {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus-4-1"):
		return 32_000
	case strings.Contains(m, "opus-4"):
		return 32_000
	case strings.Contains(m, "sonnet-4-0") || strings.Contains(m, "sonnet-4"):
		return 64_000
	case strings.Contains(m, "3-7-sonnet"):
		return 64_000
	case strings.Contains(m, "3-5-sonnet"):
		return 8_192
	case strings.Contains(m, "3-5-haiku") || strings.Contains(m, "haiku-latest"):
		return 8_192
	case strings.Contains(m, "3-haiku"):
		return 4_096
	default:
		// conservative lower bound to avoid exceeding max output for smaller models
		return 4_096
	}
}
