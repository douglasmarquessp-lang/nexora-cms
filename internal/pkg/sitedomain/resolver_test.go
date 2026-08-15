package sitedomain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

func TestNormalizeLocale(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"pt-BR", "pt"},
		{"en-US", "en"},
		{"en", "en"},
		{"pt", "pt"},
		{"PT-BR", "pt"},
		{" En-US ", "en"},
		{"", ""},
		{"  ", ""},
		{"fr-FR", "fr"},
		{"de", "de"},
	}
	for _, c := range cases {
		if got := NormalizeLocale(c.in); got != c.want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolve_Full(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow("en-US"))
	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"domain"}).AddRow("https://aiworksimple.com"))

	sc, err := New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Domain != "https://aiworksimple.com" {
		t.Errorf("domain = %q, want %q", sc.Domain, "https://aiworksimple.com")
	}
	if sc.PrimaryLanguage != "en" {
		t.Errorf("primary language = %q, want %q", sc.PrimaryLanguage, "en")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestResolve_PTSite(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow("pt-BR"))
	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"domain"}).AddRow("https://dominio-pt.com"))

	sc, err := New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Domain != "https://dominio-pt.com" {
		t.Errorf("domain = %q", sc.Domain)
	}
	if sc.PrimaryLanguage != "pt" {
		t.Errorf("primary language = %q, want %q", sc.PrimaryLanguage, "pt")
	}
}

func TestResolve_NoVerifiedDomain(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow("en"))
	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(siteID).
		WillReturnError(pgx.ErrNoRows)

	sc, err := New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err != nil {
		t.Fatalf("no verified domain must not fail resolution, got %v", err)
	}
	if sc.Domain != "" {
		t.Errorf("domain = %q, want empty (caller falls back)", sc.Domain)
	}
	if sc.PrimaryLanguage != "en" {
		t.Errorf("primary language = %q, want %q", sc.PrimaryLanguage, "en")
	}
}

func TestResolve_NoLocale(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow(""))
	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"domain"}).AddRow("https://exemplo.com"))

	sc, err := New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.PrimaryLanguage != "" {
		t.Errorf("primary language = %q, want empty (caller falls back to pt)", sc.PrimaryLanguage)
	}
	if sc.Domain != "https://exemplo.com" {
		t.Errorf("domain = %q", sc.Domain)
	}
}

func TestResolve_LocaleError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WillReturnError(errors.New("db down"))

	_, err = New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err == nil {
		t.Error("expected locale lookup failure to propagate")
	}
}

func TestPGStore_PrimaryDomainErrNoRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err = NewPGStore(mock).PrimaryDomain(context.Background(), uuid.New())
	if !errors.Is(err, ErrNoVerifiedDomain) {
		t.Errorf("expected ErrNoVerifiedDomain, got %v", err)
	}
}

func TestPGStore_NoLocaleReturnsEmpty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow(""))

	locale, err := NewPGStore(mock).SiteLocale(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locale != "" {
		t.Errorf("locale = %q, want empty", locale)
	}
}

func TestResolve_NormalizesTrailingSlash(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	siteID := uuid.New()
	mock.ExpectQuery(`SELECT COALESCE\(locale, ''\) FROM sites WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"locale"}).AddRow("en-US"))
	mock.ExpectQuery(`SELECT domain FROM site_domains WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"domain"}).AddRow("https://aiworksimple.com/"))

	sc, err := New(NewPGStore(mock)).Resolve(context.Background(), siteID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Domain != "https://aiworksimple.com" {
		t.Errorf("domain = %q, want trailing slash trimmed", sc.Domain)
	}
}
