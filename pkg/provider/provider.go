package provider

import (
	"errors"
	"fmt"
	"net/http"
)

// Options represents common inputs to build an API payload.
type Options struct {
	Model           string
	Instructions    string
	Message         string
	Verbosity       string
	ReasoningEffort string
	// Properties holds the parsed properties map from CLI (--format shorthand).
	// Providers wrap this into their schema representation and mark all keys required.
	Properties map[string]interface{}
	// MaxTokens is the provider-specific maximum output tokens, if applicable
	// (e.g., Anthropic Messages API). 0 means unspecified.
	MaxTokens int
}

// RequestOptions represents options for building an HTTP request.
type RequestOptions struct {
	// APIKey is optional. If empty, provider may resolve from env.
	APIKey string
	// ExtraHeaders allows provider-agnostic additions.
	ExtraHeaders map[string]string
}

// Provider abstracts LLM API differences.
type Provider interface {
	// DefaultOptions returns provider-specific default options (e.g., model, max tokens).
	DefaultOptions() Options
	// BuildAPIPayload builds a provider-specific payload from options.
	BuildAPIPayload(opts Options) (map[string]interface{}, error)
	// BuildAPIRequest creates the HTTP request to send the payload.
	BuildAPIRequest(payload map[string]interface{}, baseURL string, reqOpts RequestOptions) (*http.Request, error)
	// ParseAPIResponse extracts the text output from raw response bytes.
	ParseAPIResponse(respBody []byte) (string, error)
}

// Factory returns the Provider implementation by name.
func New(name string) (Provider, error) {
	switch name {
	case "openai", "oa", "default", "":
		return &OpenAIProvider{}, nil
	case "anthropic", "claude", "anth":
		return &AnthropicProvider{}, nil
	case "gemini", "google", "gai":
		return &GeminiProvider{}, nil
	default:
		return nil, ErrUnknownProvider{name: name}
	}
}

// ErrUnknownProvider indicates an unsupported provider name.
type ErrUnknownProvider struct{ name string }

func (e ErrUnknownProvider) Error() string { return "unknown provider: " + e.name }

// ErrMissingAPIKey is a sentinel error that indicates the API key is missing.
var ErrMissingAPIKey = errors.New("missing API key")

// MissingAPIKeyError includes details about which provider/env var is missing.
// It unwraps to ErrMissingAPIKey so callers can use errors.Is/As.
type MissingAPIKeyError struct {
	Provider string
	EnvVar   string
}

func (e MissingAPIKeyError) Error() string {
	if e.Provider != "" && e.EnvVar != "" {
		return fmt.Sprintf("%s: %s is not set", e.Provider, e.EnvVar)
	}
	if e.EnvVar != "" {
		return fmt.Sprintf("%s is not set", e.EnvVar)
	}
	return ErrMissingAPIKey.Error()
}

func (e MissingAPIKeyError) Unwrap() error { return ErrMissingAPIKey }
