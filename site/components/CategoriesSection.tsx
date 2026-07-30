import type { Category } from "@/site/lib/api";

interface CategoriesSectionProps {
  categories: Category[];
}

function getCategoryIcon(cat: Category): string {
  const iconName = cat.icon || cat.name.toLowerCase();
  const icons: Record<string, string> = {
    tecnologia: "⚡",
    tech: "⚡",
    ciência: "🔬",
    science: "🔬",
    saúde: "🏥",
    health: "🏥",
    esportes: "⚽",
    sports: "⚽",
    entretenimento: "🎬",
    entertainment: "🎬",
    política: "🏛️",
    politics: "🏛️",
    economia: "📈",
    economy: "📈",
    cultura: "🎭",
    culture: "🎭",
    educação: "📚",
    education: "📚",
    turismo: "✈️",
    travel: "✈️",
    gastronomia: "🍽️",
    food: "🍽️",
    música: "🎵",
    music: "🎵",
    games: "🎮",
    jogos: "🎮",
    negócios: "💼",
    business: "💼",
    design: "🎨",
    geral: "📰",
    general: "📰",
  };
  return icons[iconName] || "📰";
}

export default function CategoriesSection({ categories }: CategoriesSectionProps) {
  if (categories.length === 0) {
    return null;
  }

  return (
    <section>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Categorias</h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {categories.map((cat) => (
          <a
            key={cat.id}
            href={`/categoria/${cat.slug}`}
            className="group flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-5 transition-all hover:shadow-md hover:border-gray-300"
          >
            <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-gray-50 text-xl group-hover:bg-blue-50 transition-colors" aria-hidden="true">
              {getCategoryIcon(cat)}
            </span>
            <div className="min-w-0">
              <h3 className="font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                {cat.name}
              </h3>
              {cat.description && (
                <p className="mt-0.5 text-sm text-gray-500 line-clamp-1">{cat.description}</p>
              )}
            </div>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="ml-auto shrink-0 text-gray-300 group-hover:text-blue-500 transition-colors"
              aria-hidden="true"
            >
              <path d="M9 18l6-6-6-6" />
            </svg>
          </a>
        ))}
      </div>
    </section>
  );
}
