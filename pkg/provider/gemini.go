package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// GeminiProvider implements Provider for Google Gemini GenerateContent API.
type GeminiProvider struct{}

func (p *GeminiProvider) DefaultOptions() Options {
	return Options{
		Model: "gemini-2.0-flash",
		// Leave MaxTokens 0 (unspecified) unless overridden by flag
	}
}

func (p *GeminiProvider) BuildAPIPayload(opts Options) (map[string]interface{}, error) {
	// Build contents with a single user turn.
	contents := []map[string]interface{}{
		{
			"parts": []map[string]interface{}{
				{"text": opts.Message},
			},
		},
	}

	payload := map[string]interface{}{
		// Retain model in payload for BuildAPIRequest to read, but strip before send
		"model":    opts.Model,
		"contents": contents,
	}

	// Optional system instruction
	if strings.TrimSpace(opts.Instructions) != "" {
		payload["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": opts.Instructions},
			},
		}
	}

	// generationConfig: maxOutputTokens, JSON mode, etc.
	genCfg := map[string]interface{}{}

	if opts.MaxTokens > 0 {
		genCfg["maxOutputTokens"] = opts.MaxTokens
	}

	// If properties are provided (via --format), request JSON output.
	if len(opts.Properties) > 0 {
		genCfg["responseMimeType"] = "application/json"
		genCfg["responseSchema"] = buildGeminiObjectSchema(opts.Properties)
	}

	if len(genCfg) > 0 {
		payload["generationConfig"] = genCfg
	}

	return payload, nil
}

// buildGeminiObjectSchema converts our shorthand properties map into
// Gemini's simplified schema representation for JSON mode.
func buildGeminiObjectSchema(properties map[string]interface{}) map[string]interface{} {
	// Collect required keys (all provided properties are required)
	required := make([]string, 0, len(properties))
	convProps := make(map[string]interface{}, len(properties))
	for k, v := range properties {
		required = append(required, k)
		// v is a map with keys like type, items
		if m, ok := v.(map[string]interface{}); ok {
			convProps[k] = convertGeminiSchemaForProperty(m)
		}
	}
	return map[string]interface{}{
		"type":       "OBJECT",
		"properties": convProps,
		"required":   required,
	}
}

func convertGeminiSchemaForProperty(m map[string]interface{}) map[string]interface{} {
	t, _ := m["type"].(string)
	switch strings.ToLower(t) {
	case "array":
		// Items element type
		itemType := ""
		if rawItems, ok := m["items"].(map[string]interface{}); ok {
			if it, ok := rawItems["type"].(string); ok {
				itemType = it
			}
		}
		out := map[string]interface{}{
			"type": "ARRAY",
		}
		if itemType != "" {
			out["items"] = map[string]interface{}{
				"type": toGeminiType(itemType),
			}
		}
		return out
	default:
		return map[string]interface{}{
			"type": toGeminiType(t),
		}
	}
}

func toGeminiType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string":
		return "STRING"
	case "integer":
		return "INTEGER"
	case "number":
		return "NUMBER"
	case "boolean":
		return "BOOLEAN"
	case "object":
		return "OBJECT"
	case "array":
		return "ARRAY"
	default:
		// Fallback: best effort
		if t == "" {
			return "STRING"
		}
		return strings.ToUpper(t)
	}
}

func (p *GeminiProvider) BuildAPIRequest(payload map[string]interface{}, baseURL string, reqOpts RequestOptions) (*http.Request, error) {
	// Extract model for URL path, and remove it from the body payload.
	model, _ := payload["model"].(string)
	delete(payload, "model")

	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("gemini: model is required")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	// Build URL: {base}/v1beta/models/{model}:generateContent?key=API_KEY
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/v1beta/models/" + url.PathEscape(model) + ":generateContent")
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	apiKey := reqOpts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, MissingAPIKeyError{Provider: "gemini", EnvVar: "GEMINI_API_KEY"}
	}

	q := u.Query()
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range reqOpts.ExtraHeaders {
		if k == "" || v == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	return req, nil
}

func (p *GeminiProvider) ParseAPIResponse(respBody []byte) (string, error) {
	// Extract aggregated text across candidate parts.
	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	var b strings.Builder
	if len(apiResp.Candidates) > 0 {
		cand := apiResp.Candidates[0]
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				b.WriteString(part.Text)
			}
		}
	}
	return b.String(), nil
}
