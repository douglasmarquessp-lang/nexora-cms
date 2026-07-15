package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ctxKey string

func (k ctxKey) String() string { return string(k) }

const CtxUserID ctxKey = "user_id"

func GetUserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(CtxUserID)
	if v == nil {
		return uuid.Nil, false
	}
	uid, ok := v.(uuid.UUID)
	return uid, ok
}

type User struct {
	ID           uuid.UUID              `json:"id"`
	Email        string                 `json:"email"`
	PasswordHash string                 `json:"-"`
	Name         string                 `json:"name"`
	Avatar       string                 `json:"avatar,omitempty"`
	Role         string                 `json:"role"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	MFAEnabled   bool                   `json:"mfa_enabled"`
	MFASecret    string                 `json:"-"`
	LastLogin    *time.Time             `json:"last_login,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	RefreshToken string    `json:"-"`
	DeviceInfo   string    `json:"device_info,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type OAuthAccount struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Provider     string    `json:"provider"`
	ProviderID   string    `json:"provider_id"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type MFAConfig struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Secret    string    `json:"-"`
	Enabled   bool      `json:"enabled"`
	Method    string    `json:"method"`
	BackupCodes []string `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code,omitempty"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	User         User   `json:"user"`
}

type OAuthRequest struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	Redirect string `json:"redirect_uri"`
}

type OAuthURLResponse struct {
	URL string `json:"url"`
}

type MFAEnrollResponse struct {
	Secret     string   `json:"secret"`
	QRCodeURL  string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

type MFAVerifyRequest struct {
	Code string `json:"code"`
}

type MFADisableRequest struct {
	Code   string `json:"code"`
	Password string `json:"password"`
}
