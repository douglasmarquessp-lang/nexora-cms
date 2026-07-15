import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/api/client";
import { useQuery } from "@tanstack/react-query";

export function DashboardPage() {
  const navigate = useNavigate();
  const { user, isAuthenticated, isLoading, logout, checkAuth } = useAuthStore();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate("/admin/login");
    }
  }, [isLoading, isAuthenticated, navigate]);

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: () => api.get<{ status: string; version: string; timestamp: string }>("/health"),
    enabled: isAuthenticated,
  });

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-gray-500">Carregando...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
          <h1 className="text-xl font-bold text-gray-900">Nexora CMS</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500">{user?.name}</span>
            <button
              onClick={logout}
              className="rounded-md px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
            >
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8">
        <h2 className="mb-6 text-2xl font-semibold text-gray-900">Dashboard</h2>

        <div className="grid gap-6 md:grid-cols-3">
          <div className="rounded-lg border bg-white p-6 shadow-sm">
            <h3 className="text-sm font-medium text-gray-500">Status do Sistema</h3>
            <p className="mt-2 text-lg font-semibold text-green-600">
              {health?.status ?? "---"}
            </p>
          </div>

          <div className="rounded-lg border bg-white p-6 shadow-sm">
            <h3 className="text-sm font-medium text-gray-500">Versão</h3>
            <p className="mt-2 text-lg font-semibold text-gray-900">
              v{health?.version ?? "---"}
            </p>
          </div>

          <div className="rounded-lg border bg-white p-6 shadow-sm">
            <h3 className="text-sm font-medium text-gray-500">Bem-vindo</h3>
            <p className="mt-2 text-lg font-semibold text-gray-900">{user?.name}</p>
          </div>
        </div>
      </main>
    </div>
  );
}
