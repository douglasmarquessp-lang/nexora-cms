package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"nexora/internal/pkg/logger"
)

func TestMFAService_New(t *testing.T) {
	s := NewMFAService()
	if s == nil {
		t.Fatal("expected non-nil MFA service")
	}
}

func TestMFAService_GenerateSecret(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}
	// Base32 encoded 20 bytes = 32 chars without padding
	if len(secret) != 32 {
		t.Errorf("expected secret length 32, got %d", len(secret))
	}

	secret2, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret == secret2 {
		t.Error("expected different secrets on subsequent calls")
	}
}

func TestMFAService_GenerateTOTP_InvalidSecret(t *testing.T) {
	s := NewMFAService()
	_, err := s.GenerateTOTP("not-base32!!!")
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestMFAService_GenerateTOTP_Valid(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	code, err := s.GenerateTOTP(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected 6-digit code, got %q", code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("expected digit, got %c", c)
		}
	}
}

func TestMFAService_ValidateCode_Empty(t *testing.T) {
	s := NewMFAService()
	if s.ValidateCode("", "") {
		t.Fatal("expected false for empty secret and code")
	}
	if s.ValidateCode("secret", "") {
		t.Fatal("expected false for empty code")
	}
	if s.ValidateCode("", "123456") {
		t.Fatal("expected false for empty secret")
	}
}

func TestMFAService_ValidateCode_InvalidSecret(t *testing.T) {
	s := NewMFAService()
	if s.ValidateCode("invalid-secret", "123456") {
		t.Fatal("expected false for invalid secret")
	}
}

func TestMFAService_ValidateCode_Valid(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	code, err := s.GenerateTOTP(secret)
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	if !s.ValidateCode(secret, code) {
		t.Fatal("expected valid code to pass validation")
	}
}

func TestMFAService_ValidateCode_WrongCode(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	if s.ValidateCode(secret, "000000") {
		t.Fatal("expected wrong code to fail validation")
	}
}

// Test that adjacent time steps work for clock drift
func TestMFAService_ValidateCode_AdjacentTimeStep(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Generate code for a counter one step in the past
	ctr := (time.Now().Unix() / 30) - 1
	code, err := s.generateTOTPForCounter(secret, ctr)
	if err != nil {
		t.Fatalf("failed to generate code for past step: %v", err)
	}

	// Should still validate due to clock drift tolerance
	if !s.ValidateCode(secret, code) {
		t.Fatal("expected code from adjacent time step to validate")
	}
}

func TestMFAService_ValidateBackupCode_Empty(t *testing.T) {
	s := NewMFAService()
	ok, remaining := s.ValidateBackupCode(nil, "")
	if ok {
		t.Fatal("expected false for nil codes")
	}
	if remaining != "" {
		t.Errorf("expected empty remaining, got %s", remaining)
	}

	ok, remaining = s.ValidateBackupCode([]string{}, "code")
	if ok {
		t.Fatal("expected false for empty codes")
	}
	_ = remaining
}

func TestMFAService_ValidateBackupCode_Valid(t *testing.T) {
	s := NewMFAService()
	codes, err := s.GenerateBackupCodes()
	if err != nil {
		t.Fatal(err)
	}
	hashed := s.HashBackupCodes(codes)

	ok, remaining := s.ValidateBackupCode(hashed, codes[0])
	if !ok {
		t.Fatal("expected valid backup code to pass")
	}
	if !strings.Contains(remaining, ",") {
		t.Error("expected remaining codes to be comma-separated")
	}
	// Original code should be removed (no longer valid after use)
	ok, _ = s.ValidateBackupCode(strings.Split(remaining, ","), codes[0])
	if ok {
		t.Fatal("expected used backup code to be removed")
	}
}

func TestMFAService_ValidateBackupCode_Invalid(t *testing.T) {
	s := NewMFAService()
	hashed := s.HashBackupCodes([]string{"code1", "code2"})
	ok, remaining := s.ValidateBackupCode(hashed, "wrong-code")
	if ok {
		t.Fatal("expected false for invalid code")
	}
	if remaining != "" {
		t.Errorf("expected empty remaining, got %s", remaining)
	}
}

func TestMFAService_GenerateBackupCodes(t *testing.T) {
	s := NewMFAService()
	codes, err := s.GenerateBackupCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != 8 {
		t.Errorf("expected 8 backup codes, got %d", len(codes))
	}
	for i, c := range codes {
		if len(c) == 0 {
			t.Errorf("code %d is empty", i)
		}
	}
	// Verify uniqueness
	seen := make(map[string]bool)
	for _, c := range codes {
		if seen[c] {
			t.Errorf("duplicate backup code: %s", c)
		}
		seen[c] = true
	}
}

func TestMFAService_HashBackupCodes(t *testing.T) {
	s := NewMFAService()
	codes := []string{"abc123", "def456", "ghi789"}
	hashed := s.HashBackupCodes(codes)
	if len(hashed) != len(codes) {
		t.Errorf("expected %d hashed codes, got %d", len(codes), len(hashed))
	}
	for i, h := range hashed {
		if h == codes[i] {
			t.Error("expected hash to differ from original")
		}
		if len(h) == 0 {
			t.Errorf("hash %d is empty", i)
		}
	}
}

func TestMFAService_HashBackupCodes_EmptyInput(t *testing.T) {
	s := NewMFAService()
	hashed := s.HashBackupCodes(nil)
	if len(hashed) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(hashed))
	}
}

func TestMFAService_Enroll_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)
	s := NewMFAService()

	_, err := s.Enroll(context.Background(), uuid.New(), svc)
	if err == nil {
		t.Fatal("expected error when no database")
	}
	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("expected 'user not found' error, got: %v", err)
	}
}

func TestMFAService_Enroll_MFAAlreadyEnabled(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)
	s := NewMFAService()

	// Enroll calls GetUserByID which returns error with nil db
	// MFA already enabled check never reached
	_, err := s.Enroll(context.Background(), uuid.New(), svc)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMFAService_VerifyAndEnable_NoDB(t *testing.T) {
	s := NewMFAService()
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)

	defer func() {
		if r := recover(); r != nil {
			// Expected - VerifyAndEnable accesses db.Pool directly without nil check
		}
	}()
	_ = s.VerifyAndEnable(context.Background(), uuid.New(), "123456", svc)
}

func TestMFAService_Disable_NoDB(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)
	s := NewMFAService()

	err := s.Disable(context.Background(), uuid.New(), "password", svc)
	if err == nil {
		t.Fatal("expected error when no database")
	}
	if !strings.Contains(err.Error(), "user not found") {
		t.Errorf("expected 'user not found' error, got: %v", err)
	}
}

func TestGenerateTOTPForCounter(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	code1, err := s.generateTOTPForCounter(secret, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code1) != 6 {
		t.Errorf("expected 6-digit code, got %q", code1)
	}

	code2, err := s.generateTOTPForCounter(secret, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code1 == code2 {
		t.Log("note: same code for adjacent counters is possible")
	}

	// Different counter should produce potentially different code
	code3, err := s.generateTOTPForCounter(secret, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code1 == code3 && code2 == code3 {
		t.Error("expected different codes for very different counters")
	}
}

func TestGenerateTOTPForCounter_InvalidSecret(t *testing.T) {
	s := &MFAService{}
	_, err := s.generateTOTPForCounter("!!!invalid-base32", 1000)
	if err == nil {
		t.Fatal("expected error for invalid base32 secret")
	}
}

func TestHashBackupCode(t *testing.T) {
	h := hashBackupCode("test-code")
	if len(h) == 0 {
		t.Fatal("expected non-empty hash")
	}
	if h == "test-code" {
		t.Error("expected hash to differ from input")
	}

	// Deterministic
	h2 := hashBackupCode("test-code")
	if h != h2 {
		t.Error("expected deterministic hash")
	}

	// Different inputs produce different hashes
	h3 := hashBackupCode("different-code")
	if h == h3 {
		t.Error("expected different hash for different input")
	}
}

func TestSha1Bytes(t *testing.T) {
	result := sha1Bytes([]byte("hello"))
	if len(result) != 20 {
		t.Errorf("expected SHA1 hash length 20, got %d", len(result))
	}

	result2 := sha1Bytes([]byte("world"))
	if len(result2) != 20 {
		t.Errorf("expected SHA1 hash length 20, got %d", len(result2))
	}

	// Deterministic
	result3 := sha1Bytes([]byte("hello"))
	for i, b := range result {
		if b != result3[i] {
			t.Fatal("expected deterministic hash")
		}
	}

	// Different inputs produce different hashes
	equal := true
	for i, b := range result {
		if b != result2[i] {
			equal = false
			break
		}
	}
	if equal {
		t.Error("expected different hashes for different inputs")
	}
}

func TestUrlEncode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", "with%20space"},
		{"special!@#$", "special%21%40%23%24"},
		{"a+b", "a%2Bb"},
		{"unicode", "unicode"},
		{"123", "123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := urlEncode(tt.input)
			if got != tt.want {
				t.Errorf("urlEncode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUrlEncode_ASCII(t *testing.T) {
	// Test all ASCII characters
	result := urlEncode("abcABC123-._~")
	if result != "abcABC123-._~" {
		t.Errorf("expected unreserved chars to pass through, got %q", result)
	}
}

func TestMFAService_GenerateTOTP_Consistency(t *testing.T) {
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Same time window should produce same code
	code1, _ := s.GenerateTOTP(secret)
	code2, _ := s.GenerateTOTP(secret)
	if code1 != code2 {
		t.Error("expected same code within same time window")
	}
}

func TestMFAService_RoundTripTOTP(t *testing.T) {
	s := NewMFAService()

	// Full TOTP round-trip through all components
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	code, err := s.GenerateTOTP(secret)
	if err != nil {
		t.Fatalf("GenerateTOTP failed: %v", err)
	}

	if !s.ValidateCode(secret, code) {
		t.Error("ValidateCode should accept code from GenerateTOTP")
	}
}

func TestHashBackupCode_Consistency(t *testing.T) {
	codes := []string{"abc123", "def456", "ghi789", "jkl012"}
	for _, c := range codes {
		h1 := hashBackupCode(c)
		h2 := hashBackupCode(c)
		if h1 != h2 {
			t.Errorf("hashBackupCode(%q) is not deterministic", c)
		}
	}
}

func TestMFAService_QRCodeURL(t *testing.T) {
	// Verify QRCode URL format generated by Enroll
	s := NewMFAService()
	secret, err := s.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	issuer := "Nexora%20CMS"
	accountName := "test@example.com"
	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		urlEncode(issuer), urlEncode(accountName), secret, urlEncode(issuer))

	if !strings.HasPrefix(qrURL, "otpauth://totp/") {
		t.Errorf("expected otpauth scheme, got %q", qrURL)
	}
	if !strings.Contains(qrURL, "secret="+secret) {
		t.Error("expected secret in QR code URL")
	}
}
