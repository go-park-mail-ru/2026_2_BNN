package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"bnn/internal/models"

	githubapi "github.com/google/go-github/v93/github"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"
)

const (
	apiURL       = "https://api.github.com/"
	callbackPath = "/api/auth/github/callback"
)

var errNoEmail = errors.New("github account has no verified primary email")

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
			Endpoint:     oauthgithub.Endpoint,
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

	client, err := githubapi.NewClient(
		githubapi.WithHTTPClient(r.config.Client(ctx, token)),
		githubapi.WithURLs(&r.apiURL, nil),
	)
	if err != nil {
		return models.OAuthProfile{}, fmt.Errorf("create github client: %w", err)
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return models.OAuthProfile{}, fmt.Errorf("get github user: %w", err)
	}

	email := user.GetEmail()
	if email == "" {
		emails, _, err := client.Users.ListEmails(ctx, nil)
		if err != nil {
			return models.OAuthProfile{}, fmt.Errorf("get github emails: %w", err)
		}

		primary, ok := lo.Find(emails, func(e *githubapi.UserEmail) bool {
			return e.GetPrimary() && e.GetVerified()
		})
		if ok {
			email = primary.GetEmail()
		}
	}

	if email == "" {
		return models.OAuthProfile{}, errNoEmail
	}

	return models.OAuthProfile{
		Email:     email,
		AvatarURL: httpsURL(user.GetAvatarURL()),
	}, nil
}

func httpsURL(raw string) string {
	link, err := url.Parse(raw)
	if err != nil || link.Scheme != "https" {
		return ""
	}
	return raw
}
