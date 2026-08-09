package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/ai"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

// fakeGeneratedPublisher records how many times PublishGeneratedArticle is
// called and returns a configured result, so the workflow publish step can be
// tested deterministically without a real publisher engine.
type fakeGeneratedPublisher struct {
	calls   int
	pub     *publisher.Publication
	err     error
	lastReq *publisher.PublishGeneratedRequest
}

func (f *fakeGeneratedPublisher) PublishGeneratedArticle(_ context.Context, req publisher.PublishGeneratedRequest) (*publisher.Publication, error) {
	f.calls++
	f.lastReq = &req
	if f.err != nil {
		return nil, f.err
	}
	return f.pub, nil
}

// workflowAIManager builds an AI manager with the deterministic MockProvider,
// so the AI pipeline stages (research, writer, editorial, seo, quality) run
// offline in tests — no network, no real API calls.
func workflowAIManager() *ai.Manager {
	log := logger.New(&config.Config{})
	m := ai.NewManager(ai.DefaultConfig(), log)
	p := ai.NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ai.ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	return m
}

var wfJobUserID = uuid.New()

var wfJobCols = []string{
	"id", "site_id", "user_id", "title", "content_type",
	"language", "target_language", "status", "current_step", "progress",
	"priority", "word_count", "tone", "audience", "keywords", "style_slug",
	"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
	"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
	"cancelled_at", "created_by", "created_at", "updated_at",
}

func wfJobRow(jobID, siteID uuid.UUID, status JobStatus, progress float64) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(wfJobCols).AddRow(
		jobID, siteID, &wfJobUserID, "TITULO-UNICO-7", "article", "pt", "", status, "", progress,
		5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, now, nil, nil, &wfJobUserID, now, now,
	)
}

var wfStepCols = []string{
	"id", "workflow_job_id", "step_name", "display_name", "status", "progress",
	"depends_on", "retry_count", "max_retries", "started_at", "completed_at",
	"duration_ms", "error_message", "metadata", "created_at", "updated_at",
}

func wfStepRow(jobID uuid.UUID, name string, status StepStatus) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(wfStepCols).AddRow(
		uuid.New(), jobID, name, name, status, float64(0),
		[]string{}, 0, 3, nil, nil, int64(0), "", "{}", now, now,
	)
}

// wfStepsRows returns all 8 workflow steps with the ones listed in `done`
// marked completed — feeds calcProgress (listSteps). The completed count over
// the 8 rows drives the progress value, so the callers must pass the exact
// prefix of completed steps.
func wfStepsRows(jobID uuid.UUID, done []string) *pgxmock.Rows {
	now := time.Now()
	rows := pgxmock.NewRows(wfStepCols)
	for _, st := range AllWorkflowSteps {
		status := StepStatusPending
		for _, d := range done {
			if string(st) == d {
				status = StepStatus("completed")
				break
			}
		}
		rows.AddRow(uuid.New(), jobID, string(st), string(st), status, float64(0),
			[]string{}, 0, 3, nil, nil, int64(0), "", "{}", now, now)
	}
	return rows
}

// wfMockStepOrder mirrors AllWorkflowSteps execution order (human_writer is
// not AI-mapped and is skipped between writer and editorial_engine).
var wfMockStepOrder = []string{
	"research", "writer", "human_writer", "editorial_engine", "seo_engine", "quality_check",
}

// wfMappedSteps are the AI-mapped steps — they contribute to the completed
// count used by calcProgress (human_writer is skipped, not completed).
var wfMappedSteps = []string{"research", "writer", "editorial_engine", "seo_engine", "quality_check"}

func stepsBefore(step string, list []string) []string {
	for i, s := range list {
		if s == step {
			return list[:i]
		}
	}
	return list
}

// expectMappedStep queues the mocked DB sequence for one AI-mapped workflow
// step: getStepByName -> mark running -> mark completed -> current_step ->
// calcProgress (listSteps) -> progress write -> getJobByID rebuild.
func expectMappedStep(t *testing.T, mock pgxmock.PgxPoolIface, jobID uuid.UUID, step string, done []string) {
	t.Helper()

	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, step).
		WillReturnRows(wfStepRow(jobID, step, StepStatusPending))

	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(jobID, step).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE workflow_steps SET status = 'completed'`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), jobID, step).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE workflow_jobs SET current_step`).
		WithArgs(step, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	progress := float64(len(done)) / float64(len(AllWorkflowSteps)) * 100
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 ORDER BY created_at ASC`).
		WithArgs(jobID).
		WillReturnRows(wfStepsRows(jobID, done))

	mock.ExpectExec(`UPDATE workflow_jobs SET progress`).
		WithArgs(progress, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, pgxmock.AnyArg()).
		WillReturnRows(wfJobRow(jobID, uuid.New(), JobStatusRunning, progress))
}

// TestExecuteWorkflow_PublisherStepFailsOnGate verifies that when publishing is
// blocked (e.g. SEO gate), the publisher step is marked failed, the job is
// marked failed with current_step = publisher and progress < 100, the
// "finished" step is never executed, PublishGeneratedArticle is called exactly
// once, and the history entry carries the real error message.
func TestExecuteWorkflow_PublisherStepFailsOnGate(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	jobID := uuid.New()
	siteID := uuid.New()
	pubErrMsg := "seo score below minimum for publishing"

	fake := &fakeGeneratedPublisher{err: errors.New(pubErrMsg)}
	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.publisherSvc = fake
	svc.aiManager = workflowAIManager()

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, 0))

	for _, step := range wfMockStepOrder {
		if step == "human_writer" {
			// not AI-mapped -> skipped
			mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
				WithArgs(jobID, step).
				WillReturnRows(wfStepRow(jobID, step, StepStatusPending))
			mock.ExpectExec(`UPDATE workflow_steps SET status = 'skipped'`).
				WithArgs(jobID, step).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			continue
		}
		done := stepsBefore(step, wfMappedSteps)
		expectMappedStep(t, mock, jobID, step, done)
	}

	// publisher: running then failed by the blocked publish
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "publisher").
		WillReturnRows(wfStepRow(jobID, "publisher", StepStatusPending))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(jobID, "publisher").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE workflow_steps SET status = 'failed', error_message = \$1`).
		WithArgs(pubErrMsg, jobID, "publisher").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'failed', error_message = \$1, current_step = \$2`).
		WithArgs(pubErrMsg, "publisher", jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	progress := float64(len(wfMappedSteps)) / float64(len(AllWorkflowSteps)) * 100 // 62.5
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 ORDER BY created_at ASC`).
		WithArgs(jobID).
		WillReturnRows(wfStepsRows(jobID, wfMappedSteps))

	mock.ExpectExec(`UPDATE workflow_jobs SET progress`).
		WithArgs(progress, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`INSERT INTO workflow_history`).
		WithArgs(pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), pgxmock.AnyArg(), "workflow.completed",
			pgxmock.AnyArg(), pgxmock.AnyArg(), "running", "failed", pgxmock.AnyArg(),
			pubErrMsg, &wfJobUserID, int64(0), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc.executeWorkflowAsync(context.Background(), siteID, jobID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected PublishGeneratedArticle called exactly once, got %d", fake.calls)
	}
	// The "finished" step must NOT be executed: any finished query/update would
	// leave an unmet expectation above (see ExpectationsWereMet).
}

// TestExecuteWorkflow_PublisherStepSuccess verifies the happy path: publisher
// completed, "finished" executed afterwards, job completed with progress 100,
// publication_id persisted, and PublishGeneratedArticle called exactly once
// with the job title.
func TestExecuteWorkflow_PublisherStepSuccess(t *testing.T) {
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

	fake := &fakeGeneratedPublisher{pub: &publisher.Publication{ID: pubID, Title: "Artigo Teste"}}
	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.publisherSvc = fake
	svc.aiManager = workflowAIManager()

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, 0))

	expectStepsThroughPublisher(t, mock, jobID, pubID)

	// finished: only executed after the publisher step succeeded
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
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, progressFinal))

	// post-loop: job completed
	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'completed', progress = 100`).
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	svc.executeWorkflowAsync(context.Background(), siteID, jobID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected PublishGeneratedArticle called exactly once, got %d", fake.calls)
	}
	if fake.lastReq == nil || fake.lastReq.Title != "TITULO-UNICO-7" {
		t.Errorf("expected publish request carrying the job title, got title=%q", fake.lastReq.Title)
	}
}

// expectStepsThroughPublisher registers the pgxmock expectations shared by the
// publish-flow tests: job load, mapped steps (human_writer skipped), publisher
// step success (publication created), progress calc, and the immediate
// publication_id persistence on the job row.
func expectStepsThroughPublisher(t *testing.T, mock pgxmock.PgxPoolIface, jobID, pubID uuid.UUID) {
	t.Helper()
	for _, step := range wfMockStepOrder {
		if step == "human_writer" {
			// not AI-mapped -> skipped
			mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
				WithArgs(jobID, step).
				WillReturnRows(wfStepRow(jobID, step, StepStatusPending))
			mock.ExpectExec(`UPDATE workflow_steps SET status = 'skipped'`).
				WithArgs(jobID, step).
				WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			continue
		}
		done := stepsBefore(step, wfMappedSteps)
		expectMappedStep(t, mock, jobID, step, done)
	}

	// publisher: running -> success (publication created)
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "publisher").
		WillReturnRows(wfStepRow(jobID, "publisher", StepStatusPending))
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

	// publication_id persisted immediately after the publish step succeeds,
	// so a post-publication stage failure never orphans the job from its article
	mock.ExpectExec(`UPDATE workflow_jobs SET publication_id`).
		WithArgs(pubID, jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

// reviewFailingProvider wraps a MockProvider and returns a provider error for
// the final-review prompt ("Review the content for quality...") — the only
// prompt unique to the "finished" stage. All other stages behave like the
// mock, so publishing succeeds and only the final review fails.
type reviewFailingProvider struct {
	base      *ai.MockProvider
	reviewHit bool
}

func (r *reviewFailingProvider) Generate(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResult, error) {
	if strings.Contains(req.Prompt, "Review the content for quality") {
		r.reviewHit = true
		return nil, ai.ErrProviderUnavailable
	}
	return r.base.Generate(ctx, req)
}
func (r *reviewFailingProvider) GenerateStream(ctx context.Context, req ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return r.base.GenerateStream(ctx, req)
}
func (r *reviewFailingProvider) Embeddings(ctx context.Context, input string) (*ai.EmbeddingResult, error) {
	return r.base.Embeddings(ctx, input)
}
func (r *reviewFailingProvider) Summarize(ctx context.Context, req ai.SummarizeRequest) (string, error) {
	return r.base.Summarize(ctx, req)
}
func (r *reviewFailingProvider) Rewrite(ctx context.Context, req ai.RewriteRequest) (string, error) {
	return r.base.Rewrite(ctx, req)
}
func (r *reviewFailingProvider) Classify(ctx context.Context, req ai.ClassifyRequest) (*ai.ClassifyResult, error) {
	return r.base.Classify(ctx, req)
}
func (r *reviewFailingProvider) Health(ctx context.Context) (*ai.HealthStatus, error) {
	return r.base.Health(ctx)
}
func (r *reviewFailingProvider) Name() string {
	return r.base.Name()
}
func (r *reviewFailingProvider) Capabilities() []ai.Capability {
	return r.base.Capabilities()
}

// TestExecuteWorkflow_PostPublishFailureCompletesJob guards the guarantee that
// a failed stage AFTER a successful publish (e.g. the final review hitting a
// provider outage) never marks the job failed and never loses the
// publication_id link: the job is completed and the article stays published.
func TestExecuteWorkflow_PostPublishFailureCompletesJob(t *testing.T) {
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

	fake := &fakeGeneratedPublisher{pub: &publisher.Publication{ID: pubID, Title: "Artigo Teste"}}
	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.publisherSvc = fake

	m := ai.NewManager(ai.DefaultConfig(), log)
	rp := &reviewFailingProvider{base: ai.NewMockProvider("mock", "mock-model", nil)}
	m.RegisterProvider(rp, ai.ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	svc.aiManager = m

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRow(jobID, siteID, JobStatusRunning, 0))

	expectStepsThroughPublisher(t, mock, jobID, pubID)

	// finished: the final-review stage fails (provider outage)
	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE workflow_job_id = \$1 AND step_name = \$2`).
		WithArgs(jobID, "finished").
		WillReturnRows(wfStepRow(jobID, "finished", StepStatusPending))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(jobID, "finished").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE workflow_steps SET status = 'failed', error_message = \$1`).
		WithArgs(pgxmock.AnyArg(), jobID, "finished").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// the article was already published: job must be completed, not failed
	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'completed', progress = 100`).
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	svc.executeWorkflowAsync(context.Background(), siteID, jobID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("expected PublishGeneratedArticle called exactly once, got %d", fake.calls)
	}
	if !rp.reviewHit {
		t.Error("expected the final-review prompt to have failed the provider")
	}
}
