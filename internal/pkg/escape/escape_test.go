package escape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert.Equal(t, "&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;", String("<script>alert('x')</script>"))
	assert.Equal(t, "Tom &amp; Jerry", String("Tom & Jerry"))
	assert.Equal(t, "обычный текст", String("обычный текст"))
}

func TestMap(t *testing.T) {
	content := map[string]any{
		"type": "text",
		"text": "<b>жирный</b>",
		"meta": map[string]any{"title": `"кавычки"`},
		"items": []any{
			"<img src=x onerror=alert(1)>",
			map[string]any{"href": "a&b"},
			42.0,
			true,
			nil,
		},
	}

	expected := map[string]any{
		"type": "text",
		"text": "&lt;b&gt;жирный&lt;/b&gt;",
		"meta": map[string]any{"title": "&#34;кавычки&#34;"},
		"items": []any{
			"&lt;img src=x onerror=alert(1)&gt;",
			map[string]any{"href": "a&amp;b"},
			42.0,
			true,
			nil,
		},
	}

	assert.Equal(t, expected, Map(content))
}
