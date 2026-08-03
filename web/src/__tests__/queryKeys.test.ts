import { describe, it, expect, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { siteQueryKey, NO_SITE_KEY, useCurrentSiteId } from "@/lib/queryKeys";

const { useSiteStoreMock } = vi.hoisted(() => ({ useSiteStoreMock: vi.fn() }));

vi.mock("@/stores/site", () => ({
  useSiteStore: useSiteStoreMock,
}));

function mockCurrentSite(id: string | null) {
  useSiteStoreMock.mockImplementation((selector?: (s: unknown) => unknown) => {
    const state = {
      currentSite: id ? { id, name: `Site ${id}`, slug: id, status: "active" } : null,
    };
    return selector ? selector(state) : state;
  });
}

describe("siteQueryKey", () => {
  it("produces different keys for Site A and Site B", () => {
    const keyA = siteQueryKey(["media", null, ""], "site-a");
    const keyB = siteQueryKey(["media", null, ""], "site-b");
    expect(keyA).not.toEqual(keyB);
    expect(JSON.stringify(keyA)).not.toBe(JSON.stringify(keyB));
  });

  it("produces identical keys for the same site", () => {
    expect(siteQueryKey(["workflow-jobs"], "site-1")).toEqual(
      siteQueryKey(["workflow-jobs"], "site-1"),
    );
  });

  it("has a stable, predictable shape with the site id right after the resource name", () => {
    expect(siteQueryKey(["media", null, "test"], "site-1")).toEqual(["media", "site-1", null, "test"]);
    expect(siteQueryKey(["folders"], "site-1")).toEqual(["folders", "site-1"]);
    expect(siteQueryKey(["workflow-queue"], "site-1")).toEqual(["workflow-queue", "site-1"]);
  });

  it("uses a stable placeholder when no site id is available", () => {
    expect(siteQueryKey(["media"], null)).toEqual(["media", NO_SITE_KEY]);
    expect(siteQueryKey(["workflow-dashboard"], undefined)).toEqual(["workflow-dashboard", NO_SITE_KEY]);
  });

  it("keeps resource-name prefix invalidation working", async () => {
    const client = new QueryClient();
    client.setQueryData(siteQueryKey(["media"], "site-a"), [{ id: "m1" }]);
    client.setQueryData(siteQueryKey(["media"], "site-b"), [{ id: "m2" }]);
    const count = await client.invalidateQueries({ queryKey: ["media"] });
    expect(count).toBe(2);
  });

  it("does not share cached data between Site A and Site B", () => {
    const client = new QueryClient();
    const keyA = siteQueryKey(["media"], "site-a");
    const keyB = siteQueryKey(["media"], "site-b");
    client.setQueryData(keyA, [{ id: "from-site-a" }]);
    expect(client.getQueryData(keyB)).toBeUndefined();
    expect(client.getQueryData(keyA)).toEqual([{ id: "from-site-a" }]);
  });

  it("exposes a reactive current site id through useCurrentSiteId", () => {
    mockCurrentSite("site-1");
    const { result, rerender } = renderHook(() => useCurrentSiteId());
    expect(result.current).toBe("site-1");

    mockCurrentSite("site-2");
    rerender();
    expect(result.current).toBe("site-2");

    mockCurrentSite(null);
    rerender();
    expect(result.current).toBeNull();
  });
});