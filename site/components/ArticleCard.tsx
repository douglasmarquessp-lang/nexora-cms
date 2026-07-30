import Link from "next/link";
import type { Article } from "@/site/lib/api";
import { formatDate, readingTimeLabel } from "@/site/lib/api";

interface ArticleCardProps {
  article: Article;
  featured?: boolean;
}

export default function ArticleCard({ article, featured }: ArticleCardProps) {
  return (
    <article className={`group relative flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white transition-all hover:shadow-lg hover:border-gray-300 ${featured ? "sm:flex-row" : ""}`}>
      <Link
        href={`/${article.slug}`}
        className={`relative block overflow-hidden ${featured ? "sm:w-2/5 sm:shrink-0" : "aspect-[16/9]"}`}
        aria-label={article.title}
      >
        {article.featured_image_url ? (
          <img
            src={article.featured_image_url}
            alt={article.title}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200">
            <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" className="text-gray-400" aria-hidden="true">
              <rect width="18" height="18" x="3" y="3" rx="2" ry="2" /><circle cx="9" cy="9" r="2" /><path d="m21 15-3.1-3.1a2 2 0 0 0-2.8 0L6 21" />
            </svg>
          </div>
        )}
      </Link>

      <div className="flex flex-1 flex-col justify-between p-5">
        <div>
          {article.categories && article.categories.length > 0 && (
            <span className="mb-2 inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
              {article.categories[0]}
            </span>
          )}

          <h2 className={`mt-1 font-bold leading-tight text-gray-900 ${featured ? "text-xl sm:text-2xl" : "text-lg"}`}>
            <Link href={`/${article.slug}`} className="hover:text-blue-600 transition-colors">
              {article.title}
            </Link>
          </h2>

          {article.excerpt && (
            <p className="mt-2 text-sm leading-relaxed text-gray-500 line-clamp-2">
              {article.excerpt}
            </p>
          )}
        </div>

        <div className="mt-4 flex items-center gap-3 text-xs text-gray-400">
          {article.published_at && (
            <time dateTime={article.published_at}>{formatDate(article.published_at)}</time>
          )}
          <span className="text-gray-300" aria-hidden="true">·</span>
          <span>{readingTimeLabel(article.reading_time)}</span>
        </div>
      </div>
    </article>
  );
}
