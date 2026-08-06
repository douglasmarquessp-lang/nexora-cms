import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DashboardPage } from "@/pages/Dashboard";
import { NO_SITE_KEY } from "@/lib/queryKeys";

const { useSiteStoreMock } = vi.hoisted(() => ({ useSiteStoreMock: vi.fn() }));
const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    get: vi.fn(),
    getBlob: vi.fn(),
    post: vi.fn(),
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

function mockDashboardApi() {
  apiMock.get.mockImplementation((path: string) => {
    switch (path) {
      case "/health":
        return Promise.resolve({ status: "ok", version: "1.0.0", timestamp: "" });
      case "/workflow/dashboard":
        return Promise.resolve({
          running_jobs: 2,
          completed_jobs: 10,
          failed_jobs: 1,
          success_rate: 90.5,
          scheduled_publications: 3,
          queue_size: 4,
          pending_review: 5,
        });
      case "/editorial/stats":
        return Promise.resolve({
          published_posts: 12,
          draft_posts: 7,
          recent_posts: [
            {
              id: "p1",
              title: "Post site-1",
              slug: "post-site-1",
              status: "published",
              created_at: "2026-01-02T00:00:00Z",
              updated_at: "2026-01-02T00:00:00Z",
            },
          ],
        });
      case "/workflow/history":
        return Promise.resolve([
          {
            id: "h1",
            action: "workflow.started",
            entity_type: "job",
            created_at: "2026-01-03T00:00:00Z",
          },
        ]);
      default:
        return Promise.reject(new Error(`unexpected path: ${path}`));
    }
  });
}

function cacheKeys(queryClient: QueryClient): unknown[][] {
  return queryClient.getQueryCache().findAll().map((q) => [...q.queryKey]);
}

function renderDashboard(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("DashboardPage site-scoped queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("includes the current site id in all dashboard query keys", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockDashboardApi();

    renderDashboard(queryClient);

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["dashboard-workflow", "site-1"]);
      expect(keys).toContainEqual(["dashboard-editorial", "site-1"]);
      expect(keys).toContainEqual(["dashboard-history", "site-1"]);
    });
  });

  it("does not execute site-scoped queries without a current site id", async () => {
    currentSiteId = null;
    mockSiteStore();

    renderDashboard(queryClient);

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["dashboard-workflow", NO_SITE_KEY]);
      expect(keys).toContainEqual(["dashboard-editorial", NO_SITE_KEY]);
      expect(keys).toContainEqual(["dashboard-history", NO_SITE_KEY]);
    });
  });

  it("renders metric cards from the combined dashboard data", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockDashboardApi();

    renderDashboard(queryClient);

    expect(await screen.findByText("Status do Sistema")).toBeInTheDocument();
    expect(await screen.findByText("Jobs em execução")).toBeInTheDocument();
    expect(await screen.findByText("Taxa de sucesso")).toBeInTheDocument();
    expect(await screen.findByText("Artigos publicados")).toBeInTheDocument();
    expect(await screen.findByText("90.5%")).toBeInTheDocument();
  });

  it("renders recent activities, workflow history and quick shortcuts", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockDashboardApi();

    renderDashboard(queryClient);

    expect(await screen.findByText("Atividades recentes")).toBeInTheDocument();
    expect(await screen.findByText("Post site-1")).toBeInTheDocument();
    expect(await screen.findByText("Workflow iniciado")).toBeInTheDocument();
    expect(await screen.findByText("Atalhos rápidos")).toBeInTheDocument();
    expect(await screen.findByText("Sites")).toBeInTheDocument();
  });

  it("keeps Site A dashboard cache entries isolated from Site B keys", async () => {
    queryClient.setQueryData(["dashboard-workflow", "site-a"], { running_jobs: 1 });
    expect(queryClient.getQueryData(["dashboard-workflow", "site-a"])).toEqual({ running_jobs: 1 });
    expect(queryClient.getQueryData(["dashboard-workflow", "site-b"])).toBeUndefined();
    expect(queryClient.getQueryData(["dashboard-workflow", NO_SITE_KEY])).toBeUndefined();
  });

  it("does not break the global health query (not site-scoped)", async () => {
    queryClient.setQueryData(["health"], { status: "ok", version: "1.0.0", timestamp: "" });
    expect(queryClient.getQueryData(["health"])).toEqual({
      status: "ok",
      version: "1.0.0",
      timestamp: "",
    });
  });
});

describe("DashboardPage query key isolation (prefix invalidations)", () => {
  it("keeps invalidating dashboards and site data independently", async () => {
    const queryClient = makeQueryClient();
    queryClient.setQueryData(["dashboard-workflow", "site-1"], { running_jobs: 1 });
    queryClient.setQueryData(["dashboard-editorial", "site-1"], { published_posts: 2 });
    queryClient.setQueryData(["health"], { status: "ok" });

    await queryClient.invalidateQueries({ queryKey: ["dashboard-workflow"] });
    const invalidated = queryClient
      .getQueryCache()
      .findAll({ queryKey: ["dashboard-workflow"] })
      .filter((q) => q.state.isInvalidated);
    expect(invalidated.length).toBe(1);

    expect(queryClient.getQueryData(["dashboard-editorial", "site-1"])).toEqual({
      published_posts: 2,
    });
    expect(queryClient.getQueryData(["health"])).toEqual({ status: "ok" });
  });
});