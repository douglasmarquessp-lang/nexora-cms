import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AdminLayout } from "@/components/AdminLayout";

const { authStoreMock } = vi.hoisted(() => ({ authStoreMock: vi.fn() }));
const { siteStoreMock } = vi.hoisted(() => ({ siteStoreMock: vi.fn() }));

vi.mock("@/stores/auth", () => ({ useAuthStore: authStoreMock }));
vi.mock("@/stores/site", () => ({ useSiteStore: siteStoreMock }));
vi.mock("@/components/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("@/components/Header", () => ({ Header: () => null }));
vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: any) => <>{children}</>,
  SheetContent: ({ children }: any) => <>{children}</>,
}));
vi.mock("@/components/ui/sonner", () => ({ Toaster: () => null }));

function mockSite(id: string) {
  return {
    id,
    name: `Site ${id}`,
    slug: id,
    status: "active",
    owner_id: "u1",
    created_at: "",
    updated_at: "",
  };
}

describe("AdminLayout site load banner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  function renderLayout(siteState: Record<string, unknown>) {
    authStoreMock.mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      checkAuth: vi.fn(),
    });
    siteStoreMock.mockReturnValue({
      sites: [],
      currentSite: null,
      status: "idle",
      isLoading: false,
      error: null,
      attempts: 0,
      fetchSites: vi.fn(),
      retrySites: vi.fn(),
      setCurrentSite: vi.fn(),
      clearCurrentSite: vi.fn(),
      ...siteState,
    });

    return render(
      <MemoryRouter initialEntries={["/admin"]}>
        <Routes>
          <Route path="/admin" element={<AdminLayout />}>
            <Route index element={<div>pagina</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
  }

  it("calls fetchSites exactly once when the user is authenticated", () => {
    const fetchSites = vi.fn();
    renderLayout({ status: "loading", isLoading: true, attempts: 1, fetchSites });

    expect(fetchSites).toHaveBeenCalledTimes(1);
  });

  it("shows a clear error banner with a working retry button when loading sites fails", () => {
    const retrySites = vi.fn();
    renderLayout({ status: "error", error: "Network error", attempts: 3, retrySites });

    expect(screen.getByTestId("site-load-banner")).toBeInTheDocument();
    expect(screen.getByText(/não foi possível carregar os sites/i)).toBeInTheDocument();
    expect(screen.queryByText("Nenhum site disponível para este usuário.")).not.toBeInTheDocument();
    expect(screen.queryByTestId("site-load-banner-empty")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /tentar novamente/i }));
    expect(retrySites).toHaveBeenCalledTimes(1);
  });

  it("shows a distinct 'no sites available' banner (not an error) when the list is empty", () => {
    renderLayout({ status: "empty", error: null, attempts: 1, sites: [], currentSite: null });

    expect(screen.getByTestId("site-load-banner-empty")).toBeInTheDocument();
    expect(screen.getByText("Nenhum site disponível para este usuário.")).toBeInTheDocument();
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByTestId("site-load-banner")).not.toBeInTheDocument();
    expect(screen.queryByText(/não foi possível carregar os sites/i)).not.toBeInTheDocument();
  });

  it("shows no banner while sites are loading or after success", () => {
    const first = renderLayout({ status: "loading", isLoading: true, attempts: 1 });
    expect(screen.queryByTestId("site-load-banner")).not.toBeInTheDocument();
    expect(screen.queryByTestId("site-load-banner-empty")).not.toBeInTheDocument();
    first.unmount();

    renderLayout({
      status: "success",
      sites: [mockSite("site-1")],
      currentSite: mockSite("site-1"),
      attempts: 1,
    });
    expect(screen.queryByTestId("site-load-banner")).not.toBeInTheDocument();
    expect(screen.queryByTestId("site-load-banner-empty")).not.toBeInTheDocument();
  });
});