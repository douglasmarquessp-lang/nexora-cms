import { describe, it, expect, beforeEach, vi } from "vitest";
import { queryClient } from "@/lib/queryClient";

// Mock site store
const { siteStoreMock } = vi.hoisted(() => ({
  siteStoreMock: {
    currentSite: { id: "test-site-id" },
    reset: vi.fn(() => {
      window.localStorage?.removeItem?.("current_site_id");
    }),
  },
}));

vi.mock("@/stores/site", () => ({
  useSiteStore: {
    getState: () => siteStoreMock,
  },
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {
    access_token: "test-access-token",
    refresh_token: "test-refresh-token",
  };
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value; }),
    removeItem: vi.fn((key: string) => { delete store[key]; }),
    clear: vi.fn(() => { store = {}; }),
  };
})();
Object.defineProperty(window, "localStorage", { value: localStorageMock });

// Mock fetch globally
const mockFetch = vi.fn();
globalThis.fetch = mockFetch;

// Mock window.location
Object.defineProperty(window, "location", {
  value: { origin: "http://localhost:3000", href: "http://localhost:3000/admin/dashboard" },
  writable: true,
});

describe("API Client", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    queryClient.clear();
    localStorageMock.setItem("access_token", "test-access-token");
    localStorageMock.setItem("refresh_token", "test-refresh-token");
  });

  it("should include Authorization header with Bearer token", async () => {
    const { api } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: "test" }),
    });

    await api.get("/test");

    const callUrl = mockFetch.mock.calls[0][0];
    const callHeaders = mockFetch.mock.calls[0][1].headers;

    expect(callHeaders["Authorization"]).toBe("Bearer test-access-token");
    expect(callUrl).toContain("/api/v1/test");
  });

  it("should include X-Site-ID header from site store", async () => {
    const { api } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });

    await api.get("/test");

    const callHeaders = mockFetch.mock.calls[0][1].headers;
    expect(callHeaders["X-Site-ID"]).toBe("test-site-id");
  });

  it("should set Content-Type application/json by default", async () => {
    const { api } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });

    await api.post("/test", { foo: "bar" });

    const callHeaders = mockFetch.mock.calls[0][1].headers;
    expect(callHeaders["Content-Type"]).toBe("application/json");
  });

  it("should omit Content-Type for FormData", async () => {
    const { api } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });

    const formData = new FormData();
    formData.append("file", "test");

    await api.post("/upload", formData, { formData: true });

    const callHeaders = mockFetch.mock.calls[0][1].headers;
    expect(callHeaders["Content-Type"]).toBeUndefined();
  });

  it("should attempt refresh on 401 and retry", async () => {
    const { api } = await import("@/api/client");

    // First call returns 401
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: { code: "UNAUTHORIZED" } }),
    });

    // Refresh endpoint succeeds
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({
        access_token: "new-access-token",
        refresh_token: "new-refresh-token",
      }),
    });

    // Retry original request succeeds
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ data: "success" }),
    });

    const result = await api.get("/test");

    expect(result).toEqual({ data: "success" });
    expect(localStorageMock.getItem("access_token")).toBe("new-access-token");
    expect(localStorageMock.getItem("refresh_token")).toBe("new-refresh-token");
  });

  it("should redirect to login with redirect param when refresh fails", async () => {
    const { api } = await import("@/api/client");
    const originalHref = window.location.href;

    Object.defineProperty(window, "location", {
      value: {
        origin: "http://localhost:3000",
        href: "http://localhost:3000/admin/dashboard?page=2",
        pathname: "/admin/dashboard",
        search: "?page=2",
      },
      writable: true,
    });

    // First call returns 401
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: { code: "UNAUTHORIZED" } }),
    });

    // Refresh fails
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: { code: "INVALID_TOKEN" } }),
    });

    await expect(api.get("/test")).rejects.toThrow("Sessão expirada");
    expect(window.location.href).toBe("/admin/login?redirect=%2Fadmin%2Fdashboard%3Fpage%3D2");

    Object.defineProperty(window, "location", {
      value: { href: originalHref },
      writable: true,
    });
  });

  it("should return blob when blob option is set", async () => {
    const { api } = await import("@/api/client");

    const fakeBlob = new Blob(["image-bytes"], { type: "image/jpeg" });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      blob: () => Promise.resolve(fakeBlob),
    });

    const result = await api.getBlob("/media/123/file");

    expect(result).toBe(fakeBlob);
    const callUrl = mockFetch.mock.calls[0][0];
    expect(callUrl).toContain("/api/v1/media/123/file");
  });

  it("forceLogout clears the query cache, resets the site store and removes current_site_id", async () => {
    const { api } = await import("@/api/client");
    const originalHref = window.location.href;

    Object.defineProperty(window, "location", {
      value: {
        origin: "http://localhost:3000",
        href: "http://localhost:3000/admin/media",
        pathname: "/admin/media",
        search: "",
      },
      writable: true,
    });

    localStorageMock.setItem("current_site_id", "test-site-id");
    queryClient.setQueryData(["media", "test-site-id", null, ""], [{ id: "m1" }]);
    queryClient.setQueryData(["health"], { status: "ok" });

    // First call returns 401, refresh also fails -> forceLogout
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: { code: "UNAUTHORIZED" } }),
    });
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: { code: "INVALID_TOKEN" } }),
    });

    await expect(api.get("/test")).rejects.toThrow("Sessão expirada");

    expect(siteStoreMock.reset).toHaveBeenCalled();
    expect(queryClient.getQueryCache().findAll()).toHaveLength(0);
    expect(queryClient.getQueryData(["health"])).toBeUndefined();
    expect(queryClient.getQueryData(["media", "test-site-id", null, ""])).toBeUndefined();
    expect(localStorageMock.getItem("current_site_id")).toBeNull();
    expect(localStorageMock.getItem("access_token")).toBeNull();

    Object.defineProperty(window, "location", {
      value: { href: originalHref },
      writable: true,
    });
  });

  it("should throw ApiError with code and message", async () => {
    const { api, ApiError } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: () => Promise.resolve({ error: { code: "INVALID_INPUT", message: "Invalid email" } }),
    });

    try {
      await api.get("/test");
      expect.unreachable("Should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(400);
      expect((err as ApiError).code).toBe("INVALID_INPUT");
      expect((err as ApiError).message).toBe("Invalid email");
    }
  });

  it("should add query params for GET requests", async () => {
    const { api } = await import("@/api/client");

    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    });

    await api.get("/search", { params: { q: "test", page: "1" } });

    const callUrl = mockFetch.mock.calls[0][0];
    expect(callUrl).toContain("q=test");
    expect(callUrl).toContain("page=1");
  });
});
