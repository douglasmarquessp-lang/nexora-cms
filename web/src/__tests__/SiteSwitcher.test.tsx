import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SiteSwitcher } from "@/components/SiteSwitcher";

const { useSiteStoreMock, selectBridge } = vi.hoisted(() => ({
  useSiteStoreMock: vi.fn(),
  selectBridge: { onValueChange: (_v: string) => {} },
}));

// Mock site store
vi.mock("@/stores/site", () => ({
  useSiteStore: useSiteStoreMock,
}));

// Mock Radix Select (it needs a portal for rendering)
vi.mock("@radix-ui/react-select", () => ({
  Root: ({ children, onValueChange }: any) => {
    selectBridge.onValueChange = onValueChange;
    return <div data-testid="select-root">{children}</div>;
  },
  Group: ({ children }: any) => <>{children}</>,
  Trigger: ({ children }: any) => (
    <button type="button" data-testid="select-trigger">
      {children}
    </button>
  ),
  Value: ({ placeholder }: any) => <span>{placeholder}</span>,
  Icon: () => null,
  Content: ({ children }: any) => <div data-testid="select-content">{children}</div>,
  Portal: ({ children }: any) => <>{children}</>,
  Viewport: ({ children }: any) => <>{children}</>,
  ScrollUpButton: () => null,
  ScrollDownButton: () => null,
  Label: ({ children }: any) => <>{children}</>,
  Item: ({ children, value }: any) => (
    <div
      role="option"
      data-testid={`option-${value}`}
      onClick={() => selectBridge.onValueChange(value)}
    >
      {children}
    </div>
  ),
  ItemIndicator: ({ children }: any) => <>{children}</>,
  ItemText: ({ children }: any) => <>{children}</>,
  Separator: () => null,
}));

function mockSite(id: string, name = id) {
  return { id, name, slug: id, status: "active", owner_id: "u1", created_at: "", updated_at: "" };
}

describe("SiteSwitcher", () => {
  let queryClient: QueryClient;
  let siteStoreMock: any;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient();

    siteStoreMock = {
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
    };

    useSiteStoreMock.mockImplementation((selector?: any) => {
      if (selector) return selector(siteStoreMock);
      return siteStoreMock;
    });
  });

  function renderSwitcher() {
    return render(
      <QueryClientProvider client={queryClient}>
        <SiteSwitcher />
      </QueryClientProvider>,
    );
  }

  it("shows a loading skeleton while sites are being fetched", async () => {
    siteStoreMock.status = "loading";
    siteStoreMock.isLoading = true;

    renderSwitcher();

    expect(screen.getByTestId("site-switcher-skeleton")).toBeInTheDocument();
  });

  it("shows a loading skeleton while idle (before the first fetch), never a false 'no sites' message", async () => {
    siteStoreMock.status = "idle";

    renderSwitcher();

    expect(screen.getByTestId("site-switcher-skeleton")).toBeInTheDocument();
    expect(screen.queryByText("Nenhum site disponível")).not.toBeInTheDocument();
    expect(screen.queryByTestId("site-switcher-empty")).not.toBeInTheDocument();
    expect(screen.queryByTestId("site-switcher-error")).not.toBeInTheDocument();
  });

  it("keeps the selector functional when a refresh fails but sites were previously loaded", async () => {
    siteStoreMock.status = "error";
    siteStoreMock.error = "Network error";
    siteStoreMock.sites = [mockSite("site-1", "Site Alpha")];
    siteStoreMock.currentSite = siteStoreMock.sites[0];

    renderSwitcher();

    expect(screen.getByTestId("select-root")).toBeInTheDocument();
    expect(screen.queryByTestId("site-switcher-error")).not.toBeInTheDocument();
    expect(screen.queryByText("Nenhum site disponível")).not.toBeInTheDocument();
  });

  it("shows a clear 'no sites' message when the list is empty (distinct from error)", async () => {
    siteStoreMock.status = "empty";

    renderSwitcher();

    expect(screen.getByTestId("site-switcher-empty")).toBeInTheDocument();
    expect(screen.getByText("Nenhum site disponível")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("does NOT show 'no sites' when the load failed - shows error + retry instead", async () => {
    siteStoreMock.status = "error";
    siteStoreMock.error = "Network error";

    renderSwitcher();

    expect(screen.getByTestId("site-switcher-error")).toBeInTheDocument();
    expect(screen.getByText(/não foi possível carregar os sites/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /tentar/i })).toBeInTheDocument();
    expect(screen.queryByText("Nenhum site disponível")).not.toBeInTheDocument();
  });

  it("calls retrySites when the retry button is clicked", async () => {
    siteStoreMock.status = "error";
    siteStoreMock.error = "Network error";

    renderSwitcher();

    fireEvent.click(screen.getByRole("button", { name: /tentar/i }));
    expect(siteStoreMock.retrySites).toHaveBeenCalledTimes(1);
  });

  it("renders the site selector when sites are loaded successfully", async () => {
    siteStoreMock.status = "success";
    siteStoreMock.sites = [mockSite("site-1", "Site Alpha"), mockSite("site-2", "Site Beta")];
    siteStoreMock.currentSite = siteStoreMock.sites[0];

    renderSwitcher();

    expect(screen.getByTestId("select-root")).toBeInTheDocument();
  });

  it("calls setCurrentSite when a different site is selected", async () => {
    siteStoreMock.status = "success";
    siteStoreMock.sites = [mockSite("site-1", "Site Alpha"), mockSite("site-2", "Site Beta")];
    siteStoreMock.currentSite = siteStoreMock.sites[0];

    renderSwitcher();

    fireEvent.click(screen.getByTestId("option-site-2"));
    expect(siteStoreMock.setCurrentSite).toHaveBeenCalledWith(siteStoreMock.sites[1]);
  });
});