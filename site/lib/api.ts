export interface ArticleSource {
  url: string;
  title?: string;
  snippet?: string;
  published_at?: string;
  retrieved_at: string;
  freshness_score?: number;
  is_verified: boolean;
}

export interface Article {
  id: string;
  site_id: string;
  title: string;
  slug: string;
  excerpt?: string;
  content?: string;
  featured_image_url?: string;
  author_id?: string;
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

export interface ArticleListResponse {
  articles: Article[];
  total: number;
}

export interface Category {
  id: string;
  site_id: string;
  parent_id?: string;
  name: string;
  slug: string;
  description?: string;
  icon?: string;
  color?: string;
  sort_order: number;
  children?: Category[];
}

export interface CategoryListResponse {
  categories: Category[];
  total: number;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function fetchAPI<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const url = `${API_BASE}${path}`;
  const res = await fetch(url, {
    ...init,
    next: { revalidate: 60 },
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!res.ok) {
    let code = "UNKNOWN";
    try {
      const body = await res.json();
      code = body?.error?.code || code;
    } catch {}
    throw new ApiError(res.status, code, `API request failed: ${res.statusText}`);
  }

  return res.json();
}

export async function getArticles(
  options?: { limit?: number; offset?: number; language?: string },
): Promise<ArticleListResponse> {
  const params = new URLSearchParams();
  if (options?.limit) params.set("limit", String(options.limit));
  if (options?.offset) params.set("offset", String(options.offset));
  if (options?.language) params.set("language", options.language);
  const qs = params.toString();
  return fetchAPI<ArticleListResponse>(`/api/v1/articles${qs ? `?${qs}` : ""}`);
}

export async function getArticleBySlug(slug: string): Promise<Article | null> {
  try {
    return await fetchAPI<Article>(`/api/v1/articles/${slug}`);
  } catch {
    return null;
  }
}

export async function getCategories(): Promise<CategoryListResponse> {
  return fetchAPI<CategoryListResponse>("/api/v1/categories");
}

export function formatDate(dateStr?: string): string {
  if (!dateStr) return "";
  return new Date(dateStr).toLocaleDateString("pt-BR", {
    day: "numeric",
    month: "long",
    year: "numeric",
  });
}

export function readingTimeLabel(minutes: number): string {
  if (minutes <= 1) return "1 min de leitura";
  return `${minutes} min de leitura`;
}
