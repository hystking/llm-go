package provider

import "testing"

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
