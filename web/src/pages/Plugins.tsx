import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
import { cn } from "@/lib/utils";

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
    <div
      className={cn(
        "rounded-lg border bg-white shadow-sm transition-all",
        isActive ? "border-brand-200" : "border-gray-200 opacity-75",
      )}
    >
      <div className="flex items-start gap-4 p-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-50">
          <Puzzle className="h-5 w-5 text-brand-600" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between">
            <div>
              <h3 className="font-medium text-gray-900">{plugin.name}</h3>
              <p className="mt-0.5 text-sm text-gray-500">
                v{plugin.version} by {plugin.author}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
                  isActive
                    ? "bg-green-50 text-green-700"
                    : plugin.status === "installed"
                      ? "bg-blue-50 text-blue-700"
                      : "bg-gray-100 text-gray-600",
                )}
              >
                {plugin.status}
              </span>
            </div>
          </div>
          {plugin.description && (
            <p className="mt-2 text-sm text-gray-600">{plugin.description}</p>
          )}
          {plugin.dependencies.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1.5">
              {plugin.dependencies.map((dep) => (
                <span
                  key={dep.id}
                  className="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
                >
                  <Link className="h-3 w-3" />
                  {dep.id}
                  {dep.version && `@${dep.version}`}
                </span>
              ))}
            </div>
          )}
          <div className="mt-3 flex items-center gap-2">
            {isActive ? (
              <button
                onClick={() => onDeactivate(plugin.id)}
                className="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium text-amber-600 hover:bg-amber-50"
              >
                <PowerOff className="h-3.5 w-3.5" />
                Deactivate
              </button>
            ) : (
              <button
                onClick={() => onActivate(plugin.id)}
                className="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium text-green-600 hover:bg-green-50"
              >
                <Power className="h-3.5 w-3.5" />
                Activate
              </button>
            )}
            <button
              onClick={() => onDelete(plugin.id)}
              className="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Remove
            </button>
            <button
              onClick={() => setExpanded(!expanded)}
              className="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs text-gray-500 hover:bg-gray-50"
            >
              <ChevronDown
                className={cn("h-3.5 w-3.5 transition-transform", expanded && "rotate-180")}
              />
              Details
            </button>
          </div>
        </div>
      </div>

      {expanded && (
        <div className="border-t border-gray-100 px-4 py-3">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <h4 className="mb-1 flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-gray-400">
                <Info className="h-3 w-3" />
                Info
              </h4>
              <dl className="space-y-1">
                <div className="flex justify-between">
                  <dt className="text-gray-500">License</dt>
                  <dd className="text-gray-700">{plugin.license || "-"}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-500">Min Core</dt>
                  <dd className="text-gray-700">{plugin.min_core_version || "-"}</dd>
                </div>
                {plugin.homepage && (
                  <div className="flex justify-between">
                    <dt className="text-gray-500">Homepage</dt>
                    <dd className="text-brand-600">{plugin.homepage}</dd>
                  </div>
                )}
              </dl>
            </div>
            <div>
              <h4 className="mb-1 flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-gray-400">
                <Shield className="h-3 w-3" />
                Permissions
              </h4>
              {plugin.permissions.length > 0 ? (
                <ul className="space-y-1">
                  {plugin.permissions.map((perm) => (
                    <li key={perm.permission} className="text-gray-700">
                      <span className="font-medium">{perm.permission}</span>
                      {perm.description && (
                        <span className="ml-1 text-gray-500">- {perm.description}</span>
                      )}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-gray-400">None</p>
              )}
            </div>
          </div>
          {plugin.hooks.length > 0 && (
            <div className="mt-3">
              <h4 className="mb-1 text-xs font-medium uppercase tracking-wider text-gray-400">
                Hooks
              </h4>
              <div className="flex flex-wrap gap-1.5">
                {plugin.hooks.map((h) => (
                  <span
                    key={h.hook}
                    className="rounded bg-purple-50 px-2 py-0.5 text-xs text-purple-700"
                  >
                    {h.hook} (p{h.priority})
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function InstallModal({
  onClose,
  onInstall,
}: {
  onClose: () => void;
  onInstall: (source: string) => void;
}) {
  const [source, setSource] = useState("");

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="mx-4 w-full max-w-md rounded-lg bg-white shadow-xl">
        <div className="flex items-center justify-between border-b px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">Install Plugin</h2>
          <button onClick={onClose} className="rounded p-1 text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="px-6 py-4">
          <label className="block text-sm font-medium text-gray-700">Plugin Source</label>
          <p className="mt-1 text-xs text-gray-500">
            Enter the plugin directory name to install from the plugins folder.
          </p>
          <input
            type="text"
            value={source}
            onChange={(e) => setSource(e.target.value)}
            placeholder="example-plugin"
            className="mt-2 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            autoFocus
          />
        </div>
        <div className="flex justify-end gap-3 border-t px-6 py-4">
          <button
            onClick={onClose}
            className="rounded-md px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={() => {
              if (source.trim()) onInstall(source.trim());
            }}
            disabled={!source.trim()}
            className="rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Install
          </button>
        </div>
      </div>
    </div>
  );
}

export function PluginsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();
  const [search, setSearch] = useState("");
  const [showInstall, setShowInstall] = useState(false);

  useEffect(() => { checkAuth(); }, [checkAuth]);
  useEffect(() => {
    if (!isLoading && !isAuthenticated) navigate("/admin/login");
  }, [isLoading, isAuthenticated, navigate]);

  const { data, isLoading: pluginsLoading } = useQuery({
    queryKey: ["plugins"],
    queryFn: () => api.get<{ plugins: PluginItem[] }>("/plugins"),
    enabled: isAuthenticated,
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
    ? plugins.filter(
        (p) =>
          p.name.toLowerCase().includes(search.toLowerCase()) ||
          p.id.toLowerCase().includes(search.toLowerCase()),
      )
    : plugins;

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-gray-500">Carregando...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b bg-white shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
          <h1 className="text-xl font-semibold text-gray-900">Plugins</h1>
          <div className="flex items-center gap-3">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="Search plugins..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-56 rounded-md border border-gray-300 py-2 pl-10 pr-3 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              />
            </div>
            <button
              onClick={() => setShowInstall(true)}
              className="inline-flex items-center gap-2 rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              <Plus className="h-4 w-4" />
              Install
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6">
        {pluginsLoading ? (
          <div className="flex items-center justify-center py-20">
            <div className="text-gray-400">Loading...</div>
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-20">
            <Puzzle className="mb-3 h-12 w-12 text-gray-300" />
            <p className="text-sm text-gray-500">
              {search ? "No plugins match your search" : "No plugins installed"}
            </p>
            {!search && (
              <button
                onClick={() => setShowInstall(true)}
                className="mt-3 rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                Install your first plugin
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-3">
            {filtered.map((plugin) => (
              <PluginCard
                key={plugin.id}
                plugin={plugin}
                onActivate={(id) => activateMutation.mutate(id)}
                onDeactivate={(id) => deactivateMutation.mutate(id)}
                onDelete={(id) => {
                  if (window.confirm(`Remove plugin "${plugin.name}"?`)) {
                    deleteMutation.mutate(id);
                  }
                }}
              />
            ))}
          </div>
        )}
      </main>

      {showInstall && (
        <InstallModal
          onClose={() => setShowInstall(false)}
          onInstall={(source) => installMutation.mutate(source)}
        />
      )}
    </div>
  );
}
