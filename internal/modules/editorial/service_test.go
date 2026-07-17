package editorial

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/audit"
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

func setupMockDB(t *testing.T) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.auditLog = audit.New(nil, log)
	return svc, mock
}

func TestService_GetStats(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COALESCE.*FROM posts WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"published", "draft", "scheduled", "archived", "total",
		}).AddRow(5, 3, 2, 1, 11))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(20))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(7))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(15))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM editorial_tasks WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(10))

	mock.ExpectQuery(`SELECT COALESCE.*FROM editorial_tasks WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(4))

	mock.ExpectQuery(`SELECT COALESCE.*FROM approval_requests WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	stats, err := svc.GetStats(context.Background(), siteID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalPosts != 11 {
		t.Errorf("expected 11 total posts, got %d", stats.TotalPosts)
	}
	if stats.PublishedPosts != 5 {
		t.Errorf("expected 5 published, got %d", stats.PublishedPosts)
	}
}

func TestService_GetTask_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	taskID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_tasks WHERE`).
		WithArgs(taskID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetTask(context.Background(), siteID, taskID)
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestService_DeleteTask(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	taskID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM editorial_tasks WHERE`).
		WithArgs(taskID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "title", "description", "status", "priority",
			"assignee_id", "due_date", "post_id", "created_by", "completed_at", "created_at", "updated_at",
		}).AddRow(
			taskID, siteID, "Task", "Desc", "pending", "medium",
			nil, nil, nil, uuid.New(), nil, now, now,
		))

	mock.ExpectExec(`UPDATE editorial_tasks SET deleted_at`).
		WithArgs(taskID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.DeleteTask(context.Background(), siteID, taskID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
}

func TestService_ListRevisions(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{
		"id", "post_id", "site_id", "author_id", "version", "title",
		"content", "excerpt", "slug", "post_meta", "metadata", "summary", "change_log", "created_at",
	}).AddRow(
		uuid.New(), postID, siteID, uuid.New(), 2, "V2",
		`[{"type":"text"}]`, "Excerpt", "v2", "{}", "{}", "Sum", "Log", now,
	).AddRow(
		uuid.New(), postID, siteID, uuid.New(), 1, "V1",
		"[]", "Excerpt", "v1", "{}", "{}", "", "", now,
	)

	mock.ExpectQuery(`SELECT .+ FROM post_revisions WHERE`).
		WithArgs(postID, siteID).
		WillReturnRows(rows)

	revisions, err := svc.ListRevisions(context.Background(), siteID, postID)
	if err != nil {
		t.Fatalf("ListRevisions failed: %v", err)
	}

	if len(revisions) != 2 {
		t.Errorf("expected 2 revisions, got %d", len(revisions))
	}
	if revisions[0].Version != 2 {
		t.Errorf("expected version 2 first (desc order), got %d", revisions[0].Version)
	}
}

func TestService_RequestApproval(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`INSERT INTO approval_requests`).
		WithArgs(pgxmock.AnyArg(), siteID, postID, userID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	req, err := svc.RequestApproval(context.Background(), siteID, postID, userID)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}

	if req.Status != ApprovalStatusPending {
		t.Errorf("expected pending, got %q", req.Status)
	}
	if req.PostID != postID {
		t.Errorf("expected postID %s, got %s", postID, req.PostID)
	}
}

func TestService_ReviewApproval_InvalidStatus(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	approvalID := uuid.New()
	reviewerID := uuid.New()

	_, err := svc.ReviewApproval(context.Background(), siteID, postID, approvalID, reviewerID, ApprovalActionRequest{
		Status: "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestService_ListCalendarEvents(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM editorial_calendar_events WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "title", "description", "event_date", "event_type", "post_id", "color", "created_by", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), siteID, "Event 1", "Desc", "2026-07-16", "publication", nil, "", nil, now, now,
		))

	events, err := svc.ListCalendarEvents(context.Background(), siteID, "", "")
	if err != nil {
		t.Fatalf("ListCalendarEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestService_DeleteCalendarEvent(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	eventID := uuid.New()

	mock.ExpectExec(`DELETE FROM editorial_calendar_events WHERE`).
		WithArgs(eventID, siteID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := svc.DeleteCalendarEvent(context.Background(), siteID, eventID)
	if err != nil {
		t.Fatalf("DeleteCalendarEvent failed: %v", err)
	}
}

func TestService_ListWidgets(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	cfg := `{"show_recent":true}`

	mock.ExpectQuery(`SELECT .+ FROM editorial_widgets WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "widget_type", "title", "config", "position", "enabled", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), siteID, "recent_posts", "Recent Posts", cfg, 1, true, now, now,
		))

	widgets, err := svc.ListWidgets(context.Background(), siteID)
	if err != nil {
		t.Fatalf("ListWidgets failed: %v", err)
	}

	if len(widgets) != 1 {
		t.Errorf("expected 1 widget, got %d", len(widgets))
	}
	if widgets[0].WidgetType != "recent_posts" {
		t.Errorf("expected 'recent_posts', got %q", widgets[0].WidgetType)
	}
	if widgets[0].Enabled != true {
		t.Errorf("expected enabled true, got %v", widgets[0].Enabled)
	}
}

func TestService_UpdateWidget(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	widgetID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectExec(`UPDATE editorial_widgets SET`).
		WithArgs("recent_posts", `{"limit":5}`, true, widgetID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`SELECT .+ FROM editorial_widgets WHERE`).
		WithArgs(widgetID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "widget_type", "title", "config", "position", "enabled", "created_at", "updated_at",
		}).AddRow(
			widgetID, siteID, "recent_posts", "Recent", `{"limit":5}`, 0, true, now, now,
		))

	configMap := map[string]interface{}{"limit": 5}
	widget, err := svc.UpdateWidget(context.Background(), siteID, widgetID, UpdateWidgetRequest{
		WidgetType: strPtr("recent_posts"),
		Config:     &configMap,
		Enabled:    boolPtr(true),
	})
	if err != nil {
		t.Fatalf("UpdateWidget failed: %v", err)
	}

	if widget.WidgetType != "recent_posts" {
		t.Errorf("expected 'recent_posts', got %q", widget.WidgetType)
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
