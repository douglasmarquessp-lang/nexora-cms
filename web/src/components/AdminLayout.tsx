import { useState, useEffect } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { useSiteStore } from "@/stores/site";
import { Sidebar } from "./Sidebar";
import { Header } from "./Header";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { Toaster } from "@/components/ui/sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { AlertTriangle, Info, RefreshCw } from "lucide-react";

export function AdminLayout() {
  const navigate = useNavigate();
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();
  const { fetchSites } = useSiteStore();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      const redirect = encodeURIComponent(location.pathname + location.search);
      navigate(`/admin/login?redirect=${redirect}`, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate, location.pathname, location.search]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchSites();
    }
  }, [isAuthenticated, fetchSites]);

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <aside className="hidden w-60 flex-shrink-0 border-r border-sidebar-border bg-sidebar-background lg:block">
        <Sidebar />
      </aside>

      <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <SheetContent side="left" className="w-60 p-0">
          <Sidebar onNavigate={() => setSidebarOpen(false)} />
        </SheetContent>
      </Sheet>

      <div className="flex flex-1 flex-col overflow-hidden">
        <Header onMenuClick={() => setSidebarOpen(true)} />
        <SiteLoadBanner />
        <main className="flex-1 overflow-y-auto bg-muted/30 p-4 lg:p-6">
          <Outlet />
        </main>
      </div>

      <Toaster />
    </div>
  );
}

function SiteLoadBanner() {
  const { status, error, isLoading, attempts, retrySites } = useSiteStore();

  if (status === "error") {
    return (
      <div
        role="alert"
        data-testid="site-load-banner"
        className="flex flex-wrap items-center gap-2 border-b border-destructive/30 bg-destructive/5 px-4 py-2"
      >
        <AlertTriangle className="h-4 w-4 shrink-0 text-destructive" aria-hidden="true" />
        <span className="text-xs font-medium text-destructive">
          Não foi possível carregar os sites{attempts > 1 ? ` (após ${attempts} tentativas)` : ""}
        </span>
        {error && (
          <span className="hidden max-w-[45%] truncate text-xs text-destructive/70 lg:inline" title={error}>
            {error}
          </span>
        )}
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="ml-auto h-6 gap-1 px-2 text-xs"
          onClick={retrySites}
          disabled={isLoading}
        >
          <RefreshCw className="h-3 w-3" aria-hidden="true" />
          Tentar novamente
        </Button>
      </div>
    );
  }

  if (status === "empty") {
    return (
      <div
        role="status"
        data-testid="site-load-banner-empty"
        className="flex flex-wrap items-center gap-2 border-b border-muted bg-muted/40 px-4 py-2"
      >
        <Info className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        <span className="text-xs font-medium text-muted-foreground">
          Nenhum site disponível para este usuário.
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="ml-auto h-6 gap-1 px-2 text-xs"
          onClick={retrySites}
          disabled={isLoading}
        >
          <RefreshCw className="h-3 w-3" aria-hidden="true" />
          Recarregar
        </Button>
      </div>
    );
  }

  return null;
}

function LoadingScreen() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="w-80 space-y-4">
        <div className="flex items-center justify-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-sm font-bold text-white">
            N
          </div>
          <span className="text-lg font-semibold">Nexora CMS</span>
        </div>
        <div className="space-y-2">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      </div>
    </div>
  );
}
