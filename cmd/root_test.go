package cmd

import "testing"

func TestStripMarkdownCodeFences(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "json with language tag",
			in:   "```json\n{\n  \"a\": 1\n}\n```\n",
			want: "{\n  \"a\": 1\n}",
		},
		{
			name: "plain fence no language",
			in:   "```\n{\n  \"a\": 1\n}\n```\n",
			want: "{\n  \"a\": 1\n}",
		},
		{
			name: "no closing fence",
			in:   "```json\n{\n  \"a\": 1\n}",
			want: "```json\n{\n  \"a\": 1\n}",
		},
		{
			name: "incidental fence inside body only",
			in:   "prefix\n```something\nbody\n",
			want: "prefix\n```something\nbody",
		},
		{
			name: "closing fence not last non-empty line",
			in:   "```json\n{\n  \"a\": 1\n}\n```\ntrailer\n",
			want: "```json\n{\n  \"a\": 1\n}\n```\ntrailer",
		},
		{
			name: "trailing blanks after closing fence",
			in:   "```json\n{\n  \"a\": 1\n}\n```\n\n\n",
			want: "{\n  \"a\": 1\n}",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripForJsonMarshal(tc.in)
			if got != tc.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}
