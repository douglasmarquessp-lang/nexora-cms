# Sprint 3.11 — Homepage Real

## Resumo

Transformacao do frontend publico de um stub "Site em construcao" em uma homepage editorial funcional, preparada para producao, com conteudo real da API, SEO, acessibilidade e suporte multi-site.

---

## Estrutura Anterior do Frontend

O diretorio `site/` continha apenas:
- `app/page.tsx` — placeholder com "Site em construcao" (15 linhas)
- `app/layout.tsx` — metadata basico, sem OG tags ou template
- `app/globals.css` — apenas 3 diretivas Tailwind
- `app/[slug]/page.tsx` — pagina de artigo individual com tipos inline
- `next.config.mjs` — proxy `/api/*` → `localhost:8080`, imagens remotePatterns

Nao havia:
- Componentes reutilizaveis
- API client compartilhado
- Tratamento de erros
- Estados vazios ou fallback
- Suporte a multi-tenancy no frontend

---

## Endpoints Utilizados

### Endpoints novos (criados neste sprint)

| Metodo | Path | Handler | Descricao |
|--------|------|---------|-----------|
| GET | `/api/v1/articles` | `publicArticleHandler.List` | Lista artigos publicados (paginacao) |
| GET | `/api/v1/categories` | `publicCategoriesHandler.List` | Lista categorias do site |

### Endpoints existentes reutilizados

| Metodo | Path | Handler | Uso |
|--------|------|---------|-----|
| GET | `/api/v1/articles/{slug}` | `publicArticleHandler.GetBySlug` | Pagina individual de artigo |
| GET | `/api/v1/health` | `healthHandler.Check` | (nao usado no frontend) |

---

## Arquivos Modificados

### Backend (Go)

| Arquivo | Alteracao |
|---------|-----------|
| `internal/api/articles.go` | Adicionado `PublicArticleListResponse`, metodo `List`, helper `toPublicArticleResponse` (extraido de GetBySlug), `publicCategoriesHandler`, metodo `List` para categorias |
| `internal/api/routes.go` | Registro de `GET /articles` e `GET /categories` no grupo publico (middleware `siteIdentify` apenas) |

### Frontend (Next.js/TypeScript)

| Arquivo | Descricao |
|---------|-----------|
| `site/lib/api.ts` | **Novo** — API client com tipos TypeScript, helpers, tratamento de erros |
| `site/components/Header.tsx` | **Novo** — Header sticky com nav, busca, menu mobile |
| `site/components/Hero.tsx` | **Novo** — Secao de artigo em destaque com gradiente e imagem |
| `site/components/ArticleCard.tsx` | **Novo** — Card de artigo reutilizavel |
| `site/components/ArticleList.tsx` | **Novo** — Grid de artigos com estado vazio |
| `site/components/CategoriesSection.tsx` | **Novo** — Secao de categorias com icones |
| `site/components/Sidebar.tsx` | **Novo** — Sidebar com artigos recentes e categorias |
| `site/components/Footer.tsx` | **Novo** — Footer profissional |
| `site/app/page.tsx` | **Substituido** — Homepage completa com todas as secoes |
| `site/app/layout.tsx` | **Atualizado** — Metadata com template, OG, Twitter, robots |
| `site/app/globals.css` | **Atualizado** — line-clamp, focus-visible, smooth scroll |
| `site/app/[slug]/page.tsx` | **Atualizado** — Importa tipos/helpers de `lib/api.ts` |
| `site/.env.local.example` | **Novo** — Template de variaveis de ambiente |
| `site/__tests__/api.test.ts` | **Novo** — Testes de tipos, formatDate, readingTimeLabel |

---

## Integracao Frontend/API

### API Client (`site/lib/api.ts`)
- `fetchAPI<T>(path, init)` — funcao generica com tratamento de erro
- `getArticles({ limit, offset, language })` — GET `/api/v1/articles`
- `getArticleBySlug(slug)` — GET `/api/v1/articles/{slug}`
- `getCategories()` — GET `/api/v1/categories`
- `formatDate(dateStr)` — formatacao pt-BR
- `readingTimeLabel(minutes)` — label de leitura

### Tratamento de Erros
- Classe `ApiError` com `status`, `code`, `message`
- `page.tsx` usa `try/catch` com `Promise.all` para fallback gracioso
- Se API indisponivel, mostra "Servico temporariamente indisponivel" com Header/Footer

### Cache/Revalidation
- `next: { revalidate: 60 }` — ISR com 60s de revalidacao
- Cache no servidor Next.js, nao no cliente

---

## Estrategia de Multi-tenancy

- O middleware `IdentifySite` resolve o site por:
  1. Header `X-Site-ID` (prioridade)
  2. Header `Host` (fallback, com stripping de porta)
  3. Fallback para `uuid.Nil` (graceful degradation)
- O Next.js proxy (`next.config.mjs`) preserva o Host header ao fazer rewrite para o backend
- Em desenvolvimento sem dominio real, e necessario usar `X-Site-ID` header
- O API client atual nao envia `X-Site-ID` automaticamente — depende do proxy/Host header

---

## SEO Implementado

- Metadata template: `"%s | Nexora CMS"`
- Open Graph tags (title, description, type, locale, siteName)
- Twitter cards (summary_large_image)
- Robots: index, follow
- Canonical URL por artigo (via API)
- Meta title/description por artigo (via API)
- HTML lang="pt-BR"
- Headings hierarquicos (h1, h2, h3)

### Nao implementado (deferido)
- Sitemap.xml
- Robots.txt
- Paginas de categoria (`/categoria/[slug]`)
- Breadcrumbs estruturados
- JSON-LD schema

---

## Acessibilidade

- HTML semantico (`<header>`, `<nav>`, `<main>`, `<article>`, `<footer>`, `<aside>`, `<section>`)
- `aria-label` em elementos interativos
- `aria-expanded` em botoes de toggle
- `role="search"` no formulario de busca
- `role="contentinfo"` no footer
- Labels em inputs (`<label htmlFor>` + `sr-only`)
- `sr-only` para elementos visuais decorativos
- `focus-visible` outlines configurados
- Contraste: texto branco em fundo escuro (hero), texto cinza em fundo claro
- Navegacao por teclado: todos os elementos interativos sao `<a>` ou `<button>`
- Menu mobile com estados acessiveis de abertura/fechamento
- `line-clamp` para evitar overflow de texto
- Imagens com `alt` text descritivo
- SVGs decorativos com `aria-hidden="true"`

---

## Testes Adicionados

### `site/__tests__/api.test.ts`
1. `Article` type validation — campos obrigatorios
2. `Article` optional fields — campos opcionais como undefined
3. `Category` type validation — estrutura basica
4. `formatDate` — data valida em portugues
5. `formatDate` — undefined retorna vazio
6. `formatDate` — string vazia retorna vazio
7. `readingTimeLabel` — singular para 1 minuto
8. `readingTimeLabel` — plural para varios minutos
9. `readingTimeLabel` — zero minutos

### Cobertura atual
- Go toolchain indisponivel — `go test`, `go build`, `go vet` nao puderam ser executados
- Testes TypeScript validam tipos, formatacao de data, e labels de leitura
- Estrutura do codigo validada por leitura de todos os arquivos `.ts/.tsx`

---

## Limitacoes Restantes

1. **Nome do autor**: `PublicArticleResponse` so tem `author_id` — sem join com tabela de usuarios. Frontend omite autor por enquanto.
2. **Busca**: Formulario de busca submete para `/busca?q=` — nao ha pagina de resultados.
3. **Paginas de categoria**: Links para `/categoria/[slug]` nao possuem pagina correspondente.
4. **Sitemap/Robots**: Deferido para sprint de SEO tecnico.
5. **X-Site-ID**: Frontend nao envia automaticamente. Em desenvolvimento, depende do proxy/Host header.
6. **Paginacao**: Homepage carrega 9 artigos sem "load more". Link "Ver todos" nao tem destino.
7. **Newsletter**: Nao ha suporte no backend — nao implementado no frontend.
8. **Imagens**: Usa `<img>` nativo (nao `next/image`) para simplicidade — otimizacao pode ser adicionada.
9. **Estado de loading**: SSR significa que usuario ve estado completo ou fallback — sem skeleton loading.
10. **Go toolchain**: Nao foi possivel validar backend com `go build`/`go test`.

---

## Comandos de Validacao (nao executados)

```bash
# Go toolchain indisponivel neste ambiente
go build ./...
go vet ./...
go test ./...

# Frontend (requer npm install + next build)
cd site && npm run build
cd site && npm run lint
```
