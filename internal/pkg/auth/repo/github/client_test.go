package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestClientGetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/user", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email":"test@example.com","avatar_url":"https://avatar.com/1"}`))
	}))
	defer server.Close()

	config := &oauth2.Config{}
	client := &Client{
		config:  config,
		baseURL: server.URL,
	}

	var result struct {
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	validToken := &oauth2.Token{
		AccessToken: "fake-token",
		Expiry:      time.Now().Add(time.Hour),
	}

	err := client.GetJSON(context.Background(), validToken, "user", &result)

	require.NoError(t, err)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "https://avatar.com/1", result.AvatarURL)
}

func TestClientGetJSON_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	client := &Client{
		config:  &oauth2.Config{},
		baseURL: server.URL,
	}

	var result any
	validToken := &oauth2.Token{
		AccessToken: "fake-token",
		Expiry:      time.Now().Add(time.Hour),
	}

	err := client.GetJSON(context.Background(), validToken, "user", &result)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "github api error: 404")
	assert.Contains(t, err.Error(), "Not Found")
}
