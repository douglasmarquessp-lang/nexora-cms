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
