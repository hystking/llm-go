package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestAnthropicProvider_ParseAPIResponse(t *testing.T) {
	p := &AnthropicProvider{}
	tests := []struct {
		name    string
		body    []byte
		want    string
		wantErr bool
	}{
		{
			name: "single text block",
			body: []byte(`{"content":[{"type":"text","text":"Hello Claude"}]}`),
			want: "Hello Claude",
		},
		{
			name: "multiple blocks including non-text",
			body: []byte(`{"content":[{"type":"text","text":"Hello "},{"type":"tool_use","id":"x"},{"type":"text","text":"World"}]}`),
			want: "Hello World",
		},
		{
			name:    "invalid json",
			body:    []byte(`invalid`),
			wantErr: true,
		},
		{
			name: "empty content",
			body: []byte(`{"content":[]}`),
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

func TestAnthropicProvider_BuildAPIPayload_Defaults(t *testing.T) {
	p := &AnthropicProvider{}
	// Use provider defaults for model and max_tokens
	payload, err := p.BuildAPIPayload(Options{Model: "claude-3-5-haiku-latest", MaxTokens: 8_192, Message: "Hello", Instructions: "sys"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if payload["model"] != "claude-3-5-haiku-latest" {
		t.Fatalf("model fallback mismatch: %v", payload["model"])
	}
	if payload["max_tokens"] != 8_192 {
		t.Fatalf("max_tokens default mismatch: %v", payload["max_tokens"])
	}
	msgs, ok := payload["messages"].([]map[string]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages wrong type/len: %T %v", payload["messages"], payload["messages"])
	}
	if msgs[0]["role"] != "user" || msgs[0]["content"] != "Hello" {
		t.Fatalf("message content mismatch: %v", msgs[0])
	}
	if payload["system"] != "sys" {
		t.Fatalf("system mismatch: %v", payload["system"])
	}
}

func TestAnthropicProvider_BuildAPIRequest_DefaultsAndHeaders(t *testing.T) {
	p := &AnthropicProvider{}
	payload := map[string]interface{}{"model": "claude-3-5-haiku-latest", "messages": []map[string]interface{}{}}
	req, err := p.BuildAPIRequest(payload, "", RequestOptions{APIKey: "anth-key"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method mismatch: %s", req.Method)
	}
	if req.URL.String() != "https://api.anthropic.com/v1/messages" {
		t.Fatalf("url mismatch: %s", req.URL.String())
	}
	if req.Header.Get("x-api-key") != "anth-key" {
		t.Fatalf("api key header mismatch: %s", req.Header.Get("x-api-key"))
	}
	if req.Header.Get("anthropic-version") == "" {
		t.Fatalf("anthropic-version header missing")
	}
	if req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" {
		t.Fatalf("content headers mismatch")
	}
	b, _ := io.ReadAll(req.Body)
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if got["model"] != "claude-3-5-haiku-latest" {
		t.Fatalf("body model mismatch: %v", got["model"])
	}
}
