import { useState, useRef, useCallback } from "react";
import { api } from "@/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import {
  Upload,
  Image,
  Film,
  FileText,
  Music,
  File,
  Grid3X3,
  List,
  Search,
  Trash2,
  Copy,
  Pencil,
  FolderPlus,
  Folder,
  ChevronRight,
  X,
  Check,
} from "lucide-react";

interface MediaItem {
  id: string;
  site_id: string;
  folder_id: string | null;
  filename: string;
  original_name: string;
  mime_type: string;
  extension: string;
  size: number;
  width: number | null;
  height: number | null;
  duration: number;
  hash: string;
  alt_text: string;
  caption: string;
  storage_provider: string;
  storage_key: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

interface FolderItem {
  id: string;
  site_id: string;
  parent_id: string | null;
  name: string;
  slug: string;
  description: string;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

interface MediaListResponse {
  media: MediaItem[];
  total: number;
  page: number;
  per_page: number;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function getMediaIcon(mimeType: string) {
  if (mimeType.startsWith("image/")) return Image;
  if (mimeType.startsWith("video/")) return Film;
  if (mimeType.startsWith("audio/")) return Music;
  if (mimeType.startsWith("text/") || mimeType === "application/pdf") return FileText;
  return File;
}

function getThumbnailUrl(item: MediaItem): string {
  if (item.mime_type.startsWith("image/")) {
    return `/api/v1/media/${item.id}?variant=thumbnail`;
  }
  return "";
}

export function MediaLibraryPage() {
  const queryClient = useQueryClient();
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [search, setSearch] = useState("");
  const [folderId, setFolderId] = useState<string | null>(null);
  const [folderPath, setFolderPath] = useState<FolderItem[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [showUpload, setShowUpload] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [showNewFolder, setShowNewFolder] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const mediaQuery = useQuery({
    queryKey: ["media", folderId, search],
    queryFn: () => api.get<MediaListResponse>(
      "/media?" + new URLSearchParams({
        ...(folderId ? { folder_id: folderId } : {}),
        ...(search ? { search } : {}),
        page: "1",
        per_page: "50",
      }).toString(),
    ),
  });

  const foldersQuery = useQuery({
    queryKey: ["folders", folderId],
    queryFn: () => api.get<FolderItem[]>("/media/folders"),
  });

  const uploadMutation = useMutation({
    mutationFn: async (files: FileList) => {
      const formData = new FormData();
      Array.from(files).forEach((f) => formData.append("files", f));
      if (folderId) formData.append("folder_id", folderId);
      return api.post("/media/upload", formData, { formData: true });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
      setShowUpload(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (ids: string[]) =>
      Promise.all(ids.map((id) => api.delete(`/media/${id}`))),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
      setSelected(new Set());
    },
  });

  const renameMutation = useMutation({
    mutationFn: ({ id, altText }: { id: string; altText: string }) =>
      api.patch(`/media/${id}`, { alt_text: altText }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
      setEditingId(null);
    },
  });

  const createFolderMutation = useMutation({
    mutationFn: (name: string) =>
      api.post("/media/folders", { name, parent_id: folderId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      setShowNewFolder(false);
      setNewFolderName("");
    },
  });

  const moveMutation = useMutation({
    mutationFn: ({ ids, target }: { ids: string[]; target: string | null }) =>
      api.post("/media/move", { media_ids: ids, folder_id: target }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["media"] });
      setSelected(new Set());
    },
  });

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (e.target.files?.length) uploadMutation.mutate(e.target.files);
    },
    [uploadMutation],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (e.dataTransfer.files.length) uploadMutation.mutate(e.dataTransfer.files);
    },
    [uploadMutation],
  );

  const startRename = (item: MediaItem) => {
    setEditingId(item.id);
    setEditValue(item.alt_text || item.original_name);
  };

  const breadcrumbs = (
    <div className="mb-4 flex items-center gap-1 text-sm text-muted-foreground">
      <button
        onClick={() => { setFolderId(null); setFolderPath([]); }}
        className={cn(
          "rounded px-2 py-0.5 hover:bg-accent",
          !folderId && "font-medium text-brand-600",
        )}
      >
        Root
      </button>
      {folderPath.map((f) => (
        <span key={f.id} className="flex items-center gap-1">
          <ChevronRight className="h-3 w-3" />
          <button
            onClick={() => {
              setFolderId(f.id);
              setFolderPath(folderPath.slice(0, folderPath.indexOf(f) + 1));
            }}
            className="rounded px-2 py-0.5 hover:bg-accent"
          >
            {f.name}
          </button>
        </span>
      ))}
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Media Library</h1>
          <p className="text-sm text-muted-foreground">Gerencie seus arquivos de mídia</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Buscar mídia..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-48 pl-9 sm:w-56"
            />
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setViewMode("grid")}
            className={cn(viewMode === "grid" && "bg-accent text-accent-foreground")}
          >
            <Grid3X3 className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setViewMode("list")}
            className={cn(viewMode === "list" && "bg-accent text-accent-foreground")}
          >
            <List className="h-4 w-4" />
          </Button>
          <Button onClick={() => setShowUpload(true)}>
            <Upload className="mr-2 h-4 w-4" />
            Upload
          </Button>
        </div>
      </div>

      {selected.size > 0 && (
        <div className="flex items-center gap-3 rounded-lg border bg-accent/50 px-4 py-2">
          <span className="text-sm font-medium">{selected.size} selecionado(s)</span>
          <Button variant="ghost" size="sm" onClick={() => deleteMutation.mutate(Array.from(selected))}>
            <Trash2 className="mr-1 h-3.5 w-3.5" />
            Excluir
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              const target = window.prompt("ID da pasta de destino (vazio = raiz):");
              moveMutation.mutate({ ids: Array.from(selected), target: target || null });
            }}
          >
            <Folder className="mr-1 h-3.5 w-3.5" />
            Mover
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())} className="ml-auto">
            Limpar
          </Button>
        </div>
      )}

      {showNewFolder && (
        <div className="flex items-center gap-2 rounded-lg border bg-card p-3">
          <Input
            type="text"
            placeholder="Nome da pasta"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && newFolderName.trim()) {
                createFolderMutation.mutate(newFolderName.trim());
              }
              if (e.key === "Escape") setShowNewFolder(false);
            }}
          />
          <Button size="sm" onClick={() => { if (newFolderName.trim()) createFolderMutation.mutate(newFolderName.trim()); }}>
            <Check className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setShowNewFolder(false)}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      )}

      {breadcrumbs}

      {foldersQuery.data && foldersQuery.data.length > 0 && (
        <div>
          <h2 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            Pastas
          </h2>
          <div className="flex flex-wrap gap-2">
            {foldersQuery.data.map((f) => (
              <button
                key={f.id}
                onClick={() => {
                  setFolderId(f.id);
                  setFolderPath([...folderPath, f]);
                }}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm transition-colors",
                  folderId === f.id
                    ? "border-brand-300 bg-brand-50 text-brand-700"
                    : "border-border bg-card text-foreground hover:bg-accent",
                )}
              >
                <Folder className="h-4 w-4" />
                {f.name}
              </button>
            ))}
          </div>
        </div>
      )}

      {mediaQuery.isLoading ? (
        <LoadingState text="Carregando mídia..." />
      ) : mediaQuery.data?.media.length === 0 ? (
        <EmptyState
          icon={<Image className="h-10 w-10" />}
          title="Nenhum arquivo de mídia"
          description="Faça upload do seu primeiro arquivo para começar."
          action={
            <Button onClick={() => setShowUpload(true)}>
              <Upload className="mr-2 h-4 w-4" />
              Fazer upload
            </Button>
          }
        />
      ) : viewMode === "grid" ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
          {mediaQuery.data?.media.map((item) => {
            const Icon = getMediaIcon(item.mime_type);
            const thumb = getThumbnailUrl(item);
            const isSelected = selected.has(item.id);
            return (
              <Card
                key={item.id}
                className={cn(
                  "group relative overflow-hidden transition-all",
                  isSelected && "ring-2 ring-brand-500",
                )}
              >
                <button
                  onClick={() => {
                    setSelected((prev) => {
                      const next = new Set(prev);
                      if (next.has(item.id)) next.delete(item.id);
                      else next.add(item.id);
                      return next;
                    });
                  }}
                  className={cn(
                    "absolute left-2 top-2 z-10 rounded-md border bg-background p-0.5 opacity-0 transition-opacity group-hover:opacity-100",
                    isSelected && "opacity-100",
                  )}
                  aria-label={isSelected ? "Deselecionar" : "Selecionar"}
                >
                  <div className={cn("h-4 w-4 rounded border-2", isSelected ? "border-brand-600 bg-brand-600" : "border-muted-foreground")}>
                    {isSelected && <Check className="h-3 w-3 text-white" />}
                  </div>
                </button>

                {thumb ? (
                  <div className="aspect-square overflow-hidden bg-muted">
                    <img src={thumb} alt={item.original_name} className="h-full w-full object-cover" loading="lazy" />
                  </div>
                ) : (
                  <div className="flex aspect-square items-center justify-center bg-muted">
                    <Icon className="h-10 w-10 text-muted-foreground/50" />
                  </div>
                )}

                <CardContent className="p-2">
                  {editingId === item.id ? (
                    <Input
                      type="text"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => { if (editValue.trim()) renameMutation.mutate({ id: item.id, altText: editValue.trim() }); else setEditingId(null); }}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") { if (editValue.trim()) renameMutation.mutate({ id: item.id, altText: editValue.trim() }); else setEditingId(null); }
                        if (e.key === "Escape") setEditingId(null);
                      }}
                      className="h-7 text-xs"
                      autoFocus
                    />
                  ) : (
                    <p className="truncate text-xs text-foreground">{item.alt_text || item.original_name}</p>
                  )}
                  <p className="mt-0.5 text-[10px] text-muted-foreground">{formatFileSize(item.size)}</p>
                </CardContent>

                <div className="absolute right-2 top-2 hidden gap-0.5 group-hover:flex">
                  <Button variant="ghost" size="icon" className="h-6 w-6 bg-background/90" onClick={() => startRename(item)} title="Renomear">
                    <Pencil className="h-3 w-3" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-6 w-6 bg-background/90" onClick={() => navigator.clipboard.writeText(item.id)} title="Copiar ID">
                    <Copy className="h-3 w-3" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-6 w-6 bg-background/90 hover:text-destructive" onClick={() => deleteMutation.mutate([item.id])} title="Excluir">
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </Card>
            );
          })}
        </div>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Nome</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Tipo</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Tamanho</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Dimensões</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Criado</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Ações</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {mediaQuery.data?.media.map((item) => {
                  const Icon = getMediaIcon(item.mime_type);
                  return (
                    <tr key={item.id} className={cn("transition-colors hover:bg-muted/50", selected.has(item.id) && "bg-accent")}>
                      <td className="whitespace-nowrap px-4 py-3">
                        <div className="flex items-center gap-3">
                          <input
                            type="checkbox"
                            checked={selected.has(item.id)}
                            onChange={() => {
                              setSelected((prev) => {
                                const next = new Set(prev);
                                if (next.has(item.id)) next.delete(item.id);
                                else next.add(item.id);
                                return next;
                              });
                            }}
                            className="h-4 w-4 rounded border-muted-foreground text-brand-600 focus:ring-brand-500"
                          />
                          <Icon className="h-5 w-5 flex-shrink-0 text-muted-foreground" />
                          {editingId === item.id ? (
                            <Input
                              type="text"
                              value={editValue}
                              onChange={(e) => setEditValue(e.target.value)}
                              onBlur={() => { if (editValue.trim()) renameMutation.mutate({ id: item.id, altText: editValue.trim() }); else setEditingId(null); }}
                              onKeyDown={(e) => {
                                if (e.key === "Enter") { if (editValue.trim()) renameMutation.mutate({ id: item.id, altText: editValue.trim() }); else setEditingId(null); }
                                if (e.key === "Escape") setEditingId(null);
                              }}
                              className="h-7 w-40 text-sm"
                              autoFocus
                            />
                          ) : (
                            <span className="cursor-pointer text-sm text-foreground hover:text-brand-600" onClick={() => startRename(item)}>
                              {item.alt_text || item.original_name}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-muted-foreground">{item.mime_type}</td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-muted-foreground">{formatFileSize(item.size)}</td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-muted-foreground">
                        {item.width && item.height ? `${item.width}x${item.height}` : "-"}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-muted-foreground">
                        {new Date(item.created_at).toLocaleDateString()}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-right">
                        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => startRename(item)} title="Renomear">
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => navigator.clipboard.writeText(item.id)} title="Copiar ID">
                          <Copy className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-7 w-7 hover:text-destructive" onClick={() => deleteMutation.mutate([item.id])} title="Excluir">
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {showUpload && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <Card
            className="mx-4 w-full max-w-lg"
            onDragOver={(e) => e.preventDefault()}
            onDrop={handleDrop}
          >
            <div className="flex items-center justify-between border-b px-6 py-4">
              <h2 className="text-lg font-semibold">Upload Media</h2>
              <Button variant="ghost" size="icon" onClick={() => setShowUpload(false)}>
                <X className="h-5 w-5" />
              </Button>
            </div>
            <div className="px-6 py-8">
              <div
                className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 bg-muted/50 py-12 transition-colors hover:border-brand-400 hover:bg-brand-50"
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="mb-3 h-10 w-10 text-muted-foreground/50" />
                <p className="text-sm font-medium text-foreground">Arraste arquivos ou clique para upload</p>
                <p className="mt-1 text-xs text-muted-foreground">Imagens, vídeos, áudio, documentos até 100MB</p>
              </div>
              <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleFileSelect} />
              {uploadMutation.isPending && (
                <div className="mt-4 text-center text-sm text-brand-600">Enviando...</div>
              )}
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
