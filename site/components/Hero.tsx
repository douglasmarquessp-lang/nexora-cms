import Link from "next/link";
import type { Article } from "@/site/lib/api";
import { formatDate, readingTimeLabel } from "@/site/lib/api";

interface HeroProps {
  article: Article;
}

export default function Hero({ article }: HeroProps) {
  return (
    <section className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900">
      <div className="relative z-10 mx-auto flex flex-col-reverse lg:flex-row">
        <div className="flex flex-col justify-center px-6 py-10 sm:px-10 lg:w-1/2 lg:py-16">
          {article.categories && article.categories.length > 0 && (
            <span className="mb-4 inline-flex w-fit items-center rounded-full bg-blue-600 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-white">
              {article.categories[0]}
            </span>
          )}

          <h1 className="text-3xl font-bold leading-tight text-white sm:text-4xl lg:text-5xl">
            <Link href={`/${article.slug}`} className="hover:underline decoration-blue-400 underline-offset-4">
              {article.title}
            </Link>
          </h1>

          {article.excerpt && (
            <p className="mt-4 text-base leading-relaxed text-gray-300 sm:text-lg line-clamp-3">
              {article.excerpt}
            </p>
          )}

          <div className="mt-6 flex flex-wrap items-center gap-4 text-sm text-gray-400">
            {article.published_at && (
              <time dateTime={article.published_at} className="flex items-center gap-1.5">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <rect width="18" height="18" x="3" y="4" rx="2" ry="2" /><line x1="16" x2="16" y1="2" y2="6" /><line x1="8" x2="8" y1="2" y2="6" /><line x1="3" x2="21" y1="10" y2="10" />
                </svg>
                {formatDate(article.published_at)}
              </time>
            )}
            <span className="flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
              </svg>
              {readingTimeLabel(article.reading_time)}
            </span>
          </div>

          <div className="mt-8">
            <Link
              href={`/${article.slug}`}
              className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-3 text-sm font-semibold text-white shadow-lg shadow-blue-600/25 transition-all hover:bg-blue-700 hover:shadow-blue-700/30 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-900"
            >
              Ler artigo
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M5 12h14" /><path d="m12 5 7 7-7 7" />
              </svg>
            </Link>
          </div>
        </div>

        <div className="lg:w-1/2 relative min-h-[280px] sm:min-h-[360px] lg:min-h-full">
          {article.featured_image_url ? (
            <img
              src={article.featured_image_url}
              alt={article.title}
              className="absolute inset-0 h-full w-full object-cover"
              loading="eager"
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-gray-800 to-gray-900">
              <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" className="text-gray-600" aria-hidden="true">
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" /><circle cx="9" cy="9" r="2" /><path d="m21 15-3.1-3.1a2 2 0 0 0-2.8 0L6 21" />
              </svg>
            </div>
          )}
          {article.featured_image_url && (
            <div className="absolute inset-0 bg-gradient-to-r from-gray-900/60 via-transparent to-transparent lg:bg-gradient-to-r lg:from-gray-900/80 lg:via-transparent lg:to-transparent" />
          )}
        </div>
      </div>
    </section>
  );
}
