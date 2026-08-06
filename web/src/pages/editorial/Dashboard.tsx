import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/api/client";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { cn, formatDate, formatRelativeTime } from "@/lib/utils";
import { useCurrentSiteId, siteQueryKey } from "@/lib/queryKeys";
import {
  PIPELINE_STAGES,
  STAGE_COLORS,
  STAGE_LABELS,
  type PipelineItem,
  type PipelineResponse,
  type PipelineStage,
  type PipelineStats,
  type PublishReadiness,
} from "@/lib/editorial";

type MoveTarget = "scheduled" | "published" | "draft";

const MOVE_TARGETS: Record<PipelineStage, { target: MoveTarget; label: string }[]> = {
  idea: [],
  research: [],
  outline: [],
  writing: [],
  seo: [],
  eeat: [{ target: "draft", label: "Rascunho" }],
  translation: [],
  review: [{ target: "draft", label: "Reabrir como rascunho" }],
  approval: [
    { target: "scheduled", label: "Agendar" },
    { target: "published", label: "Publicar" },
  ],
  scheduled: [
    { target: "published", label: "Publicar" },
    { target: "draft", label: "Rascunho" },
  ],
  published: [],
};

function StageBadge({ stage }: { stage: PipelineStage }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded bg-muted px-2 py-0.5 text-xs font-medium">
      <span className={cn("h-2 w-2 rounded-full", STAGE_COLORS[stage]?.dot ?? "bg-muted")} />
      {STAGE_LABELS[stage] ?? stage}
    </span>
  );
}

function ScorePill({ label, value }: { label: string; value?: number | null }) {
  if (value == null) return null;
  return (
    <span className="rounded bg-muted px-1.5 py-0.5 text-[11px] font-medium">
      {label} {value.toFixed(1)}
    </span>
  );
}

function PipelineCard({
  item,
  onMove,
}: {
  item: PipelineItem;
  onMove: (item: PipelineItem, target: MoveTarget) => void;
}) {
  const targets = MOVE_TARGETS[item.stage] ?? [];
  const isPost = item.engine === "posts";

  return (
    <div
      className={cn(
        "rounded-md border bg-card p-3 shadow-sm transition-shadow hover:shadow",
        STAGE_COLORS[item.stage]?.card ?? "border-border",
        isPost && targets.length > 0 && "cursor-grab active:cursor-grabbing",
      )}
      draggable={isPost && targets.length > 0}
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", JSON.stringify({ id: item.id, stage: item.stage }));
        e.dataTransfer.effectAllowed = "move";
      }}
      data-testid={`pipeline-card-${item.id}`}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-sm font-medium">{item.title || "(sem título)"}</p>
        {isPost && item.actionable && (
          <Link
            to={`/admin/editorial/review/${item.id}`}
            className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
            aria-label="Abrir revisão"
            data-testid={`pipeline-open-${item.id}`}
          >
            <span className="text-xs underline">Abrir</span>
          </Link>
        )}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        <StageBadge stage={item.stage} />
        <ScorePill label="SEO" value={item.seo_score} />
        <ScorePill label="EEAT" value={item.eeat_score} />
      </div>
      <div className="mt-2 flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span className="truncate">
          {item.language} · {item.engine}
          {item.scheduled_at ? ` · ${formatDate(item.scheduled_at)}` : ""}
        </span>
        <span title={new Date(item.updated_at).toLocaleString()}>
          {formatRelativeTime(item.updated_at)}
        </span>
      </div>
      {isPost && targets.length > 0 && (
        <div className="mt-2">
          <Select
            value=""
            onValueChange={(value) => onMove(item, value as MoveTarget)}
          >
            <SelectTrigger className="h-7 text-xs" aria-label="Mover item" data-testid={`pipeline-move-${item.id}`}>
              <SelectValue placeholder="Mover…" />
            </SelectTrigger>
            <SelectContent>
              {targets.map((t) => (
                <SelectItem key={t.target} value={t.target}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
    </div>
  );
}

function PipelineBoard({
  items,
  onMove,
}: {
  items: PipelineItem[];
  onMove: (item: PipelineItem, target: MoveTarget) => void;
}) {
  const [dragging, setDragging] = useState<PipelineItem | null>(null);
  const [dragOver, setDragOver] = useState<PipelineStage | null>(null);

  const byStage = useMemo(() => {
    const map = new Map<PipelineStage, PipelineItem[]>();
    for (const stage of PIPELINE_STAGES) map.set(stage, []);
    for (const item of items) {
      const list = map.get(item.stage);
      if (list) list.push(item);
    }
    return map;
  }, [items]);

  function onDrop(stage: PipelineStage) {
    setDragOver(null);
    if (!dragging) return;
    const targets = MOVE_TARGETS[stage] ?? [];
    const isTarget = targets.some((t) => t.target === "scheduled" || t.target === "published" || t.target === "draft");
    if (!isTarget) return;
    onMove(dragging, stage as MoveTarget);
    setDragging(null);
  }

  return (
    <div className="flex gap-3 overflow-x-auto pb-4">
      {PIPELINE_STAGES.map((stage) => {
        const columnItems = byStage.get(stage) ?? [];
        return (
          <div
            key={stage}
            className={cn(
              "w-60 shrink-0 rounded-lg border bg-muted/30 p-2",
              dragOver === stage && "ring-2 ring-brand-500",
            )}
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(stage);
            }}
            onDragLeave={() => setDragOver((s) => (s === stage ? null : s))}
            onDrop={(e) => {
              e.preventDefault();
              onDrop(stage);
            }}
            data-testid={`pipeline-column-${stage}`}
          >
            <div className="mb-2 flex items-center justify-between px-1">
              <span className="inline-flex items-center gap-1.5 text-xs font-semibold">
                <span className={cn("h-2.5 w-2.5 rounded-full", STAGE_COLORS[stage]?.dot)} />
                {STAGE_LABELS[stage] ?? stage}
              </span>
              <span className="rounded-full bg-background px-2 py-0.5 text-xs font-medium">
                {columnItems.length}
              </span>
            </div>
            <div className="flex flex-col gap-2">
              {columnItems.map((item) => (
                <PipelineCard key={item.id} item={item} onMove={onMove} />
              ))}
              {columnItems.length === 0 && (
                <p className="rounded-md border border-dashed p-3 text-center text-xs text-muted-foreground">
                  Vazio
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ReviewsTab({ items, loading }: { items: PipelineItem[]; loading: boolean }) {
  if (loading) return <LoadingState variant="inline" text="Carregando revisões…" />;
  const actionable = items.filter((i) => i.stage === "review" || i.stage === "approval");
  if (actionable.length === 0) {
    return <EmptyState title="Nada em revisão" description="Nenhum artigo aguardando revisão ou aprovação." />;
  }
  return (
    <div className="flex flex-col gap-2">
      {actionable.map((item) => (
        <Card key={item.id} className="flex items-center justify-between gap-3 p-3">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{item.title || "(sem título)"}</p>
            <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <StageBadge stage={item.stage} />
              <span>{item.language}</span>
              <span>{formatRelativeTime(item.updated_at)}</span>
              <ScorePill label="SEO" value={item.seo_score} />
              <ScorePill label="EEAT" value={item.eeat_score} />
            </div>
          </div>
          {item.engine === "posts" && (
            <Button asChild size="sm" variant="outline" data-testid={`reviews-open-${item.id}`}>
              <Link to={`/admin/editorial/review/${item.id}`}>Abrir revisão</Link>
            </Button>
          )}
        </Card>
      ))}
    </div>
  );
}

function StatCard({ label, value, hint }: { label: string; value: string | number; hint?: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs font-medium text-muted-foreground">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
        {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
      </CardContent>
    </Card>
  );
}

function StatsTab({ stats, loading }: { stats?: PipelineStats; loading: boolean }) {
  if (loading) return <LoadingState variant="inline" text="Carregando estatísticas…" />;
  if (!stats) return <ErrorState title="Sem dados" message="Não foi possível carregar as estatísticas." onRetry={() => window.location.reload()} />;
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {stats.stage_counts.map((sc) => (
        <StatCard key={sc.stage} label={STAGE_LABELS[sc.stage] ?? sc.stage} value={sc.count} />
      ))}
      <StatCard label="Nota SEO média" value={stats.avg_seo_score != null ? stats.avg_seo_score.toFixed(1) : "—"} />
      <StatCard label="Nota E-E-A-T média" value={stats.avg_eeat_score != null ? stats.avg_eeat_score.toFixed(1) : "—"} />
      <StatCard label="Em tradução" value={stats.in_translation} />
      <StatCard label="Publicados (7 dias)" value={stats.published_this_week} />
      <StatCard
        label="Itens no pipeline"
        value={stats.total_items}
        hint={`${stats.pending_reviews} em revisão · ${stats.pending_approvals} em aprovação`}
      />
    </div>
  );
}

export function EditorialDashboardPage() {
  const [activeTab, setActiveTab] = useState<"pipeline" | "reviews" | "stats">("pipeline");
  const [scheduleDate, setScheduleDate] = useState("");
  const [panelItem, setPanelItem] = useState<PipelineItem | null>(null);
  const [readiness, setReadiness] = useState<PublishReadiness | null>(null);
  const [publishError, setPublishError] = useState("");
  const currentSiteId = useCurrentSiteId();
  const queryClient = useQueryClient();

  const pipelineQuery = useQuery({
    queryKey: siteQueryKey(["editorial-pipeline"], currentSiteId),
    queryFn: () => api.get<PipelineResponse>("/editorial/pipeline"),
    enabled: !!currentSiteId,
  });

  const statsQuery = useQuery({
    queryKey: siteQueryKey(["editorial-stats"], currentSiteId),
    queryFn: () => api.get<PipelineStats>("/editorial/pipeline/stats"),
    enabled: !!currentSiteId,
  });

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ["editorial-pipeline"] });
    queryClient.invalidateQueries({ queryKey: ["editorial-stats"] });
    queryClient.invalidateQueries({ queryKey: ["dashboard-editorial"] });
    queryClient.invalidateQueries({ queryKey: ["dashboard-history"] });
  };

  const moveMutation = useMutation({
    mutationFn: async ({ item, target }: { item: PipelineItem; target: MoveTarget }) => {
      if (target === "published") {
        return api.post<{ status: string }>("/publisher/publish", { post_id: item.id });
      }
      const body: Record<string, unknown> = { status: target };
      if (target === "scheduled" && scheduleDate) {
        body.scheduled_at = new Date(scheduleDate).toISOString();
      }
      return api.patch<{ status: string }>(`/posts/${item.id}/status`, body);
    },
    onSuccess: () => {
      toast.success("Movido com sucesso");
      invalidateAll();
    },
    onError: (error, { item, target }) => {
      if (error instanceof ApiError && error.status === 422 && target === "published") {
        setPanelItem(item);
        setPublishError(error.message);
        api
          .get<PublishReadiness>(`/editorial/publish-readiness/${item.id}`)
          .then(setReadiness)
          .catch(() => setReadiness(null));
        return;
      }
      toast.error(error instanceof ApiError ? error.message : "Falha ao mover");
    },
  });

  function handleMove(item: PipelineItem, target: MoveTarget) {
    if (item.engine !== "posts") return;
    moveMutation.mutate({ item, target });
  }

  const items = pipelineQuery.data?.items ?? [];

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">Painel Editorial</h1>
        <p className="text-sm text-muted-foreground">
          Acompanhe artigos do rascunho à publicação e valide cada etapa antes de publicar.
        </p>
      </div>

      <div className="flex items-center gap-2 border-b pb-2">
        <button
          className={cn(
            "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
            activeTab === "pipeline" ? "bg-brand-600 text-white" : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => setActiveTab("pipeline")}
          data-testid="editorial-tab-pipeline"
        >
          Pipeline
        </button>
        <button
          className={cn(
            "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
            activeTab === "reviews" ? "bg-brand-600 text-white" : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => setActiveTab("reviews")}
          data-testid="editorial-tab-reviews"
        >
          Revisões
        </button>
        <button
          className={cn(
            "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
            activeTab === "stats" ? "bg-brand-600 text-white" : "text-muted-foreground hover:text-foreground",
          )}
          onClick={() => setActiveTab("stats")}
          data-testid="editorial-tab-stats"
        >
          Estatísticas
        </button>
      </div>

      {activeTab === "pipeline" && (
        <>
          {pipelineQuery.isLoading ? (
            <LoadingState variant="inline" text="Carregando pipeline…" />
          ) : pipelineQuery.isError ? (
            <ErrorState
              title="Falha ao carregar o pipeline"
              message="Não foi possível buscar os dados do pipeline."
              onRetry={() => pipelineQuery.refetch()}
            />
          ) : items.length === 0 ? (
            <EmptyState title="Pipeline vazio" description="Nenhum item de conteúdo ainda." />
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <label htmlFor="schedule-date" className="text-xs text-muted-foreground">
                  Data de agendamento:
                </label>
                <Input
                  id="schedule-date"
                  type="datetime-local"
                  value={scheduleDate}
                  onChange={(e) => setScheduleDate(e.target.value)}
                  className="h-8 w-auto text-xs"
                  data-testid="editorial-schedule-date"
                />
              </div>
              <PipelineBoard items={items} onMove={handleMove} />
            </div>
          )}
        </>
      )}

      {activeTab === "reviews" && (
        <ReviewsTab items={items} loading={pipelineQuery.isLoading} />
      )}

      {activeTab === "stats" && (
        <StatsTab stats={statsQuery.data} loading={statsQuery.isLoading} />
      )}

      <Dialog
        open={!!panelItem && publishError !== ""}
        onOpenChange={(open) => {
          if (!open) {
            setPanelItem(null);
            setReadiness(null);
            setPublishError("");
          }
        }}
      >
        <DialogContent data-testid="editorial-readiness-dialog">
          <DialogHeader>
            <DialogTitle>Não foi possível publicar</DialogTitle>
            <DialogDescription>
              {publishError || "A publicação foi bloqueada por validações de qualidade."}
            </DialogDescription>
          </DialogHeader>
          {readiness ? (
            <div className="space-y-2" data-testid="editorial-readiness-checks">
              {readiness.checks.map((check) => (
                <div
                  key={check.stage}
                  className={cn(
                    "flex items-start justify-between gap-3 rounded-md border p-2 text-sm",
                    check.passed ? "border-green-200 bg-green-50" : "border-red-200 bg-red-50",
                  )}
                >
                  <div>
                    <p className="font-medium">{check.label}</p>
                    <p className="text-xs text-muted-foreground">{check.message}</p>
                  </div>
                  <span className="text-xs font-semibold">
                    {check.passed ? "OK" : "Falhou"}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Carregando validações…</p>
          )}
          <div className="flex justify-end gap-2">
            {panelItem && panelItem.engine === "posts" && (
              <Button asChild variant="outline">
                <Link to={`/admin/editorial/review/${panelItem.id}`}>Abrir revisão</Link>
              </Button>
            )}
            <Button
              onClick={() => {
                setPanelItem(null);
                setReadiness(null);
                setPublishError("");
              }}
            >
              Fechar
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
