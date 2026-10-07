package httpjson

import (
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		data         any
		expectedCode int
		expectedBody string
	}{
		{
			name:         "object",
			status:       http.StatusCreated,
			data:         map[string]string{"login": "seyran"},
			expectedCode: http.StatusCreated,
			expectedBody: `{"login":"seyran"}`,
		},
		{
			name:         "empty slice is [] not null",
			status:       http.StatusOK,
			data:         []string{},
			expectedCode: http.StatusOK,
			expectedBody: `[]`,
		},
		{
			name:         "unencodable value",
			status:       http.StatusOK,
			data:         math.Inf(1),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			Write(rec, tt.status, tt.data)

			require.Equal(t, tt.expectedCode, rec.Code)

			if tt.expectedBody == "" {
				assert.Empty(t, rec.Body.String())
				assert.Empty(t, rec.Header().Get("Content-Type"))

				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, rec.Body.String())
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

func TestRead(t *testing.T) {
	type request struct {
		Login string `json:"login"`
	}

	const limit = 32

	tests := []struct {
		name          string
		body          string
		failingBody   bool
		expectErr     bool
		expectTooBig  bool
		expectedLogin string
	}{
		{
			name:          "valid JSON",
			body:          `{"login":"seyran"}`,
			expectedLogin: "seyran",
		},
		{
			name:      "invalid JSON",
			body:      `{"login":`,
			expectErr: true,
		},
		{
			name:      "wrong field type",
			body:      `{"login":123}`,
			expectErr: true,
		},
		{
			name:      "two JSON objects",
			body:      `{"login":"a"}{}`,
			expectErr: true,
		},
		{
			name:         "body too large",
			body:         `{"login":"` + strings.Repeat("a", limit) + `"}`,
			expectErr:    true,
			expectTooBig: true,
		},
		{
			name:        "read error",
			failingBody: true,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if tt.failingBody {
				req = httptest.NewRequest(http.MethodPost, "/", failingReader{})
			}

			var dst request
			err := Read(httptest.NewRecorder(), req, &dst, limit)

			if !tt.expectErr {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedLogin, dst.Login)

				return
			}

			require.Error(t, err)
			assert.Equal(t, tt.expectTooBig, errors.Is(err, ErrBodyTooLarge))
		})
	}
}

func TestReadErrorStatus(t *testing.T) {
	assert.Equal(t, http.StatusRequestEntityTooLarge, ReadErrorStatus(ErrBodyTooLarge))
	assert.Equal(t, http.StatusBadRequest, ReadErrorStatus(errors.New("decode json")))
}
