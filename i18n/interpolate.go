package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

// normalizeParams converts variadic args into a key-value parameter map.
func normalizeParams(args []any) map[string]any {
	if len(args) == 0 {
		return nil
	}

	params := make(map[string]any)

	// Check if the first argument is already a map
	switch first := args[0].(type) {
	case map[string]any:
		for k, v := range first {
			params[k] = v
		}
		return params
	case map[string]string:
		for k, v := range first {
			params[k] = v
		}
		return params
	}

	// Alternating key-value pairs: "key", val, "key2", val2
	for i := 0; i < len(args); i += 2 {
		k := fmt.Sprintf("%v", args[i])
		if i+1 < len(args) {
			params[k] = args[i+1]
		} else {
			params[k] = ""
		}
	}

	return params
}

// interpolate replaces placeholders in message with values from params.
// It supports {key}, {{key}}, and :key placeholders.
func interpolate(msg string, params map[string]any) string {
	if len(params) == 0 || msg == "" {
		return msg
	}

	res := msg
	for k, v := range params {
		valStr := fmt.Sprintf("%v", v)
		res = strings.ReplaceAll(res, "{{"+k+"}}", valStr)
		res = strings.ReplaceAll(res, "{"+k+"}", valStr)
		res = strings.ReplaceAll(res, ":"+k, valStr)
	}
	return res
}

// resolvePlural picks the appropriate translation string if pluralization forms are present.
// It supports map representations (e.g. {"zero": "...", "one": "...", "other": "..."})
// as well as pipe-separated strings ("No items|One item|{count} items").
func resolvePlural(raw any, params map[string]any) (string, bool) {
	if raw == nil {
		return "", false
	}

	// Try extracting count from params
	countVal, hasCount := findCount(params)

	switch val := raw.(type) {
	case string:
		if !hasCount || !strings.Contains(val, "|") {
			return val, true
		}
		parts := strings.Split(val, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		if len(parts) == 2 {
			if countVal == 1 {
				return parts[0], true
			}
			return parts[1], true
		} else if len(parts) >= 3 {
			if countVal == 0 {
				return parts[0], true
			} else if countVal == 1 {
				return parts[1], true
			}
			return parts[2], true
		}
		return val, true

	case map[string]any:
		if !hasCount {
			if other, ok := val["other"].(string); ok {
				return other, true
			}
			return "", false
		}
		if countVal == 0 {
			if zero, ok := val["zero"].(string); ok {
				return zero, true
			}
		} else if countVal == 1 {
			if one, ok := val["one"].(string); ok {
				return one, true
			}
		}
		if other, ok := val["other"].(string); ok {
			return other, true
		}
		// Fallback to any string value in map if other not present
		for _, v := range val {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
		return "", false

	default:
		return fmt.Sprintf("%v", raw), true
	}
}

func findCount(params map[string]any) (int, bool) {
	if len(params) == 0 {
		return 0, false
	}
	for _, key := range []string{"count", "n", "total", "qty"} {
		if v, exists := params[key]; exists {
			switch n := v.(type) {
			case int:
				return n, true
			case int64:
				return int(n), true
			case int32:
				return int(n), true
			case float64:
				return int(n), true
			case string:
				if parsed, err := strconv.Atoi(n); err == nil {
					return parsed, true
				}
			}
		}
	}
	return 0, false
}
