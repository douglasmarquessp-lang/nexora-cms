package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

// TestStartJob_FailedJobRejected guards Sprint 6.7 part 16: starting a failed
// job must be rejected — otherwise the old behavior skipped every failed step
// (including publisher) and silently marked the job completed without a
// successful publication (publication_id stayed NULL).
func TestStartJob_FailedJobRejected(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	jobID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusFailed, 62.5))

	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	_, err = svc.StartJob(context.Background(), siteID, jobID)
	if !errors.Is(err, ErrJobInFailedState) {
		t.Fatalf("expected ErrJobInFailedState, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

// TestRetryStep_RelaunchesPipeline guards the Sprint 6.7 fix for jobs stuck in
// "running" forever: RetryStep resets the failed step AND relaunches the
// pipeline goroutine. Without the relaunch the job would stay in running
// forever (pre-existing bug). The async goroutine is fully mocked like the
// publish-flow tests: completed steps are skipped, the retried publisher step
// runs (fake publisher), and the finished step runs last.
func TestRetryStep_RelaunchesPipeline(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	jobID := uuid.New()
	siteID := uuid.New()
	pubID := uuid.New()

	fake := &fakeGeneratedPublisher{pub: &publisher.Publication{ID: pubID, Title: "Artigo Retry"}}
	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.publisherSvc = fake
	svc.aiManager = workflowAIManager()

	// --- sync RetryStep part ---
	// job was failed (retry_count 0 of 3)
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusFailed, 62.5))
	// the failed step (publisher)
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "publisher").
		WillReturnRows(wfStepRow(jobID, "publisher", StepStatusFailed))
	// job -> running, retry_count 1
	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'running', retry_count`).
		WithArgs(1, pgxmock.AnyArg(), jobID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// step -> running again
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running', progress = 0`).
		WithArgs(pgxmock.AnyArg(), jobID, "publisher").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// history + log
	mock.ExpectExec(`INSERT INTO workflow_history`).
		WithArgs(anyArgs(14)...).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO workflow_logs`).
		WithArgs(pgxmock.AnyArg(), jobID, "publisher", "info", pgxmock.AnyArg(), "null", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	// final sync read
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, 0))

	// --- async relaunched pipeline (executeWorkflowAsync) ---
	// job load
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, 0))
	// completed/skipped steps -> skipped with a single read each; the last
	// completed mapped step (quality_check) carries the draft in ai_content,
	// which the relaunched pipeline recovers so the publisher can publish it.
	for _, step := range []string{"research", "writer", "human_writer", "editorial_engine", "seo_engine", "quality_check"} {
		st := StepStatusCompleted
		if step == "human_writer" {
			st = StepStatusSkipped
		}
		meta := `{}`
		if step == "quality_check" {
			meta = `{"ai_content":"DRAFT-RETRY-OK"}`
		}
		mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
			WithArgs(jobID, step).
			WillReturnRows(wfStepRowMeta(jobID, step, st, meta))
	}
	// publisher (running after retry): publishWorkflowJob success
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "publisher").
		WillReturnRows(wfStepRow(jobID, "publisher", StepStatusRunning))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(jobID, "publisher").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'completed', progress = 100`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), jobID, "publisher").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_jobs SET current_step`).
		WithArgs("publisher", jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	progressPub := float64(len(wfMappedSteps)+1) / float64(len(AllWorkflowSteps)) * 100 // 75
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 ORDER BY created_at ASC`).
		WithArgs(jobID).
		WillReturnRows(wfStepsRows(jobID, append(wfMappedSteps, "publisher")))
	mock.ExpectExec(`UPDATE workflow_jobs SET progress`).
		WithArgs(progressPub, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_jobs SET publication_id`).
		WithArgs(pubID, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// finished (pending): AI final review, mirrors expectMappedStep
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "finished").
		WillReturnRows(wfStepRow(jobID, "finished", StepStatusPending))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(jobID, "finished").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'completed'`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), jobID, "finished").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_jobs SET current_step`).
		WithArgs("finished", jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	progressFinal := float64(len(wfMappedSteps)+2) / float64(len(AllWorkflowSteps)) * 100 // 87.5
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 ORDER BY created_at ASC`).
		WithArgs(jobID).
		WillReturnRows(wfStepsRows(jobID, append(wfMappedSteps, "publisher", "finished")))
	mock.ExpectExec(`UPDATE workflow_jobs SET progress`).
		WithArgs(progressFinal, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, pgxmock.AnyArg()).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, progressFinal))
	// post-loop: completed
	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'completed', progress = 100`).
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	job, err := svc.RetryStep(context.Background(), siteID, jobID, RetryRequest{StepName: "publisher"})
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.Status != JobStatusRunning {
		t.Fatalf("expected job running after retry, got %v", job)
	}

	// The relaunched pipeline runs in a goroutine — poll until every mocked
	// expectation is consumed (deadline guards a goroutine that died early).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := mock.ExpectationsWereMet(); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected the retried publisher step to run PublishGeneratedArticle once, got %d calls", fake.calls)
	}
}

func anyArgs(n int) []interface{} {
	args := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		args = append(args, pgxmock.AnyArg())
	}
	return args
}

// wfStepRowMeta is wfStepRow with a custom metadata JSONB cell, so skipped
// steps can expose the accumulated ai_content that a relaunched pipeline
// recovers (Sprint 6.7 content-recovery fix).
func wfStepRowMeta(jobID uuid.UUID, name string, status StepStatus, metadata string) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(wfStepCols).AddRow(
		uuid.New(), jobID, name, name, status, float64(0),
		[]string{}, 0, 3, nil, nil, int64(0), "", metadata, now, now,
	)
}