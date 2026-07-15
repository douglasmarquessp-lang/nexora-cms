# 🗺️ Nexora CMS — Roadmap Oficial de Desenvolvimento

> **Documento Mestre do Projeto**  
> Versão 1.0 — Planejamento estratégico para substituição do WordPress  
> Plataforma multi-site com IA, módulos independentes e sistema de plugins

---

## Sumário

- [Visão Geral](#visão-geral)
- [Fase 1 — Fundação](#fase-1--fundação)
- [Fase 2 — CMS](#fase-2--cms)
- [Fase 3 — SEO Core](#fase-3--seo-core)
- [Fase 4 — IA Base](#fase-4--ia-base)
- [Fase 5 — Dashboard Inteligente](#fase-5--dashboard-inteligente)
- [Fase 6 — Plugins Premium](#fase-6--plugins-premium)
- [Fase 7 — Escalabilidade](#fase-7--escalabilidade)
- [Cronograma Consolidado](#cronograma-consolidado)
- [Decisões Arquiteturais](#decisões-arquiteturais)
- [Glossário](#glossário)

---

## Visão Geral

### Stack Tecnológica

| Camada | Tecnologia | Finalidade |
|--------|-----------|------------|
| Backend | Go + chi + sqlc + pgx | Performance, concorrência, tipagem forte |
| Banco | PostgreSQL 17 + pgvector | Dados relacionais + busca semântica |
| Cache | Redis | Sessões, rate limit, cache de queries |
| Fila | river (PG nativo) | Background jobs sem dependência extra |
| Admin UI | React 19 + Vite + shadcn/ui | Interface administrativa |
| Site Frontend | Next.js 19 + Tailwind | Frontend público com ISR |
| Plugins | Go WASM | Sandbox seguro para extensões |
| AI Core | Python + modelos locais/OSS | IA gratuita sem APIs pagas |
| Storage | Local FS / S3 / MinIO | Flexibilidade de armazenamento |

### Princípios do Projeto

1. **Core gratuito e autossuficiente** — Nenhuma funcionalidade essencial depende de serviços pagos.
2. **Plugins Premium = apenas extras** — Recursos avançados são opcionais.
3. **IA e SEO Base são módulos do Core** — Disponíveis sem custo para todos os usuários.
4. **Modular Monolith primeiro** — Simplicidade e performance; microsserviços apenas quando necessário.
5. **Multi-site nativo** — Cada instância gerencia múltiplos sites com isolamento total.

### Legenda

| Ícone | Significado |
|-------|-------------|
| 🔴 | Prioridade Crítica |
| 🟡 | Prioridade Alta |
| 🟢 | Prioridade Média |
| 🔵 | Prioridade Baixa |

---

## Fase 1 — Fundação

> **Duração:** Semanas 1 a 6 (Mês 1 a 1.5)  
> **Versão alvo:** v0.1.0

### Objetivo

Estabelecer a base do sistema: kernel do CMS, infraestrutura de banco de dados, autenticação, multi-site e sistema de permissões. Ao final desta fase, o Nexora deve ser capaz de gerenciar múltiplos sites com usuários, autenticação e configurações isoladas.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Arquitetura** | Setup do monorepo (Go + Next.js + React Admin), estrutura de pastas, tooling (linter, codegen, testes) | 🔴 Crítica |
| **Banco de Dados** | PostgreSQL 17, schema multi-tenant (RLS), migrações versionadas com `golang-migrate`, codegen `sqlc` | 🔴 Crítica |
| **Autenticação** | JWT (access + refresh tokens), OAuth2 (Google, GitHub), MFA (TOTP), Argon2id para senhas | 🔴 Crítica |
| **Usuários** | CRUD de usuários, perfil, avatar, recuperação de senha, session management | 🔴 Crítica |
| **Permissões** | Casbin RBAC/ABAC, roles (SuperAdmin, SiteAdmin, Editor, Author, Subscriber), permissões granulares por módulo | 🟡 Alta |
| **Multi-site** | CRUD de sites, isolamento por `site_id` com RLS, domínios customizados, settings por site (JSONB) | 🔴 Crítica |
| **Configurações** | Configurações globais e por site, feature flags, cache de configurações | 🟡 Alta |

### Dependências

- Go 1.23+, Node 22+, PostgreSQL 17
- Docker + Docker Compose para ambiente dev
- Nenhuma dependência externa de API

### Sub-fases

#### Sprint 1.1 — Setup do Projeto (Semanas 1-2)
- Inicialização do monorepo
- Estrutura de pastas conforme `nexora-architecture.md`
- Configuração de linter (golangci-lint, ESLint), formatador (gofumpt, Prettier)
- Pipeline CI básico (GitHub Actions: build, lint, test)
- Docker Compose com PostgreSQL + Redis
- Configuração do `sqlc` e `golang-migrate`

**Critérios de conclusão:**
- `make build` gera binário funcional
- `docker compose up` sobe ambiente completo
- CI passa com `make lint && make test`
- Migration inicial aplica schema vazio

#### Sprint 1.2 — Kernel + Autenticação (Semanas 3-4)
- Implementação do Kernel: lifecycle, module registry, event bus
- Módulo de autenticação: register, login, logout, refresh token, OAuth2
- Middleware de rate limiting (sliding window, Redis)
- Validação dupla (Zod frontend + Go backend)

**Critérios de conclusão:**
- `POST /auth/register` cria usuário com senha hash Argon2id
- `POST /auth/login` retorna JWT válido
- `POST /auth/refresh` renova token
- OAuth2 funciona com Google e GitHub
- Rate limit bloqueia após N tentativas

#### Sprint 1.3 — Multi-site + Permissões (Semanas 5-6)
- CRUD completo de sites com RLS
- Sistema de permissões com Casbin: roles, permissões por módulo
- Usuário de um site não acessa dados de outro (testado)
- Configurações globais e por site com cache

**Critérios de conclusão:**
- API de sites completa (CRUD)
- RLS PostgreSQL funcional: cross-site leak = zero
- Casbin integrado: `site_admin` não gerencia outro site
- `GET /system/config` e `PUT /system/config` funcionais
- Feature flags operacionais

### Critérios para considerar a Fase 1 concluída

- [ ] 100% das migrations de infra aplicadas e reversíveis
- [ ] Pipeline CI/CD verde para todos os PRs
- [ ] Cobertura de testes > 60% no kernel e módulos da fase
- [ ] Autenticação + OAuth2 + MFA validados com testes E2E
- [ ] Isolamento multi-site verificado (teste de penetração básico)
- [ ] Permissões RBAC testadas para todas as roles
- [ ] Documentação da API gerada para endpoints da fase
- [ ] Docker compose funcional para novos desenvolvedores

---

## Fase 2 — CMS

> **Duração:** Semanas 7 a 14 (Mês 2 a 3)  
> **Versão alvo:** v0.2.0

### Objetivo

Construir o núcleo do CMS: artigos, taxonomias, gerenciamento de assets, editor de conteúdo, revisões e lixeira. Ao final desta fase, o Nexora deve ser capaz de criar, editar, publicar e gerenciar conteúdo completo com suporte a mídia.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Artigos** | CRUD completo, status (draft, published, archived, scheduled), slug automático único por site, content em JSONB (blocks), excerpt, autor, published_at | 🔴 Crítica |
| **Categorias** | Taxonomia hierárquica com profundidade ilimitada, slug, descrição, parent_id, CRUD, árvore | 🟡 Alta |
| **Tags** | Taxonomia plana, slug, CRUD, contagem de posts | 🟢 Média |
| **Assets** | Upload de imagens, vídeos, PDFs e documentos; storage local/S3; validação de tipo e tamanho; thumbnails automáticos; lazy loading | 🔴 Crítica |
| **Editor** | Editor de conteúdo baseado em blocos (rich text, imagens, embed, código, tabelas), preview ao vivo, autosave | 🔴 Crítica |
| **Revisões** | Histórico de versões do artigo, diff entre versões, restauração de versão anterior | 🟡 Alta |
| **Lixeira** | Soft delete com lixeira por módulo, recuperação em 30 dias, expurgo automático | 🟢 Média |

### Dependências

- Fase 1 completa (kernel, auth, multi-site, permissões)
- Editor: biblioteca de blocos (TipTap/ProseMirror)
- Assets: lib de processamento de imagens (Go: `imaging` ou similar)

### Sub-fases

#### Sprint 2.1 — Artigos + Taxonomias (Semanas 7-9)
- CRUD de posts + soft delete
- CRUD de categorias (hierárquico) + tags
- Slug automático com verificação de unicidade por site
- Post_meta (JSONB) para metadados flexíveis
- Endpoints de listagem com filtros, paginação cursor/offset, sort

**Critérios de conclusão:**
- `POST /posts` cria artigo com slug único
- `GET /posts` retorna paginação com filtros combinados
- `PATCH /posts/:id/status` altera status corretamente
- GET /categories/tree retorna hierarquia completa
- Soft delete funcional (artigo vai para lixeira)

#### Sprint 2.2 — Assets + Editor (Semanas 10-12)
- Upload de assets com validação (magic bytes, extensão, tamanho)
- Storage abstraction: local FS e S3 (interface `Storer`)
- Geração automática de thumbnails para imagens
- Galeria de assets no admin com grid, busca, filtros
- Editor de blocos (TipTap): rich text, imagens, embed, código
- Autosave a cada 30s com versão temporária

**Critérios de conclusão:**
- Upload de imagem JPG/PNG/WebP, PDF, MP4 validado
- Thumbnail gerado automaticamente no upload
- Editor salva conteúdo em JSONB com estrutura de blocos
- Autosave recupera versão após fechamento acidental
- Assets vinculáveis a posts via `post_assets`

#### Sprint 2.3 — Revisões + Lixeira (Semanas 13-14)
- Histórico de versões: criar versão a cada save explícito
- Diff entre versões (estrutural, campo a campo)
- Restauração: copia versão anterior como atual
- Lixeira: listar itens deletados, restaurar, expurgar
- Worker de expurgo automático (30 dias)

**Critérios de conclusão:**
- `GET /posts/:id/versions` lista histórico
- Restauração de versão preserva integridade dos dados
- Item na lixeira é recuperável
- Worker expurga itens com mais de 30 dias

### Critérios para considerar a Fase 2 concluída

- [ ] CRUD de artigos completo com todos os status
- [ ] Taxonomias funcionais com hierarquia e filtros
- [ ] Upload de assets validado por tipo, tamanho e magic bytes
- [ ] Storage abstraction com implementações local e S3
- [ ] Editor de blocos operacional com autosave
- [ ] Histórico de versões com diff e restauração
- [ ] Lixeira funcional com expurgo automático
- [ ] Admin UI listando posts com grid/kanban
- [ ] Cobertura de testes > 65%

---

## Fase 3 — SEO Core

> **Duração:** Semanas 15 a 20 (Mês 3.5 a 4.5)  
> **Versão alvo:** v0.3.0

### Objetivo

Implementar o módulo de SEO como parte do Core gratuito, fornecendo todas as ferramentas essenciais de otimização para mecanismos de busca: sitemap, meta tags, schema.org, breadcrumbs, redirects e RSS. Nenhuma dependência de serviços pagos.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Sitemap** | Sitemap.xml dinâmico por site, prioridade/configurável, split em múltiplos sitemaps para grandes volumes, notificação ao Google via ping | 🔴 Crítica |
| **Robots** | Robots.txt gerenciável por site, regras por user-agent, allow/disallow por path, sitemap reference | 🔴 Crítica |
| **Meta Tags** | Geração automática de title tag, meta description, keywords; customizável por post; template por site | 🔴 Crítica |
| **Open Graph** | OG:title, OG:description, OG:image, OG:type, OG:url automáticos; fallback para asset destacado | 🟡 Alta |
| **Twitter Cards** | Twitter:card, twitter:site, twitter:title, twitter:description, twitter:image | 🟢 Média |
| **Canonical** | URL canônica automática por post, customizável, prevenção de conteúdo duplicado | 🟡 Alta |
| **Breadcrumbs** | Breadcrumb schema + visual, baseado na hierarquia de categorias, JSON-LD integrado | 🟢 Média |
| **Schema.org** | Geração automática de JSON-LD: Article, BlogPosting, WebPage, BreadcrumbList, SiteNavigationElement, Organization, Person | 🟡 Alta |
| **URLs Amigáveis** | Slug automático otimizado para SEO, redirect automático de slugs antigos, lowercase + hífens | 🔴 Crítica |
| **Redirects** | Gerenciamento de redirects 301/302, regras por regex, wildcard, import/export CSV | 🟡 Alta |
| **RSS** | Feed RSS 2.0 e Atom por site, categorias, configurável (quantidade, include excerpts) | 🟢 Média |

### Dependências

- Fase 2 completa (artigos, categorias, assets)
- Nenhuma dependência de API externa

### Sub-fases

#### Sprint 3.1 — Meta Tags + Open Graph + Twitter Cards (Semanas 15-16)
- Sistema de templates de meta tags por site
- Geração automática no `AfterPostPublish`
- Preview de SEO no editor (como fica no Google, Facebook, Twitter)
- Configuração global com override por post

**Critérios de conclusão:**
- Todo post publicado tem title tag e meta description
- OG tags geram preview válido no Facebook Sharing Debugger
- Twitter Cards validados no card validator
- Preview SEO no editor funcional

#### Sprint 3.2 — Sitemap + Robots + Canonical + RSS (Semanas 17-18)
- Sitemap.xml dinâmico com splits automáticos (> 50k URLs)
- Indexação no Google via ping
- Robots.txt editável por site
- Canonical URL automática
- Feed RSS 2.0 e Atom

**Critérios de conclusão:**
- `GET /sitemap.xml` retorna XML válido (W3C)
- `GET /robots.txt` reflete configuração do site
- `GET /feed.xml` retorna RSS válido
- Ping ao Google Search Console funcional
- Canonical URL presente em todo post

#### Sprint 3.3 — Schema.org + Breadcrumbs + Redirects (Semanas 19-20)
- JSON-LD automático: Article, BreadcrumbList, Organization
- Breadcrumbs baseados na árvore de categorias
- Gerenciamento de redirects: CRUD, validação de loop, regex
- Import/export de redirects em CSV

**Critérios de conclusão:**
- Página de post contém JSON-LD Article válido (testável no Schema.org Validator)
- Breadcrumbs refletem hierarquia real
- Redirect 301 funcional com cache
- Validação de loop em redirects (impede redirect infinito)
- Export CSV de redirects funcional

### Critérios para considerar a Fase 3 concluída

- [ ] Sitemap dinâmico + split + ping ao Google
- [ ] Robots.txt customizável por site
- [ ] Meta tags + OG + Twitter Cards automáticos
- [ ] URL canônica em todo conteúdo
- [ ] Schema.org JSON-LD Article + BreadcrumbList + Organization
- [ ] Breadcrumbs visuais + marcados
- [ ] Sistema de redirects 301/302 com validação
- [ ] RSS 2.0 + Atom
- [ ] Preview SEO no editor
- [ ] Testes de integração com validadores oficiais (W3C, Schema.org)
- [ ] Documentação do módulo SEO

---

## Fase 4 — IA Base

> **Duração:** Semanas 21 a 28 (Mês 5 a 6.5)  
> **Versão alvo:** v0.4.0

### Objetivo

Construir o módulo de IA Base como parte do Core gratuito, utilizando modelos locais e open-source. O usuário pode gerar, reescrever, traduzir e otimizar conteúdo sem depender de APIs pagas como OpenAI. Modelos rodam em Python com comunicação via gRPC com o backend Go.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Gerador de Artigos** | Geração de artigos completos a partir de título/tópico usando modelo local (LLaMA, Mistral, etc.), controle de tom e tamanho | 🔴 Crítica |
| **Reescrita** | Reescrita de parágrafos/artigos com diferentes estilos (formal, informal, persuasivo), manter significado original | 🔴 Crítica |
| **Tradução PT/EN** | Tradução automática entre português e inglês mantendo formatação, links e SEO | 🟡 Alta |
| **Meta Description** | Geração automática de meta description a partir do conteúdo, otimizada para CTR | 🟡 Alta |
| **FAQs** | Extração automática de perguntas frequentes do conteúdo, geração de FAQ Schema | 🟢 Média |
| **Sugestões de Títulos** | Geração de variações de título otimizadas para clique e SEO | 🟡 Alta |
| **Links Internos** | Sugestão automática de links internos baseada em similaridade semântica (embeddings + pgvector) | 🟢 Média |

### Dependências

- Fase 2 completa (artigos devem existir para serem gerados/reescritos)
- Fase 3 completa (meta description depende de SEO Core)
- Python 3.12+ para o serviço de IA
- PyTorch + Transformers + Sentence-Transformers
- Modelos OSS: LLaMA 3 / Mistral / BERT multilingual
- Comunicação Go ↔ Python via gRPC
- pgvector para busca semântica

### Sub-fases

#### Sprint 4.1 — Infraestrutura de IA (Semanas 21-22)
- Serviço Python separado com API gRPC
- Download e cache de modelos OSS
- Proxy no backend Go para comunicação
- Queue de inferência para não bloquear requests
- Fallback se modelo não estiver disponível

**Critérios de conclusão:**
- Serviço Python sobe e se registra no kernel
- Go → Python gRPC funcional com timeout
- Queue processa requests sem bloquear API
- Fallback graceful se modelo não carregar

#### Sprint 4.2 — Geração + Reescrita (Semanas 23-25)
- Gerador de artigos: título → outline → conteúdo completo
- Controles: tamanho (curto/médio/longo), tom (formal/informal), público-alvo
- Reescrita: selecionar parágrafo, escolher estilo, manter links e formatação
- Preview antes de aplicar

**Critérios de conclusão:**
- Gerador produz artigo coerente com > 300 palavras
- Reescrita mantém links internos e formatação original
- Preview funcional no editor
- Tempo de geração < 30s (com GPU) ou < 2min (CPU)

#### Sprint 4.3 — Tradução + Meta Description + FAQs + Títulos + Links (Semanas 26-28)
- Tradução PT ↔ EN preservando Markdown/HTML
- Geração de meta description otimizada para CTR
- Extração de FAQs do conteúdo com geração de FAQ Schema
- Sugestão de 5 variações de título com score
- Links internos: similaridade por embeddings pgvector

**Critérios de conclusão:**
- Tradução PT→EN e EN→PT com formatação preservada
- Meta description gerada tem entre 150-160 caracteres
- FAQs extraídas são relevantes e não duplicadas
- Sugestões de título têm diversidade semântica
- Links internos sugeridos são contextualmente relevantes

### Critérios para considerar a Fase 4 concluída

- [ ] Serviço Python de IA operacional com gRPC
- [ ] Geração de artigos funcional com modelo OSS local
- [ ] Reescrita mantém formatação e links
- [ ] Tradução PT/EN bidirecional
- [ ] Meta description automática otimizada
- [ ] Extração de FAQs com Schema
- [ ] Sugestões de título geradas no editor
- [ ] Links internos sugeridos via busca semântica
- [ ] Todos os fallbacks funcionam sem GPU
- [ ] Documentação do módulo IA Base

---

## Fase 5 — Dashboard Inteligente

> **Duração:** Semanas 29 a 34 (Mês 7 a 8)  
> **Versão alvo:** v0.5.0

### Objetivo

Criar o dashboard inteligente do Nexora, fornecendo ao usuário uma visão completa da saúde do site, estatísticas de conteúdo, alertas proativos, sugestões automáticas e métricas de performance. Integra dados do SEO Core e IA Base para recomendações acionáveis.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Estatísticas** | Contagem de posts, visualizações (via evento próprio), crescimento semanal, categorias mais populosas, engajamento por autor | 🔴 Crítica |
| **Saúde do Site** | Score de SEO geral, velocidade estimada, links quebrados detectados, cobertura de meta tags, posts sem imagem destacada | 🔴 Crítica |
| **Alertas** | Notificações no dashboard: posts com baixo score SEO, links quebrados, conteúdo desatualizado, erros 404 frequentes, queda de performance | 🟡 Alta |
| **Sugestões** | Recomendações automáticas: "Este post poderia ser atualizado", "Você ainda não tem FAQs", "Adicione imagem destacada a 3 posts", baseadas em IA | 🟡 Alta |
| **Receita** | Métricas de receita (integrada com plugins de afiliados/ads quando disponíveis), projeções, TOP conteúdos por receita | 🟢 Média |
| **Performance** | Tempo de carregamento estimado, Core Web Vitals, tamanho de página, número de requests, sugestões de otimização | 🟡 Alta |

### Dependências

- Fase 2 completa (posts, assets para estatísticas)
- Fase 3 completa (score SEO, meta tags, links)
- Fase 4 completa (IA para sugestões inteligentes)
- Tabela `analytics_events` particionada por data

### Sub-fases

#### Sprint 5.1 — Estatísticas + Performance (Semanas 29-30)
- Coleta de eventos de visualização (pageview anônimo)
- Agregação de estatísticas por site, período, categoria, autor
- Dashboard de performance: métricas do servidor, tempo de resposta
- Widgets do admin home

**Critérios de conclusão:**
- Dashboard mostra contagens corretas de posts por status
- Gráfico de crescimento semanal de conteúdo
- Métricas de performance do servidor (latência p50/p99)
- Eventos de visualização não impactam tempo de resposta

#### Sprint 5.2 — Saúde + Alertas (Semanas 31-32)
- Score de SEO por post e geral do site
- Detecção de links quebrados (worker assíncrono)
- Verificação de meta tags ausentes
- Sistema de alertas com níveis (info, warning, critical)

**Critérios de conclusão:**
- Score SEO calculado corretamente
- Links quebrados detectados e reportados
- Alertas aparecem no dashboard com prioridade
- Notificações sumiram após correção

#### Sprint 5.3 — Sugestões Inteligentes + Receita (Semanas 33-34)
- Sugestões baseadas em IA: o que melhorar, o que criar
- Pipeline de recomendação: análise → priorização → notificação
- Widget de receita (preparado para plugins, vazio por padrão)
- Export de relatórios em PDF/CSV

**Critérios de conclusão:**
- Sugestões são relevantes e acionáveis
- Usuário pode marcar sugestão como "feita" ou "ignorar"
- Widget de receita mostra placeholder quando sem plugin
- Export CSV funcional

### Critérios para considerar a Fase 5 concluída

- [ ] Dashboard home com widgets de estatísticas
- [ ] Score de SEO por post e geral
- [ ] Detecção de links quebrados automática
- [ ] Alertas com níveis de prioridade
- [ ] Sugestões inteligentes baseadas em IA
- [ ] Métricas de performance do servidor
- [ ] Export de relatórios
- [ ] Cobertura de testes > 60%

---

## Fase 6 — Plugins Premium

> **Duração:** Semanas 35 a 48 (Mês 8.5 a 11)  
> **Versão alvo:** v1.0.0 (Core completo) + v1.1.0 a v1.5.0

### Objetivo

Disponibilizar o Plugin System e os primeiros plugins premium. O Core permanece completo e funcional sem eles; os plugins adicionam recursos avançados que podem depender de APIs pagas. Esta fase também inclui o lançamento estável do Nexora (v1.0.0).

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Plugin System** | Plugin Manager (discovery, loading, lifecycle), sandbox WASM, hooks, manifest validation, marketplace API, licenciamento | 🔴 Crítica |
| **SEO Premium** | Análise de concorrentes, cluster de tópicos, auditoria técnica, integração Google Search Console, sugestão de palavras-chave | 🟡 Alta |
| **IA Premium** | GPT-4 para geração, Dall-E/Stable Diffusion para imagens, Cloud Vision para análise, tradução multi-idioma, voice-to-text | 🟡 Alta |
| **Analytics Avançado** | Dashboards customizáveis, funis de conversão, heatmaps, segmentação de audiência, integração GA4/Plausible, export avançado | 🟢 Média |
| **Keywords** | Pesquisa de palavras-chave (Semrush, Ahrefs, Google KW Planner), volume, dificuldade, tendência, clustering | 🟢 Média |
| **Afiliados** | Gerenciamento de links, cloaking, automação de ofertas, comparação de preços, relatórios de comissão | 🔵 Baixa |
| **Newsletter** | Email marketing, templates HTML, listas segmentadas, campanhas agendadas, analytics de abertura/clique, integração Mailgun/SendGrid | 🟢 Média |
| **Automações** | Workflows visuais: "se condição → então ação", integração com webhooks, agendamento avançado | 🟡 Alta |
| **Concorrentes** | Monitoramento de concorrentes, alerta de novos conteúdos, análise de gaps, comparação de estratégia | 🔵 Baixa |

### Dependências

- Fase 1 a 5 completas
- Plugin System depende do kernel (Fase 1)
- APIs pagas para AI Premium (OpenAI, Google Cloud), SEO Premium (Google Search Console)

### Sub-fases

#### Sprint 6.1 — Plugin System + Marketplace (Semanas 35-37)
- Plugin Manager: discovery de plugins instalados, loading WASM
- Sistema de hooks para plugins interceptarem fluxos do core
- Sandbox WASM com limites de memória/CPU
- Plugin manifest (plugin.json) com validação
- API do marketplace: listar, instalar, atualizar, remover
- Sistema de licenciamento (chave pública/privada)

**Critérios de conclusão:**
- Plugin WASM é carregado e executa hook `AfterPostSave`
- Plugin com erro não derruba o core
- Manifest inválido é rejeitado na instalação
- Licenciamento verifica chave offline
- API de marketplace retorna plugins disponíveis

#### Sprint 6.2 — SEO Premium + Keywords (Semanas 38-39)
- Integração Google Search Console (GSC)
- Análise de concorrentes (extração de SEO de URLs concorrentes)
- Cluster de tópicos: agrupar conteúdo por assunto
- Keyword Research: volume, dificuldade, CPC via APIs
- Auditoria técnica automática

**Critérios de conclusão:**
- GSC integrado: impressões, cliques, posição média
- Análise de concorrente retorna meta tags e estrutura
- Cluster de tópicos agrupa posts corretamente
- Keyword Research retorna dados de API paga

#### Sprint 6.3 — IA Premium + Analytics (Semanas 40-42)
- Integração OpenAI (GPT-4, Dall-E 3)
- Geração de imagens AI para posts
- Análise de imagens (Cloud Vision)
- Tradução multi-idioma (20+ idiomas)
- Analytics: dashboards, heatmaps, funis

**Critérios de conclusão:**
- Geração de artigo com GPT-4 funcional
- Geração de imagem com Dall-E integrada ao editor
- Dashboard de analytics com dados reais
- Heatmap funcional

#### Sprint 6.4 — Afiliados + Newsletter + Automações + Concorrentes (Semanas 43-48)
- Gerenciamento de links afiliados com cloaking
- Automação de ofertas (Amazon, Hotmart, etc.)
- Newsletter: criação, envio, tracking
- Workflows visuais de automação
- Monitoramento de concorrentes

**Critérios de conclusão:**
- Link afiliado com cloaking funcional
- Newsletter enviada com tracking de abertura
- Workflow de automação executa regras corretamente
- Concorrente monitorado: alerta de novo conteúdo

### Critérios para considerar a Fase 6 concluída

- [ ] Plugin System completo com sandbox WASM
- [ ] Marketplace API operacional
- [ ] 5+ plugins premium disponíveis e testados
- [ ] Sistema de licenciamento validado
- [ ] Documentação do desenvolvedor de plugins
- [ ] Todos os plugins premium têm testes de integração
- [ ] Site de marketing com preview dos plugins

---

## Fase 7 — Escalabilidade

> **Duração:** Semanas 49 a 56 (Mês 11.5 a 13)  
> **Versão alvo:** v2.0.0

### Objetivo

Preparar o Nexora para produção em larga escala: cache em múltiplas camadas, CDN, API pública, webhooks, backup automático, monitoramento e suporte a cluster. Esta fase transforma o Nexora em uma plataforma enterprise-ready.

### Módulos

| Módulo | Descrição | Prioridade |
|--------|-----------|------------|
| **Cache** | Cache multi-camada (L1 in-memory, L2 Redis, CDN), invalidação seletiva por evento, cache warming, stale-while-revalidate | 🔴 Crítica |
| **CDN** | Integração com Cloudflare/Fastly, purging automático por evento, suporte a edge caching | 🟡 Alta |
| **API Pública** | Rate limiting por API key, documentação OpenAPI/Swagger, versões, deprecação gradual, playground interativo | 🔴 Crítica |
| **Webhooks** | Envio de webhooks para URLs externas, retry com backoff, assinatura HMAC, log de entregas, dashboard de status | 🟡 Alta |
| **Backup** | Backup automático do banco (pg_dump), assets (S3 sync), retention policy, restore via CLI | 🔴 Crítica |
| **Monitoramento** | Métricas (Prometheus), logs estruturados (OpenTelemetry), dashboards (Grafana), alertas (PagerDuty), tracing distribuído | 🟡 Alta |
| **Cluster** | Load balancer, múltiplos nodes Go stateless, conexões DB via PgBouncer, sessões Redis centralizado, deploy rolling update | 🟢 Média |

### Dependências

- Fase 1 a 6 completas
- Redis para cache distribuído
- Prometheus + Grafana para monitoramento
- OpenTelemetry para tracing

### Sub-fases

#### Sprint 7.1 — Cache + CDN (Semanas 49-50)
- Implementação de cache multi-camada com invalidação por evento
- Cache warming para posts populares
- Integração CDN: configuração, purging automático
- Suporte a `stale-while-revalidate`

**Critérios de conclusão:**
- Cache L1 + L2 funcionais com invalidação correta
- Cache warming populando posts mais acessados
- PURGE via CDN API funcional
- Stale-while-revalidate serve conteúdo mesmo com cache expirado

#### Sprint 7.2 — API Pública + Webhooks (Semanas 51-52)
- Documentação OpenAPI 3.0 completa
- API Keys com rate limiting por key
- Sistema de webhooks: registrar, disparar, retry
- Assinatura HMAC para verificação de webhooks

**Critérios de conclusão:**
- Swagger UI funcional em `/api/docs`
- API Key limita requests conforme configurado
- Webhook entregue com assinatura válida
- Retry exponencial após falha

#### Sprint 7.3 — Backup + Monitoramento (Semanas 53-54)
- Backup automático agendado do PostgreSQL
- Backup de assets para storage externo
- Restore via CLI com verificação de integridade
- Métricas exportadas para Prometheus
- Logs estruturados com OpenTelemetry
- Dashboard Grafana pré-configurado

**Critérios de conclusão:**
- Backup automático roda no schedule configurado
- Restore funcional com dados íntegros
- Prometheus coleta métricas do Go runtime + HTTP
- Grafana dashboard mostra métricas chave
- Logs têm trace_id para correlação

#### Sprint 7.4 — Cluster (Semanas 55-56)
- Suporte a múltiplos nodes Go
- PgBouncer para pool de conexões
- Sessões centralizadas no Redis
- Deploy rolling update com health check
- Testes de carga com k6

**Critérios de conclusão:**
- 3 nodes rodando atrás de load balancer
- Rolling update sem downtime
- Teste de carga: 10k requests/s sustentados
- PgBouncer gerencia pool eficientemente
- Sessões funcionam cross-node

### Critérios para considerar a Fase 7 concluída

- [ ] Cache multi-camada com invalidação correta
- [ ] CDN integrada com purging automático
- [ ] API pública documentada com OpenAPI
- [ ] Webhooks com retry e HMAC
- [ ] Backup automático com restore testado
- [ ] Monitoramento Prometheus + Grafana
- [ ] Cluster funcional com rolling update
- [ ] Teste de carga aprovado (10k req/s)
- [ ] Documentação de administração do sistema
- [ ] Runbook de produção

---

## Cronograma Consolidado

```
Mês 1    2    3    4    5    6    7    8    9    10   11   12   13
├────────┤────┤────┤────┤────┤────┤────┤────┤────┤────┤────┤────┤
████ Fase 1 — Fundação ████
     ██████ Fase 2 — CMS ██████
               ██████ Fase 3 — SEO Core ██████
                         ████████ Fase 4 — IA Base ████████
                                      ██████ Fase 5 — Dashboard ████
                                                ██████████████ Fase 6 — Plugins ████████████
                                                                     ████████ Fase 7 — Escala ████████
```

### Marcos Principais

| Marco | Data | Versão | Entregável |
|-------|------|--------|------------|
| M1 | Fim do Mês 1.5 | v0.1.0 | Fundação: kernel, auth, multi-site |
| M2 | Fim do Mês 3 | v0.2.0 | CMS completo: posts, assets, editor |
| M3 | Fim do Mês 4.5 | v0.3.0 | SEO Core: sitemap, meta tags, schema |
| M4 | Fim do Mês 6.5 | v0.4.0 | IA Base: geração, reescrita, tradução |
| M5 | Fim do Mês 8 | v0.5.0 | Dashboard inteligente |
| M6 | Fim do Mês 11 | v1.0.0 | Release estável do Core |
| M7 | Fim do Mês 13 | v2.0.0 | Escalabilidade enterprise |

### Estimativa de Equipe

| Fase | Backend (Go) | Frontend (React) | AI/ML (Python) | DevOps | Total |
|------|-------------|------------------|-----------------|--------|-------|
| Fase 1 | 2 | 1 | 0 | 1 | 4 |
| Fase 2 | 2 | 2 | 0 | 0 | 4 |
| Fase 3 | 1 | 1 | 0 | 0 | 2 |
| Fase 4 | 1 | 1 | 2 | 0 | 4 |
| Fase 5 | 1 | 1 | 1 | 0 | 3 |
| Fase 6 | 2 | 2 | 1 | 0 | 5 |
| Fase 7 | 1 | 0 | 0 | 2 | 3 |

---

## Decisões Arquiteturais

| Decisão | Impacto |
|---------|---------|
| **SEO Base no Core** | Gratuito para todos, sem dependência de plugins. Sitemap, meta tags, schema.org são funcionalidade base. |
| **IA Base no Core** | Usa modelos locais/OSS (LLaMA, Mistral). Sem custo de API. GPU é opcional (fallback CPU). |
| **Content Intelligence integrado à IA Base** | Classificação, recomendação e score são entregues junto com IA Base. |
| **Automation como módulo do Core** | Regras de automação e webhooks de saída são nativos, não exigem plugin. |
| **Media renomeado para Assets** | Nome mais abrangente: imagens, vídeos, documentos, áudio. |
| **Plugins = Premium Extras** | Core é completo sem plugins. Premium só adiciona recursos avançados. |
| **Modular Monolith primeiro** | Um binário Go para toda API. Microsserviços extraídos apenas quando necessário. |
| **Event Bus + Hooks** | Módulos do core e plugins se comunicam por eventos tipados e hooks. |
| **Python para IA** | Go faz proxy gRPC para serviço Python. Permite usar ecossistema PyTorch sem poluir o core. |
| **pgvector para busca semântica** | Embeddings armazenados no próprio PostgreSQL. Sem dependência externa de busca vetorial. |

---

## Glossário

| Termo | Definição |
|-------|-----------|
| **Core** | Conjunto de módulos essenciais distribuídos gratuitamente com o Nexora |
| **Kernel** | Núcleo do sistema: lifecycle, event bus, module registry, plugin manager |
| **Módulo** | Unidade funcional independente com handler, service, repository |
| **Plugin** | Extensão instalável via WASM, adiciona hooks ao core |
| **RLS** | Row Level Security — isolamento multi-site no PostgreSQL |
| **ISR** | Incremental Static Regeneration — Next.js atualiza páginas estáticas sem rebuild total |
| **WASM** | WebAssembly — runtime sandbox para plugins seguros |
| **sqlc** | Codegen de SQL puro para Go types — zero ORM |
| **pgvector** | Extensão PostgreSQL para busca por similaridade de vetores (embeddings) |
| **BFF** | Backend For Frontend — API Gateway específico para o admin SPA |

---

> **Arquivo gerado em:** Julho 2026  
> **Projeto:** Nexora CMS — A próxima geração de gerenciamento de conteúdo com IA  
> **Documento de referência:** `nexora-architecture.md`
