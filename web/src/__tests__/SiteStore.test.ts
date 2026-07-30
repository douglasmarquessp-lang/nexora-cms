import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock api client
vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
  },
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value; }),
    removeItem: vi.fn((key: string) => { delete store[key]; }),
    clear: vi.fn(() => { store = {}; }),
  };
})();
Object.defineProperty(window, "localStorage", { value: localStorageMock });

describe("SiteStore", () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  it("should fetch sites and set currentSite to first site", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    const mockSites = [
      { id: "site-1", name: "Site 1", slug: "site-1", status: "active", owner_id: "user-1", created_at: "", updated_at: "" },
      { id: "site-2", name: "Site 2", slug: "site-2", status: "active", owner_id: "user-1", created_at: "", updated_at: "" },
    ];

    (api.get as any).mockResolvedValue({ sites: mockSites, total: 2, page: 1, per_page: 20, total_pages: 1 });

    const store = useSiteStore.getState();
    await store.fetchSites();

    const state = useSiteStore.getState();
    expect(state.sites).toEqual(mockSites);
    expect(state.currentSite?.id).toBe("site-1");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("should restore persisted site from localStorage", async () => {
    localStorageMock.setItem("current_site_id", "site-2");

    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    const mockSites = [
      { id: "site-1", name: "Site 1", slug: "site-1", status: "active", owner_id: "user-1", created_at: "", updated_at: "" },
      { id: "site-2", name: "Site 2", slug: "site-2", status: "active", owner_id: "user-1", created_at: "", updated_at: "" },
    ];

    (api.get as any).mockResolvedValue({ sites: mockSites, total: 2, page: 1, per_page: 20, total_pages: 1 });

    const store = useSiteStore.getState();
    await store.fetchSites();

    const state = useSiteStore.getState();
    expect(state.currentSite?.id).toBe("site-2");
  });

  it("should set current site and persist to localStorage", async () => {
    const { useSiteStore } = await import("@/stores/site");

    const site = { id: "site-3", name: "Site 3", slug: "site-3", status: "active", owner_id: "user-1", created_at: "", updated_at: "" };

    const store = useSiteStore.getState();
    store.setCurrentSite(site);

    expect(useSiteStore.getState().currentSite?.id).toBe("site-3");
    expect(localStorageMock.setItem).toHaveBeenCalledWith("current_site_id", "site-3");
  });

  it("should clear current site", async () => {
    const { useSiteStore } = await import("@/stores/site");

    const site = { id: "site-1", name: "Site 1", slug: "site-1", status: "active", owner_id: "user-1", created_at: "", updated_at: "" };

    const store = useSiteStore.getState();
    store.setCurrentSite(site);
    store.clearCurrentSite();

    expect(useSiteStore.getState().currentSite).toBeNull();
    expect(localStorageMock.removeItem).toHaveBeenCalledWith("current_site_id");
  });

  it("should handle fetch errors gracefully", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockRejectedValue(new Error("Network error"));

    const store = useSiteStore.getState();
    await store.fetchSites();

    const state = useSiteStore.getState();
    expect(state.error).toBe("Network error");
    expect(state.isLoading).toBe(false);
    expect(state.sites).toEqual([]);
  });

  it("should handle empty sites list", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue({ sites: [], total: 0, page: 1, per_page: 20, total_pages: 0 });

    const store = useSiteStore.getState();
    await store.fetchSites();

    const state = useSiteStore.getState();
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
  });
});
