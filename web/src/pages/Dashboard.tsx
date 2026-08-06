import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { useCurrentSiteId, siteQueryKey } from "@/lib/queryKeys";
import { formatRelativeTime } from "@/lib/utils";
import {
  Activity,
  Archive,
  CheckCircle2,
  Clock,
  FileText,
  Globe,
  GitBranch,
  Image,
  Inbox,
  Puzzle,
  Server,
  XCircle,
} from "lucide-react";

interface WorkflowDashboard {
  total_jobs: number;
  running_jobs: number;
  completed_jobs: number;
  failed_jobs: number;
  paused_jobs: number;
  queue_size: number;
  stalled_queue: number;
  pending_review: number;
  scheduled_publications: number;
  recent_publications: number;
  avg_execution_ms: number;
  success_rate: number;
  failure_rate: number;
  throughput_hourly: number;
  worker_utilization: number;
}

interface EditorialStats {
  total_posts: number;
  published_posts: number;
  draft_posts: number;
  scheduled_posts: number;
  archived_posts: number;
  total_media: number;
  total_categories: number;
  total_tags: number;
  total_tasks: number;
  pending_tasks: number;
  pending_approvals: number;
  recent_posts: PostSummary[];
}

interface PostSummary {
  id: string;
  title: string;
  slug: string;
  status: string;
  excerpt?: string;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

interface HistoryEntry {
  id: string;
  action: string;
  entity_type: string;
  previous_status?: string;
  new_status?: string;
  error_message?: string;
  created_at: string;
}

interface ActivityItem {
  id: string;
  kind: "post" | "workflow";
  title: string;
  label: string;
  error: boolean;
  created_at: string;
}

function StatCard({
  label,
  value,
  icon,
  loading,
}: {
  label: string;
  value: string | number;
  icon?: React.ReactNode;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
        {icon && <span className="text-muted-foreground">{icon}</span>}
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-7 w-24" />
        ) : (
          <div className="text-2xl font-bold text-foreground">{value}</div>
        )}
      </CardContent>
    </Card>
  );
}

const WORKFLOW_ACTION_LABELS: Record<string, string> = {
  "workflow.started": "Workflow iniciado",
  "workflow.paused": "Workflow pausado",
  "workflow.resumed": "Workflow retomado",
  "workflow.cancelled": "Workflow cancelado",
  "workflow.retry": "Etapa repetida",
  "workflow.completed": "Workflow concluído",
};

function workflowActionLabel(action: string): string {
  return WORKFLOW_ACTION_LABELS[action] || action;
}

function postStatusLabel(status: string): string {
  if (status === "published") return "Publicado";
  if (status === "draft") return "Rascunho";
  if (status === "scheduled") return "Agendado";
  return status;
}

function buildActivities(recentPosts: PostSummary[], history: HistoryEntry[]): ActivityItem[] {
  const posts: ActivityItem[] = (recentPosts || []).map((p) => ({
    id: `post-${p.id}`,
    kind: "post",
    title: p.title,
    label: postStatusLabel(p.status),
    error: false,
    created_at: p.published_at || p.created_at,
  }));

  const events: ActivityItem[] = (history || []).map((e) => ({
    id: `hist-${e.id}`,
    kind: "workflow",
    title: workflowActionLabel(e.action),
    label: e.error_message ? "Falha" : "Processamento",
    error: !!e.error_message,
    created_at: e.created_at,
  }));

  return [...posts, ...events]
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 10);
}

function StatusDot({ error }: { error: boolean }) {
  return (
    <span
      className={`inline-block h-2 w-2 shrink-0 rounded-full ${
        error ? "bg-red-500" : "bg-brand-600"
      }`}
      aria-hidden="true"
    />
  );
}

function ActivitiesPanel({
  activities,
  loading,
  error,
  onRetry,
}: {
  activities: ActivityItem[];
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}) {
  if (loading) {
    return (
      <Card className="p-4">
        <h3 className="mb-3 text-sm font-medium text-muted-foreground">Atividades recentes</h3>
        <LoadingState variant="inline" text="Carregando atividades..." />
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="p-4">
        <h3 className="mb-3 text-sm font-medium text-muted-foreground">Atividades recentes</h3>
        <ErrorState
          title="Não foi possível carregar as atividades"
          message="Ocorreu um erro ao buscar as atividades recentes. Tente novamente."
          onRetry={onRetry}
        />
      </Card>
    );
  }

  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Atividades recentes</h3>
      <div className="space-y-2">
        {activities.length === 0 ? (
          <EmptyState title="Nenhuma atividade" description="Nenhuma atividade recente no momento." />
        ) : (
          activities.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2"
            >
              <div className="flex min-w-0 items-center gap-2">
                <StatusDot error={item.error} />
                <span className="truncate text-sm text-foreground">{item.title}</span>
                <span
                  className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide ${
                    item.error
                      ? "bg-red-50 text-red-700"
                      : item.kind === "post"
                        ? "bg-green-50 text-green-700"
                        : "bg-blue-50 text-blue-700"
                  }`}
                >
                  {item.label}
                </span>
              </div>
              <span className="ml-2 shrink-0 text-xs text-muted-foreground">
                {formatRelativeTime(item.created_at)}
              </span>
            </div>
          ))
        )}
      </div>
    </Card>
  );
}

function QuickAccessCard() {
  const shortcuts = [
    { label: "Workflow", path: "/admin/workflow", icon: <GitBranch className="h-4 w-4" />, soon: false },
    { label: "Media Library", path: "/admin/media", icon: <Image className="h-4 w-4" />, soon: false },
    { label: "Plugins", path: "/admin/plugins", icon: <Puzzle className="h-4 w-4" />, soon: false },
    { label: "Sites", path: "/admin/sites", icon: <Globe className="h-4 w-4" />, soon: true },
  ];

  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Atalhos rápidos</h3>
      <div className="flex flex-wrap gap-2">
        {shortcuts.map((s) =>
          s.soon ? (
            <span
              key={s.label}
              title="Em breve"
              className="inline-flex cursor-not-allowed items-center gap-2 rounded-md border border-input bg-muted/50 px-3 py-1.5 text-sm font-medium text-muted-foreground/50"
            >
              {s.icon}
              {s.label}
              <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-muted-foreground/50">
                Em breve
              </span>
            </span>
          ) : (
            <Button key={s.label} variant="outline" size="sm" asChild>
              <Link to={s.path}>
                {s.icon}
                {s.label}
              </Link>
            </Button>
          ),
        )}
      </div>
    </Card>
  );
}

export function DashboardPage() {
  const currentSiteId = useCurrentSiteId();

  const { data: health } = useQuery({
    queryKey: ["health"],
    queryFn: () => api.get<{ status: string; version: string; timestamp: string }>("/health"),
  });

  const {
    data: dash,
    isLoading: dashLoading,
    isError: dashError,
    refetch: refetchDash,
  } = useQuery({
    queryKey: siteQueryKey(["dashboard-workflow"], currentSiteId),
    queryFn: () => api.get<WorkflowDashboard>("/workflow/dashboard"),
    enabled: !!currentSiteId,
  });

  const {
    data: stats,
    isLoading: statsLoading,
    isError: statsError,
    refetch: refetchStats,
  } = useQuery({
    queryKey: siteQueryKey(["dashboard-editorial"], currentSiteId),
    queryFn: () => api.get<EditorialStats>("/editorial/stats"),
    enabled: !!currentSiteId,
  });

  const {
    data: history,
    isLoading: historyLoading,
    isError: historyError,
    refetch: refetchHistory,
  } = useQuery({
    queryKey: siteQueryKey(["dashboard-history"], currentSiteId),
    queryFn: () => api.get<HistoryEntry[]>("/workflow/history", { params: { limit: "10" } }),
    enabled: !!currentSiteId,
  });

  const siteQueriesLoading = dashLoading || statsLoading;
  const siteQueriesError = dashError || statsError;
  const retryAll = () => {
    refetchDash();
    refetchStats();
    refetchHistory();
  };

  const activities = buildActivities(stats?.recent_posts || [], history || []);
  const successRate = dash ? `${(dash.success_rate ?? 0).toFixed(1)}%` : "---";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">Visão geral do sistema</p>
      </div>

      {siteQueriesError && (
        <ErrorState
          title="Não foi possível carregar os dados"
          message="Ocorreu um erro ao buscar os dados do dashboard. Tente novamente."
          onRetry={retryAll}
        />
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <StatCard
          label="Status do Sistema"
          value={health?.status ?? "---"}
          icon={<Server className="h-4 w-4" />}
        />
        <StatCard
          label="Jobs em execução"
          value={dash?.running_jobs ?? 0}
          icon={<Activity className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Jobs concluídos"
          value={dash?.completed_jobs ?? 0}
          icon={<CheckCircle2 className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Jobs com erro"
          value={dash?.failed_jobs ?? 0}
          icon={<XCircle className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Taxa de sucesso"
          value={successRate}
          icon={<Activity className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Jobs agendados"
          value={dash?.scheduled_publications ?? 0}
          icon={<Clock className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Itens na fila"
          value={dash?.queue_size ?? 0}
          icon={<Inbox className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Artigos publicados"
          value={stats?.published_posts ?? 0}
          icon={<FileText className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Artigos em rascunho"
          value={stats?.draft_posts ?? 0}
          icon={<Archive className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
        <StatCard
          label="Conteúdo pendente"
          value={dash?.pending_review ?? 0}
          icon={<Clock className="h-4 w-4" />}
          loading={siteQueriesLoading}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <ActivitiesPanel
          activities={activities}
          loading={historyLoading}
          error={historyError}
          onRetry={refetchHistory}
        />
        <QuickAccessCard />
      </div>
    </div>
  );
}
