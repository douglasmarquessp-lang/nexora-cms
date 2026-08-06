import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { WorkflowDashboardPage } from "@/pages/workflow/Dashboard";
import { NO_SITE_KEY } from "@/lib/queryKeys";

const { useSiteStoreMock } = vi.hoisted(() => ({ useSiteStoreMock: vi.fn() }));
const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    get: vi.fn(),
    getBlob: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/stores/site", () => ({
  useSiteStore: useSiteStoreMock,
}));

vi.mock("@/api/client", () => ({
  api: apiMock,
}));

function site(id: string) {
  return {
    id,
    name: `Site ${id}`,
    slug: id,
    status: "active",
    owner_id: "user-1",
    created_at: "",
    updated_at: "",
  };
}

function notification(id: string, overrides: Partial<{
  title: string;
  message: string;
  severity: string;
  read: boolean;
  action_url?: string;
}> = {}) {
  return {
    id,
    notification_type: "workflow.completed",
    title: overrides.title ?? `Notification ${id}`,
    message: overrides.message ?? `Message for ${id}`,
    severity: overrides.severity ?? "info",
    read: overrides.read ?? false,
    action_url: overrides.action_url,
    created_at: "2026-01-01T10:00:00Z",
  };
}

function notificationsList(notifs: ReturnType<typeof notification>[], unread?: number) {
  return {
    notifications: notifs,
    total: notifs.length,
    unread: unread ?? notifs.filter((n) => !n.read).length,
  };
}

let currentSiteId: string | null;

function mockSiteStore() {
  useSiteStoreMock.mockImplementation((selector?: (s: unknown) => unknown) => {
    const state = { currentSite: currentSiteId ? site(currentSiteId) : null };
    return selector ? selector(state) : state;
  });
}

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function mockWorkflowApi() {
  apiMock.get.mockImplementation((path: string) => {
    switch (path) {
      case "/workflow/dashboard":
        return Promise.resolve({});
      case "/workflow":
        return Promise.resolve([]);
      case "/workflow/queue":
        return Promise.resolve([]);
      case "/workflow/metrics":
        return Promise.resolve({});
      case "/workflow/notifications":
        return Promise.resolve(notificationsList([
          notification("n1", { title: "Job concluído", severity: "success", read: false }),
          notification("n2", { title: "Erro no job", severity: "error", read: true }),
        ]));
      default:
        return Promise.reject(new Error(`unexpected path: ${path}`));
    }
  });
}

function cacheKeys(queryClient: QueryClient): unknown[][] {
  return queryClient.getQueryCache().findAll().map((q) => [...q.queryKey]);
}

async function switchToNotificationsTab() {
  fireEvent.click(screen.getByRole("button", { name: /notifications/i }));
}

describe("Workflow notifications tab", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("uses a site-scoped query key for notifications", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["workflow-notifications", "site-1"]);
    });
  });

  it("does not execute the notifications query without a current site id", async () => {
    currentSiteId = null;
    mockSiteStore();

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    expect(apiMock.get).not.toHaveBeenCalled();

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["workflow-notifications", NO_SITE_KEY]);
    });
  });

  it("renders notifications with title, message and severity badge", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    expect(await screen.findByText("Job concluído")).toBeInTheDocument();
    expect(screen.getByText("Erro no job")).toBeInTheDocument();
    expect(screen.getByText("Success")).toBeInTheDocument();
    expect(screen.getByText("Error")).toBeInTheDocument();
  });

  it("shows the unread count badge on the notifications tab", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("notifications-unread-badge")).toHaveTextContent("1");
    });
  });

  it("distinguishes unread notifications from read ones", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    await waitFor(() => {
      expect(screen.getAllByTestId("notification-unread-dot")).toHaveLength(1);
    });
    expect(screen.getByRole("button", { name: /marcar como lida/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /marcar todas como lidas/i })).toBeInTheDocument();
  });

  it("shows the empty state when there are no notifications", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/notifications") {
        return Promise.resolve(notificationsList([]));
      }
      return Promise.resolve(path === "/workflow" || path === "/workflow/queue" ? [] : {});
    });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    expect(await screen.findByText("Nenhuma notificação")).toBeInTheDocument();
  });

  it("shows the error state and retries on failure", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/notifications") {
        return Promise.reject(new Error("network down"));
      }
      return Promise.resolve(path === "/workflow" || path === "/workflow/queue" ? [] : {});
    });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    expect(await screen.findByText(/não foi possível carregar as notificações/i)).toBeInTheDocument();

    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/notifications") {
        return Promise.resolve(notificationsList([notification("n1", { title: "Recuperado" })]));
      }
      return Promise.resolve(path === "/workflow" || path === "/workflow/queue" ? [] : {});
    });

    fireEvent.click(screen.getByRole("button", { name: /tentar novamente/i }));

    expect(await screen.findByText("Recuperado")).toBeInTheDocument();
  });

  it("calls PUT to mark a single notification read and invalidates the prefix", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();
    apiMock.put.mockResolvedValue({ status: "ok" });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    fireEvent.click(
      await screen.findByTestId("notifications-mark-read-n1"),
    );

    await waitFor(() => {
      expect(apiMock.put).toHaveBeenCalledWith("/workflow/notifications/n1/read");
      expect(apiMock.get).toHaveBeenCalledWith("/workflow/notifications", {
        params: { limit: "50" },
      });
    });
  });

  it("calls POST to mark all notifications read and invalidates the prefix", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockWorkflowApi();
    apiMock.post.mockResolvedValue({ status: "ok" });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    fireEvent.click(await screen.findByTestId("notifications-mark-all-read"));

    await waitFor(() => {
      expect(apiMock.post).toHaveBeenCalledWith("/workflow/notifications/read-all");
    });
  });

  it("does not render a link for javascript: action URLs", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/notifications") {
        return Promise.resolve(notificationsList([
          notification("n1", { title: "Perigoso", action_url: "javascript:alert(1)" }),
        ]));
      }
      return Promise.resolve(path === "/workflow" || path === "/workflow/queue" ? [] : {});
    });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    expect(await screen.findByText("Perigoso")).toBeInTheDocument();
    expect(screen.queryByText("Ver detalhes")).not.toBeInTheDocument();
    expect(document.querySelector("a[href*='javascript:']")).not.toBeInTheDocument();
  });

  it("renders a safe link for https action URLs", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/notifications") {
        return Promise.resolve(notificationsList([
          notification("n1", { title: "Com link", action_url: "https://example.com/jobs/1" }),
        ]));
      }
      return Promise.resolve(path === "/workflow" || path === "/workflow/queue" ? [] : {});
    });

    render(
      <QueryClientProvider client={queryClient}>
        <WorkflowDashboardPage />
      </QueryClientProvider>,
    );

    await switchToNotificationsTab();

    const link = await screen.findByText("Ver detalhes");
    expect(link).toHaveAttribute("href", "https://example.com/jobs/1");
    expect(link).toHaveAttribute("rel", "noreferrer noopener");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
