# Auditoria de Deployment para Vercel — Nexora CMS

**Data:** 2026-07-30
**Objetivo:** Preparar Admin SPA + Go API para produção na Vercel
**Escopo:** `web/`, `cmd/api/`, `deploy/`, CI/CD, CORS, autenticação, arquitetura

---

## 1. Estrutura Atual do Projeto

```
nexora-cms/
├── cmd/api/main.go              # Go API server (chi router, porta 8080)
├── cmd/migrate/main.go          # Runner de migrations
├── cmd/worker/main.go           # Background worker
├── web/                         # Admin SPA (Vite 6 + React 19 + TypeScript)
│   ├── vite.config.ts           # Dev proxy /api → localhost:8080
│   ├── package.json             # build: tsc -b && vite build
│   ├── src/api/client.ts        # API_BASE = "/api/v1" (relativo)
│   ├── index.html               # Entry point Vite
│   └── Dockerfile.admin         # Apenas dev (npm run dev)
├── site/                        # Public Site (Next.js 15)
├── deploy/
│   ├── Dockerfile               # Go API (3 estágios: builder, dev, prod)
│   ├── Dockerfile.admin         # Admin SPA dev-only
│   ├── Dockerfile.site          # Public Site dev-only
│   ├── docker-compose.yml       # 5 serviços (postgres, redis, api, admin, site)
│   └── nginx.conf               # Proxy reverso local (.nexora.local)
├── .env / .env.example          # Config de ambiente
├── .github/workflows/ci.yml     # CI apenas (sem CD/deploy)
└── Makefile                      # build, run, dev, test...
```

---

## 2. Admin SPA (web/) — Build e Deploy

### 2.1 Comando de build
```json
// web/package.json
"build": "tsc -b && vite build"
```

### 2.2 Diretório de saída
- **Padrão Vite:** `web/dist/`
- `vite.config.ts` **não define** `build.outDir` → usa o padrão `dist/`
- `tsconfig.json` tem `"noEmit": true` (TypeScript só para type checking, Vite faz o bundle)

### 2.3 Roteamento de API no Admin
```typescript
// web/src/api/client.ts
const API_BASE = "/api/v1";

// Exemplo de uso:
const url = new URL(`${API_BASE}${path}`, window.location.origin);
// Resolve para: window.location.origin + /api/v1/...
```

- **Usa caminho relativo** — resolve contra `window.location.origin`
- Em dev: Vite proxy (`/api` → `http://localhost:8080`)
- Em produção (Vercel): resolve contra o domínio do Admin SPA

### 2.4 Proxy Vite (dev apenas)
```typescript
// web/vite.config.ts
proxy: {
  "/api": {
    target: process.env.API_PROXY_TARGET || "http://localhost:8080",
    changeOrigin: true,
  },
}
```
**⚠️ Crítico:** Este proxy NÃO existe em produção. O Admin SPA em Vercel fará fetch para `/api/v1/...` que **não existe** no servidor estático da Vercel sem configuração de rewrites.

### 2.5 Headers enviados pelo Admin
| Header | Origem | Finalidade |
|--------|--------|------------|
| `Authorization: Bearer <token>` | `localStorage.getItem("access_token")` | Autenticação JWT |
| `X-Site-ID` | `useSiteStore.getState().currentSite.id` | Identificação do site (multi-tenancy) |
| `Content-Type: application/json` | Padrão | Formato da requisição |
| `Content-Type: multipart/form-data` | Quando `formData: true` | Upload de arquivos |

### 2.6 Refresh Token
- `access_token` + `refresh_token` armazenados em `localStorage`
- Refresh automático via `attemptRefresh()` quando recebe 401
- **Vulnerabilidade:** `localStorage` é acessível via XSS. Ideal seria httpOnly cookie.

---

## 3. Go API (cmd/api/) — Build e Deploy

### 3.1 Comando de build
```makefile
# Makefile
build:
	go build -o bin/nexora ./cmd/api
```

```dockerfile
# deploy/Dockerfile (stage prod)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/nexora ./cmd/api/main.go
```

### 3.2 Rotas expostas (~170+ endpoints)
Todas sob `/api/v1/`:

| Grupo | Prefixo | Auth | Site | Exemplos |
|-------|---------|------|------|----------|
| Health | `/api/v1/health` | ❌ | ❌ | `GET /health` |
| Setup | `/api/v1/setup/*` | ❌ | ❌ | `POST /setup/init` |
| Auth | `/api/v1/auth/*` | Parcial | ❌ | `POST /login`, `POST /refresh`, `GET /me` |
| Public | `/api/v1/articles`, `/api/v1/categories` | ❌ | ✅ | `GET /articles`, `GET /articles/{slug}` |
| Admin | `/api/v1/*` (todo o resto) | ✅ Bearer JWT | ✅ | Sites, Posts, Categories, Tags, Media, Editorial, Research, Writer, Generator, AI, Publisher, Workflow, Plugins, Config |

### 3.3 Dependências de infraestrutura
| Dependência | Obrigatória? | Função |
|-------------|-------------|--------|
| PostgreSQL | Sim (degradado sem) | Dados principais |
| Redis | Não (fallback cache em memória) | Cache, rate limit |
| Storage (disco/S3) | Sim (local path) | Upload de mídia |

### 3.4 Server config
```go
// cmd/api/main.go
addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
// Padrão: 0.0.0.0:8080
srv := &http.Server{
    Addr:           addr,
    Handler:        router,  // chi.Mux
    ReadTimeout:    cfg.Server.Timeout,
    WriteTimeout:   cfg.Server.Timeout,
    IdleTimeout:    120 * time.Second,
    MaxHeaderBytes: 1 << 20,
}
```

---

## 4. CORS — Estado Atual

```go
// internal/api/rest/router.go
rt.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    ExposedHeaders:   []string{"Link"},
    AllowCredentials: false,
    MaxAge:           300,
}))
```

### Problemas identificados:
| Problema | Severidade | Detalhe |
|----------|-----------|---------|
| `AllowedOrigins` hardcoded para localhost | 🔴 **Bloqueante** | Produção precisa incluir o domínio Vercel |
| `X-Site-ID` não está em `AllowedHeaders` | 🔴 **Bloqueante** | Multi-tenancy quebra em CORS |
| `AllowCredentials: false` | 🟡 **Alto** | Cookies de refresh token não funcionarão |
| Sem suporte a múltiplos origins dinâmicos | 🟡 **Médio** | Precisa de validação por domínio |

---

## 5. Análise de Arquiteturas para Vercel

### Opção A: Admin SPA na Vercel + API em servidor separado (RECOMENDADA)

```
[Admin SPA]──(/api/v1/*)──Vercel Rewrite──► [Go API: api.nexora.com]
   Vercel                         
```

| Aspecto | Avaliação |
|---------|-----------|
| **Build** | ✅ `npm run build` → `web/dist/` → Vercel static |
| **API** | ✅ Go API mantida em VPS/Railway/Fly.io com Docker |
| **Roteamento** | ✅ Vercel rewrites: `/api/*` → `https://api.nexora.com/api/*` |
| **CORS** | ✅ Mesma origem se usar rewrites. Ou configurar CORS para domínio Vercel |
| **CDN** | ✅ Assets estáticos na edge Vercel |
| **Complexidade** | 🟡 Média — 2 deploys, 2 domains ou 1 com rewrite |
| **Custo** | 🟢 Vercel free + servidor API (~$5-10/mês) |

### Opção B: Admin SPA + API no mesmo domínio (Vercel Rewrite)

```
[Vercel]──(/api/v1/*)──Vercel Rewrite──► [Go API: api.nexora.com]
          └──(/*)──► Admin SPA static files
```

| Aspecto | Avaliação |
|---------|-----------|
| **Same-origin** | ✅ `window.location.origin/api/v1/...` funciona sem CORS |
| **CORS** | ✅ Zero configuração de CORS para o Admin |
| **API externa** | ✅ Continua em servidor separado |
| **Vercel Rewrite** | ✅ `vercel.json` com `rewrites` roteia `/api/*` |
| **Complexidade** | 🟢 Baixa — 1 domínio, configuração simples |

### Opção C: Go API como Vercel Serverless Functions

| Aspecto | Avaliação |
|---------|-----------|
| **Compatibilidade chi** | 🔴 Chi não é compatível com Vercel Go runtime (`http.Handler`) |
| **Adaptação** | 🔴 Exigiria reescrever todo o router ou usar adaptador |
| **Dependências** | 🔴 PostgreSQL/Redis externos continuam necessários |
| **Stateful** | 🔴 Serverless é stateless — workers/jobs longos quebram |
| **Custo** | 🔴 Vercel serverless Go é caro comparado a VPS |
| **Conclusão** | ❌ **Descartada** — retrabalho enorme, benefício mínimo |

### Opção D: Go API serve Admin SPA (mesmo deployment)

| Aspecto | Avaliação |
|---------|-----------|
| **CORS** | ✅ Zero — mesma origem |
| **Infra** | 🟡 Go precisa servir `web/dist/` como static files |
| **CDN** | 🔴 Perde benefícios de CDN edge da Vercel |
| **Scaling** | 🔴 Go server escala menos que Vercel static |
| **Conclusão** | ⚠️ Viável mas não ideal — perderia CDN, builds separados |

---

## 6. Recomendação Final

### Arquitetura Escolhida: **Opção A+B Híbrida — Admin na Vercel com Rewrite para API externa**

**Motivos:**
1. Admin SPA é React estático → ideal para Vercel (CDN edge, deploy zero-config)
2. Go API tem dependências pesadas (PostgreSQL, Redis, workers) → melhor em VPS/Docker
3. Vercel Rewrites resolvem same-origin sem CORS complexo
4. CI/CD separado para cada componente (independência de deploy)
5. Custo mínimo: Vercel free para Admin + servidor ~$5-10/mês para API

### Topologia Final

```
🌐 admin.nexora.com
   ├── / ─────────────────────────► web/dist/ (Vercel static)
   ├── /api/v1/* ── Vercel Rewrite ──► https://api.nexora.com/api/v1/*
   └── /assets/* ── Vercel Rewrite ──► https://api.nexora.com/uploads/*

🌐 api.nexora.com
   └── Go API (chi) ──► PostgreSQL + Redis (VPS/Docker)
```

---

## 7. Plano de Implementação por Etapas

### Etapa 1 — Configurar Admin SPA para produção

**Arquivos a modificar:**

| Arquivo | O que fazer |
|---------|-------------|
| `web/vite.config.ts` | Adicionar `base: '/admin'` se deploy em subpath, ou manter `/` se domínio dedicado. Adicionar `build.outDir: 'dist'` (já é padrão, explicitar) |
| `web/src/api/client.ts` | Mudar `API_BASE` para usar variável de ambiente: `import.meta.env.VITE_API_BASE || '/api/v1'` |
| `web/.env.production` | Criar com `VITE_API_BASE=/api/v1` (relativo, rewrites resolvem) |
| `web/Dockerfile.admin` | **Remover ou refatorar** — não será mais usado para produção. Adicionar stage `prod` |
| `web/package.json` | (Opcional) Adicionar `"preview": "vite preview"` |

### Etapa 2 — Criar vercel.json

**Arquivo a criar:**

`web/vercel.json`:
```json
{
  "rewrites": [
    { "source": "/api/:path*", "destination": "https://api.nexora.com/api/:path*" }
  ],
  "headers": [
    {
      "source": "/(.*)",
      "headers": [
        { "key": "X-Frame-Options", "value": "DENY" },
        { "key": "X-Content-Type-Options", "value": "nosniff" },
        { "key": "Referrer-Policy", "value": "strict-origin-when-cross-origin" }
      ]
    }
  ]
}
```

**Nota:** Se Admin SPA ficar em subpath (ex: `nexora.com/admin`), usar `base: '/admin'` no Vite e ajustar `source` nos rewrites.

### Etapa 3 — Corrigir CORS na Go API

**Arquivo a modificar:**

`internal/api/rest/router.go`:

```go
AllowedOrigins: []string{
    "http://localhost:3000",
    "http://localhost:3001",
    "https://admin.nexora.com",       // Produção Vercel
    "https://nexora.com",             // Se Admin no root
    "https://www.nexora.com",
},
AllowedHeaders: []string{
    "Accept", "Authorization", "Content-Type", "X-CSRF-Token",
    "X-Site-ID",                      // ← ADICIONAR
},
AllowCredentials: true,               // ← MUDAR para true
```

**Recomendação futura:** Implementar validação dinâmica de origins (allowlist em variável de ambiente).

### Etapa 4 — Configurar CI/CD para Vercel

**Arquivos a modificar:**

`.github/workflows/ci.yml` — Adicionar job de deploy Vercel:

```yaml
deploy-admin:
    name: Deploy Admin SPA to Vercel
    runs-on: ubuntu-latest
    needs: [build-web]
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
          vercel-args: '--prod'
          working-directory: web
```

### Etapa 5 — Ajustar variáveis de ambiente

**Criar/Otimizar:**

| Arquivo | Conteúdo |
|---------|----------|
| `web/.env.production` | `VITE_API_BASE=/api/v1` |
| `web/.env.staging` | `VITE_API_BASE=https://staging-api.nexora.com/api/v1` |
| Vercel Dashboard | `VERCEL_TOKEN`, `VERCEL_ORG_ID`, `VERCEL_PROJECT_ID` |

### Etapa 6 — Configurar deploy da API Go

**Opção recomendada:** Docker + VPS (qualquer provider com Docker)

- Usar `deploy/Dockerfile` com target `prod`
- Porta 8080 atrás de reverse proxy (Caddy/Nginx) com TLS
- Health check: `GET /api/v1/health`
- Banco: PostgreSQL gerenciado (Supabase, Neon, Railway) ou self-hosted
- Redis: Upstash ou self-hosted

**Arquivo a modificar:**

`internal/pkg/config/config.go` — Adicionar suporte a `CORS_ORIGINS` via env var:

```go
// Novo campo em Config.Server
AllowedOrigins []string

// No Load()
allowedOrigins := getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001")
cfg.Server.AllowedOrigins = strings.Split(allowedOrigins, ",")
```

### Etapa 7 — Segurança: Refresh Token em httpOnly Cookie

**Arquivos a modificar:**

| Arquivo | O que fazer |
|---------|-------------|
| `internal/modules/auth/service.go` | Adicionar endpoint `POST /auth/refresh/cookie` que seta httpOnly cookie |
| `web/src/api/client.ts` | Mudar `attemptRefresh()` para usar cookie em vez de localStorage |
| `internal/api/rest/router.go` | Adicionar `"Set-Cookie"` em `ExposedHeaders` |

---

## 8. Resumo de Arquivos a Criar/Modificar

### Arquivos a criar:
| # | Arquivo | Finalidade |
|---|---------|-----------|
| 1 | `web/vercel.json` | Rewrites `/*` para SPA + `/api/*` para backend |
| 2 | `web/.env.production` | `VITE_API_BASE=/api/v1` |

### Arquivos a modificar:
| # | Arquivo | Mudança |
|---|---------|---------|
| 3 | `web/src/api/client.ts` | Trocar `const API_BASE = "/api/v1"` por `import.meta.env.VITE_API_BASE || "/api/v1"` |
| 4 | `internal/api/rest/router.go` | Adicionar `X-Site-ID` em `AllowedHeaders`; `AllowCredentials: true`; origins dinâmicas via env var |
| 5 | `internal/pkg/config/config.go` | Adicionar campo `Server.AllowedOrigins` com parse de env var `CORS_ORIGINS` |
| 6 | `.github/workflows/ci.yml` | Adicionar job `deploy-admin` com Vercel action |
| 7 | `deploy/Dockerfile.admin` | Adicionar stage `prod` com `npm run build` + `npm run preview` (opcional, se quiser manter Docker) |

---

## 9. Matriz de Decisão

| Critério | Opção A (Rewrite) | Opção B (Same-Origin) | Opção C (Serverless Go) | Opção D (Go serve Admin) |
|----------|:---:|:---:|:---:|:---:|
| CORS zero-config | ❌ | ✅ | ✅ | ✅ |
| CDN Vercel | ✅ | ✅ | 🟡 | ❌ |
| Manutenção Go API | ✅ | ✅ | 🔴 | ✅ |
| Deploys independentes | ✅ | ✅ | 🔴 | ❌ |
| Custo infra | 🟢 | 🟢 | 🔴 | 🟢 |
| Esforço de migração | 🟢 | 🟢 | 🔴 | 🟡 |
| **Total** | **✅** | **✅✅** | **❌** | **🟡** |

---

## 10. Conclusão Final

**Arquitetura recomendada: Admin SPA na Vercel + Go API em servidor separado com Vercel Rewrites.**

- Admin SPA será servido como static site na Vercel (CDN edge)
- Vercel Rewrites em `vercel.json` roteiam `/api/v1/*` para o backend Go
- Go API permanece em Docker (VPS/Railway/Fly.io) sem modificações estruturais
- CORS precisa de ajustes mínimos (adicionar `X-Site-ID`, trocar `AllowCredentials`)
- CI/CD da Vercel separado do deploy da API
- 7 etapas de implementação, 7 arquivos para criar/modificar
- Esforço estimado: 2-3 dias de desenvolvimento
