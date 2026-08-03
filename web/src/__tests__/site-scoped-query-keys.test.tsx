import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MediaLibraryPage } from "@/pages/MediaLibrary";
import { WorkflowDashboardPage } from "@/pages/workflow/Dashboard";
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

function makeMediaItem(id: string, siteId: string) {
  return {
    id,
    site_id: siteId,
    folder_id: null,
    filename: `${id}.pdf`,
    original_name: `media-${siteId}`,
    mime_type: "application/pdf",
    extension: "pdf",
    size: 1234,
    width: null,
    height: null,
    duration: 0,
    hash: `hash-${id}`,
    alt_text: `media-${siteId}`,
    caption: "",
    storage_provider: "local",
    storage_key: `${id}.pdf`,
    created_by: "user-1",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    deleted_at: null,
  };
}

// Media API returns the current site's data, mirroring what the backend does
// with the X-Site-ID header.
function mockMediaApi() {
  apiMock.get.mockImplementation((path: string) => {
    if (path.startsWith("/media/folders")) {
      return Promise.resolve({ folders: [], total: 0 });
    }
    if (path.startsWith("/media")) {
      const id = currentSiteId || "nosite";
      return Promise.resolve({
        media: [makeMediaItem(`m-${id}`, id)],
        total: 1,
        page: 1,
        per_page: 50,
      });
    }
    return Promise.reject(new Error(`unexpected path: ${path}`));
  });
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
      default:
        return Promise.reject(new Error(`unexpected path: ${path}`));
    }
  });
}

function cacheKeys(queryClient: QueryClient): unknown[][] {
  return queryClient.getQueryCache().findAll().map((q) => [...q.queryKey]);
}

describe("MediaLibrary page site-scoped queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("includes the current site id in the media and folders query keys", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockMediaApi();

    render(
      <QueryClientProvider client={queryClient}>
        <MediaLibraryPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["media", "site-1", null, ""]);
      expect(keys).toContainEqual(["folders", "site-1", null]);
    });
  });

  it("does not execute site-scoped queries without a current site id", async () => {
    currentSiteId = null;
    mockSiteStore();

    render(
      <QueryClientProvider client={queryClient}>
        <MediaLibraryPage />
      </QueryClientProvider>,
    );

    expect(apiMock.get).not.toHaveBeenCalled();

    await waitFor(() => {
      const keys = cacheKeys(queryClient);
      expect(keys).toContainEqual(["media", NO_SITE_KEY, null, ""]);
      expect(keys).toContainEqual(["folders", NO_SITE_KEY, null]);
    });
  });

  it("shows Site A data, then replaces it with Site B data after switching", async () => {
    currentSiteId = "site-a";
    mockSiteStore();
    mockMediaApi();

    const view = render(
      <QueryClientProvider client={queryClient}>
        <MediaLibraryPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("media-site-a")).toBeInTheDocument();

    currentSiteId = "site-b";
    view.rerender(
      <QueryClientProvider client={queryClient}>
        <MediaLibraryPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.queryByText("media-site-a")).not.toBeInTheDocument();
    });
    expect(await screen.findByText("media-site-b")).toBeInTheDocument();
  });
});

describe("WorkflowDashboard page queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    queryClient = makeQueryClient();
  });

  it("uses the current site id in all workflow query keys", async () => {
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
      expect(keys).toContainEqual(["workflow-dashboard", "site-1"]);
      expect(keys).toContainEqual(["workflow-jobs", "site-1"]);
      expect(keys).toContainEqual(["workflow-queue", "site-1"]);
      expect(keys).toContainEqual(["workflow-metrics", "site-1"]);
    });
  });

  it("does not execute workflow queries without a current site id", async () => {
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
      expect(keys).toContainEqual(["workflow-dashboard", NO_SITE_KEY]);
      expect(keys).toContainEqual(["workflow-jobs", NO_SITE_KEY]);
      expect(keys).toContainEqual(["workflow-queue", NO_SITE_KEY]);
      expect(keys).toContainEqual(["workflow-metrics", NO_SITE_KEY]);
    });
  });

  it("keeps Site A workflow cache entries isolated from Site B keys", async () => {
    queryClient.setQueryData(["workflow-jobs", "site-a"], [{ id: "job-from-site-a" }]);
    expect(queryClient.getQueryData(["workflow-jobs", "site-a"])).toEqual([{ id: "job-from-site-a" }]);
    expect(queryClient.getQueryData(["workflow-jobs", "site-b"])).toBeUndefined();
    expect(queryClient.getQueryData(["workflow-jobs", NO_SITE_KEY])).toBeUndefined();
  });
});