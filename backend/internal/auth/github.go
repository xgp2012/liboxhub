package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubUser GitHub API /user 的子集。
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// OAuthClient 封装 GitHub OAuth 流程。
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	http         *http.Client
}

func NewOAuthClient(clientID, secret, redirect string) *OAuthClient {
	return &OAuthClient{
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectURI:  redirect,
		http:         &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled 表示是否配置了 OAuth 凭证。
func (c *OAuthClient) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// AuthorizeURL 返回 GitHub 授权页地址。
func (c *OAuthClient) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURI)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

// Exchange 用 code 换取 access token。
func (c *OAuthClient) Exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exchange status %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		AccessToken      string `json:"access_token"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("exchange decode: %w", err)
	}
	if out.Error != "" || out.AccessToken == "" {
		return "", fmt.Errorf("exchange failed: %s %s", out.Error, out.ErrorDescription)
	}
	return out.AccessToken, nil
}

// FetchUser 用 access token 拉取用户信息。
func (c *OAuthClient) FetchUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "boxli-hub")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch user status %d: %s", resp.StatusCode, string(body))
	}
	var u GitHubUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("fetch user decode: %w", err)
	}
	return &u, nil
}
