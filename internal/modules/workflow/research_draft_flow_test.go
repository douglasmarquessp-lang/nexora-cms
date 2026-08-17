package workflow

import (
	"context"
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

// recordingProvider wraps a MockProvider and records every Generate request,
// so tests can assert what actually reached the model prompt (word counts,
// briefing/outline inline content, research grounding).
type recordingProvider struct {
	base      *ai.MockProvider
	prompts   []string
	grounding int
}

func (r *recordingProvider) Generate(ctx context.Context, req ai.CompletionRequest) (*ai.CompletionResult, error) {
	r.prompts = append(r.prompts, req.Prompt)
	if req.Grounding != nil && req.Grounding.Enabled {
		r.grounding++
	}
	return r.base.Generate(ctx, req)
}
func (r *recordingProvider) GenerateStream(ctx context.Context, req ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return r.base.GenerateStream(ctx, req)
}
func (r *recordingProvider) Embeddings(ctx context.Context, input string) (*ai.EmbeddingResult, error) {
	return r.base.Embeddings(ctx, input)
}
func (r *recordingProvider) Summarize(ctx context.Context, req ai.SummarizeRequest) (string, error) {
	return r.base.Summarize(ctx, req)
}
func (r *recordingProvider) Rewrite(ctx context.Context, req ai.RewriteRequest) (string, error) {
	return r.base.Rewrite(ctx, req)
}
func (r *recordingProvider) Classify(ctx context.Context, req ai.ClassifyRequest) (*ai.ClassifyResult, error) {
	return r.base.Classify(ctx, req)
}
func (r *recordingProvider) Health(ctx context.Context) (*ai.HealthStatus, error) {
	return r.base.Health(ctx)
}
func (r *recordingProvider) Name() string {
	return r.base.Name()
}
func (r *recordingProvider) Capabilities() []ai.Capability {
	return r.base.Capabilities()
}

// wfJobRowWC is wfJobRow with configurable language and word_count (the two
// fields the pipeline depth fix depends on).
func wfJobRowWC(jobID, siteID uuid.UUID, status JobStatus, progress float64, lang string, wordCount int) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(wfJobCols).AddRow(
		jobID, siteID, &wfJobUserID, "TITULO-UNICO-7", "article", lang, "", status, "", progress,
		5, wordCount, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, now, nil, nil, &wfJobUserID, now, now,
	)
}

// expectHappyPath registers the full success-sequence expectations: job load,
// mapped steps (human_writer skipped), publisher success with publication_id
// persistence, finished step, and final job completion.
func expectHappyPath(t *testing.T, mock pgxmock.PgxPoolIface, jobID, siteID, pubID uuid.UUID, wordCount int) {
	t.Helper()
	expectStepsThroughPublisher(t, mock, jobID, pubID, wordCount)

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

	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'completed', progress = 100`).
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

// TestExecuteWorkflow_DraftPromptDepthDefaults guards the root-cause fix of the
// shallow-article bug: a job with word_count = 0 must produce a draft prompt
// with the package defaults (never "Word Count: 0"), the briefing and outline
// must run inline before the writer, and the research stage must run with
// grounding enabled.
func TestExecuteWorkflow_DraftPromptDepthDefaults(t *testing.T) {
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
	rp := &recordingProvider{base: ai.NewMockProvider("mock", "mock-model", nil)}
	m.RegisterProvider(rp, ai.ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	svc.aiManager = m

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRowWC(jobID, siteID, JobStatusRunning, 0, "en", 0))

	expectHappyPath(t, mock, jobID, siteID, pubID, 0)

	svc.executeWorkflowAsync(context.Background(), siteID, jobID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}

	if len(rp.prompts) < 4 {
		t.Fatalf("expected at least research+briefing+outline+draft prompts, got %d", len(rp.prompts))
	}
	// Stage order: research -> inline briefing -> inline outline -> draft.
	if !strings.Contains(rp.prompts[0], "TITULO-UNICO-7") {
		t.Errorf("research prompt should carry the topic, got: %.60s…", rp.prompts[0])
	}
	if rp.grounding != 1 {
		t.Errorf("expected the research stage to run with grounding enabled, got %d", rp.grounding)
	}
	draft := rp.prompts[3]
	// The rebuild row feeding the writer step uses the default job language
	// (pt), so the template renders in PT — the numbers are what matters.
	if !strings.Contains(draft, "Tamanho Alvo: 1200 palavras (mínimo de 1000 palavras") {
		t.Errorf("draft prompt must default to 1200/1000 words, got: %.100s…", draft)
	}
	if !strings.Contains(draft, "Mock response for:") {
		t.Errorf("draft prompt must include the inline briefing/outline content, got: %.100s…", draft)
	}
	if strings.Contains(draft, "Tamanho Alvo: 0") {
		t.Errorf("draft prompt must never contain word count 0: %.100s…", draft)
	}
	if fake.lastReq == nil {
		t.Fatal("expected PublishGeneratedArticle to have been called")
	}
	if fake.lastReq.ContentType != "article" {
		t.Errorf("expected ContentType article, got %q", fake.lastReq.ContentType)
	}
	if fake.lastReq.ResearchFacts != 0 {
		t.Errorf("expected ResearchFacts passthrough (grounding fallback has no facts), got %d", fake.lastReq.ResearchFacts)
	}
}

// TestExecuteWorkflow_CustomWordCountPreserved asserts a caller-specified word
// count reaches the draft prompt verbatim while the minimum stays at the
// default.
func TestExecuteWorkflow_CustomWordCountPreserved(t *testing.T) {
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

	m := ai.NewManager(ai.DefaultConfig(), log)
	rp := &recordingProvider{base: ai.NewMockProvider("mock", "mock-model", nil)}
	m.RegisterProvider(rp, ai.ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	svc.aiManager = m

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(wfJobRowWC(jobID, siteID, JobStatusRunning, 0, "en", 1500))

	expectHappyPath(t, mock, jobID, siteID, pubID, 1500)

	svc.executeWorkflowAsync(context.Background(), siteID, jobID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
	if len(rp.prompts) < 4 {
		t.Fatalf("expected at least research+briefing+outline+draft prompts, got %d", len(rp.prompts))
	}
	draft := rp.prompts[3]
	if !strings.Contains(draft, "Tamanho Alvo: 1500 palavras (mínimo de 1000 palavras") {
		t.Errorf("draft prompt must preserve the caller word count 1500, got: %.100s…", draft)
	}
}
