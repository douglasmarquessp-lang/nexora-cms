import { useState } from "react";
import { api } from "@/api/client";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { cn } from "@/lib/utils";

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
  created_at: string;
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

  const { data: dash, refetch: refetchDash } = useQuery({
    queryKey: ["workflow-dashboard"],
    queryFn: () => api.get<Dashboard>("/workflow/dashboard"),
  });

  const { data: jobs } = useQuery({
    queryKey: ["workflow-jobs"],
    queryFn: () => api.get<WorkflowJob[]>("/workflow", { params: { limit: "10" } }),
  });

  const { data: queueData } = useQuery({
    queryKey: ["workflow-queue"],
    queryFn: () => api.get<QueueItem[]>("/workflow/queue", { params: { limit: "10" } }),
  });

  const { data: metrics } = useQuery({
    queryKey: ["workflow-metrics"],
    queryFn: () => api.get<Metrics>("/workflow/metrics"),
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Workflow Dashboard</h1>
          <p className="text-sm text-muted-foreground">Monitore e gerencie seus workflows de conteúdo</p>
        </div>
        <div className="flex gap-2">
          {tabs.map((tab) => (
            <Button
              key={tab}
              variant={activeTab === tab ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveTab(tab)}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </Button>
          ))}
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
        <div className="space-y-2">
          {(jobs || []).length === 0 ? (
            <EmptyState title="Nenhuma notificação" description="Você está em dia." />
          ) : null}
        </div>
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
