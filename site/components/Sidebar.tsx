import Link from "next/link";
import type { Article, Category } from "@/site/lib/api";
import { formatDate, readingTimeLabel } from "@/site/lib/api";

interface SidebarProps {
  recentArticles: Article[];
  categories: Category[];
}

export default function Sidebar({ recentArticles, categories }: SidebarProps) {
  return (
    <aside className="space-y-8" aria-label="Barra lateral">
      {recentArticles.length > 0 && (
        <div>
          <h3 className="text-lg font-bold text-gray-900 mb-4">Recentes</h3>
          <ul className="space-y-4">
            {recentArticles.map((article) => (
              <li key={article.id}>
                <Link href={`/${article.slug}`} className="group flex gap-3">
                  <div className="shrink-0 w-16 h-16 rounded-lg overflow-hidden bg-gray-100">
                    {article.featured_image_url ? (
                      <img
                        src={article.featured_image_url}
                        alt={article.title}
                        className="h-full w-full object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" className="text-gray-400" aria-hidden="true">
                          <rect width="18" height="18" x="3" y="3" rx="2" ry="2" /><circle cx="9" cy="9" r="2" /><path d="m21 15-3.1-3.1a2 2 0 0 0-2.8 0L6 21" />
                        </svg>
                      </div>
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <h4 className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors line-clamp-2">
                      {article.title}
                    </h4>
                    <p className="mt-1 text-xs text-gray-400">
                      {article.published_at ? formatDate(article.published_at) : readingTimeLabel(article.reading_time)}
                    </p>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}

      {categories.length > 0 && (
        <div>
          <h3 className="text-lg font-bold text-gray-900 mb-4">Categorias</h3>
          <ul className="space-y-2">
            {categories.map((cat) => (
              <li key={cat.id}>
                <Link
                  href={`/categoria/${cat.slug}`}
                  className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 hover:text-blue-600 transition-colors"
                >
                  <span>{cat.name}</span>
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-300" aria-hidden="true">
                    <path d="M9 18l6-6-6-6" />
                  </svg>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </aside>
  );
}
