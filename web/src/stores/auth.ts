import { create } from "zustand";
import { api } from "@/api/client";

interface User {
  id: string;
  email: string;
  name: string;
  role: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  checkAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  login: async (email: string, password: string) => {
    const response = await api.post<{
      access_token: string;
      refresh_token: string;
      user: User;
    }>("/auth/login", { email, password });

    localStorage.setItem("access_token", response.access_token);
    localStorage.setItem("refresh_token", response.refresh_token);

    set({
      user: response.user,
      isAuthenticated: true,
    });
  },

  logout: () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    set({ user: null, isAuthenticated: false });
  },

  checkAuth: async () => {
    const token = localStorage.getItem("access_token");
    if (!token) {
      set({ isLoading: false });
      return;
    }

    try {
      const response = await api.get<{ user: User }>("/auth/me");
      set({ user: response.user, isAuthenticated: true, isLoading: false });
    } catch {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
}));
