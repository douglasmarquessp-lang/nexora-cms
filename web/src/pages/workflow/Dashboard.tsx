import { useState } from "react";
import { api } from "@/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { cn, formatDate, formatRelativeTime } from "@/lib/utils";
import { useCurrentSiteId, siteQueryKey } from "@/lib/queryKeys";
import { Bell, CheckCheck, ExternalLink } from "lucide-react";

interface Dashboard {
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

interface WorkflowJob {
  id: string;
  title: string;
  status: string;
  current_step: string;
  progress: number;
  language: string;
  created_at: string;
}

interface QueueItem {
  id: string;
  title: string;
  status: string;
  priority: number;
  language: string;
  scheduled_for: string | null;
}

interface Notification {
  id: string;
  notification_type: string;
  title: string;
  message: string;
  severity: string;
  read: boolean;
  action_url?: string;
  created_at: string;
}

interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  unread: number;
}

interface Metrics {
  total_jobs: number;
  running_jobs: number;
  completed_jobs: number;
  failed_jobs: number;
  avg_success_rate: number;
  avg_failure_rate: number;
  queue_size: number;
  throughput_hourly: number;
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    draft: "bg-muted text-muted-foreground",
    pending: "bg-yellow-50 text-yellow-700",
    running: "bg-blue-50 text-blue-700",
    paused: "bg-purple-50 text-purple-700",
    completed: "bg-green-50 text-green-700",
    failed: "bg-red-50 text-red-700",
    cancelled: "bg-muted text-muted-foreground",
  };

  return (
    <span className={`rounded px-2 py-0.5 text-xs font-medium ${colors[status] || "bg-muted text-muted-foreground"}`}>
      {status}
    </span>
  );
}

export function WorkflowDashboardPage() {
  const [activeTab, setActiveTab] = useState("overview");
  const [actionMsg, setActionMsg] = useState("");
  const currentSiteId = useCurrentSiteId();
  const queryClient = useQueryClient();

  const { data: dash, refetch: refetchDash } = useQuery({
    queryKey: siteQueryKey(["workflow-dashboard"], currentSiteId),
    queryFn: () => api.get<Dashboard>("/workflow/dashboard"),
    enabled: !!currentSiteId,
  });

  const { data: jobs } = useQuery({
    queryKey: siteQueryKey(["workflow-jobs"], currentSiteId),
    queryFn: () => api.get<WorkflowJob[]>("/workflow", { params: { limit: "10" } }),
    enabled: !!currentSiteId,
  });

  const { data: queueData } = useQuery({
    queryKey: siteQueryKey(["workflow-queue"], currentSiteId),
    queryFn: () => api.get<QueueItem[]>("/workflow/queue", { params: { limit: "10" } }),
    enabled: !!currentSiteId,
  });

  const { data: metrics } = useQuery({
    queryKey: siteQueryKey(["workflow-metrics"], currentSiteId),
    queryFn: () => api.get<Metrics>("/workflow/metrics"),
    enabled: !!currentSiteId,
  });

  const { data: notificationsData, isLoading: notificationsLoading, isError: notificationsError, refetch: refetchNotifications } =
    useQuery({
      queryKey: siteQueryKey(["workflow-notifications"], currentSiteId),
      queryFn: () =>
        api.get<NotificationListResponse>("/workflow/notifications", { params: { limit: "50" } }),
      enabled: !!currentSiteId,
    });

  const markReadMutation = useMutation({
    mutationFn: (notificationId: string) => api.put(`/workflow/notifications/${notificationId}/read`),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["workflow-notifications"] }),
  });

  const markAllReadMutation = useMutation({
    mutationFn: () => api.post("/workflow/notifications/read-all"),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ["workflow-notifications"] }),
  });

  const actionMutation = useMutation({
    mutationFn: (action: { action: string; title?: string; job_id?: string }) =>
      api.post("/workflow/actions", action),
    onSuccess: () => {
      setActionMsg("Ação executada com sucesso!");
      refetchDash();
      setTimeout(() => setActionMsg(""), 3000);
    },
    onError: () => {
      setActionMsg("Falha ao executar ação. Verifique os logs.");
      setTimeout(() => setActionMsg(""), 3000);
    },
  });

  const tabs = ["overview", "jobs", "queue", "notifications"];

  const runningJobs = (jobs || []).filter((j) => j.status === "running");
  const notifUnread = notificationsData?.unread ?? 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Workflow Dashboard</h1>
          <p className="text-sm text-muted-foreground">Monitore e gerencie seus workflows de conteúdo</p>
        </div>
        <div className="flex gap-2">
          {tabs.map((tab) => {
            const label = tab === "notifications" ? "Notifications" : tab.charAt(0).toUpperCase() + tab.slice(1);
            return (
              <Button
                key={tab}
                variant={activeTab === tab ? "default" : "outline"}
                size="sm"
                onClick={() => setActiveTab(tab)}
              >
                {label}
                {tab === "notifications" && notifUnread > 0 && (
                  <span
                    data-testid="notifications-unread-badge"
                    className="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-brand-600 px-1 text-[10px] font-semibold text-white"
                  >
                    {notifUnread}
                  </span>
                )}
              </Button>
            );
          })}
        </div>
      </div>

      {actionMsg && (
        <div className="rounded-lg bg-brand-600 px-4 py-3 text-sm text-white">
          {actionMsg}
        </div>
      )}

      {activeTab === "overview" && (
        <>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Running Jobs" value={dash?.running_jobs ?? metrics?.running_jobs ?? 0} />
            <StatCard label="Completed" value={dash?.completed_jobs ?? 0} />
            <StatCard label="Failed" value={dash?.failed_jobs ?? 0} />
            <StatCard label="Success Rate" value={`${(dash?.success_rate ?? 0).toFixed(1)}%`} />
            <StatCard label="Queue Size" value={dash?.queue_size ?? 0} />
            <StatCard label="Pending Review" value={dash?.pending_review ?? 0} />
            <StatCard label="Scheduled" value={dash?.scheduled_publications ?? 0} />
            <StatCard label="Throughput (hr)" value={`${(dash?.throughput_hourly ?? 0).toFixed(1)}`} />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <QuickActionsCard onAction={(a) => actionMutation.mutate(a)} />
            <WorkflowVisualization jobs={jobs || []} />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <RecentActivityCard jobs={jobs || []} />
            <QueueMonitorCard items={queueData || []} />
          </div>
        </>
      )}

      {activeTab === "jobs" && (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Title</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Step</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Progress</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Language</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Created</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {(jobs || []).length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                      Nenhum job encontrado.
                    </td>
                  </tr>
                ) : (jobs || []).map((job) => (
                  <tr key={job.id} className="hover:bg-muted/50">
                    <td className="px-4 py-3 font-medium text-foreground">{job.title}</td>
                    <td className="px-4 py-3"><StatusBadge status={job.status} /></td>
                    <td className="px-4 py-3 text-muted-foreground">{job.current_step || "---"}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-24 rounded-full bg-muted">
                          <div className="h-2 rounded-full bg-brand-500 transition-all" style={{ width: `${job.progress}%` }} />
                        </div>
                        <span className="text-xs text-muted-foreground">{job.progress.toFixed(0)}%</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-muted px-2 py-0.5 text-xs uppercase">{job.language}</span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{new Date(job.created_at).toLocaleDateString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {activeTab === "queue" && (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Title</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Priority</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Language</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Scheduled</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {(queueData || []).length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                      Fila vazia.
                    </td>
                  </tr>
                ) : (queueData || []).map((item) => (
                  <tr key={item.id} className="hover:bg-muted/50">
                    <td className="px-4 py-3 font-medium text-foreground">{item.title}</td>
                    <td className="px-4 py-3"><StatusBadge status={item.status} /></td>
                    <td className="px-4 py-3">
                      <span className={cn(
                        "rounded px-2 py-0.5 text-xs font-medium",
                        item.priority <= 3 ? "bg-red-50 text-red-700" :
                        item.priority <= 6 ? "bg-yellow-50 text-yellow-700" :
                        "bg-muted text-muted-foreground",
                      )}>P{item.priority}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-muted px-2 py-0.5 text-xs uppercase">{item.language}</span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {item.scheduled_for ? new Date(item.scheduled_for).toLocaleDateString() : "Immediate"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      {activeTab === "notifications" && (
        <NotificationsPanel
          data={notificationsData}
          isLoading={notificationsLoading}
          isError={notificationsError}
          onRetry={refetchNotifications}
          onMarkRead={(id) => markReadMutation.mutate(id)}
          onMarkAllRead={() => markAllReadMutation.mutate()}
          markAllPending={markAllReadMutation.isPending}
        />
      )}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-xs font-medium text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold text-foreground">{value}</div>
      </CardContent>
    </Card>
  );
}

function QuickActionsCard({ onAction }: { onAction: (a: { action: string; title?: string }) => void }) {
  const [title, setTitle] = useState("");

  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Quick Actions</h3>
      <div className="mb-3">
        <Input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Article title..."
        />
      </div>
      <div className="flex flex-wrap gap-2">
        {[
          { action: "generate_article", label: "Generate Article (PT)" },
          { action: "generate_pt_en", label: "Generate PT + EN" },
        ].map((a) => (
          <Button key={a.action} size="sm" onClick={() => onAction({ action: a.action, title: title || undefined })}>
            {a.label}
          </Button>
        ))}
      </div>
    </Card>
  );
}

function WorkflowVisualization({ jobs }: { jobs: WorkflowJob[] }) {
  const steps = ["research", "writer", "human_writer", "editorial_engine", "seo_engine", "quality_check", "publisher", "finished"];
  const displayNames: Record<string, string> = {
    research: "Research", writer: "Writer", human_writer: "Human Writer",
    editorial_engine: "Editorial Engine", seo_engine: "SEO Engine",
    quality_check: "Quality Check", publisher: "Publisher", finished: "Finished",
  };

  const runningJobs = jobs.filter((j) => j.status === "running");

  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Workflow Pipeline</h3>
      <div className="flex flex-wrap gap-1.5">
        {steps.map((step, i) => {
          const isActive = runningJobs.some((j) => j.current_step === step);
          const isCompleted = jobs.some((j) => step === "finished" && j.status === "completed");
          return (
            <div key={step} className="flex items-center">
              <span className={cn(
                "rounded-md px-2.5 py-1 text-xs font-medium",
                isActive ? "bg-brand-600 text-white" :
                isCompleted ? "bg-green-50 text-green-700" :
                "bg-muted text-muted-foreground",
              )}>{displayNames[step] || step}</span>
              {i < steps.length - 1 && <span className="mx-1 text-muted-foreground">→</span>}
            </div>
          );
        })}
      </div>
      {runningJobs.length > 0 && <p className="mt-2 text-xs text-brand-600">{runningJobs.length} job(s) em andamento</p>}
    </Card>
  );
}

function RecentActivityCard({ jobs }: { jobs: WorkflowJob[] }) {
  const recent = jobs.slice(0, 5);
  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Recent Activity</h3>
      <div className="space-y-2">
        {recent.length === 0 ? (
          <p className="text-sm text-muted-foreground">Nenhuma atividade recente.</p>
        ) : recent.map((job) => (
          <div key={job.id} className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2">
            <div className="flex items-center gap-2">
              <StatusBadge status={job.status} />
              <span className="text-sm text-foreground truncate max-w-[200px]">{job.title}</span>
            </div>
            <span className="text-xs text-muted-foreground">{new Date(job.created_at).toLocaleString()}</span>
          </div>
        ))}
      </div>
    </Card>
  );
}

function QueueMonitorCard({ items }: { items: QueueItem[] }) {
  const pending = items.filter((i) => i.status === "pending").length;
  return (
    <Card className="p-4">
      <h3 className="mb-3 text-sm font-medium text-muted-foreground">Queue Monitor</h3>
      <div className="mb-3 flex items-center gap-4">
        <div>
          <p className="text-2xl font-bold text-foreground">{items.length}</p>
          <p className="text-xs text-muted-foreground">Total in queue</p>
        </div>
        <div>
          <p className="text-2xl font-bold text-brand-600">{pending}</p>
          <p className="text-xs text-muted-foreground">Pending</p>
        </div>
      </div>
      <div className="space-y-1">
        {items.slice(0, 4).map((item) => (
          <div key={item.id} className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-1.5">
            <span className="text-sm text-foreground truncate max-w-[180px]">{item.title}</span>
            <StatusBadge status={item.status} />
          </div>
        ))}
      </div>
    </Card>
  );
}

const NOTIFICATION_SEVERITY_COLORS: Record<string, string> = {
  info: "bg-blue-50 text-blue-700",
  warning: "bg-yellow-50 text-yellow-700",
  error: "bg-red-50 text-red-700",
  critical: "bg-red-100 text-red-800",
  success: "bg-green-50 text-green-700",
};

const NOTIFICATION_SEVERITY_LABELS: Record<string, string> = {
  info: "Info",
  warning: "Warning",
  error: "Error",
  critical: "Critical",
  success: "Success",
};

function SeverityBadge({ severity }: { severity: string }) {
  return (
    <span
      className={`rounded px-2 py-0.5 text-xs font-medium ${
        NOTIFICATION_SEVERITY_COLORS[severity] || "bg-muted text-muted-foreground"
      }`}
    >
      {NOTIFICATION_SEVERITY_LABELS[severity] || severity}
    </span>
  );
}

function safeActionUrl(actionUrl?: string): string | null {
  if (!actionUrl) return null;
  try {
    const url = new URL(actionUrl);
    if (url.protocol !== "http:" && url.protocol !== "https:") return null;
    return url.toString();
  } catch {
    return null;
  }
}

interface NotificationsPanelProps {
  data: NotificationListResponse | undefined;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  onMarkRead: (id: string) => void;
  onMarkAllRead: () => void;
  markAllPending: boolean;
}

function NotificationsPanel({
  data,
  isLoading,
  isError,
  onRetry,
  onMarkRead,
  onMarkAllRead,
  markAllPending,
}: NotificationsPanelProps) {
  if (isLoading) {
    return <LoadingState variant="inline" text="Carregando notificações..." />;
  }

  if (isError) {
    return (
      <ErrorState
        title="Não foi possível carregar as notificações"
        message="Ocorreu um erro ao buscar as notificações. Tente novamente."
        onRetry={onRetry}
      />
    );
  }

  const notifications = data?.notifications ?? [];

  if (notifications.length === 0) {
    return <EmptyState title="Nenhuma notificação" description="Você está em dia." />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {data?.total ?? notifications.length} notificação(ões), {data?.unread ?? 0} não lida(s)
        </p>
        {data && data.unread > 0 && (
          <Button
            variant="outline"
            size="sm"
            onClick={onMarkAllRead}
            disabled={markAllPending}
            data-testid="notifications-mark-all-read"
          >
            <CheckCheck className="h-4 w-4" />
            Marcar todas como lidas
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {notifications.map((notification) => (
          <Card
            key={notification.id}
            className={cn("p-4", !notification.read && "border-brand-500 bg-brand-50/50")}
            data-testid="notification-item"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <SeverityBadge severity={notification.severity} />
                  {!notification.read && (
                    <span className="h-2 w-2 rounded-full bg-brand-600" data-testid="notification-unread-dot" />
                  )}
                </div>
                <h4 className="mt-1 font-medium text-foreground">{notification.title}</h4>
                <p className="mt-1 text-sm text-muted-foreground">{notification.message}</p>
                <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{formatRelativeTime(notification.created_at)}</span>
                  <span>·</span>
                  <span>{formatDate(notification.created_at)}</span>
                  {safeActionUrl(notification.action_url) && (
                    <a
                      href={safeActionUrl(notification.action_url) ?? undefined}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="inline-flex items-center gap-1 text-brand-600 hover:underline"
                    >
                      <ExternalLink className="h-3 w-3" />
                      Ver detalhes
                    </a>
                  )}
                </div>
              </div>
              {!notification.read && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onMarkRead(notification.id)}
                  data-testid={`notifications-mark-read-${notification.id}`}
                >
                  <CheckCheck className="h-4 w-4" />
                  Marcar como lida
                </Button>
              )}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
