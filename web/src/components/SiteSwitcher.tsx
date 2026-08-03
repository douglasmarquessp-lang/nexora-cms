import { useSiteStore } from "@/stores/site";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { AlertTriangle, RefreshCw } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function SiteSwitcher() {
  const { sites, currentSite, setCurrentSite, status, error, isLoading, attempts, retrySites } =
    useSiteStore();

  // "idle" (nothing requested yet) and "loading" both mean the sites are not
  // available yet. Showing the "no sites" message during idle would make the
  // user believe there are no sites before the first fetch even runs.
  if (status === "idle" || status === "loading") {
    return (
      <Skeleton
        className="h-8 w-[180px]"
        data-testid="site-switcher-skeleton"
        aria-label="Carregando sites"
      />
    );
  }

  // A failed refresh only produces the error pill when there is no previously
  // loaded data. If sites were loaded before (e.g. a background refresh failed),
  // keep the selector functional so the user can still switch sites; the
  // AdminLayout banner communicates the failed refresh.
  if (status === "error" && sites.length === 0) {
    return (
      <div
        role="alert"
        data-testid="site-switcher-error"
        className="flex max-w-[260px] items-center gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-2 py-1.5"
      >
        <AlertTriangle className="h-4 w-4 shrink-0 text-destructive" aria-hidden="true" />
        <span
          className="hidden min-w-0 flex-1 truncate text-xs font-medium text-destructive sm:inline"
          title={error || undefined}
        >
          Não foi possível carregar os sites
          {attempts > 1 ? ` (após ${attempts} tentativas)` : ""}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-6 shrink-0 gap-1 px-2 text-xs"
          onClick={retrySites}
          disabled={isLoading}
          aria-label="Tentar carregar os sites novamente"
        >
          <RefreshCw className="h-3 w-3" aria-hidden="true" />
          <span className="hidden md:inline">Tentar novamente</span>
          <span className="md:hidden">Tentar</span>
        </Button>
      </div>
    );
  }

  if (status === "empty") {
    return (
      <span
        data-testid="site-switcher-empty"
        className="text-xs text-muted-foreground"
        title="Nenhum site disponível para este usuário"
      >
        Nenhum site disponível
      </span>
    );
  }

  return (
    <Select
      value={currentSite?.id || ""}
      onValueChange={(value) => {
        const site = sites.find((s) => s.id === value);
        if (site) setCurrentSite(site);
      }}
    >
      <SelectTrigger className="h-8 w-[180px] border-none bg-sidebar-muted/50 text-xs text-sidebar-foreground">
        <SelectValue placeholder="Selecionar site" />
      </SelectTrigger>
      <SelectContent>
        {sites.map((site) => (
          <SelectItem key={site.id} value={site.id}>
            <div className="flex items-center gap-2">
              <div className="flex h-5 w-5 items-center justify-center rounded bg-brand-600/20 text-[10px] font-bold text-brand-600">
                {site.name.charAt(0).toUpperCase()}
              </div>
              <span className="text-xs">{site.name}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}