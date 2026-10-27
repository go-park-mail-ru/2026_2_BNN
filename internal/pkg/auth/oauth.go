package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type OAuthConfig struct {
	GitHub *oauth2.Config
}

func NewOAuthConfig(githubClientID, githubSecret, redirectBase string) *OAuthConfig {
	return &OAuthConfig{
		GitHub: &oauth2.Config{
			ClientID:     githubClientID,
			ClientSecret: githubSecret,
			RedirectURL:  redirectBase + "/api/auth/github/callback",
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
	}
}
