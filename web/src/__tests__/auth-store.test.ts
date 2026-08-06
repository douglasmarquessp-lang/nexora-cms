import { describe, it, expect, beforeEach, vi } from "vitest";
import { queryClient } from "@/lib/queryClient";
import { useSiteStore } from "@/stores/site";
import { useAuthStore } from "@/stores/auth";

// Mock api client
vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
  ApiError: class ApiError extends Error {
    constructor(
      public status: number,
      public code: string,
      message: string,
    ) {
      super(message);
      this.name = "ApiError";
    }
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

describe("AuthStore", () => {
  beforeEach(async () => {
    localStorageMock.clear();
    vi.clearAllMocks();
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: true,
    });
    useSiteStore.setState({
      sites: [],
      currentSite: null,
      status: "idle",
      isLoading: false,
      error: null,
      attempts: 0,
    });
    queryClient.clear();
  });

  it("should initialize with unauthenticated state", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const state = useAuthStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
    expect(state.isLoading).toBe(true);
  });

  it("should login and store tokens", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    const mockUser = { id: "user-1", email: "test@test.com", name: "Test", role: "admin" };

    (api.post as any).mockResolvedValue({
      access_token: "access-token-123",
      refresh_token: "refresh-token-456",
      token_type: "Bearer",
      expires_in: 3600,
      user: mockUser,
    });

    const store = useAuthStore.getState();
    await store.login("test@test.com", "password");

    expect(localStorageMock.setItem).toHaveBeenCalledWith("access_token", "access-token-123");
    expect(localStorageMock.setItem).toHaveBeenCalledWith("refresh_token", "refresh-token-456");
    expect(useAuthStore.getState().user).toEqual(mockUser);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
  });

  it("should return mfa_required without storing tokens", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    (api.post as any).mockResolvedValue({
      status: "mfa_required",
      message: "MFA code required",
    });

    const store = useAuthStore.getState();
    const result = await store.login("test@test.com", "password");

    expect(result.status).toBe("mfa_required");
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
    expect(localStorageMock.setItem).not.toHaveBeenCalled();
  });

  it("should reject and not store tokens when credentials are invalid", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api, ApiError } = await import("@/api/client");

    (api.post as any).mockRejectedValue(
      new ApiError(401, "INVALID_CREDENTIALS", "invalid email or password"),
    );

    const store = useAuthStore.getState();
    await expect(store.login("wrong@test.com", "wrong-password")).rejects.toThrow(
      "invalid email or password",
    );

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
    expect(localStorageMock.setItem).not.toHaveBeenCalled();
  });

  it("should reject and keep session untouched on connection error", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    (api.post as any).mockRejectedValue(new TypeError("Failed to fetch"));

    const store = useAuthStore.getState();
    await expect(store.login("test@test.com", "password")).rejects.toThrow(
      "Failed to fetch",
    );

    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
    expect(localStorageMock.setItem).not.toHaveBeenCalled();
  });

  it("should login with MFA code and store tokens", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    const mockUser = { id: "user-1", email: "test@test.com", name: "Test", role: "admin" };

    (api.post as any).mockResolvedValue({
      access_token: "access-token-mfa",
      refresh_token: "refresh-token-mfa",
      token_type: "Bearer",
      expires_in: 3600,
      user: mockUser,
    });

    const store = useAuthStore.getState();
    const result = await store.login("test@test.com", "password", "123456");

    expect(api.post).toHaveBeenCalledWith(
      "/auth/login",
      expect.objectContaining({ email: "test@test.com", password: "password", mfa_code: "123456" }),
    );
    expect(result.status).toBe("ok");
    expect(localStorageMock.setItem).toHaveBeenCalledWith("access_token", "access-token-mfa");
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
  });

  it("should logout and clear tokens", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    (api.post as any).mockResolvedValue({});

    // Set logged in state
    useAuthStore.setState({
      user: { id: "user-1", email: "test@test.com", name: "Test", role: "admin" },
      isAuthenticated: true,
    });

    localStorageMock.setItem("access_token", "test-token");

    await useAuthStore.getState().logout();

    expect(api.post).toHaveBeenCalledWith("/auth/logout");
    expect(localStorageMock.removeItem).toHaveBeenCalledWith("access_token");
    expect(localStorageMock.removeItem).toHaveBeenCalledWith("refresh_token");
    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it("logout removes current_site_id and fully resets the SiteStore", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    (api.post as any).mockResolvedValue({});

    const siteA = site("site-1");
    useSiteStore.setState({
      sites: [siteA],
      currentSite: siteA,
      status: "success",
      isLoading: false,
      error: null,
      attempts: 2,
    });
    localStorageMock.setItem("access_token", "test-token");
    localStorageMock.setItem("refresh_token", "refresh-token");
    localStorageMock.setItem("current_site_id", "site-1");

    await useAuthStore.getState().logout();

    expect(localStorageMock.getItem("current_site_id")).toBeNull();
    const state = useSiteStore.getState();
    expect(state.sites).toEqual([]);
    expect(state.currentSite).toBeNull();
    expect(state.status).toBe("idle");
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.attempts).toBe(0);
  });

  it("logout clears the React Query cache (health, plugins, media, workflow)", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    (api.post as any).mockResolvedValue({});
    localStorageMock.setItem("access_token", "test-token");

    queryClient.setQueryData(["health"], { status: "ok" });
    queryClient.setQueryData(["plugins"], { plugins: [] });
    queryClient.setQueryData(["media", "site-1", null, ""], [{ id: "m1" }]);
    queryClient.setQueryData(["folders", "site-1", null], [{ id: "f1" }]);
    queryClient.setQueryData(["workflow-dashboard", "site-1"], { total_jobs: 1 });
    queryClient.setQueryData(["workflow-jobs", "site-1"], []);
    queryClient.setQueryData(["workflow-queue", "site-1"], []);
    queryClient.setQueryData(["workflow-metrics", "site-1"], {});

    await useAuthStore.getState().logout();

    expect(queryClient.getQueryCache().findAll()).toHaveLength(0);
    expect(queryClient.getQueryData(["health"])).toBeUndefined();
    expect(queryClient.getQueryData(["plugins"])).toBeUndefined();
    expect(queryClient.getQueryData(["media", "site-1", null, ""])).toBeUndefined();
    expect(queryClient.getQueryData(["workflow-jobs", "site-1"])).toBeUndefined();
  });

  it("logout still clears everything when the /auth/logout endpoint fails", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api, ApiError } = await import("@/api/client");

    (api.post as any).mockRejectedValue(new ApiError(500, "INTERNAL", "logout failed"));

    const siteA = site("site-1");
    useSiteStore.setState({
      sites: [siteA],
      currentSite: siteA,
      status: "success",
      isLoading: false,
      error: null,
      attempts: 2,
    });
    localStorageMock.setItem("access_token", "test-token");
    localStorageMock.setItem("refresh_token", "refresh-token");
    localStorageMock.setItem("current_site_id", "site-1");
    queryClient.setQueryData(["media", "site-1", null, ""], [{ id: "m1" }]);

    await useAuthStore.getState().logout();

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(localStorageMock.getItem("access_token")).toBeNull();
    expect(localStorageMock.getItem("refresh_token")).toBeNull();
    expect(localStorageMock.getItem("current_site_id")).toBeNull();
    expect(useSiteStore.getState().status).toBe("idle");
    expect(useSiteStore.getState().sites).toEqual([]);
    expect(useSiteStore.getState().currentSite).toBeNull();
    expect(queryClient.getQueryCache().findAll()).toHaveLength(0);
  });

  it("should check auth successfully", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    localStorageMock.setItem("access_token", "test-token");

    const mockUser = { id: "user-1", email: "test@test.com", name: "Test", role: "admin" };

    (api.get as any).mockResolvedValue(mockUser);

    const store = useAuthStore.getState();
    await store.checkAuth();

    expect(useAuthStore.getState().user).toEqual(mockUser);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it("should handle checkAuth with no token", async () => {
    const { useAuthStore } = await import("@/stores/auth");

    useAuthStore.setState({ isLoading: true });

    const store = useAuthStore.getState();
    await store.checkAuth();

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().isLoading).toBe(false);
  });

  it("should handle checkAuth failure", async () => {
    const { useAuthStore } = await import("@/stores/auth");
    const { api } = await import("@/api/client");

    localStorageMock.setItem("access_token", "expired-token");

    (api.get as any).mockRejectedValue(new Error("Token expired"));

    const store = useAuthStore.getState();
    await store.checkAuth();

    expect(useAuthStore.getState().user).toBeNull();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
