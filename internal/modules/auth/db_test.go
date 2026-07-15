package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/api/rest"
	"nexora/internal/pkg/audit"
	pkgauth "nexora/internal/pkg/auth"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func setupMockDB(t *testing.T) (*Service, pgxmock.PgxPoolIface, *config.Config) {
	t.Helper()
	cfg := testConfig()
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	svc := NewService(cfg, log, &database.Database{Pool: mock})
	// Replace audit logger with one that skips (nil pool) so tests don't need audit INSERT expectations
	svc.auditLog = audit.New(nil, log)
	return svc, mock, cfg
}

func closeMock(t *testing.T, mock pgxmock.PgxPoolIface) {
	_ = t
	mock.Close()
}

func TestService_Register_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("new@example.com").
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "new@example.com", pgxmock.AnyArg(), "New User").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
			AddRow(uuid.New(), uuid.New(), "refresh-token", "", "", time.Now().Add(time.Hour), time.Now()))

	defer closeMock(t, mock)

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "new@example.com",
		Password: "securePassword123!",
		Name:     "New User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.User.Email != "new@example.com" {
		t.Errorf("expected new@example.com, got %s", resp.User.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "existing@example.com", "hash", "Existing", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("existing@example.com").
		WillReturnRows(rows)

	defer closeMock(t, mock)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "existing@example.com",
		Password: "securePassword123!",
		Name:     "Existing User",
	})
	if err != ErrEmailAlreadyExists {
		t.Errorf("expected ErrEmailAlreadyExists, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Register_ValidationError(t *testing.T) {
	svc, _, _ := setupMockDB(t)

	_, err := svc.Register(context.Background(), RegisterRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestService_HashVerifyRoundTrip(t *testing.T) {
	svc, _, _ := setupMockDB(t)

	password := "test-password-123"
	hash, err := svc.hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}
	t.Logf("hash: %s", hash)

	ok, err := pkgauth.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatal("password should match")
	}
}

func TestService_Login_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	hash, err := svc.hashPassword("correct-password")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "user@example.com", hash, "User", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery("SELECT").
		WithArgs("user@example.com").
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE users SET last_login = NOW\(\), updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(uid, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
			AddRow(uuid.New(), uid, "new-refresh-token", "", "", time.Now().Add(time.Hour), time.Now()))

	defer closeMock(t, mock)

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Login_MFARequired(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	hash, _ := svc.hashPassword("password")

	mfaSecret, _ := svc.mfaService.GenerateSecret()
	mfaSecretPtr := &mfaSecret
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", hash, "MFA User", "", "user", []byte("{}"), true, mfaSecretPtr, nil, time.Now(), time.Now())

	mock.ExpectQuery("SELECT").
		WithArgs("mfa@example.com").
		WillReturnRows(rows)

	defer closeMock(t, mock)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "mfa@example.com",
		Password: "password",
	})
	if err != ErrMFARequired {
		t.Errorf("expected ErrMFARequired, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Login_InvalidMFACode(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	hash, _ := svc.hashPassword("password")

	mfaSecret, _ := svc.mfaService.GenerateSecret()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", hash, "MFA User", "", "user", []byte("{}"), true, &mfaSecret, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("mfa@example.com").
		WillReturnRows(rows)

	defer closeMock(t, mock)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "mfa@example.com",
		Password: "password",
		MFACode:  "000000",
	})
	if err != ErrInvalidMFACode {
		t.Errorf("expected ErrInvalidMFACode, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_RefreshToken_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	refreshToken, err := svc.tokenManager.GenerateRefreshToken(uid.String())
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	sessionID := uuid.New()
	sessionRows := pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
		AddRow(sessionID, uid, refreshToken, "", "", time.Now().Add(time.Hour), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM sessions WHERE refresh_token = \$1 AND expires_at > NOW\(\)`).
		WithArgs(refreshToken).
		WillReturnRows(sessionRows)

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "user@example.com", "hash", "User", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(userRows)

	mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1`).
		WithArgs(sessionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(uid, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
			AddRow(uuid.New(), uid, "new-refresh-token", "", "", time.Now().Add(time.Hour), time.Now()))

	defer closeMock(t, mock)

	resp, err := svc.RefreshToken(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_RefreshToken_ExpiredSession(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	refreshToken, err := svc.tokenManager.GenerateRefreshToken(uid.String())
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	sessionID := uuid.New()
	sessionRows := pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
		AddRow(sessionID, uid, refreshToken, "", "", time.Now().Add(-time.Hour), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM sessions WHERE refresh_token = \$1 AND expires_at > NOW\(\)`).
		WithArgs(refreshToken).
		WillReturnRows(sessionRows)

	defer closeMock(t, mock)

	_, err = svc.RefreshToken(context.Background(), refreshToken)
	if err != ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Logout_WithValidRefreshToken(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	refreshToken := "some-refresh-token"
	sessionID := uuid.New()

	sessionRows := pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
		AddRow(sessionID, uid, refreshToken, "", "", time.Now().Add(time.Hour), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM sessions WHERE refresh_token = \$1 AND expires_at > NOW\(\)`).
		WithArgs(refreshToken).
		WillReturnRows(sessionRows)

	mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1`).
		WithArgs(sessionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`DELETE FROM sessions WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	defer closeMock(t, mock)

	err := svc.Logout(context.Background(), uid, refreshToken)
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_FindUserByID_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "found@example.com", "hash", "Found", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(rows)

	defer closeMock(t, mock)

	user, err := svc.findUserByID(context.Background(), uid)
	if err != nil {
		t.Fatalf("findUserByID failed: %v", err)
	}
	if user.Email != "found@example.com" {
		t.Errorf("expected found@example.com, got %s", user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_FindUserByEmail_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "test@example.com", "hash", "Test", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("test@example.com").
		WillReturnRows(rows)

	defer closeMock(t, mock)

	user, err := svc.findUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("findUserByEmail failed: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_CreateSession_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	refreshToken := "new-refresh-token"

	rows := pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
		AddRow(uuid.New(), uid, refreshToken, "device", "127.0.0.1", time.Now().Add(time.Hour), time.Now())

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(uid, refreshToken, "device", "127.0.0.1", pgxmock.AnyArg()).
		WillReturnRows(rows)

	defer closeMock(t, mock)

	session, err := svc.createSession(context.Background(), uid, refreshToken, "device", "127.0.0.1")
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}
	if session.RefreshToken != refreshToken {
		t.Errorf("expected %s, got %s", refreshToken, session.RefreshToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_DeleteUserSessions_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	mock.ExpectExec(`DELETE FROM sessions WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	defer closeMock(t, mock)

	err := svc.deleteUserSessions(context.Background(), uid)
	if err != nil {
		t.Fatalf("deleteUserSessions failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_UpdateLastLogin_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	mock.ExpectExec(`UPDATE users SET last_login = NOW\(\), updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	defer closeMock(t, mock)

	svc.updateLastLogin(context.Background(), uid)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_DeleteSession_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	sessionID := uuid.New()

	mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1`).
		WithArgs(sessionID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	defer closeMock(t, mock)

	err := svc.deleteSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("deleteSession failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_FindSessionByRefreshToken_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	refreshToken := "valid-refresh-token"
	sessionID := uuid.New()

	rows := pgxmock.NewRows([]string{"id", "user_id", "refresh_token", "device_info", "ip_address", "expires_at", "created_at"}).
		AddRow(sessionID, uid, refreshToken, "", "", time.Now().Add(time.Hour), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM sessions WHERE refresh_token = \$1 AND expires_at > NOW\(\)`).
		WithArgs(refreshToken).
		WillReturnRows(rows)

	defer closeMock(t, mock)

	session, err := svc.findSessionByRefreshToken(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("findSessionByRefreshToken failed: %v", err)
	}
	if session.RefreshToken != refreshToken {
		t.Errorf("expected %s, got %s", refreshToken, session.RefreshToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_FindOAuthAccount_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	rows := pgxmock.NewRows([]string{"id", "user_id", "provider", "provider_id", "access_token", "refresh_token", "expires_at", "created_at"}).
		AddRow(uuid.New(), uid, "google", "google-123", "access", "refresh", nil, time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM oauth_accounts WHERE provider = \$1 AND provider_id = \$2`).
		WithArgs("google", "google-123").
		WillReturnRows(rows)

	defer closeMock(t, mock)

	account, err := svc.findOAuthAccount(context.Background(), "google", "google-123")
	if err != nil {
		t.Fatalf("findOAuthAccount failed: %v", err)
	}
	if account.Provider != "google" {
		t.Errorf("expected google, got %s", account.Provider)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_UpsertOAuthAccount_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	mock.ExpectExec(`INSERT INTO oauth_accounts`).
		WithArgs(uid, "google", "google-123").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	defer closeMock(t, mock)

	err := svc.upsertOAuthAccount(context.Background(), uid, &OAuthUserInfo{
		Email:      "test@example.com",
		Provider:   "google",
		ProviderID: "google-123",
	})
	if err != nil {
		t.Fatalf("upsertOAuthAccount failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_CreateOAuthUser_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "oauth@example.com", "OAuth User", "https://avatar.url").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	defer closeMock(t, mock)

	user, err := svc.createOAuthUser(context.Background(), &OAuthUserInfo{
		Email:      "oauth@example.com",
		Name:       "OAuth User",
		Avatar:     "https://avatar.url",
		Provider:   "google",
		ProviderID: "google-123",
	})
	if err != nil {
		t.Fatalf("createOAuthUser failed: %v", err)
	}
	if user.Email != "oauth@example.com" {
		t.Errorf("expected oauth@example.com, got %s", user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_HandleOAuthCallback_ProviderError(t *testing.T) {
	svc, _, _ := setupMockDB(t)

	_, err := svc.HandleOAuthCallback(context.Background(), "unsupported-provider", "code")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestService_FindOrCreateOAuthUser_ExistingOAuthAccount(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	rows := pgxmock.NewRows([]string{"id", "user_id", "provider", "provider_id", "access_token", "refresh_token", "expires_at", "created_at"}).
		AddRow(uuid.New(), uid, "google", "google-123", "access", "refresh", nil, time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM oauth_accounts WHERE provider = \$1 AND provider_id = \$2`).
		WithArgs("google", "google-123").
		WillReturnRows(rows)

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "test@example.com", "hash", "Test", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(userRows)

	defer closeMock(t, mock)

	user, err := svc.findOrCreateOAuthUser(context.Background(), &OAuthUserInfo{
		Email:      "test@example.com",
		Name:       "Test User",
		Provider:   "google",
		ProviderID: "google-123",
	})
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if user.ID != uid {
		t.Errorf("expected %s, got %s", uid, user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_FindOrCreateOAuthUser_CreateNew(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mockErr := fmt.Errorf("not found")
	// findOAuthAccount fails
	mock.ExpectQuery(`SELECT[\s\S]+FROM oauth_accounts WHERE provider = \$1 AND provider_id = \$2`).
		WithArgs("google", "google-123").
		WillReturnError(mockErr)

	// findUserByEmail fails
	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE email = \$1 AND deleted_at IS NULL`).
		WithArgs("new@example.com").
		WillReturnError(mockErr)

	// createOAuthUser succeeds
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "new@example.com", "New User", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// upsertOAuthAccount succeeds
	mock.ExpectExec(`INSERT INTO oauth_accounts`).
		WithArgs(pgxmock.AnyArg(), "google", "google-123").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	defer closeMock(t, mock)

	user, err := svc.findOrCreateOAuthUser(context.Background(), &OAuthUserInfo{
		Email:      "new@example.com",
		Name:       "New User",
		Avatar:     "",
		Provider:   "google",
		ProviderID: "google-123",
	})
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("expected new@example.com, got %s", user.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMFAService_Enroll_DB_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(userRows)

	mock.ExpectExec(`INSERT INTO mfa_configs`).
		WithArgs(uid, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	defer closeMock(t, mock)

	s := NewMFAService()
	resp, err := s.Enroll(context.Background(), uid, svc)
	if err != nil {
		t.Fatalf("Enroll failed: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if len(resp.BackupCodes) != 8 {
		t.Errorf("expected 8 backup codes, got %d", len(resp.BackupCodes))
	}
	if resp.QRCodeURL == "" {
		t.Error("expected non-empty QR code URL")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMFAService_Enroll_DB_AlreadyEnabled(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), true, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(userRows)

	defer closeMock(t, mock)

	s := NewMFAService()
	_, err := s.Enroll(context.Background(), uid, svc)
	if err == nil {
		t.Fatal("expected error when MFA already enabled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMFAService_VerifyAndEnable_DB_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret, _ := svc.mfaService.GenerateSecret()

	rows := pgxmock.NewRows([]string{"secret"}).
		AddRow(secret)

	mock.ExpectQuery(`SELECT secret FROM mfa_configs WHERE user_id = \$1 AND enabled = false`).
		WithArgs(uid).
		WillReturnRows(rows)

	mock.ExpectExec(`UPDATE mfa_configs SET enabled = true, updated_at = NOW\(\) WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE users SET mfa_enabled = true, updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	defer closeMock(t, mock)

	code, err := svc.mfaService.GenerateTOTP(secret)
	if err != nil {
		t.Fatalf("GenerateTOTP failed: %v", err)
	}

	err = svc.mfaService.VerifyAndEnable(context.Background(), uid, code, svc)
	if err != nil {
		t.Fatalf("VerifyAndEnable failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestMFAService_Disable_DB_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), true, nil, nil, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT[\s\S]+FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(uid).
		WillReturnRows(userRows)

	mock.ExpectExec(`DELETE FROM mfa_configs WHERE user_id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`UPDATE users SET mfa_enabled = false, mfa_secret = NULL, updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	defer closeMock(t, mock)

	s := NewMFAService()
	err := s.Disable(context.Background(), uid, "password", svc)
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_NewService_WithDB(t *testing.T) {
	svc, mock, _ := setupMockDB(t)
	defer closeMock(t, mock)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.db == nil {
		t.Fatal("expected non-nil database")
	}
	if svc.db.Pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestService_NewService_AuditLog(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)

	svc := NewService(cfg, log, nil)
	if svc.auditLog == nil {
		t.Fatal("expected audit log to be initialized")
	}
}

func TestMockPoolType(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer closeMock(t, mock)

	db := &database.Database{Pool: mock}
	if db.Pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestService_Register_DBError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectQuery("SELECT").WithArgs("fail@example.com").WillReturnRows(pgxmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO users").WithArgs(pgxmock.AnyArg(), "fail@example.com", pgxmock.AnyArg(), "Fail User").
		WillReturnError(fmt.Errorf("db error"))
	defer closeMock(t, mock)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "fail@example.com",
		Password: "password123",
		Name:     "Fail User",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to create user") {
		t.Errorf("expected DB error, got: %v", err)
	}
}

func TestService_Register_SessionError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectQuery("SELECT").WithArgs("fail@example.com").WillReturnRows(pgxmock.NewRows(nil))
	mock.ExpectExec("INSERT INTO users").WithArgs(pgxmock.AnyArg(), "fail@example.com", pgxmock.AnyArg(), "Fail User").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery("INSERT INTO sessions").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("session error"))
	defer closeMock(t, mock)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "fail@example.com",
		Password: "password123",
		Name:     "Fail User",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("expected session error, got: %v", err)
	}
}

func TestService_FindUserByID_WithMFASecret(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret := "test-mfa-secret"
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "test@example.com", "hash", "Test", "", "user", []byte("{}"), true, &secret, &now, now, now)
	mock.ExpectQuery("SELECT").WithArgs(uid).WillReturnRows(rows)
	defer closeMock(t, mock)

	user, err := svc.findUserByID(context.Background(), uid)
	if err != nil {
		t.Fatalf("findUserByID failed: %v", err)
	}
	if user.MFASecret != secret {
		t.Errorf("expected MFASecret=%s, got %s", secret, user.MFASecret)
	}
	if !user.MFAEnabled {
		t.Error("expected MFAEnabled=true")
	}
	if user.LastLogin == nil {
		t.Error("expected non-nil LastLogin")
	}
}

func TestService_FindSessionByRefreshToken_NotFound(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectQuery("SELECT").WithArgs("nonexistent-token").
		WillReturnRows(pgxmock.NewRows(nil))
	defer closeMock(t, mock)

	_, err := svc.findSessionByRefreshToken(context.Background(), "nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestMFAService_VerifyAndEnable_InvalidCode(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret, _ := svc.mfaService.GenerateSecret()

	rows := pgxmock.NewRows([]string{"secret"}).
		AddRow(secret)

	mock.ExpectQuery(`SELECT secret FROM mfa_configs`).WithArgs(uid).WillReturnRows(rows)
	defer closeMock(t, mock)

	err := svc.mfaService.VerifyAndEnable(context.Background(), uid, "000000", svc)
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
}

func TestMFAService_VerifyAndEnable_QueryError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	mock.ExpectQuery(`SELECT secret FROM mfa_configs`).WithArgs(uid).
		WillReturnError(fmt.Errorf("no pending enrollment"))
	defer closeMock(t, mock)

	err := svc.mfaService.VerifyAndEnable(context.Background(), uid, "123456", svc)
	if err == nil {
		t.Fatal("expected error for query failure")
	}
}

func TestMFAService_VerifyAndEnable_UpdateMFACodeError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret, _ := svc.mfaService.GenerateSecret()
	code, _ := svc.mfaService.GenerateTOTP(secret)

	rows := pgxmock.NewRows([]string{"secret"}).AddRow(secret)
	mock.ExpectQuery(`SELECT secret FROM mfa_configs`).WithArgs(uid).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE mfa_configs SET enabled`).WithArgs(uid).
		WillReturnError(fmt.Errorf("update error"))
	defer closeMock(t, mock)

	err := svc.mfaService.VerifyAndEnable(context.Background(), uid, code, svc)
	if err == nil {
		t.Fatal("expected error for update failure")
	}
}

func TestMFAService_VerifyAndEnable_UpdateUserError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret, _ := svc.mfaService.GenerateSecret()
	code, _ := svc.mfaService.GenerateTOTP(secret)

	rows := pgxmock.NewRows([]string{"secret"}).AddRow(secret)
	mock.ExpectQuery(`SELECT secret FROM mfa_configs`).WithArgs(uid).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE mfa_configs SET enabled`).WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE users SET mfa_enabled`).WithArgs(uid).
		WillReturnError(fmt.Errorf("user update error"))
	defer closeMock(t, mock)

	err := svc.mfaService.VerifyAndEnable(context.Background(), uid, code, svc)
	if err == nil {
		t.Fatal("expected error for user update failure")
	}
}

func TestMFAService_GenerateSecret_Success(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Error("expected non-empty secret")
	}
}

func TestService_Login_CreateSessionError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	hash, _ := svc.hashPassword("password")

	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "user@example.com", hash, "User", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs("user@example.com").WillReturnRows(rows)
	mock.ExpectQuery("INSERT INTO sessions").WithArgs(uid, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("session error"))
	defer closeMock(t, mock)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "password",
	})
	if err == nil || !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("expected session error, got: %v", err)
	}
}

func TestHandler_DisableMFA_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)
	h := NewHandler(svc, logger.New(testConfig()))

	uid := uuid.New()
	hash, _ := svc.hashPassword("password")

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", hash, "MFA User", "", "user", []byte("{}"), true, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs(uid).WillReturnRows(userRows)
	mock.ExpectExec("DELETE FROM mfa_configs").WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE users SET mfa_enabled").WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	defer mock.Close()

	body := MFADisableRequest{Password: "password"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/auth/mfa/disable", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(authContext(uid))
	w := httptest.NewRecorder()

	ctx := &rest.Context{ResponseWriter: w, Request: req}
	h.DisableMFA(ctx)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_EnrollMFA_Success(t *testing.T) {
	svc, mock, _ := setupMockDB(t)
	h := NewHandler(svc, logger.New(testConfig()))

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs(uid).WillReturnRows(userRows)
	mock.ExpectExec("INSERT INTO mfa_configs").WithArgs(uid, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	defer mock.Close()

	req := httptest.NewRequest("POST", "/api/auth/mfa/enroll", nil)
	req = req.WithContext(authContext(uid))
	w := httptest.NewRecorder()

	ctx := &rest.Context{ResponseWriter: w, Request: req}
	h.EnrollMFA(ctx)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp MFAEnrollResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected non-empty secret")
	}
}

func TestMFAService_Disable_DB_DeleteError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), true, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs(uid).WillReturnRows(userRows)
	mock.ExpectExec("DELETE FROM mfa_configs").WithArgs(uid).
		WillReturnError(fmt.Errorf("delete error"))
	defer closeMock(t, mock)

	s := NewMFAService()
	err := s.Disable(context.Background(), uid, "password", svc)
	if err == nil {
		t.Fatal("expected error for delete failure")
	}
}

func TestMFAService_Disable_DB_UpdateError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()

	userRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "mfa@example.com", "hash", "MFA User", "", "user", []byte("{}"), true, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs(uid).WillReturnRows(userRows)
	mock.ExpectExec("DELETE FROM mfa_configs").WithArgs(uid).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("UPDATE users SET mfa_enabled").WithArgs(uid).
		WillReturnError(fmt.Errorf("update error"))
	defer closeMock(t, mock)

	s := NewMFAService()
	err := s.Disable(context.Background(), uid, "password", svc)
	if err == nil {
		t.Fatal("expected error for update failure")
	}
}

func TestService_FindOrCreateOAuthUser_ExistingEmail(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	// No OAuth account found
	mock.ExpectQuery("SELECT").WithArgs("google", "provider123").
		WillReturnRows(pgxmock.NewRows(nil))
	// User with this email already exists
	existingUserRows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "existing@example.com", "hash", "Existing", "", "user", []byte("{}"), false, nil, nil, time.Now(), time.Now())
	mock.ExpectQuery("SELECT").WithArgs("existing@example.com").WillReturnRows(existingUserRows)
	// Should link OAuth account to existing user
	mock.ExpectExec("INSERT INTO oauth_accounts").WithArgs(uid, "google", "provider123").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	defer closeMock(t, mock)

	user, err := svc.findOrCreateOAuthUser(context.Background(), &OAuthUserInfo{
		Provider:   "google",
		ProviderID: "provider123",
		Email:      "existing@example.com",
		Name:       "Existing User",
	})
	if err != nil {
		t.Fatalf("findOrCreateOAuthUser failed: %v", err)
	}
	if user.ID != uid {
		t.Errorf("expected user ID %v, got %v", uid, user.ID)
	}
}

func TestService_CreateOAuthUser_DBError(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	mock.ExpectExec("INSERT INTO users").WithArgs(pgxmock.AnyArg(), "new@example.com", pgxmock.AnyArg(), "New").
		WillReturnError(fmt.Errorf("db error"))
	defer closeMock(t, mock)

	_, err := svc.createOAuthUser(context.Background(), &OAuthUserInfo{
		Provider:   "google",
		ProviderID: "provider123",
		Email:      "new@example.com",
		Name:       "New",
	})
	if err == nil {
		t.Fatal("expected error for DB failure")
	}
}

func TestService_FindUserByEmail_WithMFASecret(t *testing.T) {
	svc, mock, _ := setupMockDB(t)

	uid := uuid.New()
	secret := "mfa-test-secret"
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "name", "avatar", "role", "metadata", "mfa_enabled", "mfa_secret", "last_login", "created_at", "updated_at"}).
		AddRow(uid, "test@example.com", "hash", "Test", "", "user", []byte("{}"), true, &secret, &now, now, now)
	mock.ExpectQuery("SELECT").WithArgs("test@example.com").WillReturnRows(rows)
	defer closeMock(t, mock)

	user, err := svc.findUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("findUserByEmail failed: %v", err)
	}
	if user.MFASecret != secret {
		t.Errorf("expected MFASecret=%s, got %s", secret, user.MFASecret)
	}
	if !user.MFAEnabled {
		t.Error("expected MFAEnabled=true")
	}
	if user.LastLogin == nil {
		t.Error("expected non-nil LastLogin")
	}
}

