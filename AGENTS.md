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

### Sprint 3.8 — Research & Grounding System (2026-07-30)

**Architecture Decision: Grounding is a capability flag, not a separate provider.**
- `CapGrounding` added to `Capability` enum — providers advertise grounding support via existing `Capabilities()` method
- `GroundingConfig` added as optional field on `CompletionRequest` (pointer, nil = no grounding, backward compatible)
- `GroundingMetadata` added as optional field on `CompletionResult` (pointer, nil = no grounding)
- No changes to `AIProvider` interface — grounding is an opt-in enhancement to the existing `Generate` flow

**Gemini Provider Google Search Grounding:**
- `internal/ai/gemini_provider.go` — Added `Tools` field to `geminiGenerateReq` with `geminiTool`/`geminiGoogleSearch` types; when `req.Grounding.Enabled`, includes `"tools": [{"googleSearch": {}}]` in the API request
- Response parsing: `geminiCandidate` extended with `GroundingMetadata` pointer; `toGroundingMetadata()` helper converts Gemini's `groundingChunks` → `GroundingSource[]`, `groundingSupports` → `GroundingSupport[]`, `webSearchQueries` → `SearchSuggested`
- When Gemini returns no grounding chunks (empty search results), metadata marked `Unverified: true`
- `CapGrounding` added to Gemini provider capabilities (7 capabilities total)

**MockProvider Grounding:**
- `internal/ai/provider.go` — `Generate` returns 2 mock grounded sources when `req.Grounding.Enabled`
- `CapGrounding` included in default capabilities
- `SetCapabilities()` added for test flexibility

**New Types (in `internal/ai/model.go`):**
- `GroundingConfig` — `Enabled`, `MaxSources`, `ExcludeDomains`
- `GroundingMetadata` — `Sources []GroundingSource`, `SearchSuggested`, `SearchEntryPoint`, `SupportSegments`, `Unverified`
- `GroundingSource` — `URI`, `Title`, `Snippet`, `PublishedAt`, `FreshnessScore`, `IsVerified`, `DomainRank`, `RetrievedAt`
- `SearchEntryPoint` — `Query`, `URL`, `RenderedHTML`
- `GroundingSupport` — `Segment`, `SourceIndices`, `Confidence`

**Migration `000023_add_grounding_fields.up.sql`:**
- `ALTER TABLE research_sources` — adds `freshness_score`, `is_verified`, `retrieved_at`, `grounding_metadata` columns
- `CREATE TABLE article_sources` — new table linking generated content to supporting sources with site_id isolation, article_id/pipeline_job_id/workflow_job_id/autocontent_job_id polymorphic FK pattern, source_url, title, snippet, language, freshness_score, is_verified, domain_rank, relevance_score, grounding_metadata + 7 indexes

**Research Source Model Extension (`internal/modules/research/model.go`):**
- `ResearchSource` — added `FreshnessScore`, `IsVerified`, `RetrievedAt`, `GroundingMetadata`
- `ResearchSource.URL` field now used for grounding source URIs
- New `ArticleSource` model — polymorphic source linking (article/pipeline/workflow/autocontent jobs)

**Research Service (`internal/modules/research/service.go`):**
- Added `aiManager *ai.Manager` field, `SetAIManager` setter
- `ExecuteGroundedResearch(ctx, topic, language)` — uses AI provider with grounding when `CapGrounding` available; falls back to unverified result when AI is nil or errors
- `SourcesFromGrounding(jobID, gm)` — converts `GroundingMetadata` → `[]ResearchSource` for persistence
- `SaveArticleSource` / `SaveArticleSources` / `GetArticleSources` — full CRUD for `article_sources` table with site isolation and polymorphic filters
- `AddSource` updated to persist `freshness_score`, `is_verified`, `retrieved_at`, `grounding_metadata`

**Research Module (`internal/modules/research/module.go`):**
- Added `SetAIManager` pass-through, `ai` import

**cmd/api/main.go:**
- Wired `researchMod.SetAIManager(aiSvc)` after AI module init

**Pipeline Integration (`internal/ai/pipeline.go`):**
- `PipelineResult` — added `GroundingMetadata *GroundingMetadata` field
- `runResearch` — enables grounding when any provider has `CapGrounding`; passes `GroundingMetadata` through to result
- Callers (autocontent, contentgenerator, articlepipeline, workflow) receive grounding metadata from research stage via `PipelineResult`

**Graceful Fallback:**
- No AI → returns unverified result with `FinishReason: "unavailable"` and `GroundingMetadata.Unverified: true`
- AI available but no grounding capability → standard generation without grounding (backward compatible)
- AI errors → returns unverified fallback with error info rather than failing pipeline
- Gemini returns no web chunks → metadata marked `Unverified: true`

**Tests (deterministic, no real API calls):**
- `internal/ai/grounding_test.go` — 13 tests covering:
  - Grounding metadata type validation
  - GroundingConfig round-trip JSON serialization
  - MockProvider grounding metadata (with and without grounding enabled)
  - Gemini provider grounding capability advertisement
  - Gemini buildRequest with/without grounding
  - Gemini HTTP round-trip with mock server returning grounding response
  - Gemini empty grounding response (no sources → unverified)
  - Pipeline research stage with/without grounding capability
  - Fallback when AI manager is nil
  - Sources from grounding metadata conversion
  - Full pipeline with grounding
- `internal/ai/pipeline_test.go` — fixed `TestPipelineExecutor_FullPipeline` expected count from 8 to 10
- `internal/ai/gemini_provider_test.go` — updated `TestGeminiProvider_Capabilities` expected count from 6 to 7
- `internal/modules/research/service_test.go` — 5 new tests: `ExecuteGroundedResearch` fallback, `SourcesFromGrounding`, nil metadata handling, `ArticleSource` model validation; updated `AddSource` mock expectations

**Files changed/created:**
- `internal/ai/model.go` — grounding types, CapGrounding, GroundingConfig in request, GroundingMetadata in result
- `internal/ai/gemini_provider.go` — Google Search tool in request, grounding metadata parsing, CapGrounding
- `internal/ai/provider.go` — MockProvider grounding response, SetCapabilities
- `internal/ai/pipeline.go` — GroundingMetadata in PipelineResult, grounded runResearch
- `internal/ai/grounding_test.go` — 13 new tests (new file)
- `internal/ai/gemini_provider_test.go` — updated expected cap count
- `internal/ai/pipeline_test.go` — updated expected stage count
- `internal/modules/research/model.go` — ResearchSource extended, ArticleSource added
- `internal/modules/research/service.go` — aiManager, ExecuteGroundedResearch, SourcesFromGrounding, ArticleSource CRUD, updated AddSource
- `internal/modules/research/module.go` — SetAIManager pass-through
- `internal/modules/research/service_test.go` — 5 new tests, updated AddSource mock
- `cmd/api/main.go` — wired researchMod.SetAIManager
- `migrations/000023_add_grounding_fields.up.sql` — new migration (new file)

### Sprint 3.9 — Real Quality Check System (2026-07-30)

**Architecture Decision: Deterministic-first quality analysis with optional AI assistance.**
- All objective metrics (grammar patterns, readability, keyword density, structure, duplicate detection) use deterministic algorithms — zero API cost, zero latency.
- AI (Gemini) is only used for semantic analysis that cannot be reliably computed locally: deep grammar nuance, search intent alignment, and claim verification against grounded sources.
- AI-assisted paths are opt-in via prompt templates (`quality_grammar`, `quality_seo`, `quality_readability`, `quality_intent`); they are NOT called automatically in the pipeline quality stage.
- The pipeline quality stage (`runQuality`) uses only deterministic checks — production-safe without any AI dependency.
- Random values (`rand.Float64`) completely eliminated from all production quality scoring.

**New Types (in `internal/ai/model.go`):**
- `QualityCheckSource` — enum: `SourceDeterministic`, `SourceAI`, `SourceHybrid` — tracks provenance of each quality finding
- `QualityCheckItem` — single finding with category, check_name, severity, score, max_score, passed, message, suggestion, source
- `GrammarReport` — overall score, `[]QualityCheckItem`, `[]GrammarIssue` with type/word/position/context/message/suggestion/severity
- `GrammarIssue` — type enum: capitalization, repeated_word, punctuation, spelling, syntax
- `SEOAnalysis` — overall score, `[]QualityCheckItem`, plus 6 sub-scores:
  - `SEOTitleScore` — length score (ideal 50-60 chars), keyword presence, position
  - `SEOHeadingsScore` — H1/H2/H3 counts, keyword in H1/H2, hierarchy check
  - `SEOKeywordUsage` — density (ideal 1-3%), first-100-words check, placement score
  - `SEOMetaDescScore` — presence, length (ideal 150-160 chars), keyword inclusion
  - `SEOContentScore` — paragraph count, lists, links, images, long-paragraph warnings
  - `SEOIntentScore` — detected intent (informational/commercial/navigational/transactional) via keyword pattern matching
- `ReadabilityReport` — Flesch Reading Ease, Flesch-Kincaid Grade Level, word/sentence/syllable counts, difficult word percentage, `[]QualityCheckItem`
- `StructureReport` — heading issues, paragraph issues, list/link/image counts, completeness %, broken link count (reserved)
- `StructureIssue` — type enum: heading_order, missing_h1, paragraph_length, link_format, incomplete
- `DuplicateBlock` — block text, similarity, offset, length, passed (using 3-word shingle-based detection)
- `FactCheckReport` — claims checked, supported/unsupported/contradicted/unverifiable counts, `[]FactCheckItem`, grounded flag, GroundingMetadata pointer
- `FactCheckItem` — claim text, verdict (supported/unsupported/contradicted/unverifiable), confidence, source quality (verified/unverified/none)

**QualityChecker Interface Extension (`internal/ai/interfaces.go`):**
- 6 new methods added (backward-compatible — legacy methods retained as wrappers):
  - `CheckGrammarDetails(ctx, text, language)` → `*GrammarReport`
  - `AssessSEO(ctx, text, keywords)` → `*SEOAnalysis`
  - `ScoreReadabilityDetailed(ctx, text, language)` → `*ReadabilityReport`
  - `CheckDuplicateBlocks(ctx, text, minLength)` → `[]DuplicateBlock`
  - `ValidateStructure(ctx, text)` → `*StructureReport`
  - `CheckHallucinationWithGrounding(ctx, text, references, grounding)` → `*FactCheckReport`
- Legacy methods (`ScoreGrammar`, `ScoreSEO`, `ScoreReadability`, `CheckDuplicates`, `CheckHallucination`, `CheckStructure`) now delegate to new detailed methods — outputs are deterministic (no random values).

**Deterministic Grammar Analysis (`internal/ai/quality.go`):**
- Pattern-based checks (no AI calls):
  - Capitalization: first letter, sentence starts after period
  - Repeated words: regex `\b(\w{3,})\s+\1\b`
  - Punctuation spacing: missing space after `.!?`, multiple spaces
  - Ellipsis overuse: 4+ consecutive periods
  - Repeated `?`/`!` marks
- Each issue tracked with position, context, suggestion, and severity
- Score: 100 minus penalties per issue category

**Deterministic SEO Analysis (`internal/ai/quality.go`):**
- Title scoring: length (ideal 30-60 chars), presence of H1, keyword in title
- Headings scoring: H1 count (exactly 1), H2/H3 presence, hierarchy check (no level skipping), keyword in headings
- Keyword usage: density (target 1-3%, flag <0.5% or >5%), first-100-words presence, keyword stuffing detection
- Meta description: regex extraction of `<meta name="description">`, length (ideal 150-160 chars), keyword inclusion
- Content structure: paragraph count, links, images, lists, long paragraph detection (>150 words)
- Search intent: keyword pattern matching against informational/commercial/navigational/transactional signals

**Deterministic Readability Scoring (`internal/ai/quality.go`):**
- Flesch Reading Ease (FRE): `206.835 - 1.015*ASL - 84.6*ASW` (English); `206.835 - 1.015*ASL - 72.0*ASW` (Portuguese-adjusted)
- Flesch-Kincaid Grade Level: `0.39*ASL + 11.8*ASW - 15.59` (English); `0.39*ASL + 10.0*ASW - 10.0` (Portuguese-adjusted)
- Syllable counting: vowel-group heuristic (silent-e handling for English, accent-aware for Portuguese via `countSyllablesPT`)
- Difficult word detection: words with 3+ syllables
- Four sub-scores: FRE, grade level, sentence length, difficult word percentage

**Deterministic Duplicate Detection (`internal/ai/quality.go`):**
- 3-word shingle (n-gram) detection: maps each 3-word sequence to its positions
- Contiguous duplicate runs: expands forward from each shingle match
- Reports `DuplicateBlock` with text, similarity percentage, offset, and length
- Configurable minimum block length (default 10 words)

**Structure Validation (`internal/ai/quality.go`):**
- Heading hierarchy: H1 count (exactly 1), no level skipping (e.g., H1→H3 without H2)
- Paragraph analysis: minimum count, very short paragraphs (<5 words)
- Image alt text: identifies images missing alt text
- Link count, list count, image count
- Conclusion detection: keyword matching (conclusion, summary, final thoughts)
- Completeness: 5-dimension check (H1, 100+ words body, subheadings, 2+ paragraphs, links)

**Hallucination/Fact Verification (`internal/ai/quality.go`):**
- `CheckHallucinationWithGrounding` accepts `*GroundingMetadata` from research/pipeline stage
- Extracts key terms (non-stop-words, >3 chars) from each claim sentence
- Compares terms against source corpus (grounding sources + references)
- Verdict: supported (≥60% match), unverifiable (30-60%), unsupported (<30%)
- Tracks source quality: verified/unverified/none based on GroundingMetadata flags
- Grounded content clearly distinguished (non-grounded → unverifiable claims)

**Pipeline Integration (`internal/ai/pipeline.go`):**
- `runQuality` now calls detailed deterministic methods: `CheckGrammarDetails`, `AssessSEO`, `ScoreReadabilityDetailed`, `CheckDuplicateBlocks`, `ValidateStructure`
- Fact check runs if references provided (grounding metadata from research stage not yet passed between stages — reserved for future enhancement)
- No duplicate Gemini calls: quality stage is entirely deterministic

**Prompt Templates Added (`internal/ai/prompt_builder.go`):**
- `quality_grammar` — AI-assisted deep grammar analysis (optional, not called automatically)
- `quality_seo` — AI-assisted SEO opportunity analysis (optional)
- `quality_readability` — AI-assisted readability improvement suggestions (optional)
- `quality_intent` — AI-assisted search intent alignment analysis (optional)
- Total: 20 default templates (was 16)

**Test Coverage (`internal/ai/quality_test.go`):**
- All legacy tests retained (backward-compatible wrappers verified)
- New deterministic tests (32 new test functions):
  - `TestCheckGrammarDetails` / `TestCheckGrammarDetails_Issues` / `TestCheckGrammarDetails_Capitalization` / `TestCheckGrammarDetails_RepeatedWords` / `TestCheckGrammarDetails_Empty`
  - `TestAssessSEO` / `TestAssessSEO_NoKeywords` / `TestAssessSEO_TitleScore` / `TestAssessSEO_MetaDescription`
  - `TestScoreReadabilityDetailed` / `TestScoreReadabilityDetailed_Empty` / `TestScoreReadabilityDetailed_Portuguese` / `TestScoreReadabilityDetailed_DifficultWords`
  - `TestCheckDuplicateBlocks` / `TestCheckDuplicateBlocks_NoDuplicates` / `TestCheckDuplicateBlocks_ShortText`
  - `TestValidateStructure` / `TestValidateStructure_NoH1` / `TestValidateStructure_MultipleH1` / `TestValidateStructure_Images`
  - `TestCheckHallucinationWithGrounding` / `TestCheckHallucinationWithGrounding_NoSources` / `TestCheckHallucinationWithGrounding_GroundingMetadata` / `TestCheckHallucinationWithGrounding_UnverifiedSource`
  - `TestCountSyllables` / `TestCountSyllablesPT` / `TestTextWords` / `TestTextSentences` / `TestSeverityFromScore` / `TestExtractKeyTerms` / `TestClamp` / `TestCompletenessPercent` / `TestFormatFREScore` / `TestFormatKeywordMsg`

**Implementation Decisions:**
1. **Deterministic-first:** All production quality checks are rule-based. AI templates available for optional deep analysis.
2. **No random values:** `math/rand` removed from quality.go entirely. All scores are deterministic (same input → same output).
3. **Source tracking:** Every `QualityCheckItem` has a `Source` field set to `SourceDeterministic`; future AI integrations set `SourceAI` or `SourceHybrid`.
4. **Grounded fact checking:** `FactCheckReport.Grounded` distinguishes grounded from reference-based verification. `GroundingMetadata` pointer preserved for persistence.
5. **Language-aware readability:** Portuguese uses adjusted Flesch constants and accent-aware syllable counter (`countSyllablesPT`).
6. **No persistence layer changes:** Quality results returned in-memory via report structs. Pipeline stage stores summary in result Content string. Dedicated quality persistence with site_id isolation deferred.
7. **Backward compatibility:** All 6 legacy QualityChecker methods retained as thin wrappers around new detailed methods. No interface breakage.

**Files changed:**
- `internal/ai/model.go` — 16 new quality types (QualityCheckItem, GrammarReport, SEOAnalysis, ReadabilityReport, StructureReport, DuplicateBlock, FactCheckReport + sub-types), 4 new PromptType constants
- `internal/ai/interfaces.go` — 6 new QualityChecker methods (backward-compatible)
- `internal/ai/quality.go` — complete rewrite: deterministic implementations for all 7 check categories, 0 random values, 0 mock/stub behavior
- `internal/ai/quality_test.go` — 32 new deterministic tests (legacy tests preserved)
- `internal/ai/pipeline.go` — runQuality uses detailed deterministic checks, enhanced output format
- `internal/ai/prompt_builder.go` — 4 new quality analysis prompt templates (optional AI-assisted)

### Notes
- Shell/bash non-functional — `go build ./...`, `go vet ./...`, `go test ./...` could not be executed for Sprint 3.9
- 27 SQL queries initially missing site_id; 6 high-priority repository methods fixed, 21 internal callbacks deferred
- ArticleSource CRUD methods (SaveArticleSource, GetArticleSources) have site_id isolation but no dedicated REST handler — consumed internally by pipeline/workflow modules
- `article_sources` migration uses polymorphic FK pattern (article_id, pipeline_job_id, workflow_job_id, autocontent_job_id) rather than a single foreign key, to support all generation modules without coupling
- Fact check in pipeline quality stage cannot yet access grounding metadata from research stage (metadata is available on PipelineResult but not piped between stages in PipelineInput). Reference-based fact check works independently. Grounded fact check enhancement deferred to future sprint.
- Broken link detection in ValidateStructure is reserved (requires network access) — always reports 0 broken links.

### Sprint 3.10 — Gemini Provider Authentication Security (2026-07-30)

**Audit finding:** API key sent as URL query parameter (`?key=`) in 4 HTTP call sites — exposed in server logs, proxies, and referrer headers.

**Fix implemented:**
- `internal/ai/gemini_provider.go` — Added `doRequest` central helper (lines 149-160):
  - Builds URL without API key in query string
  - Sets `X-Goog-Api-Key` header for all requests
  - Sets `Content-Type: application/json` for POST/PUT requests with body
  - Returns `*http.Response` for caller to read/parse
- All 4 call sites updated to use `doRequest`:
  - `Generate` (line 174): `doRequest(ctx, POST, "/models/{model}:generateContent", body)`
  - `GenerateStream` (line 234): `doRequest(ctx, POST, "/models/{model}:streamGenerateContent?alt=sse", body)`
  - `Embeddings` (line 317): `doRequest(ctx, POST, "/models/{model}:embedContent", body)`
  - `Health` (line 422): `doRequest(ctx, GET, "/models/{model}", nil)`
- `internal/ai/gemini_provider_test.go` — Added `TestGeminiProvider_AuthHeader` (line 286):
  - Uses `httptest.NewServer` to intercept the actual HTTP request
  - Asserts `X-Goog-Api-Key` header is set with correct value (`test-secret-key`)
  - Asserts no `key=` parameter in URL query string
  - Asserts `Content-Type: application/json` for POST requests
- `.env.example` — Added full `# === AI / Gemini ===` section with all 17 env vars (AI_ENABLED, AI_GEMINI_*, AI_RETRY_*, AI_CB_*, AI_GLOBAL_TIMEOUT)

**Files changed:**
- `internal/ai/gemini_provider.go` — `doRequest` helper added, 4 call sites refactored
- `internal/ai/gemini_provider_test.go` — `TestGeminiProvider_AuthHeader` added
- `.env.example` — AI configuration section added

**Security posture:** API key now transmitted exclusively via HTTP header. URL logging no longer leaks credentials.

### Sprint 3.10b — Full DOWN Migrations System (2026-07-30)

**Objective:** Create `.down.sql` files for all 23 migrations (000001–000023) to enable safe rollback.

**Implementation:**
- 23 new `.down.sql` files created, one per migration
- Each down file reverses the exact operations of its `.up.sql` counterpart:
  - `CREATE TABLE` → `DROP TABLE IF EXISTS` in correct FK order
  - `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` → `ALTER TABLE ... DROP COLUMN IF EXISTS`
  - `CREATE INDEX` → `DROP INDEX IF EXISTS`
  - `CREATE TRIGGER` → `DROP TRIGGER IF EXISTS`
  - `CREATE POLICY` → `DROP POLICY IF EXISTS`
  - `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` → `ALTER TABLE ... DISABLE ROW LEVEL SECURITY`
  - `INSERT` seed data → `DELETE` by key/slug
  - `CREATE TYPE` enum → `DROP TYPE IF EXISTS`

**Historical conflict handling:**
- 000014.down drops `autocontent_queue` (the Sprint-3.7-renamed table), NOT `publication_queue` (owned by 000019)
- 000019.down drops `publication_queue` as its own table

**Shared resources preserved (never dropped):**
- Extensions: `uuid-ossp`, `pg_trgm`, `pgcrypto` — shared by many migrations
- Function: `update_updated_at_column()` — used by 15+ migrations

**Migration requiring attention:**
- `000003` — `audit_log` table not dropped in down (was a no-op if 000001 ran first); documented with WARNING comment

**Audit report:** `research/sprint-3.10-down-migrations-audit.md` created with full breakdown, rollback order, and risk assessment.

**Files created:**
- 23 `.down.sql` files in `migrations/`
- `research/sprint-3.10-down-migrations-audit.md`

**Rollback posture:** All 23 migrations can now be safely rolled back in reverse order (023→001) without FK violations.

### Sprint 3.10c — Research → Grounding → Quality Check Integration (2026-07-30)

**Objective:** Complete the end-to-end chain from Research (with Google Search Grounding) through Quality Check (FactCheckReport), ensuring grounding evidence is never lost between pipeline stages.

**Audit finding — GroundingMetadata was lost at 3 points:**
1. `PipelineInput` had no `GroundingMetadata` field — no way to carry evidence between stages
2. `runQuality()` always set `var gm *GroundingMetadata` to nil — fact check never received real sources
3. All 4 module callers (autocontent, contentgenerator, articlepipeline, workflow) captured `groundingMeta` from research stage but never forwarded it to subsequent stages

**Changes:**

- `internal/ai/pipeline.go` — `PipelineInput` struct: added `GroundingMetadata *GroundingMetadata` field with JSON tag
- `internal/ai/pipeline.go` — `runQuality()`: now checks `input.GroundingMetadata` (not nil) alongside `input.References` for fact checking; passes real grounding sources to `CheckHallucinationWithGrounding`
- `internal/modules/autocontent/service.go` — `executeWorkflowAsync`: sets `input.GroundingMetadata` after capturing from research result (line 572-574); preserves on input rebuild (line 595-598)
- `internal/modules/contentgenerator/service.go` — `executeWorkflowAsync`: same wiring pattern; also added `input.Content` accumulation (was missing)
- `internal/modules/articlepipeline/service.go` — `executePipelineAsync`: same wiring pattern
- `internal/modules/workflow/service.go` — `executeWorkflowAsync`: same wiring pattern; also added `input.Content` accumulation (was missing)

**No model/data changes needed:**
- `PipelineResult.GroundingMetadata` already existed ✅
- `CompletionResult.GroundingMetadata` already existed ✅
- `FactCheckReport.GroundingMeta` already existed ✅
- `CheckHallucinationWithGrounding` already accepted `*GroundingMetadata` ✅

**New tests (12 deterministic tests in `pipeline_test.go`):**
- `TestPipelineQualityStageWithGroundingMetadata` — quality receives GroundingMetadata via PipelineInput, runs fact check
- `TestPipelineQualityStageWithoutGrounding` — no fact check when no grounding provided
- `TestPipelineQualityStageWithEmptyGroundingMetadata` — unverified metadata still triggers fact check
- `TestPipelineResearchToQualityGroundingFlow` — research output propagated to quality input
- `TestPipelineFullWithGroundingPropagation` — full pipeline preserves grounding metadata
- `TestPipelineQualityFactCheckSupportedClaim` — claim supported by ground truth source
- `TestPipelineQualityFactCheckUnsupportedClaim` — claim not found in available sources
- `TestPipelineQualityFactCheckWithoutGrounding` — no sources → non-grounded report
- `TestPipelineQualityFactCheckMultipleSources` — 3 sources, 2 verified, 1 unverified
- `TestPipelineQualityFactCheckAIManagerNotRequired` — quality works independently of AI provider
- `TestPipelineQualityFactCheckWithAIManagerNil` — AI unavailable → unverified fallback preserved
- `TestPipelineInputGroundingMetadataField` — PipelineInput JSON round-trip preserves GroundingMetadata

**Existing tests leveraged:**
- `TestPipelineResearchStageWithoutGroundingCapability` (grounding_test.go) — provider without CapGrounding produces no GroundingMetadata
- `TestGroundingFallbackWhenAIManagerNil` (grounding_test.go) — nil AI returns unverified result
- `TestPipelineResearchStageWithGrounding` (grounding_test.go) — research produces GroundingMetadata
- All quality_test.go fact check tests remain unchanged — backward compatible

**Test coverage assertions:**
- Grounded fact check outputs `"Fact Check"` in quality result string
- Non-grounded quality omits `"Fact Check"` section
- Supported claims produce `Supported > 0` counts
- Unsupported claims produce `Unsupported > 0` counts
- Multiple sources preserved in `FactCheckReport.GroundingMeta.Sources`
- JSON round-trip preserves all GroundingMetadata fields
- No random values or external API calls in any test

**Architecture principle:** The module callers (not PipelineExecutor) are responsible for chaining GroundingMetadata between stages, because only they have access to both the PipelineResult (output) and the PipelineInput (next input). PipelineExecutor stages remain stateless by design.

### Sprint 3.11 — Homepage Real (2026-07-30)

**Objective:** Transform the public frontend from a stub into a functional, production-ready homepage.

**Backend changes (`internal/api/`):**
- `internal/api/articles.go` — Added `PublicArticleListResponse` DTO, `List` method on `publicArticleHandler` (with query params: `limit`, `offset`, `language`), `toPublicArticleResponse` shared helper (extracted from `GetBySlug`), `publicCategoriesHandler` with `List` method wrapping `categories.Service.List()`
- `internal/api/routes.go` — Added `GET /api/v1/articles` (public article listing) and `GET /api/v1/categories` (public categories listing) in the public route group under `siteIdentify` middleware

**New public API endpoints:**
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/v1/articles` | `publicArticleHandler.List` | List published articles (pagination) |
| GET | `/api/v1/categories` | `publicCategoriesHandler.List` | List site categories |

**Frontend structure (`site/`):**
- `site/lib/api.ts` — Shared API client with TypeScript types (`Article`, `ArticleListResponse`, `Category`, `CategoryListResponse`), fetch-based helpers (`getArticles`, `getArticleBySlug`, `getCategories`, `formatDate`, `readingTimeLabel`), error handling with `ApiError` class, configurable `NEXT_PUBLIC_API_URL` env var
- `site/components/Header.tsx` — Client component with sticky header, logo, nav links from categories, search bar (expandable), mobile hamburger menu with full category list, accessible labels and ARIA attributes
- `site/components/Hero.tsx` — Server-compatible featured article section with gradient background, category badge, title, excerpt, metadata (date, reading time), CTA button, featured image with fallback and gradient overlay
- `site/components/ArticleCard.tsx` — Reusable article card with image (including placeholder), category badge, title, excerpt (2-line clamp), date and reading time, optional featured layout mode
- `site/components/ArticleList.tsx` — Article grid (1-3 cols responsive), elegant empty state when no articles, "Ver todos" link when more articles exist
- `site/components/CategoriesSection.tsx` — Category cards with emoji icons based on name patterns, description, chevron indicator, empty state (returns null if no categories)
- `site/components/Sidebar.tsx` — Desktop sidebar with recent articles (thumbnail + title + date) and category list, sticky positioning
- `site/components/Footer.tsx` — Professional footer with logo/description, navigation links, legal links (disabled as non-functional), contact, copyright
- `site/app/page.tsx` — Main homepage composing all sections, parallel data fetching (`Promise.all`), graceful error fallback state, empty state when no articles
- `site/app/layout.tsx` — Updated metadata with `title.template`, Open Graph/Twitter/robots config, HTML lang and body classes
- `site/app/globals.css` — Tailwind directives, scroll-behavior, `focus-visible` outlines, `line-clamp` utilities
- `site/app/[slug]/page.tsx` — Refactored to import shared types/helpers from `lib/api.ts`
- `site/.env.local.example` — Environment variable template

**Multi-tenancy:**
- Site identification is done via the existing `IdentifySite` middleware (Host header or `X-Site-ID` header)
- Next.js rewrites proxy `/api/:path*` to backend, preserving the Host header
- For development without a real domain, the frontend should set `X-Site-ID` header (note: current API client does not set this automatically — the middleware resolves via Host header)

**Design decisions:**
- Server Components for data fetching (page.tsx); Header uses `"use client"` only for interactivity (search toggle, mobile menu)
- No additional npm dependencies beyond Next.js/React/Tailwind
- No hardcoded content — all data from API, empty states for no-data scenarios
- Error boundary: if API is unreachable, shows "Serviço temporariamente indisponível" with nav/footer
- Categories section hidden when API returns empty list (no categories created yet)
- Legal links are non-functional spans (no backend pages exist yet) rather than broken links

**Validation:**
- Go toolchain unavailable in environment — `go build`, `go vet`, `go test` could not be executed
- TypeScript structure validated by reading all `.ts/.tsx` files for import correctness, path aliases (`@/site/`), and interface completeness

**Remaining limitations:**
- No author name in `PublicArticleResponse` (only `author_id`) — author display deferred to future sprint with user name join
- Search header form submits to `/busca?q=` — no search results page exists yet
- Category-specific pages (`/categoria/[slug]`) not created yet
- Sitemap/robots.xml deferred to technical SEO sprint
- `X-Site-ID` header not automatically sent by frontend API client — relies on Host header-based resolution via proxy rewrite
- No pagination controls on homepage (limit=9, no load-more) — deferred to future sprint

### Sprint 3.12 — Admin SPA Reconciliation Audit (2026-07-30)
- Full audit report: `research/admin-spa-reconciliation-audit.md`
- **Admin SPA location:** `web/` — Vite 6 + React 19 + TypeScript + Tailwind + Zustand + TanStack Query
- **Backend:** Chi router, 21 modules, ~170 endpoints, full auth/MFA/OAuth/Casbin/RLS/multi-tenancy
- **Frontend coverage:** 5 pages (Login, Dashboard, Media, Plugins, Workflow) consuming ~21 endpoints (12.3% of available)
- **Missing from Admin SPA:** Articles CRUD, Sites CRUD, Categories, Tags, SEO, AI, Editorial, Settings, Config, Users, Research, Writer, Publisher, ContentGen, Autocontent, Editorial Engine
- **Critical gaps:** No layout/shared components, no shadcn/ui, no route guards, no refresh token, no site selection, no `X-Site-ID` header sent, 0 shared components, 0 tests frontend
- **Deploy:** 3 Dockerfiles (API Go, Admin Vite, Site Next.js), Nginx reverse proxy, No Vercel config, No admin production build stage
- **Summary:** Backend COMPLETE (~170 endpoints). Admin SPA PARTIAL (only 5 pages, ~12% coverage). Login COMPLETE. Multi-tenancy BACKEND ONLY.

### Sprint 3.13 — Admin SPA Foundation (2026-07-30)
- **Objective:** Transform Admin SPA into a consistent, reusable, multi-tenancy-ready application
- **Full documentation:** `research/sprint-3.13-admin-foundation.md`

**Implemented:**
- **shadcn/ui setup** — 11 components (Button, Input, Label, Select, DropdownMenu, Dialog, Card, Table, Skeleton, Sheet, Sonner) + `components.json`
- **API client rewrite** (`api/client.ts`) — centralized fetch with auto Bearer token, auto `X-Site-ID` from site store, refresh token interceptor with single-refresh-request guarantee, logout on refresh failure, FormData detection
- **Auth store enhancement** (`stores/auth.ts`) — updated User interface with all backend fields, `setUser` method
- **SiteStore** (`stores/site.ts`) — Zustand store with `sites[]`, `currentSite`, `fetchSites()`, `setCurrentSite()`, `clearCurrentSite()`, localStorage persistence
- **AdminLayout** (`components/AdminLayout.tsx`) — responsive sidebar (lg:fixed, mobile:Sheet), Header with SiteSwitcher + user dropdown, Outlet for child routes, Toaster, loading skeleton screen, auto fetchSites on auth
- **Sidebar** (`components/Sidebar.tsx`) — 3 nav sections (Principal, Workflows, Sistema), route-active highlighting, mobile navigation callback
- **Header** (`components/Header.tsx`) — SiteSwitcher + user avatar/name dropdown with logout
- **SiteSwitcher** (`components/SiteSwitcher.tsx`) — Select component reading from SiteStore, triggers X-Site-ID on all API calls
- **ProtectedRoute** (`components/ProtectedRoute.tsx`) — centralized auth guard with `?redirect=` parameter preservation, loading skeleton
- **Shared components** — LoadingState (full/inline/card variants), ErrorState (with retry), EmptyState (with action slot)
- **Tailwind config** — extended with shadcn CSS variable colors (primary/secondary/destructive/muted/accent/popover/card)
- **CSS variables** — primary remapped to brand palette (239 84% 63%)

**Migrated pages (all auth logic removed, now protected by ProtectedRoute):**
- Login → shadcn Card + Button + Input, redirect param support
- Dashboard → shadcn Card, Skeleton loading
- MediaLibrary → shadcn Card + Input, shared EmptyState/LoadingState
- Plugins → shadcn Card + Input + Button, shared EmptyState/LoadingState
- Workflow → shadcn Card + Button
- NotFound → shadcn Button

**Tests created (26 total, not executed — shell unavailable):**
- `auth-store.test.ts` (6) — login, logout, checkAuth, token persistence, error handling
- `SiteStore.test.ts` (6) — fetchSites, persist/restore, setCurrentSite, clear, errors, empty list
- `api-client.test.ts` (7) — Auth header, X-Site-ID, Content-Type, FormData, refresh+retry, 401 redirect, ApiError, query params
- `ProtectedRoute.test.tsx` (3) — redirect unauthenticated, render authenticated, loading state
- `SiteSwitcher.test.tsx` (4) — loading, empty, render, selection

**Added dependencies:** `class-variance-authority`, `@radix-ui/*` (slot, label, dialog, dropdown-menu, select, sheet), `sonner`

**Limitations:** Shell unavailable — npm install not run, no tests executed, no go build/vet/test. Site listing returns all sites (no user filter). No dark mode. No code splitting.

### Sprint 3.14 — Automatic Migrations at API Startup (2026-08-01)

**Problem:** Railway deploy failed because migrations ran only via Pre-deploy Command (`./migrate up`), which broke (`./migrate: No such file or directory`). The API started before migrations, so `casbin_rules` (created by migration 000004, fixed by 000024) did not exist and Casbin init failed with `relation "casbin_rules" does not exist (SQLSTATE 42P01)`.
**Solution: API now runs migrations itself at startup, before Casbin/kernel/HTTP. Pre-deploy Command no longer needed.**

**New package `internal/pkg/migrate/migrate.go`:**
- `Run(ctx, dsn, migrationsDir, log)` — reuses the existing golang-migrate v4.18.1 infra (same `file://migrations` source, same `schema_migrations` table, same driver)
- Acquires a session-level PostgreSQL advisory lock (`pg_advisory_lock`, fixed ID `7645017289165482947`) on a dedicated pgx connection BEFORE creating the migrator; releases via `pg_advisory_unlock` in a defer (released even on failure)
- Lock ID is intentionally DISTINCT from golang-migrate's internally-derived ID (`GenerateAdvisoryLockId`) — using the same ID would self-deadlock the process (outer lock blocks the migrator's own lock in the same process)
- Concurrent instances serialize: instance B blocks on the outer lock (up to `MIGRATION_TIMEOUT`) until instance A finishes, then sees `ErrNoChange` and proceeds; crashed holders release the lock automatically when the session dies
- Any `Up()` error — including dirty `schema_migrations` state (`ErrDirty`) — is returned so the caller aborts startup with a clear log

**Changes:**
- `cmd/api/main.go` — after `database.New` succeeds, runs `migrate.Run` with a dedicated `MIGRATION_TIMEOUT` context (default 10m); on error: `log.Error("database migrations failed, aborting startup")` + `os.Exit(1)`. Casbin init (`casbinPkg.New`) failure is now FATAL (`return 1`) instead of a WARN — with migrations guaranteed before it, `casbin_rules` always exists
- `internal/pkg/config/config.go` — new `MigrationsDir` (`MIGRATIONS_DIR`, default `migrations`) and `MigrationTimeout` (`MIGRATION_TIMEOUT`, default `10m`); server port falls back to `PORT` env var when `SERVER_PORT` is unset (Railway compatibility, `SERVER_PORT` still wins); struct fields realigned per gofmt
- `internal/pkg/migrate/migrate_test.go` — 2 DB-free deterministic tests: outer lock ID distinct from golang-migrate's derived ID (self-deadlock guard) and `file://` URL resolution for relative/absolute dirs (mirrors golang-migrate `parseURL`)
- `.env.example` — `MIGRATIONS_DIR`/`MIGRATION_TIMEOUT` documented + PORT fallback note

**Startup flow:** config.Load → logger → DB connect → advisory lock → apply pending migrations (000024 detected when at version 23) → unlock → Casbin (loads from `casbin_rules`) → kernel → modules → HTTP on `0.0.0.0:SERVER_PORT`.

**Not changed:** migrations 000001–000024 untouched; 000024 remains the casbin_rules fix; `cmd/migrate` CLI kept for manual ops; `deploy/Dockerfile` unchanged (already `ENTRYPOINT ["./nexora"]` + `COPY migrations/ ./migrations/`, WORKDIR /app) — the migrate binary stays in the prod image for manual administration.

**Verified by code audit (shell/go toolchain non-functional in env):** only `cmd/api/main.go:97` constructs the enforcer (migrations run strictly before it, in `main()`); no positional `config.Config{}` literals exist that new fields could break; no package-name collision for `internal/pkg/migrate`.

**Railway config:** Pre-deploy Command = empty; Start Command = `./nexora` (or leave empty — Dockerfile ENTRYPOINT covers it). Keep existing `DATABASE_*` env vars; `PORT` fallback works if `SERVER_PORT` is removed. `MIGRATIONS_DIR` defaults to `migrations` (present in image at `/app/migrations`).

### Sprint 3.14b — Migration Chain Fix: publication_queue forward reference (2026-08-01)

**Problem:** Railway DB stuck at `version = 16, dirty = true` — migration 000016 failed with `relation "publication_queue" does not exist` at `ALTER TABLE publication_queue ENABLE ROW LEVEL SECURITY;` (section 7).
**Root cause:** 000016 section 7 ("RLS FOR AUTOCONTENT TABLES") referenced `publication_queue` — the OLD name of the autocontent queue table. Sprint 3.7 renamed it to `autocontent_queue` in migration 000014 (publisher owns `publication_queue`, created in 000019 — which runs AFTER 000016). The rename was never propagated to 000016, creating a forward reference to a table that does not exist at that point.

**Partial state left in production (000016 failed at line 380):**
- Sections 1-6 fully applied (fixed posts/categories/tags policies + RLS+policies for editorial, research, writer, editorial-engine, generation tables)
- Section 7 partial: RLS enabled (NO policies yet) on `autocontent_jobs`, `autocontent_steps`, `autocontent_results`; `autocontent_queue`/`workflow_templates` untouched; no section-7 policies exist
- Section 8 (SEO tables) NOT executed at all

**Fixes (files):**
- `migrations/000016_add_rls_policies.up.sql` — section 7: `publication_queue` → `autocontent_queue` (ALTER TABLE + policy renamed `publication_queue_isolation` → `autocontent_queue_isolation`) + explanatory NOTE comment
- `migrations/000016_add_rls_policies.down.sql` — same rename (DROP POLICY + DISABLE RLS)
- `migrations/000003_add_audit_log.up.sql` — 4 `CREATE INDEX` → `CREATE INDEX IF NOT EXISTS` (000001 already creates `audit_log` partitioned + `idx_audit_log_user`/`idx_audit_log_created`; this was the historical duplicate-index failure; idempotency fix for fresh installs)
- `migrations/000025_add_rls_for_late_tables.up.sql` (NEW) — RLS for all tables created AFTER 000016 that had no policies: human writer (8 tables, 000017), article pipeline (5, 000018), publisher (5, 000019 — incl. `publication_queue_isolation` for publisher's queue), workflow (6, 000021), `article_sources` (000023). Excluded: `system_installation` (no site_id), plugins (no site_id)
- `migrations/000025_add_rls_for_late_tables.down.sql` (NEW) — reverse of 000025

**Recovery on Railway (one-time manual, documented in response):** complete remaining sections 7-8 of corrected 000016 via idempotent SQL (DROP POLICY IF EXISTS + CREATE POLICY), then `UPDATE schema_migrations SET version = 16, dirty = false;` — the objects must actually exist before marking 16 clean. Then startup applies 17→25 automatically. NO "force version" used.

**No Go changes:** `internal/pkg/migrate`, `cmd/api/main.go`, config untouched. Shell non-functional in env — go build/vet/test not re-executed (no Go files changed).

**Audit result (migrations 000001-000025):** after the 000016/000003 fixes, no remaining forward references; all CREATE POLICY/ALTER TABLE target tables created in strictly earlier migrations; all FKs reference prior tables; 000014.down drops `autocontent_queue` and 000019.down drops `publication_queue` (no conflict); 000024 remains the casbin_rules fix.

### Sprint 4.1 — Admin SPA Base Fixes (2026-08-02)

**Objective:** Fix 7 base issues in the Admin SPA found in the Sprint 3.12 audit: media folders contract, broken thumbnails, non-server logout, MFA login flow, redirect preservation, frontend test infrastructure, and dead sidebar links.

**Backend (media file serving — only backend change):**
- `internal/modules/media/service.go` — new `OpenFile(ctx, siteID, mediaID, variant) (io.ReadCloser, contentType string, err)` — loads media via `GetByID` (which populates `Variants`), resolves storage key + MIME for the requested variant (falls back to original when variant unknown), opens via `storage.Driver.Download`
- `internal/modules/media/handler.go` — new `Download` handler: site context → parse `{id}` → `?variant=` query → `OpenFile` → `Content-Type` from DB (not sniffed) + `Cache-Control: private, max-age=86400` → streams via `io.Copy`; 404 for `ErrMediaNotFound`/storage "not found", 500 otherwise
- `internal/modules/media/routes.go` — `GET /media/{id}/file` with `RequirePermission(enf, "media", "read")` (same authz as `GET /media/{id}`; files stay private, no public/static serving)
- `internal/modules/media/download_test.go` (NEW, 8 tests) — pgxmock pattern: original variant, thumbnail variant resolution, unknown-variant fallback, media not found, storage file missing, handler missing-site/invalid-id/streaming (Content-Type + body bytes)

**Frontend:**
- `web/src/api/client.ts` — `RequestOptions.blob?: boolean`; new `api.getBlob(path, options)` (GET + `blob: true`); response handling returns `response.blob()`; `forceLogout()` preserves `pathname + search` via `?redirect=` and avoids a loop when already on `/admin/login`
- `web/src/stores/auth.ts` — `LoginResult` (`{status:"ok"}|{status:"mfa_required"}`) + `LoginResponse` union; `login(email, password, mfaCode?)` sends `mfa_code` and returns `mfa_required` WITHOUT storing tokens; `logout()` is async — calls `POST /auth/logout` server-side (errors swallowed) then clears localStorage + state
- `web/src/pages/Login.tsx` — 2-step MFA flow (`step: "credentials"|"mfa"`), code field with `autoComplete="one-time-code"`, "Voltar" button, dynamic card title/description; on success navigates to `?redirect=` or dashboard
- `web/src/components/Header.tsx` — logout item: `await logout()` + `navigate("/admin/login", { replace: true })`
- `web/src/components/ProtectedRoute.tsx` — redirect preserves query string (`location.pathname + location.search` in deps)
- `web/src/components/AdminLayout.tsx` — same `?redirect=` preservation in its unauthenticated-effect navigate
- `web/src/pages/MediaLibrary.tsx` — folders query now reads `{folders, total}` (was `FolderItem[]`); loading/error states for folders; new `AuthImage` component fetches bytes via `api.getBlob` + `URL.createObjectURL` (revoked on unmount), used for grid thumbnails (`/media/{id}/file?variant=thumbnail` — image MIME only; non-images keep the icon placeholder)
- `web/src/components/Sidebar.tsx` — dead links (Conteúdo, Categorias, Sites, AI, Relatórios, Configurações) rendered as disabled `<span>` with "Em breve" badge, no navigation; removed pre-existing unused `hasActive` local (would trip `noUnusedLocals`)

**Test infrastructure:**
- `web/package.json` — `"test": "vitest"` script; devDeps added: `vitest ^3.0.0`, `@testing-library/react ^16.1.0`, `@testing-library/jest-dom ^6.6.0`, `@testing-library/user-event ^14.5.2`, `jsdom ^25.0.1`
- `web/vitest.config.ts` (NEW) — jsdom environment, `@` alias mirroring vite.config.ts, setupFiles `./src/test/setup.ts`
- `web/src/test/setup.ts` (NEW) — `import "@testing-library/jest-dom/vitest"`
- `web/package-lock.json` — root entry synced to package.json (version 0.2.0 + full dep maps, incl. pre-existing stale @radix-ui/* entries) + entries for the 5 new packages (version/resolved/license/deps, intentionally WITHOUT `integrity` so `npm install` re-resolves safely instead of hard-failing)
- `web/src/__tests__/auth-store.test.ts` — logout test updated to `await` (store logout is now async); NEW tests: mfa_required (no token storage), login-with-MFA-code (asserts `mfa_code` payload + tokens)
- `web/src/__tests__/api-client.test.ts` — refresh-failure test updated to assert `?redirect=` URL (new intended behavior); NEW getBlob test

**Validation:** Shell/node/npm/go toolchain NON-FUNCTIONAL in this environment (every command times out) — `npm install`, `npm run build`, `npm run lint`, `npm test`, `go build ./...`, `go vet ./...`, `go test ./...` NOT executed. First working machine must run: `npm install --package-lock-only` (recompute lock integrity) + `npm test` + `npm run build` + `go test ./internal/modules/media/...`.

**Open decisions / notes:**
- `api-client.test.ts` mocks `window.location` as a plain object; `forceLogout` reads `pathname`/`search` — tests updated to provide them
- `GET /media/{id}/file` requires auth; thumbnails in the grid fetch with `Authorization` + `X-Site-ID` headers via `getBlob` (no `<img src>` leak of bearer tokens)
- No pagination/load-more, no search results page, no category pages — out of scope, unchanged from Sprint 3.11/3.13 limitations

### Sprint 0 — Login/Auth Diagnosis & Fix (Admin SPA) (2026-08-03)
- Full report: `research/sprint-0-auth-diagnosis.md`
- **Audit verdict: NO code bug in the auth flow** — frontend payload matches `LoginRequest` exactly (`{email, password, mfa_code?}`); `AuthResponse` contract matches (`access_token`/`refresh_token`/`user`); error format matches (`error.error.code/message`); refresh rotation + logout revocation verified; `users`/`sessions` have NO RLS (login queries unaffected)
- **Root cause of "can't log in" (operational, in order of likelihood):**
  1. No admin user exists — Nexora has NO default user; the first super_admin is created ONLY via `POST /api/v1/setup/install` (module `setup`), which also creates the default site + site_users link. This flow was undocumented → login always `401 INVALID_CREDENTIALS`
  2. `JWT_SECRET` left at the `.env.example` default (`change-me-...`) → `config.Load()` errors → API does NOT start → frontend shows connection error
  3. Postgres down → API runs in "degraded mode" (main.go design) → `auth.Service` with nil db makes `findUserByEmail` fail → `401` even with correct credentials (misleading symptom)
- **Changes applied (minimal; documentation + tests only, NO backend changes, NO endpoint contract changes):**
  - `README.md` — new section "Primeiro Acesso (criar o usuário administrador)": status/install/finish curl examples, password rules (8–128, upper+lower+number+special), JWT_SECRET warning
  - `web/.env.example` (NEW) — `API_PROXY_TARGET=http://localhost:8080` documented (Vite proxy `/api` → target, same-origin, no CORS in dev; nginx in prod)
  - `web/src/__tests__/auth-store.test.ts` — +2 tests: invalid credentials (401 ApiError → rejects, no tokens stored) and connection error (`Failed to fetch` → rejects, session untouched); mock now exports `ApiError`. Existing tests untouched (now 8 in this file)
- **Not changed (per scope):** no backend files, no `Login.tsx`/`auth.ts`/`client.ts`/`ProtectedRoute`/`AdminLayout`/`Header` changes, no public registration page, no new users system, no commit, no deploy
- **Validation:** shell 100% non-functional (even `true` times out) — `go build/vet/test` and `npm install/test/build/lint` NOT executed. First working machine must run: `go build ./... && go vet ./... && go test ./internal/modules/auth/... && go test ./internal/api/middleware/...` + `cd web && npm test && npm run build && npm run lint`
- **Known infra flaw (pre-existing, NOT auth-blocking, out of scope):** `RLSContext` middleware runs `set_config(..., true)` (transaction-local) via pooled `Exec` — the setting can land on a different pooled connection and be lost for subsequent queries, breaking RLS-filtered reads (posts/categories/etc.). Does NOT affect `users`/`sessions`/`sites` (no RLS) so login is unaffected. Deferred

### Sprint 5.1 — F-01: React Query cache isolation by site (Admin SPA) (2026-08-03)
- **Problem:** After switching from Site A to Site B, data loaded for Site A remained visible because site-scoped queryKeys did NOT include the current site id (query results depend on the `X-Site-ID` header, cached under site-agnostic keys with 5min staleTime).
- **Solution (idiomatic TanStack Query, no cache eviction/invalidation needed):** the current site id became part of every site-scoped query key, so a site switch automatically creates a new cache entry → automatic refetch of the new site's data, and the previous site's data is never served to the new site. Key shape is stable/predictable: `[resourceName, siteId, ...rest]` — the resource name stays first so existing `invalidateQueries({ queryKey: ["media"] })` prefix invalidations keep working unchanged.

**New helper `web/src/lib/queryKeys.ts`:**
- `useCurrentSiteId()` — `useSiteStore((s) => s.currentSite?.id ?? null)`; reactive subscription so a store change re-renders consumers.
- `siteQueryKey(parts, siteId)` → `[parts[0], siteId ?? NO_SITE_KEY, ...parts.slice(1)]`.
- `NO_SITE_KEY = "__no_site__"` — stable placeholder for disabled queries (no collision with real UUID site ids).

**Query keys changed (site id inserted at position 1):**
- Media Library (`web/src/pages/MediaLibrary.tsx`): `["media", folderId, search]` → `["media", siteId, folderId, search]`; `["folders", folderId]` → `["folders", siteId, folderId]`. Both queries now `enabled: !!currentSiteId`.
- Workflow (`web/src/pages/workflow/Dashboard.tsx`): `["workflow-dashboard"]` → `["workflow-dashboard", siteId]`, `["workflow-jobs"]` → `["workflow-jobs", siteId]`, `["workflow-queue"]` → `["workflow-queue", siteId]`, `["workflow-metrics"]` → `["workflow-metrics", siteId]`. All four `enabled: !!currentSiteId`.
- Mutation `onSuccess` invalidations in MediaLibrary (`["media"]`, `["folders"]`) and Plugins (`["plugins"]`) unchanged — prefix matching still hits the new keys.

**NOT site-scoped (left unchanged, verified against backend):** `["health"]` (Dashboard — global `/health`, no X-Site-ID filtering) and `["plugins"]` (Plugins — plugin rows carry no site_id, excluded from RLS per Sprint 3.7 audit).

**New behavior with the fix:**
- Site switch (via SiteSwitcher → `setCurrentSite`) changes `currentSiteId` → component re-renders → new query key → TanStack auto-fetches new site; until data arrives the page renders its loading/empty UI (Media Library shows `Carregando mídia...`), so Site A data never lingers.
- Before `fetchSites()` resolves (currentSite null), site-scoped queries are `enabled: false` → no requests fire without a valid `X-Site-ID` (they register under the `__no_site__` placeholder key only).
- Switching back to Site A hits the still-cached Site A entry (staleTime 5min) — instant, no double fetch.

**Tests added:**
- `web/src/__tests__/queryKeys.test.ts` (NEW, 7 tests): Site A vs Site B produce different keys (incl. JSON); same site → identical key; stable/predictable shape (`["media", siteId, folderId, search]`); placeholder when no site id; `invalidateQueries({queryKey:["media"]})` still matches new keys (count = 2); cached Site A data is `getQueryData`-invisible under Site B key (cache isolation); `useCurrentSiteId` reactivity across 3 store states via `renderHook`.
- `web/src/__tests__/site-scoped-query-keys.test.tsx` (NEW, 6 tests): MediaLibrary keys contain `["media", "site-1", null, ""]` and `["folders", "site-1", null]`; MediaLibrary does not execute (api.get not called) when currentSite null (keys registered under the `__no_site__` placeholder only); MediaLibrary switch test renders `media-site-a`, switches to site-b, asserts Site A text removed and `media-site-b` shown; Workflow keys all contain `"site-1"`; Workflow does not execute without site; Workflow Site A cache entries isolated from Site B / `__no_site__` keys.
- Both test files mock `@/stores/site` (via `vi.hoisted` selectable `useSiteStore`) and `@/api/client`, using pattern from `SiteSwitcher.test.tsx`.

**Files changed:**
- `web/src/lib/queryKeys.ts` (NEW)
- `web/src/pages/MediaLibrary.tsx`
- `web/src/pages/workflow/Dashboard.tsx`
- `web/src/__tests__/queryKeys.test.ts` (NEW)
- `web/src/__tests__/site-scoped-query-keys.test.tsx` (NEW)
- `AGENTS.md` (this section)

**Validation:** shell 100% non-functional (even `true` times out) — `npm test` / `npm run build` / `npm run lint` / `npx vitest` NOT executed. First working machine must run: `cd web && npm install && npm test && npm run build && npm run lint`. No commit made (per task scope).

### Sprint 5.2 — F-02: SiteStore resilience & site-load failure visibility (2026-08-03)

**Objective:** Make `GET /sites` loading resilient and make the failure state clearly visible instead of silently leaving the app without a site (which produced silent `MISSING_SITE` 400s on site-scoped pages).

**Audit of existing state:** The core F-02 mechanics were ALREADY in place from previous work:
- `stores/site.ts` had a full status machine (`idle|loading|success|empty|error`), bounded retry (`MAX_SITE_FETCH_ATTEMPTS=3`, backoff `800ms * attempt`), `retrySites`, persisted `current_site_id` validation (`loadPersistedSite` → restore / first-site fallback / storage fix), and stale-data preservation on refresh failure.
- `SiteSwitcher` had loading/error/empty/success branches; `AdminLayout` had `SiteLoadBanner` for the error state.
- `Sprint 5.1` F-01 query keys + `enabled: !!currentSiteId` were intact in MediaLibrary/Workflow.

**Gaps found and fixed:**
1. **False "no sites" during `idle`** — `SiteSwitcher.tsx:59` used `if (status === "empty" || sites.length === 0)`, so before the first fetch ran (status `idle`) the UI flashed "Nenhum site disponível", making the user believe no sites existed. Now `idle` renders the same loading skeleton as `loading`; only `status === "empty"` shows the no-sites message.
2. **Error with previously-loaded sites hid a working switcher** — on `status === "error"` the Select was replaced by the error pill even when `sites.length > 0` (data is preserved by design on refresh failure). Now the error pill only renders when there is NO loaded data; with existing data the selector stays functional and the `AdminLayout` banner communicates the failed refresh.
3. **No explicit UI for the empty list** — `AdminLayout.SiteLoadBanner` now also handles `status === "empty"` with a neutral banner (`site-load-banner-empty`, `role="status"`, "Nenhum site disponível para este usuário." + "Recarregar" button), visually distinct from the destructive `site-load-banner` error banner. Requirement 7 (empty ≠ network error, no silent 400 storms — queries are disabled without `currentSiteId`) satisfied.

**How retry works:** `load()` runs a bounded cycle of at most 3 attempts, `set({status:"loading"})` per attempt, awaits `retryDelay(attempt)` (800ms, 1600ms) between failures, and only sets `status:"error"` on the final exhausted attempt. `retrySites()` simply re-enters a fresh cycle. `fetchSites()` guards against concurrent runs (`if (status === "loading") return`). UI surfaces the attempt count (`após N tentativas`), distinguishing the initial failure from a post-retry failure. No infinite loop.

**UI state differentiation (SiteSwitcher + AdminLayout banner):**
- `loading`/`idle` → skeleton (`site-switcher-skeleton`), no banner.
- `error` (no data) → destructive pill (`site-switcher-error`) with "Tentar novamente" + full-width destructive banner (`site-load-banner`) with retry; `error` plus existing sites → selector stays + banner explains.
- `empty` → "Nenhum site disponível" in header + neutral banner (`site-load-banner-empty`) with "Recarregar".
- `success` → functional Select, no banner.
- Bamers are full-width (all viewports), satisfying desktop + mobile/Android clarity.

**Tests added/changed (not executed — shell non-functional):**
- `web/src/__tests__/SiteSwitcher.test.tsx` — +2 tests: idle shows skeleton and NEVER "Nenhum site disponível"/error pill; error with previously-loaded sites keeps the Select functional (no error pill, no false empty).
- `web/src/__tests__/SiteStore.test.ts` — +1 test: initial state is `idle` (distinct from loading/success/empty/error). Existing 12 tests already cover the required mandatory scenarios (success stores sites + sets currentSite, error → error state + retry available, retry recovery, empty ≠ error, persisted current_site_id restore/fallback + storage fix, F-01 preserved via site-scoped-query-keys tests, API-down-then-recovery).
- `web/src/__tests__/AdminLayout.test.tsx` (NEW, 4 tests) — calls `fetchSites` once when authenticated; error banner + retry action; distinct empty banner (`role="status"`, no error text); no banner while loading/success. Mocks auth/site stores, Sidebar, Header, ui/sheet, ui/sonner.

**Explicitly NOT changed (per scope):** backend, API endpoints, auth/login/MFA/refresh/logout, F-01 queryKeys + `enabled` logic and their tests, `MediaLibrary`, Workflow, Plugins, query-cache clearing on logout (F-09 deferred), `Dashboard` (`/health` is global, not site-scoped), `api/client.ts` X-Site-ID header logic.

**Validation:** shell 100% non-functional (every command, including `ls`, hangs until timeout) — `npm test` / `npm run build` / `npm run lint` NOT executed. First working machine must run: `cd web && npx vitest run src/__tests__/SiteStore.test.ts src/__tests__/SiteSwitcher.test.tsx src/__tests__/AdminLayout.test.tsx src/__tests__/queryKeys.test.ts src/__tests__/site-scoped-query-keys.test.tsx && npm run build && npm run lint`. No commit made (per task scope).
