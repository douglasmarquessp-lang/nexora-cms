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

### Sprint 5.3 — F-10: Workflow Notifications tab (Admin SPA) (2026-08-03)

**Objective:** Make the "Notifications" tab of `web/src/pages/workflow/Dashboard.tsx` real. Previously it was a stub: it rendered `null` when `jobs.length > 0` and used `(jobs || []).length === 0` as a proxy for "no notifications" (wrong semantics — jobs are not notifications). Frontend-only; no backend changes; no commit.

**Backend endpoints consumed (verified read-only, unchanged):**
- `GET /workflow/notifications` (`ListNotifications`, handler.go:602) — returns `NotificationList { notifications[], total, unread }` (model.go:335-339); supports `type`/`unread`/`limit`/`offset` query params; requires site context
- `PUT /workflow/notifications/{notifID}/read` (`MarkNotificationRead`, handler.go:624) — returns `{"status":"ok"}`
- `POST /workflow/notifications/read-all` (`MarkAllNotificationsRead`, handler.go:651) — returns `{"status":"ok"}`
- `Notification` model (model.go:194-206): `id`, `site_id`, `notification_type`, `title`, `message`, `severity` (string: info|warning|error|critical|success), `read` (bool), `action_url` (omitempty), `created_at`

**Changes to `web/src/pages/workflow/Dashboard.tsx`:**
- New frontend types: `Notification` (id, notification_type, title, message, severity, read, action_url?, created_at), `NotificationListResponse` (notifications, total, unread)
- Query (site-scoped per F-01 pattern, `web/src/lib/queryKeys.ts`):
  - `queryKey: siteQueryKey(["workflow-notifications"], currentSiteId)` → key shape `["workflow-notifications", siteId]`
  - `queryFn: () => api.get<NotificationListResponse>("/workflow/notifications", { params: { limit: "50" } })`
  - `enabled: !!currentSiteId` — no requests without a valid site (registers under `NO_SITE_KEY` placeholder only)
- Mutations:
  - `markReadMutation` — `api.put(/workflow/notifications/${id}/read)`, `onSettled: invalidateQueries({ queryKey: ["workflow-notifications"] })` (prefix invalidation covers the site-scoped key)
  - `markAllReadMutation` — `api.post("/workflow/notifications/read-all")`, same invalidation
- Tab bar: "Notifications" button shows an unread count badge (`notifications-unread-badge`, `data-testid`) rendered only when `notifUnread > 0`
- New local components (bottom of file):
  - `NotificationsPanel` — receives data/isLoading/isError/onRetry/onMarkRead/onMarkAllRead/markAllPending; renders `LoadingState variant="inline"` (loading), `ErrorState` with retry (error, pt-BR), `EmptyState` "Nenhuma notificação" / "Você está em dia." (only when `notifications.length === 0`), list otherwise
  - `NotificationItem` logic inline — Card with severity badge (`SeverityBadge`), unread dot (`notification-unread-dot` data-testid), title, message, relative + formatted date (`formatRelativeTime`/`formatDate` from `@/lib/utils`), "Marcar como lida" ghost button (only when unread, `notifications-mark-read-{id}` data-testid), "Marcar todas como lidas" outline button (only when `unread > 0`, `notifications-mark-all-read` data-testid)
  - `SeverityBadge` — color + label maps for info/warning/error/critical/success, fallback to muted
  - `safeActionUrl(actionUrl)` — `new URL()` parse; only `http:`/`https:` protocols accepted; any other (incl. `javascript:`) → `null`; link rendered with `target="_blank"` + `rel="noreferrer noopener"`, never `dangerouslySetInnerHTML`
- Summary line: `{total} notificação(ões), {unread} não lida(s)`

**Tests (`web/src/__tests__/workflow-notifications.test.tsx`, NEW, 12 tests):**
- site-scoped query key `["workflow-notifications", "site-1"]` registered
- no execution without site (`apiMock.get` never called; key registered under `NO_SITE_KEY`)
- renders title + message + severity badge (Success/Error)
- unread count badge on tab button
- unread vs read distinction (1 unread dot, mark-read + mark-all buttons present)
- empty state only when notifications array is empty
- error state + "Tentar novamente" retry recovers
- mark-read calls `apiMock.put("/workflow/notifications/n1/read")` and query used `{ params: { limit: "50" } }`
- mark-all calls `apiMock.post("/workflow/notifications/read-all")`
- `javascript:` action_url renders NO link (no `Ver detalhes`, no `a[href*=javascript:]`)
- https action_url renders safe link (`href`, `rel="noreferrer noopener"`, `target="_blank"`)
- Mocks: `@/stores/site` (`useSiteStoreMock` via `vi.hoisted`), `@/api/client` (`apiMock` incl. `put`), pattern from `site-scoped-query-keys.test.tsx`

**Also updated:** `web/src/__tests__/site-scoped-query-keys.test.tsx` — `mockWorkflowApi` now handles `/workflow/notifications` (returns empty list); assertions added for `["workflow-notifications", "site-1"]` and `["workflow-notifications", NO_SITE_KEY]` keys.

**Explicitly NOT changed (per task spec):** backend (incl. workflow handler/model), auth/login/MFA/refresh/logout/forceLogout/sessionReset/SiteStore, F-01 queryKeys + `enabled` logic, MediaLibrary/Plugins/Dashboard pages, F-09 (query-cache clearing on logout — NOT documented in this sprint). No commit made.

**Validation:** shell 100% non-functional (every command, including `true`, hangs until timeout) — `npm test` / `npm run build` / `npm run lint` / `npx vitest` NOT executed. First working machine must run: `cd web && npm install && npx vitest run src/__tests__/workflow-notifications.test.tsx src/__tests__/site-scoped-query-keys.test.tsx && npm run build && npm run lint`.

### Sprint 5.4 — Frontend Test Suite Green (2026-08-04)

**Objective:** Make the full vitest suite + `tsc -b` + `npm run lint` + `npm run build` pass. Only tests and test infrastructure changed — no production code, no F-01/F-02/F-09/F-10 logic, no commit.

**Final state (validated):** `npx vitest run` = 10 files, **82/82 tests pass**; `npx tsc -b` exit 0; `npm run lint` 0 errors (45 pre-existing `no-explicit-any` warnings); `npm run build` exit 0 (single >500kB chunk warning, pre-existing).

**Fixes applied (tests only):**

1. **Global cleanup** (`src/test/setup.ts`) — added `afterEach(() => { cleanup(); })` from `@testing-library/react`. RTL 16 only auto-cleans when `afterEach` is global; vitest runs without `globals: true`, so DOM mounted by earlier tests leaked into later ones (multiple `Notifications` components → duplicate `notifications-unread-badge`; tab-render count became test index). Alone this fixed 6 failures (4 `AdminLayout` + 2 `workflow-notifications`) and eliminated all 4 "Unhandled Errors" (`jobs.filter is not a function` from `Dashboard.tsx:382`).
2. **`workflow-notifications.test.tsx`** — `mockWorkflowApi` defensive `Promise.resolve({})` → explicit per-path returns (`/workflow`, `/workflow/queue` → `[]`; dashboard/metrics → `{}`); simplified to respect switch narrowing (removed impossible `path === "/workflow"` comparisons inside `case "/workflow/dashboard"`).
3. **`auth-store.test.ts`** — added `useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: true })` to `beforeEach` (store leaked auth state between tests; prior tests left `isAuthenticated: true`).
4. **`SiteSwitcher.test.tsx`** — mock store migrated to `vi.hoisted` (`useSiteStoreMock`) to fix TS build error (`mockImplementation` on typed import); Radix mock completed (was missing `Group`, `Viewport`, `ScrollUpButton/Down`, `Label`, `ItemIndicator`, `ItemText`, `Separator` — `select.tsx:7` destructures `SelectPrimitive.Group`); replaced native-`<select>` mock with a `vi.hoisted` `selectBridge` + context-free clickable `role="option"` items (jsdom's native select only honors a value if a matching `<option>` is a direct child — div-wrapped options made the controlled value stick as `""`); change test now clicks `option-site-2` instead of `fireEvent.change` on the native select.
5. **`ProtectedRoute.test.tsx`** — mocks migrated to `vi.hoisted` (`useAuthStoreMock`, `useSiteStoreMock`) — same TS build fix as #4.
6. **`vitest.config.ts`** — `testTimeout: 15000`, `hookTimeout: 15000` (ProtectedRoute runs ~12.6s in the full parallel suite; 5s default caused intermittent timeouts).
7. **`queryKeys.test.ts`** — "keeps resource-name prefix invalidation working" asserted the TanStack v4 contract (`invalidateQueries` returns a count). v5 returns `void`; rewritten to `getQueryCache().findAll({ queryKey: ["media"] })` + `q.state.isInvalidated` (2 invalidated, 2 total).
8. **`api-client.test.ts`** — `mock.calls[0]` → `mock.calls[0]!` (7× `noUncheckedIndexedAccess`); `err as ApiError` → `err as InstanceType<typeof ApiError>` (3× — `ApiError` destructured from dynamic `await import` is a value-only binding).

**Not changed:** any production file (`src/pages/*`, `src/components/*` except none, `src/stores/*`, `src/api/client.ts`, `src/lib/queryKeys.ts`, `Dashboard.tsx`, etc.), `package.json`/`package-lock.json` (no dep changes, no npm install), F-01/F-02/F-09/F-10 logic and their test semantics.

**Validation:** EXECUTED in this environment (shell functional): `npx vitest run` (82/82), `npx tsc -b` (0), `npm run lint` (0 errors), `npm run build` (0). No commit made.

### Sprint 5.5 — F-11: Dashboard Inteligente (executive panel) (2026-08-05)

**Objective:** Transform the main Admin Dashboard from a 2-card health stub into an executive CMS panel consuming pre-existing backend endpoints. No refactor, no architecture change, F-01/F-02/F-09/F-10 intact.

**Endpoints consumed (all verified against backend contracts):**
- `GET /health` — kept as-is (global, not site-scoped, `["health"]` key unchanged)
- `GET /workflow/dashboard` — `WorkflowDashboard` struct (workflow/model.go:208: `running_jobs`, `completed_jobs`, `failed_jobs`, `success_rate`, `scheduled_publications`, `queue_size`, `pending_review`, ...)
- `GET /editorial/stats` — `DashboardStats` struct (editorial/model.go:62: `published_posts`, `draft_posts`, `recent_posts[]`)
- `GET /workflow/history?limit=10` — `HistoryEntry[]` (workflow/model.go:177; action strings `workflow.started|paused|resumed|cancelled|retry|completed`, `error_message` for failures)

**Changes — single file rewrite: `web/src/pages/Dashboard.tsx`**
- 3 new site-scoped React Query queries using the F-01 pattern: `siteQueryKey(["dashboard-workflow"], currentSiteId)`, `["dashboard-editorial"]`, `["dashboard-history"]` — all `enabled: !!currentSiteId` (no requests without valid site; register under NO_SITE_KEY only). `["health"]` stays global. Prefix invalidations work via resource-name-first keys.
- 10 metric cards (grid `md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`): Status do Sistema (health), Jobs em execução (running_jobs), Jobs concluídos (completed_jobs), Jobs com erro (failed_jobs), Taxa de sucesso (success_rate %), Jobs agendados (scheduled_publications), Itens na fila (queue_size), Artigos publicados (published_posts), Artigos em rascunho (draft_posts), Conteúdo pendente (pending_review). Per-card Skeleton while site queries load.
- `ActivitiesPanel` — merges `recent_posts` (post kind, Publicado/Rascunho/Agendado -> green) + workflow history (workflow kind, action label / Falha if `error_message` -> red; blue otherwise), sorted desc, top 10, `formatRelativeTime`; `LoadingState`/`ErrorState`(retry)/`EmptyState` from shared components.
- `QuickAccessCard` — shortcuts Workflow (`/admin/workflow`), Media Library (`/admin/media`), Plugins (`/admin/plugins`) as `Button asChild`+`Link`; Sites (`/admin/sites`) rendered as disabled "Em breve" span (route doesn't exist yet — consistent with Sidebar `soon: true`).
- Shared `StatCard` local component with optional icon + loading skeleton; `buildActivities`/`workflowActionLabel`/`postStatusLabel`/`StatusDot` helpers. Uses existing `Card`/`Button`/`Skeleton`/`LoadingState`/`EmptyState`/`ErrorState`/lucide icons — no new components, no new dependencies.

**Tests (`web/src/__tests__/dashboard.test.tsx`, NEW, 7 tests):** site-scoped keys `["dashboard-workflow|dashboard-editorial|dashboard-history", "site-1"]` registered; no site-scoped execution without site (`apiMock.get` not called, keys under NO_SITE_KEY); renders metric cards + combined values (90.5%); renders activities (Post site-1, Workflow iniciado), quick shortcuts, Sites "Em breve"; Site A cache isolated from Site B / NO_SITE_KEY; `["health"]` remains global; prefix invalidation `["dashboard-workflow"]` isolates from `["dashboard-editorial"]`/`["health"]`. Uses `vi.hoisted` `useSiteStoreMock`/`apiMock` + `mockDashboardApi` switch pattern from `site-scoped-query-keys.test.tsx`.

**Validation:** EXECUTED in this environment: `npx tsc -b` (0), `npm run build` (0, pre-existing >500kB chunk warning), `npm run lint` (0 errors, 45 pre-existing warnings — no new), `npx vitest run` = **11 files, 89/89 pass** (82 existing + 7 new). No commit made.

### Sprint 5.6 — F-21: SEO Intelligence Engine real (2026-08-05)

**Objective:** Replace every fake `simScore()` heuristic in the `seoengine` module with real deterministic analysis (weighted 0–100), persist real audits with articles, and add an optional configurable publish block (422 when SEO score < `SEO_MIN_PUBLISH_SCORE`).

**Architecture decisions:**
1. **Deterministic-only analysis.** All scores computed locally from text — zero API cost, zero latency, no `math/rand`. `simScore()` (which returned a constant midpoint) deleted.
2. **`internal/ai` reused for the heavy checks** (readability via `ScoreReadabilityDetailed`, duplicate blocks via `CheckDuplicateBlocks`) — injected as `ai.QualityChecker`; nil-safe fallbacks keep the module fully functional without AI.
3. **Publish gate is an interface defined in `publisher`** (`PublishGate`), implemented by `seoengine` — no hard module import from publisher to seoengine (seoengine imports publisher for the input type only; no cycle).
4. **Fail-open gate semantics:** gate errors → allow publish (logged); stored audit score on the linked post wins, otherwise the gate runs the same deterministic analysis inline on title+content (no DB write).
5. **Keyword research stays out of scope** (user decision): `AnalyzeKeywords` volume/difficulty/etc. now use stable FNV-1a hashes per keyword (`stableScore`) instead of constants — deterministic, no real research claims.

**New file `internal/modules/seoengine/analyzer.go`:**
- Weights (sum 100): title 15, meta 10, headings 15, keyword 20, readability 10, internal links 10, external links 5, EEAT 10, images/ALT 5
- `ArticleAnalysisInput{Title, MetaDescription, Slug, Content, Keyword, Language}` + `ArticleAnalysis` (12 sub-scores + density, word count, link/image counts, `DuplicateCount`, `Suggestions []AuditIssue`)
- `AnalyzeArticle(ctx, in, qc)` — pure function, no DB: `analyzeTitle` (30–60 chars ideal, keyword presence), `analyzeMeta` (150–160 chars, keyword), `analyzeHeadings` (markdown `#`/`##`/`###` + HTML h1-h3 regexps; exactly one H1, H2≥2, H3≥1), `analyzeKeyword` (density band 1–3%, first-100-words + title presence), `analyzeReadability` (delegates to `qc.ScoreReadabilityDetailed`; nil → 50), `analyzeLinks` (markdown links relative→internal, http(s)→external; bare-URL regex with markdown destinations stripped to avoid double-counting), `analyzeEEAT` (author/date/source/keyword-coverage signals), `analyzeImages` (markdown `![alt]` + HTML alt), `analyzeSlug`, `analyzeSchema` (JSON-LD regex)
- Additional deterministic dimensions: `analyzePassiveVoice` (PT `foi/foram/é/são… + -ado/-ida` regex, EN `is/was/been… + -ed` regex; score = 100 − count×8), `analyzeSentenceVariation` (avg sentence length 12–22 ideal), `analyzeFreshness` (year ≥ current−2 → 100, any year → 60, none → 30), `analyzeDuplicates` (via `qc.CheckDuplicateBlocks`, score = 100 − blocks×20), `TopicalAuthorityScore` = keyword term coverage × 100, `ParagraphScore` = (headings+readability)/2
- Bilingual messages: `bi{pt, en}` pair struct + `issue(field, msg, sugg bi, score, priority, lang)`; EN selected when `Language == "en"`, default PT
- `extractContentText(contentJSON)` — converts posts JSONB blocks (`{"type":"heading|text", "text":…}` incl. nested `content`) to markdown-ish text; invalid JSON falls back to raw string
- `shingleSimilarity(a, b)` — 3-word shingle Jaccard for pairwise duplicate detection; `deriveKeyword` (longest non-stopword from title), `tokenize`, `clampScore`, `round2`, `keywordCoverage`

**Service rewrites (`internal/modules/seoengine/service.go`):**
- `Service` gains `qualityChecker ai.QualityChecker` + `SetQualityChecker`; wired in `cmd/api/main.go` as `seoengineSvc.SetQualityChecker(aiModule.NewQualityChecker())`
- `RunFullAudit` — loads project + linked post (title/slug/content::text/excerpt/metadata->>'meta_description'), runs `AnalyzeArticle`, persists real scores into `seo_audits` (incl. paragraph/passive/sentence-variation/duplicate/freshness), updates `seo_projects` (seo_score, readability, eeat, freshness, technical = avg(title,meta,slug,heading,schema), checklist), inserts `seo_scores` (content = avg(readability,eeat,freshness,keyword), linking = avg(internal,external), metadata = avg(meta,title), topical authority, multilingual = 50 neutral constant), and updates the post row when `PostID` set
- `buildChecklist(analysis)` — derives `ChecklistItem`s from real `Suggestions` via field→category map (title→CategoryTitle, internal/external_link→CategoryLink, passive_voice/sentence_variation→CategoryReadability, …)
- `GenerateChecklist` — runs the real analysis, persists to `seo_projects.checklist`
- `AnalyzeContent` / `AnalyzeTechnical` — real analysis filtered by field (readability/eeat/freshness/passive/sentence-variation vs title/meta/slug/headings/image_alt/schema)
- `GetInternalLinkingSuggestions` — queries up to 25 site posts, scores each by `keywordCoverage(title+slug)`, top 5 with relevance ≥ 40, real slug/title anchors
- `DetectDuplicates` — loads up to 20 site posts (excluding project's own), pairwise `shingleSimilarity` ≥ 0.6 → `DuplicateContent`
- `AnalyzeKeywords` — deterministic `stableScore(keyword+":dimension")` FNV-1a values; cannibalization only when keywords overlap; `avgKeywordScore` cluster authority
- `CheckPublishScore(ctx, publisher.PublishGateInput)` — stored `posts.seo_score` when `seo_analyzed_at` set (DB errors fail over to inline), else inline deterministic analysis
- `simScore` deleted; helpers `stableScore`/`avgKeywordScore`/`fnv64` added

**Publish gate (`internal/modules/publisher/gate.go` + service/handler):**
- `PublishGateInput{SiteID, PostID, Title, Content, Language}` + `PublishGate` interface + `ErrSEOPublishBlocked`
- `Service.publishGate` + `minPublishScore` (from `cfg.SEO.MinPublishScore`; 0 disables) + `SetPublishGate` setter + `checkPublishGate` (skip when disabled/empty; gate error → fail-open; score < min → `ErrSEOPublishBlocked` wrapping score info)
- Enforced in `PublishArticle` when `req.PostID != nil` (stored-score path) and always in `PublishGeneratedArticle` (inline path) — the single funnel covers all 4 auto-content engines
- `handler.go` `Publish`: new branch `ErrSEOPublishBlocked` → `422 SEO_SCORE_BELOW_MINIMUM`
- Wired in `cmd/api/main.go`: `publisherSvc.SetPublishGate(seoengineSvc)` right after `SetQualityChecker`

**Config + migration:**
- `internal/pkg/config/config.go` — `SEOConfig{MinPublishScore float64}` (zero = gate disabled), `getEnvFloat` helper, `SEO_MIN_PUBLISH_SCORE=80` default; `.env.example` new `# === SEO Intelligence Engine ===` section
- `migrations/000026_add_post_seo_columns.{up,down}.sql` (NEW) — `posts`: `seo_score NUMERIC(5,2) DEFAULT 0`, `seo_analyzed_at TIMESTAMPTZ`, `seo_issues JSONB DEFAULT '[]'`, index `idx_posts_site_seo_score`

**Tests:**
- `internal/modules/seoengine/analyzer_test.go` (NEW, 44 tests) — determinism, score ranges, heading detection, density band, title length bands, keyword penalty, meta empty, no/multi H1, passive voice, freshness (no date/recent year), images alt, JSON-LD, links (internal/external + no double-count), EEAT signals, slug, weights sum, nil quality checker, `extractContentText` (headings/nested/invalid/empty), shingle similarity, clamp/round2, buildChecklist, filterIssues, issueMessages, stableScore, tokenize, keywordCoverage, deriveKeyword
- `internal/modules/seoengine/gate_test.go` (NEW, 7 tests) — stored score path (pgxmock), inline fallback on ErrNoRows/DB error/no post, no-DB inline, determinism
- `internal/modules/publisher/gate_test.go` (NEW, 7 tests) — disabled gate (min=0 / nil gate), empty content skip, blocks low score with input passthrough, allows high score with PostID passthrough, fail-open on gate error, `PublishGeneratedArticle` blocked before any DB access, manual publish unchanged without gate
- Pre-existing `internal/ai` failures confirmed NOT caused by this sprint (verified via `git stash` + test on clean tree): `TestGeminiProvider_GenerateStream_InvalidKey`, `TestGeminiProvider_Health_InvalidKey` (network-dependent), `TestCheckGrammarDetails_Issues`, `TestFindRepeatedWords/multiple_pairs`, `TestScoreReadabilityDetailed`, `TestCountSyllables` (Sprint 3.9 tests never executed — shell was broken then)

**Validation:** EXECUTED: `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 26 packages ok; only the 6 pre-existing `internal/ai` failures above (unrelated). No commit made.

### Sprint 5.7 — Translation Intelligence Engine (2026-08-05)

**Objective:** New `translation` module: PT↔EN article translation pipeline with native-quality review, per-language SEO regeneration, cultural localization, persistent glossary, persisted quality scores, and per-language publishing.

**Architecture decisions:**
1. **Pipeline: translate → native_review → seo_review → publish** (`StageOrder` in model.go). `ApproveStage` resumes at `nextStageType`; `RejectStage` reopens the *previous* stage with `attempt+1` and sets job to `waiting_review`. Job transitions: pending → running → waiting_review → running → completed; cancel allowed from pending/running.
2. **Deterministic-first quality gating, AI for translation only.** `runNativeReview` scores the AI translation deterministically (grammar via `ai.QualityChecker.CheckGrammarDetails`, fluency via `ScoreReadabilityDetailed` + literal-marker/repeated-word penalties) and re-rewrites the worst paragraph via `PromptTypeRevision` up to `MaxRewriteAttempts=2` until `MinNativeReviewScore=70`. `runSEOReview` requires `MinSEOScore=60` (recompute + rewrite fallback, then `fallbackSEO` deterministic metadata). No `math/rand` anywhere.
3. **Seamless mock fallback.** `ai.Manager`/`Prompts().Build` optional; when Build fails (mock provider returns non-JSON `"Mock response for: …"`), translate falls back to source text and `generateSEO` falls back to deterministic metadata (title from heading, `truncateTo` 155-char description) — the whole pipeline runs DB-only in tests.
4. **Concurrency safety.** `StartJob`/`ApproveStage`/`RejectStage` lock the job row with `SELECT … FOR UPDATE`; `executePipeline` runs in a goroutine and re-checks status each stage.
5. **Cultural localization** (`localize.go`): R$→US$, R$X.XXX,XX→$X,XXX.XX, km→miles, dates (dd/mm/yyyy→MM/DD/YYYY), Brasil→Estados Unidos (PT→EN) and reverse; applied on top of AI output.
6. **Glossary** (`glossary.go`): global + per-project terms, direction-aware (source/target language), word-boundary + case-insensitive replacement, `forbidden` terms never translated (protected), `GlossaryConsistency` counts protected/applied for the score.
7. **Score persistence** (`score.go`): `TranslationScore` with Grammar .25 / Fluency .20 / Naturalness .20 / SEO .15 (via `seoengine.AnalyzeArticle`) / Consistency .10 / Localization .10 → Overall; stored as JSONB in `translation_jobs.translation_score`.
8. **Publishing** (`runPublish`): creates a new post in the target site via `posts.Service.Create` (content blocks via `textToBlocks`, `PostMeta{language, translated_from}`), then `publisher.Service.PublishArticle` (SEO gate applies); falls back to `PublishGeneratedArticle` when no valid user ID.
9. **pgxmock v3 quirk discovered (test-only):** reflect-based `rowSets.Scan` cannot assign a plain `string` cell into a *named* string type dest (`JobStatus`) — it returns an error which `connRow.Scan` silently swallows, leaving the field zero. Fix: `scanJob` scans into a plain `string` and converts to `JobStatus(status)`. Same pattern already used in `loadStages`.
10. **go regexp has no backreferences** — repeated-word detection uses a deterministic token-scan `countRepeatedWords` (adjacent equal tokens, len ≥ 4) instead of `\b(\w+)\s+\1\b`.

**New files:**
- `migrations/000027_add_translation_tables.{up,down}.sql` — `translation_jobs`, `translation_stages`, `glossary_terms` + RLS policies (`translation_jobs_isolation`, `translation_stages_isolation`, `glossary_terms_isolation`) + `update_updated_at_column()` triggers + indexes + down migration
- `internal/modules/translation/model.go` — TranslationJob, TranslationStage, GlossaryTerm, StageResult, TranslationScore, 8 statuses, 4 stage types, DTOs, 14 EventBus events, 12 sentinel errors, `MinNativeReviewScore=70`, `MinSEOScore=60`, `MaxRewriteAttempts=2`
- `internal/modules/translation/detect.go` — `DetectLanguage` (stopword + diacritic heuristics, min confidence 0.5), `GenerateSlug` (diacritics stripped, PT truncated to 70), `DeriveKeyword`, `blocksToText`/`textToBlocks` (heading/text blocks, JSON round-trip)
- `internal/modules/translation/localize.go` — currency/units/dates/country expressions (PT↔EN)
- `internal/modules/translation/glossary.go` — ApplyGlossary, GlossaryConsistency, normalizeTerm
- `internal/modules/translation/score.go` — ComputeTranslationScore (deterministic, uses `ai.QualityChecker` + `seoengine.AnalyzeArticle`, nil-safe fallbacks)
- `internal/modules/translation/service.go` — CreateJob/GetJob/ListJobs/StartJob/CancelJob/GetScore, ApproveStage/RejectStage, glossary CRUD, audit + events
- `internal/modules/translation/pipeline.go` — executePipeline, runTranslate/runNativeReview/runSEOReview/runPublish, generateSEO/fallbackSEO, parseJSONObject, truncateTo, worstSection, splitParagraphs, rewriteSection
- `internal/modules/translation/handler.go` — 15 REST endpoints under `/api/v1/translation/` (jobs CRUD/start/cancel/stages/score/approve/reject, detect, glossary CRUD)
- `internal/modules/translation/module.go` — kernel module with SetAIManager/SetQualityChecker/SetPostsSvc/SetPublisherSvc

**Wiring:**
- `internal/api/routes.go` — `Dependencies.TranslationSvc`, `registerTranslationRoutes` (called after `registerWorkflowRoutes`, before `registerAIRoutes`)
- `cmd/api/main.go` — `translationMod` after `workflowMod`, added to kernel module list; `SetAIManager(aiSvc)`, `SetQualityChecker(aiModule.NewQualityChecker())`, `SetPostsSvc(postsSvc)`, `SetPublisherSvc(publisherSvc)`; `TranslationSvc` in Dependencies literal

**Tests (75 tests, all pass):** detect_test.go, localize_test.go, glossary_test.go, score_test.go, service_test.go (pgxmock state-only, jobRow helper — plain strings only in AddRow, never pointers), pipeline_test.go (translate with/without AI, native review determinism, worstSection, parseJSONObject, fallbackSEO, finalScore with nil DB). Notable pgxmock fixes during the run: named-string-type scan (see decision 9), `*string` cell values → plain strings, INSERT arg count (11 not 12 — `$11` reused).

**Validation:** EXECUTED: `go build ./...` (0), `go vet ./...` (0), `go test ./...` — only the 6 known pre-existing `internal/ai` failures (verified unrelated in Sprint 5.6): `TestGeminiProvider_GenerateStream_InvalidKey`, `TestGeminiProvider_Health_InvalidKey` (network), `TestCheckGrammarDetails_Issues`, `TestFindRepeatedWords/multiple_pairs`, `TestScoreReadabilityDetailed`, `TestCountSyllables`. No commit made.

### Sprint 5.8 — AI Research Intelligence (2026-08-05)

**Objective:** Research before article generation. Before any auto-content workflow writes a draft, deep-research the topic across multiple sources (Google search grounding, official sites, docs, articles), fight "surface content" by extracting real data (dates, numbers, versions, companies), rank sources by domain reliability, produce a structured briefing + persisted fact base, cache the result for 24h, and feed it into the generation pipeline AND the Translation Engine (research once → Article PT → Article EN → review → SEO → publish).

**Architecture decisions:**
1. **Deterministic-first, AI-optional everywhere.** Every AI-assisted path (fact extraction, briefing synthesis) computes the deterministic result first and falls back to it whenever AI is nil or returns unparsable JSON (mock provider returns non-JSON `"Mock response for: …"`). No `math/rand`, no package-level mutable state (fact extraction dedupes via a **per-call** `known map[string]bool` — a package-level map was rejected as a race hazard).
2. **Reliability is a score, not a flag.** `internal/ai/reliability.go` owns the deterministic domain ranking (used by the pipeline, so it stays DB-free): allowlist (openai/google/microsoft/nature/science/pnas=100, reuters/apnews/who/un/nih=95, bbc/nytimes/wsj/ft/guardian/wp/who/gov=90, arxiv/ieee/acm/springer=90, statista=80, wikipedia=70, github=75), suffix rules (.gov/.gov.br/.mil/.int=90 official, .edu/.edu.br=75 established), unknown domain=30 (low), empty=0 (unknown). `ExtractDomain` strips scheme/path/query/port + loops prefixes (www./m./blog./news./docs. + language subdomains en./pt./es./de./fr./it./nl./sv./ru./ar./zh./ja./ko.); bare single-label hosts (blog.google→google) get `.com` appended so scoring stays consistent. `ReliabilityOfDomain` also progressively strips leading labels (en.wikipedia.org → wikipedia.org) checking allowlist + suffix rules each iteration. Labels: verified(≥90)/official(≥75)/established(≥55)/low(≥30)/unknown.
3. **Cache is the contract between engines.** Cache key `(site_id, topic_hash, language)`, `topicHash = sha256(lower(topic))[:16]` hex, TTL `cfg.Research.CacheTTL` (`RESEARCH_CACHE_TTL`, default 24h; zero/negative → `DefaultResearchCacheTTL`). `getCached` requires `expires_at > NOW()`, bumps `hit_count` on hit. `GetCachedResearch`/`GetCachedSummary` NEVER trigger a search (translation engine guarantee); `DeepResearchSummary` always runs-or-returns-cache.
4. **runResearch priority chain (pipeline):** preloaded `input.Research` (from cache) → `input.ResearchFn` (injected site-scoped callback into the 4 generation modules) → grounding-only AI fallback which still builds a deterministic `ResearchSummary` via `summaryFromGrounding`. `researchContext(input)` appends the fact base ("use these verified facts") to `runBriefing` and `runDraft` prompt inputs so drafts cannot invent dates/numbers.
5. **Fact extraction** (`research/factbase.go`): deterministic regex extractors — versions `\b(?i:v)?\d+\.\d+(\.\d+)?([-.]?(alpha|beta|rc|stable|latest)\d*)?\b`, prices (US$/R$/$/USD/EUR/BRL + PT words), ISO+long dates (EN+PT month names), numbers with unit words (%/million/billion/users/anos…), event sentences (launch/announcement verbs); hardcoded company term list (openai…nubank/itau/bradesco/caixa) and tech term list (gpt, gemini, claude, llama, llm, api, gpu, rag…). Confidence defaults: version 80, price 70, date 60, company 75, tech 70, number 65, event 70. Snippet preferred over title. AI contract (prompt `fact_base`): JSON array `{type, entity, value, source, confidence}` — invalid types dropped, confidence out of [1,100] → 50, `parseJSONArray` grabs first `[…]` span.
6. **Briefing** (`research/briefing.go`): deterministic `BuildBriefing` fills summary/key_points (top source titles + number facts, cap 12)/statistics/dates/companies/products/data_found; conclusions from corroboration (≥3 sources with IsVerified or reliability ≥75 → corroborated; ≥1 → partial; 0 → unverified warning). AI contract (prompt `deep_research`/`deep_research_pt`): JSON object `{summary, key_points[], data_found[], statistics[], dates[], companies[], products[], conclusions[]}` — empty summary → fallback. `rankSources` sorts reliability desc then relevance desc, never mutates input.
7. **Deep research flow** (`research/deep.go`): cache lookup → `ExecuteGroundedResearch` → `CreateJob` (userID `uuid.Nil`; pt/en only) → `SourcesFromGrounding` → `decorateSources` (domain + reliability) → `rankSources` → persist via `AddSource` (19 params now, skip empty URLs) → `ExtractFactBaseAI` → `saveFactBase` (row per fact) → `BuildBriefingAI` → `persistBriefing` (wraps doc into `ResearchBriefing` via existing `SaveBriefing`) → `saveCache` (UPSERT + refresh expires_at; failures logged, not fatal) → `CompleteJob` (errors ignored) → `DeepResearchReport`. All AI/DB errors abort with wrapped errors except cache write and job completion.
8. **Translation integration:** `translation.Service` gains `researchSvc` + `SetResearchSvc`; `runTranslate` calls `researchContext(ctx, job)` which does a **cache-only** `GetCachedSummary(ctx, job.SiteID, job.Title, job.TargetLanguage)` and appends briefing + "Verified facts:" lines to the translation prompt (deterministic fallback: nil service / cache miss / error → prompt unchanged). Research happens once per topic; both language versions share the same facts.

**New types/constants:** `FactType` enum (company/product/version/price/date/event/technology/number), `DefaultResearchCacheTTL`, `EventResearchCached`, `ErrCacheEntryNotFound`, `ResearchSource.Domain/ReliabilityScore/ReliabilityLabel`, `FactBaseEntry`, `ResearchBriefingDoc` (9 JSON fields), `DeepResearchReport` (embeds ResearchJob + Briefing/Facts/Sources/Cached), `CachedResearch`; ai package: `ResearchFact`, `ResearchSourceSummary`, `ResearchSummary`, `ResearchFn`, `PipelineInput.Research/ResearchFn` (json:"-"), `PipelineResult.Research`, `PromptTypeDeepResearch`/`PromptTypeFactBase` (+ `_pt` variants, 4 new templates in prompt_builder.go — total 24).

**Files changed/created:**
- `migrations/000028_add_research_intelligence.{up,down}.sql` (NEW) — `research_cache` (+ unique index `(site_id, topic_hash, language)`, expires index), `research_fact_base` (+ job/type indexes), `ALTER research_sources ADD domain + reliability_score`, RLS `research_cache_isolation`/`research_fact_base_isolation`
- `internal/ai/reliability.go` (NEW) — scoring map, ExtractDomain, ReliabilityOfDomain, labels
- `internal/ai/pipeline.go` — tri-path runResearch, summaryFromGrounding, formatResearchSummary, researchContext, fact injection in runBriefing/runDraft
- `internal/ai/model.go`, `internal/ai/prompt_builder.go` — new prompt types/templates
- `internal/modules/research/{factbase.go,briefing.go,deep.go}` (NEW) — extraction, briefing, orchestration
- `internal/modules/research/{model.go,service.go,handler.go,module.go}` — types, `cacheTTL`+`SetCacheTTL`, `GetFactBase`, 6 new endpoints
- `internal/api/routes.go` — `POST /research/deep`, `GET /research/deep/{id}`, `GET /research/{id}/facts`, `GET /research/cache/{topic}?language=`, `GET /research/reliability`
- `internal/pkg/config/config.go` — `ResearchConfig{CacheTTL}` + `RESEARCH_CACHE_TTL`; `.env.example` new section
- `internal/modules/{autocontent,contentgenerator,articlepipeline,workflow}/service.go` — `researchFn(siteID)` helper (nil-safe) + `input.ResearchFn` at both input-construction sites (before loop + after rebuild)
- `internal/modules/translation/{service.go,module.go,pipeline.go}`, `cmd/api/main.go` — `SetResearchSvc` + cache-only researchContext in runTranslate

**Tests (70 new, all pass):**
- `internal/ai/reliability_test.go` (6) — ExtractDomain (scheme/prefix/lang/bare-SLD), ReliabilityOfDomain (allowlist/gov/edu/unknown case-fold), labels, determinism, ranking order
- `internal/ai/research_pipeline_test.go` (13) — preloaded Research path, ResearchFn path/error/nil-fallback, priority (preloaded beats fn), summaryFromGrounding (scoring, nil/empty metadata), formatResearchSummary, researchContext, fact injection into Briefing and Draft stages
- `internal/modules/research/factbase_test.go` (20) — versions/prices(EN+PT)/dates(ISO+long+PT)/companies/tech/numbers/events, version dedup, snippet-over-title, deterministic, empty sources, AI fallback (nil manager, unparsable), valid JSON, invalid types dropped, confidence clamp, parseJSONArray, corpus
- `internal/modules/research/briefing_test.go` (14) — sections, 3 conclusion branches, empty, deterministic, AI fallbacks (nil/unparsable/empty-summary), rankSources (order + no mutation), dedupeStrings, parseJSONObject, JSON round-trip
- `internal/modules/research/deep_test.go` (17) — topic hash (case-fold/trim), cache-hit full row, cache-miss full flow (cache→CreateJob→SaveBriefing→GetBriefing→saveCache→CompleteJob incl. GetJob+UPDATE), invalid inputs, no-DB, GetCachedResearch not-found, GetCachedSummary never-triggers-research + entry conversion, DeepResearchSummary non-nil, decorateSources, reportFromCache, formatBriefingDoc, TTL default/set, AI entry conversions, summaryFromReport
- `internal/modules/research/ai_stub_test.go` (NEW helper) — stub AIProvider with fixed Generate content + `newAIManagerReturning`
- `internal/modules/research/service_test.go` — `TestService_AddSource` updated for the new 19-param INSERT (domain + reliability_score)

**Validation:** EXECUTED: `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 26 packages ok; only the 6 known pre-existing `internal/ai` failures (network-only Gemini + Sprint 3.9 grammar/syllable tests): `TestGeminiProvider_GenerateStream_InvalidKey`, `TestGeminiProvider_Health_InvalidKey`, `TestCheckGrammarDetails_Issues`, `TestFindRepeatedWords/multiple_pairs`, `TestScoreReadabilityDetailed`, `TestCountSyllables`. No commit made.

### Sprint 5.9 — E-E-A-T + Linking Intelligence + Rich SEO (2026-08-06)

**Objective:** Before every generated-content publish, (1) score the article across the 4 E-E-A-T pillars with a weighted final AND bilingual explanations for every lost point; (2) auto-insert internal links ranked by keyword + subject/entity + category + intent; (3) auto-attach external links only to high-reliability official/research sources — never competitor domains; (4) compute Topic Authority (does the site already own the topic?); (5) detect Content Gaps against the research fact base, optionally AI-fill them; (6) enrich generated content in the publisher funnel via a fail-open `ContentEnhancer`; (7) public article API emits hreflang + canonical + OG/Twitter Cards + JSON-LD rich snippets (Article/NewsArticle/FAQ/HowTo/Breadcrumb/Organization/WebSite) and a public `GET /schema/site` endpoint.

**Architecture decisions:**
1. **Deterministic-first, AI-optional everywhere.** No `math/rand`, no package-level mutable state. Every AI-assisted path (gap-fill, EEAT explanations) computes a deterministic result first and falls back on nil AI / mock / error.
2. **EEAT is a weighted business score** (`eeat.go`): Experience 0.25 / Expertise 0.30 / Authority 0.25 / Trust 0.20 (asserted to sum to 1.0). `AnalyzeEEAT(in ArticleAnalysisInput) → EEATReport` is DB-free, pure, and honors `AuthorName`/`Category` for signal weights; each pillar records per-signal points + `Missing []string` and a bilingual explanation; constant detectors for Experience (first-person "em nossos testes", "escrevi", comparison words), Expertise (credentials: "especialista", "certificado", qualifications, Google Web/Threat, "foi" (PT) pass) and Authority (`.gov.br`/`.gov` links, authoritative domains, inbound credentials, co-authorship) and Trust (citations/fontes, sources, published date).
3. **`analyzeEEAT` in `analyzer.go` delegates to the module service** (was a heuristic returning fixed points) — full audit now reports `eeat.*` findings via `AuditIssue{Field: "eeat."+pillar}` at `PriorityHigh` when the pillar score < threshold.
4. **Internal linker (`linking.go`)** — `ScoreInternalLinkCandidate` blends keyword (35%) + subject/entity (35%) + category (15%) + intent (15%); intent detected deterministically (`detectIntent`: informational → "como fazer/guia/tutorial/o que é/what is/how to", transactional → "comprar/preço/onde baixar/buy", navigational → brandish). `SelectInternalLinks` reads up to 100 site posts, keeps score ≥ `internalLinkMinScore` (default 40), returns ≤ `internalLinkMax` (default 5), sorted relevance desc.
5. **External links** — `SelectExternalLinks` reads `research_sources` (domain + reliability_score), keeps reliability ≥ `externalLinkMinReliability` (default 75), **excludes competitor domains** (`isCompetitorDomain` — exact or subdomain of each `SEO_COMPETITOR_DOMAINS` entry), sorts reliability desc then relevance, URLs never include `javascript:`; enriched with `ReliabilityLabel` from `ai.ReliabilityOfDomain`.
6. ** Topic Authority** (`topicauthority.go`) — `TopicAuthorityScore(relatedCount)` maps to a log2 curve (1→~18, 3→~40, 10→~68, 25→~85, 50+→100); DB aggregations (`TopicAuthority`, `FillGapsTopContent`) count posts whose title/slug cover the topic terms.
7. **Content gaps** (`contentgap.go`) — 8 deterministic dimensions (price, availability, requirements, limitations, roadmap, comparison, installation, support); each checks the fact base (`GetFactBase` over `research_sources`) and the article text; `DetectContentGaps` returns a coverage ratio + bilingual suggestions (`suggestionForGap`, PT). `FillContentGapsAI` optionally asks the AI provider to draft the missing section (JSON schema enforced; mock/text error → deterministic fallback paragraph), never overwrites the article.
8. **Enhancer in the publisher funnel** (`publisher/enhance.go` + `service.go`) — `ContentEnhancer` interface (`EnhanceBeforePublish(ctx, in) → *ContentEnhancement`); wired only on generated content via `PublishGeneratedArticle` — the single funnel for autocontent, contentgenerator, articlepipeline, workflow (and translation non-user path); it calls `s.contentEnhancer.EnhanceBeforePublish` BEFORE the SEO gate (`checkPublishGate`), so a low SEO score still blocks; NULL/empty result → original content; gate error → fail-open. `seoengine.Service` implements the enhancer: `appendRelatedLinks` appends "### Leia também"/"### Related reading" + a bullet with backlinks; `var _ publisher.ContentEnhancer = (*Service)(nil)`.
9. **Public JSON-rich SEO** (`internal/api/article_seo.go`) — `buildArticleSEO(pub, siteDomain)` builds `PublicArticleSEO{Canonical, Hreflang[], OgType, OgImage, OgLocale, TwitterCard, SchemaJSONLD[], SiteSchemaJSONLD[], StructuralData}`; `deriveSiteDomain` from `pub.URL` or `pub.CanonicalURL` (fallback `https://example.com`). 7 JSON-LD builders use `renderSchema`:
   - Weighted average also correctly reuses the existing `isNewsArticle`/`*Q:/A:`/numbered-line heuristics: FAQ when `Q:/A:` pairs exist; HowTo when "como fazer"/"passo a passo" + numbered steps; Org/WebSite attached to the requested; always present Article. Breadcrumb always present.
10. **Config** — `SEOConfig{CompetitorDomains, InternalLinkMinScore, InternalLinkMax, ExternalLinkMinReliability, MinPublishScore}`; new `.env.example`: `SEO_COMPETITOR_DOMAINS`, `SEO_INTERNAL_LINK_MIN_SCORE` (default 40), `SEO_INTERNAL_LINK_MAX` (5), `SEO_EXTERNAL_LINK_MIN_RELIABILITY` (75).

**New REST endpoints (`internal/modules/seoengine`):**
- `POST /seoengine/analyze-eeat` (AnalyzeEEAT), `POST /seoengine/topic-authority`, `POST /seoengine/internal-links`, `POST /seoengine/external-links` (Reliability), `POST /seoengine/content-gap` (Gap), `POST /seoengine/enhance`.
- `GET /public/schema/site` (in public group, site middleware) returns Org + WebSite JSON-LD.
- Public article GET now enriched with hreflang + canonical + OG/Twitter + 7 JSON-LD blobs (only `article` schemas gated to published + accessible media).

**Publisher `PublishGeneratedArticle` flow (enhancer + gate ordering):**
`verify site → build draft → enhanceContent(draft, kw) → checkPublishGate(enhanced) → repository.SaveGeneratedArticle → PublishArticle(enhanced)` — the SEO gate now sees the ENHANCED content, so a bad (unscored) draft cannot pass the gate by skipping the enhancer.

**Tests (all EXECUTED and passing except pre-existing ai):**
- `internal/modules/seoengine/intelligence_service_test.go` (NEW, 8 tests) — service-level pgxmock: SelectInternalLinks filter-by-score → only ≥40, cap (needs `id <> $2` second arg), no-DB fallback; SelectExternalLinks reliability + competitor + dedupe; TopicAuthority counts related; EnhanceBeforePublish returns unchanged content on nil DB; nil-service no-op.
- `internal/modules/seoengine/intelligence_test.go` (NEW, 21 tests) — AnalyzeEEAT empty/weights-sum-1/deterministic/author-name-boost/authority gov-br/competitor-link detection; Descriptioner; content-gap (full/missing/price regex hurting `R$`/deterministic); ScoreTopicAuthority scale/determinism; category boost; intent detection both languages; `appendRelatedLinks` (PT/EN/empty); all 7 schema builders produce valid JSON + required keys.
- `internal/api/article_seo_test.go` (NEW, 10 tests) — hreflang self + EN, Article/Breadcrumb/FAQ/HowTo schema presence, no-FAQ-without-pairs, OG/Twitter fields, Org+WebSite, `isNewsArticle` PT/EN, extractFAQPairs, extractHowToSteps, deriveSiteDomain, siteNameFromDomain.
- `internal/modules/publisher/gate_test.go` (+6 tests) — fakeEnhancer: nil/no-op, empty content skip, applies result, fail-open on error and on empty result, and PublishGeneratedArticle runs enhancer before DB (so a nil-DB path proves the enhancer ran + the input carries the original draft).
- `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 26 packages ok; still ONLY the 6 known pre-existing `internal/ai` failures (network Gemini + Sprint 3.9 grammar/syllable). No commit made.

### Sprint 5.10 — Freshness Engine + News Intelligence (2026-08-06)

**Objective:** Guarantee Nexora never uses old information to produce news. Before any research: (1) automatically classify intent (NEWS/EVERGREEN/UPDATE/REVIEW/TUTORIAL) which changes the whole research strategy; (2) dynamic temporal windows (NEWS: 24h→7d→max 30d, never >90d; EVERGREEN: date not priority; UPDATE: always latest version — changelog/docs/roadmap); (3) per-source Freshness Score (published, updated, age, category, source); (4) official-source-first priority (official site → docs → official blog → changelog → Reuters/AP/Bloomberg → specialized sites → others); (5) obsolete-information detection (mentions of GPT-4 when GPT-6 exists → Obsolete, never primary source); (6) once-per-day automatic re-evaluation of published articles → mark "Needs Update"; (7) version history (v1/v2/v3, changes, new sources, date); (8) version diffs (price/context/limits/API/benchmark); (9) duplicate-news detection (same subject + same fact + same day → UPDATE existing article instead of creating a new one); (10) independent PT/EN strategies (PT: Brasil→Portugal; EN: EUA/Canadá/Reino Unido/Austrália; never literal-translate a BR news when official EN sources exist).

**Architecture decisions:**
1. **Deterministic-only engines; no AI, no `math/rand`, no network.** All 10 requirements are pure functions (intent.go, score.go, obsolete.go, versions.go, dedup.go) — same input → same output, fully unit-tested. DB is optional (pool nil → DB-free results still returned).
2. **New module `internal/modules/freshness/`** — no changes to existing modules; endpoints under `/api/v1/freshness` registered after translation routes; `FreshnessSvc` in Dependencies; module in kernel list (registered last).
3. **Intent classifier** (`intent.go`): per-language cue tables (PT/EN regex + weights); version-token signal (`versionSignal`, e.g. "GPT-6", "Gemini 2.5", "v4.1") adds an UPDATE boost; fallback EVERGREEN with `fallback_evergreen` signal; confidence = winner/(winner+runner), clamped [0.5,1].
4. **Temporal windows** (`model.go WindowStrategy`): NEWS {Priority 1, Recent 7, Max 30, Never>90}; UPDATE {Recent 30, Max 90, VersionFirst}; REVIEW {Recent 30, Max 90, Never>365}; TUTORIAL {Recent 90, no max}; EVERGREEN {no date priority}. Config overrides `FRESHNESS_NEWS_MAX_DAYS`/`FRESHNESS_NEWS_NEVER_OLDER_DAYS` applied via `Service.windowFor`.
5. **Freshness Score** (`score.go`): weighted 0.50×age + 0.30×update + 0.20×source, clamped [0,100]; age component linear-decays across the window (unlimited windows use tiered flat values: ≤2y 85, ≤10y 70, else 45); update component (≤1d 95 … ≥90d 60, nil=50 neutral); source component from priority tier (official 100, agency 98, docs/changelog 95, blog 92, specialized 80, other 60, unknown 40). Outside-window sources are `Usable=false` with -40 penalty. "Publicado há 2 dias + atualizado ontem" lands ~95 (≈ the spec's 98).
6. **Official-first priority** (`score.go SourcePriorityClassify`): marker substring check (changelog/release notes → docs → blog), news-agency allowlist (reuters/apnews/ap.org/bloomberg/afp/france24/ft), specialized allowlist (theverge/techcrunch/engadget/wired/techradar), then target-entity official match (registrable domain OR first hostname label equals the derived entity, e.g. `gemini.google.com` + entity "gemini" → official). `SortSourcesByPriorityAndScore` sorts tier → score, obsolete last.
7. **Obsolete detection** (`obsolete.go`): `tokenRE` captures version-ish tokens (`[a-z0-9]{2,}[\s\-]?(v?\d+(?:\.\d+){0,2})` — NOTE `{0,2}` not `{1,2}`: the `{1,2}` form fails on plain "GPT-4" because RE2 requires a decimal part); `CheckObsolete` matches entity brand adjacency and compares versions (2.5 < 6). Obsolete sources are flagged and rank last (never primary). `DetectObsoleteSources` runs over a list of `EntityVersion{Entity, Current}`.
8. **Version history + diffs** (`versions.go`): `NextVersion` bumps v1.2→v1.3; `DiffVersions` compares 6 facets (price/context/limits/api/benchmark/features) via keyword line extraction; `Fingerprint` = FNV-64a hex of normalized topic+first-sentence (stable); `TokenJaccard` near-duplicate similarity; persisted in `article_versions` (changes/diff/sources as JSONB).
9. **Dedupe / update-not-duplicate** (`dedup.go`): `CheckDuplicate` — duplicate = token overlap ≥0.6 AND same calendar day; on match the caller updates the existing article. `CheckDuplicate` (service) registers fingerprints in `news_dedup` (idempotent: no insert on duplicate). `SourceMatchesRegion` enforces PT (.br/.pt) vs EN (.com/.ca/.co.uk/.com.au) region rules; `shouldBlockTranslation` implements "never literal-translate a BR news when official EN sources exist".
10. **Daily sweep** (`RunDailySweep` + `ReEvaluateArticleWithWindow`): once-per-day guard via `freshness_sweeps` row (skipped same-day; disabled via `FRESHNESS_SWEEP_ENABLED=false`); each published article is re-evaluated — outside its temporal window OR a newer `news_dedup` for the same topic → `content_updates` row with status `needs_update`, reason `outside_temporal_window`/`newer_source_found`, old/new scores, details JSONB.

**Migration `000029_add_freshness_tables.{up,down}.sql`** (NEW): 6 tables — `news_intents` (intent cache, unique (site_id, topic_hash, language)), `source_freshness_scores` (per-source scores, index on research_job_id), `article_versions` (version history), `news_dedup` (unique (site_id, fingerprint), created_on date), `content_updates` (Needs Update flags), `freshness_sweeps` (once-per-day guard, site_id PK). All RLS `{table}_isolation USING (site_id = current_setting('app.current_site_id')::UUID)`; `update_updated_at_column()` triggers guarded by function existence; down drops triggers → policies → disables RLS → tables.

**New REST endpoints (`/api/v1/freshness`, all behind site middleware):**
| Method | Path | Purpose |
|---|---|---|
| POST | `/freshness/classify` | Intent classification + temporal window (cached in news_intents) |
| POST | `/freshness/score` | Per-source freshness scoring + obsolete flags (persisted) |
| POST | `/freshness/obsolete` | Outdated entity-version detection on a text |
| POST | `/freshness/versions` | Save a version record |
| GET | `/freshness/versions/{publicationID}` | Version history |
| POST | `/freshness/dedup` | Duplicate check + fingerprint registration |
| POST | `/freshness/sweep` | Once-per-day re-evaluation (409 when already run) |
| GET | `/freshness/updates` | Needs Update flags |

**Config:** `FreshnessConfig{SweepEnabled (default true), NewsMaxDays (30), NewsNeverOlderDays (90)}` in `internal/pkg/config/config.go`; env `FRESHNESS_SWEEP_ENABLED`, `FRESHNESS_NEWS_MAX_DAYS`, `FRESHNESS_NEWS_NEVER_OLDER_DAYS` documented in `.env.example` under a new `# === Freshness Engine + News Intelligence ===` section.

**Wiring:** `internal/api/routes.go` — `freshnessModule` import, `Dependencies.FreshnessSvc`, `registerFreshnessRoutes` (after translation, before AI routes); `cmd/api/main.go` — `freshnessMod` created with `(cfg, log, db)` (no cache — module doesn't need it), added to the kernel module list, `freshnessSvc := freshnessMod.Service()`, `SetEventBus`, `FreshnessSvc: freshnessSvc` in the Dependencies literal.

**Tests (61 tests, all EXECUTED and passing):**
- `internal/modules/freshness/engine_test.go` (38) — windows (news 1/7/30/90, evergreen unlimited, update version-first); classifier (news PT/EN, evergreen docs + fallback, update + version-token boost, review, tutorial EN, invalid language, empty input, confidence bounds); freshness scoring (fresh news ~high score + changelog priority, 45d outside window unusable, 120d never-usable + penalized, evergreen 2y usable + moderate, official/blog/docs/agency/specialized priorities, obsolete-last sort, brand-domain official via first label); obsolete (GPT-4 vs current 6, current not obsolete, no-match, multi-entity, newer-not-obsolete, version compare); versions (NextVersion, DiffVersions change/no-change, Fingerprint stability, CheckDuplicate same-day match + no-match, ReEvaluateArticle needs-update/newer-source/fresh, DailySweepOnce guard); language (PT/EN regions, region-domain matching, block-literal-translation); DeriveMainEntity.
- `internal/modules/freshness/service_test.go` (15) — pgxmock: ClassifyAndWindow caches news_intents (update intent + version-first window), DB-free classify, ScoreSources persists + usable + agency priority, obsolete marked + reasons, empty-sources error, SaveVersion+ListVersions round-trip (JSONB decode), CheckDuplicate registers (no match) + finds same-day match (no insert), RunDailySweep marks needs_update + records guard, same-day block, disabled sweep, ListUpdates decode (typed strings scanned as plain strings then converted — pgxmock named-string-type quirk, same pattern as translation Sprint 5.7).
- `internal/modules/freshness/handler_test.go` (8) — classify/score/obsolete/dedup endpoints (200 + intent/has_obsolete assertions), missing-site → 400 MISSING_SITE, rest.AdaptHandler contract.
- `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 29 packages ok; still ONLY the 6 known pre-existing `internal/ai` failures (network Gemini + Sprint 3.9 grammar/syllable). No commit made.

**Open notes:**
- The intent classifier is consumed via the freshness API/service; wiring it *automatically before* the research module's `DeepResearchSummary`/`ExecuteGroundedResearch` calls is a cross-module hook (research does not import freshness yet) — the endpoints exist for callers to classify first and pass `FRESHNESS_NEWS_*`-aware windows. Deferred as an explicit integration to avoid touching the research module's test surface.
- Migration numbering: 000029 is the first migration after the pre-existing duplicate-version pair 000026 (session_metadata + post_seo_columns — golang-migrate treats both as version 26; known pre-existing issue, untouched).

### Sprint 5.11 — AI Editorial Brain (2026-08-06)

**Objective:** Before writing, the module produces an intelligent Editorial Brief (search intent, reader persona, article angle, complete outline, required questions). Before publishing, it runs a full editorial note (coverage, fluency, evidence, per-block confidence, semantic SEO, weighted final score) and auto-returns the article to review when the final score is below the threshold (default 90). Publisher gate is fail-open (no review/error → publish allowed).

**Architecture decisions:**
1. **Deterministic-only engines; no AI, no `math/rand`, no network.** All scoring is pure and unit-tested. AI (seoengine.AnalyzeArticle/AnalyzeEEAT via injected `ai.QualityChecker`) is optional and nil-safe.
2. **Per-language cue gating** (intent.go/persona.go): cue tables are `map[intent]map[lang][]signal` (pt/en); only the topic language's cues fire (fixes brand-word double counting, e.g. "gemini" in both PT+EN tables). `pre()` uses leading-boundary-only matching so plurals ("criadores") match singular cues ("criador"). Each cue group is one weighted signal; several groups accumulate (e.g. "comparação" + " vs " = 3.6).
3. **Confidence** = winner/(winner+runner) clamped [0.5,1] (0.8 when only one intent fires). Tie-break precedence: breaking_news > tutorial > comparison > commercial > navigational > update > informational (personas: developer > business > creator > general). Version tokens (`GPT-6`, `Gemini 2.5` via `versionTokenRE`) add +0.6 to UPDATE.
4. **Final score weights (sum 1.0):** SEO 0.20, EEAT 0.20, Freshness 0.15, Coverage 0.20, Naturalness 0.15, Confidence 0.10. `Final < threshold` → `needs_review`; threshold ≤0 → default 90.
5. **Evidence is claim-first:** `LinkEvidence(text, facts, sources, language)` marks important claims (numeric token OR claim word OR >18 tokens), matches against the fact/source corpus via full-token Jaccard (≥0.5), confidence 100/90/80/70 by source score (≥90 official note), 45 unverified. Evidence score = avg claim confidence (100 when no claims). Evidence flows to `ScoreBlocks` per block (score = 40 + verified×15 + official×10, +5 single block).
6. **pgxmock v3.4.0 gotchas (test-only):** `val.Type().AssignableTo(destVal.Elem().Type())` — row cells must use the exact named types (`SearchIntent("tutorial")`, `Persona("developer")`, `DecisionNeedsReview`) and exact numeric kinds (`float64(90)` for score columns, never int); `connRow.Scan` (QueryRow path) swallows scan errors silently, leaving zero fields (typed cells are mandatory); audit `INSERT INTO audit_log` args are `*uuid.UUID`/`*string` pointers — use `pgxmock.AnyArg()` for UserID/SiteID/EntityID/IPAddress.
7. **Publisher gate ordering:** `checkEditorialGate` runs AFTER `checkPublishGate` in both publish paths; missing review (content_hash miss) → gate reports 100 (fail-open). `ErrEditorialScoreBelowMinimum` → 422 `EDITORIAL_SCORE_BELOW_MINIMUM`.

**Migration `000030_add_editorial_brain_tables.{up,down}.sql`** (NEW): 4 tables — `editorial_briefs` (unique (site_id, topic_hash, language)), `editorial_reviews` (content_hash, per-dimension NUMERIC(5,2) score columns, decision, threshold, coverage/fluency/semantic JSONB), `editorial_block_scores` (review_id CASCADE), `editorial_evidence` (claim/verified/source_title/source_url/confidence/note). RLS `{table}_isolation USING (site_id = current_setting('app.current_site_id')::UUID)`; updated_at trigger on briefs.

**New module `internal/modules/editorialbrain/`** — model.go (SearchIntent/Persona/SectionType/QuestionID/FacetID enums, bi-bilingual structs, EditorialScore/EditorialBrief/EditorialReview, 3 events, 7 sentinels), helpers.go (fnv64a 16-hex contentHash/topicHash, tokenize w/ accent runes, sentences, Jaccard termOverlap, stopWords, `b()` constructor), intent.go, persona.go, outline.go, questions.go, coverage.go (8 facets), fluency.go (shingle 4-gram + word-repeat + PT/EN passive regexes + 150-word paragraphs + readability 35/25/15/15/10 blend), evidence.go, blocks.go, semantic.go (entity/concept/terms/faq/synonym blend), score.go, service.go (BuildBrief/ReviewArticle/CheckEditorialScore/Get/List + `loadResearch` via optional ResearchProvider), repo.go (saveBrief upsert, saveReview + block/evidence rows), handler.go (15 endpoints), module.go (`Name()="editorialbrain"`), engine_test.go (40) + service_test.go (14) + handler_test.go (15) = 69 tests.

**New REST endpoints (`/api/v1/editorialbrain`, all behind site middleware):**
| Method | Path | Purpose |
|---|---|---|
| POST | `/editorialbrain/intent` | Search intent classification (+confidence/signals) |
| POST | `/editorialbrain/persona` | Reader persona detection |
| POST | `/editorialbrain/outline` | Outline + suggested title + FAQs + table/callout flags |
| POST | `/editorialbrain/questions` | Required question list (or verification when `text` given) |
| POST | `/editorialbrain/coverage` | 8-facet coverage + bilingual missing list |
| POST | `/editorialbrain/fluency` | Reading fluency report |
| POST | `/editorialbrain/evidence` | Claim→research evidence links (optional `research_job_id`) |
| POST | `/editorialbrain/semantic` | Entities/concepts/missing-terms/FAQ/synonym check |
| POST | `/editorialbrain/score` | Final weighted note from components |
| POST | `/editorialbrain/brief` | Build + persist editorial brief |
| GET | `/editorialbrain/briefs` | List briefs (limit/offset) |
| GET | `/editorialbrain/brief/{id}` | Brief detail |
| POST | `/editorialbrain/review` | Full editorial review + gate decision |
| GET | `/editorialbrain/reviews` | List reviews (?decision=approved\|needs_review) |
| GET | `/editorialbrain/review/{id}` | Review detail incl. blocks + evidence |

**Config:** `EditorialConfig{MinFinalScore}` + env `EDITORIAL_MIN_FINAL_SCORE` (default 90, via getEnvFloat); `.env.example` "=== AI Editorial Brain ===" block before "=== Modo de Desenvolvimento ===".

**Wiring:** routes.go `Dependencies.EditorialBrainSvc` + `registerEditorialBrainRoutes` (after freshness, nil-safe); main.go registers `editorialBrainMod` last in the kernel module list, `SetQualityChecker(aiModule.NewQualityChecker())`, `SetResearchProvider(editorialResearchAdapter{svc: researchSvc})` (maps GetFactBase→FactEntry, GetCachedResearch→SourceRef with Snippet=s.Summary), `publisherSvc.SetEditorialGate(editorialBrainSvc)`.

**Fixes during implementation:** kernel import path is `nexora/internal/kernel` (NOT internal/pkg/kernel); `rest.Context` has no QueryInt (custom `queryInt`); HTML heading regex cannot use `\1` backreferences (RE2) — uses `</h[1-6]>`; tie-break test now a true tie ("comparação e preço e comprar" = 3.6 vs 3.6); evidence verified test claim matches source tokens exactly (full-token Jaccard ≥0.5).

**Validation:** EXECUTED: `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 30 packages ok; still ONLY the 6 known pre-existing `internal/ai` failures (network Gemini ×2 + Sprint 3.9 grammar/syllable ×4). No commit made.

### Sprint 5.12 — Editorial Pipeline Board + ISR Revalidation (2026-08-06)

**Objective:** (1) Editorial pipeline control plane: a board that tracks every content item across 11 editorial stages, per-stage stats, an article review screen with the deterministic "IA recomenda" block, and a fail-open publish-readiness checklist; (2) JIT publishing foundation: `posts.scheduled_at` + draw a live ISR revalidate trigger (backend → Next.js).

**Architecture decisions:**
1. **Read-only UNION pipeline.** `pipelineUnionSQL` merges 11 sources (briefs, research_jobs, 4 generation engines, translation, posts seo/eeat/review/approval/scheduled/published) into one column shape ordered by `updated_at DESC` with `LIMIT $2` (default 200). Every branch filters `site_id = $1`. Post rows appear in exactly one stage via branch precedence (published > scheduled > approval > review > eeat > seo). No writes, no triggers — board reads stay side-effect free.
2. **Fail-open readiness checklist** (`GetPublishReadiness`): pipeline→SEO→EEAT→Freshness→Editorial note; missing editorial review NEVER blocks a manual publish; blocking stage name returned for the UI dialog. Thresholds come from `cfg.SEO.MinPublishScore` / `cfg.Editorial.MinFinalScore`.
3. **Fail-open link suggestor.** `LinkSuggestor` interface (SelectInternalLinks/SelectExternalLinks) wrapped; seoengine implements it in `cmd/api/main.go` (`editorialSvc.SetLinkSuggestor(seoengineSvc)`).
4. **Deterministic recommendations.** "IA recomenda" block derives statuses (ok/warning/info/fail) purely from scores, problems and source verification ratio. Problems bucket: seo/coverage/fluency/semantic/sources/evidence/editorial.
5. **ISR revalidation is fail-open by design** (never blocks publish): `internal/pkg/revalidate` POSTs `{base}/api/revalidate` with `x-revalidate-token`; a per-site failure is logged, error ONLY when every site fails. Subscriber in `cmd/api/main.go` on `EventPubCachePurge` → go-around.
6. **`posts.scheduled_at` already existed** (migration 000005) — migration **000031** only adds the missing partial index `idx_posts_site_scheduled ON posts(site_id, scheduled_at) WHERE status='scheduled' AND deleted_at IS NULL`. `SetStatus` now writes `scheduled_at` via new `SetStatusWithSchedule(ctx, siteID, postID, status, scheduledAt *time.Time)` (same signature as `SetStatus` kept; `SetStatusRequest.scheduled_at` wired in the handler).

**New files:**
- `internal/modules/editorial/pipeline.go` (NEW) — pipeline + stats + readiness + article-review composition: `GetPipeline`, `GetPipelineStats` (11 counts + review-score averages + published-this-week), `GetPublishReadiness`, `GetArticleReview` (post + latest review + JSONB details + sources + unverified evidence + stored `posts.seo_issues` + deterministic problems/recommendations + link suggestions), `LinkSuggestor` interface, `deriveKeyword` (4+ chars, non-stopword, longest), bilingual problem messages.
- `internal/modules/editorial/model.go` — pipeline types (`PipelineStage` constants + `PipelineStageOrder`, `PipelineItem`, `StageCount`, `PipelineResponse`, `PipelineStats`, `ReviewPost/Scores/Source/Link/Problem/Recommendation/ReadinessCheck/PublishReadiness/ArticleReview`), `ErrPostNotFound`, `ErrDatabaseNotAvail`.
- `internal/modules/editorial/service.go` — Service gains `cfg *config.Config` and `linkSuggestor`, `SetLinkSuggestor`.
- `internal/modules/editorial/handler.go` — new endpoints `GET /editorial/pipeline`, `GET /editorial/pipeline/stats`, `GET /editorial/review/{id}`, `GET /editorial/publish-readiness/{id}`.
- `internal/api/routes.go` — the 4 new editorial routes.
- `internal/pkg/revalidate/revalidate.go` (NEW) — `Client.New(urls, token, enabled, timeout, log)`, `Enabled()`, `Revalidate(ctx, slug)` with `TokenHeader = "x-revalidate-token"`; per-site collect + all-fail error.
- `internal/pkg/config/config.go` — `RevalidateConfig{PublicURLs []string, Token string, Enabled bool, Timeout time.Duration}` loaded from `SITE_PUBLIC_URLS` (falls back to `SITE_PUBLIC_URL`), `SITE_REVALIDATE_TOKEN`, `SITE_REVALIDATION_ENABLED` (default true), `SITE_REVALIDATE_TIMEOUT` (default 5s); new helpers `firstNonEmpty`, `splitCSV`.
- `migrations/000031_add_post_scheduled_index.{up,down}.sql` (NEW) — partial index on posts(site_id, scheduled_at).
- `.env.example` — `# === Revalidação ISR (Next.js) ===` block.
- `cmd/api/main.go` — revalidate Client + EventBus subscriber (EventPubCachePurge → go-revalidate), `editorialSvc.SetLinkSuggestor(seoengineSvc)`.
- `site/lib/api.ts` — fetch now sets `next: { revalidate: 60, tags: ["articles"] }`.
- `site/app/api/revalidate/route.ts` (NEW) — POST handler: token header check (missing/mismatch → 401), slug validation (400), `revalidatePath("/")` + `revalidatePath("/"+slug)` + `revalidateTag("articles")`, `{revalidated: true, slug}`.
- `site/next.config.mjs` — proxy rewrites use `API_PROXY_TARGET` (fallback `http://localhost:8080`).
- `deploy/docker-compose.yml` — site service env `NEXT_PUBLIC_API_URL: http://api:8080`, `API_PROXY_TARGET: http://api:8080`.
- `site/.env.local.example` — `SITE_REVALIDATE_TOKEN=`, `API_PROXY_TARGET=http://localhost:8080`.

**Frontend (Admin SPA) — Editorial + Dashboard:**
- `web/src/lib/editorial.ts` (NEW) — TS types mirroring Go JSON (PipelineStage union, `PIPELINE_STAGES`, `STAGE_LABELS` pt, `STAGE_COLORS` dot+card), `PipelineItem/Response/Stats`, `ArticleReview`, `ApprovalRequest`, `ReadinessCheck`, `PublishReadiness`, Review types.
- `web/src/components/Sidebar.tsx` — new "Editorial" nav item wired to `/admin/editorial`.
- `web/src/App.tsx` — routes `editorial` + `editorial/review/:id` under `/admin`.
- `web/src/pages/editorial/Dashboard.tsx` (NEW) — 3 tabs (Pipeline/Revisões/Estatísticas), 11-column drag-and-drop board (native HTML5 dnd, `dataTransfer` JSON), PipelineCard with StageBadge/ScorePill + "Abrir" link (review/approval stage) + "Mover…" Select (sch/to/published), `PATCH /posts/{id}/status` (+ scheduled_at via `editorial-schedule-date`), `POST /publisher/publish` with 422 → readiness Dialog (`/editorial/publish-readiness/{id}`). Queries site-scoped (`siteQueryKey(["editorial-pipeline"], siteId)` etc.) with `enabled: !!currentSiteId`; mutations invalidate pipeline/stats/dashboard prefixes.
- `web/src/pages/editorial/Review.tsx` (NEW) — header with StatusBadge + actions (Rascunho/Agendar/Solicitar aprovação/Aprovar via `PUT /editorial/posts/{id}/approvals/{id}/review`/Publicar w/ 422 readiness dialog), score grid (SEO/EEAT/Freshness/Cobertura/Naturalidade/Confiança/Final + threshold), "IA recomenda" RecommendationBlock, ProblemsBlock (severity colors), SourcesBlock, LinksBlock, approvals (client-side post filter because ListApprovals handler doesn't filter).

**Tests added (all EXECUTED and passing):**
- `internal/modules/editorial/pipeline_test.go` (13 tests): GetPipeline rows/stages/no-DB, GetPipelineStats full + empty averages, GetPublishReadiness blocked-seo/ready/fail-open, GetArticleReview no-review (sources problem + readiness) / with-review (all problem kinds + recommendations statuses + editorial blocking) / not-found, `loadReviewEvidence` isolated, `deriveKeyword`. (pgxmock notes: cells must be pointer-typed for nullable dests → scan-silent swallowing keeps pointer fields zero; QueryRow swallows scan errors — all cells typed.)
- `internal/pkg/revalidate/revalidate_test.go` (9 tests): disabled flag/no-config/empty-slug no-op, success asserts token header + slug + path normalization, fail-open partial, all-fail error, network error message, context cancellation.
- `internal/modules/posts/service_test.go` — `TestService_SetStatusScheduledAt` asserts the exact `scheduled_at` argument; the 2 existing SetStatus expectations updated (`scheduled_at = $5`, `pgxmock.AnyArg()`).
- `web/src/__tests__/editorial.test.tsx` (NEW, 6 tests): pipeline/review keys `["editorial-pipeline", siteId]`, `["editorial-review", siteId, id]` site-scoped; NO_SITE_KEY registration without execution; board renders title/tabs/"Abrir"; review renders title + decision badge; cache isolation.
- `web/vitest.config.ts` — testTimeout 30000 / hookTimeout 20000 (Protectorled18s under full parallel load).

**Validation:** EXECUTED: `go build ./...` (0), `go vet ./...` (0), `go test ./...` — 26 packages ok (6 known pre-existing `internal/ai` failures: network Gemini ×2 + Sprint 3.9 grammar/syllable ×4); `npx tsc -b` (0), `npx vitest run` = **12 files, 95/95 pass**, `npm run lint` (0 errors, 45 pre-existing warnings), `npm run build` (0, pre-existing chunk warning). No commit made.

### Sprint 6.1 — Railway 404 na raiz: serviço único (API + Admin SPA) (2026-08-07)

**Problema:** No Railway, `GET /` respondia 404. Causa raiz: a API Go era o único serviço deployado e o chi router **não tinha nenhuma rota na raiz** (nem `/` nem `/admin`). O frontend admin (`web/`) existe no repo, mas o Dockerfile de produção (`deploy/Dockerfile`) só compilava os binários Go — `deploy/Dockerfile.admin` e `deploy/Dockerfile.site` são **só dev** (`npm run dev`, sem build/serve) e nunca entraram no Railway. Nixpacks (Go) tampouco serviria o SPA.

**Arquitetura escolhida: um único serviço.** A API embute o Admin SPA compilado no binário via `go:embed` e serve tudo na mesma origem: `GET /` → painel, `/admin*` → fallback SPA para `index.html`, `/api/v1/*` inalterado, `/ping` intacto. Sem CORS, sem proxy, sem segundo serviço. O client admin já usa caminhos relativos (`API_BASE = "/api/v1"`), então nenhuma `VITE_API_URL`/`NEXT_PUBLIC_API_URL` é necessária em produção.

**Novo pacote `internal/webui/`:**
- `webui.go` — `//go:embed all:dist`; `SPAHandler()` = `NewSPAHandler(mustSub(distFS, "dist"))`; handler: GET/HEAD apenas (405 com JSON), path `api*` → JSON 404 (nunca cai no SPA), arquivo real ou fallback `index.html` (history-API), Content-Type por extensão, `Cache-Control: public, max-age=31536000, immutable` para `/assets/*`, `no-cache` no resto
- `dist/.gitkeep` (commitado) — garante `go build`/`go test`/CI sem build Node; o Docker injeta os arquivos reais antes do `go build`
- `webui_test.go` (7 testes, embed de `testdata/dist`) — raiz serve index, fallback SPA para `/admin`/`/admin/login`/deep paths, asset com cache header + JS mime, `/api/*` nunca cai no SPA (JSON NOT_FOUND), 405 para POST/PUT/DELETE, HEAD sem corpo, FS vazio → 404
- **Nota chi:** paths desconhecidos DENTRO de `/api/v1` são respondidos pelo NotFound do subrouter (404 texto puro, comportamento pré-existente); o handler SPA só recebe `/api/*` fora do grupo

**Mudanças:**
- `internal/api/routes.go` — `router.Handle("/*", webui.SPAHandler())` após o grupo `/api/v1` (última rota; `/api/v1/*` e `/ping` mantêm precedência); import `nexora/internal/webui`
- `Dockerfile` (NOVO, raiz) — canônico para Railway: stage `web-builder` (node:22-alpine, `npm ci` + `npm run build` em `web/`), stage `builder` (Go 1.26; `COPY --from=web-builder /build/web/dist ./internal/webui/dist/` ANTES do `go build`, mantém o .gitkeep), stages `dev` (air, para compose) e `prod` (alpine + binaries + migrations)
- `deploy/docker-compose.yml` — serviços `api` e `migrate` apontam para o `Dockerfile` raiz (target dev)
- `deploy/Dockerfile` — REMOVIDO (substituído pelo Dockerfile raiz; Railway detecta automaticamente)
- `.gitignore` — negação `!internal/webui/dist/` + `!internal/webui/dist/.gitkeep` (o padrão global `dist/` ignoraria o placeholder do embed)
- `web/public/favicon.svg` (NOVO) — o index.html referenciava `/favicon.svg` sem arquivo (public/ não existia)
- `web/.env.example` — documentado: produção = mesma origem, sem variável de API URL
- `README.md` — seção "Deploy no Railway (um único serviço)": builder Dockerfile, variáveis de ambiente (backend NÃO lê `DATABASE_URL` — usar `DATABASE_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`; `SERVER_PORT` tem prioridade, senão `PORT` do Railway), passo 1: `POST /api/v1/setup/install` antes do primeiro login, rotas esperadas

**Validação:** EXECUTADO: `npm run build` em `web/` (0, warning de chunk >500kB pré-existente); `go build ./...` (0); `go vet ./...` (0); `go test ./internal/webui/... ./internal/api/...` (0); smoke test real: boot do binário com o dist embutido → `GET /` 200 text/html, `GET /admin` 200, `GET /admin/login` serve index.html do React, `GET /api/v1/health` JSON ok (db conectado), `GET /assets/index-*.js` 200 com `Cache-Control immutable` + JS mime. Nenhum commit feito. (gofmt: arquivos novos formatados; repo tem muitos pré-existentes fora do gofmt — não tocados.)

**Passos restantes no Railway (documentados no README):** trocar builder do serviço para Dockerfile (ou limpar "Dockerfile Path" se apontava para deploy/Dockerfile), garantir env vars, rodar setup/install, acessar `https://SEU-SERVICO.up.railway.app/admin`. Site público Next.js (`site/`) permanece serviço separado opcional (Vercel via vercel.json ou 2º serviço Railway).
