package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func newTestLogger(t *testing.T, m pgxmock.PgxPoolIface) *Logger {
	t.Helper()
	cfg := &config.Config{}
	_ = cfg
	return New(m, logger.New(cfg))
}

func TestLog_NilUserIDConvertedWhenUUIDNil(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	userIDNul := uuid.Nil
	sid := uuid.New()
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(nil, &sid, "publish", "post", nil, map[string]interface{}{}, "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	l := newTestLogger(t, m)
	l.Log(context.Background(), Entry{
		UserID:     &userIDNul,
		SiteID:     &sid,
		Action:     "publish",
		EntityType: "post",
		Payload:    map[string]interface{}{},
	})
}

func TestLog_NilSiteIDConvertedWhenUUIDNil(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	siteIDNul := uuid.Nil
	uid := uuid.New()
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(&uid, nil, "workflow.job.created", "workflow_job", nil, map[string]interface{}{}, "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	l := newTestLogger(t, m)
	l.Log(context.Background(), Entry{
		UserID:     &uid,
		SiteID:     &siteIDNul,
		Action:     "workflow.job.created",
		EntityType: "workflow_job",
		Payload:    map[string]interface{}{},
	})
}

func TestLog_NilPayloadNormalized(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	uid := uuid.New()
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(&uid, nil, "user.login", "user", &uid, map[string]interface{}{}, "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	l := newTestLogger(t, m)
	l.Log(context.Background(), Entry{
		UserID:     &uid,
		Action:     "user.login",
		EntityType: "user",
		EntityID:   &uid,
		Payload:    nil,
	})
}

func TestLog_RealUserIDKept(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	uid := uuid.New()
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(&uid, nil, "user.login", "user", nil, map[string]interface{}{}, "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	l := newTestLogger(t, m)
	l.Log(context.Background(), Entry{
		UserID:     &uid,
		Action:     "user.login",
		EntityType: "user",
		Payload:    map[string]interface{}{},
	})
}
