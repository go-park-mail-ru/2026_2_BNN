package httpjson

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "body too large",
			err:          &http.MaxBytesError{Limit: 10},
			expectedCode: http.StatusRequestEntityTooLarge,
		},
		{
			name:         "bad json",
			err:          errors.New("can't decode JSON body"),
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			RequestError(rec, httptest.NewRequest(http.MethodPost, "/", nil), tt.err)

			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Empty(t, rec.Body.String())
		})
	}
}

func TestResponseError(t *testing.T) {
	rec := httptest.NewRecorder()

	ResponseError(rec, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("db is down"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, rec.Body.String())
}
