package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseFormat parses a format string like "key1:type,key2:type,..." into a JSON schema
// Supports array types with element specifications: "key:array[element_type]"
func ParseFormat(format string) (map[string]interface{}, []string, error) {
	if format == "" {
		// Default format
		return map[string]interface{}{
			"message": map[string]interface{}{
				"type": "string",
			},
		}, []string{"message"}, nil
	}

	properties := make(map[string]interface{})
	var required []string

    pairs := strings.Split(format, ",")
    for _, pair := range pairs {
        trimmed := strings.TrimSpace(pair)
        if trimmed == "" {
            return nil, nil, fmt.Errorf("invalid format pair: %s", pair)
        }

        parts := strings.SplitN(trimmed, ":", 2)
        key := strings.TrimSpace(parts[0])
        // default type to string when omitted or empty (e.g., "name" or "name:")
        typeStr := "string"
        if len(parts) == 2 {
            // if extra colon remains in type portion, treat as error (e.g., name:string:string)
            if strings.Contains(parts[1], ":") {
                return nil, nil, fmt.Errorf("invalid format pair: %s", pair)
            }
            if ts := strings.TrimSpace(parts[1]); ts != "" {
                typeStr = ts
            }
        }

		if key == "" {
			return nil, nil, fmt.Errorf("empty key in format pair: %s", pair)
		}

        // Only support type[] style arrays (e.g., string[])
        if strings.HasSuffix(typeStr, "[]") {
            elementType := strings.TrimSpace(strings.TrimSuffix(typeStr, "[]"))
            if elementType == "" {
                return nil, nil, fmt.Errorf("empty element type in array specification: %s", typeStr)
            }
            if strings.HasSuffix(elementType, "[]") {
                return nil, nil, fmt.Errorf("nested array types are not supported: %s", typeStr)
            }

            properties[key] = map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": elementType,
                },
            }
        } else if strings.HasPrefix(typeStr, "array[") {
            // legacy style no longer supported
            return nil, nil, fmt.Errorf("invalid array specification: use type[] syntax, got %s", typeStr)
        } else {
            properties[key] = map[string]interface{}{"type": typeStr}
        }

		required = append(required, key)
	}

	return properties, required, nil
}

// ParseAPIResponse parses the API response and extracts the output text
// Prefers output_text field, falls back to output[].content[].text with type "output_text"
func ParseAPIResponse(respBody []byte) (string, error) {
	var apiResp struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Prefer the convenience field, then fall back to output[].content[].text
	textOut := apiResp.OutputText
	if textOut == "" {
		for _, item := range apiResp.Output {
			for _, c := range item.Content {
				if c.Type == "output_text" && c.Text != "" {
					textOut = c.Text
					break
				}
			}
			if textOut != "" {
				break
			}
		}
	}

	return textOut, nil
}
