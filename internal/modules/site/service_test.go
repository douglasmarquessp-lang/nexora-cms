package site

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestService_NewService_WithCache(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	svc := NewService(cfg, log, nil, ch)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestService_CreateSite_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSite(context.Background(), uuid.New(), CreateSiteRequest{
		Slug: "test-site",
		Name: "Test Site",
	})
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_CreateSite_Validation_SlugRequired(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSite(context.Background(), uuid.New(), CreateSiteRequest{
		Slug: "",
		Name: "Test Site",
	})
	if err == nil {
		t.Fatal("expected error for empty slug")
	}
}

func TestService_CreateSite_Validation_NameRequired(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSite(context.Background(), uuid.New(), CreateSiteRequest{
		Slug: "test-site",
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestService_GetSite_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetSite(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_GetSiteBySlug_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetSiteBySlug(context.Background(), "test-slug")
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_GetSiteByDomain_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetSiteByDomain(context.Background(), "example.com")
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_ListSites_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 1, 20)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_UpdateSite_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.UpdateSite(context.Background(), uuid.New(), UpdateSiteRequest{})
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_DeleteSite_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.DeleteSite(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_AddDomain_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.AddDomain(context.Background(), uuid.New(), AddDomainRequest{
		Domain: "example.com",
	})
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_AddDomain_InvalidDomain(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.AddDomain(context.Background(), uuid.New(), AddDomainRequest{
		Domain: "",
	})
	if err != ErrInvalidDomain {
		t.Errorf("expected ErrInvalidDomain, got: %v", err)
	}
}

func TestService_RemoveDomain_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.RemoveDomain(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_ListDomains_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListDomains(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetPrimaryDomain_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.SetPrimaryDomain(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_GetGlobalSetting_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetGlobalSetting(context.Background(), "test-key")
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetGlobalSetting_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.SetGlobalSetting(context.Background(), UpdateGlobalSettingRequest{
		Key:   "test-key",
		Value: "test-value",
	})
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetGlobalSetting_InvalidType(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.SetGlobalSetting(context.Background(), UpdateGlobalSettingRequest{
		Key:   "test-key",
		Value: "test-value",
		Type:  "invalid-type",
	})
	if err != ErrInvalidSettingType {
		t.Errorf("expected ErrInvalidSettingType, got: %v", err)
	}
}

func TestService_ListGlobalSettings_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListGlobalSettings(context.Background())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_GetSiteSetting_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetSiteSetting(context.Background(), uuid.New(), "test-key")
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetSiteSetting_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.SetSiteSetting(context.Background(), uuid.New(), SetSiteSettingRequest{
		Key:   "test-key",
		Value: "test-value",
	})
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetSiteSetting_KeyRequired(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.SetSiteSetting(context.Background(), uuid.New(), SetSiteSettingRequest{
		Key:   "",
		Value: "test-value",
	})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestService_ListSiteSettings_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSiteSettings(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetEventBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	bus := kernel.NewEventBus(log)

	svc.SetEventBus(bus)
}

func TestService_SetEventBus_Nil(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.SetEventBus(nil)
}

func TestService_FireEvent_WithBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	svc.fireEvent(context.Background(), EventSiteCreated, map[string]interface{}{
		"site_id": uuid.New().String(),
	})
}

func TestService_FireEvent_WithoutBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.fireEvent(context.Background(), EventSiteCreated, map[string]interface{}{
		"site_id": uuid.New().String(),
	})
}

func TestService_SetRLSContext(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	ctx := svc.SetRLSContext(context.Background(), uuid.New(), "admin")
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestService_ListSites_DefaultPagination(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 0, 0)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_ListSites_PaginationOverLimit(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 1, 200)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestSiteStatusConstants(t *testing.T) {
	if SiteStatusActive != "active" {
		t.Errorf("expected 'active', got '%s'", SiteStatusActive)
	}
	if SiteStatusInactive != "inactive" {
		t.Errorf("expected 'inactive', got '%s'", SiteStatusInactive)
	}
	if SiteStatusSuspended != "suspended" {
		t.Errorf("expected 'suspended', got '%s'", SiteStatusSuspended)
	}
	if SiteStatusMaintenance != "maintenance" {
		t.Errorf("expected 'maintenance', got '%s'", SiteStatusMaintenance)
	}
}

func TestErrorConstants(t *testing.T) {
	if ErrSiteNotFound.Error() != "site not found" {
		t.Errorf("unexpected message: %s", ErrSiteNotFound.Error())
	}
	if ErrSiteSlugAlreadyExists.Error() != "site slug already exists" {
		t.Errorf("unexpected message: %s", ErrSiteSlugAlreadyExists.Error())
	}
	if ErrDomainAlreadyExists.Error() != "domain already exists" {
		t.Errorf("unexpected message: %s", ErrDomainAlreadyExists.Error())
	}
	if ErrDomainNotFound.Error() != "domain not found" {
		t.Errorf("unexpected message: %s", ErrDomainNotFound.Error())
	}
	if ErrInvalidDomain.Error() != "invalid domain format" {
		t.Errorf("unexpected message: %s", ErrInvalidDomain.Error())
	}
	if ErrSiteNotAvailable.Error() != "site not available" {
		t.Errorf("unexpected message: %s", ErrSiteNotAvailable.Error())
	}
	if ErrDatabaseNotAvailable.Error() != "database not available" {
		t.Errorf("unexpected message: %s", ErrDatabaseNotAvailable.Error())
	}
	if ErrGlobalSettingNotFound.Error() != "global setting not found" {
		t.Errorf("unexpected message: %s", ErrGlobalSettingNotFound.Error())
	}
	if ErrSiteSettingNotFound.Error() != "site setting not found" {
		t.Errorf("unexpected message: %s", ErrSiteSettingNotFound.Error())
	}
	if ErrInvalidSettingType.Error() != "invalid setting type" {
		t.Errorf("unexpected message: %s", ErrInvalidSettingType.Error())
	}
}

func TestEventConstants(t *testing.T) {
	_ = []string{
		string(EventSiteCreated),
		string(EventSiteUpdated),
		string(EventSiteDeleted),
		string(EventDomainAdded),
		string(EventDomainRemoved),
	}
}

func TestDomainRegex_Valid(t *testing.T) {
	validDomains := []string{
		"example.com",
		"sub.example.com",
		"my-site.io",
		"example123.org",
		"a.b.co",
		"my.co.uk",
		"xn--n1a.xyz",
		"test.example.com.br",
		"localhost.localdomain",
		"abc.def.ghi.jk",
	}

	for _, domain := range validDomains {
		t.Run(domain, func(t *testing.T) {
			if !domainRegex.MatchString(domain) {
				t.Errorf("expected domain %q to be valid", domain)
			}
		})
	}
}

func TestDomainRegex_Invalid(t *testing.T) {
	invalidDomains := []string{
		"",
		"-example.com",
		"example..com",
		"example",
		".com",
		"example.c",
		"example.c-m",
		"-site.org",
		"site-.org",
		"a.-b.com",
		"a.b-.com",
		"a..b.com",
		"example .com",
		"*.example.com",
		"exa mple.com",
		".example.com",
	}

	for _, domain := range invalidDomains {
		t.Run(domain, func(t *testing.T) {
			if domainRegex.MatchString(domain) {
				t.Errorf("expected domain %q to be invalid", domain)
			}
		})
	}
}

func TestValidSettingTypes(t *testing.T) {
	expected := map[string]bool{
		"string": true, "number": true, "boolean": true, "json": true, "array": true,
	}
	for k, v := range validSettingTypes {
		if expected[k] != v {
			t.Errorf("unexpected validSettingTypes[%q] = %v", k, v)
		}
	}
	if len(validSettingTypes) != len(expected) {
		t.Errorf("expected %d setting types, got %d", len(expected), len(validSettingTypes))
	}
}

func TestService_inferType(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	tests := []struct {
		val      interface{}
		expected string
	}{
		{nil, "json"},
		{true, "boolean"},
		{false, "boolean"},
		{float64(42), "number"},
		{float64(3.14), "number"},
		{"hello", "string"},
		{[]interface{}{1, 2, 3}, "array"},
		{map[string]interface{}{"a": 1}, "json"},
		{42, "json"},
		{[]string{"a"}, "json"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := svc.inferType(tt.val)
			if result != tt.expected {
				t.Errorf("inferType(%v) = %q, want %q", tt.val, result, tt.expected)
			}
		})
	}
}

func TestService_pool_NilDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	_, err := svc.pool()
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_pool_NilDBPool(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	db := &database.Database{Pool: nil}
	svc := NewService(cfg, log, db, nil)
	_, err := svc.pool()
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_cacheGet_NoCache(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	ctx := context.Background()
	var dest interface{}
	ok := svc.cacheGet(ctx, "somekey", &dest)
	if ok {
		t.Error("expected false when cache is nil")
	}
}

func TestService_cacheGet_WithCache_Hit(t *testing.T) {
	ch := cache.New(false)
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, ch)

	ctx := context.Background()
	err := ch.SetJSON(ctx, "testkey", map[string]interface{}{"name": "test"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("SetJSON failed: %v", err)
	}

	var dest map[string]interface{}
	ok := svc.cacheGet(ctx, "testkey", &dest)
	if !ok {
		t.Fatal("expected true for cached key")
	}
	if dest["name"] != "test" {
		t.Errorf("expected 'test', got %v", dest["name"])
	}
}

func TestService_cacheGet_WithCache_Miss(t *testing.T) {
	ch := cache.New(false)
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, ch)

	ctx := context.Background()
	var dest interface{}
	ok := svc.cacheGet(ctx, "nonexistent", &dest)
	if ok {
		t.Error("expected false for missing key")
	}
}

func TestService_cacheSet_NoCache(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.cacheSet(context.Background(), "key", "value")
}

func TestService_cacheSet_WithCache(t *testing.T) {
	ch := cache.New(false)
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, ch)

	ctx := context.Background()
	svc.cacheSet(ctx, "settest", "setvalue")

	var dest string
	ok := svc.cacheGet(ctx, "settest", &dest)
	if !ok {
		t.Fatal("expected true after cacheSet")
	}
	if dest != "setvalue" {
		t.Errorf("expected 'setvalue', got '%s'", dest)
	}
}

func TestService_cacheDel_NoCache(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.cacheDel(context.Background(), "key")
}

func TestService_cacheDel_WithCache(t *testing.T) {
	ch := cache.New(false)
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, ch)

	ctx := context.Background()
	svc.cacheSet(ctx, "deltest", "delvalue")
	svc.cacheDel(ctx, "deltest")

	var dest string
	ok := svc.cacheGet(ctx, "deltest", &dest)
	if ok {
		t.Error("expected false after cacheDel")
	}
}

func TestService_cacheDel_MultipleKeys(t *testing.T) {
	ch := cache.New(false)
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, ch)

	ctx := context.Background()
	svc.cacheSet(ctx, "key1", "val1")
	svc.cacheSet(ctx, "key2", "val2")
	svc.cacheDel(ctx, "key1", "key2")

	var dest string
	if svc.cacheGet(ctx, "key1", &dest) {
		t.Error("expected key1 to be deleted")
	}
	if svc.cacheGet(ctx, "key2", &dest) {
		t.Error("expected key2 to be deleted")
	}
}

func TestService_SetRLSContext_Values(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	userID := uuid.New()
	ctx := svc.SetRLSContext(context.Background(), userID, "admin")

	uidStr, ok := ctx.Value("app.current_user_id").(string)
	if !ok {
		t.Fatal("expected app.current_user_id in context")
	}
	if uidStr != userID.String() {
		t.Errorf("expected %s, got %s", userID.String(), uidStr)
	}

	role, ok := ctx.Value("app.current_user_role").(string)
	if !ok {
		t.Fatal("expected app.current_user_role in context")
	}
	if role != "admin" {
		t.Errorf("expected 'admin', got '%s'", role)
	}
}

func TestService_SetRLSContext_DifferentRoles(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	roles := []string{"superadmin", "siteadmin", "editor", "author", "subscriber"}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			ctx := svc.SetRLSContext(context.Background(), uuid.New(), role)
			r, ok := ctx.Value("app.current_user_role").(string)
			if !ok || r != role {
				t.Errorf("expected role %q, got %q", role, r)
			}
		})
	}
}

func TestService_FireEvent_WithSubscriber(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	received := make(chan string, 1)
	bus.Subscribe(EventSiteCreated, func(ctx context.Context, event kernel.Event) error {
		if p, ok := event.Payload.(map[string]interface{}); ok {
			if slug, ok := p["slug"]; ok {
				received <- slug.(string)
			}
		}
		return nil
	})

	svc.fireEvent(context.Background(), EventSiteCreated, map[string]interface{}{
		"site_id": uuid.New().String(),
		"slug":    "my-test-site",
	})

	select {
	case slug := <-received:
		if slug != "my-test-site" {
			t.Errorf("expected 'my-test-site', got '%s'", slug)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestService_NewService_WithConfig(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	db := &database.Database{Pool: nil}
	ch := cache.New(false)
	svc := NewService(cfg, log, db, ch)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.log != log {
		t.Error("logger mismatch")
	}
	if svc.db != db {
		t.Error("db mismatch")
	}
	if svc.cache != ch {
		t.Error("cache mismatch")
	}
}

func TestService_ListSites_PageNormalization(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 0, 0)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_ListSites_PerPageClampHigh(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 1, 200)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_ListSites_PerPageClampLow(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListSites(context.Background(), uuid.New(), 1, -1)
	if err != ErrDatabaseNotAvailable {
		t.Errorf("expected ErrDatabaseNotAvailable, got: %v", err)
	}
}

func TestService_SetSiteSetting_KeyRequired_Message(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.SetSiteSetting(context.Background(), uuid.New(), SetSiteSettingRequest{
		Key:   "",
		Value: "test",
	})
	if err == nil || err.Error() != "key is required" {
		t.Errorf("expected 'key is required', got: %v", err)
	}
}

func TestService_CreateSite_Validation_SlugRequired_Message(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSite(context.Background(), uuid.New(), CreateSiteRequest{
		Slug: "",
		Name: "Test",
	})
	if err == nil || err.Error() != "slug is required" {
		t.Errorf("expected 'slug is required', got: %v", err)
	}
}

func TestService_CreateSite_Validation_NameRequired_Message(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSite(context.Background(), uuid.New(), CreateSiteRequest{
		Slug: "test-site",
		Name: "",
	})
	if err == nil || err.Error() != "name is required" {
		t.Errorf("expected 'name is required', got: %v", err)
	}
}

func TestService_AddDomain_InvalidDomain_EdgeCases(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	invalidDomains := []string{
		"",
		"not-a-domain",
		"-leading.com",
		"trailing-.com",
		"spaces in.com",
		"a",
		"a..b.com",
	}

	for _, d := range invalidDomains {
		t.Run(d, func(t *testing.T) {
			_, err := svc.AddDomain(context.Background(), uuid.New(), AddDomainRequest{
				Domain: d,
			})
			if err != ErrInvalidDomain {
				t.Errorf("expected ErrInvalidDomain for %q, got: %v", d, err)
			}
		})
	}
}

func TestService_EventConstants_Values(t *testing.T) {
	events := map[kernel.EventType]string{
		EventSiteCreated:   "site.created",
		EventSiteUpdated:   "site.updated",
		EventSiteDeleted:   "site.deleted",
		EventDomainAdded:   "site.domain.added",
		EventDomainRemoved: "site.domain.removed",
	}
	for evt, expected := range events {
		if string(evt) != expected {
			t.Errorf("expected %q, got %q", expected, string(evt))
		}
	}
}

func TestService_GetSite_WithMockDB(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &database.Database{Pool: mock}
	ch := cache.New(false)
	svc := NewService(cfg, log, db, ch)

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "name", "slug", "coalesce", "status", "owner_id",
		"settings", "feature_flags", "theme", "locale", "timezone",
		"created_at", "updated_at", "deleted_at"}).
		AddRow(siteID, "Test Site", "test-site", "desc", "active", uuid.New(),
			[]byte(`{}`), []byte(`{}`), "default", "en-US", "UTC",
			now, now, nil)

	mock.ExpectQuery(`SELECT id, name, slug, COALESCE\(description, ''\), status, owner_id,`).
		WithArgs(siteID).
		WillReturnRows(rows)

	site, err := svc.GetSite(ctx, siteID)
	if err != nil {
		t.Fatalf("GetSite failed: %v", err)
	}
	if site.ID != siteID {
		t.Errorf("expected site ID %s, got %s", siteID, site.ID)
	}
	if site.Name != "Test Site" {
		t.Errorf("expected 'Test Site', got '%s'", site.Name)
	}
	if site.Slug != "test-site" {
		t.Errorf("expected 'test-site', got '%s'", site.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetSite_WithCacheHit(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	ch := cache.New(false)
	svc := NewService(cfg, log, nil, ch)

	siteID := uuid.New()
	expected := &Site{
		ID:   siteID,
		Name: "Cached Site",
		Slug: "cached-site",
	}

	ch.SetJSON(ctx, "site:"+siteID.String(), expected, 5*time.Minute)

	site, err := svc.GetSite(ctx, siteID)
	if err != nil {
		t.Fatalf("GetSite failed: %v", err)
	}
	if site.Name != "Cached Site" {
		t.Errorf("expected 'Cached Site', got '%s'", site.Name)
	}
}

func TestService_GetSiteBySlug_WithCacheHit(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	ch := cache.New(false)
	svc := NewService(cfg, log, nil, ch)

	expected := &Site{
		ID:   uuid.New(),
		Name: "Slug Site",
		Slug: "my-slug",
	}

	ch.SetJSON(ctx, "site:slug:my-slug", expected, 5*time.Minute)

	site, err := svc.GetSiteBySlug(ctx, "my-slug")
	if err != nil {
		t.Fatalf("GetSiteBySlug failed: %v", err)
	}
	if site.Name != "Slug Site" {
		t.Errorf("expected 'Slug Site', got '%s'", site.Name)
	}
}

func TestService_GetSiteByDomain_WithCacheHit(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	ch := cache.New(false)
	svc := NewService(cfg, log, nil, ch)

	expected := &Site{
		ID:   uuid.New(),
		Name: "Domain Site",
		Slug: "domain-site",
	}

	ch.SetJSON(ctx, "site:domain:example.com", expected, 5*time.Minute)

	site, err := svc.GetSiteByDomain(ctx, "example.com")
	if err != nil {
		t.Fatalf("GetSiteByDomain failed: %v", err)
	}
	if site.Name != "Domain Site" {
		t.Errorf("expected 'Domain Site', got '%s'", site.Name)
	}
}

func TestService_UpdateSite_NoChanges(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &database.Database{Pool: mock}
	svc := NewService(cfg, log, db, nil)

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "status", "owner_id",
		"settings", "feature_flags", "theme", "locale", "timezone",
		"created_at", "updated_at", "deleted_at"}).
		AddRow(siteID, "Original", "original", nil, "active", uuid.New(),
			[]byte(`{}`), []byte(`{}`), nil, nil, nil,
			now, now, nil)

	mock.ExpectQuery(`SELECT id, name, slug, COALESCE\(description, ''\), status, owner_id,`).
		WithArgs(siteID).
		WillReturnRows(rows)

	site, err := svc.UpdateSite(ctx, siteID, UpdateSiteRequest{})
	if err != nil {
		t.Fatalf("UpdateSite failed: %v", err)
	}
	if site.Name != "Original" {
		t.Errorf("expected 'Original', got '%s'", site.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_ListSites_EmptyResult(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &database.Database{Pool: mock}
	svc := NewService(cfg, log, db, nil)

	rows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "status", "owner_id",
		"settings", "feature_flags", "theme", "locale", "timezone",
		"created_at", "updated_at"})

	mock.ExpectQuery(`SELECT id, name, slug, COALESCE\(description, ''\), status, owner_id,`).
		WithArgs(20, 0).
		WillReturnRows(rows)

	resp, err := svc.ListSites(ctx, uuid.New(), 1, 20)
	if err != nil {
		t.Fatalf("ListSites failed: %v", err)
	}
	if len(resp.Sites) != 0 {
		t.Errorf("expected 0 sites, got %d", len(resp.Sites))
	}
	if resp.Total != 0 {
		t.Errorf("expected total 0, got %d", resp.Total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetGlobalSetting_WithMockDB(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &database.Database{Pool: mock}
	svc := NewService(cfg, log, db, nil)

	settingID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "key", "value", "type", "description", "created_at", "updated_at"}).
		AddRow(settingID, "site.name", []byte(`"My Site"`), "string", nil, now, now)

	mock.ExpectQuery(`SELECT id, key, value::text, type, COALESCE\(description, ''\), created_at, updated_at`).
		WithArgs("site.name").
		WillReturnRows(rows)

	gs, err := svc.GetGlobalSetting(ctx, "site.name")
	if err != nil {
		t.Fatalf("GetGlobalSetting failed: %v", err)
	}
	if gs.Key != "site.name" {
		t.Errorf("expected 'site.name', got '%s'", gs.Key)
	}
	if gs.Type != "string" {
		t.Errorf("expected 'string', got '%s'", gs.Type)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetSiteSetting_WithMockDB(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := &database.Database{Pool: mock}
	svc := NewService(cfg, log, db, nil)

	siteID := uuid.New()
	settingID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "key", "value", "created_at", "updated_at"}).
		AddRow(settingID, siteID, "theme", []byte(`"dark"`), now, now)

	mock.ExpectQuery(`SELECT id, site_id, key, value::text, created_at, updated_at`).
		WithArgs(siteID, "theme").
		WillReturnRows(rows)

	ss, err := svc.GetSiteSetting(ctx, siteID, "theme")
	if err != nil {
		t.Fatalf("GetSiteSetting failed: %v", err)
	}
	if ss.Key != "theme" {
		t.Errorf("expected 'theme', got '%s'", ss.Key)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_SiteStatusConstants(t *testing.T) {
	tests := []struct {
		status   SiteStatus
		expected string
	}{
		{SiteStatusActive, "active"},
		{SiteStatusInactive, "inactive"},
		{SiteStatusSuspended, "suspended"},
		{SiteStatusMaintenance, "maintenance"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}
