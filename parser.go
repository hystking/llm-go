package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseFormat parses a format string like "key1:type,key2:type,..." into a JSON schema
// Supports array types with element specifications: "key:array[element_type]"
func parseFormat(format string) (map[string]interface{}, []string, error) {
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
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid format pair: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		typeStr := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, nil, fmt.Errorf("empty key in format pair: %s", pair)
		}

		// Check for array[element_type] format
		if strings.HasPrefix(typeStr, "array[") && strings.HasSuffix(typeStr, "]") {
			// Extract element type from array[element_type]
			elementType := typeStr[6 : len(typeStr)-1] // Remove "array[" and "]"
			elementType = strings.TrimSpace(elementType)

			if elementType == "" {
				return nil, nil, fmt.Errorf("empty element type in array specification: %s", typeStr)
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

		required = append(required, key)
	}

	return properties, required, nil
}

// parseAPIResponse parses the API response and extracts the output text
// Prefers output_text field, falls back to output[].content[].text with type "output_text"
func parseAPIResponse(respBody []byte) (string, error) {
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