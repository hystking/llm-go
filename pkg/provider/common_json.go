package provider

import (
	"fmt"
	"sort"
	"strings"
)

// buildStrictJSONSystem returns a unified system instruction string that
// enforces strict JSON output. It merges the given instruction with a concise
// schema hint derived from properties. The function accepts properties first
// to match repository conventions.
func buildStrictJSONSystem(properties map[string]interface{}, instruction string) string {
	instr := strings.TrimSpace(instruction)

	// If no properties, only return the original instruction.
	if len(properties) == 0 {
		return instr
	}

	// Build a concise schema hint for fields and types based on provided
	// properties. Keep ordering deterministic for tests and logs.
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("RETURN ONLY A STRICT JSON OBJECT. NO PROSE, NO EXPLANATIONS, NO MARKDOWN.\n")
	b.WriteString("Fields (all required): ")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		// Best-effort type extraction from shorthand
		t := "string"
		if m, ok := properties[k].(map[string]interface{}); ok {
			if tt, ok := m["type"].(string); ok {
				if strings.EqualFold(tt, "array") {
					if it, ok := m["items"].(map[string]interface{}); ok {
						if itype, ok := it["type"].(string); ok {
							t = fmt.Sprintf("array<%s>", itype)
						}
					}
				} else {
					t = tt
				}
			}
		}
		b.WriteString(k + ": " + t)
	}

	schemaHint := b.String()
	if instr == "" {
		return schemaHint
	}
	return instr + "\n\n" + schemaHint
}
