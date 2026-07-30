import { useState } from "react";
import { api } from "@/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { cn } from "@/lib/utils";
import {
  Puzzle,
  Power,
  PowerOff,
  Trash2,
  Plus,
  Search,
  X,
  ChevronDown,
  Info,
  Shield,
  Link,
} from "lucide-react";

interface PluginItem {
  id: string;
  name: string;
  version: string;
  author: string;
  description: string;
  license: string;
  homepage: string;
  min_core_version: string;
  status: string;
  dependencies: { id: string; version: string }[];
  permissions: { permission: string; description: string; default_roles: string[] }[];
  hooks: { hook: string; priority: number }[];
  admin_pages: { title: string; path: string; icon: string; position: number }[];
  has_settings: boolean;
}

function PluginCard({
  plugin,
  onActivate,
  onDeactivate,
  onDelete,
}: {
  plugin: PluginItem;
  onActivate: (id: string) => void;
  onDeactivate: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const isActive = plugin.status === "active";

  return (
    <Card className={cn("transition-all", isActive ? "border-brand-200" : "opacity-75")}>
      <div className="flex items-start gap-4 p-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-50">
          <Puzzle className="h-5 w-5 text-brand-600" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between">
            <div>
              <h3 className="font-medium text-foreground">{plugin.name}</h3>
              <p className="mt-0.5 text-sm text-muted-foreground">v{plugin.version} por {plugin.author}</p>
            </div>
            <span className={cn(
              "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
              isActive ? "bg-green-50 text-green-700" :
              plugin.status === "installed" ? "bg-blue-50 text-blue-700" :
              "bg-muted text-muted-foreground",
            )}>
              {plugin.status}
            </span>
          </div>
          {plugin.description && <p className="mt-2 text-sm text-muted-foreground">{plugin.description}</p>}
          {plugin.dependencies.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1.5">
              {plugin.dependencies.map((dep) => (
                <span key={dep.id} className="inline-flex items-center gap-1 rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                  <Link className="h-3 w-3" />{dep.id}{dep.version && `@${dep.version}`}
                </span>
              ))}
            </div>
          )}
          <div className="mt-3 flex items-center gap-2">
            {isActive ? (
              <Button variant="outline" size="sm" onClick={() => onDeactivate(plugin.id)}>
                <PowerOff className="mr-1 h-3.5 w-3.5" />Desativar
              </Button>
            ) : (
              <Button variant="outline" size="sm" onClick={() => onActivate(plugin.id)}>
                <Power className="mr-1 h-3.5 w-3.5" />Ativar
              </Button>
            )}
            <Button variant="ghost" size="sm" className="text-destructive" onClick={() => onDelete(plugin.id)}>
              <Trash2 className="mr-1 h-3.5 w-3.5" />Remover
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setExpanded(!expanded)}>
              <ChevronDown className={cn("mr-1 h-3.5 w-3.5 transition-transform", expanded && "rotate-180")} />
              Detalhes
            </Button>
          </div>
        </div>
      </div>

      {expanded && (
        <div className="border-t px-4 py-3">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <h4 className="mb-1 flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                <Info className="h-3 w-3" />Info
              </h4>
              <dl className="space-y-1">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Licença</dt>
                  <dd className="text-foreground">{plugin.license || "-"}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Core Mín.</dt>
                  <dd className="text-foreground">{plugin.min_core_version || "-"}</dd>
                </div>
                {plugin.homepage && (
                  <div className="flex justify-between">
                    <dt className="text-muted-foreground">Homepage</dt>
                    <dd className="text-brand-600">{plugin.homepage}</dd>
                  </div>
                )}
              </dl>
            </div>
            <div>
              <h4 className="mb-1 flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                <Shield className="h-3 w-3" />Permissões
              </h4>
              {plugin.permissions.length > 0 ? (
                <ul className="space-y-1">
                  {plugin.permissions.map((perm) => (
                    <li key={perm.permission} className="text-foreground">
                      <span className="font-medium">{perm.permission}</span>
                      {perm.description && <span className="ml-1 text-muted-foreground">- {perm.description}</span>}
                    </li>
                  ))}
                </ul>
              ) : <p className="text-muted-foreground">Nenhuma</p>}
            </div>
          </div>
          {plugin.hooks.length > 0 && (
            <div className="mt-3">
              <h4 className="mb-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">Hooks</h4>
              <div className="flex flex-wrap gap-1.5">
                {plugin.hooks.map((h) => (
                  <span key={h.hook} className="rounded bg-purple-50 px-2 py-0.5 text-xs text-purple-700">{h.hook} (p{h.priority})</span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

export function PluginsPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [showInstall, setShowInstall] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["plugins"],
    queryFn: () => api.get<{ plugins: PluginItem[] }>("/plugins"),
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) => api.post("/plugins/activate", { plugin_id: id }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["plugins"] }),
  });

  const deactivateMutation = useMutation({
    mutationFn: (id: string) => api.post("/plugins/deactivate", { plugin_id: id }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["plugins"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/plugins/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["plugins"] }),
  });

  const installMutation = useMutation({
    mutationFn: (source: string) => api.post("/plugins/install", { source }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plugins"] });
      setShowInstall(false);
    },
  });

  const plugins = data?.plugins || [];
  const filtered = search
    ? plugins.filter((p) => p.name.toLowerCase().includes(search.toLowerCase()) || p.id.toLowerCase().includes(search.toLowerCase()))
    : plugins;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Plugins</h1>
          <p className="text-sm text-muted-foreground">Gerencie os plugins do sistema</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Buscar plugins..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-48 pl-9 sm:w-56"
            />
          </div>
          <Button onClick={() => setShowInstall(true)}>
            <Plus className="mr-2 h-4 w-4" />Instalar
          </Button>
        </div>
      </div>

      {isLoading ? (
        <LoadingState text="Carregando plugins..." />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Puzzle className="h-10 w-10" />}
          title={search ? "Nenhum plugin encontrado" : "Nenhum plugin instalado"}
          description={search ? "Tente outro termo de busca." : "Instale seu primeiro plugin para começar."}
          action={!search && <Button onClick={() => setShowInstall(true)}><Plus className="mr-2 h-4 w-4" />Instalar plugin</Button>}
        />
      ) : (
        <div className="space-y-3">
          {filtered.map((plugin) => (
            <PluginCard
              key={plugin.id}
              plugin={plugin}
              onActivate={(id) => activateMutation.mutate(id)}
              onDeactivate={(id) => deactivateMutation.mutate(id)}
              onDelete={(id) => { if (window.confirm(`Remover plugin "${plugin.name}"?`)) deleteMutation.mutate(id); }}
            />
          ))}
        </div>
      )}

      {showInstall && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <Card className="mx-4 w-full max-w-md">
            <div className="flex items-center justify-between border-b px-6 py-4">
              <h2 className="text-lg font-semibold">Instalar Plugin</h2>
              <Button variant="ghost" size="icon" onClick={() => setShowInstall(false)}>
                <X className="h-5 w-5" />
              </Button>
            </div>
            <div className="px-6 py-4">
              <label className="block text-sm font-medium text-foreground">Fonte do Plugin</label>
              <p className="mt-1 text-xs text-muted-foreground">Digite o nome do diretório do plugin na pasta plugins.</p>
              <Input
                type="text"
                placeholder="exemplo-plugin"
                className="mt-2"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.target as HTMLInputElement).value.trim()) {
                    installMutation.mutate((e.target as HTMLInputElement).value.trim());
                  }
                }}
              />
            </div>
            <div className="flex justify-end gap-3 border-t px-6 py-4">
              <Button variant="outline" onClick={() => setShowInstall(false)}>Cancelar</Button>
              <Button onClick={() => {
                const input = document.querySelector<HTMLInputElement>('[placeholder="exemplo-plugin"]');
                if (input?.value.trim()) installMutation.mutate(input.value.trim());
              }}>Instalar</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
