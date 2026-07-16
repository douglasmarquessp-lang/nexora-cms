package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

type OAuthUserInfo struct {
	Email      string
	Name       string
	Avatar     string
	Provider   string
	ProviderID string
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

type OAuthService struct {
	providers  map[string]*OAuthProviderConfig
	stateStore map[string]time.Time
	log        *logger.Logger
}

func NewOAuthService(cfg *config.OAuthConfig, log *logger.Logger) *OAuthService {
	providers := make(map[string]*OAuthProviderConfig)

	if cfg != nil && cfg.Google.ClientID != "" {
		providers["google"] = &OAuthProviderConfig{
			ClientID:     cfg.Google.ClientID,
			ClientSecret: cfg.Google.ClientSecret,
			RedirectURL:  cfg.Google.RedirectURL,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:       []string{"openid", "email", "profile"},
		}
	}

	if cfg != nil && cfg.GitHub.ClientID != "" {
		providers["github"] = &OAuthProviderConfig{
			ClientID:     cfg.GitHub.ClientID,
			ClientSecret: cfg.GitHub.ClientSecret,
			RedirectURL:  cfg.GitHub.RedirectURL,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       []string{"read:user", "user:email"},
		}
	}

	return &OAuthService{
		providers:  providers,
		stateStore: make(map[string]time.Time),
		log:        log,
	}
}

func (s *OAuthService) GetAuthorizationURL(provider, redirectURI string) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	s.stateStore[state] = time.Now().Add(10 * time.Minute)

	u, _ := url.Parse(p.AuthURL)
	q := u.Query()
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(p.Scopes, " "))
	q.Set("state", state)
	if redirectURI != "" {
		q.Set("redirect_uri", redirectURI)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (s *OAuthService) ExchangeCodeAndGetUserInfo(ctx context.Context, provider, code string) (*OAuthUserInfo, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported provider: %s", ErrOAuthProviderError, provider)
	}

	token, err := s.exchangeCode(p, code)
	if err != nil {
		return nil, fmt.Errorf("%w: token exchange failed: %v", ErrOAuthProviderError, err)
	}

	rawInfo, err := s.getUserInfo(p, token, provider)
	if err != nil {
		return nil, fmt.Errorf("%w: user info fetch failed: %v", ErrOAuthProviderError, err)
	}

	email, _ := rawInfo["email"].(string)
	if email == "" {
		return nil, fmt.Errorf("%w: email not provided by provider", ErrOAuthProviderError)
	}

	name, _ := rawInfo["name"].(string)
	if name == "" {
		name = email[:strings.Index(email, "@")]
	}

	avatar, _ := rawInfo["avatar"].(string)
	if avatar == "" {
		if provider == "google" {
			if picture, ok := rawInfo["picture"].(string); ok {
				avatar = picture
			}
		} else if provider == "github" {
			if av, ok := rawInfo["avatar_url"].(string); ok {
				avatar = av
			}
		}
	}

	providerID, _ := rawInfo["id"].(string)
	if providerID == "" {
		if provider == "google" {
			if sub, ok := rawInfo["sub"].(string); ok {
				providerID = sub
			}
		}
	}
	if providerID == "" {
		return nil, fmt.Errorf("%w: provider ID not found in user info", ErrOAuthProviderError)
	}

	return &OAuthUserInfo{
		Email:      email,
		Name:       name,
		Avatar:     avatar,
		Provider:   provider,
		ProviderID: providerID,
	}, nil
}

func (s *OAuthService) ValidateState(state string) bool {
	expiry, ok := s.stateStore[state]
	if !ok {
		return false
	}
	delete(s.stateStore, state)
	return time.Now().Before(expiry)
}

func (s *OAuthService) GetProvider(name string) *OAuthProviderConfig {
	return s.providers[name]
}

func (s *OAuthService) exchangeCode(p *OAuthProviderConfig, code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", p.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", p.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

func (s *OAuthService) getUserInfo(p *OAuthProviderConfig, accessToken, provider string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", p.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
