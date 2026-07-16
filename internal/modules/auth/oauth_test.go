package auth

import (
	"context"
	"testing"
	"time"

	"nexora/internal/pkg/config"
)

func TestOAuthService_ExchangeCodeAndGetUserInfo_UnsupportedProvider(t *testing.T) {
	s := NewOAuthService(nil, nil)
	_, err := s.ExchangeCodeAndGetUserInfo(context.Background(), "unsupported", "code")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestOAuthUserInfo_Fields(t *testing.T) {
	info := &OAuthUserInfo{
		Email:      "test@example.com",
		Name:       "Test User",
		Avatar:     "https://example.com/avatar.png",
		Provider:   "google",
		ProviderID: "12345",
	}

	if info.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", info.Email)
	}
	if info.Provider != "google" {
		t.Errorf("expected google, got %s", info.Provider)
	}
	if info.ProviderID != "12345" {
		t.Errorf("expected 12345, got %s", info.ProviderID)
	}
}

func TestOAuthService_GetAuthorizationURL_UnsupportedProvider(t *testing.T) {
	s := NewOAuthService(nil, nil)
	_, err := s.GetAuthorizationURL("unsupported", "")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestOAuthService_ValidateState_Invalid(t *testing.T) {
	s := NewOAuthService(nil, nil)
	if s.ValidateState("nonexistent") {
		t.Fatal("expected false for invalid state")
	}
}

func TestOAuthService_GetProvider_Nil(t *testing.T) {
	s := NewOAuthService(nil, nil)
	if p := s.GetProvider("unsupported"); p != nil {
		t.Fatal("expected nil for unsupported provider")
	}
}

func TestNewOAuthService_WithProviders(t *testing.T) {
	cfg := &struct {
		Google  struct{ ClientID, ClientSecret, RedirectURL string }
		GitHub  struct{ ClientID, ClientSecret, RedirectURL string }
	}{
		Google: struct{ ClientID, ClientSecret, RedirectURL string }{
			ClientID:     "google-client-id",
			ClientSecret: "google-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
		},
		GitHub: struct{ ClientID, ClientSecret, RedirectURL string }{
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
		},
	}

	oauthCfg := &config.OAuthConfig{}
	oauthCfg.Google.ClientID = cfg.Google.ClientID
	oauthCfg.Google.ClientSecret = cfg.Google.ClientSecret
	oauthCfg.Google.RedirectURL = cfg.Google.RedirectURL
	oauthCfg.GitHub.ClientID = cfg.GitHub.ClientID
	oauthCfg.GitHub.ClientSecret = cfg.GitHub.ClientSecret
	oauthCfg.GitHub.RedirectURL = cfg.GitHub.RedirectURL

	s := NewOAuthService(oauthCfg, nil)
	if s.GetProvider("google") == nil {
		t.Error("expected google provider config")
	}
	if s.GetProvider("github") == nil {
		t.Error("expected github provider config")
	}
}

func TestNewOAuthService_WithGoogleOnly(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.Google.ClientID = "google-client-id"
	cfg.Google.ClientSecret = "google-client-secret"
	cfg.Google.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)
	if s.GetProvider("google") == nil {
		t.Error("expected google provider config")
	}
	if s.GetProvider("github") != nil {
		t.Error("expected no github provider config")
	}
}

func TestNewOAuthService_WithGitHubOnly(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.GitHub.ClientID = "github-client-id"
	cfg.GitHub.ClientSecret = "github-client-secret"
	cfg.GitHub.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)
	if s.GetProvider("github") == nil {
		t.Error("expected github provider config")
	}
	if s.GetProvider("google") != nil {
		t.Error("expected no google provider config")
	}
}

func TestOAuthService_GetAuthorizationURL_Google(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.Google.ClientID = "google-client-id"
	cfg.Google.ClientSecret = "google-client-secret"
	cfg.Google.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)
	url, err := s.GetAuthorizationURL("google", "http://localhost:3000/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestOAuthService_GetAuthorizationURL_GitHub(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.GitHub.ClientID = "github-client-id"
	cfg.GitHub.ClientSecret = "github-client-secret"
	cfg.GitHub.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)
	url, err := s.GetAuthorizationURL("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestOAuthService_ValidateState_Valid(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.Google.ClientID = "google-client-id"
	cfg.Google.ClientSecret = "google-client-secret"
	cfg.Google.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)

	// Generate a valid state by calling GetAuthorizationURL
	_, err := s.GetAuthorizationURL("google", "")
	if err != nil {
		t.Fatalf("GetAuthorizationURL failed: %v", err)
	}

	// Find the generated state
	var state string
	for k := range s.stateStore {
		state = k
		break
	}

	if state == "" {
		t.Fatal("no state found in store")
	}

	if !s.ValidateState(state) {
		t.Fatal("expected state to be valid")
	}

	// State should be consumed (deleted after validation)
	if s.ValidateState(state) {
		t.Fatal("expected state to be consumed after validation")
	}
}

func TestOAuthService_ValidateState_Expired(t *testing.T) {
	s := NewOAuthService(nil, nil)

	// Add an expired state
	expiredState := "expired-state"
	s.stateStore[expiredState] = time.Now().Add(-1 * time.Hour)

	if s.ValidateState(expiredState) {
		t.Fatal("expected expired state to be invalid")
	}
}

func TestOAuthService_GetProvider_Valid(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.Google.ClientID = "google-client-id"
	cfg.Google.ClientSecret = "google-client-secret"
	cfg.Google.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)

	p := s.GetProvider("google")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.ClientID != "google-client-id" {
		t.Errorf("expected google-client-id, got %s", p.ClientID)
	}
	if p.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Errorf("unexpected auth URL: %s", p.AuthURL)
	}
}

func TestOAuthService_GetAuthorizationURL_WithCustomRedirect(t *testing.T) {
	cfg := &config.OAuthConfig{}
	cfg.Google.ClientID = "google-client-id"
	cfg.Google.ClientSecret = "google-client-secret"
	cfg.Google.RedirectURL = "http://localhost:8080/callback"

	s := NewOAuthService(cfg, nil)
	url, err := s.GetAuthorizationURL("google", "http://localhost:3000/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
}

func TestOAuthService_ExchangeCodeAndGetUserInfo_UnsupportedProvider_Detailed(t *testing.T) {
	s := NewOAuthService(nil, nil)
	_, err := s.ExchangeCodeAndGetUserInfo(context.Background(), "unknown", "code")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOAuthUserInfo_EmptyStrings(t *testing.T) {
	info := &OAuthUserInfo{}
	if info.Email != "" {
		t.Errorf("expected empty email, got %s", info.Email)
	}
	if info.Provider != "" {
		t.Errorf("expected empty provider, got %s", info.Provider)
	}
}

func TestOAuthProviderConfig_Defaults(t *testing.T) {
	p := &OAuthProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
		Scopes:       []string{"openid", "email"},
	}
	if len(p.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(p.Scopes))
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := generateState()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := generateState()
	if err != nil {
		t.Fatal(err)
	}

	if len(s1) != 32 {
		t.Errorf("expected hex string length 32, got %d", len(s1))
	}
	if s1 == s2 {
		t.Error("expected different states")
	}
}
