package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestGeminiProvider_ParseAPIResponse(t *testing.T) {
	p := &GeminiProvider{}
	tests := []struct {
		name    string
		body    []byte
		want    string
		wantErr bool
	}{
		{
			name: "single candidate, multiple parts",
			body: []byte(`{"candidates":[{"content":{"parts":[{"text":"Hello "},{"text":"Gemini"}]}}]}`),
			want: "Hello Gemini",
		},
		{
			name: "uses only first candidate",
			body: []byte(`{"candidates":[{"content":{"parts":[{"text":"First"}]}},{"content":{"parts":[{"text":"Second"}]}}]}`),
			want: "First",
		},
		{
			name:    "invalid json",
			body:    []byte(`invalid`),
			wantErr: true,
		},
		{
			name: "empty",
			body: []byte(`{"candidates":[]}`),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseAPIResponse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeminiProvider_BuildAPIPayload_DefaultsAndSchema(t *testing.T) {
	p := &GeminiProvider{}
	opts := Options{
		Model:        "gemini-2.0-flash",
		Message:      "Hello",
		Instructions: "be concise",
		MaxTokens:    321,
		Properties: map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"tags": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
	}
	payload, err := p.BuildAPIPayload(opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if payload["model"] != "gemini-2.0-flash" {
		t.Fatalf("model mismatch: %v", payload["model"])
	}

	// contents check
	contents, ok := payload["contents"].([]map[string]interface{})
	if !ok || len(contents) != 1 {
		t.Fatalf("contents wrong type/len: %T %v", payload["contents"], payload["contents"])
	}
	parts, ok := contents[0]["parts"].([]map[string]interface{})
	if !ok || len(parts) != 1 || parts[0]["text"] != "Hello" {
		t.Fatalf("parts mismatch: %v", contents[0]["parts"])
	}

	// systemInstruction check
	sys, ok := payload["systemInstruction"].(map[string]interface{})
	if !ok {
		t.Fatalf("systemInstruction missing")
	}
	sparts, ok := sys["parts"].([]map[string]interface{})
	if !ok || len(sparts) != 1 || sparts[0]["text"] != "be concise" {
		t.Fatalf("systemInstruction parts mismatch: %v", sys)
	}

	// generationConfig check
	gen, ok := payload["generationConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("generationConfig missing")
	}
	if gen["responseMimeType"] != "application/json" {
		t.Fatalf("responseMimeType mismatch: %v", gen["responseMimeType"])
	}
	if mv := gen["maxOutputTokens"]; mv != nil {
		switch v := mv.(type) {
		case int:
			if v != 321 {
				t.Fatalf("maxOutputTokens mismatch: %v", v)
			}
		case int64:
			if v != 321 {
				t.Fatalf("maxOutputTokens mismatch: %v", v)
			}
		case float64:
			if int(v) != 321 {
				t.Fatalf("maxOutputTokens mismatch: %v", v)
			}
		default:
			t.Fatalf("maxOutputTokens unexpected type: %T", v)
		}
	} else {
		t.Fatalf("maxOutputTokens missing in generationConfig")
	}
	schema, ok := gen["responseSchema"].(map[string]interface{})
	if !ok {
		t.Fatalf("responseSchema missing")
	}
	if strings.ToUpper(schema["type"].(string)) != "OBJECT" {
		t.Fatalf("schema type mismatch: %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema properties missing")
	}
	// name -> STRING
	if pname, ok := props["name"].(map[string]interface{}); !ok || strings.ToUpper(pname["type"].(string)) != "STRING" {
		t.Fatalf("name property type mismatch: %v", props["name"])
	}
	// tags -> ARRAY(items STRING)
	ptags, ok := props["tags"].(map[string]interface{})
	if !ok || strings.ToUpper(ptags["type"].(string)) != "ARRAY" {
		t.Fatalf("tags type mismatch: %v", props["tags"])
	}
	items, _ := ptags["items"].(map[string]interface{})
	if strings.ToUpper(items["type"].(string)) != "STRING" {
		t.Fatalf("tags.items.type mismatch: %v", items["type"])
	}
	// required should include both keys (order not guaranteed)
	have := map[string]bool{}
	switch rv := schema["required"].(type) {
	case []interface{}:
		for _, r := range rv {
			if s, ok := r.(string); ok {
				have[s] = true
			}
		}
	case []string:
		for _, s := range rv {
			have[s] = true
		}
	default:
		t.Fatalf("required missing or wrong type: %T", schema["required"])
	}
	if !have["name"] || !have["tags"] {
		t.Fatalf("required should include name and tags: %v", schema["required"])
	}

	// When MaxTokens is zero, maxOutputTokens should not appear
	payload, err = p.BuildAPIPayload(Options{Model: "gemini-2.0-flash", Message: "x"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gc, ok := payload["generationConfig"].(map[string]interface{}); ok {
		if _, exists := gc["maxOutputTokens"]; exists {
			t.Fatalf("maxOutputTokens should be omitted when zero")
		}
	}
}

func TestGeminiProvider_BuildAPIRequest_DefaultsAndHeaders(t *testing.T) {
	p := &GeminiProvider{}
	// minimal payload with model present
	payload := map[string]interface{}{
		"model":    "gemini-2.0-flash",
		"contents": []map[string]interface{}{},
	}

	req, err := p.BuildAPIRequest(payload, "", RequestOptions{APIKey: "gk-test"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method mismatch: %s", req.Method)
	}

	// URL should be default base with model path and key query param
	u, _ := url.Parse(req.URL.String())
	if !strings.Contains(u.Path, "/v1beta/models/gemini-2.0-flash:generateContent") {
		t.Fatalf("url path mismatch: %s", u.Path)
	}
	if u.Query().Get("key") != "gk-test" {
		t.Fatalf("missing or wrong key query param")
	}

	// Headers
	if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" {
		t.Fatalf("headers mismatch")
	}

	// Body must not contain the model field (it is removed before send)
	b, _ := io.ReadAll(req.Body)
	var body map[string]interface{}
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if _, exists := body["model"]; exists {
		t.Fatalf("body should not contain model field")
	}
}
