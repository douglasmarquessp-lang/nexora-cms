# Auditoria do Admin SPA — Nexora CMS

**Data:** 2026-07-30
**Propósito:** Mapear o estado atual do painel administrativo antes de implementar novas funcionalidades.

---

## 1. Estrutura Atual do Admin SPA

### Localização
`/web/` — diretório raiz do Admin SPA.

### Framework
| Item | Status | Detalhes |
|------|--------|----------|
| Nome do projeto | `nexora-admin` | package.json |
| Framework | React 19 | `react@^19.0.0` |
| Linguagem | TypeScript 5.6 | strict mode, `@paths` alias `@/` |
| Bundler | Vite 6 | `vite@^6.0.0` |
| Roteamento | React Router DOM v6 | `react-router-dom@^6.28.0` |
| Estado global | Zustand v5 | `zustand@^5.0.0` — apenas auth store |
| Server state | TanStack React Query v5 | `@tanstack/react-query@^5.60.0` |
| Forms | react-hook-form v7 + zod v3 | Instalados mas **não utilizados** em nenhuma página |
| Estilização | Tailwind CSS 3.4 | Custom `brand` palette |
| Ícones | lucide-react | `lucide-react@^0.460.0` |
| Utilitários | clsx + tailwind-merge | `cn()` helper em `@/lib/utils` |
| shadcn/ui | **NÃO instalado** | `clsx` + `tailwind-merge` presentes, mas sem `components.json`, sem `components/ui/`, sem `npx shadcn` |

### Entrypoint
`web/index.html` → `web/src/main.tsx` → `web/src/App.tsx`
- `main.tsx` configura QueryClientProvider + BrowserRouter
- Sem StrictMode wrapper em produção
- Sem lazy loading, sem code splitting

### Rotas Existentes
| Path | Página | Protegida? |
|------|--------|------------|
| `/` | Redirect → `/admin/dashboard` | Não |
| `/admin/login` | LoginPage | Pública |
| `/admin/dashboard` | DashboardPage | Sim (checkAuth) |
| `/admin/workflow` | WorkflowDashboardPage | Sim (checkAuth) |
| `/admin/media` | MediaLibraryPage | Sim (checkAuth) |
| `/admin/plugins` | PluginsPage | Sim (checkAuth) |
| `*` | NotFoundPage | Pública |

### Páginas — Estado
| Página | Arquivo | Linhas | Funcionalidade |
|--------|---------|--------|----------------|
| Login | `pages/Login.tsx` | 79 | Email/senha, chamada auth store, redirect |
| Dashboard | `pages/Dashboard.tsx` | 78 | Header com user + logout, 3 cards de health |
| Media Library | `pages/MediaLibrary.tsx` | 696 | Upload, pastas, grid/list view, rename, delete, search |
| Plugins | `pages/Plugins.tsx` | 396 | Lista, instalar, ativar/desativar, remover |
| Workflow | `pages/workflow/Dashboard.tsx` | 656 | Tabs (overview/jobs/queue/notifications), Quick Actions |
| NotFound | `pages/NotFound.tsx` | 16 | 404 + link voltar |

### Componentes Compartilhados
**Não existe diretório `components/`.** Zero componentes compartilhados (tabelas, inputs, modais, sidebars, layouts). Cada página implementa seus próprios elementos inline.

### Layout
**Não existe layout compartilhado.** Cada página tem seu próprio `<header>`, `<main>`, estrutura CSS. Não há:
- Sidebar
- Navbar compartilhada
- Layout wrapper
- Breadcrumbs

### Estado Global (Zustand)
**Apenas 1 store:** `stores/auth.ts`
- `user`, `isAuthenticated`, `isLoading`
- `login()`, `logout()`, `checkAuth()`
- Tokens em `localStorage` (`access_token`, `refresh_token`)
- **Sem store de sites**, **sem store de UI** (sidebar, tema, etc.)

### API Client
`api/client.ts`
- `fetch` puro (sem axios, sem ky)
- Bearer token automático via `localStorage`
- `ApiError` class com `status`, `code`, `message`
- Métodos: `get`, `post`, `put`, `patch`, `delete`
- Content-Type: `application/json` fixo (não muda para FormData)
- **Sem refresh token automático**
- **Sem logout automático em 401**
- **Sem retry/redirect**

### Tratamento de Erros
- Login: `catch` básico, exibe mensagem
- Dashboard/Media/Plugins/Workflow: TanStack Query trata erros silenciosamente
- **Sem error boundaries** (React Error Boundary)
- **Sem toasts/sonner/notificações**
- **Sem loading skeletons** (apenas "Carregando..." texto)
- **Sem empty states padronizados**

### Loading States
- Apenas `isLoading` → `<div>Carregando...</div>` ou `Loading...`
- **Sem skeletons**, **sem spinners animados**

### Environment Variables
- `VITE_*`: nenhuma definida
- `API_PROXY_TARGET` no `vite.config.ts` (proxy `/api` para backend)

---

## 2. Estado do Backend (Endpoints)

### Endpoints Públicos
| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/health` | Health Check |
| GET | `/api/v1/articles` | List publications |
| GET | `/api/v1/articles/{slug}` | Get article by slug |
| GET | `/api/v1/categories` | List categories |

### Endpoints de Autenticação
| Method | Path | Protegido? |
|--------|------|------------|
| POST | `/api/v1/auth/register` | Não |
| POST | `/api/v1/auth/login` | Não |
| POST | `/api/v1/auth/refresh` | Não |
| POST | `/api/v1/auth/logout` | Sim |
| GET | `/api/v1/auth/me` | Sim |
| GET | `/api/v1/auth/oauth/url` | Não |
| POST | `/api/v1/auth/oauth/callback` | Não |
| POST | `/api/v1/auth/mfa/enroll` | Sim |
| POST | `/api/v1/auth/mfa/verify` | Sim |
| POST | `/api/v1/auth/mfa/disable` | Sim |

### Endpoints Administrativos (protegidos por auth + site + RLS)
| Módulo | Qtd Endpoints | Cobertura no Admin SPA |
|--------|---------------|------------------------|
| **Sites** (CRUD + domains + settings + global) | 12 | ❌ Nenhum |
| **Posts** (CRUD + autosave + status) | 9 | ❌ Nenhum |
| **Categories** (CRUD + tree) | 6 | ❌ Nenhum |
| **Tags** (CRUD) | 5 | ❌ Nenhum |
| **Assets** (CRUD + link) | 8 | ❌ Nenhum |
| **Media** (com folders, upload, move) | ~10 (via RegisterRoutes) | ✅ Lista, upload, rename, delete, folders |
| **Plugins** (list, install, activate, deactivate, delete) | ~6 | ✅ Lista, install, activate, deactivate |
| **Editorial** (dashboard, tasks, revisions, approvals, calendar, widgets) | 19 | ❌ Nenhum |
| **Editorial Engine** (pipelines, styles, SEO, quality, translations, prompts) | 17 | ❌ Nenhum |
| **Content Generator** (jobs, pipeline, quality, stats, promps) | 16 | ❌ Nenhum |
| **Autocontent** (21 endpoints via RegisterRoutes) | 21 | ❌ Nenhum |
| **Human Writer** | ~5 | ❌ Nenhum |
| **Article Pipeline** | ~5 | ❌ Nenhum |
| **Publisher** (publications CRUD) | ~5 | ❌ Nenhum |
| **SEO Engine** | ~5 | ❌ Nenhum |
| **Workflow** (dashboard, jobs, queue, notifications, metrics, actions) | ~12 | ✅ Dashboard (parcial: apenas visão geral) |
| **Research** (jobs, search, briefing) | 7 | ❌ Nenhum |
| **Writer** (styles, jobs, outlines, sections, versions) | 15 | ❌ Nenhum |
| **AI** (providers, health, test, prompt, capabilities) | 5 | ❌ Nenhum |
| **Setup** | ~3 | ❌ Nenhum |

### Total de Endpoints Administrativos
**~170+ endpoints** (excluindo públicos e auth)

### Endpoints Consumidos pelo Admin SPA
| Endpoint | Página |
|----------|--------|
| `POST /auth/login` | Login |
| `GET /auth/me` | Todas (checkAuth) |
| `GET /health` | Dashboard |
| `GET /media?...` | Media Library |
| `GET /media/folders` | Media Library |
| `POST /media/upload` | Media Library |
| `DELETE /media/{id}` | Media Library |
| `PATCH /media/{id}` | Media Library |
| `POST /media/folders` | Media Library |
| `POST /media/move` | Media Library |
| `GET /plugins` | Plugins |
| `POST /plugins/activate` | Plugins |
| `POST /plugins/deactivate` | Plugins |
| `DELETE /plugins/{id}` | Plugins |
| `POST /plugins/install` | Plugins |
| `GET /workflow/dashboard` | Workflow |
| `GET /workflow` | Workflow |
| `GET /workflow/queue` | Workflow |
| `GET /workflow/notifications` | Workflow |
| `GET /workflow/metrics` | Workflow |
| `POST /workflow/actions` | Workflow |

**Total consumido: ~21 endpoints de ~170 disponíveis (12.3%)**

---

## 3. Matriz Funcionalidade × Estado

| Funcionalidade | Backend | Frontend (Admin SPA) | Classificação |
|---------------|---------|---------------------|---------------|
| Login (email/senha) | COMPLETE | COMPLETE | **COMPLETE** |
| Login (OAuth) | COMPLETE | MISSING | BACKEND ONLY |
| MFA | COMPLETE | MISSING | BACKEND ONLY |
| Logout | COMPLETE | COMPLETE | **COMPLETE** |
| Session check (/auth/me) | COMPLETE | COMPLETE | **COMPLETE** |
| Refresh token | COMPLETE | PARTIAL (no client code) | PARTIAL |
| Listar sites | COMPLETE | MISSING | BACKEND ONLY |
| Criar site | COMPLETE | MISSING | BACKEND ONLY |
| Editar site | COMPLETE | MISSING | BACKEND ONLY |
| Excluir site | COMPLETE | MISSING | BACKEND ONLY |
| Domínios por site | COMPLETE | MISSING | BACKEND ONLY |
| Configurações de site | COMPLETE | MISSING | BACKEND ONLY |
| Configurações globais | COMPLETE | MISSING | BACKEND ONLY |
| Listar artigos (posts) | COMPLETE | MISSING | BACKEND ONLY |
| Criar artigo | COMPLETE | MISSING | BACKEND ONLY |
| Editar artigo | COMPLETE | MISSING | BACKEND ONLY |
| Publicar artigo | COMPLETE | MISSING | BACKEND ONLY |
| Excluir artigo | COMPLETE | MISSING | BACKEND ONLY |
| Autosave | COMPLETE | MISSING | BACKEND ONLY |
| Listar categorias | COMPLETE | MISSING | BACKEND ONLY |
| Criar categoria | COMPLETE | MISSING | BACKEND ONLY |
| Árvore de categorias | COMPLETE | MISSING | BACKEND ONLY |
| Listar tags | COMPLETE | MISSING | BACKEND ONLY |
| Criar/editar/excluir tags | COMPLETE | MISSING | BACKEND ONLY |
| Media Library | COMPLETE | COMPLETE | **COMPLETE** |
| Upload mídia | COMPLETE | COMPLETE | **COMPLETE** |
| Pastas de mídia | COMPLETE | COMPLETE | **COMPLETE** |
| Dashboard (health) | COMPLETE | COMPLETE | **COMPLETE** |
| Editorial Dashboard | COMPLETE | MISSING | BACKEND ONLY |
| Editorial Tasks | COMPLETE | MISSING | BACKEND ONLY |
| Editorial Calendar | COMPLETE | MISSING | BACKEND ONLY |
| Editorial Revisions | COMPLETE | MISSING | BACKEND ONLY |
| Editorial Approvals | COMPLETE | MISSING | BACKEND ONLY |
| Workflow Dashboard | COMPLETE | COMPLETE | **COMPLETE** |
| Workflow Jobs | COMPLETE | COMPLETE | **COMPLETE** |
| Workflow Queue | COMPLETE | COMPLETE | **COMPLETE** |
| Workflow Actions | COMPLETE | COMPLETE | **COMPLETE** |
| Workflow Notifications | COMPLETE | COMPLETE | **COMPLETE** |
| Plugins | COMPLETE | COMPLETE | **COMPLETE** |
| AI Providers | COMPLETE | MISSING | BACKEND ONLY |
| AI Test | COMPLETE | MISSING | BACKEND ONLY |
| AI Prompt Preview | COMPLETE | MISSING | BACKEND ONLY |
| Content Generator | COMPLETE | MISSING | BACKEND ONLY |
| Autocontent | COMPLETE | MISSING | BACKEND ONLY |
| Article Pipeline | COMPLETE | MISSING | BACKEND ONLY |
| Research | COMPLETE | MISSING | BACKEND ONLY |
| Writer | COMPLETE | MISSING | BACKEND ONLY |
| SEO Engine | COMPLETE | MISSING | BACKEND ONLY |
| Publisher | COMPLETE | MISSING | BACKEND ONLY |
| Human Writer | COMPLETE | MISSING | BACKEND ONLY |
| Editorial Engine | COMPLETE | MISSING | BACKEND ONLY |
| User management | COMPLETE | MISSING | BACKEND ONLY |
| Setup/install | COMPLETE | MISSING | BACKEND ONLY |
| Assets | COMPLETE | MISSING | BACKEND ONLY |

---

## 4. Problemas de Autenticação

1. **Refresh token não implementado no frontend.** O backend retorna `refresh_token`, e o store salva em localStorage, mas o API client nunca o utiliza. Quando o access token expira, o usuário é forçado a fazer login novamente.

2. **Sem interceptador 401.** O API client não detecta respostas 401 para tentar refresh automático ou redirecionar ao login.

3. **checkAuth em cada página via useEffect.** Cada página repete o padrão:
   ```tsx
   useEffect(() => { checkAuth(); }, [checkAuth]);
   useEffect(() => { if (!isLoading && !isAuthenticated) navigate("/admin/login"); }, [...]);
   ```
   Isso causa um flash de "Carregando..." em toda navegação. Não há um guard de rota centralizado.

4. **Token armazenado sem prefixo de site/contexto.** Em multi-tenancy, o mesmo token pode ser usado para acessar dados de qualquer site.

---

## 5. Problemas de Autorização

1. **Casbin implementado no backend mas não refletido no frontend.** O frontend não sabe o role/permissões do usuário.
2. **Sem menu adaptativo.** Não há ocultação de funcionalidades baseada em permissões.
3. **Sem forbidden handling no frontend.** Se o backend retorna 403, o frontend apenas mostra erro genérico.

---

## 6. Problemas de Multi-tenancy

1. **Admin SPA não envia `X-Site-ID`.** O middleware `IdentifySite` depende de `Host` header ou `X-Site-ID`. Vite dev proxy não define `X-Site-ID`. O frontend nunca solicita seleção de site.

2. **Sem seletor de site no Admin SPA.** Não há interface para o usuário selecionar qual site está gerenciando.

3. **Sem store de site no Zustand.** Não há estado global para `currentSiteId`, `currentSiteSlug`.

4. **Rotas não possuem site_id no path.** As URLs do admin não incluem `/admin/site/{id}/...`.

5. **Risco de vazamento de dados:** Se o backend não resolve o site (uuid.Nil), e o RLS depende de `current_site_id = uuid.Nil`, queries sem site_id podem retornar dados de múltiplos sites. Vários módulos têm `site_id` parcial em queries (conforme auditado no Sprint 3.7).

6. **Frontend público (`site/`) envia Host, que é resolvido pelo IdentifySite.** Funciona para multi-tenancy público mas não para o admin.

---

## 7. Estado do Deploy

| Item | Status |
|------|--------|
| Dockerfile API (Go) | ✅ Existe |
| Dockerfile Admin | ✅ Existe (dev apenas) |
| Dockerfile Site | ✅ Existe (dev apenas) |
| docker-compose | ✅ Existe |
| Nginx config | ✅ Existe (admin.nexora.local / api.nexora.local / *.nexora.local) |
| Vercel config | ❌ Não existe |
| GitHub Actions CI | ✅ Existe |
| Scripts build admin | `npm run build` (tsc + vite) |
| Scripts build site | `next build` |
| Admin SPA publicado? | ❌ Não há pipeline de publicação |
| Admin SPA + Site separados? | ✅ Projetos separados (`web/` e `site/`) |
| Admin SPA rota /admin | ✅ Rotas administradas pelo React Router **no próprio SPA**, não no backend |

### Observações de Deploy
- `Dockerfile.admin` é apenas dev (cópia de package.json, sem build stage)
- Nginx mapeia `admin.nexora.local` → admin:3000 (Vite dev server)
- Sem stage de produção para o admin SPA (sem build + nginx serve static)
- CI.yml executa `go test` e `go build` para API, sem menção ao frontend

---

## 8. Variáveis de Ambiente Necessárias

### Admin SPA (web/)
| Variável | Onde | Obrigatória? |
|----------|------|-------------|
| `API_PROXY_TARGET` | vite.config.ts | Não (default localhost:8080) |

### Backend (Go)
| Variável | Obrigatória? |
|----------|-------------|
| `DATABASE_*` (host, port, user, password, name) | Sim |
| `JWT_SECRET` | Sim |
| `JWT_ACCESS_TTL` | Sim |
| `JWT_REFRESH_TTL` | Sim |

### Frontend Público (site/)
| Variável | Obrigatória? |
|----------|-------------|
| `NEXT_PUBLIC_API_URL` | Sim |

---

## 9. Arquitetura Planejada vs Real

| Item Planejado | Real | Onde |
|----------------|------|------|
| Admin SPA | ✅ `web/` | Vite + React 19 |
| React | ✅ | 19.0 |
| TypeScript | ✅ | strict mode |
| React 19 | ✅ | 19.0 |
| Vite | ✅ | 6.0 |
| Tailwind | ✅ | 3.4 |
| shadcn/ui | ❌ Não instalado | Apenas clsx + twMerge |
| Zustand | ✅ | 5.0 (apenas auth) |
| TanStack Query | ✅ | 5.60 |
| Autenticação | ✅ | Backend + Frontend (parcial) |
| Gerenciamento de sites | ❌ Frontend ausente | Backend COMPLETE |
| Gerenciamento de artigos | ❌ Frontend ausente | Backend COMPLETE |
| Mídia | ✅ | Frontend + Backend COMPLETE |
| Categorias | ❌ Frontend ausente | Backend COMPLETE |
| Tags | ❌ Frontend ausente | Backend COMPLETE |
| SEO | ❌ Frontend ausente | Backend COMPLETE |
| Workflows | ✅ Frontend parcial | Workflow Dashboard apenas |
| AI | ❌ Frontend ausente | Backend COMPLETE |
| Configurações | ❌ Frontend ausente | Backend COMPLETE |
| Plugins | ✅ | Frontend + Backend COMPLETE |

---

## 10. Roadmap Recomendado

### Fase 1 — Fundação (Sprint 3.13)
1. **Layout compartilhado** — Sidebar + Header + Layout wrapper (React Router outlet)
2. **Proteção de rotas centralizada** — Route guard component (substituir useEffect em cada página)
3. **Refresh token automático** — Interceptador 401 no API client
4. **Seleção de site** — Site store (Zustand), seletor no header, envio de `X-Site-ID`
5. **shadcn/ui** — Setup oficial (components.json, Button, Card, Input, etc.)

### Fase 2 — Núcleo (Sprints 3.14-3.15)
6. **Gerenciamento de artigos** — Lista + Criar/Editar (react-hook-form + zod + editor)
7. **Gerenciamento de categorias** — CRUD + árvore
8. **Gerenciamento de tags** — CRUD
9. **Gerenciamento de sites** — CRUD + domínios + configurações

### Fase 3 — Workflows + AI (Sprint 3.16)
10. **AI Dashboard** — List providers, test, prompt preview
11. **Content Generator UI** — Full pipeline management
12. **Research UI** — Search, topics, grounding

### Fase 4 — Editorial + SEO (Sprint 3.17)
13. **Editorial Dashboard** — Tasks, calendar, revisions, approvals
14. **SEO Engine UI** — Meta analysis, suggestions
15. **Publisher UI** — Queue, schedule, publications

### Fase 5 — Produção (Sprint 3.18)
16. **Build production do Admin SPA** (Dockerfile multi-stage, nginx serve static)
17. **GitHub Actions** — Deploy admin + site + API
18. **Testes E2E** com Playwright/Cypress

---

## 11. Resumo Numérico

| Métrica | Valor |
|---------|-------|
| Total endpoints backend | ~170+ |
| Endpoints consumidos pelo Admin SPA | ~21 (12.3%) |
| Páginas Admin SPA | 5 |
| Páginas ausentes | ~25+ |
| Componentes compartilhados | 0 |
| Stores Zustand | 1 (auth) |
| shadcn/ui componentes | 0 |
| Layouts | 0 |
| Error Boundaries | 0 |
| Loading skeletons | 0 |
| Testes frontend | 0 |

---

## 12. Checklist Final

```
Admin SPA:                  PARTIAL (existe mas cobre ~15% das funcionalidades)
Login:                      COMPLETE
Dashboard:                  PARTIAL (apenas health, sem métricas reais)
Gerenciamento de sites:     BACKEND ONLY
Gerenciamento de artigos:   BACKEND ONLY
Gerenciamento de mídia:     COMPLETE
Categorias:                 BACKEND ONLY
Tags:                       BACKEND ONLY
SEO:                        BACKEND ONLY
AI:                         BACKEND ONLY
Workflows:                  COMPLETE (Frontend + Backend)
Plugins:                    COMPLETE (Frontend + Backend)
Editorial:                  BACKEND ONLY
Configurações:              BACKEND ONLY
shadcn/ui:                  MISSING
Layout responsivo:          PARTIAL (páginas isoladas, sem consistência)
Refresh token:              MISSING
Multi-tenancy frontend:     MISSING
```

## Observações Finais

- O shell/bash estava indisponível para execução. Nenhum build, vet ou teste foi executado.
- A auditoria é estritamente estrutural baseada em leitura de arquivos.
- Go toolchain não verificada.
- ~170 endpoints backend disponíveis, ~21 consumidos pelo Admin SPA.
- Backend está robusto e completo. O gap principal é o frontend administrativo.
