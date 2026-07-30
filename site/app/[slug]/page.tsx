import type { Metadata } from "next";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface ArticleSource {
  url: string;
  title?: string;
  snippet?: string;
  published_at?: string;
  retrieved_at: string;
  freshness_score?: number;
  is_verified: boolean;
}

interface ArticleResponse {
  id: string;
  title: string;
  slug: string;
  excerpt?: string;
  content?: string;
  featured_image_url?: string;
  published_at?: string;
  meta_title?: string;
  meta_description?: string;
  og_image?: string;
  canonical_url?: string;
  language: string;
  tags?: string[];
  categories?: string[];
  word_count: number;
  reading_time: number;
  freshness_score?: number;
  sources?: ArticleSource[];
}

interface Props {
  params: Promise<{ slug: string }>;
}

async function getArticle(slug: string): Promise<ArticleResponse | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/articles/${slug}`, {
      next: { revalidate: 60 },
    });
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const article = await getArticle(slug);
  if (!article) return { title: "Not Found" };

  return {
    title: article.meta_title || article.title,
    description: article.meta_description || article.excerpt,
    openGraph: {
      title: article.meta_title || article.title,
      description: article.meta_description || article.excerpt,
      images: article.og_image ? [{ url: article.og_image }] : undefined,
    },
    alternates: {
      canonical: article.canonical_url || undefined,
    },
  };
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return "";
  return new Date(dateStr).toLocaleDateString("pt-BR", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

function readingTimeLabel(minutes: number): string {
  if (minutes <= 1) return "1 min de leitura";
  return `${minutes} min de leitura`;
}

export default async function ArticlePage({ params }: Props) {
  const { slug } = await params;
  const article = await getArticle(slug);

  if (!article) {
    return (
      <main className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">
            Artigo não encontrado
          </h1>
          <p className="text-gray-600">
            O artigo que você procura não existe ou foi removido.
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-white">
      <article className="max-w-3xl mx-auto px-4 py-12">
        {article.categories && article.categories.length > 0 && (
          <div className="flex gap-2 mb-6">
            {article.categories.map((cat) => (
              <span
                key={cat}
                className="text-sm text-blue-600 bg-blue-50 px-3 py-1 rounded-full"
              >
                {cat}
              </span>
            ))}
          </div>
        )}

        <h1 className="text-4xl font-bold text-gray-900 mb-4 leading-tight">
          {article.title}
        </h1>

        <div className="flex items-center gap-4 text-sm text-gray-500 mb-8">
          {article.published_at && (
            <time dateTime={article.published_at}>
              {formatDate(article.published_at)}
            </time>
          )}
          <span>{readingTimeLabel(article.reading_time)}</span>
          {article.word_count > 0 && (
            <span>{article.word_count.toLocaleString()} palavras</span>
          )}
        </div>

        {article.featured_image_url && (
          <div className="mb-8 rounded-lg overflow-hidden">
            <img
              src={article.featured_image_url}
              alt={article.title}
              className="w-full h-auto object-cover"
            />
          </div>
        )}

        {article.excerpt && (
          <p className="text-xl text-gray-600 mb-8 leading-relaxed">
            {article.excerpt}
          </p>
        )}

        {article.content && (
          <div
            className="prose prose-lg max-w-none"
            dangerouslySetInnerHTML={{ __html: article.content }}
          />
        )}

        {article.tags && article.tags.length > 0 && (
          <div className="mt-12 pt-8 border-t border-gray-200">
            <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
              Tags
            </h2>
            <div className="flex flex-wrap gap-2">
              {article.tags.map((tag) => (
                <span
                  key={tag}
                  className="text-sm bg-gray-100 text-gray-700 px-3 py-1 rounded-full"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}

        {article.sources && article.sources.length > 0 && (
          <div className="mt-12 pt-8 border-t border-gray-200">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              Fontes
            </h2>
            <ul className="space-y-4">
              {article.sources.map((src, i) => (
                <li key={i} className="flex items-start gap-3">
                  <span className="text-blue-500 mt-1 shrink-0">
                    {src.is_verified ? "✓" : "○"}
                  </span>
                  <div>
                    <a
                      href={src.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline font-medium"
                    >
                      {src.title || src.url}
                    </a>
                    {src.snippet && (
                      <p className="text-sm text-gray-600 mt-1">{src.snippet}</p>
                    )}
                    {src.freshness_score !== undefined && (
                      <p className="text-xs text-gray-400 mt-1">
                        Frescor: {(src.freshness_score * 100).toFixed(0)}%
                      </p>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}
      </article>
    </main>
  );
}
