package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bnn/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newTestRepo(t *testing.T, user, emails string) *Repo {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(user))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		if emails == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(emails))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &Repo{
		config: &oauth2.Config{
			ClientID:     "id",
			ClientSecret: "secret",
			Endpoint:     oauth2.Endpoint{TokenURL: server.URL + "/token"},
		},
		apiURL: server.URL + "/",
	}
}

func TestReadProfile(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		emails   string
		expected models.OAuthProfile
		wantErr  bool
	}{
		{
			name:     "public email",
			user:     `{"email":"a@b.ru","avatar_url":"https://avatars/1"}`,
			expected: models.OAuthProfile{Email: "a@b.ru", AvatarURL: "https://avatars/1"},
		},
		{
			name:     "primary verified email",
			user:     `{"email":"","avatar_url":"https://avatars/1"}`,
			emails:   `[{"email":"old@b.ru","primary":false,"verified":true},{"email":"main@b.ru","primary":true,"verified":true}]`,
			expected: models.OAuthProfile{Email: "main@b.ru", AvatarURL: "https://avatars/1"},
		},
		{
			name:    "primary not verified",
			user:    `{"email":""}`,
			emails:  `[{"email":"main@b.ru","primary":true,"verified":false}]`,
			wantErr: true,
		},
		{
			name:    "emails request failed",
			user:    `{"email":""}`,
			wantErr: true,
		},
		{
			name:    "broken json",
			user:    `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestRepo(t, tt.user, tt.emails)

			profile, err := repo.ReadProfile(context.Background(), "code")

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, profile)
		})
	}
}

func TestAuthURL(t *testing.T) {
	repo, err := NewRepo("client-id", "secret", "https://brandnewnotes.ru")
	require.NoError(t, err)

	link := repo.AuthURL("state-1")

	assert.True(t, strings.HasPrefix(link, "https://github.com/login/oauth/authorize?"))
	assert.Contains(t, link, "client_id=client-id")
	assert.Contains(t, link, "state=state-1")
}

func TestCallbackURL(t *testing.T) {
	repo, err := NewRepo("client-id", "secret", "https://brandnewnotes.ru")
	require.NoError(t, err)
	assert.Equal(t, "https://brandnewnotes.ru/api/auth/github/callback", repo.config.RedirectURL)
}

func TestHTTPSURL(t *testing.T) {
	tests := []struct {
		raw      string
		expected string
	}{
		{raw: "https://avatars.githubusercontent.com/u/1?v=4", expected: "https://avatars.githubusercontent.com/u/1?v=4"},
		{raw: "javascript:alert(1)", expected: ""},
		{raw: "http://example.com/a.png", expected: ""},
		{raw: "", expected: ""},
		{raw: "%zz", expected: ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, httpsURL(tt.raw), tt.raw)
	}
}
