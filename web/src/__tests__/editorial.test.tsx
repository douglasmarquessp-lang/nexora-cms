import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { EditorialDashboardPage } from "@/pages/editorial/Dashboard";
import { EditorialReviewPage } from "@/pages/editorial/Review";
import { NO_SITE_KEY } from "@/lib/queryKeys";

const { useSiteStoreMock } = vi.hoisted(() => ({ useSiteStoreMock: vi.fn() }));
const { apiMock } = vi.hoisted(() => ({
  apiMock: {
    get: vi.fn(),
    getBlob: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    put: vi.fn(),
  },
}));

vi.mock("@/stores/site", () => ({
  useSiteStore: useSiteStoreMock,
}));

vi.mock("@/api/client", () => ({
  api: apiMock,
  ApiError: class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
  Query: undefined,
}));

function site(id: string) {
  return { id, name: `Site ${id}`, slug: id, status: "active", owner_id: "u1", created_at: "", updated_at: "" };
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

function cacheKeys(queryClient: QueryClient): unknown[][] {
  return queryClient.getQueryCache().findAll().map((q) => [...q.queryKey]);
}

const pipelineResponse = {
  items: [
    {
      id: "p1",
      title: "Artigo em revisão",
      slug: "artigo",
      stage: "review",
      engine: "posts",
      engine_id: "p1",
      language: "pt",
      seo_score: 92,
      status: "needs_review",
      updated_at: "2026-01-02T00:00:00Z",
      actionable: true,
    },
  ],
  total: 1,
  stages: [],
};

const statsResponse = {
  total_items: 1,
  avg_seo_score: 92,
  avg_eeat_score: null,
  pending_reviews: 0,
  pending_approvals: 0,
  in_translation: 0,
  published_week: 0,
  stage_counts: [],
};

const articleReview = {
  post: {
    id: "p1",
    title: "Artigo em revisão",
    slug: "artigo",
    status: "draft",
    language: "pt",
    updated_at: "2026-01-02T00:00:00Z",
  },
  review: {
    seo: 90,
    eeat: 88,
    freshness: 95,
    coverage: 85,
    naturalness: 91,
    confidence: 90,
    final: 85,
    decision: "needs_review",
    threshold: 90,
    created_at: "2026-01-02T00:00:00Z",
  },
  sources: [],
  problems: [],
  recommendations: [],
  internal_links: [],
  external_links: [],
};

function mockEditorialApi() {
  apiMock.get.mockImplementation((path: string) => {
    if (path === "/editorial/pipeline") return Promise.resolve(pipelineResponse);
    if (path === "/editorial/pipeline/stats") return Promise.resolve(statsResponse);
    if (path === "/editorial/review/p1") return Promise.resolve(articleReview);
    if (path === "/editorial/approvals") return Promise.resolve([]);
    return Promise.reject(new Error(`unexpected path: ${path}`));
  });
}

function renderDashboard(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <EditorialDashboardPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderReview(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/p1"]}>
        <Routes>
          <Route path=":id" element={<EditorialReviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("EditorialDashboardPage site-scoped queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("includes the current site id in pipeline and stats query keys", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockEditorialApi();

    renderDashboard(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-pipeline", "site-1"]);
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-stats", "site-1"]);
    });
  });

  it("registers keys under NO_SITE_KEY without executing when no site", async () => {
    currentSiteId = null;
    mockSiteStore();

    renderDashboard(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-pipeline", NO_SITE_KEY]);
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-stats", NO_SITE_KEY]);
    });
    expect(apiMock.get).not.toHaveBeenCalled();
  });

  it("renders the pipeline board and tabs", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockEditorialApi();

    renderDashboard(queryClient);

    expect(await screen.findByText("Artigo em revisão")).toBeInTheDocument();
    expect(await screen.findByText("Pipeline")).toBeInTheDocument();
    expect(await screen.findByText("Estatísticas")).toBeInTheDocument();
    expect(await screen.findByText("Abrir")).toBeInTheDocument();
  });
});

describe("EditorialReviewPage site-scoped queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("includes the current site id and post id in review query keys", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockEditorialApi();

    renderReview(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-review", "site-1", "p1"]);
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-approvals", "site-1", "p1"]);
    });
  });

  it("renders the reviewed article title and score summary", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockEditorialApi();

    renderReview(queryClient);

    expect(await screen.findByText("Artigo em revisão")).toBeInTheDocument();
    expect(await screen.findByText("needs_review")).toBeInTheDocument();
  });

  it("does not execute without a site", async () => {
    currentSiteId = null;
    mockSiteStore();

    renderReview(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["editorial-review", NO_SITE_KEY, "p1"]);
    });
    expect(apiMock.get).not.toHaveBeenCalled();
  });
});