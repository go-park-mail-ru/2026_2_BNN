package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Client struct {
	config  *oauth2.Config
	baseURL string
}

func NewClient(clientID, clientSecret, appBaseURL string) (*Client, error) {
	callbackURL, err := url.JoinPath(appBaseURL, callbackPath)
	if err != nil {
		return nil, fmt.Errorf("build github callback url: %w", err)
	}

	return &Client{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
		baseURL: apiURL,
	}, nil
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.config.Exchange(ctx, code)
}

func (c *Client) AuthURL(state string) string {
	return c.config.AuthCodeURL(state)
}

func (c *Client) GetJSON(ctx context.Context, token *oauth2.Token, path string, dst any) error {
	client := c.config.Client(ctx, token)
	reqURL := c.baseURL + "/" + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api error: %s, body: %s", resp.Status, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	return nil
}
