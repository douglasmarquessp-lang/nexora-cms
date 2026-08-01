package migrate

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/database"
)

// TestAdvisoryLockIDDistinctFromGolangMigrate guards against a self-deadlock:
// the outer advisory lock must never use the same ID that golang-migrate
// derives internally for the same database, or a process would block on its
// own lock and fail with ErrLockTimeout.
func TestAdvisoryLockIDDistinctFromGolangMigrate(t *testing.T) {
	internalID, err := database.GenerateAdvisoryLockId("railway", "public", "schema_migrations")
	if err != nil {
		t.Fatalf("GenerateAdvisoryLockId: %v", err)
	}

	var internalNum int64
	if _, err := fmt.Sscanf(internalID, "%d", &internalNum); err != nil {
		t.Fatalf("parse internal lock id: %v", err)
	}

	if advisoryLockID == internalNum {
		t.Fatalf("advisoryLockID %d collides with golang-migrate's internal lock id", advisoryLockID)
	}
}

// TestSourceURLResolution mirrors golang-migrate's file source parseURL logic
// to ensure the composed "file://"+dir URL resolves to the expected directory
// for both relative (local dev) and absolute (container) paths.
func TestSourceURLResolution(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		want string
	}{
		{name: "relative", dir: "migrations", want: "migrations"},
		{name: "absolute", dir: "/app/migrations", want: "/app/migrations"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := "file://" + tc.dir

			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("url.Parse(%q): %v", raw, err)
			}

			p := u.Opaque
			if len(p) == 0 {
				p = u.Host + u.Path
			}
			if len(p) == 0 {
				t.Fatal("empty path")
			}
			if p[0:1] == "." || p[0:1] != "/" {
				abs, err := filepath.Abs(p)
				if err != nil {
					t.Fatalf("filepath.Abs: %v", err)
				}
				p = abs
			}

			if !strings.HasSuffix(p, tc.want) {
				t.Fatalf("resolved path %q does not end with %q", p, tc.want)
			}
		})
	}
}
