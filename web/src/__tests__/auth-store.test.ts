import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock api client
vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
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

  it("should logout and clear tokens", async () => {
    const { useAuthStore } = await import("@/stores/auth");

    // Set logged in state
    useAuthStore.setState({
      user: { id: "user-1", email: "test@test.com", name: "Test", role: "admin" },
      isAuthenticated: true,
    });

    localStorageMock.setItem("access_token", "test-token");

    useAuthStore.getState().logout();

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
