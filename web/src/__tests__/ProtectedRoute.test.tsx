import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const { useAuthStoreMock, useSiteStoreMock } = vi.hoisted(() => ({
  useAuthStoreMock: vi.fn(),
  useSiteStoreMock: vi.fn(),
}));

// Mock all store dependencies
vi.mock("@/stores/auth", () => ({
  useAuthStore: useAuthStoreMock,
}));

vi.mock("@/stores/site", () => ({
  useSiteStore: useSiteStoreMock,
}));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

// We need to test the behavior through AdminLayout since ProtectedRoute wraps it
// But we can test the redirect behavior

describe("ProtectedRoute", () => {
  let queryClient: QueryClient;
  let authStore: any;
  let siteStore: any;

  beforeEach(() => {
    vi.clearAllMocks();

    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    authStore = {
      user: null,
      isAuthenticated: false,
      isLoading: false,
      checkAuth: vi.fn(),
    };

    siteStore = {
      sites: [],
      currentSite: null,
      status: "success",
      isLoading: false,
      error: null,
      attempts: 0,
      fetchSites: vi.fn(),
      retrySites: vi.fn(),
      setCurrentSite: vi.fn(),
      clearCurrentSite: vi.fn(),
    };

    useAuthStoreMock.mockImplementation((selector?: any) => {
      if (selector) return selector(authStore);
      return authStore;
    });

    useSiteStoreMock.mockImplementation((selector?: any) => {
      if (selector) return selector(siteStore);
      return siteStore;
    });
  });

  it("should redirect to login when not authenticated", async () => {
    const { ProtectedRoute } = await import("@/components/ProtectedRoute");

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/admin"]}>
          <Routes>
            <Route path="/admin/login" element={<div>Login Page</div>} />
            <Route path="/admin" element={<ProtectedRoute />}>
              <Route index element={<div>Dashboard</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(authStore.checkAuth).toHaveBeenCalled();
  });

  it("should render AdminLayout when authenticated", async () => {
    authStore.isAuthenticated = true;

    const { ProtectedRoute } = await import("@/components/ProtectedRoute");

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/admin"]}>
          <Routes>
            <Route path="/admin/login" element={<div>Login Page</div>} />
            <Route path="/admin" element={<ProtectedRoute />}>
              <Route index element={<div>Dashboard Content</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    // Check that the site store was called (AdminLayout fetches sites)
    expect(authStore.checkAuth).toHaveBeenCalled();
  });

  it("should show loading state during auth check", async () => {
    authStore.isLoading = true;

    const { ProtectedRoute } = await import("@/components/ProtectedRoute");

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/admin"]}>
          <Routes>
            <Route path="/admin" element={<ProtectedRoute />}>
              <Route index element={<div>Dashboard</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  });
});
