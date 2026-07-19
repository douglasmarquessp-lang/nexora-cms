package setup

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func setupMockDB(t *testing.T) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-long-enough-for-hmac-sha256",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 604800000000000,
		},
	}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	repo := NewRepository(&database.Database{Pool: mock})
	svc := NewService(cfg, log, repo)
	return svc, mock
}

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestStatus_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)

	_, err := svc.Status(context.Background())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestStatus_NotInstalled(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM system_installation`).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "installed", "installed_at", "cms_name", "admin_name",
			"admin_email", "default_site", "version", "locale", "timezone",
			"created_at", "updated_at",
		}))

	resp, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Installed {
		t.Error("expected installed=false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestStatus_Installed(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM system_installation`).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "installed", "installed_at", "cms_name", "admin_name",
			"admin_email", "default_site", "version", "locale", "timezone",
			"created_at", "updated_at",
        }).AddRow(
			"00000000-0000-0000-0000-000000000001", true, nil, "Nexora", "Admin",
			"admin@test.com", "Site", "0.1.0", "pt-BR", "UTC",
			time.Now(), time.Now(),
		))

	resp, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Installed {
		t.Error("expected installed=true")
	}
	if resp.CmsName != "Nexora" {
		t.Errorf("expected CmsName 'Nexora', got %q", resp.CmsName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestInstall_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)

	_, err := svc.Install(context.Background(), InstallRequest{
		CmsName:   "Nexora",
		AdminName: "Admin",
	})
	if err == nil {
		t.Error("expected error for no database")
	}
}

func TestInstall_ValidationError(t *testing.T) {
	svc, mock := setupMockDB(t)

	_, err := svc.Install(context.Background(), InstallRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var valErr *ValidationError
	if !as(err, &valErr) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM system_installation`).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "installed", "installed_at", "cms_name", "admin_name",
			"admin_email", "default_site", "version", "locale", "timezone",
			"created_at", "updated_at",
		}).AddRow(
			"00000000-0000-0000-0000-000000000001", true, nil, "Nexora", "Admin",
			"admin@test.com", "Site", "0.1.0", "pt-BR", "UTC",
			time.Now(), time.Now(),
		))

	_, err := svc.Install(context.Background(), InstallRequest{
		CmsName:    "Nexora",
		AdminName:  "Admin",
		AdminEmail: "admin@test.com",
		Password:   "Str0ng!Pass",
		SiteName:   "Site",
		Language:   "pt-BR",
		Timezone:   "UTC",
	})
	if err != ErrAlreadyInstalled {
		t.Errorf("expected ErrAlreadyInstalled, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestFinish_NoDB(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)

	_, err := svc.Finish(context.Background())
	if err == nil {
		t.Error("expected error for no database")
	}
}

func TestFinish_NotInstalled(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM system_installation`).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "installed", "installed_at", "cms_name", "admin_name",
			"admin_email", "default_site", "version", "locale", "timezone",
			"created_at", "updated_at",
		}))

	_, err := svc.Finish(context.Background())
	if err != ErrNotInstalled {
		t.Errorf("expected ErrNotInstalled, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestGetConfig(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)

	resp, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Locales) == 0 {
		t.Error("expected non-empty locales")
	}
	if len(resp.Timezones) == 0 {
		t.Error("expected non-empty timezones")
	}
	if len(resp.Themes) == 0 {
		t.Error("expected non-empty themes")
	}
	if len(resp.AIProviders) == 0 {
		t.Error("expected non-empty AI providers")
	}
}

func TestCoalesceStr(t *testing.T) {
	if coalesceStr("hello", "fallback") != "hello" {
		t.Error("expected first value")
	}
	if coalesceStr("", "fallback") != "fallback" {
		t.Error("expected fallback value")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My First Site", "my-first-site"},
		{"Hello World!", "hello-world"},
		{"  Spaces  ", "spaces"},
		{"", "default-site"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "specialchars"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// as is errors.As helper for testing
func as(err error, target interface{}) bool {
	return errorsAs(err, target)
}

func errorsAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	// Simple type assertion for testing
	switch e := err.(type) {
	case *ValidationError:
		if t, ok := target.(**ValidationError); ok {
			*t = e
			return true
		}
	}
	return false
}
