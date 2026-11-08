package escape

import (
	"encoding/json"
	"html"
)

func String(s string) string {
	return html.EscapeString(s)
}

func JSON(raw []byte) ([]byte, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return json.Marshal(value(data))
}

func value(v any) any {
	switch t := v.(type) {
	case string:
		return html.EscapeString(t)
	case map[string]any:
		for key, item := range t {
			t[key] = value(item)
		}
		return t
	case []any:
		for i, item := range t {
			t[i] = value(item)
		}
		return t
	default:
		return v
	}
}
