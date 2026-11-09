package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"bnn/internal/models"

	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	apiURL       = "https://api.github.com"
	callbackPath = "/api/auth/github/callback"
)

type APIClient interface {
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	GetJSON(ctx context.Context, token *oauth2.Token, path string, dst any) error
	AuthURL(state string) string
}

type Repo struct {
	client APIClient
}

func NewRepo(client APIClient) *Repo {
	return &Repo{client: client}
}

type githubUser struct {
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (r *Repo) ReadProfile(ctx context.Context, code string) (models.OAuthProfile, error) {
	token, err := r.client.ExchangeCode(ctx, code)
	if err != nil {
		return models.OAuthProfile{}, fmt.Errorf("exchange github code: %w", err)
	}

	var user githubUser
	if err := r.client.GetJSON(ctx, token, "user", &user); err != nil {
		return models.OAuthProfile{}, err
	}

	email := user.Email
	if email == "" {
		var emails []githubEmail
		if err := r.client.GetJSON(ctx, token, "user/emails", &emails); err != nil {
			return models.OAuthProfile{}, err
		}

		primary, ok := lo.Find(emails, func(e githubEmail) bool {
			return e.Primary && e.Verified
		})
		if ok {
			email = primary.Email
		}
	}

	if email == "" {
		return models.OAuthProfile{}, errors.New("no verified email provided by github")
	}

	return models.OAuthProfile{
		Email:     email,
		AvatarURL: httpsURL(user.AvatarURL),
	}, nil
}

func httpsURL(raw string) string {
	link, err := url.Parse(raw)
	if err != nil || link.Scheme != "https" {
		return ""
	}
	return raw
}

func (r *Repo) AuthURL(state string) string {
	return r.client.AuthURL(state)
}
