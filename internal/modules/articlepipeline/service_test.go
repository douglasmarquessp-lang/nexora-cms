package articlepipeline

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
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

func TestSetEventBus_Nil(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	svc.SetEventBus(nil)
}

// --- Validation Tests ---

func TestCreatePipeline_EmptyTitle(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreatePipeline(context.Background(), uuid.New(), uuid.New(), CreatePipelineRequest{
		Title:    "",
		Language: "pt",
	})
	if err != ErrInvalidTitle {
		t.Errorf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestCreatePipeline_InvalidLanguage(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreatePipeline(context.Background(), uuid.New(), uuid.New(), CreatePipelineRequest{
		Title:    "Test Article",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreatePipeline_InvalidPriority(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	priority := 99

	_, err := svc.CreatePipeline(context.Background(), uuid.New(), uuid.New(), CreatePipelineRequest{
		Title:    "Test",
		Language: "pt",
		Priority: &priority,
	})
	if err != ErrInvalidPriority {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestCreatePipeline_DBError(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreatePipeline(context.Background(), uuid.New(), uuid.New(), CreatePipelineRequest{
		Title:    "Test",
		Language: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetPipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipelineDetail_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetPipelineDetail(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListPipelines_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListPipelines(context.Background(), uuid.New(), "", "", 10, 0)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestUpdatePipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.UpdatePipeline(context.Background(), uuid.New(), uuid.New(), UpdatePipelineRequest{})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestDeletePipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	err := svc.DeletePipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestStartPipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.StartPipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestPausePipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.PausePipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestResumePipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ResumePipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCancelPipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CancelPipeline(context.Background(), uuid.New(), uuid.New(), "test")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestRetryStage_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.RetryStage(context.Background(), uuid.New(), uuid.New(), "research")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestRestartPipeline_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.RestartPipeline(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipelineStages_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetPipelineStages(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestUpdateStage_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.UpdateStage(context.Background(), uuid.New(), uuid.New(), "research", UpdateStageRequest{
		Status: StepStatusRunning,
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestRecordMetric_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.RecordMetric(context.Background(), uuid.New(), CreateMetricRequest{
		MetricName:  "test_metric",
		MetricValue: 1.0,
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipelineMetrics_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetPipelineMetrics(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateQualityReport_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateQualityReport(context.Background(), uuid.New(), CreateQualityReportRequest{
		StageName: "research",
		Score:     85.0,
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetQualityReports_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetQualityReports(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateCandidate_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateCandidate(context.Background(), uuid.New(), uuid.New(), CreateCandidateRequest{
		Title: "Test Article",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateCandidate_EmptyTitle(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateCandidate(context.Background(), uuid.New(), uuid.New(), CreateCandidateRequest{
		Title: "",
	})
	if err != ErrInvalidTitle {
		t.Errorf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestListCandidates_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListCandidates(context.Background(), uuid.New(), "", 10, 0)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipelineStats_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetPipelineStats(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

// --- Helper Tests ---

func TestCoalesceStr(t *testing.T) {
	if v := coalesceStr("", "default"); v != "default" {
		t.Errorf("expected default, got %s", v)
	}
	if v := coalesceStr("hello", "default"); v != "hello" {
		t.Errorf("expected hello, got %s", v)
	}
}

func TestCoalesceInt(t *testing.T) {
	if v := coalesceInt(0, 5); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
	if v := coalesceInt(3, 5); v != 3 {
		t.Errorf("expected 3, got %d", v)
	}
}

func TestToJSON(t *testing.T) {
	v := toJSON(map[string]string{"key": "value"})
	if v == "{}" {
		t.Error("expected non-empty JSON")
	}
}

func TestParseJSON_Empty(t *testing.T) {
	m := parseJSON("")
	if m != nil {
		t.Error("expected nil for empty input")
	}
}

func TestParseJSON_Valid(t *testing.T) {
	m := parseJSON(`{"key":"value"}`)
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if m["key"] != "value" {
		t.Errorf("expected value, got %v", m["key"])
	}
}

func TestCoalesceIntPtr(t *testing.T) {
	if v := coalesceIntPtr(nil); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
	v := 10
	if r := coalesceIntPtr(&v); r != 10 {
		t.Errorf("expected 10, got %d", r)
	}
}

func TestParseChecksJSON_Empty(t *testing.T) {
	c := parseChecksJSON("")
	if c != nil {
		t.Error("expected nil for empty input")
	}
}

func TestParseChecksJSON_Valid(t *testing.T) {
	c := parseChecksJSON(`[{"name":"test","passed":true}]`)
	if c == nil || len(c) != 1 {
		t.Fatal("expected 1 check")
	}
	if c[0].Name != "test" || !c[0].Passed {
		t.Errorf("unexpected check: %+v", c[0])
	}
}
