package parser

import (
	"fmt"
	"strings"
)

// ParseFormat parses a format string like "key1:type,key2:type,..." into properties map.
// Supports array types: "key:type[]". All fields are considered required by consumers.
func ParseFormat(format string) (map[string]interface{}, error) {
	if format == "" {
		return make(map[string]interface{}), nil
	}

	properties := make(map[string]interface{})

	pairs := strings.Split(format, ",")
	for _, pair := range pairs {
		trimmed := strings.TrimSpace(pair)
		if trimmed == "" {
			return nil, fmt.Errorf("invalid format pair: %s", pair)
		}

		parts := strings.SplitN(trimmed, ":", 2)
		key := strings.TrimSpace(parts[0])
		keyIsArray := false
		if strings.HasSuffix(key, "[]") {
			keyIsArray = true
			key = strings.TrimSpace(strings.TrimSuffix(key, "[]"))
		}
		// default type to string when omitted or empty (e.g., "name" or "name:")
		typeStr := "string"
		if len(parts) == 2 {
			// if extra colon remains in type portion, treat as error (e.g., name:string:string)
			if strings.Contains(parts[1], ":") {
				return nil, fmt.Errorf("invalid format pair: %s", pair)
			}
			if ts := strings.TrimSpace(parts[1]); ts != "" {
				typeStr = ts
			}
		}

		if key == "" {
			return nil, fmt.Errorf("empty key in format pair: %s", pair)
		}

		// Support arrays specified either in type (e.g., string[]) or as key[] shorthand.
		// If both key[] and type[] are used together, treat as nested which is unsupported.
		if keyIsArray && strings.HasSuffix(typeStr, "[]") {
			return nil, fmt.Errorf("nested array types are not supported: %s", trimmed)
		}

		if keyIsArray {
			// key[] with omitted or empty type defaults to string[]
			elementType := strings.TrimSpace(typeStr)
			if elementType == "" {
				elementType = "string"
			}
			if strings.HasSuffix(elementType, "[]") {
				return nil, fmt.Errorf("nested array types are not supported: %s", trimmed)
			}
			properties[key] = map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": elementType,
				},
			}
		} else if strings.HasSuffix(typeStr, "[]") {
			elementType := strings.TrimSpace(strings.TrimSuffix(typeStr, "[]"))
			if elementType == "" {
				return nil, fmt.Errorf("empty element type in array specification: %s", typeStr)
			}
			if strings.HasSuffix(elementType, "[]") {
				return nil, fmt.Errorf("nested array types are not supported: %s", typeStr)
			}

			properties[key] = map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": elementType,
				},
			}
		} else {
			properties[key] = map[string]interface{}{"type": typeStr}
		}

	}

	return properties, nil
}
