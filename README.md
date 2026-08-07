# Nexora CMS

> Plataforma de gerenciamento de conteúdo multi-site com IA, módulos independentes e sistema de plugins.

**Status:** Em desenvolvimento (v0.1.0)

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Backend | Go 1.26 + chi + sqlc + pgx |
| Banco | PostgreSQL 17 + pgvector |
| Cache | Redis 7 |
| Admin UI | React 19 + Vite + shadcn/ui |
| Site Frontend | Next.js 15 + Tailwind |
| IA | Python + modelos locais/OSS |
| Plugins | WebAssembly (WASM) |
| Infra | Docker Compose |

---

## Estrutura do Projeto

```
nexora/
├── cmd/                     # Entrypoints da aplicação
│   ├── api/                 # Servidor HTTP da API
│   ├── migrate/             # CLI de migrações
│   ├── frontend/            # Proxy/servidor do frontend
│   └── worker/              # Background jobs
│
├── internal/                # Código privado
│   ├── kernel/              # Núcleo do sistema
│   ├── modules/             # Módulos de negócio
│   ├── plugins/             # Sistema de plugins
│   ├── api/                 # Transport layer (HTTP)
│   └── pkg/                 # Pacotes compartilhados
│
├── web/                     # Admin SPA (React)
├── site/                    # Frontend dos sites (Next.js)
├── plugins/                 # Plugins instalados
├── migrations/              # Migrações SQL
├── deploy/                  # Configuração de deploy
└── data/                    # Dados locais (dev)
```

---

## Pré-requisitos

- Go 1.26+
- Node.js 22+
- Docker + Docker Compose
- PostgreSQL 17 (via Docker ou local)
- Redis 7 (via Docker ou local)

---

## Desenvolvimento

### 1. Clonar e configurar

```bash
git clone <repo-url> nexora
cd nexora
cp .env.example .env
```

### 2. Iniciar ambiente com Docker

```bash
make dev
```

Isso inicia: PostgreSQL, Redis, API (com hot reload), Admin SPA e Site Frontend.

### 3. Aplicar migrations

```bash
make migrate-up
```

### 4. Acessar

| Serviço | URL |
|---------|-----|
| API | http://localhost:8080 |
| Admin | http://localhost:3000 |
| Site | http://localhost:3001 |
| Health | http://localhost:8080/api/v1/health |

### 5. Comandos úteis

```bash
make build          # Compilar binário
make run            # Executar localmente
make test           # Rodar testes
make lint           # Verificar lint
make migrate-up     # Aplicar migrations
make migrate-down   # Reverter migration
make docker-up      # Iniciar containers
make docker-down    # Parar containers
```

---

## Deploy no Railway (um único serviço)

A arquitetura de produção entrega **só a API Go**, e ela própria serve o
Admin SPA compilado (`web/` → Vite → `go:embed` em `internal/webui`). Um
único domínio expõe tudo na mesma origem — **sem CORS, sem proxy, sem
segundo serviço**:

| Rota | Resposta |
|------|----------|
| `/` | Painel admin (SPA) |
| `/admin`, `/admin/login` | App (fallback do SPA para `index.html`) |
| `/api/v1/*` | API (inalterada) |
| `/ping` | Heartbeat do chi |

### Passos no Railway

1. **Build:** o repositório tem um `Dockerfile` na raiz → Railway o detecta
   automaticamente (builder de contêiner). Se o serviço já existir como
   Nixpacks/Go, troque em *Service → Settings → Build → Builder* para
   **Dockerfile** (path deixa em branco para usar a raiz).

2. **Variáveis de ambiente** (o backend não lê `DATABASE_URL`; use o mesmo
   conjunto do `.env.example`):
   `SERVER_HOST=0.0.0.0`, `DATABASE_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`,
   `JWT_SECRET` (obrigatório, não pode ser o valor padrão), `REDIS_HOST`
   (se usar), `AI_ENABLED`/`AI_GEMINI_API_KEY` (opcional).
   `SERVER_PORT` tem prioridade; se ausente, o **`PORT`** do Railway é
   usado automaticamente.

3. **Registrar no domínio público**: o Railway expõe
   `https://SEU-SERVICO.up.railway.app`. O 404 da raiz desaparece: `GET /`
   abre o painel e, sem sessão, o app redireciona para `GET /admin/login`.

4. **Primeiro acesso** (sem ele o login dá `401 INVALID_CREDENTIALS`):
   ```bash
   curl https://SEU-SERVICO.up.railway.app/api/v1/setup/status
   curl -X POST https://SEU-SERVICO.up.railway.app/api/v1/setup/install \
     -H "Content-Type: application/json" \
     -d '{"admin_name":"Administrador","admin_email":"admin@exemplo.com",
          "password":"SenhaForte@2026","site_name":"Meu Site",
          "language":"pt-BR","timezone":"America/Sao_Paulo"}'
   ```
   
5. Acesse `https://SEU-SERVICO.up.railway.app/admin` para a tela de login.

### Por que funciona sem CORS e sem `VITE_API_URL`

- O client admin (`web/src/api/client.ts`) chama caminhos **relativos**
  (`/api/v1`), resolvidos na própria origem.
- Em desenvolvimento o Vite proxy `/api → :8080` (só dev); em produção o
  mesmo domínio faz API e static.
- Não existem cookies de sessão — JWT no `localStorage` com refresh
  rotation; sem configuração extra de cookies.

### Site público (Next.js, `site/`)

O frontend público é um serviço **separado** e opcional (pode ir para
Vercel, como já previsto em `vercel.json`, ou um segundo serviço Railway
usando `NEXT_PUBLIC_API_URL` + `API_PROXY_TARGET` apontando para a API).
Ele não é necessário para o painel admin.

---

## Primeiro Acesso (criar o usuário administrador)

O Nexora não possui usuário padrão: o **primeiro usuário (super_admin) é criado pelo
fluxo de setup da API** (`/api/v1/setup/*`), que também cria o site padrão e vincula o
admin a ele. Sem isso, o login no Admin SPA sempre retorna `401 INVALID_CREDENTIALS`.

```bash
# 1. Verificar se o sistema já está instalado
curl http://localhost:8080/api/v1/setup/status
# → {"installed": false}

# 2. Instalar: cria o admin + site padrão (executar UMA única vez)
curl -X POST http://localhost:8080/api/v1/setup/install \
  -H "Content-Type: application/json" \
  -d '{
    "cms_name": "Nexora CMS",
    "admin_name": "Administrador",
    "admin_email": "admin@exemplo.com",
    "password": "SenhaForte@2026",
    "site_name": "Meu Site",
    "site_description": "Site padrão",
    "language": "pt-BR",
    "timezone": "America/Sao_Paulo",
    "site_url": "http://localhost:3001"
  }'

# 3. Confirmar instalação
curl -X POST http://localhost:8080/api/v1/setup/finish
```

**Requisitos da senha do admin:** 8–128 caracteres, com maiúscula, minúscula, número e
caractere especial. **Importante:** `JWT_SECRET` deve ser alterado do valor padrão do
`.env.example` — caso contrário a API não inicia.

Depois do setup, acesse `http://localhost:3000/admin/login` com o e-mail e a senha
definidos no passo 2.

---

## API

Base URL: `http://localhost:8080/api/v1`

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

Resposta:
```json
{
  "status": "ok",
  "version": "0.1.0",
  "timestamp": "2026-07-15T12:00:00Z",
  "database": "connected"
}
```

---

## Migrações

```bash
# Criar nova migration
make migrate-create

# Aplicar pendentes
make migrate-up

# Reverter 1 passo
make migrate-down

# Reverter N passos
make migrate-down steps=3
```

---

## Arquitetura

Consulte o documento `nexora-architecture.md` para detalhes completos da arquitetura.

Princípios:
- **Modular Monolith**: um binário, múltiplos módulos independentes
- **Event-Driven**: comunicação entre módulos via event bus
- **Multi-tenant nativo**: Row Level Security no PostgreSQL
- **Core gratuito**: SEO e IA Base são módulos do Core, sem custo
- **Plugins Premium**: apenas recursos avançados são plugins pagos

---

## ROADMAP

Consulte `ROADMAP.md` para o plano de desenvolvimento completo.

---

## Licença

Este projeto está sob licença proprietária. Todos os direitos reservados.
