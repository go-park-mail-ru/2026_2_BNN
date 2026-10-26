package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/vk"
)

type OAuthConfig struct {
	GitHub *oauth2.Config
	VK     *oauth2.Config
}

func NewOAuthConfig(githubClientID, githubSecret, vkClientID, vkSecret, redirectBase string) *OAuthConfig {
	return &OAuthConfig{
		GitHub: &oauth2.Config{
			ClientID:     githubClientID,
			ClientSecret: githubSecret,
			RedirectURL:  redirectBase + "/api/auth/github/callback",
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
		VK: &oauth2.Config{
			ClientID:     vkClientID,
			ClientSecret: vkSecret,
			RedirectURL:  redirectBase + "/api/auth/vk/callback",
			Endpoint:     vk.Endpoint,
			Scopes:       []string{"email"},
		},
	}
}
