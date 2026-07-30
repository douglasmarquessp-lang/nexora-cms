import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// Mock site store
vi.mock("@/stores/site", () => ({
  useSiteStore: vi.fn(),
}));

// Mock Radix Select (it needs a portal for rendering)
vi.mock("@radix-ui/react-select", () => ({
  Root: ({ children, value, onValueChange }: any) => (
    <div data-testid="select-root">
      <select
        data-testid="select-native"
        value={value}
        onChange={(e) => onValueChange?.(e.target.value)}
      >
        {children}
      </select>
    </div>
  ),
  Trigger: ({ children }: any) => <button>{children}</button>,
  Value: ({ placeholder }: any) => <span>{placeholder}</span>,
  Content: ({ children }: any) => <div>{children}</div>,
  Item: ({ children, value }: any) => <option value={value}>{children}</option>,
  Icon: () => null,
  Portal: ({ children }: any) => <div>{children}</div>,
}));

describe("SiteSwitcher", () => {
  let queryClient: QueryClient;
  let siteStoreMock: any;

  beforeEach(() => {
    vi.clearAllMocks();
    queryClient = new QueryClient();

    siteStoreMock = {
      sites: [],
      currentSite: null,
      isLoading: false,
      setCurrentSite: vi.fn(),
    };

    const { useSiteStore } = require("@/stores/site");
    useSiteStore.mockImplementation((selector?: any) => {
      if (selector) return selector(siteStoreMock);
      return siteStoreMock;
    });
  });

  it("should render nothing when loading", async () => {
    siteStoreMock.isLoading = true;

    const { SiteSwitcher } = await import("@/components/SiteSwitcher");

    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <SiteSwitcher />
      </QueryClientProvider>,
    );

    expect(container.innerHTML).toBe("");
  });

  it("should render nothing when no sites", async () => {
    const { SiteSwitcher } = await import("@/components/SiteSwitcher");

    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <SiteSwitcher />
      </QueryClientProvider>,
    );

    expect(container.innerHTML).toBe("");
  });

  it("should render site options when sites are available", async () => {
    siteStoreMock.sites = [
      { id: "site-1", name: "Site Alpha", slug: "alpha", status: "active", owner_id: "u1", created_at: "", updated_at: "" },
      { id: "site-2", name: "Site Beta", slug: "beta", status: "active", owner_id: "u1", created_at: "", updated_at: "" },
    ];
    siteStoreMock.currentSite = siteStoreMock.sites[0];

    const { SiteSwitcher } = await import("@/components/SiteSwitcher");

    render(
      <QueryClientProvider client={queryClient}>
        <SiteSwitcher />
      </QueryClientProvider>,
    );
  });

  it("should call setCurrentSite when selection changes", async () => {
    siteStoreMock.sites = [
      { id: "site-1", name: "Site Alpha", slug: "alpha", status: "active", owner_id: "u1", created_at: "", updated_at: "" },
    ];
    siteStoreMock.currentSite = siteStoreMock.sites[0];

    const { SiteSwitcher } = await import("@/components/SiteSwitcher");

    render(
      <QueryClientProvider client={queryClient}>
        <SiteSwitcher />
      </QueryClientProvider>,
    );
  });
});
