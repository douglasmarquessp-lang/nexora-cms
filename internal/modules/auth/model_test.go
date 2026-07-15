package auth

import (
	"context"
	"testing"

	"time"

	"github.com/google/uuid"
)

func TestCtxKey_String(t *testing.T) {
	k := ctxKey("test_key")
	if k.String() != "test_key" {
		t.Errorf("expected 'test_key', got '%s'", k.String())
	}
}

func TestCtxKey_String_Default(t *testing.T) {
	k := CtxUserID
	if k.String() != "user_id" {
		t.Errorf("expected 'user_id', got '%s'", k.String())
	}
}

func TestGetUserIDFromCtx_NotFound(t *testing.T) {
	ctx := context.Background()
	_, ok := GetUserIDFromCtx(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}
}

func TestGetUserIDFromCtx_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxUserID, "not-a-uuid")
	_, ok := GetUserIDFromCtx(ctx)
	if ok {
		t.Fatal("expected false for wrong type")
	}
}

func TestGetUserIDFromCtx_Found(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), CtxUserID, uid)
	result, ok := GetUserIDFromCtx(ctx)
	if !ok {
		t.Fatal("expected true for valid context")
	}
	if result != uid {
		t.Errorf("expected %s, got %s", uid, result)
	}
}

func TestUserStruct(t *testing.T) {
	uid := uuid.New()
	now := time.Now()
	u := User{
		ID:           uid,
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
		Name:         "Test User",
		Avatar:       "https://example.com/avatar.png",
		Role:         "admin",
		Metadata:     map[string]interface{}{"key": "value"},
		MFAEnabled:   true,
		MFASecret:    "mfa-secret",
		LastLogin:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if u.ID != uid {
		t.Errorf("expected %s, got %s", uid, u.ID)
	}
	if u.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", u.Email)
	}
}

func TestSessionStruct(t *testing.T) {
	uid := uuid.New()
	s := Session{
		ID:           uuid.New(),
		UserID:       uid,
		RefreshToken: "refresh-token",
		DeviceInfo:   "Mozilla/5.0",
		IPAddress:    "127.0.0.1",
	}

	if s.UserID != uid {
		t.Errorf("expected %s, got %s", uid, s.UserID)
	}
	if s.RefreshToken != "refresh-token" {
		t.Errorf("expected refresh-token, got %s", s.RefreshToken)
	}
}

func TestOAuthAccountStruct(t *testing.T) {
	a := OAuthAccount{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Provider:     "google",
		ProviderID:   "12345",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	if a.Provider != "google" {
		t.Errorf("expected google, got %s", a.Provider)
	}
	if a.ProviderID != "12345" {
		t.Errorf("expected 12345, got %s", a.ProviderID)
	}
}

func TestMFAConfigStruct(t *testing.T) {
	c := MFAConfig{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Secret:      "secret",
		Enabled:     true,
		Method:      "totp",
		BackupCodes: []string{"code1", "code2"},
	}

	if !c.Enabled {
		t.Error("expected enabled")
	}
	if c.Method != "totp" {
		t.Errorf("expected totp, got %s", c.Method)
	}
}

func TestLoginRequest(t *testing.T) {
	r := LoginRequest{
		Email:    "test@example.com",
		Password: "password",
		MFACode:  "123456",
	}
	if r.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", r.Email)
	}
	if r.MFACode != "123456" {
		t.Errorf("expected 123456, got %s", r.MFACode)
	}
}

func TestRegisterRequest(t *testing.T) {
	r := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	if r.Name != "Test User" {
		t.Errorf("expected 'Test User', got '%s'", r.Name)
	}
}

func TestRefreshRequest(t *testing.T) {
	r := RefreshRequest{RefreshToken: "token123"}
	if r.RefreshToken != "token123" {
		t.Errorf("expected token123, got %s", r.RefreshToken)
	}
}

func TestAuthResponse(t *testing.T) {
	u := User{ID: uuid.New(), Email: "test@example.com"}
	r := AuthResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User:         u,
	}
	if r.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", r.TokenType)
	}
	if r.ExpiresIn != 3600 {
		t.Errorf("expected 3600, got %d", r.ExpiresIn)
	}
}

func TestOAuthRequest(t *testing.T) {
	r := OAuthRequest{
		Provider: "google",
		Code:     "auth-code",
		Redirect: "http://localhost:3000/callback",
	}
	if r.Provider != "google" {
		t.Errorf("expected google, got %s", r.Provider)
	}
	if r.Redirect != "http://localhost:3000/callback" {
		t.Errorf("expected callback URL, got %s", r.Redirect)
	}
}

func TestOAuthURLResponse(t *testing.T) {
	r := OAuthURLResponse{URL: "https://example.com/auth"}
	if r.URL != "https://example.com/auth" {
		t.Errorf("expected URL, got %s", r.URL)
	}
}

func TestMFAEnrollResponse(t *testing.T) {
	r := MFAEnrollResponse{
		Secret:      "ABCD1234",
		QRCodeURL:   "otpauth://totp/...",
		BackupCodes: []string{"code1", "code2"},
	}
	if r.Secret != "ABCD1234" {
		t.Errorf("expected ABCD1234, got %s", r.Secret)
	}
	if len(r.BackupCodes) != 2 {
		t.Errorf("expected 2 backup codes, got %d", len(r.BackupCodes))
	}
}

func TestMFAVerifyRequest(t *testing.T) {
	r := MFAVerifyRequest{Code: "123456"}
	if r.Code != "123456" {
		t.Errorf("expected 123456, got %s", r.Code)
	}
}

func TestMFADisableRequest(t *testing.T) {
	r := MFADisableRequest{
		Code:     "123456",
		Password: "mypassword",
	}
	if r.Password != "mypassword" {
		t.Errorf("expected mypassword, got %s", r.Password)
	}
}
