import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center">
      <h1 className="text-4xl font-bold text-gray-900">404</h1>
      <p className="mt-2 text-gray-500">Página não encontrada</p>
      <Link
        to="/admin/dashboard"
        className="mt-4 text-sm text-brand-600 hover:text-brand-700"
      >
        Voltar ao dashboard
      </Link>
    </div>
  );
}
