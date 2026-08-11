import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { WorkflowReviewPage } from "@/pages/workflow/Review";
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

const reviewResponse = {
  job: {
    id: "j1",
    title: "5 Simple AI Tools",
    status: "completed",
    current_step: "finished",
    progress: 100,
    language: "en",
    review_status: "generated",
    revision: 1,
    created_at: "2026-01-02T00:00:00Z",
  },
  article: {
    title: "5 Simple AI Tools",
    slug: "5-simple-ai-tools",
    content: "# 5 Simple AI Tools\n\nThis is the generated article body.",
    keyword: "ai tools",
    meta_title: "5 Simple AI Tools",
    meta_description: "Discover five simple AI tools.",
    featured_image_url: "https://images.example.com/photo.jpg",
    featured_image_alt: "AI tools on a desk",
    featured_image_credit: {
      photographer: "Jane Doe",
      photographer_url: "https://pexels.com/@janedoe",
      source_url: "https://pexels.com/photo/123",
    },
    author_name: "AIWorkSimple",
    language: "en",
    excerpt: "Discover five simple AI tools.",
    categories: [],
    tags: ["ai", "tools"],
    word_count: 42,
    reading_time: 1,
  },
  seo: {
    score: 82.45,
    min_score: 70,
    passes: true,
    title: 100,
    meta: 75,
    headings: 90,
    keyword: 100,
    readability: 64,
    internal_links: 100,
    external_links: 100,
    eeat: 66.5,
    images: 100,
    keyword_density: 1.2,
    word_count: 42,
    issues: [],
  },
  version: 1,
  sources: [
    {
      id: "s1",
      url: "https://blog.google/ai/",
      title: "Google AI Blog",
      domain: "blog.google",
      reliability_score: 90,
      reliability_label: "verified",
      is_verified: true,
    },
  ],
};

function mockReviewApi() {
  apiMock.get.mockImplementation((path: string) => {
    if (path === "/workflow/j1/review") return Promise.resolve(reviewResponse);
    return Promise.reject(new Error(`unexpected path: ${path}`));
  });
}

function renderReview(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/j1"]}>
        <Routes>
          <Route path=":id" element={<WorkflowReviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkflowReviewPage site-scoped queries", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    vi.clearAllMocks();
    currentSiteId = null;
    apiMock.post.mockResolvedValue({});
    apiMock.put.mockResolvedValue({});
    queryClient = makeQueryClient();
  });

  it("includes the current site id and job id in the review query key", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockReviewApi();

    renderReview(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["workflow-review", "site-1", "j1"]);
    });
  });

  it("registers the key under NO_SITE_KEY without executing when no site", async () => {
    currentSiteId = null;
    mockSiteStore();

    renderReview(queryClient);

    await waitFor(() => {
      expect(cacheKeys(queryClient)).toContainEqual(["workflow-review", NO_SITE_KEY, "j1"]);
    });
    expect(apiMock.get).not.toHaveBeenCalled();
  });

  it("renders the article title, status badge, content and SEO breakdown", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockReviewApi();

    renderReview(queryClient);

    expect(await screen.findByText("5 Simple AI Tools")).toBeInTheDocument();
    expect(await screen.findByText("Aguardando revisão")).toBeInTheDocument();
    expect(await screen.findByText(/generated article body/)).toBeInTheDocument();
    expect(await screen.findByTestId("review-seo-passes")).toHaveTextContent("82.45");
    expect(await screen.findByTestId("review-featured-image")).toHaveAttribute("src", "https://images.example.com/photo.jpg");
    expect(await screen.findByText("Jane Doe")).toBeInTheDocument();
    expect(await screen.findByText("Google AI Blog")).toBeInTheDocument();
    expect(await screen.findByText("Densidade da keyword: 1.20%")).toBeInTheDocument();
  });

  it("honestly shows when no featured image is available", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    const withoutImage = JSON.parse(JSON.stringify(reviewResponse));
    withoutImage.article.featured_image_url = "";
    withoutImage.article.featured_image_credit = undefined;
    apiMock.get.mockImplementation((path: string) => {
      if (path === "/workflow/j1/review") return Promise.resolve(withoutImage);
      return Promise.reject(new Error(`unexpected path: ${path}`));
    });

    renderReview(queryClient);

    expect(await screen.findByTestId("review-no-image")).toBeInTheDocument();
    expect(screen.queryByTestId("review-featured-image")).not.toBeInTheDocument();
  });

  it("saves draft meta via PUT and approves via POST with overrides", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockReviewApi();
    const user = userEvent.setup();

    renderReview(queryClient);

    await screen.findByTestId("review-keyword");
    await user.type(screen.getByTestId("review-keyword"), "ai tools for teams");
    await user.click(screen.getByTestId("review-save-draft"));
    await waitFor(() => {
      expect(apiMock.put).toHaveBeenCalledWith("/workflow/j1/review/draft", { keyword: "ai tools for teams" });
    });

    await user.click(screen.getByTestId("review-approve-start"));
    await user.click(screen.getByTestId("review-approve-confirm"));
    await waitFor(() => {
      expect(apiMock.post).toHaveBeenCalledWith("/workflow/j1/review/approve", {
        keyword: "ai tools for teams",
      });
    });
  });

  it("requires a rejection reason before submitting", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockReviewApi();
    const user = userEvent.setup();

    renderReview(queryClient);

    const rejectButton = await screen.findByTestId("review-reject-submit");
    expect(rejectButton).toBeDisabled();

    await user.type(screen.getByTestId("review-reject-reason"), "Faltam dados comparativos");
    expect(rejectButton).toBeEnabled();
    await user.click(rejectButton);

    await waitFor(() => {
      expect(apiMock.post).toHaveBeenCalledWith("/workflow/j1/review/reject", {
        reason: "Faltam dados comparativos",
      });
    });
  });

  it("regenerates into a new revision via POST", async () => {
    currentSiteId = "site-1";
    mockSiteStore();
    mockReviewApi();
    const user = userEvent.setup();

    renderReview(queryClient);

    await user.click(await screen.findByTestId("review-regenerate-button"));
    expect(await screen.findByText("Regenerar (nova revisão 2)")).toBeInTheDocument();
    await user.click(screen.getByTestId("review-regenerate-confirm"));

    await waitFor(() => {
      expect(apiMock.post).toHaveBeenCalledWith("/workflow/j1/review/regenerate");
    });
  });
});
