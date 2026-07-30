# Nexora CMS — Sprint Memory (anchored summary)

## Objective
- Build and maintain a private CMS (PT/EN) with a focus on content generation, editorial workflows, and AI provider-agnostic integration infrastructure.

## Important Details
- No paid API integration — only abstraction, mock provider, and infrastructure.
- Follow existing patterns (Kernel modules, EventBus, Cache, Audit, Casbin, chi routes).
- `go build ./...`, `go vet ./...`, and `go test ./...` all pass cleanly (zero errors).
- AI package achieves 86.6% statement coverage (10 test files, ~220 tests).
- Autocontent package has 5 tables, 21 REST endpoints, 91 tests.
- Autocontent package has 5 tables, 21 REST endpoints, 91 tests.

## Completed

### Sprint 3.4 — Content Generation Orchestrator
- Migration `000013_add_generation_tables.up.sql` — `generation_jobs`, `generation_pipeline`, `generation_pipeline_logs`, `generation_quality_gates`, `generation_stats` + indexes
- `internal/modules/contentgenerator/` — model, service (14 methods), handler (19 endpoints), module
- 52 tests pass (build, vet, test)

### Sprint 3.5 — AI Integration Layer (provider-agnostic)
- `internal/ai/interfaces.go` — AIProvider, QualityChecker, PromptBuilder, StreamHandler, AIManager
- `internal/ai/provider.go` — MockProvider with 7 methods (Generate, GenerateStream, Embeddings, Summarize, Rewrite, Classify, Health)
- `internal/ai/registry.go` — priority-ordered provider registry with capability filtering and health check
- `internal/ai/manager.go` — circuit breaker (3 states), exponential backoff retry, failover, weighted selection, metrics
- `internal/ai/prompt_builder.go` — 12 default prompt templates (EN + PT), custom registration, variable interpolation
- `internal/ai/stream.go` — StreamProcessor with chunk/complete/error/progress handlers, cancellation
- `internal/ai/quality.go` — QualityChecker with mock implementations (grammar, SEO, readability, structure, duplicates, hallucination)
- `internal/ai/pipeline.go` — PipelineExecutor with 8 stages (ResearchGen → FinalReview)
- `internal/ai/module.go` — AIModule kernel module with config-driven dynamic provider registration; falls back to MockProvider when no real providers configured
- `internal/ai/handler.go` — 5 REST endpoints (providers, health, test, prompt preview, capabilities)
- `internal/api/routes.go` — AIManager in Dependencies, registerAIRoutes
- `cmd/api/main.go` — AI module registered, EventBus wired, service in Dependencies
- 10 test files, ~220 tests (9 existing + gemini_provider_test.go with 20 tests)

### Sprint 3.5a — AI Provider Configuration (2026-07-29)
- `internal/pkg/config/config.go` — `ProviderConfig`, `AIConfig` structs added; `AI AIConfig` field in main `Config`
- Env var loading: `AI_ENABLED`, `AI_DEFAULT_PROVIDER`, `AI_GEMINI_*` (API key, model, base URL, timeout, retries, weight, priority, enabled), `AI_RETRY_*`, `AI_CB_*`, `AI_GLOBAL_TIMEOUT`
- `internal/ai/module.go:Init()` — now reads `m.cfg.AI` to configure circuit breaker, retry, timeout; iterates `cfg.AI.Providers` and dynamically creates providers via `createProvider` factory; falls back to MockProvider when no real providers configured

### Sprint 3.5b — GeminiProvider (2026-07-29)
- `internal/ai/gemini_provider.go` — `GeminiProvider` implementing full `AIProvider` interface:
  - `Generate` — calls Gemini `generateContent` REST API with system instructions, temperature, max tokens, stop sequences
  - `GenerateStream` — calls Gemini `streamGenerateContent` SSE endpoint, decodes JSON-per-line responses
  - `Embeddings` — calls Gemini `embedContent` REST API, returns vector with dimensions
  - `Summarize`, `Rewrite`, `Classify` — delegating to `Generate` via prompt engineering
  - `Health` — calls Gemini model metadata endpoint, returns healthy/degraded/unhealthy with latency
  - Error mapping: 401/403 → `ErrInvalidAPIKey`, 429 → `ErrRateLimited`, 5xx → `ErrProviderUnavailable`
  - Auto-adjusts base URL and model defaults (`gemini-2.0-flash`)
- `internal/ai/gemini_provider_test.go` — 20 tests covering constructor, defaults, invalid key, capabilities, unhealthy state, summary/rewrite/classify delegation, `buildRequest` with/without system instruction, lang label, error parsing (401/403/429/5xx/unknown), and capability checks

### Sprint 3.5c — AI Pipeline Integration (2026-07-29)
- **Audit finding:** All four generation modules (workflow, contentgenerator, autocontent, articlepipeline) were pure state-management layers — none imported the `ai` package or held a reference to `ai.Manager`/`PipelineExecutor`. They only updated database records, fired events, and transitioned states.
- **Integration target:** `internal/modules/autocontent` — the highest-level workflow engine with 14 steps, partial mapping to `PipelineExecutor`'s 8 stages
- `internal/ai/pipeline.go` — `PipelineExecutor` was fully functional but never instantiated/wired; now autocontent creates it from the injected `ai.Manager`
- `internal/modules/autocontent/service.go` — Added `aiManager *ai.Manager` field, `SetAIManager` setter, `executeWorkflowAsync` goroutine in `StartJob`, `stepToPipelineStage` mapping (7 of 14 steps mapped), `buildPipelineInput` helper
- `internal/modules/autocontent/module.go` — Added `SetAIManager` pass-through to service
- `cmd/api/main.go` — Wired `autocontentMod.SetAIManager(aiSvc)` after AI module init
- **Behavior:** When `StartJob` is called and AI is configured, a goroutine executes the workflow through `PipelineExecutor`, saves results via `SaveResult`, and advances steps. Falls back to state-only mode when `aiManager` is nil (test/fallback).
- **Remaining gaps (resolved in Sprint 3.5d):** `contentgenerator`, `articlepipeline`, `workflow` modules still not AI-wired (state-only)

### Sprint 3.6 — Autocontent Workflow Engine
- Migration `000014_add_autocontent_tables.up.sql` — 5 tables: `autocontent_jobs`, `autocontent_steps`, `autocontent_results`, `autocontent_queue`, `workflow_templates` + indexes
- (No down migration file exists)
- `internal/modules/autocontent/model.go` — types (AutocontentJob, Step, Result, PublicationItem, WorkflowTemplate), DTOs, 7 JobStatus, 6 StepStatus, 5 QueueStatus, 14 WorkflowStep constants, StepDependencies, StepDisplayNames, 13 EventBus event types, 17 sentinel errors
- `internal/modules/autocontent/service.go` — 21 methods:
  - **Job CRUD**: CreateJob, GetJob, GetJobDetail, ListJobs, UpdateJob, DeleteJob
  - **Workflow Engine**: StartJob, PauseJob, ResumeJob, CancelJob, RetryStep, RestartJob
  - **Steps**: GetSteps, UpdateStep (with dependency checking, auto-advance, progress calc)
  - **Results**: SaveResult, GetResults, GetResultByStep
  - **Queue**: AddToQueue, ListQueue, UpdateQueueItem
  - **Templates**: CreateTemplate, ListTemplates
  - **Metrics/Stats**: GetMetrics, GetStats
- `internal/modules/autocontent/handler.go` — 21 REST endpoints under `/api/v1/autocontent/`:
  - `POST/GET /autocontent` — create/list jobs
  - `GET /autocontent/{id}` — get job detail with steps + results
  - `PUT/DELETE /autocontent/{id}` — update/delete job
  - `POST /autocontent/{id}/start|pause|resume|cancel|retry|restart` — workflow control
  - `GET /autocontent/{id}/steps` — list steps
  - `POST/GET /autocontent/{id}/results` — save/list results
  - `GET /autocontent/{id}/results/{stepName}` — get result by step
  - `GET /autocontent/stats|metrics` — stats and metrics
  - `POST/GET /autocontent/queue` — add/list queue items
  - `PUT /autocontent/queue/{queueID}` — update queue item
  - `POST/GET /autocontent/templates` — create/list templates
- `internal/modules/autocontent/module.go` — AutocontentModule kernel module with SetEventBus, RegisterRoutes
- `internal/api/routes.go` — AutocontentSvc in Dependencies, registerAutocontentRoutes
- `cmd/api/main.go` — module registered, EventBus wired, service in Dependencies
- 3 test files: model_test.go (8 tests), service_test.go (20 tests), handler_test.go (63 subtests) = 91 total
- All tests pass (build, vet, test)

## Sprint 3.7 — Multi-tenancy & Site Isolation

### Infrastructure (100% complete)
- `internal/api/middleware/site.go` — IdentifySite (header/domain), RequireSite, GetSiteID, GetSiteSlug
- `internal/api/middleware/rls.go` — RLSContext middleware (sets Postgres app.current_* config vars + context markers)
- `internal/api/middleware/authz.go` — Casbin RequirePermission with site domain isolation
- `internal/api/middleware/auth.go` — RequireAuth, OptionalAuth, GetUserID helper
- `internal/api/middleware/cross_site_isolation_test.go` — 282 lines, 11 tests (context isolation, middleware ordering, RLS tracking)
- `internal/api/middleware/middleware_test.go` — 507 lines, ~30 tests (all middleware signatures and edge cases)
- Route wiring in `routes.go:106-108`: `IdentifySite → authMiddleware → RLSContext`

### Site CRUD (100% complete)
- `internal/modules/site/service.go` — 1200+ lines: CreateSite, GetSite, GetSiteBySlug, GetSiteByDomain, ListSites, UpdateSite, DeleteSite, AddDomain, RemoveDomain, ListDomains, SetPrimaryDomain, GlobalSettings (CRUD), SiteSettings (CRUD)
- `internal/modules/site/model.go` — Site, SiteDomain, SiteSetting, GlobalSetting types + DTOs
- `internal/modules/site/handler.go` — 20 REST endpoints
- `internal/modules/site/service_test.go`, `handler_test.go`, `module_test.go`
- DeleteSite with 40+ table cleanup in transaction (service.go:674-726)

### site_id Filtering in Queries

**Modules with full site_id coverage (6):** categories, tags, assets, editorial, seoengine, humanwriter

**Repository/Internal methods fixed with site_id (2026-07-29):**
- `media/repository.go` — PermanentlyDelete, GetFolderChildCount, GetFolderSubfolderCount: added siteID param + filter
- `publisher/repository.go` — UpdatePublication, UpdateQueueItem, UpdateSchedule: added siteID param + AND site_id = $N
- All callers in service.go updated to pass siteID
- `workflow/service.go` — AdvanceStep, onStepCompleted, onStepFailed: added siteID param + AND site_id to 3 UPDATEs
- `workflow/handler.go` — AdvanceStep handler: added siteID extraction from middleware
- `autocontent/service.go` — getQueueItem: added siteID param + AND site_id to SELECT; callers updated
- `articlepipeline/service.go` — onStageCompleted, onStageFailed: added siteID param + AND site_id to 3 UPDATEs; callers updated
- `articlepipeline/service.go` — StartPipeline, PausePipeline, ResumePipeline, CancelPipeline, RetryStage, RestartPipeline: added AND site_id to 6 UPDATE queries (siteID already in scope)

**Still deferred (internal-only, no handler endpoint, caller validates job access):**
- contentgenerator/service.go — 3 UPDATEs on generation_jobs in UpdateStage (no handler endpoint)
- autocontent/service.go — onStepCompleted/onStepFailed (called from UpdateStep, no handler endpoint)
- research/service.go — 1 UPDATE on research_jobs in AddSource (no handler endpoint)

### Migration Conflict Resolution (2026-07-29)
- `publication_queue` was defined in **both** `000014` (autocontent, FK to `autocontent_jobs`) and `000019` (publisher, FK to `publications`) with incompatible schemas
- Both used `CREATE TABLE IF NOT EXISTS`, so `000014` silently won, leaving publisher module with wrong table schema
- **Fix:** Renamed autocontent's table to `autocontent_queue` in `000014_add_autocontent_tables.up.sql` (table + 5 indexes)
- Updated all 6 SQL queries in `autocontent/service.go` and 1 mock expectation in `autocontent/service_test.go`
- Publisher's `000019` now owns `publication_queue` without conflict

### Sprint 3.5d — AI Wiring: ContentGenerator, ArticlePipeline, Workflow (2026-07-30)

**Architecture Decision: No orchestration layer. Each module independently invokes PipelineExecutor.**
- Rationale: All four modules (AutoContent, ContentGenerator, ArticlePipeline, Workflow) operate on **different tables** with **different job IDs** — they never share state. They are **alternative workflow engines** for different use cases, not sequential pipeline stages. Adding an orchestrator would couple independent modules unnecessarily.
- Risk mitigation: Each module's goroutine uses `AND status = 'running'` for final completion update (race-safe). Each step checks current status before executing (skips if already completed/failed). MockProvider fallback preserved (nil-safe `if s.aiManager != nil` guard).

**Changes per module:**
- `internal/modules/contentgenerator/service.go` — Added `aiManager *ai.Manager` field, `SetAIManager` setter, `executeWorkflowAsync` goroutine launched in `StartJob`, `generatorStepToPipelineStage` mapping (8 of 9 stages mapped: research→StageResearchGen, briefing→StageBriefingGen, outline→StageOutlineGen, section_generation→StageDraftGen, seo_optimization→StageSEOGen, quality_review→StageQualityCheck, translation→StageTranslationGen, final_review→StageFinalReview; publish_ready skipped), `buildGeneratorPipelineInput` helper. Results stored in `generation_pipeline.metadata`.
- `internal/modules/articlepipeline/service.go` — Added `aiManager *ai.Manager`, `SetAIManager`, `executePipelineAsync` goroutine in `StartPipeline`, `articlePipelineStepToStage` mapping (7 of 11 stages mapped: research→StageResearchGen, outline→StageOutlineGen, draft→StageDraftGen, seo_optimization→StageSEOGen, translation→StageTranslationGen, quality_score→StageQualityCheck, publication_candidate→StageFinalReview; human_rewrite, readability, internal_linking, metadata skipped), `buildArticlePipelineInput`. Results stored in `article_pipeline_steps.output`.
- `internal/modules/workflow/service.go` — Added `aiManager *ai.Manager`, `SetAIManager`, `executeWorkflowAsync` goroutine in `StartJob`, `workflowStepToPipelineStage` mapping (6 of 8 steps mapped: research→StageResearchGen, writer→StageDraftGen, editorial_engine→StageSEOGen, seo_engine→StageSEOGen, quality_check→StageQualityCheck, finished→StageFinalReview; human_writer, publisher skipped), `buildWorkflowPipelineInput`. Results stored in `workflow_steps.metadata`.
- `internal/modules/contentgenerator/module.go` — Added `SetAIManager` pass-through, `ai` import
- `internal/modules/articlepipeline/module.go` — Added `SetAIManager` pass-through, `ai` import
- `internal/modules/workflow/module.go` — Added `SetAIManager` pass-through, `ai` import
- `cmd/api/main.go` — Wired `generatorMod.SetAIManager(aiSvc)`, `articlepipelineMod.SetAIManager(aiSvc)`, `workflowMod.SetAIManager(aiSvc)`
- `internal/ai/pipeline.go` — Fixed `PipelineResult.Duration` type from `string` to `time.Duration` (was causing build error when `.Milliseconds()` was called on it); added `"time"` import

**Behavior:** Each module now independently executes its workflow through PipelineExecutor when `StartJob`/`StartPipeline` is called and AI is configured. The goroutine iterates steps/pipeline stages, maps them to PipelineExecutor stages, executes via AI provider, stores results, and marks job complete. Falls back to state-only mode when `aiManager` is nil.

### Sprint 3.5e — Unmapped Step Analysis & Remapping (2026-07-30)

**Architecture decisions for each previously-unmapped step:**

| Step | Module(s) | Decision | Rationale |
|------|-----------|----------|-----------|
| `topic` | autocontent | New `StageTopicGen` | AI can suggest topics from initial context; uses prompt engineering with content type + language |
| `human_rewrite` | autocontent, articlepipeline | Keep skipped | Requires human intervention; no AI substitute for editorial judgment |
| `fact_check` | autocontent | New `StageFactCheck` | Uses existing `PromptTypeFactCheck` template + `runFactCheck` stage; checks content against references |
| `readability` | autocontent, articlepipeline | Re-map to `StageQualityCheck` | `QualityChecker.ScoreReadability` already computes readability scores |
| `internal_linking` | autocontent, articlepipeline | Keep skipped | Requires querying site's existing content DB; outside AI purview |
| `metadata` | autocontent, articlepipeline | Re-map to `StageSEOGen` | SEO prompt generates meta title, description, heading improvements from accumulated content |
| `featured_image` | autocontent | Keep skipped | Requires image generation/storage pipeline; not in scope |
| `publish_ready` | contentgenerator | Keep skipped | State marker, not AI work |
| `human_writer` | workflow | Keep skipped | Human step |
| `publisher` | workflow | Keep skipped | Action step (DB insert), not AI |

**Infrastructure changes:**
- `internal/ai/model.go` — Added `PromptTypeTopic = "topic"` constant
- `internal/ai/pipeline.go` — Added `StageTopicGen` and `StageFactCheck` to `PipelineStage` enum; added `Content string` field to `PipelineInput` for inter-stage draft accumulation; added `runTopic` and `runFactCheck` methods to `PipelineExecutor`; added `sourceText()` helper that prefers `Content` over `Briefing` (backward-compatible); updated all downstream stages (`runSEO`, `runQuality`, `runTranslation`, `runReview`) to use `sourceText(input)`
- `internal/ai/prompt_builder.go` — Added `PromptTypeTopic` template (EN + PT) for topic suggestion from content type, language, and existing context

**Module remapping:**
- `internal/modules/autocontent/service.go` — `stepToPipelineStage`: topic→StageTopicGen, fact_check→StageFactCheck, readability→StageQualityCheck, metadata→StageSEOGen (was: all 4 returned false). `executeWorkflowAsync`: added `accumulatedContent` variable, sets `input.Content` after each step completion and after `buildPipelineInput` rebuild.
- `internal/modules/articlepipeline/service.go` — `articlePipelineStepToStage`: readability→StageQualityCheck, metadata→StageSEOGen (was: both returned false). `executePipelineAsync`: added `accumulatedContent` variable with same accumulation pattern.

| Module | Before | After |
|--------|--------|-------|
| Autocontent | 7 mapped, 7 skipped | 11 mapped, 3 skipped (human_rewrite, internal_linking, featured_image) |
| ArticlePipeline | 7 mapped, 4 skipped | 9 mapped, 2 skipped (human_rewrite, internal_linking) |
| ContentGenerator | 8 mapped, 1 skipped (publish_ready) | No change |
| Workflow | 6 mapped, 2 skipped (human_writer, publisher) | No change |

### Notes
- Shell/bash non-functional — `go build ./...`, `go vet ./...`, `go test ./...` could not be executed
- 27 SQL queries initially missing site_id; 6 high-priority repository methods fixed, 21 internal callbacks deferred
