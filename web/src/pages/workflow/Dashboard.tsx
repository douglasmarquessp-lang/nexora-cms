import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/api/client";
import { useQuery, useMutation } from "@tanstack/react-query";

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

function classNames(...classes: (string | false | null | undefined)[]) {
  return classes.filter(Boolean).join(" ");
}

export function WorkflowDashboardPage() {
  const navigate = useNavigate();
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();
  const [activeTab, setActiveTab] = useState("overview");
  const [actionMsg, setActionMsg] = useState("");

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate("/admin/login");
    }
  }, [isLoading, isAuthenticated, navigate]);

  const { data: dash, refetch: refetchDash } = useQuery({
    queryKey: ["workflow-dashboard"],
    queryFn: () => api.get<Dashboard>("/workflow/dashboard"),
    enabled: isAuthenticated,
  });

  const { data: jobs } = useQuery({
    queryKey: ["workflow-jobs"],
    queryFn: () => api.get<WorkflowJob[]>("/workflow", { params: { limit: "10" } }),
    enabled: isAuthenticated,
  });

  const { data: queueData } = useQuery({
    queryKey: ["workflow-queue"],
    queryFn: () => api.get<QueueItem[]>("/workflow/queue", { params: { limit: "10" } }),
    enabled: isAuthenticated,
  });

  const { data: notifData } = useQuery({
    queryKey: ["workflow-notifications"],
    queryFn: () =>
      api.get<{ notifications: Notification[]; total: number; unread: number }>(
        "/workflow/notifications",
        { params: { limit: "5" } },
      ),
    enabled: isAuthenticated,
  });

  const { data: metrics } = useQuery({
    queryKey: ["workflow-metrics"],
    queryFn: () => api.get<Metrics>("/workflow/metrics"),
    enabled: isAuthenticated,
  });

  const actionMutation = useMutation({
    mutationFn: (action: { action: string; title?: string; job_id?: string }) =>
      api.post("/workflow/actions", action),
    onSuccess: () => {
      setActionMsg("Action executed successfully!");
      refetchDash();
      setTimeout(() => setActionMsg(""), 3000);
    },
    onError: () => {
      setActionMsg("Action failed. See logs for details.");
      setTimeout(() => setActionMsg(""), 3000);
    },
  });

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-950">
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <header className="border-b border-gray-800 bg-gray-900">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-white">Nexora CMS</h1>
            <span className="rounded bg-indigo-600 px-2 py-0.5 text-xs font-medium text-white">
              Workflow
            </span>
          </div>
          <div className="flex items-center gap-4">
            {notifData && notifData.unread > 0 && (
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-red-500 text-xs font-bold">
                {notifData.unread}
              </span>
            )}
            <button
              onClick={() => navigate("/admin/dashboard")}
              className="rounded-md px-3 py-1.5 text-sm text-gray-400 hover:text-white"
            >
              Main Dashboard
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-6 py-6">
        {actionMsg && (
          <div className="mb-4 rounded-lg bg-indigo-600 px-4 py-3 text-sm text-white">
            {actionMsg}
          </div>
        )}

        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-2xl font-semibold text-white">
            {activeTab === "overview" && "Workflow Dashboard"}
            {activeTab === "jobs" && "Workflow Jobs"}
            {activeTab === "queue" && "Publication Queue"}
            {activeTab === "notifications" && "Notifications"}
          </h2>
          <div className="flex gap-2">
            {["overview", "jobs", "queue", "notifications"].map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={classNames(
                  "rounded-md px-4 py-2 text-sm font-medium transition-colors",
                  activeTab === tab
                    ? "bg-indigo-600 text-white"
                    : "bg-gray-800 text-gray-400 hover:text-white",
                )}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {activeTab === "overview" && (
          <>
            <div className="mb-6 grid gap-4 md:grid-cols-4">
              <StatCard
                label="Running Jobs"
                value={dash?.running_jobs ?? metrics?.running_jobs ?? 0}
                color="text-blue-400"
              />
              <StatCard
                label="Completed"
                value={dash?.completed_jobs ?? 0}
                color="text-green-400"
              />
              <StatCard
                label="Failed"
                value={dash?.failed_jobs ?? 0}
                color="text-red-400"
              />
              <StatCard
                label="Success Rate"
                value={`${(dash?.success_rate ?? 0).toFixed(1)}%`}
                color="text-emerald-400"
              />
              <StatCard
                label="Queue Size"
                value={dash?.queue_size ?? 0}
                color="text-yellow-400"
              />
              <StatCard
                label="Pending Review"
                value={dash?.pending_review ?? 0}
                color="text-purple-400"
              />
              <StatCard
                label="Scheduled"
                value={dash?.scheduled_publications ?? 0}
                color="text-cyan-400"
              />
              <StatCard
                label="Throughput (hr)"
                value={`${(dash?.throughput_hourly ?? 0).toFixed(1)}`}
                color="text-orange-400"
              />
            </div>

            <div className="mb-6 grid gap-4 md:grid-cols-2">
              <QuickActionsCard onAction={(a) => actionMutation.mutate(a)} />
              <WorkflowVisualization jobs={jobs ?? []} />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <RecentActivityCard jobs={jobs ?? []} />
              <QueueMonitorCard items={queueData ?? []} />
            </div>
          </>
        )}

        {activeTab === "jobs" && (
          <div className="rounded-lg border border-gray-800 bg-gray-900">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-gray-800 text-gray-400">
                  <tr>
                    <th className="px-4 py-3 font-medium">Title</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Step</th>
                    <th className="px-4 py-3 font-medium">Progress</th>
                    <th className="px-4 py-3 font-medium">Language</th>
                    <th className="px-4 py-3 font-medium">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {(jobs ?? []).map((job) => (
                    <tr key={job.id} className="hover:bg-gray-800/50">
                      <td className="px-4 py-3 font-medium text-white">{job.title}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={job.status} />
                      </td>
                      <td className="px-4 py-3 text-gray-300">
                        {job.current_step || "---"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-24 rounded-full bg-gray-700">
                            <div
                              className="h-2 rounded-full bg-indigo-500 transition-all"
                              style={{ width: `${job.progress}%` }}
                            />
                          </div>
                          <span className="text-xs text-gray-400">
                            {job.progress.toFixed(0)}%
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className="rounded bg-gray-800 px-2 py-0.5 text-xs uppercase">
                          {job.language}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-400">
                        {new Date(job.created_at).toLocaleDateString()}
                      </td>
                    </tr>
                  ))}
                  {(jobs ?? []).length === 0 && (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                        No jobs yet. Create one using the Quick Actions.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === "queue" && (
          <div className="rounded-lg border border-gray-800 bg-gray-900">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-gray-800 text-gray-400">
                  <tr>
                    <th className="px-4 py-3 font-medium">Title</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Priority</th>
                    <th className="px-4 py-3 font-medium">Language</th>
                    <th className="px-4 py-3 font-medium">Scheduled</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-800">
                  {(queueData ?? []).map((item) => (
                    <tr key={item.id} className="hover:bg-gray-800/50">
                      <td className="px-4 py-3 font-medium text-white">{item.title}</td>
                      <td className="px-4 py-3">
                        <QueueStatusBadge status={item.status} />
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={classNames(
                            "rounded px-2 py-0.5 text-xs font-medium",
                            item.priority <= 3
                              ? "bg-red-900 text-red-300"
                              : item.priority <= 6
                                ? "bg-yellow-900 text-yellow-300"
                                : "bg-gray-800 text-gray-300",
                          )}
                        >
                          P{item.priority}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="rounded bg-gray-800 px-2 py-0.5 text-xs uppercase">
                          {item.language}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-400">
                        {item.scheduled_for
                          ? new Date(item.scheduled_for).toLocaleDateString()
                          : "Immediate"}
                      </td>
                    </tr>
                  ))}
                  {(queueData ?? []).length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                        Queue is empty.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === "notifications" && (
          <div className="space-y-2">
            {(notifData?.notifications ?? []).map((n) => (
              <div
                key={n.id}
                className={classNames(
                  "rounded-lg border p-4",
                  n.read ? "border-gray-800 bg-gray-900" : "border-indigo-800 bg-gray-800",
                )}
              >
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <SeverityDot severity={n.severity} />
                      <span className="font-medium text-white">{n.title}</span>
                      {!n.read && (
                        <span className="rounded bg-indigo-600 px-1.5 py-0.5 text-xs">
                          New
                        </span>
                      )}
                    </div>
                    {n.message && (
                      <p className="mt-1 text-sm text-gray-400">{n.message}</p>
                    )}
                  </div>
                  <span className="text-xs text-gray-500">
                    {new Date(n.created_at).toLocaleString()}
                  </span>
                </div>
              </div>
            ))}
            {(notifData?.notifications ?? []).length === 0 && (
              <div className="rounded-lg border border-gray-800 bg-gray-900 p-8 text-center text-gray-500">
                No notifications.
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

function StatCard({
  label,
  value,
  color,
}: {
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <p className="text-xs font-medium text-gray-500">{label}</p>
      <p className={`mt-1 text-2xl font-bold ${color}`}>{value}</p>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    draft: "bg-gray-800 text-gray-300",
    pending: "bg-yellow-900 text-yellow-300",
    running: "bg-blue-900 text-blue-300",
    paused: "bg-purple-900 text-purple-300",
    completed: "bg-green-900 text-green-300",
    failed: "bg-red-900 text-red-300",
    cancelled: "bg-gray-800 text-gray-400",
  };

  return (
    <span
      className={`rounded px-2 py-0.5 text-xs font-medium ${colors[status] || "bg-gray-800 text-gray-300"}`}
    >
      {status}
    </span>
  );
}

function QueueStatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    pending: "bg-yellow-900 text-yellow-300",
    running: "bg-blue-900 text-blue-300",
    paused: "bg-purple-900 text-purple-300",
    completed: "bg-green-900 text-green-300",
    failed: "bg-red-900 text-red-300",
    cancelled: "bg-gray-800 text-gray-400",
  };

  return (
    <span
      className={`rounded px-2 py-0.5 text-xs font-medium ${colors[status] || "bg-gray-800 text-gray-300"}`}
    >
      {status}
    </span>
  );
}

function SeverityDot({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    info: "bg-blue-500",
    warning: "bg-yellow-500",
    error: "bg-red-500",
    critical: "bg-red-600",
    success: "bg-green-500",
  };

  return (
    <span
      className={`inline-block h-2 w-2 rounded-full ${colors[severity] || "bg-gray-500"}`}
    />
  );
}

function QuickActionsCard({
  onAction,
}: {
  onAction: (a: { action: string; title?: string }) => void;
}) {
  const [title, setTitle] = useState("");

  const actions = [
    { action: "generate_article", label: "Generate Article (PT)", icon: "✏️" },
    { action: "generate_pt_en", label: "Generate PT + EN", icon: "🌐" },
  ];

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <h3 className="mb-3 text-sm font-medium text-gray-400">Quick Actions</h3>
      <div className="mb-3">
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Article title..."
          className="w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white placeholder-gray-500 focus:border-indigo-500 focus:outline-none"
        />
      </div>
      <div className="flex flex-wrap gap-2">
        {actions.map((a) => (
          <button
            key={a.action}
            onClick={() => onAction({ action: a.action, title: title || undefined })}
            className="flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
          >
            <span>{a.icon}</span>
            <span>{a.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function WorkflowVisualization({ jobs }: { jobs: WorkflowJob[] }) {
  const steps = [
    "research",
    "writer",
    "human_writer",
    "editorial_engine",
    "seo_engine",
    "quality_check",
    "publisher",
    "finished",
  ];
  const displayNames: Record<string, string> = {
    research: "Research",
    writer: "Writer",
    human_writer: "Human Writer",
    editorial_engine: "Editorial Engine",
    seo_engine: "SEO Engine",
    quality_check: "Quality Check",
    publisher: "Publisher",
    finished: "Finished",
  };

  const runningJobs = jobs.filter((j) => j.status === "running");

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <h3 className="mb-3 text-sm font-medium text-gray-400">Workflow Pipeline</h3>
      <div className="flex flex-wrap gap-1.5">
        {steps.map((step, i) => {
          const isActive = runningJobs.some((j) => j.current_step === step);
          const isCompleted = jobs.some(
            (j) =>
              step === "finished" && j.status === "completed",
          );
          return (
            <div key={step} className="flex items-center">
              <div
                className={classNames(
                  "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                  isActive
                    ? "bg-indigo-600 text-white"
                    : isCompleted
                      ? "bg-green-900 text-green-300"
                      : "bg-gray-800 text-gray-400",
                )}
              >
                {displayNames[step] || step}
              </div>
              {i < steps.length - 1 && (
                <span className="mx-1 text-gray-600">→</span>
              )}
            </div>
          );
        })}
      </div>
      {runningJobs.length > 0 && (
        <p className="mt-2 text-xs text-indigo-400">
          {runningJobs.length} job(s) in progress
        </p>
      )}
    </div>
  );
}

function RecentActivityCard({ jobs }: { jobs: WorkflowJob[] }) {
  const recent = jobs.slice(0, 5);

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <h3 className="mb-3 text-sm font-medium text-gray-400">Recent Activity</h3>
      <div className="space-y-2">
        {recent.map((job) => (
          <div
            key={job.id}
            className="flex items-center justify-between rounded-md bg-gray-800/50 px-3 py-2"
          >
            <div className="flex items-center gap-2">
              <StatusBadge status={job.status} />
              <span className="text-sm text-gray-300 truncate max-w-[200px]">
                {job.title}
              </span>
            </div>
            <span className="text-xs text-gray-500">
              {new Date(job.created_at).toLocaleString()}
            </span>
          </div>
        ))}
        {recent.length === 0 && (
          <p className="text-center text-sm text-gray-500">No recent activity.</p>
        )}
      </div>
    </div>
  );
}

function QueueMonitorCard({ items }: { items: QueueItem[] }) {
  const pending = items.filter((i) => i.status === "pending").length;

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
      <h3 className="mb-3 text-sm font-medium text-gray-400">Queue Monitor</h3>
      <div className="mb-3 flex items-center gap-4">
        <div>
          <p className="text-2xl font-bold text-white">{items.length}</p>
          <p className="text-xs text-gray-500">Total in queue</p>
        </div>
        <div>
          <p className="text-2xl font-bold text-yellow-400">{pending}</p>
          <p className="text-xs text-gray-500">Pending</p>
        </div>
      </div>
      <div className="space-y-1">
        {items.slice(0, 4).map((item) => (
          <div
            key={item.id}
            className="flex items-center justify-between rounded-md bg-gray-800/50 px-3 py-1.5"
          >
            <span className="text-sm text-gray-300 truncate max-w-[180px]">
              {item.title}
            </span>
            <QueueStatusBadge status={item.status} />
          </div>
        ))}
      </div>
    </div>
  );
}
