package provider

import "testing"

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
)

func TestOpenAIProvider_ParseAPIResponse(t *testing.T) {
	p := &OpenAIProvider{}
	tests := []struct {
		name    string
		body    []byte
		want    string
		wantErr bool
	}{
		{
			name: "output_text present",
			body: []byte(`{"output_text":"Hello","output":[]}`),
			want: "Hello",
		},
		{
			name: "fallback to output[].content[].text",
			body: []byte(`{"output":[{"content":[{"type":"output_text","text":"Fallback"}]}]}`),
			want: "Fallback",
		},
		{
			name:    "invalid json",
			body:    []byte(`invalid`),
			wantErr: true,
		},
		{
			name: "no text",
			body: []byte(`{"output":[]}`),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.ParseAPIResponse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: %v, wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_BuildAPIPayload_DefaultsAndSchema(t *testing.T) {
	p := &OpenAIProvider{}
	opts := Options{
		Model:           "", // should fallback to gpt-5-nano
		Instructions:    "be brief",
		Message:         "Hello",
		Verbosity:       "low",
		ReasoningEffort: "minimal",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "integer"},
		},
	}
	payload, err := p.BuildAPIPayload(opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if got := payload["model"]; got != "gpt-5-nano" {
		t.Fatalf("model fallback mismatch: %v", got)
	}
	if got := payload["instructions"]; got != "be brief" {
		t.Fatalf("instructions mismatch: %v", got)
	}
	if got := payload["input"]; got != "Hello" {
		t.Fatalf("input mismatch: %v", got)
	}
	text, ok := payload["text"].(map[string]interface{})
	if !ok {
		t.Fatalf("text missing or wrong type")
	}
	if text["verbosity"] != "low" {
		t.Fatalf("verbosity mismatch: %v", text["verbosity"])
	}
	// Check schema wrapper
	format, ok := text["format"].(map[string]interface{})
	if !ok {
		t.Fatalf("format missing or wrong type")
	}
	if format["type"] != "json_schema" || format["strict"] != true {
		t.Fatalf("format header mismatch: %v", format)
	}
	schema, ok := format["schema"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema missing or wrong type")
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type mismatch: %v", schema["type"])
	}
	// properties should match opts.Properties
	if !reflect.DeepEqual(schema["properties"], opts.Properties) {
		t.Fatalf("properties mismatch: got=%v want=%v", schema["properties"], opts.Properties)
	}
	// required must include all keys (order independent)
    gotSet := map[string]bool{}
    switch rv := schema["required"].(type) {
    case []interface{}:
        for _, v := range rv {
            if s, ok := v.(string); ok {
                gotSet[s] = true
            }
        }
    case []string:
        for _, s := range rv {
            gotSet[s] = true
        }
    default:
        t.Fatalf("required missing or wrong type: %T", schema["required"])
    }
	for k := range opts.Properties {
		if !gotSet[k] {
			t.Fatalf("required missing key: %s", k)
		}
	}
}

func TestOpenAIProvider_BuildAPIRequest_DefaultsAndHeaders(t *testing.T) {
	p := &OpenAIProvider{}
	payload := map[string]interface{}{"model": "gpt-5-nano"}
	req, err := p.BuildAPIRequest(payload, "", RequestOptions{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method mismatch: %s", req.Method)
	}
	if req.URL.String() != "https://api.openai.com/v1/responses" {
		t.Fatalf("url mismatch: %s", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer sk-test" {
		t.Fatalf("auth header mismatch: %s", req.Header.Get("Authorization"))
	}
	if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" {
		t.Fatalf("content headers mismatch")
	}
	// body decodes as JSON
	b, _ := io.ReadAll(req.Body)
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if got["model"] != "gpt-5-nano" {
		t.Fatalf("body model mismatch: %v", got["model"])
	}
}
