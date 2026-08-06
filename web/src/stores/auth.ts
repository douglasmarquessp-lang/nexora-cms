import { create } from "zustand";
import { api } from "@/api/client";
import { resetSession } from "@/lib/sessionReset";

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  avatar?: string;
  mfa_enabled?: boolean;
  last_login?: string;
  created_at?: string;
  updated_at?: string;
}

export type LoginResult = { status: "ok" } | { status: "mfa_required" };

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string, mfaCode?: string) => Promise<LoginResult>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
  setUser: (user: User | null) => void;
}

type LoginResponse =
  | { status: "mfa_required" }
  | {
      status: "ok";
      access_token: string;
      refresh_token: string;
      token_type: string;
      expires_in: number;
      user: User;
    };

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  login: async (email: string, password: string, mfaCode?: string) => {
    const payload: Record<string, string> = { email, password };
    if (mfaCode) {
      payload.mfa_code = mfaCode;
    }

    const response = await api.post<LoginResponse>("/auth/login", payload);

    if (response.status === "mfa_required") {
      return { status: "mfa_required" };
    }

    localStorage.setItem("access_token", response.access_token);
    localStorage.setItem("refresh_token", response.refresh_token);

    set({
      user: response.user,
      isAuthenticated: true,
    });

    return { status: "ok" };
  },

  logout: async () => {
    try {
      await api.post("/auth/logout");
    } catch {
      /* noop */
    }

    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    resetSession();
    set({ user: null, isAuthenticated: false });
  },

  checkAuth: async () => {
    const token = localStorage.getItem("access_token");
    if (!token) {
      set({ isLoading: false });
      return;
    }

    try {
      const response = await api.get<User>("/auth/me");
      set({ user: response, isAuthenticated: true, isLoading: false });
    } catch {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  setUser: (user) => {
    set({ user, isAuthenticated: user !== null });
  },
}));
