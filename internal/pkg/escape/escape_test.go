package escape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	assert.Equal(t, "&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;", String("<script>alert('x')</script>"))
	assert.Equal(t, "Tom &amp; Jerry", String("Tom & Jerry"))
	assert.Equal(t, "обычный текст", String("обычный текст"))
}

func TestJSON(t *testing.T) {
	raw := []byte(`{"type":"text","text":"<b>жирный</b>","meta":{"title":"\"кавычки\""},"items":["<img src=x onerror=alert(1)>",{"href":"a&b"},42,true,null]}`)

	got, err := JSON(raw)

	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"text","text":"&lt;b&gt;жирный&lt;/b&gt;","meta":{"title":"&#34;кавычки&#34;"},"items":["&lt;img src=x onerror=alert(1)&gt;",{"href":"a&amp;b"},42,true,null]}`, string(got))
}

func TestJSONInvalid(t *testing.T) {
	_, err := JSON([]byte(`{`))

	assert.Error(t, err)
}
