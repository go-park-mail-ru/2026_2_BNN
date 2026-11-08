package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"bnn/internal/models"

	"github.com/samber/lo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	apiURL       = "https://api.github.com"
	callbackPath = "/api/auth/github/callback"
)

var errNoEmail = errors.New("github account has no verified primary email")

type githubUser struct {
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type Repo struct {
	config *oauth2.Config
	apiURL string
}

func NewRepo(clientID, clientSecret, appBaseURL string) (*Repo, error) {
	callbackURL, err := url.JoinPath(appBaseURL, callbackPath)
	if err != nil {
		return nil, fmt.Errorf("build github callback url: %w", err)
	}

	return &Repo{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
		apiURL: apiURL,
	}, nil
}

func (r *Repo) AuthURL(state string) string {
	return r.config.AuthCodeURL(state)
}

func (r *Repo) ReadProfile(ctx context.Context, code string) (models.OAuthProfile, error) {
	token, err := r.config.Exchange(ctx, code)
	if err != nil {
		return models.OAuthProfile{}, fmt.Errorf("exchange github code: %w", err)
	}
	client := r.config.Client(ctx, token)

	var user githubUser
	if err := r.getJSON(ctx, client, "user", &user); err != nil {
		return models.OAuthProfile{}, err
	}

	email := user.Email
	if email == "" {
		var emails []githubEmail
		if err := r.getJSON(ctx, client, "user/emails", &emails); err != nil {
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
		return models.OAuthProfile{}, errNoEmail
	}

	return models.OAuthProfile{
		Email:     email,
		AvatarURL: httpsURL(user.AvatarURL),
	}, nil
}

func (r *Repo) getJSON(ctx context.Context, client *http.Client, path string, dst any) error {
	link, err := url.JoinPath(r.apiURL, path)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("get github %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get github %s: status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode github %s: %w", path, err)
	}

	return nil
}

func httpsURL(raw string) string {
	link, err := url.Parse(raw)
	if err != nil || link.Scheme != "https" {
		return ""
	}
	return raw
}
