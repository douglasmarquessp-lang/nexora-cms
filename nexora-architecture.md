# 🏗️ Nexora CMS — Arquitetura do Sistema

> **Versão 1.0** — Arquitetura completa para substituição do WordPress  
> Plataforma multi-site com IA, módulos independentes e sistema de plugins

---

## Índice

1. [Arquitetura Recomendada](#1-arquitetura-recomendada)
2. [Linguagens Recomendadas](#2-linguagens-recomendadas)
3. [Framework Recomendado](#3-framework-recomendado)
4. [Banco de Dados](#4-banco-de-dados)
5. [Estrutura de Pastas](#5-estrutura-de-pastas)
6. [Comunicação entre Módulos](#6-comunicação-entre-módulos)
7. [Sistema de Plugins](#7-sistema-de-plugins)
8. [Escalabilidade](#8-escalabilidade)
9. [Segurança](#9-segurança)
10. [Cache](#10-cache)
11. [API](#11-api)
12. [Estratégia para Futuras Atualizações](#12-estratégia-para-futuras-atualizações)

---

## 1. Arquitetura Recomendada

**Arquitetura Híbrida: Modular Monolith + Microsserviços Seletivos**

```
┌──────────────────────────────────────────────────────────┐
│                    Presentation Layer                     │
│      Admin SPA (React)  │  Site Frontend (Next.js)       │
├──────────────────────────────────────────────────────────┤
│                     API Gateway (BFF)                     │
├──────────────────────────────────────────────────────────┤
│                   Core Kernel (Núcleo)                    │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│  │ Site │ │ Post │ │ User │ │Assets│ │Config│ │ SEO  │  │
│  │ Mngr │ │ Mngr │ │ Mngr │ │ Mngr │ │ Mngr │ │ Base │  │
│  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘  │
│  ┌──────┐ ┌──────────────────┐ ┌───────────┐             │
│  │  AI  │ │   Content        │ │Automation │             │
│  │ Base │ │   Intelligence   │ │           │             │
│  └──────┘ └──────────────────┘ └───────────┘             │
├──────────────────────────────────────────────────────────┤
│              Plugin System (Premium Extras)               │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐  │
│  │SEO   │ │  AI  │ │Analyt│ │ Aff  │ │ News │ │ Ads  │  │
│  │ Pro  │ │ Pro  │ │ ics  │ │      │ │letter│ │      │  │
│  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘  │
├──────────────────────────────────────────────────────────┤
│              Infrastructure Layer                         │
│  Cache │ Queue │ Search │ Storage │ Event Bus             │
└──────────────────────────────────────────────────────────┘
```

**Por que Modular Monolith primeiro?**  
CMS monousuário/admin típico não exige escala de Twitter. Modular Monolith oferece:
- Menor latência (sem overhead de rede entre módulos)
- Deployment único (mais simples)
- Transações ACID entre módulos
- Performance muito superior para operações administrativas

**Quando escalar para microsserviços?**  
Módulos com carga isolada (ex: Analytics pesado, AI processing) podem ser extraídos gradualmente como microsserviços independentes, usando o mesmo contrato de interface.

---

## 2. Linguagens Recomendadas

| Camada | Linguagem | Motivo |
|--------|-----------|--------|
| **Backend Core** | **Go** | Performance, concorrência nativa, binário único, deployment simples, tipagem forte, excelente para APIs REST/gRPC |
| **Admin Frontend** | **TypeScript** | Type safety, ecossistema React, tooling maduro |
| **Frontend dos Sites** | **TypeScript** | Next.js SSR/SSG, performance, SEO |
| **Scripts de plugin** | **Go (nativo) + WebAssembly (sandbox)** | Plugins em WASM para segurança, módulos premium podem ser Go nativo |
| **AI/ML Pipeline** | **Python** (módulo separado) | Ecossistema de IA maduro (PyTorch, Transformers, LangChain) |

---

## 3. Framework Recomendado

### Backend (Go)
- **HTTP Router:** `chi` (leve, idiomático, middleware composável)
- **ORM:** `sqlc` + `pgx` (codegen de SQL puro → Go types — sem runtime reflection, performance máxima)
- **Auth:** `casbin` (RBAC/ABAC flexível por site)
- **Event Bus:** Nativo com canais + Redis Pub/Sub para módulos
- **Background Jobs:** `river` (filas PostgreSQL nativas)

### Admin Frontend (TypeScript)
- **Framework:** `React 19 + Vite`
- **Estado Global:** `Zustand` (leve, sem boilerplate)
- **UI Kit:** `shadcn/ui` (Radix primitives + Tailwind)
- **Formulários:** `React Hook Form + Zod`
- **Data Fetching:** `TanStack Query`
- **Internacionalização:** `react-i18next`

### Site Frontend (TypeScript)
- **Framework:** `Next.js 19` (App Router)
- **Rendering:** ISR (Incremental Static Regeneration) para artigos com SSG + revalidação sob demanda
- **Estilização:** `Tailwind CSS`
- **Data Fetching:** Server Components direto na API

---

## 4. Banco de Dados

### Primário: PostgreSQL 17
- **Justificativa:** JSONB para metadata flexível, índices GIN/GIST para full-text search, recursos como NOTIFY/LISTEN para eventos, CTE para hierarquia de comentários/categorias, particionamento nativo
- **Extensões:** `pg_trgm` (busca fuzzy), `pgvector` (embeddings de IA), `postgis` (se necessário)

### Schema Multi-tenant (Single Database + Row Level Security)

```
core_sites
├── id, uuid, slug, name, domain, settings (jsonb), status, created_at

core_users
├── id, uuid, email, password_hash, role, avatar, metadata, last_login
└── site_users (pivot: user_id, site_id, role, permissions)

core_posts
├── id, uuid, site_id (FK), type (post/page), title, slug, content (jsonb)
├── excerpt, status (draft/published/archived), author_id
├── published_at, created_at, updated_at
└── post_meta (jsonb) ← SEO fields, AI metadata, custom fields

core_categories / core_tags
├── site_id, name, slug, parent_id (categories)
└── post_categorization (post_id, category_id/tag_id, type)

core_assets
├── site_id, filename, mime_type, size, width, height, alt
├── url, disk_prefix, storage_type (local/s3), uploaded_by
└── post_assets (pivot: post_id, asset_id, order)

core_comments
├── site_id, post_id, parent_id, user_id/guest_id
├── content, status (pending/approved/spam), depth
└── índices GIN para full-text search, índices por site+status+post

analytics_events         ← tabela separada (particionada por data)
├── site_id, event_type, page_url, user_agent, ip (hash), session_id
├── payload (jsonb), created_at
└── particionamento mensal + retention de 90 dias (raw)

plugin_store_*           ← tabelas dinâmicas de plugins
├── prefixo "plugin_" + slug do módulo
└── plugins registram schemas via migração programática
```

### sqlc + Migrations
- **Migrações:** `golang-migrate` com versionamento semântico
- **Codegen:** `sqlc` gera tipos Go type-safe a partir de SQL puro — zero ORM overhead

---

## 5. Estrutura de Pastas

```
nexora/
├── cmd/
│   ├── api/                  # Entrypoint da API REST (admin)
│   │   └── main.go
│   ├── frontend/             # Entrypoint do site Next.js
│   │   └── main.go (proxy/serve)
│   ├── worker/               # Background jobs (newsletter, analytics agg)
│   │   └── main.go
│   └── migrate/              # CLI de migração
│       └── main.go
│
├── internal/                 # Código privado do core
│   ├── kernel/               # Núcleo do sistema
│   │   ├── kernel.go         # Inicialização, lifecycle, plugin manager
│   │   ├── eventbus.go       # Event dispatcher síncrono/assíncrono
│   │   └── registry.go       # Registro de módulos
│   │
│   ├── modules/              # Módulos do Core (gratuitos)
│   │   ├── site/
│   │   │   ├── handler.go    # Transport layer (HTTP handlers)
│   │   │   ├── service.go    # Business logic
│   │   │   ├── repository.go # Data access (sqlc queries)
│   │   │   ├── model.go      # Domain types
│   │   │   └── queries.sql   # SQL queries (sqlc)
│   │   ├── post/
│   │   ├── category/
│   │   ├── tag/
│   │   ├── assets/           # Antigo "media"
│   │   ├── user/
│   │   ├── comment/
│   │   ├── config/
│   │   ├── seo/              # SEO Base — core gratuito
│   │   ├── ai/               # IA Base — core gratuito (modelos locais/OSS)
│   │   ├── content_intelligence/  # Content Intelligence
│   │   └── automation/       # Automação de fluxos de conteúdo
│   │
│   ├── plugins/              # Core plugin abstrações
│   │   ├── plugin.go         # Interface Plugin
│   │   ├── sandbox.go        # WASM sandbox runtime
│   │   └── manifest.go       # plugin.json parser
│   │
│   ├── pkg/                  # Pacotes compartilhados
│   │   ├── auth/             # JWT, session, OAuth2
│   │   ├── cache/            # Cache layer (redis/in-memory)
│   │   ├── storage/          # Filesystem/S3 abstraction
│   │   ├── search/           # Full-text search interface
│   │   ├── queue/            # Job queue
│   │   ├── i18n/             # Internacionalização
│   │   └── validator/        # Validação de input
│   │
│   └── api/                  # API contracts
│       ├── rest/             # REST handlers, middleware
│       ├── dto/              # Request/Response types
│       └── middleware/       # Rate limit, CORS, auth, audit
│
├── plugins/                  # Plugins Premium (apenas recursos extras)
│   ├── seo-pro/              # SEO Avançado: análise concorrentes, cluster tópicos
│   ├── ai-pro/               # IA Premium: GPT-4, Cloud Vision, Dall-E
│   ├── analytics/            # Analytics avançado com dashboards
│   ├── affiliate/            # Gerenciamento de links afiliados
│   ├── ads/                  # Gerenciamento de anúncios
│   ├── newsletter/           # Email marketing automatizado
│   └── keyword-research/     # Pesquisa de palavras-chave (APIs pagas)
│
├── web/                      # Frontend Admin (SPA)
│   ├── src/
│   │   ├── pages/            # React Router pages
│   │   ├── components/       # Shared UI components
│   │   ├── hooks/            # Custom hooks
│   │   ├── stores/           # Zustand stores
│   │   ├── api/              # API client (TanStack Query)
│   │   ├── i18n/             # Translation files
│   │   └── lib/              # Utility functions
│   ├── package.json
│   └── vite.config.ts
│
├── site/                     # Frontend dos sites (Next.js)
│   ├── app/                  # App Router pages
│   ├── components/
│   ├── lib/
│   └── package.json
│
├── migrations/               # SQL migrations versionadas
│   ├── 000001_create_sites.up.sql
│   ├── 000001_create_sites.down.sql
│   └── ...
│
└── deploy/                   # Configuração de deployment
    ├── docker-compose.yml
    ├── Dockerfile
    └── nginx.conf
```

---

## 6. Comunicação entre Módulos

### Padrão: Event-Driven via Kernel

Cada módulo expõe **três camadas**:

```
┌────────────────────────────────────────────┐
│               HTTP Handler                 │  ← REST endpoints
├────────────────────────────────────────────┤
│              Service Layer                 │  ← Regras de negócio
├────────────────────────────────────────────┤
│              Repository (sqlc)             │  ← Acesso a dados
└────────────────────────────────────────────┘
```

**Comunicação entre módulos:**

| Método | Quando usar | Exemplo |
|--------|-------------|---------|
| **Chamada direta de Service** | Módulo A precisa de dado do B (mesmo processo) | `postService.GetByID(id)` |
| **Event Bus (síncrono)** | Módulo precisa notificar outros e aguardar | `kernel.Emit("post.created", event)` |
| **Event Bus (assíncrono/fila)** | Operação não-crítica, pode ser retardada | Analytics, email, revalidação de cache |
| **Hook points** | Plugins interceptam fluxo do core | `BeforePostSave`, `AfterPostPublish` |

### Contrato de Eventos

```go
// Todos os módulos publicam/consomem eventos tipados
type Event struct {
    ID        string      // UUID
    Type      EventType   // "post.created", "user.registered"
    Timestamp time.Time
    Payload   interface{} // Dados do evento
    SiteID    string      // Contexto multi-site
}

// Hooks: plugins implementam interfaces
type Hook interface {
    Priority() int        // Ordem de execução
    Handle(ctx Context, payload interface{}) error
}
```

**Fluxo típico (publicar artigo):**

```
1. POST /api/posts → SiteHandler.Create
2. SiteService.Create valida, persiste no DB
3. Kernel.Emit("post.created", PostEvent{...})
4. Módulo SEO (core): gera meta tags automáticas, salva em post_meta
5. Módulo AI (core): gera resumo, extrai keywords com modelo local/OSS
6. Módulo Content Intelligence: classifica conteúdo, sugere categorias
7. Módulo Automation: dispara regras de automação (publicação agendada, etc.)
8. Plugin Analytics (premium, async): enfileira evento de analytics
9. Plugin Cache (core): invalida cache de página
10. Retorna 201 Created
```

**Isolamento:** Plugins com erro não quebram o fluxo principal (trap de panic, timeout por contexto, fallback silencioso).

---

## 7. Sistema de Plugins

### Arquitetura de Plugins

```
┌─────────────────────────────────────────────┐
│               Plugin Manager                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Discovery │ │  Loader  │ │ Sandbox  │     │
│  └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │  Hooks   │ │  Events  │ │ Perms    │     │
│  └──────────┘ └──────────┘ └──────────┘     │
└─────────────────────────────────────────────┘
```

### Plugin Manifest (plugin.json)

```json
{
  "slug": "nexora-seo",
  "name": "SEO Analyzer",
  "version": "1.0.0",
  "type": "premium",
  "entrypoint": "plugin.wasm",
  "hooks": ["AfterPostSave", "BeforeRenderPage"],
  "permissions": ["posts:read", "posts:write", "settings:read"],
  "tables": ["plugin_seo_scores", "plugin_seo_keywords"]
}
```

### Tipos de Plugin

| Tipo | Runtime | Segurança | Performance |
|------|---------|-----------|-------------|
| **Core** (primeira party) | Go nativo compilado junto | Total | Máxima |
| **Premium** (marketplace) | WASM + Go host functions | Alta (sandbox) | Média |
| **Community** | WASM restrito | Máxima (no filesystem, no network) | Limitada |

> **Nota:** SEO Base e IA Base são módulos do Core (não plugins). Apenas funcionalidades avançadas (SEO Pro, AI Pro) são plugins premium.

### Plugin Marketplace

- Plugins premium têm acesso a APIs pagas (OpenAI, etc) via proxy do core
- Plugins gratuitos têm limitação de recursos (rate limit, memória WASM)
- Sistema de licenciamento com chave pública/privada e verificação off-line

### Hooks Disponíveis

```
Core Hooks:
├── BeforePostSave / AfterPostSave
├── BeforePostPublish / AfterPostPublish
├── BeforePostDelete / AfterPostDelete
├── BeforeRenderPage / AfterRenderPage
├── BeforeCommentSave / AfterCommentSave
├── BeforeUserRegister / AfterUserRegister
├── BeforeLogin / AfterLogin
├── OnSearchQuery
└── OnSitemapGenerate

Plugin Hooks:
├── OnPluginActivate / OnPluginDeactivate
├── OnPluginUpdate
└── OnLicenseVerify
```

---

## 8. Escalabilidade

### Vertical (Fase 1 — até 500 sites)
- Modular Monolith em Go + PostgreSQL
- Cache Redis em memória
- CDN para mídia estática
- Single server: 8 vCPU, 32 GB RAM → suporta ~500 sites com 10k artigos cada

### Horizontal (Fase 2 — 500+ sites)

```
                 ┌──────────────┐
                 │  Load Balancer│
                 └──────┬───────┘
                        │
         ┌──────────────┼──────────────┐
         │              │              │
    ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
    │ API     │   │ API     │   │ API     │
    │ Node 1  │   │ Node 2  │   │ Node N  │  (Go stateless)
    └────┬────┘   └────┬────┘   └────┬────┘
         │              │              │
    ┌────┴────────────────────────────┴────┐
    │         PostgreSQL (Primary)         │
    │         + Read Replicas              │
    └──────────────────────────────────────┘
         │              │              │
    ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
    │ Redis   │   │ RabbitMQ│   │  S3/    │
    │ Cache   │   │ Queue   │   │  MinIO  │
    └─────────┘   └─────────┘   └─────────┘
```

### Estratégias de Escala

1. **Conexões DB:** `pgx` connection pooling + PgBouncer
2. **Read Replicas:** Consultas de leitura (sites públicos) roteadas para réplicas
3. **Sharding de Sites:** Fase 3, partition key = `site_id`, cada shard com seu PostgreSQL
4. **Caching em camadas:**
   - L1: `sync.Map` in-memory (por pod) — TTL segundos
   - L2: Redis cluster — TTL minutos
   - L3: CDN (Cloudflare/Fastly) — TTL horas para assets estáticos
5. **Background Jobs:** Worker pool independente, escala horizontalmente

---

## 9. Segurança

### Multi-tenant Security

```
┌──────────────────────────────────────────┐
│           Layer 1: Authentication         │
│  JWT (access + refresh tokens)           │
│  OAuth2 (Google, GitHub, Apple)          │
│  MFA (TOTP) para admins                  │
│  Session rotation a cada login           │
├──────────────────────────────────────────┤
│           Layer 2: Authorization          │
│  Casbin RBAC/ABAC por site               │
│  Roles: SuperAdmin → SiteAdmin → Editor  │
│         → Author → Subscriber            │
│  Permissions granulares por módulo       │
├──────────────────────────────────────────┤
│           Layer 3: Row Level Security     │
│  PostgreSQL RLS: `site_id = current_setting` │
│  Usuário do site A NUNCA vê dados do B  │
├──────────────────────────────────────────┤
│           Layer 4: Input Validation       │
│  Zod schemas no frontend                 │
│  Validação dupla no backend (Go types)   │
│  SQL parameterized (sqlc gera $1, $2...)│
├──────────────────────────────────────────┤
│           Layer 5: Rate Limiting          │
│  Por endpoint, por site, por IP          │
│  Algoritmo: Sliding Window (Redis)       │
│  Limites diferentes: API (100/min),      │
│  Auth (5/min), Public (1000/min)         │
├──────────────────────────────────────────┤
│           Layer 6: Audit Trail            │
│  Toda mutação logada: quem, quando, o quê│
│  Logs imutáveis (append-only table)      │
│  Rotação automática (90 dias)             │
└──────────────────────────────────────────┘
```

### Medidas Específicas

- **Senhas:** Argon2id (memória 64MB, iter 3, paralelismo 4)
- **API Keys:** Hash SHA-256 armazenado, chave mostrada 1x
- **CORS:** Whitelist por domínio do site
- **Upload Seguro:** Extensão validada, magic bytes, scan ClamAV (opcional), filename random UUID
- **XSS:** Content-Security-Policy strict, sanitização DOMPurify no render
- **CSRF:** SameSite=Strict + CSRF token rotativo
- **SQL Injection:** Zero — sqlc gera queries parametrizadas

---

## 10. Cache

### Estratégia Multi-camada

```
                    ┌─────────────┐
                    │  Request    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   CDN Edge  │  ← Cloudflare/Fastly (público)
                    │  (HTML/CSS) │    Cache por URL + stale-while-revalidate
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  Full Page  │  ← NGINX/Envoy reverse proxy
                    │  Cache      │    Cache por site+path+query
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  Application│  ← Redis Cluster
                    │  Cache      │    Cache de queries, sessões, rate limit
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  In-Memory  │  ← sync.Map (Go, por pod)
                    │  L1 Cache   │    Cache de configurações, traduções
                    └─────────────┘
```

### Políticas de Invalidação

| Evento | Ação | Cache Afetado |
|--------|------|---------------|
| Post publicado | `PURGE /posts/{slug}` | CDN + Page + Redis |
| Post atualizado | Revalida ISR + PURGE | Next.js ISR + Page |
| Config alterada | `DEL config:*` | Redis + L1 |
| Plugin ativado/desativado | `FLUSH` parcial | Redis + L1 |
| Asset enviado | `PURGE /assets/*` | CDN |
| Comentário novo | Invalida só o post | Page cache do post |

### Cache Patterns

1. **Cache-Aside (Lazy Loading):** `get(key) → if miss → load from DB → set cache → return`
2. **Write-Through:** Para config/translations — `set(key, value) → write DB + write cache → return`
3. **Cache Warming:** Worker regenera cache de posts populares a cada N minutos
4. **Stale-While-Revalidate (SWR):** Serve cache expirado enquanto atualiza em background

---

## 11. API

### Design: REST (com GraphQL futuro)

**Versão inicial: REST puro** (simplicidade, cacheabilidade, maturidade)

```
Base URL: /api/v1/{site_slug}/

Sites
├── GET    /sites                  → Listar sites
├── POST   /sites                  → Criar site
├── GET    /sites/:id              → Detalhes do site
├── PUT    /sites/:id              → Atualizar site
└── DELETE /sites/:id              → Remover site

Posts
├── GET    /posts                  → Listar (filtros: status, category, tag, author, date)
├── POST   /posts                  → Criar
├── GET    /posts/:id              → Obter
├── PUT    /posts/:id              → Atualizar
├── DELETE /posts/:id              → Deletar (soft delete)
├── PATCH  /posts/:id/status       → Mudar status (draft/published/archived)
├── POST   /posts/:id/duplicate    → Duplicar
└── GET    /posts/:id/versions     → Histórico de versões

Categories / Tags
├── CRUD padrão (GET, POST, PUT, DELETE)
└── GET    /categories/tree        → Árvore hierárquica

Assets
├── POST   /assets/upload          → Upload (multipart, signed URL)
├── GET    /assets                 → Listar (filtros: mime, date)
├── DELETE /assets/:id             → Deletar
└── PATCH  /assets/:id             → Atualizar metadados

Users
├── GET    /users/me               → Perfil atual
├── PUT    /users/me               → Atualizar perfil
├── GET    /users                  → Listar (admin)
├── POST   /users                  → Criar (admin)
├── PUT    /users/:id              → Atualizar (admin)
└── DELETE /users/:id              → Remover (admin)

Comments
├── GET    /posts/:id/comments     → Listar por post
├── POST   /posts/:id/comments     → Criar
├── PUT    /comments/:id           → Atualizar
├── DELETE /comments/:id           → Deletar
└── PATCH  /comments/:id/status    → Aprovar/spam

Auth
├── POST   /auth/login             → Login (email + password)
├── POST   /auth/register          → Registrar
├── POST   /auth/logout            → Logout
├── POST   /auth/refresh           → Refresh token
├── POST   /auth/forgot-password   → Esqueci senha
├── POST   /auth/reset-password    → Resetar senha
└── POST   /auth/oauth/:provider   → OAuth login

System
├── GET    /system/health          → Health check
├── GET    /system/info            → Versão, módulos ativos
├── GET    /system/config          → Configurações
└── PUT    /system/config          → Atualizar configurações
```

### Respostas Padronizadas

```json
// Success
{
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}

// Error
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "O título é obrigatório",
    "details": [
      { "field": "title", "message": "mínimo 3 caracteres" }
    ],
    "request_id": "req_abc123"
  }
}
```

### Paginação

- **Padrão:** Cursor-based (performance) para lists grandes
- **Fallback:** Offset-based para UI admin (page + per_page)
- **Sort:** `?sort=created_at:desc,published_at:asc`
- **Filtros:** `?status=published&category=tech&q=search+term`
- **Fields:** `?fields=id,title,excerpt` (selective fields)

### Versionamento

- Header `Accept: application/vnd.nexora.v1+json`
- Versão na URL como fallback (`/api/v1/`)
- Breaking changes apenas em major version
- Deprecation avisada com header `Warning: 299 - "v1 deprecated, use v2"`

---

## 12. Estratégia para Futuras Atualizações

### Filosofia: "Core Evergreen, Modules Versioned"

```
Core (kernel + módulos essenciais)  ← Sempre atualizado, não quebra
    ├── v1.0.0
    ├── v1.1.0  (feature)
    ├── v1.2.0  (feature)
    └── v2.0.0  (breaking — mas raro)

Plugins Premium/Community          ← Versionados independentemente
    ├── seo@1.0.0, seo@1.1.0
    ├── ai@2.0.0 (requer core >=1.5)
    └── analytics@1.0.0
```

### Mecanismo de Atualização

1. **Migration Programática**
   - Cada versão do core + plugins tem migrations up/down
   - `nexora migrate up` aplica pendentes em ordem
   - Downgrade possível: `nexora migrate down 1`

2. **Feature Flags**
   - Novas funcionalidades atrás de toggle
   - `nexora feature:enable analytics-v2`
   - Permite rollout gradual e rollback sem deploy

3. **Schema Version Tracking**
   - Tabela `_migrations` com hash de verificação
   - Se migration falhar → rollback automático

4. **Zero Downtime Updates**
   - API: rolling update (stateless)
   - Migrations: backward-compatible (add columns, não drop)
   - Cache warming pós-deploy automático

5. **Plugin API Contract**
   - Plugins declaram versão mínima do core
   - Se core atualizar quebrando → plugin desativado até update
   - Mensagem clara: "Plugin SEO requer core >=2.0.0"

### Release Strategy

| Version | Ciclo | Mudanças |
|---------|-------|----------|
| **v1.0.0-alpha** | Mês 1-2 | Core: Sites, Posts, Auth, Assets, Users, SEO Base |
| **v1.0.0-beta** | Mês 3 | + Categorias, Tags, Comentários, AI Base, API completa |
| **v1.0.0-rc** | Mês 4 | + Content Intelligence, Automation, Plugin System, i18n PT/EN |
| **v1.0.0** | Mês 5 | Release estável com todos os módulos do Core |
| **v1.1.0** | Mês 6 | Plugin Premium: AI Pro (GPT-4, Dall-E, Cloud Vision) |
| **v1.2.0** | Mês 7 | Plugin Premium: SEO Pro + Keyword Research |
| **v1.3.0** | Mês 8 | Plugin Premium: Analytics |
| **v1.4.0** | Mês 9 | Plugin Premium: Newsletter |
| **v1.5.0** | Mês 10 | Plugin Premium: Afiliados + Anúncios |
| **v2.0.0** | Mês 12+ | API GraphQL, Webhooks, Performance mode |

### Compatibilidade

- Toda versão nova do core roda todas as migrations anteriores
- Plugins declarativos: `requires: ">=1.0.0 <2.0.0"`
- Testes de regressão com `docker-compose` + banco populado com dados de versões anteriores
- Changelog automático gerado a partir de conventional commits

---

## Resumo da Pilha Tecnológica

| Camada | Tecnologia | Motivo |
|--------|-----------|--------|
| **Backend** | Go + chi + sqlc | Performance + segurança |
| **Banco** | PostgreSQL 17 + pgvector | Confiabilidade + IA |
| **Cache** | Redis | Velocidade |
| **Fila** | pgx + river | DB nativo, sem dependência extra |
| **Admin UI** | React 19 + shadcn/ui | Experiência dev + UX |
| **Site Frontend** | Next.js 19 + Tailwind | SEO + performance |
| **Plugins** | Go WASM | Sandbox seguro |
| **Storage** | Local FS / S3 / MinIO | Flexibilidade |
| **Auth** | Casbin + JWT + Argon2id | Segurança enterprise |
| **Search** | PostgreSQL full-text (Fase 1) / Meilisearch (Fase 2) | Sem dependência extra |
| **AI (Core)** | Python (módulo externo, modelos locais/OSS gratuitos) | Sem dependência de APIs pagas |
| **AI (Premium)** | Python + APIs pagas (OpenAI, Google Cloud) | Recursos avançados opcionais |

---

> **Arquivo gerado em:** Julho 2026  
> **Projeto:** Nexora CMS — A próxima geração de gerenciamento de conteúdo com IA
