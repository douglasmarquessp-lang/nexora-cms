# Sprint 3.13 — Admin SPA Foundation

**Data:** 2026-07-30
**Propósito:** Implementar a fundação do painel administrativo do Nexora CMS.

---

## 1. Arquitetura Final do Admin SPA

```
web/
├── components.json                    # shadcn/ui configuration
├── package.json                       # Dependencies (added radix + cva + sonner)
├── tailwind.config.js                 # Extended with shadcn CSS variable colors
├── src/
│   ├── main.tsx                       # QueryClientProvider + BrowserRouter (unchanged)
│   ├── App.tsx                        # Routes with ProtectedRoute + AdminLayout
│   ├── api/
│   │   └── client.ts                  # Centralized fetch with auth + X-Site-ID + refresh
│   ├── stores/
│   │   ├── auth.ts                    # Zustand: user, login, logout, checkAuth
│   │   └── site.ts                    # Zustand: sites list, currentSite, fetchSites
│   ├── components/
│   │   ├── AdminLayout.tsx            # Sidebar + Header + Outlet + Toaster
│   │   ├── Sidebar.tsx                # Responsive nav with sections
│   │   ├── Header.tsx                 # SiteSwitcher + User dropdown + logout
│   │   ├── SiteSwitcher.tsx           # Select component for site selection
│   │   ├── ProtectedRoute.tsx         # Auth guard with redirect preservation
│   │   ├── LoadingState.tsx           # Loading with skeleton/spinner variants
│   │   ├── ErrorState.tsx             # Error display with retry button
│   │   ├── EmptyState.tsx             # Empty state with action slot
│   │   └── ui/                        # shadcn/ui components
│   │       ├── button.tsx
│   │       ├── input.tsx
│   │       ├── label.tsx
│   │       ├── select.tsx
│   │       ├── dropdown-menu.tsx
│   │       ├── dialog.tsx
│   │       ├── card.tsx
│   │       ├── table.tsx
│   │       ├── skeleton.tsx
│   │       ├── sheet.tsx
│   │       └── sonner.tsx
│   ├── pages/
│   │   ├── Login.tsx                  # Migrated: uses shadcn Card/Button/Input
│   │   ├── Dashboard.tsx              # Migrated: uses AdminLayout via Outlet
│   │   ├── MediaLibrary.tsx           # Migrated: uses AdminLayout, auth removed
│   │   ├── Plugins.tsx                # Migrated: uses AdminLayout, auth removed
│   │   ├── NotFound.tsx               # Migrated: uses shadcn Button
│   │   └── workflow/
│   │       └── Dashboard.tsx          # Migrated: uses AdminLayout, auth removed
│   └── __tests__/
│       ├── auth-store.test.ts         # 6 tests
│       ├── SiteStore.test.ts          # 6 tests
│       ├── api-client.test.ts         # 7 tests
│       ├── ProtectedRoute.test.tsx    # 3 tests
│       └── SiteSwitcher.test.tsx      # 4 tests
```

---

## 2. Componentes Criados

| Componente | Arquivo | Propósito |
|---|---|---|
| AdminLayout | `components/AdminLayout.tsx` | Layout global com sidebar + header + main outlet |
| Sidebar | `components/Sidebar.tsx` | Navegação lateral com seções e estado ativo |
| Header | `components/Header.tsx` | Top bar com SiteSwitcher + dropdown de usuário |
| SiteSwitcher | `components/SiteSwitcher.tsx` | Seletor de site com persistência |
| ProtectedRoute | `components/ProtectedRoute.tsx` | Guard de rota com redirect preservado |
| LoadingState | `components/LoadingState.tsx` | Loading com variantes (full/inline/card) |
| ErrorState | `components/ErrorState.tsx` | Estado de erro com retry |
| EmptyState | `components/EmptyState.tsx` | Estado vazio com slot para ação |

### shadcn/ui Components (11)

Button, Input, Label, Select, DropdownMenu, Dialog, Card, Table, Skeleton, Sheet, Sonner

---

## 3. Fluxo de Autenticação

```
LoginPage
  ├── checkAuth() (redirect if already logged in)
  ├── login(email, password)
  │     └── POST /api/v1/auth/login
  │     └── Store access_token + refresh_token in localStorage
  │     └── Set user in Zustand auth store
  └── Navigate to ?redirect param or /admin/dashboard

ProtectedRoute
  ├── checkAuth() on mount
  ├── If not authenticated + not loading → redirect to /admin/login?redirect=<current>
  └── If authenticated → render AdminLayout

AdminLayout
  ├── checkAuth() on mount
  ├── If not authenticated → redirect to /admin/login
  ├── If authenticated → fetchSites() (load available sites)
  └── Render Sidebar + Header + <Outlet />

Logout
  ├── Clear access_token + refresh_token from localStorage
  ├── Clear user in auth store
  └── Navigate to /admin/login
```

---

## 4. Fluxo de Refresh Token

```
API Client (api/client.ts)
  │
  ├── Request → 401 Unauthorized?
  │     ├── YES → attemptRefresh()
  │     │         ├── POST /api/v1/auth/refresh { refresh_token }
  │     │         ├── Success → store new tokens, retry original request
  │     │         └── Fail → forceLogout() → redirect to /admin/login
  │     └── NO → return response
  │
  ├── Multiple simultaneous 401s?
  │     └── Single refreshPromise shared (only one refresh at a time)
  │
  └── Refresh token rotation (backend):
        Old refresh_token → DELETE session → CREATE new session
        Old token cannot be reused
```

### Backend Contract (confirmed via audit)
- **Endpoint:** `POST /api/v1/auth/refresh` (public, no auth middleware)
- **Input:** `{ "refresh_token": "string" }`
- **Output:** Same as login `AuthResponse` with new `access_token`, `refresh_token`, `user`
- **Rotation:** Yes — old session deleted, new session created
- **Error 401:** `INVALID_TOKEN` or `SESSION_EXPIRED`
- **expires_in:** Configurable JWT_ACCESS_TTL (seconds)

---

## 5. Fluxo de Seleção de Site

```
AdminLayout mounts
  └── fetchSites()
        └── GET /api/v1/sites (returns all non-deleted sites)
        └── Store in Zustand site store
        └── Restore persisted site from localStorage (current_site_id)
        └── If no persisted site → select first available

SiteSwitcher (in Header)
  └── Shows current site name
  └── On change → setCurrentSite(site)
        └── Persist to localStorage
        └── All subsequent API calls include X-Site-ID header

API Client
  └── Before each request:
        ├── Read currentSite.id from site store (useSiteStore.getState())
        └── Set header: X-Site-ID: <currentSite.id>
```

### Limitação Conhecida
O endpoint `GET /sites` não filtra por `userID` (o parâmetro é recebido mas ignorado na query SQL). O frontend lista todos os sites. A segurança é garantida pelo backend (Casbin + RLS) — se o usuário não tem acesso a um site, o backend retornará 403. Em sprint futuro, adicionar `site_members` ou filtrar por `owner_id`.

---

## 6. Estratégia de X-Site-ID

**Centralizada no API Client.** Nenhuma página precisa definir `X-Site-ID` manualmente:

```typescript
// api/client.ts
const siteState = useSiteStore.getState();
if (siteState.currentSite) {
  headers["X-Site-ID"] = siteState.currentSite.id;
}
```

### Mecanismo de Resolução do Backend
1. `IdentifySite` middleware (site.go) — lê `X-Site-ID` header ou Host header
2. `RequireAuth` middleware (auth.go) — valida JWT, extrai userID
3. `RLSContext` middleware (rls.go) — define `app.current_user_id`, `app.current_user_role`, `app.current_site_id` no PostgreSQL
4. Casbin `RequirePermission` — verifica `role:domain:obj:act`

---

## 7. Integração com Casbin/RLS

O frontend não tem acesso direto ao Casbin. A segurança funciona em 3 camadas:

| Camada | Mecanismo | Onde |
|---|---|---|
| Autenticação | JWT Bearer → ValidateAccessToken | `middleware/auth.go` |
| Site Isolation | X-Site-ID header + IdentifySite | `middleware/site.go` |
| Row-Level Security | RLSContext → Postgres session vars | `middleware/rls.go` |
| Autorização | Casbin Enforce(role, domain, obj, act) | `middleware/authz.go` |

O frontend envia `X-Site-ID` e `Authorization: Bearer <token>`. O backend decide se a operação é permitida.

---

## 8. Páginas Migradas

| Página | Antes | Depois |
|---|---|---|
| Login | auth check inline, raw HTML | shadcn Card + Button + Input, redirect param |
| Dashboard | checkAuth + navigate inline | Removed auth (via ProtectedRoute), shadcn Card |
| MediaLibrary | checkAuth + navigate inline | Removed auth, shadcn Card + Input, shared EmptyState |
| Plugins | checkAuth + navigate inline | Removed auth, shadcn Card + Input + Button |
| Workflow | checkAuth + navigate inline | Removed auth, shadcn Card + Button |
| NotFound | raw HTML | shadcn Button |

### Funcionalidade Preservada
- Media: Upload, pastas, grid/list, rename, delete, search, move ✅
- Plugins: Lista, install, activate, deactivate, details ✅
- Workflow: Overview, jobs, queue, notifications, quick actions ✅
- Dashboard: Health status cards ✅
- Login: Email/password login ✅

---

## 9. Testes Criados

| Arquivo | Tests | Status |
|---|---|---|
| `auth-store.test.ts` | 6 | Criados, não executados |
| `SiteStore.test.ts` | 6 | Criados, não executados |
| `api-client.test.ts` | 7 | Criados, não executados |
| `ProtectedRoute.test.tsx` | 3 | Criados, não executados |
| `SiteSwitcher.test.tsx` | 4 | Criados, não executados |
| **Total** | **26** | **Não executados (shell indisponível)** |

### Dependências de Teste
Para executar os testes, instalar:
```
npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom
```

---

## 10. Pacotes Instalados (adicionados ao package.json)

| Pacote | Versão | Uso |
|---|---|---|
| `class-variance-authority` | ^0.7.1 | shadcn Button variants |
| `@radix-ui/react-slot` | ^1.1.0 | shadcn Button asChild |
| `@radix-ui/react-label` | ^2.1.0 | shadcn Label |
| `@radix-ui/react-dialog` | ^1.1.0 | shadcn Dialog |
| `@radix-ui/react-dropdown-menu` | ^2.1.0 | shadcn DropdownMenu |
| `@radix-ui/react-select` | ^2.1.0 | shadcn Select + SiteSwitcher |
| `@radix-ui/react-sheet` | ^1.1.0 | shadcn Sheet (mobile sidebar) |
| `sonner` | ^1.7.0 | Toast notifications |

---

## 11. Limitações Conhecidas

1. **Shell/bash indisponível** — npm install, go build, go vet, go test não executados
2. **Site listing sem filtro por usuário** — `GET /sites` retorna todos os sites. O `userID` é passado mas ignorado na SQL. Para multi-tenancy real, é necessário adicionar `site_members` ou filtrar por `owner_id`
3. **Nenhum teste foi executado** — arquivos de teste criados, mas sem runner configurado
4. **Páginas de navegação placeholder** — Sidebar inclui links para Conteúdo, Categorias, Sites, AI, Relatórios, Configurações que ainda não existem (BACKEND ONLY por enquanto)
5. **Sem lazy loading/code splitting** — todas as páginas são carregadas no bundle inicial
6. **Sem dark mode** — cores fixas no CSS (modo claro apenas)
7. **Sonner sem next-themes** — `Toaster` usa `theme="light"` fixo (sem dependência de next-themes)

---

## 12. Próximo Sprint Recomendado

**Sprint 3.14 — Gerenciamento de Sites (Admin UI)**
- Páginas CRUD de sites (listar, criar, editar, excluir)
- Gerenciamento de domínios
- Configurações de site
- Sincronização com SiteSwitcher
- Validação de acesso (mostrar apenas sites que o usuário pode gerenciar)
