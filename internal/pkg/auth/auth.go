package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

type PasswordConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultPasswordConfig = PasswordConfig{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

func HashPassword(password string, cfg *PasswordConfig) (string, error) {
	if cfg == nil {
		cfg = &DefaultPasswordConfig
	}

	salt := make([]byte, cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		cfg.Iterations,
		cfg.Memory,
		cfg.Parallelism,
		cfg.KeyLength,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		cfg.Memory,
		cfg.Iterations,
		cfg.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) < 6 {
		return false, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}

	// parts[0] = "", parts[1] = "argon2id", parts[2] = "v=19", parts[3] = "m=N,t=N,p=N"
	// parts[4] = salt (base64), parts[5] = hash (base64)
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("invalid hash format: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid salt encoding: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid hash encoding: %w", err)
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	return hmac.Equal(actualHash, expectedHash), nil
}

type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (tm *TokenManager) GenerateAccessToken(userID string) (string, error) {
	return tm.generateToken(userID, tm.accessTTL, "access")
}

func (tm *TokenManager) GenerateRefreshToken(userID string) (string, error) {
	return tm.generateToken(userID, tm.refreshTTL, "refresh")
}

func (tm *TokenManager) generateToken(userID string, ttl time.Duration, purpose string) (string, error) {
	expires := time.Now().Add(ttl).Unix()

	payload := base64.RawStdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s:%d", userID, purpose, expires),
	))

	mac := hmac.New(sha256.New, tm.secret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, sig), nil
}

func (tm *TokenManager) ValidateToken(token, expectedPurpose string) (string, error) {
	parts := splitToken(token)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}

	payload, sig := parts[0], parts[1]

	decoded, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("invalid token encoding: %w", err)
	}

	mac := hmac.New(sha256.New, tm.secret)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid token signature")
	}

	parts = splitString(string(decoded), ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid token payload")
	}

	purpose := parts[1]
	if purpose != expectedPurpose {
		return "", fmt.Errorf("token purpose mismatch: expected %q, got %q", expectedPurpose, purpose)
	}

	expires := parseInt64(parts[2])
	if time.Now().Unix() > expires {
		return "", fmt.Errorf("token expired")
	}

	return parts[0], nil
}

func splitToken(token string) []string {
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return nil
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
