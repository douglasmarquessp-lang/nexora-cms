import { useEffect, useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
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
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();

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

  useEffect(() => { checkAuth(); }, [checkAuth]);
  useEffect(() => {
    if (!isLoading && !isAuthenticated) navigate("/admin/login");
  }, [isLoading, isAuthenticated, navigate]);

  const mediaQuery = useQuery({
    queryKey: ["media", folderId, search],
    queryFn: () => api.get<MediaListResponse>("/media?" + new URLSearchParams({
      ...(folderId ? { folder_id: folderId } : {}),
      ...(search ? { search } : {}),
      page: "1",
      per_page: "50",
    }).toString()),
    enabled: isAuthenticated,
  });

  const foldersQuery = useQuery({
    queryKey: ["folders", folderId],
    queryFn: () => api.get<FolderItem[]>("/media/folders"),
    enabled: isAuthenticated,
  });

  const uploadMutation = useMutation({
    mutationFn: async (files: FileList) => {
      const formData = new FormData();
      Array.from(files).forEach((f) => formData.append("files", f));
      if (folderId) formData.append("folder_id", folderId);
      return api.post("/media/upload", formData);
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
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold text-gray-900">Media Library</h1>
            <button
              onClick={() => setShowNewFolder(!showNewFolder)}
              className="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-brand-600 hover:bg-brand-50"
            >
              <FolderPlus className="h-4 w-4" />
              New Folder
            </button>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="Search media..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-64 rounded-md border border-gray-300 py-2 pl-10 pr-3 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              />
            </div>
            <button
              onClick={() => setViewMode("grid")}
              className={cn(
                "rounded-md p-2",
                viewMode === "grid" ? "bg-brand-50 text-brand-600" : "text-gray-400 hover:text-gray-600",
              )}
            >
              <Grid3X3 className="h-4 w-4" />
            </button>
            <button
              onClick={() => setViewMode("list")}
              className={cn(
                "rounded-md p-2",
                viewMode === "list" ? "bg-brand-50 text-brand-600" : "text-gray-400 hover:text-gray-600",
              )}
            >
              <List className="h-4 w-4" />
            </button>
            <button
              onClick={() => setShowUpload(true)}
              className="inline-flex items-center gap-2 rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              <Upload className="h-4 w-4" />
              Upload
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-6">
        {selected.size > 0 && (
          <div className="mb-4 flex items-center gap-3 rounded-lg border border-brand-200 bg-brand-50 px-4 py-2">
            <span className="text-sm font-medium text-brand-700">
              {selected.size} selected
            </span>
            <button
              onClick={() => deleteMutation.mutate(Array.from(selected))}
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm text-red-600 hover:bg-red-50"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Delete
            </button>
            <button
              onClick={() => {
                const target = window.prompt("Move to folder ID (empty for root):");
                moveMutation.mutate({
                  ids: Array.from(selected),
                  target: target || null,
                });
              }}
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm text-brand-600 hover:bg-brand-100"
            >
              <Folder className="h-3.5 w-3.5" />
              Move
            </button>
            <button
              onClick={() => setSelected(new Set())}
              className="ml-auto text-sm text-gray-500 hover:text-gray-700"
            >
              Clear
            </button>
          </div>
        )}

        {showNewFolder && (
          <div className="mb-4 flex items-center gap-2 rounded-lg border border-gray-200 bg-white p-3">
            <input
              type="text"
              placeholder="Folder name"
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter" && newFolderName.trim()) {
                  createFolderMutation.mutate(newFolderName.trim());
                }
                if (e.key === "Escape") setShowNewFolder(false);
              }}
            />
            <button
              onClick={() => {
                if (newFolderName.trim()) createFolderMutation.mutate(newFolderName.trim());
              }}
              className="rounded-md bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"
            >
              <Check className="h-4 w-4" />
            </button>
            <button
              onClick={() => setShowNewFolder(false)}
              className="rounded-md px-2 py-1.5 text-gray-400 hover:text-gray-600"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        <div className="mb-4 flex items-center gap-1 text-sm text-gray-500">
          <button
            onClick={() => setFolderId(null)}
            className={cn(
              "rounded px-2 py-0.5 hover:bg-gray-100",
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
                className="rounded px-2 py-0.5 hover:bg-gray-100"
              >
                {f.name}
              </button>
            </span>
          ))}
        </div>

        {foldersQuery.data && foldersQuery.data.length > 0 && (
          <div className="mb-6">
            <h2 className="mb-2 text-xs font-medium uppercase tracking-wider text-gray-400">
              Folders
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
                      : "border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50",
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
          <div className="flex items-center justify-center py-20">
            <div className="text-gray-400">Loading...</div>
          </div>
        ) : mediaQuery.data?.media.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-20">
            <Image className="mb-3 h-12 w-12 text-gray-300" />
            <p className="text-sm text-gray-500">No media files yet</p>
            <button
              onClick={() => setShowUpload(true)}
              className="mt-3 rounded-md bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              Upload your first file
            </button>
          </div>
        ) : viewMode === "grid" ? (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            {mediaQuery.data?.media.map((item) => {
              const Icon = getMediaIcon(item.mime_type);
              const thumb = getThumbnailUrl(item);
              const isSelected = selected.has(item.id);
              return (
                <div
                  key={item.id}
                  className={cn(
                    "group relative overflow-hidden rounded-lg border bg-white shadow-sm transition-all hover:shadow-md",
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
                      "absolute left-2 top-2 z-10 rounded-md border bg-white p-0.5 opacity-0 transition-opacity group-hover:opacity-100",
                      isSelected && "opacity-100",
                    )}
                  >
                    <div
                      className={cn(
                        "h-4 w-4 rounded border-2",
                        isSelected
                          ? "border-brand-600 bg-brand-600"
                          : "border-gray-300",
                      )}
                    >
                      {isSelected && (
                        <Check className="h-3 w-3 text-white" />
                      )}
                    </div>
                  </button>

                  {thumb ? (
                    <div className="aspect-square overflow-hidden bg-gray-100">
                      <img
                        src={thumb}
                        alt={item.original_name}
                        className="h-full w-full object-cover"
                        loading="lazy"
                      />
                    </div>
                  ) : (
                    <div className="flex aspect-square items-center justify-center bg-gray-50">
                      <Icon className="h-10 w-10 text-gray-300" />
                    </div>
                  )}

                  <div className="p-2">
                    {editingId === item.id ? (
                      <input
                        type="text"
                        value={editValue}
                        onChange={(e) => setEditValue(e.target.value)}
                        onBlur={() => {
                          if (editValue.trim()) {
                            renameMutation.mutate({ id: item.id, altText: editValue.trim() });
                          } else {
                            setEditingId(null);
                          }
                        }}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            if (editValue.trim()) {
                              renameMutation.mutate({ id: item.id, altText: editValue.trim() });
                            } else {
                              setEditingId(null);
                            }
                          }
                          if (e.key === "Escape") setEditingId(null);
                        }}
                        className="w-full rounded border border-brand-300 px-1 py-0.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                        autoFocus
                      />
                    ) : (
                      <p className="truncate text-xs text-gray-700">
                        {item.alt_text || item.original_name}
                      </p>
                    )}
                    <p className="mt-0.5 text-[10px] text-gray-400">
                      {formatFileSize(item.size)}
                    </p>
                  </div>

                  <div className="absolute right-2 top-2 hidden gap-0.5 group-hover:flex">
                    <button
                      onClick={() => startRename(item)}
                      className="rounded bg-white/90 p-1 text-gray-500 shadow hover:bg-white hover:text-brand-600"
                      title="Rename"
                    >
                      <Pencil className="h-3 w-3" />
                    </button>
                    <button
                      onClick={() => {
                        navigator.clipboard.writeText(item.id);
                      }}
                      className="rounded bg-white/90 p-1 text-gray-500 shadow hover:bg-white hover:text-brand-600"
                      title="Copy ID"
                    >
                      <Copy className="h-3 w-3" />
                    </button>
                    <button
                      onClick={() => deleteMutation.mutate([item.id])}
                      className="rounded bg-white/90 p-1 text-gray-500 shadow hover:bg-white hover:text-red-600"
                      title="Delete"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border bg-white shadow-sm">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Name
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Size
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Dimensions
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    Created
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {mediaQuery.data?.media.map((item) => {
                  const Icon = getMediaIcon(item.mime_type);
                  return (
                    <tr
                      key={item.id}
                      className={cn(
                        "transition-colors hover:bg-gray-50",
                        selected.has(item.id) && "bg-brand-50",
                      )}
                    >
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
                            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                          />
                          <Icon className="h-5 w-5 flex-shrink-0 text-gray-400" />
                          {editingId === item.id ? (
                            <input
                              type="text"
                              value={editValue}
                              onChange={(e) => setEditValue(e.target.value)}
                              onBlur={() => {
                                if (editValue.trim()) {
                                  renameMutation.mutate({ id: item.id, altText: editValue.trim() });
                                } else setEditingId(null);
                              }}
                              onKeyDown={(e) => {
                                if (e.key === "Enter") {
                                  if (editValue.trim()) renameMutation.mutate({ id: item.id, altText: editValue.trim() });
                                  else setEditingId(null);
                                }
                                if (e.key === "Escape") setEditingId(null);
                              }}
                              className="rounded border border-brand-300 px-2 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-brand-500"
                              autoFocus
                            />
                          ) : (
                            <span
                              className="cursor-pointer text-sm text-gray-700 hover:text-brand-600"
                              onClick={() => startRename(item)}
                            >
                              {item.alt_text || item.original_name}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                        {item.mime_type}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                        {formatFileSize(item.size)}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                        {item.width && item.height ? `${item.width}x${item.height}` : "-"}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-500">
                        {new Date(item.created_at).toLocaleDateString()}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-right">
                        <button
                          onClick={() => startRename(item)}
                          className="rounded p-1 text-gray-400 hover:text-brand-600"
                          title="Rename"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => navigator.clipboard.writeText(item.id)}
                          className="rounded p-1 text-gray-400 hover:text-brand-600"
                          title="Copy ID"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => deleteMutation.mutate([item.id])}
                          className="rounded p-1 text-gray-400 hover:text-red-600"
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </main>

      {showUpload && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div
            className="mx-4 w-full max-w-lg rounded-lg bg-white shadow-xl"
            onDragOver={(e) => e.preventDefault()}
            onDrop={handleDrop}
          >
            <div className="flex items-center justify-between border-b px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Upload Media</h2>
              <button
                onClick={() => setShowUpload(false)}
                className="rounded p-1 text-gray-400 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="px-6 py-8">
              <div
                className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 py-12 transition-colors hover:border-brand-400 hover:bg-brand-50"
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="mb-3 h-10 w-10 text-gray-300" />
                <p className="text-sm font-medium text-gray-600">
                  Drop files here or click to upload
                </p>
                <p className="mt-1 text-xs text-gray-400">
                  Images, videos, audio, documents up to 100MB
                </p>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={handleFileSelect}
              />
              {uploadMutation.isPending && (
                <div className="mt-4 text-center text-sm text-brand-600">
                  Uploading...
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
