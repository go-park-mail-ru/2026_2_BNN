package escape

import "html"

func String(s string) string {
	return html.EscapeString(s)
}

func Map(m map[string]any) map[string]any {
	for key, val := range m {
		m[key] = value(val)
	}
	return m
}

func value(v any) any {
	switch t := v.(type) {
	case string:
		return html.EscapeString(t)
	case map[string]any:
		return Map(t)
	case []any:
		for i, item := range t {
			t[i] = value(item)
		}
		return t
	default:
		return v
	}
}
