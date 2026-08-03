# Sprint 0 — Diagnóstico e Correção do Acesso/Login do Admin SPA

**Data:** 2026-08-03
**Objetivo:** Diagnosticar e corrigir exclusivamente o acesso/login do Admin SPA.

---

## 1. Veredicto da Auditoria

| Item | Veredicto |
|---|---|
| Página /admin/login | Funcional — `LoginPage` com fluxo MFA em 2 passos |
| Frontend alcança backend | Sim — proxy Vite `/api` → `API_PROXY_TARGET` (default `http://localhost:8080`), `changeOrigin: true` |
| Payload do login | **Match exato** — frontend envia `{email, password, mfa_code?}`; backend `LoginRequest` idem |
| Contrato de resposta | **Match exato** — `AuthResponse` snake_case: `access_token`, `refresh_token`, `token_type`, `expires_in`, `user` |
| Tratamento de erros no frontend | 401/400/403/500 → `ApiError` com `error.error.code/message`; backend `rest.Context.Error` gera o mesmo formato |
| MFA | `mfa_required` → 200 `{status:"mfa_required"}` → frontend 2ª etapa; código `mfa_code` |
| Armazenamento de sessão | `localStorage` (`access_token`/`refresh_token`) + Zustand; `checkAuth` valida via `/auth/me` |
| Refresh | Interceptor único (shared promise), rotação no backend (delete+create session) |
| Logout | `POST /auth/logout` (RequireAuth) revoga TODAS as sessões do usuário + limpa localStorage |
| X-Site-ID | Enviado automaticamente quando `currentSite` definido; middleware `IdentifySite` tolerante (uuid.Nil) |
| CORS | `localhost:3000`/`3001` — irrelevante em dev (proxy same-origin) |
| Banco/schema | `users`/`sessions`/`mfa_configs` sem RLS — consultas de login não afetadas |
| Tokens | HMAC-SHA256 (`userID:purpose:expires.sig`), access 15m, refresh 7d |

**Conclusão:** **Nenhum bug de código no fluxo de autenticação** (backend e frontend).
O contrato está alinhado ponta a ponta. O bloqueio real é operacional (abaixo).

---

## 2. Causa Raiz do "não consigo logar"

O fluxo de código está completo, mas **o login só funciona se um usuário existir no banco**.
Causas prováveis, em ordem de probabilidade:

1. **Nenhum usuário admin existe** — o `users` está vazio. O login retorna sempre
   `401 INVALID_CREDENTIALS`. O Nexora NÃO tem usuário padrão: o primeiro
   usuário (super_admin) é criado exclusivamente pelo fluxo `POST /api/v1/setup/install`
   (módulo `setup`), que também cria o site padrão. Esse fluxo não era documentado
   e o Admin SPA não tem página de setup (propositalmente fora do escopo).
2. **`JWT_SECRET` com o valor padrão** — se o `.env` foi copiado do `.env.example`
   sem trocar `JWT_SECRET=change-me-...`, `config.Load()` retorna erro e a API
   **nem inicia** (`failed to load config`). O frontend então mostra erro de conexão.
3. **Postgres fora do ar** — o `main.go` sobe em "degraded mode" sem banco; o
   `auth.Service` com `db == nil` faz `findUserByEmail` falhar → `401 INVALID_CREDENTIALS`
   mesmo com credenciais corretas (sintoma enganoso).
4. **Credenciais erradas / MFA ativado sem conhecimento** — fluxo normal de 401.

---

## 3. Correções Aplicadas (mínimas, somente documentação/testes)

### 3.1 README.md — "Primeiro Acesso (criar o usuário administrador)"
Documentado o mecanismo existente (nenhum código novo):
- `GET /api/v1/setup/status` → verificar `installed: false`
- `POST /api/v1/setup/install` → cria admin (super_admin) + site padrão + vínculo
- `POST /api/v1/setup/finish` → confirmar
- Requisitos de senha (8–128, maiúscula, minúscula, número, especial)
- Aviso do `JWT_SECRET`

### 3.2 web/.env.example (novo)
- `API_PROXY_TARGET=http://localhost:8080` documentado
- Nota sobre o proxy Vite (same-origin, sem CORS em dev) e nginx em produção

### 3.3 web/src/__tests__/auth-store.test.ts (2 testes novos)
- Login inválido (401 `INVALID_CREDENTIALS`): rejeita, não armazena tokens
- Erro de conexão (`Failed to fetch`): rejeita, sessão intacta
- Mock do `@/api/client` agora exporta `ApiError` (necessário para os testes)

Nenhum teste existente foi removido. Nenhum mock para mascarar problema real.

---

## 4. Nada Alterado (por escopo)

- Nenhum endpoint backend alterado (contrato preservado)
- Nenhuma alteração em `Login.tsx`, `auth.ts`, `client.ts`, `ProtectedRoute`,
  `AdminLayout`, `Header` (já corretos)
- Nenhum commit, nenhum deploy
- Sem página de cadastro público; sem sistema de usuários novo

---

## 5. Fase 7/8 — Validação e Teste Manual (NÃO executados)

O ambiente NÃO permite executar shell (todo comando dá timeout — nem `true` roda).
Pendências obrigatórias em máquina funcional:

```bash
# Backend
go build ./...
go vet ./...
go test ./internal/modules/auth/...
go test ./internal/api/middleware/...

# Frontend
cd web && npm install
npm test
npm run build
npm run lint

# Manual (com Postgres + API + SPA no ar)
curl http://localhost:8080/api/v1/setup/status        # installed: false
curl -X POST http://localhost:8080/api/v1/setup/install -H "Content-Type: application/json" -d '{...}'
# Abrir http://localhost:3000/admin/login → logar → /admin/dashboard → /auth/me → refresh → logout
```

---

## 6. Checklist Fase 2 (12 pontos)

1. /admin/login abre? **Sim** (código verificado)
2. Frontend alcança backend? **Sim** (proxy Vite)
3. Proxy /api funcionando? **Sim** (config verificada)
4. POST /auth/login responde? **Sim** (handler/service verificados)
5. Payload = DTO? **Sim** (match exato)
6. Tratamento de erros? **Sim** (ApiError + mfa_required)
7. Pós-login (tokens, user, Zustand, navegação)? **Sim**
8. ProtectedRoute + /auth/me + Bearer + X-Site-ID? **Sim**
9. Refresh após expiração? **Sim** (interceptor + rotação)
10. Logout revoga sessão? **Sim** (`deleteUserSessions`)
11. Usuário válido no banco? **Provavelmente NÃO** — precisa rodar setup/install
12. Config local? **Parcial** — faltava `web/.env.example` (criado)

---

## 7. Arquivos Tocados

| Arquivo | Mudança |
|---|---|
| `README.md` | Seção "Primeiro Acesso" (documentação do setup) |
| `web/.env.example` | **Novo** — API_PROXY_TARGET documentado |
| `web/src/__tests__/auth-store.test.ts` | +2 testes (login inválido, erro de conexão) + ApiError no mock |
| `research/sprint-0-auth-diagnosis.md` | Este relatório |
