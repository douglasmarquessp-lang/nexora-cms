package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"nexora/internal/kernel"
	"nexora/internal/pkg/auth"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

type Service struct {
	cfg      *config.Config
	log      *logger.Logger
	repo     *Repository
	eventBus *kernel.EventBus
}

func NewService(cfg *config.Config, log *logger.Logger, repo *Repository) *Service {
	return &Service{
		cfg:  cfg,
		log:  log,
		repo: repo,
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, "")
	}
}

func (s *Service) Status(ctx context.Context) (*StatusResponse, error) {
	inst, err := s.repo.GetInstallation(ctx)
	if err != nil {
		return nil, err
	}

	if inst == nil {
		return &StatusResponse{Installed: false}, nil
	}

	return &StatusResponse{
		Installed:   inst.Installed,
		InstalledAt: inst.InstalledAt,
		CmsName:     inst.CmsName,
		Version:     inst.Version,
	}, nil
}

func (s *Service) Install(ctx context.Context, req InstallRequest) (*SystemInstallation, error) {
	if err := validateInstallRequest(req); err != nil {
		return nil, err
	}

	inst, err := s.repo.GetInstallation(ctx)
	if err != nil {
		return nil, err
	}
	if inst != nil && inst.Installed {
		return nil, ErrAlreadyInstalled
	}

	req.Locale = coalesceStr(req.Locale, "pt-BR")
	req.Timezone = coalesceStr(req.Timezone, "America/Sao_Paulo")
	req.CmsName = coalesceStr(req.CmsName, "Nexora CMS")

	s.fireEvent(ctx, EventSetupStarted, map[string]interface{}{
		"cms_name":  req.CmsName,
		"admin_email": req.AdminEmail,
	})

	passwordHash, err := auth.HashPassword(req.Password, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	adminID := uuid.New()
	if err := s.repo.CreateAdminUser(ctx, adminID, req.AdminName, req.AdminEmail, passwordHash); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("email already registered: %s", req.AdminEmail)
		}
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	siteID := uuid.New()
	slug := slugify(req.SiteName)
	if err := s.repo.CreateSite(ctx, siteID, adminID, req.SiteName, slug, req.SiteDescription, req.Locale, req.Timezone, req.SiteURL); err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	if err := s.repo.CreateSiteUser(ctx, adminID, siteID, "admin"); err != nil {
		return nil, fmt.Errorf("failed to assign admin to site: %w", err)
	}

	installation := &SystemInstallation{
		ID:          uuid.New(),
		CmsName:     req.CmsName,
		AdminName:   req.AdminName,
		AdminEmail:  req.AdminEmail,
		DefaultSite: req.SiteName,
		Version:     "0.1.0",
		Locale:      req.Locale,
		Timezone:    req.Timezone,
	}

	if err := s.repo.CreateInstallation(ctx, installation); err != nil {
		return nil, fmt.Errorf("failed to save installation: %w", err)
	}

	s.log.Info("system installation completed",
		"admin_email", req.AdminEmail,
		"site_name", req.SiteName,
	)

	s.fireEvent(ctx, EventSetupFinished, map[string]interface{}{
		"installation_id": installation.ID.String(),
		"admin_email":     req.AdminEmail,
		"site_name":       req.SiteName,
	})

	return installation, nil
}

func (s *Service) Finish(ctx context.Context) (map[string]string, error) {
	inst, err := s.repo.GetInstallation(ctx)
	if err != nil {
		return nil, err
	}
	if inst == nil || !inst.Installed {
		return nil, ErrNotInstalled
	}

	s.log.Info("system installation verified, redirecting to login",
		"admin_email", inst.AdminEmail,
		"cms_name", inst.CmsName,
	)

	return map[string]string{
		"status":  "installed",
		"message": "System installed. Please log in.",
	}, nil
}

func (s *Service) GetConfig(ctx context.Context) (*ConfigResponse, error) {
	return &ConfigResponse{
		Locales: []string{
			"pt-BR", "pt-PT", "en-US", "en-GB", "es-ES",
			"fr-FR", "de-DE", "it-IT", "ja-JP", "zh-CN",
		},
		Timezones: []string{
			"America/Sao_Paulo", "America/New_York", "America/Chicago",
			"America/Denver", "America/Los_Angeles", "Europe/London",
			"Europe/Paris", "Europe/Berlin", "Europe/Lisbon",
			"Asia/Tokyo", "Asia/Shanghai", "Asia/Kolkata",
			"Australia/Sydney", "Pacific/Auckland", "UTC",
		},
		Themes: []string{
			"default", "dark", "light", "modern", "minimal",
		},
		AIProviders: []string{
			"openai", "anthropic", "gemini", "openrouter", "ollama", "none",
		},
	}, nil
}

func coalesceStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' {
			return '-'
		}
		return -1
	}, slug)
	slug = strings.Trim(slug, "-_")
	if slug == "" {
		return "default-site"
	}
	return slug
}
