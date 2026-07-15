package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/auth"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrSessionExpired     = errors.New("session expired")
	ErrMFARequired        = errors.New("MFA code required")
	ErrInvalidMFACode     = errors.New("invalid MFA code")
	ErrUserNotFound       = errors.New("user not found")
	ErrOAuthProviderError = errors.New("OAuth provider error")
	ErrOAuthEmailExists   = errors.New("an account with this email already exists")
)

type Service struct {
	cfg          *config.AuthConfig
	log          *logger.Logger
	db           *database.Database
	tokenManager *auth.TokenManager
	mfaService   *MFAService
	oauthService *OAuthService
	eventBus     *kernel.EventBus
	auditLog     *audit.Logger
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database) *Service {
	tm := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTAccessTTL, cfg.Auth.JWTRefreshTTL)

	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}

	return &Service{
		cfg:          &cfg.Auth,
		log:          log,
		db:           db,
		tokenManager: tm,
		mfaService:   NewMFAService(),
		oauthService: NewOAuthService(&cfg.OAuth, log),
		auditLog:     audit.New(pool, log),
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	existing, err := s.findUserByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, role, metadata)
		 VALUES ($1, $2, $3, $4, 'user', '{}')`,
		userID, req.Email, hash, req.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if _, err := s.createSession(ctx, userID, refreshToken, "", ""); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.auditLog.LogUserAction(ctx, userID, audit.ActionUserRegistered, map[string]interface{}{
		"email": req.Email,
	})

	s.fireEvent(ctx, kernel.EventUserRegistered, map[string]interface{}{
		"user_id": userID.String(),
		"email":   req.Email,
	})

	user := User{
		ID:    userID,
		Email: req.Email,
		Name:  req.Name,
		Role:  "user",
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.findUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	if user.MFAEnabled {
		if req.MFACode == "" {
			return nil, ErrMFARequired
		}
		if !s.mfaService.ValidateCode(user.MFASecret, req.MFACode) {
			return nil, ErrInvalidMFACode
		}
	}

	s.updateLastLogin(ctx, user.ID)

	s.auditLog.LogUserAction(ctx, user.ID, audit.ActionUserLogin, nil)

	s.fireEvent(ctx, kernel.EventUserLogin, map[string]interface{}{
		"user_id": user.ID.String(),
	})

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if _, err := s.createSession(ctx, user.ID, refreshToken, "", ""); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
		User:         *user,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	userID, err := s.tokenManager.ValidateToken(refreshToken, "refresh")
	if err != nil {
		return nil, ErrInvalidToken
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	session, err := s.findSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrSessionExpired
	}

	if time.Now().After(session.ExpiresAt) {
		s.deleteSession(ctx, session.ID)
		return nil, ErrSessionExpired
	}

	user, err := s.findUserByID(ctx, uid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	s.deleteSession(ctx, session.ID)

	newAccessToken, err := s.tokenManager.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if _, err := s.createSession(ctx, user.ID, newRefreshToken, "", ""); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
		User:         *user,
	}, nil
}

func (s *Service) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	if refreshToken != "" {
		session, err := s.findSessionByRefreshToken(ctx, refreshToken)
		if err == nil && session.UserID == userID {
			s.deleteSession(ctx, session.ID)
		}
	}

	s.deleteUserSessions(ctx, userID)

	s.auditLog.LogUserAction(ctx, userID, audit.ActionUserLogout, nil)

	s.fireEvent(ctx, kernel.EventUserLogout, map[string]interface{}{
		"user_id": userID.String(),
	})

	return nil
}

func (s *Service) ValidateAccessToken(token string) (uuid.UUID, error) {
	userID, err := s.tokenManager.ValidateToken(token, "access")
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return uid, nil
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.findUserByID(ctx, id)
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) HandleOAuthCallback(ctx context.Context, provider, code string) (*AuthResponse, error) {
	info, err := s.oauthService.ExchangeCodeAndGetUserInfo(ctx, provider, code)
	if err != nil {
		return nil, err
	}

	user, err := s.findOrCreateOAuthUser(ctx, info)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if _, err := s.createSession(ctx, user.ID, refreshToken, "", ""); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.updateLastLogin(ctx, user.ID)

	s.auditLog.LogUserAction(ctx, user.ID, audit.ActionOAuthLogin, map[string]interface{}{
		"provider": provider,
	})

	s.fireEvent(ctx, kernel.EventOAuthLogin, map[string]interface{}{
		"user_id":  user.ID.String(),
		"provider": provider,
	})

	user.PasswordHash = ""

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
		User:         *user,
	}, nil
}

func (s *Service) findOrCreateOAuthUser(ctx context.Context, info *OAuthUserInfo) (*User, error) {
	account, err := s.findOAuthAccount(ctx, info.Provider, info.ProviderID)
	if err == nil && account != nil {
		user, err := s.findUserByID(ctx, account.UserID)
		if err == nil {
			return user, nil
		}
	}

	user, err := s.findUserByEmail(ctx, info.Email)
	if err != nil {
		user, err = s.createOAuthUser(ctx, info)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err := s.upsertOAuthAccount(ctx, user.ID, info); err != nil {
		return nil, fmt.Errorf("failed to link OAuth account: %w", err)
	}

	return user, nil
}

func (s *Service) createOAuthUser(ctx context.Context, info *OAuthUserInfo) (*User, error) {
	if s.db == nil || s.db.Pool == nil {
		return &User{
			ID:    uuid.New(),
			Email: info.Email,
			Name:  info.Name,
			Avatar: info.Avatar,
			Role:  "user",
		}, nil
	}

	userID := uuid.New()
	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, avatar, role, metadata)
		 VALUES ($1, $2, '', $3, $4, 'user', '{}')`,
		userID, info.Email, info.Name, info.Avatar,
	)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:    userID,
		Email: info.Email,
		Name:  info.Name,
		Avatar: info.Avatar,
		Role:  "user",
	}, nil
}

func (s *Service) findOAuthAccount(ctx context.Context, provider, providerID string) (*OAuthAccount, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrUserNotFound
	}

	var account OAuthAccount
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, user_id, provider, provider_id, COALESCE(access_token, ''), COALESCE(refresh_token, ''), expires_at, created_at
		 FROM oauth_accounts WHERE provider = $1 AND provider_id = $2`,
		provider, providerID,
	).Scan(
		&account.ID, &account.UserID, &account.Provider, &account.ProviderID,
		&account.AccessToken, &account.RefreshToken, &account.ExpiresAt, &account.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *Service) upsertOAuthAccount(ctx context.Context, userID uuid.UUID, info *OAuthUserInfo) error {
	if s.db == nil || s.db.Pool == nil {
		return nil
	}

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_id, access_token, refresh_token)
		 VALUES ($1, $2, $3, '', '')
		 ON CONFLICT (provider, provider_id) DO UPDATE SET user_id = EXCLUDED.user_id`,
		userID, info.Provider, info.ProviderID,
	)
	return err
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, "")
	}
}

func (s *Service) logOAuthLogin(ctx context.Context, userID uuid.UUID, provider string) {
	s.auditLog.LogUserAction(ctx, userID, audit.ActionOAuthLogin, map[string]interface{}{
		"provider": provider,
	})
}

func (s *Service) GetOAuthService() *OAuthService {
	return s.oauthService
}

func (s *Service) GetMFAService() *MFAService {
	return s.mfaService
}

func (s *Service) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		64*1024, 3, 4,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func (s *Service) findUserByEmail(ctx context.Context, email string) (*User, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrUserNotFound
	}

	var user User
	var metadata []byte
	var mfaSecret *string
	var lastLogin *time.Time

	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, COALESCE(avatar, ''), role,
		        COALESCE(metadata::text, '{}'), COALESCE(mfa_enabled, false), mfa_secret,
		        last_login, created_at, updated_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Avatar,
		&user.Role, &metadata, &user.MFAEnabled, &mfaSecret,
		&lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if mfaSecret != nil {
		user.MFASecret = *mfaSecret
	}
	if lastLogin != nil {
		user.LastLogin = lastLogin
	}

	return &user, nil
}

func (s *Service) findUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrUserNotFound
	}

	var user User
	var metadata []byte
	var mfaSecret *string
	var lastLogin *time.Time

	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, COALESCE(avatar, ''), role,
		        COALESCE(metadata::text, '{}'), COALESCE(mfa_enabled, false), mfa_secret,
		        last_login, created_at, updated_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Avatar,
		&user.Role, &metadata, &user.MFAEnabled, &mfaSecret,
		&lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if mfaSecret != nil {
		user.MFASecret = *mfaSecret
	}
	if lastLogin != nil {
		user.LastLogin = lastLogin
	}

	return &user, nil
}

func (s *Service) createSession(ctx context.Context, userID uuid.UUID, refreshToken, deviceInfo, ipAddress string) (*Session, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrSessionExpired
	}

	var session Session
	err := s.db.Pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, refresh_token, device_info, ip_address, expires_at)
		 VALUES ($1, $2, $3, $4, NOW() + $5::interval)
		 RETURNING id, user_id, refresh_token, COALESCE(device_info, ''), COALESCE(ip_address, ''), expires_at, created_at`,
		userID, refreshToken, deviceInfo, ipAddress, fmt.Sprintf("%.0f seconds", s.cfg.JWTRefreshTTL.Seconds()),
	).Scan(
		&session.ID, &session.UserID, &session.RefreshToken,
		&session.DeviceInfo, &session.IPAddress,
		&session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Service) findSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	var session Session
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token, COALESCE(device_info, ''), COALESCE(ip_address, ''), expires_at, created_at
		 FROM sessions WHERE refresh_token = $1 AND expires_at > NOW()`,
		refreshToken,
	).Scan(
		&session.ID, &session.UserID, &session.RefreshToken,
		&session.DeviceInfo, &session.IPAddress,
		&session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *Service) deleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (s *Service) deleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	if s.db == nil || s.db.Pool == nil {
		return nil
	}
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (s *Service) updateLastLogin(ctx context.Context, userID uuid.UUID) {
	if s.db == nil || s.db.Pool == nil {
		return
	}
	_, _ = s.db.Pool.Exec(ctx, `UPDATE users SET last_login = NOW(), updated_at = NOW() WHERE id = $1`, userID)
}

func validateRegisterRequest(req RegisterRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if req.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
