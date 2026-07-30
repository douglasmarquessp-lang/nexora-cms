import { getArticles, getCategories } from "@/site/lib/api";
import Header from "@/site/components/Header";
import Hero from "@/site/components/Hero";
import ArticleList from "@/site/components/ArticleList";
import CategoriesSection from "@/site/components/CategoriesSection";
import Sidebar from "@/site/components/Sidebar";
import Footer from "@/site/components/Footer";

export default async function HomePage() {
  let articles;
  let categories;

  try {
    [articles, categories] = await Promise.all([
      getArticles({ limit: 9 }),
      getCategories().catch(() => null),
    ]);
  } catch {
    return (
      <div className="flex min-h-screen flex-col">
        <Header />
        <main className="flex-1 flex items-center justify-center">
          <div className="text-center px-4">
            <h1 className="text-4xl font-bold text-gray-900 mb-4">Nexora CMS</h1>
            <p className="text-lg text-gray-500">
              Serviço temporariamente indisponível. Tente novamente mais tarde.
            </p>
          </div>
        </main>
        <Footer />
      </div>
    );
  }

  const catList = categories?.categories ?? [];
  const featuredArticle = articles.articles.length > 0 ? articles.articles[0] : null;
  const remainingArticles = articles.articles.length > 1 ? articles.articles.slice(1) : [];
  const sidebarArticles = articles.articles.slice(0, 5);

  return (
    <div className="flex min-h-screen flex-col">
      <Header siteName="Nexora" categories={catList.map((c) => ({ name: c.name, slug: c.slug }))} />

      <main className="flex-1">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          {featuredArticle ? (
            <div className="mt-8">
              <Hero article={featuredArticle} />
            </div>
          ) : (
            <div className="mt-8 mb-8 flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-gray-200 py-24 text-center">
              <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" className="text-gray-300 mb-6" aria-hidden="true">
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
                <polyline points="14 2 14 8 20 8" />
                <line x1="16" x2="8" y1="13" y2="13" />
                <line x1="16" x2="8" y1="17" y2="17" />
              </svg>
              <h1 className="text-2xl font-bold text-gray-900 mb-2">Bem-vindo ao Nexora CMS</h1>
              <p className="text-gray-500 max-w-md text-center">
                Este é o início da sua jornada de conteúdo. Publique seu primeiro artigo para vê-lo aqui em destaque.
              </p>
            </div>
          )}

          <div className="mt-12 lg:grid lg:grid-cols-3 lg:gap-10">
            <div className="lg:col-span-2 space-y-12">
              {featuredArticle && (
                <section>
                  <ArticleList articles={remainingArticles} total={articles.total} />
                </section>
              )}

              <CategoriesSection categories={catList} />
            </div>

            <aside className="mt-12 lg:mt-0">
              <div className="lg:sticky lg:top-24">
                <Sidebar recentArticles={sidebarArticles} categories={catList} />
              </div>
            </aside>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}
