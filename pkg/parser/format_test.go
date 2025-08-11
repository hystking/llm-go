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
		wantErr        bool
	}{
		{
			name:           "empty format returns empty properties",
			format:         "",
			wantProperties: map[string]interface{}{},
			wantErr:        false,
		},
		{
			name:   "simple string field",
			format: "name:string",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantErr: false,
		},
		{
			name:   "multiple fields",
			format: "name:string,age:integer,active:boolean",
			wantProperties: map[string]interface{}{
				"name":   map[string]interface{}{"type": "string"},
				"age":    map[string]interface{}{"type": "integer"},
				"active": map[string]interface{}{"type": "boolean"},
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
		},
		{
			name:   "omitted type defaults to string",
			format: "invalid",
			wantProperties: map[string]interface{}{
				"invalid": map[string]interface{}{"type": "string"},
			},
			wantErr: false,
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
			wantErr: false,
		},
		{
			name:   "trailing colon becomes string",
			format: "name:",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantErr: false,
		},
		{
			name:    "empty element type in array[]",
			format:  "tags:[]",
			wantErr: true,
		},
		{
			name:   "whitespace around keys and types",
			format: " name : string , age : integer ",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
				"age":  map[string]interface{}{"type": "integer"},
			},
			wantErr: false,
		},
		{
			name:    "trailing comma is invalid",
			format:  "name:string,",
			wantErr: true,
		},
		{
			name:    "leading comma is invalid",
			format:  ",name:string",
			wantErr: true,
		},
		{
			name:    "empty pair between commas is invalid",
			format:  "name:string, ,age:integer",
			wantErr: true,
		},
		{
			name:   "duplicate keys last one wins",
			format: "a:string,a:integer",
			wantProperties: map[string]interface{}{
				"a": map[string]interface{}{"type": "integer"},
			},
			wantErr: false,
		},
		{
			name:   "array type with space before brackets",
			format: "tags: string[]",
			wantProperties: map[string]interface{}{
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "trailing spaces after colon default string",
			format: "name:   ",
			wantProperties: map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
			},
			wantErr: false,
		},
		{
			name:    "whitespace-only format is invalid",
			format:  "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProperties, err := ParseFormat(tt.format)

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
		})
	}
}
