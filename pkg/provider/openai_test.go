package provider

import "testing"

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
