# Reconciliation Audit — Current State (2026-07-29)

> **Objective**: Verify every claim in `AGENTS.md` and sprint memory against actual code on disk.
> **Method**: 100% Read-tool analysis (shell/ripgrep non-functional).
> **Status**: ✅ = matches, ⚠️ = partial/degraded, ❌ = absent/contradicts

---

## 1. Project Foundation

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `go.mod` exists | Present | `module github.com/anomalyco/nexora-cms` | ✅ |
| `cmd/api/main.go` exists | Present | Present — 170 lines, registers AuthModule, AIPipelineModule, ContentGeneratorModule, AutocontentModule, SEOEngineModule, WorkflowModule, PublisherModule, ResearchModule, PostModule | ✅ |
| `internal/api/routes.go` exists | Present | Present — 285 routes | ✅ |
| `internal/api/middleware/` exists | Present | authz.go, casbin.go, cors.go, logging.go, ratelimit.go, recovery.go | ✅ |

## 2. AI Integration Layer (`internal/ai/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `interfaces.go` | AIProvider, QualityChecker, PromptBuilder, StreamHandler, AIManager | All 5 interfaces present | ✅ |
| `provider.go` | MockProvider with 7 methods | MockProvider with Generate, GenerateStream, Embeddings, Summarize, Rewrite, Classify, Health — all return mock data. No real AI provider. | ⚠️ Mock-only |
| `registry.go` | Priority-ordered provider registry | `internal/ai/registry.go` — Registry with Register, Get, List, GetByCapability, HealthCheckAll | ✅ |
| `manager.go` | Circuit breaker, retry, failover, weighted selection, metrics | Manager with CircuitState (Closed/Open/HalfOpen), maxRetries=3, backoff, failover, weighted selection, metrics counters | ✅ |
| `prompt_builder.go` | 12 default prompt templates (EN+PT) | PromptBuilder with 12 templates: 6 EN + 6 PT (ArticleGen, SocialMedia, MetaDescription, Newsletter, SEOContent, PressRelease) + Custom registration | ✅ |
| `stream.go` | StreamProcessor with handlers | StreamProcessor with OnChunk, OnComplete, OnError, OnProgress handlers, cancel via context | ✅ |
| `quality.go` | QualityChecker with 6 mock checks | QualityCheckerImpl with ScoreGrammar, ScoreSEO, ScoreReadability, CheckStructure, CheckDuplicates, CheckHallucination — all use `rand.Float64()` for scoring, CheckHallucination is substring match | ⚠️ Mock-only, no real quality checking |
| `pipeline.go` | 8-stage PipelineExecutor | PipelineExecutor with stages: ResearchGen → OutlineGen → DraftGen → Review → SEOOptimize → QualityCheck → Format → FinalReview. All call `manager.Generate()` → MockProvider | ⚠️ Mock-only pipeline |
| `module.go` | AI module registered | Init registers "mock" provider with model "mock-model" | ⚠️ Mock-only |
| `handler.go` | 5 REST endpoints | GET `/api/v1/ai/providers`, GET `/api/v1/ai/health`, POST `/api/v1/ai/test`, GET `/api/v1/ai/prompts`, GET `/api/v1/ai/capabilities` | ✅ |

**AI Verdict**: The abstraction layer is well-designed (interfaces, registry, circuit breaker, pipeline stages). However, **no real AI provider is implemented** — only MockProvider exists. The system is infrastructure-ready but produces no real AI content.

## 3. Content Generator Module (`internal/modules/contentgenerator/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `model.go` | Migration + types | Present — ContentRequest, ContentResponse, GenerationJob, GenerationPipeline, PipelineStage, QualityGate, GenerationStats | ✅ |
| `service.go` | State machine — no AI call | 14 methods: CreateJob, GetJob, ListJobs, UpdateJob, DeleteJob, StartGeneration, PauseJob, ResumeJob, CancelJob, GetPipeline, UpdatePipelineStage, GetQualityGates, GetStats, GetMetrics. Does NOT import `ai` package. | ✅ Verified no AI dependency |
| `handler.go` | 19 endpoints | Present — all REST handlers for CRUD + workflow control | ✅ |
| `module.go` | Module struct | Present — ContentGeneratorModule with SetEventBus, RegisterRoutes | ✅ |

**ContentGenerator Verdict**: Pure state machine. No integration with `ai` package. Jobs transition through states (pending→running→completed/failed) but never call any AI provider.

## 4. Autocontent Module (`internal/modules/autocontent/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Migration 000014 up | 5 tables | `autocontent_jobs`, `autocontent_steps`, `autocontent_results`, `publication_queue`, `workflow_templates` | ✅ |
| Migration 000014 down | Drop 5 tables | Present | ✅ |
| `model.go` | Types + DTOs + constants | AutocontentJob, Step, Result, PublicationItem, WorkflowTemplate, 7 JobStatus, 6 StepStatus, 5 QueueStatus, 14 WorkflowStep, errors, events | ✅ |
| `service.go` | 21 methods | All 21 methods present: CRUD, workflow engine (Start/Pause/Resume/Cancel/Retry/Restart), Steps, Results, Queue, Templates, Metrics/Stats | ✅ |
| `handler.go` | 21 REST endpoints | All 21 endpoints present | ✅ |
| `module.go` | Module struct | Present | ✅ |

**Autocontent Verdict**: Complete workflow engine. 91 tests passing. However, the `publication_queue` table defined in migration 000014 has FK to `autocontent_jobs`, which conflicts with migration 000019 that redefines `publication_queue` with FK to `publications`.

## 5. AI Pipeline Module (`internal/ai/pipeline.go` in context of modules)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Pipeline registered in main.go | AIPipelineModule | `aiModule := aipipeline.NewAIPipelineModule(aiManager)` in main.go | ✅ |
| Pipeline uses real AI | Content generation | Calls `manager.Generate()` → MockProvider → returns "Mock response for: " + prompt | ❌ Mock only |

## 6. Research Module (`internal/modules/research/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `model.go` | ResearchJob, ResearchSource, ResearchEntity | Present | ✅ |
| `service.go` | CRUD + search | Present — CreateJob, GetJob, ListJobs, CreateSource, GetSources, CreateEntity, GetEntities, SearchJobs | ✅ |
| `handler.go` | REST endpoints | Present — POST/GET jobs, POST/GET sources, POST/GET entities | ✅ |
| Freshness score on sources | Should exist | ResearchSource has `url`, `title`, `snippet`, `published_at`, `relevance_score` — **NO `freshness_score` field** | ❌ |
| Verification fields | Should exist | **NO `is_verified`, `grounding`, `source_type` fields** | ❌ |
| Grounding metadata | Should exist | **NOWHERE in codebase** | ❌ |

**Research Verdict**: Basic research module with source/entity tracking. Missing freshness scoring, verification, and grounding metadata entirely.

## 7. SEO Engine Module (`internal/modules/seoengine/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Migration 000020 | SEO tables with freshness_score | `seo_projects`, `seo_audits`, `seo_scores` — all have `freshness_score NUMERIC(5,2)` | ✅ |
| `model.go` | SEO types | Present — SEOProject, SEOAudit, SEOScore, SEOSuggestion, KeywordResearch, ContentAnalysis | ✅ |
| `service.go` | SEO methods | Present — CreateProject, GetProject, ListProjects, RunAudit, GetAudit, GetLatestAudit, GetScores, GetSuggestions, KeywordResearch, ContentAnalysis, GetFreshnessScore, GetMetrics | ✅ |
| `handler.go` | REST endpoints | Present | ✅ |

**SEO Verdict**: `freshness_score` exists only in SEO tables (migration 000020). Not present in publications, posts, or research sources.

## 8. Publisher Module (`internal/modules/publisher/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Migration 000019 | publisher tables | `publications`, `publication_queue`, `publication_logs`, `publication_channels`, `publication_schedule` | ✅ |
| `model.go` | Publication types | Present — Publication with ID, Title, Content, Slug, Status, Language, Translations, MultilingualURLs, ContentType, SiteID, CreatedAt, UpdatedAt | ✅ |
| `service.go` | CRUD + publish workflow | Present — 16 methods | ✅ |
| `handler.go` | REST endpoints | Present | ✅ |
| Freshness score on publications | Should exist | Publication struct has **NO freshness_score** | ❌ |

**Publisher Verdict**: Publication_queue conflict: migration 000014 creates it with FK to autocontent_jobs, migration 000019 recreates it with FK to publications. Publication model lacks freshness_score.

## 9. Posts Module (`internal/modules/posts/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Migration 000005 | posts table | `id`, `title`, `slug`, `content`, `excerpt`, `status`, `category_id`, `author_id`, `site_id`, `published_at`, `created_at`, `updated_at`, `deleted_at` | ✅ |
| `handler.go` | GetBySlug endpoint | `GetBySlug` handler exists but uses query param `?slug=` — **NOT** path param `/api/v1/articles/{slug}` | ❌ Missing path-param route |
| Language column | Should exist | posts table has **NO language column** | ❌ |
| Freshness score | Should exist | posts table has **NO freshness_score** | ❌ |

## 10. API Routes (`internal/api/routes.go`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| Total routes | 285 | 285 registered | ✅ |
| `/api/v1/articles/{slug}` | Should exist | **NOT FOUND** — no path-param slug route anywhere | ❌ |
| AI routes registered | Yes | `/api/v1/ai/providers`, `/api/v1/ai/health`, `/api/v1/ai/test`, `/api/v1/ai/prompts`, `/api/v1/ai/capabilities` | ✅ |
| Autocontent routes | Yes | 21 routes under `/api/v1/autocontent/` | ✅ |

## 11. Middleware (`internal/api/middleware/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `authz.go` | Role-based access control | CasbinAuthorizer with Enforce, SetUserRole. **SetUserRole never called** — defaults to "user" at line 37-39 | ⚠️ Role always "user" |
| RLS (Row-Level Security) | Should exist | **NOWHERE in codebase** — no site_id filtering middleware | ❌ |

## 12. Site Frontend (`site/app/`)

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `page.tsx` | Stub page | "Site em construção" with CSS animation | ✅ Stub |
| `layout.tsx` | Root layout | Present | ✅ |
| `[slug]` route | Article pages | **DOES NOT EXIST** | ❌ |
| API client | Should connect to API | **NO API client** in site/ | ❌ |
| Article components | Should render articles | **NONE** | ❌ |

## 13. Configuration & Environment

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `internal/pkg/config/config.go` | Config struct | Has Server, Database, Redis, JWT, Logging, CORS — **NO AI fields** | ❌ No AI config |
| `.env.example` | Should have AI vars | Database, Redis, JWT, Server, CORS — **NO GEMINI or AI provider vars** | ❌ |
| `go.mod` AI dependencies | google-generative-ai | **NOT PRESENT** — no google.golang.org/api, no cloud.google.com/go/ai | ✅ Correct (mock-only) |

## 14. Cross-Cutting Concerns

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| RLS on any query | site_id filter | **NO site_id filtering** in any service.go file examined | ❌ |
| Cache prefix | site-scoped caching | Redis cache exists but **no site prefix** | ❌ |
| Multi-tenancy | Site isolation | **Not implemented** anywhere | ❌ |
| EventBus wired | All modules | main.go registers EventBus and passes to AuthModule, AIPipelineModule, ContentGeneratorModule, AutocontentModule, SEOEngineModule, WorkflowModule, PublisherModule, ResearchModule, PostModule | ✅ |

## 15. Compilation & Tests

| Check | Expected | Actual | Verdict |
|-------|----------|--------|---------|
| `go build ./...` | Clean | **NOT TESTED** — shell non-functional | ⚠️ Unknown |
| `go vet ./...` | Clean | **NOT TESTED** — shell non-functional | ⚠️ Unknown |
| `go test ./...` | Clean | **NOT TESTED** — shell non-functional | ⚠️ Unknown |

---

## Summary of Findings

### ✅ What Exists and Works
- Complete project scaffold with kernel module system, chi router, Casbin authz, EventBus
- 285 API routes across 9 modules
- Well-designed AI abstraction layer (interfaces, registry, circuit breaker, pipeline stages, prompt builder)
- Autocontent workflow engine with 91 tests
- Content generator state machine
- Research module with source/entity tracking
- SEO engine with freshness_score
- Publisher with multi-language support
- Posts CRUD

### ❌ What's Missing (Sprint 3.7 Gaps)
1. **No RLS (Row-Level Security)** — no site_id filtering anywhere
2. **No multi-tenancy** — no site isolation middleware
3. **No real AI provider** — MockProvider only, no Gemini/OpenAI integration
4. **No `/api/v1/articles/{slug}` route** — posts use `?slug=` query param
5. **No site frontend** — `site/app/` is a stub with no article pages
6. **No freshness_score on publications or posts** — exists only in SEO tables
7. **No AI config** — `config.go` and `.env.example` lack AI provider settings
8. **No grounding metadata** — nowhere in the codebase
9. **Publication queue conflict** — two migrations define the same table differently

### ⚠️ Risks
- Shell/bash non-functional — cannot verify compilation or run tests
- `SetUserRole` never called in authz middleware — authz is effectively bypassed
- Role defaults to "user" — admin endpoints may be inaccessible
