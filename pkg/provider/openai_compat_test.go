package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAICompatProvider_ParseAPIResponse(t *testing.T) {
	p := &OpenAICompatProvider{}
	body := []byte(`{
        "id": "chatcmpl-xyz",
        "object": "chat.completion",
        "created": 1741569952,
        "model": "gpt-4o-2025-04-14",
        "choices": [
          {
            "index": 0,
            "message": {"role": "assistant", "content": "Hello!"},
            "finish_reason": "stop"
          }
        ]
      }`)
	got, err := p.ParseAPIResponse(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "Hello!" {
		t.Fatalf("got %q, want %q", got, "Hello!")
	}
}

// Note: some OpenAI-compatible providers may return content wrapped in markdown
// code fences. We deliberately do not strip fences here; the CLI will error if
// the model does not return strict JSON as instructed.

func TestOpenAICompatProvider_BuildAPIPayload_SchemaAndMessages(t *testing.T) {
	p := &OpenAICompatProvider{}
	opts := Options{
		Model:        "gpt-4o-mini",
		Instructions: "be brief",
		Message:      "Hello",
		Properties: map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
			"error":   map[string]interface{}{"type": "string"},
		},
		MaxTokens: 321,
	}
	payload, err := p.BuildAPIPayload(opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if payload["model"] != opts.Model {
		t.Fatalf("model mismatch: %v", payload["model"])
	}
	// messages
	msgs, ok := payload["messages"].([]map[string]interface{})
	if !ok {
		// Some JSON marshaling paths may coerce types; try generic slice
		raw, ok2 := payload["messages"].([]interface{})
		if !ok2 {
			t.Fatalf("messages missing or wrong type: %T", payload["messages"])
		}
		if len(raw) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(raw))
		}
		// Check first is system containing strict JSON hint
		m0, _ := raw[0].(map[string]interface{})
		if m0["role"] != "system" {
			t.Fatalf("first role should be system: %v", m0["role"])
		}
		if c, _ := m0["content"].(string); !strings.Contains(c, "RETURN ONLY A STRICT JSON OBJECT") {
			t.Fatalf("system content missing strict JSON hint: %q", c)
		}
		// Check second is user with given message
		m1, _ := raw[1].(map[string]interface{})
		if m1["role"] != "user" || m1["content"] != "Hello" {
			t.Fatalf("user message mismatch: %v", m1)
		}
	} else {
		if len(msgs) < 2 {
			t.Fatalf("expected at least 2 messages, got %d", len(msgs))
		}
	}

	if payload["max_tokens"] != 321 {
		t.Fatalf("expected max_tokens=321, got %v", payload["max_tokens"])
	}
}

func TestOpenAICompatProvider_BuildAPIRequest(t *testing.T) {
	p := &OpenAICompatProvider{}
	payload := map[string]interface{}{"model": "gpt-4o-mini", "messages": []interface{}{}}
	req, err := p.BuildAPIRequest(payload, "", RequestOptions{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method mismatch: %s", req.Method)
	}
	if req.URL.String() != "https://api.openai.com/v1/chat/completions" {
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
	if got["model"] != "gpt-4o-mini" {
		t.Fatalf("body model mismatch: %v", got["model"])
	}
}
