import { describe, it, expect, beforeEach, vi } from "vitest";

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

describe("AuthStore", () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllMocks();
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
