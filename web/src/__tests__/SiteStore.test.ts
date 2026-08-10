import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock api client
vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
  },
}));

// Mock the site-selection config. The array is mutated per-test so existing
// tests keep the default behavior (empty whitelist = no restriction) while
// dedicated tests exercise the AIWorkSimple-only restriction.
const siteSelectionMock = vi.hoisted(() => ({
  ADMIN_ALLOWED_SITE_IDS: [] as string[],
}));
vi.mock("@/config/siteSelection", () => siteSelectionMock);

const AIWORK_SIMPLE_ID = "a64d7d72-b97f-4f31-96fd-8aeb15f6184c";

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

function site(id: string, name = id) {
  return { id, name, slug: id, status: "active", owner_id: "user-1", created_at: "", updated_at: "" };
}

function listResponse(sites: ReturnType<typeof site>[]) {
  return { sites, total: sites.length, page: 1, per_page: 20, total_pages: Math.max(1, sites.length) };
}

async function resetStore() {
  const { useSiteStore } = await import("@/stores/site");
  useSiteStore.setState({
    sites: [],
    currentSite: null,
    status: "idle",
    isLoading: false,
    error: null,
    attempts: 0,
  });
}

describe("SiteStore", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    siteSelectionMock.ADMIN_ALLOWED_SITE_IDS.length = 0;
    localStorageMock.clear();
    await resetStore();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts in the idle state, distinct from loading/success/empty/error", async () => {
    const { useSiteStore } = await import("@/stores/site");

    const state = useSiteStore.getState();
    expect(state.status).toBe("idle");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.attempts).toBe(0);
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
  });

  it("loads sites successfully and selects the first site", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.sites).toHaveLength(2);
    expect(state.currentSite?.id).toBe("site-1");
    expect(state.status).toBe("success");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
    expect(localStorageMock.getItem("current_site_id")).toBe("site-1");
  });

  it("restores the persisted current site when it still exists", async () => {
    localStorageMock.setItem("current_site_id", "site-2");

    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.currentSite?.id).toBe("site-2");
    expect(state.status).toBe("success");
  });

  it("falls back to the first site when the persisted site no longer exists and fixes the stored id", async () => {
    localStorageMock.setItem("current_site_id", "ghost-site");

    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.currentSite?.id).toBe("site-1");
    expect(localStorageMock.getItem("current_site_id")).toBe("site-1");
  });

  it("exposes a loading state while sites are being fetched", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    let resolveFetch: (value: unknown) => void;
    (api.get as any).mockImplementation(
      () => new Promise((res) => { resolveFetch = res; }),
    );

    const promise = useSiteStore.getState().fetchSites();

    expect(useSiteStore.getState().status).toBe("loading");
    expect(useSiteStore.getState().isLoading).toBe(true);

    resolveFetch!(listResponse([site("site-1")]));
    await promise;

    expect(useSiteStore.getState().status).toBe("success");
    expect(useSiteStore.getState().currentSite?.id).toBe("site-1");
  });

  it("marks the store as error (not empty) when all attempts fail", async () => {
    vi.useFakeTimers();

    const { useSiteStore, MAX_SITE_FETCH_ATTEMPTS } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockRejectedValue(new Error("Network error"));

    const promise = useSiteStore.getState().fetchSites();
    await vi.runAllTimersAsync();
    await promise;

    const state = useSiteStore.getState();
    expect(state.status).toBe("error");
    expect(state.error).toBe("Network error");
    expect(api.get).toHaveBeenCalledTimes(MAX_SITE_FETCH_ATTEMPTS);
    expect(state.attempts).toBe(MAX_SITE_FETCH_ATTEMPTS);
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.isLoading).toBe(false);
  });

  it("recovers via retry after an initial failure (API down then back)", async () => {
    vi.useFakeTimers();

    const { useSiteStore, MAX_SITE_FETCH_ATTEMPTS } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockRejectedValue(new Error("Network error"));

    const initial = useSiteStore.getState().fetchSites();
    await vi.runAllTimersAsync();
    await initial;

    expect(useSiteStore.getState().status).toBe("error");
    expect(api.get).toHaveBeenCalledTimes(MAX_SITE_FETCH_ATTEMPTS);

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));

    await useSiteStore.getState().retrySites();

    const state = useSiteStore.getState();
    expect(state.status).toBe("success");
    expect(state.error).toBeNull();
    expect(state.isLoading).toBe(false);
    expect(state.currentSite?.id).toBe("site-1");
    expect(localStorageMock.getItem("current_site_id")).toBe("site-1");
  });

  it("does not loop forever while retrying", async () => {
    vi.useFakeTimers();

    const { useSiteStore, MAX_SITE_FETCH_ATTEMPTS } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockRejectedValue(new Error("Network error"));

    const promise = useSiteStore.getState().fetchSites();
    await vi.runAllTimersAsync();
    await promise;

    expect(api.get).toHaveBeenCalledTimes(MAX_SITE_FETCH_ATTEMPTS);
    expect(useSiteStore.getState().status).toBe("error");
  });

  it("treats an empty list as 'no sites available', distinct from an error", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([]));

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.status).toBe("empty");
    expect(state.error).toBeNull();
    expect(state.isLoading).toBe(false);
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();

    const invalid = localStorageMock.getItem("current_site_id");
    expect(invalid).toBeNull();
  });

  it("does not clear previously loaded sites/currentSite when a refresh fails", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));
    await useSiteStore.getState().fetchSites();
    expect(useSiteStore.getState().currentSite?.id).toBe("site-1");

    vi.useFakeTimers();
    (api.get as any).mockRejectedValue(new Error("Network error"));

    const promise = useSiteStore.getState().retrySites();
    await vi.runAllTimersAsync();
    await promise;

    const state = useSiteStore.getState();
    expect(state.status).toBe("error");
    expect(state.sites).toHaveLength(2);
    expect(state.currentSite?.id).toBe("site-1");
  });

  it("should set current site and persist to localStorage", async () => {
    const { useSiteStore } = await import("@/stores/site");

    const s = site("site-3", "Site 3");
    useSiteStore.getState().setCurrentSite(s);

    expect(useSiteStore.getState().currentSite?.id).toBe("site-3");
    expect(localStorageMock.setItem).toHaveBeenCalledWith("current_site_id", "site-3");
  });

  it("should clear current site", async () => {
    const { useSiteStore } = await import("@/stores/site");

    useSiteStore.getState().setCurrentSite(site("site-1", "Site 1"));
    useSiteStore.getState().clearCurrentSite();

    expect(useSiteStore.getState().currentSite).toBeNull();
    expect(localStorageMock.removeItem).toHaveBeenCalledWith("current_site_id");
  });

  it("reset clears all state and removes the persisted site id", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    (api.get as any).mockResolvedValue(listResponse([site("site-1"), site("site-2")]));
    await useSiteStore.getState().fetchSites();
    expect(useSiteStore.getState().status).toBe("success");
    expect(localStorageMock.getItem("current_site_id")).toBe("site-1");

    useSiteStore.getState().reset();

    const state = useSiteStore.getState();
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.status).toBe("idle");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.attempts).toBe(0);
    expect(localStorageMock.getItem("current_site_id")).toBeNull();
  });

  it("reset is safe to call repeatedly", async () => {
    const { useSiteStore } = await import("@/stores/site");

    localStorageMock.setItem("current_site_id", "site-1");
    useSiteStore.getState().reset();
    useSiteStore.getState().reset();

    expect(useSiteStore.getState().status).toBe("idle");
    expect(useSiteStore.getState().currentSite).toBeNull();
    expect(useSiteStore.getState().sites).toEqual([]);
    expect(localStorageMock.getItem("current_site_id")).toBeNull();
  });

  it("ignores a fetchSites started before reset when it resolves afterwards", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    let resolveFetch: (value: unknown) => void;
    (api.get as any).mockImplementation(
      () => new Promise((res) => { resolveFetch = res; }),
    );

    const promise = useSiteStore.getState().fetchSites();
    expect(useSiteStore.getState().status).toBe("loading");

    useSiteStore.getState().reset();
    expect(useSiteStore.getState().status).toBe("idle");

    resolveFetch!(listResponse([site("site-1")]));
    await promise;

    const state = useSiteStore.getState();
    expect(state.status).toBe("idle");
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.error).toBeNull();
    expect(localStorageMock.getItem("current_site_id")).toBeNull();
  });

  it("lets a new session start clean: a stale fetch can never overwrite the new session", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    let resolveA: (value: unknown) => void;
    let resolveB: (value: unknown) => void;
    (api.get as any)
      .mockImplementationOnce(() => new Promise((res) => { resolveA = res; }))
      .mockImplementationOnce(() => new Promise((res) => { resolveB = res; }));

    const fetchA = useSiteStore.getState().fetchSites();
    useSiteStore.getState().reset();
    const fetchB = useSiteStore.getState().fetchSites();

    resolveA!(listResponse([site("site-a")]));
    await fetchA;
    expect(useSiteStore.getState().currentSite).toBeNull();

    resolveB!(listResponse([site("site-b")]));
    await fetchB;

    const state = useSiteStore.getState();
    expect(state.currentSite?.id).toBe("site-b");
    expect(state.sites).toHaveLength(1);
    expect(state.status).toBe("success");
    expect(localStorageMock.getItem("current_site_id")).toBe("site-b");
  });

  it("keeps only the Admin allowed sites when the whitelist is set", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    siteSelectionMock.ADMIN_ALLOWED_SITE_IDS.push(AIWORK_SIMPLE_ID);
    (api.get as any).mockResolvedValue(
      listResponse([site("site-other"), site(AIWORK_SIMPLE_ID, "AIWorkSimple")])
    );

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.sites).toHaveLength(1);
    expect(state.sites[0]?.id).toBe(AIWORK_SIMPLE_ID);
    expect(state.currentSite?.id).toBe(AIWORK_SIMPLE_ID);
    expect(state.status).toBe("success");
  });

  it("never selects a non-allowed site, even when it is the first returned", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    siteSelectionMock.ADMIN_ALLOWED_SITE_IDS.push(AIWORK_SIMPLE_ID);
    (api.get as any).mockResolvedValue(
      listResponse([site("site-other"), site(AIWORK_SIMPLE_ID, "AIWorkSimple")])
    );

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.sites.find((s) => s.id === "site-other")).toBeUndefined();
    expect(state.currentSite?.id).toBe(AIWORK_SIMPLE_ID);
    expect(localStorageMock.getItem("current_site_id")).toBe(AIWORK_SIMPLE_ID);
  });

  it("drops a persisted site that is no longer allowed and restores the first allowed one", async () => {
    const { useSiteStore } = await import("@/stores/site");
    const { api } = await import("@/api/client");

    siteSelectionMock.ADMIN_ALLOWED_SITE_IDS.push(AIWORK_SIMPLE_ID);
    localStorageMock.setItem("current_site_id", "site-other");
    (api.get as any).mockResolvedValue(
      listResponse([site("site-other"), site(AIWORK_SIMPLE_ID, "AIWorkSimple")])
    );

    await useSiteStore.getState().fetchSites();

    const state = useSiteStore.getState();
    expect(state.currentSite?.id).toBe(AIWORK_SIMPLE_ID);
    expect(state.sites).toHaveLength(1);
    expect(localStorageMock.getItem("current_site_id")).toBe(AIWORK_SIMPLE_ID);
  });
});