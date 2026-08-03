import { describe, it, expect, beforeEach, vi } from "vitest";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
}));

import { queryClient } from "@/lib/queryClient";
import { resetSession } from "@/lib/sessionReset";
import { useSiteStore } from "@/stores/site";

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

function listResponse(sites: ReturnType<typeof site>[]) {
  return { sites, total: sites.length, page: 1, per_page: 20, total_pages: Math.max(1, sites.length) };
}

describe("resetSession", () => {
  beforeEach(() => {
    queryClient.clear();
    window.localStorage.clear();
    useSiteStore.setState({
      sites: [],
      currentSite: null,
      status: "idle",
      isLoading: false,
      error: null,
      attempts: 0,
    });
  });

  it("clears every cached query (health, plugins, media, workflow)", () => {
    queryClient.setQueryData(["health"], { status: "ok" });
    queryClient.setQueryData(["plugins"], { plugins: [] });
    queryClient.setQueryData(["media", "site-1", null, ""], [{ id: "m1" }]);
    queryClient.setQueryData(["folders", "site-1", null], [{ id: "f1" }]);
    queryClient.setQueryData(["workflow-dashboard", "site-1"], { total_jobs: 3 });
    queryClient.setQueryData(["workflow-jobs", "site-1"], []);
    queryClient.setQueryData(["workflow-queue", "site-1"], []);
    queryClient.setQueryData(["workflow-metrics", "site-1"], {});

    resetSession();

    expect(queryClient.getQueryCache().findAll()).toHaveLength(0);
    expect(queryClient.getQueryData(["health"])).toBeUndefined();
    expect(queryClient.getQueryData(["plugins"])).toBeUndefined();
    expect(queryClient.getQueryData(["media", "site-1", null, ""])).toBeUndefined();
    expect(queryClient.getQueryData(["workflow-jobs", "site-1"])).toBeUndefined();
  });

  it("fully resets the SiteStore and removes the persisted site id", () => {
    const s = site("site-1");
    useSiteStore.setState({
      sites: [s],
      currentSite: s,
      status: "success",
      isLoading: false,
      error: null,
      attempts: 2,
    });
    window.localStorage.setItem("current_site_id", "site-1");

    resetSession();

    const state = useSiteStore.getState();
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.status).toBe("idle");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.attempts).toBe(0);
    expect(window.localStorage.getItem("current_site_id")).toBeNull();
  });

  it("is safe to call repeatedly", () => {
    window.localStorage.setItem("current_site_id", "site-1");
    queryClient.setQueryData(["health"], { status: "ok" });

    resetSession();
    resetSession();

    expect(useSiteStore.getState().status).toBe("idle");
    expect(useSiteStore.getState().currentSite).toBeNull();
    expect(window.localStorage.getItem("current_site_id")).toBeNull();
    expect(queryClient.getQueryCache().findAll()).toHaveLength(0);
  });

  it("invalidates a fetchSites that was already in flight before the reset", async () => {
    const { api } = await import("@/api/client");

    let resolveFetch: (value: unknown) => void;
    (api.get as any).mockImplementation(
      () => new Promise((res) => { resolveFetch = res; }),
    );

    const promise = useSiteStore.getState().fetchSites();
    expect(useSiteStore.getState().status).toBe("loading");

    resetSession();
    expect(useSiteStore.getState().status).toBe("idle");

    resolveFetch!(listResponse([site("site-1")]));
    await promise;

    const state = useSiteStore.getState();
    expect(state.status).toBe("idle");
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.error).toBeNull();
    expect(window.localStorage.getItem("current_site_id")).toBeNull();
  });
});
