package github

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakeClient struct {
	exchangeErr error
	getJSONData map[string]string
	getJSONErr  error
}

func (f *fakeClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	return &oauth2.Token{AccessToken: "fake-token"}, nil
}

func (f *fakeClient) GetJSON(ctx context.Context, token *oauth2.Token, path string, dst any) error {
	if f.getJSONErr != nil {
		return f.getJSONErr
	}

	switch path {
	case "user":
		u := dst.(*githubUser)
		u.Email = "test@example.com"
		u.AvatarURL = "https://avatar.com/1"
	case "user/emails":
		e := dst.(*[]githubEmail)
		*e = []githubEmail{
			{Email: "test@example.com", Primary: true, Verified: true},
		}
	}
	return nil
}

func (f *fakeClient) AuthURL(state string) string {
	return "https://github.com/login/oauth/authorize?state=" + state
}

func TestReadProfile(t *testing.T) {
	t.Run("Success with public email", func(t *testing.T) {
		client := &fakeClient{}
		repo := NewRepo(client)

		_, err := repo.ReadProfile(context.Background(), "code")
		assert.NoError(t, err)
	})

	t.Run("Exchange error", func(t *testing.T) {
		client := &fakeClient{exchangeErr: errors.New("exchange failed")}
		repo := NewRepo(client)

		_, err := repo.ReadProfile(context.Background(), "code")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exchange github code")
	})
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
