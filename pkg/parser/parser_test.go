package parser

import (
	"reflect"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		wantProperties map[string]interface{}
		wantRequired   []string
		wantErr        bool
	}{
		{
			name:   "empty format returns default",
			format: "",
			wantProperties: map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
			wantRequired: []string{"message"},
			wantErr:      false,
		},
		{
			name:   "simple string field",
			format: "name:string",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantRequired: []string{"name"},
			wantErr:      false,
		},
		{
			name:   "multiple fields",
			format: "name:string,age:integer,active:boolean",
			wantProperties: map[string]interface{}{
				"name":   map[string]interface{}{"type": "string"},
				"age":    map[string]interface{}{"type": "integer"},
				"active": map[string]interface{}{"type": "boolean"},
			},
			wantRequired: []string{"name", "age", "active"},
			wantErr:      false,
		},
		{
			name:   "array field",
			format: "tags:string[]",
			wantProperties: map[string]interface{}{
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantRequired: []string{"tags"},
			wantErr:      false,
		},
		{
			name:   "mixed fields with array",
			format: "name:string,tags:string[],count:integer",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"count": map[string]interface{}{"type": "integer"},
			},
			wantRequired: []string{"name", "tags", "count"},
			wantErr:      false,
		},
		{
			name:   "array with number elements",
			format: "scores:number[]",
			wantProperties: map[string]interface{}{
				"scores": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
			wantRequired: []string{"scores"},
			wantErr:      false,
		},
		{
			name:   "omitted type defaults to string",
			format: "invalid",
			wantProperties: map[string]interface{}{
				"invalid": map[string]interface{}{"type": "string"},
			},
			wantRequired: []string{"invalid"},
			wantErr:      false,
		},
		{
			name:    "empty key",
			format:  ":string",
			wantErr: true,
		},
		{
			name:    "nested type[] arrays not supported",
			format:  "tags:string[][]",
			wantErr: true,
		},
		{
			name:    "multiple colons in field",
			format:  "name:string:string",
			wantErr: true,
		},
		{
			name:   "key only becomes string",
			format: "name",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantRequired: []string{"name"},
			wantErr:      false,
		},
		{
			name:   "trailing colon becomes string",
			format: "name:",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantRequired: []string{"name"},
			wantErr:      false,
		},
		{
			name:    "empty element type in array[]",
			format:  "tags:[]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProperties, gotRequired, err := ParseFormat(tt.format)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(gotProperties, tt.wantProperties) {
				t.Errorf("ParseFormat() gotProperties = %v, want %v", gotProperties, tt.wantProperties)
			}

			if !reflect.DeepEqual(gotRequired, tt.wantRequired) {
				t.Errorf("ParseFormat() gotRequired = %v, want %v", gotRequired, tt.wantRequired)
			}
		})
	}
}

func TestParseAPIResponse(t *testing.T) {
	tests := []struct {
		name     string
		respBody []byte
		want     string
		wantErr  bool
	}{
		{
			name: "response with output_text field",
			respBody: []byte(`{
				"output_text": "Hello World",
				"output": []
			}`),
			want:    "Hello World",
			wantErr: false,
		},
		{
			name: "response with output array fallback",
			respBody: []byte(`{
				"output": [
					{
						"content": [
							{
								"type": "output_text",
								"text": "Fallback text"
							}
						]
					}
				]
			}`),
			want:    "Fallback text",
			wantErr: false,
		},
		{
			name: "response with multiple content items",
			respBody: []byte(`{
				"output": [
					{
						"content": [
							{
								"type": "other",
								"text": "Should ignore this"
							},
							{
								"type": "output_text",
								"text": "Should use this"
							}
						]
					}
				]
			}`),
			want:    "Should use this",
			wantErr: false,
		},
		{
			name: "response with both fields prefers output_text",
			respBody: []byte(`{
				"output_text": "Preferred text",
				"output": [
					{
						"content": [
							{
								"type": "output_text",
								"text": "Fallback text"
							}
						]
					}
				]
			}`),
			want:    "Preferred text",
			wantErr: false,
		},
		{
			name: "empty response",
			respBody: []byte(`{
				"output": []
			}`),
			want:    "",
			wantErr: false,
		},
		{
			name: "no matching content type",
			respBody: []byte(`{
				"output": [
					{
						"content": [
							{
								"type": "other_type",
								"text": "Wrong type"
							}
						]
					}
				]
			}`),
			want:    "",
			wantErr: false,
		},
		{
			name:     "invalid JSON",
			respBody: []byte(`invalid json`),
			want:     "",
			wantErr:  true,
		},
		{
			name:     "empty JSON",
			respBody: []byte(`{}`),
			want:     "",
			wantErr:  false,
		},
		{
			name: "multiple output items",
			respBody: []byte(`{
				"output": [
					{
						"content": [
							{
								"type": "other",
								"text": "Wrong type"
							}
						]
					},
					{
						"content": [
							{
								"type": "output_text",
								"text": "Found it"
							}
						]
					}
				]
			}`),
			want:    "Found it",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAPIResponse(tt.respBody)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAPIResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ParseAPIResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}
