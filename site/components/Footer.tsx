import Link from "next/link";

interface FooterProps {
  siteName?: string;
}

export default function Footer({ siteName = "Nexora" }: FooterProps) {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="mt-16 border-t border-gray-200 bg-gray-50" role="contentinfo">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          <div className="sm:col-span-2 lg:col-span-1">
            <Link href="/" className="text-lg font-bold text-gray-900">
              {siteName}
            </Link>
            <p className="mt-3 text-sm leading-relaxed text-gray-500">
              Plataforma de gerenciamento de conteúdo com inteligência artificial.
              Publique, gerencie e otimize seu conteúdo editorial.
            </p>
          </div>

          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-900">
              Navegação
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <Link href="/" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">
                  Início
                </Link>
              </li>
              <li>
                <Link href="/artigos" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">
                  Artigos
                </Link>
              </li>
              <li>
                <Link href="/categorias" className="text-sm text-gray-500 hover:text-gray-900 transition-colors">
                  Categorias
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-900">
              Legal
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <span className="text-sm text-gray-400 cursor-default">
                  Política Editorial
                </span>
              </li>
              <li>
                <span className="text-sm text-gray-400 cursor-default">
                  Termos de Uso
                </span>
              </li>
              <li>
                <span className="text-sm text-gray-400 cursor-default">
                  Política de Privacidade
                </span>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-900">
              Contato
            </h3>
            <ul className="mt-4 space-y-3">
              <li>
                <span className="text-sm text-gray-400 cursor-default">
                  contato@nexora-cms.com
                </span>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-10 border-t border-gray-200 pt-8">
          <p className="text-center text-xs text-gray-400">
            &copy; {currentYear} {siteName}. Todos os direitos reservados.
          </p>
        </div>
      </div>
    </footer>
  );
}
