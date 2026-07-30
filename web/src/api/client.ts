import { useSiteStore } from "@/stores/site";

const API_BASE = "/api/v1";

export interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  params?: Record<string, string>;
  formData?: boolean;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

let refreshPromise: Promise<boolean> | null = null;

async function attemptRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    const refreshToken = localStorage.getItem("refresh_token");
    if (!refreshToken) return false;

    try {
      const response = await fetch(`${API_BASE}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!response.ok) return false;

      const data = await response.json();
      localStorage.setItem("access_token", data.access_token);
      localStorage.setItem("refresh_token", data.refresh_token);
      return true;
    } catch {
      return false;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

function forceLogout() {
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
  window.location.href = "/admin/login";
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, window.location.origin);

  if (options.params) {
    Object.entries(options.params).forEach(([key, value]) => {
      url.searchParams.set(key, value);
    });
  }

  const isFormData = options.formData || options.body instanceof FormData;

  const headers: Record<string, string> = {
    ...options.headers,
  };

  if (!isFormData) {
    headers["Content-Type"] = "application/json";
  }

  const token = localStorage.getItem("access_token");
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const siteState = useSiteStore.getState();
  if (siteState.currentSite) {
    headers["X-Site-ID"] = siteState.currentSite.id;
  }

  const fetchOptions: RequestInit = {
    method: options.method || "GET",
    headers,
    body: isFormData ? (options.body as FormData) : options.body ? JSON.stringify(options.body) : undefined,
  };

  let response = await fetch(url.toString(), fetchOptions);

  if (response.status === 401 && token) {
    const refreshed = await attemptRefresh();
    if (refreshed) {
      const newToken = localStorage.getItem("access_token");
      if (newToken) {
        headers["Authorization"] = `Bearer ${newToken}`;
      }
      fetchOptions.headers = { ...headers };
      response = await fetch(url.toString(), fetchOptions);
    } else {
      forceLogout();
      throw new ApiError(401, "SESSION_EXPIRED", "Sessão expirada. Faça login novamente.");
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "UNKNOWN", message: "Erro desconhecido" },
    }));
    throw new ApiError(
      response.status,
      error.error?.code || "UNKNOWN",
      error.error?.message || "Ocorreu um erro",
    );
  }

  if (response.status === 204) return undefined as T;
  return response.json();
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "GET" }),

  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "POST", body }),

  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PUT", body }),

  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "PATCH", body }),

  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { ...options, method: "DELETE" }),
};
