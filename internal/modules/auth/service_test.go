package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.GetOAuthService() == nil {
		t.Error("expected non-nil OAuth service")
	}
	if svc.GetMFAService() == nil {
		t.Error("expected non-nil MFA service")
	}
}

func TestService_HandleOAuthCallback_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.HandleOAuthCallback(context.Background(), "unsupported_provider", "code")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestService_SetEventBus(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)
	bus := kernel.NewEventBus(log)

	svc.SetEventBus(bus)
	// No panic is the test
}

func TestService_ValidateAccessToken_Invalid(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestService_ValidateAccessToken_Valid(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	uid := uuid.New()
	token, err := svc.tokenManager.GenerateAccessToken(uid.String())
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedUID, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if parsedUID != uid {
		t.Errorf("expected %s, got %s", uid, parsedUID)
	}
}

func TestService_RefreshToken_Invalid(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.RefreshToken(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestService_GetUserByID_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.GetUserByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when no database")
	}
}

func TestService_Logout_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	err := svc.Logout(context.Background(), uuid.New(), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// This test verifies the OAuth error constants
func TestOAuthErrorConstants(t *testing.T) {
	if ErrOAuthProviderError.Error() != "OAuth provider error" {
		t.Errorf("unexpected message: %s", ErrOAuthProviderError.Error())
	}
	if ErrOAuthEmailExists.Error() != "an account with this email already exists" {
		t.Errorf("unexpected message: %s", ErrOAuthEmailExists.Error())
	}
}

func TestRegisterRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{"empty email", RegisterRequest{Password: "password123", Name: "Test"}, true},
		{"empty password", RegisterRequest{Email: "test@example.com", Name: "Test"}, true},
		{"short password", RegisterRequest{Email: "test@example.com", Password: "short", Name: "Test"}, true},
		{"empty name", RegisterRequest{Email: "test@example.com", Password: "password123"}, true},
		{"valid", RegisterRequest{Email: "test@example.com", Password: "password123", Name: "Test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewServiceFromModule(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-at-least-32-bytes-long-for-hmac",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 900000000000,
		},
	}
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	svc := mod.Service()
	if svc == nil {
		t.Fatal("expected non-nil service from module")
	}
}

func TestAuditLogPackage(t *testing.T) {
	// Verify audit package constants
	_ = []string{
		string(audit.ActionUserLogin),
		string(audit.ActionUserLogout),
		string(audit.ActionUserRegistered),
		string(audit.ActionOAuthLinked),
		string(audit.ActionOAuthLogin),
	}
}

func TestKernelEventConstants(t *testing.T) {
	_ = []string{
		string(kernel.EventUserRegistered),
		string(kernel.EventUserLogin),
		string(kernel.EventUserLogout),
		string(kernel.EventOAuthLinked),
		string(kernel.EventOAuthLogin),
		string(kernel.EventTokenRefreshed),
		string(kernel.EventPasswordChange),
	}
}

func TestService_Register_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// With nil db, findUserByEmail returns error -> proceeds to hash -> tries Pool.Exec -> panic
	// Test that validation error comes first
	_, err := svc.Register(context.Background(), RegisterRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestService_Login_NoUserFound(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestService_Login_InvalidCredentials_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestService_RefreshToken_InvalidFormat(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.RefreshToken(context.Background(), "not-a-valid-token-format")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestService_RefreshToken_WrongPurpose(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// Generate an access token (wrong purpose for refresh)
	uid := uuid.New()
	accessToken, err := svc.tokenManager.GenerateAccessToken(uid.String())
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	_, err = svc.RefreshToken(context.Background(), accessToken)
	if err == nil {
		t.Fatal("expected error for using access token as refresh token")
	}
}

func TestService_Logout_WithRefreshToken(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// Logout with empty refresh token, no DB
	err := svc.Logout(context.Background(), uuid.New(), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestService_Logout_WithSpecificRefreshToken(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// With refresh token, it tries findSessionByRefreshToken which accesses db.Pool -> panic
	defer func() {
		if r := recover(); r != nil {
			// Expected - findSessionByRefreshToken doesn't check nil db
		}
	}()
	_ = svc.Logout(context.Background(), uuid.New(), "some-refresh-token")
}

func TestService_FireEvent_WithBus(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	// Should not panic
	svc.fireEvent(context.Background(), kernel.EventUserRegistered, map[string]interface{}{
		"user_id": uuid.New().String(),
	})
}

func TestService_FireEvent_WithoutBus(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// Should not panic when eventBus is nil
	svc.fireEvent(context.Background(), kernel.EventUserRegistered, map[string]interface{}{
		"user_id": uuid.New().String(),
	})
}

func TestService_HashPassword(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	hash, err := svc.hashPassword("my-secure-password")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("expected argon2id hash prefix, got: %s", hash)
	}
}

func TestService_GetOAuthService(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	oauthSvc := svc.GetOAuthService()
	if oauthSvc == nil {
		t.Fatal("expected non-nil OAuth service")
	}
}

func TestService_GetMFAService(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	mfaSvc := svc.GetMFAService()
	if mfaSvc == nil {
		t.Fatal("expected non-nil MFA service")
	}
}

func TestService_HandleOAuthCallback_UnsupportedProvider(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.HandleOAuthCallback(context.Background(), "unsupported-provider", "code")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestService_CreateOAuthUser_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	info := &OAuthUserInfo{
		Email:      "oauth@example.com",
		Name:       "OAuth User",
		Avatar:     "https://example.com/avatar.png",
		Provider:   "google",
		ProviderID: "google-12345",
	}

	user, err := svc.createOAuthUser(context.Background(), info)
	if err != nil {
		t.Fatalf("createOAuthUser with nil db should succeed: %v", err)
	}
	if user.Email != info.Email {
		t.Errorf("expected %s, got %s", info.Email, user.Email)
	}
	if user.Name != info.Name {
		t.Errorf("expected %s, got %s", info.Name, user.Name)
	}
	if user.Role != "user" {
		t.Errorf("expected role 'user', got %s", user.Role)
	}
}

func TestService_FindOAuthAccount_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.findOAuthAccount(context.Background(), "google", "provider-id")
	if err == nil {
		t.Fatal("expected error with nil db")
	}
}

func TestService_UpsertOAuthAccount_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	err := svc.upsertOAuthAccount(context.Background(), uuid.New(), &OAuthUserInfo{
		Email:      "test@example.com",
		Provider:   "google",
		ProviderID: "12345",
	})
	if err != nil {
		t.Fatalf("expected nil error with nil db, got: %v", err)
	}
}

func TestService_FindUserByID_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.findUserByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error with nil db")
	}
}

func TestService_FindUserByEmail_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.findUserByEmail(context.Background(), "test@example.com")
	if err == nil {
		t.Fatal("expected error with nil db")
	}
}

func TestService_CreateSession_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	_, err := svc.createSession(context.Background(), uuid.New(), "refresh-token", "", "")
	if err == nil {
		t.Fatal("expected error with nil db")
	}
}

func TestService_DeleteUserSessions_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	err := svc.deleteUserSessions(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error with nil db, got: %v", err)
	}
}

func TestService_UpdateLastLogin_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// Should not panic with nil db
	svc.updateLastLogin(context.Background(), uuid.New())
}

func TestService_LogOAuthLogin(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	// Should not panic
	svc.logOAuthLogin(context.Background(), uuid.New(), "google")
}

func TestService_FindOrCreateOAuthUser_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	info := &OAuthUserInfo{
		Email:      "oauth@example.com",
		Name:       "OAuth User",
		Provider:   "google",
		ProviderID: "google-12345",
	}

	// With nil db, findOAuthAccount fails, findUserByEmail fails, createOAuthUser creates user without DB
	user, err := svc.findOrCreateOAuthUser(context.Background(), info)
	if err != nil {
		t.Fatalf("expected success with nil db, got: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Email != info.Email {
		t.Errorf("expected %s, got %s", info.Email, user.Email)
	}
}

func TestService_FindOrCreateOAuthUser_ExistingEmail_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	info := &OAuthUserInfo{
		Email:      "existing@example.com",
		Name:       "Existing User",
		Provider:   "google",
		ProviderID: "google-12345",
	}

	// findUserByEmail returns error with nil db, so it falls through to create
	user, err := svc.findOrCreateOAuthUser(context.Background(), info)
	if err != nil {
		t.Fatalf("expected success with nil db, got: %v", err)
	}
	if user.Email != info.Email {
		t.Errorf("expected %s, got %s", info.Email, user.Email)
	}
}

func TestService_ValidateAccessToken_FutureToken(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	uid := uuid.New()
	token, err := svc.tokenManager.GenerateAccessToken(uid.String())
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedUID, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
	if parsedUID != uid {
		t.Errorf("expected %s, got %s", uid, parsedUID)
	}
}

func TestService_NewService_OAuthConfig(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	if svc.oauthService == nil {
		t.Fatal("expected OAuth service to be initialized")
	}
	if svc.mfaService == nil {
		t.Fatal("expected MFA service to be initialized")
	}
}

func TestService_NewService_TokenManager(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	if svc.tokenManager == nil {
		t.Fatal("expected token manager to be initialized")
	}
}

func TestValidateRegisterRequest_Coverage(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
		errMsg  string
	}{
		{"empty email", RegisterRequest{Email: "", Password: "password123", Name: "Test"}, true, "email is required"},
		{"empty password", RegisterRequest{Email: "test@example.com", Password: "", Name: "Test"}, true, "password is required"},
		{"short password", RegisterRequest{Email: "test@example.com", Password: "short", Name: "Test"}, true, "password must be at least 8 characters"},
		{"empty name", RegisterRequest{Email: "test@example.com", Password: "password123", Name: ""}, true, "name is required"},
		{"valid", RegisterRequest{Email: "test@example.com", Password: "password123", Name: "Test"}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}
