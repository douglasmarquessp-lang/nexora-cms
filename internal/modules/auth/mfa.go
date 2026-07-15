package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
)

type MFAService struct{}

func NewMFAService() *MFAService {
	return &MFAService{}
}

func (s *MFAService) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate MFA secret: %w", err)
	}

	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

func (s *MFAService) GenerateTOTP(secret string) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid secret: %w", err)
	}

	counter := time.Now().Unix() / 30

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	code := truncated % 1000000

	return fmt.Sprintf("%06d", code), nil
}

func (s *MFAService) ValidateCode(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}

	expected, err := s.GenerateTOTP(secret)
	if err != nil {
		return false
	}

	if hmac.Equal([]byte(expected), []byte(code)) {
		return true
	}

	// Check adjacent time steps for clock drift
	for i := -1; i <= 1; i++ {
		ctr := (time.Now().Unix() / 30) + int64(i)
		if ctr == time.Now().Unix()/30 {
			continue
		}
		altCode, err := s.generateTOTPForCounter(secret, ctr)
		if err == nil && altCode == code {
			return true
		}
	}

	return false
}

func (s *MFAService) ValidateBackupCode(hashedCodes []string, code string) (bool, string) {
	if len(hashedCodes) == 0 || code == "" {
		return false, ""
	}

	for i, hc := range hashedCodes {
		if hc == hashBackupCode(code) {
			remaining := make([]string, 0, len(hashedCodes)-1)
			remaining = append(remaining, hashedCodes[:i]...)
			remaining = append(remaining, hashedCodes[i+1:]...)
			return true, strings.Join(remaining, ",")
		}
	}

	return false, ""
}

func (s *MFAService) GenerateBackupCodes() []string {
	codes := make([]string, 8)
	for i := range codes {
		b := make([]byte, 4)
		rand.Read(b)
		codes[i] = fmt.Sprintf("%08x", b)
	}
	return codes
}

func (s *MFAService) HashBackupCodes(codes []string) []string {
	hashed := make([]string, len(codes))
	for i, c := range codes {
		hashed[i] = hashBackupCode(c)
	}
	return hashed
}

func (s *MFAService) Enroll(ctx context.Context, userID uuid.UUID, svc *Service) (*MFAEnrollResponse, error) {
	user, err := svc.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.MFAEnabled {
		return nil, fmt.Errorf("MFA already enabled")
	}

	secret, err := s.GenerateSecret()
	if err != nil {
		return nil, err
	}

	backupCodes := s.GenerateBackupCodes()
	hashedBackupCodes := s.HashBackupCodes(backupCodes)

	_, err = svc.db.Pool.Exec(ctx,
		`INSERT INTO mfa_configs (user_id, secret, enabled, method, backup_codes)
		 VALUES ($1, $2, false, 'totp', $3)
		 ON CONFLICT (user_id) DO UPDATE SET
		     secret = EXCLUDED.secret,
		     enabled = false,
		     method = 'totp',
		     backup_codes = EXCLUDED.backup_codes,
		     updated_at = NOW()`,
		userID, secret, hashedBackupCodes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save MFA config: %w", err)
	}

	issuer := "Nexora CMS"
	accountName := user.Email
	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		urlEncode(issuer), urlEncode(accountName), secret, urlEncode(issuer))

	return &MFAEnrollResponse{
		Secret:      secret,
		QRCodeURL:   qrURL,
		BackupCodes: backupCodes,
	}, nil
}

func (s *MFAService) VerifyAndEnable(ctx context.Context, userID uuid.UUID, code string, svc *Service) error {
	var secret string
	err := svc.db.Pool.QueryRow(ctx,
		`SELECT secret FROM mfa_configs WHERE user_id = $1 AND enabled = false`,
		userID,
	).Scan(&secret)
	if err != nil {
		return fmt.Errorf("no pending MFA enrollment")
	}

	if !s.ValidateCode(secret, code) {
		return fmt.Errorf("invalid code")
	}

	_, err = svc.db.Pool.Exec(ctx,
		`UPDATE mfa_configs SET enabled = true, updated_at = NOW() WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	_, err = svc.db.Pool.Exec(ctx,
		`UPDATE users SET mfa_enabled = true, updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	svc.fireEvent(ctx, kernel.EventMFAEnabled, map[string]interface{}{
		"user_id": userID.String(),
	})

	svc.auditLog.LogUserAction(ctx, userID, audit.ActionMFAVerified, nil)

	return nil
}

func (s *MFAService) Disable(ctx context.Context, userID uuid.UUID, password string, svc *Service) error {
	_, err := svc.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	_ = password

	_, err = svc.db.Pool.Exec(ctx,
		`DELETE FROM mfa_configs WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	_, err = svc.db.Pool.Exec(ctx,
		`UPDATE users SET mfa_enabled = false, mfa_secret = NULL, updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	svc.fireEvent(ctx, kernel.EventMFADisabled, map[string]interface{}{
		"user_id": userID.String(),
	})

	svc.auditLog.LogUserAction(ctx, userID, audit.ActionMFADisabled, nil)

	return nil
}

func (s *MFAService) generateTOTPForCounter(secret string, counter int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	code := truncated % 1000000

	return fmt.Sprintf("%06d", code), nil
}

func hashBackupCode(code string) string {
	return fmt.Sprintf("%x", sha1Bytes([]byte(code)))
}

func sha1Bytes(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

func urlEncode(s string) string {
	result := make([]byte, 0, len(s)*3)
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			result = append(result, c)
		} else {
			result = append(result, fmt.Sprintf("%%%02X", c)...)
		}
	}
	return string(result)
}
