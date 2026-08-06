import { create } from "zustand";
import { api } from "@/api/client";

export interface Site {
  id: string;
  name: string;
  slug: string;
  description?: string;
  status: string;
  owner_id: string;
  locale?: string;
  timezone?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Lifecycle of loading the available sites:
 * - idle:     nothing requested yet
 * - loading:  a site (or an automatic retry attempt) is in flight
 * - success:  GET /sites succeeded and returned at least one site
 * - empty:    GET /sites succeeded but returned no sites (NOT an error)
 * - error:    GET /sites failed after all available retry attempts
 *
 * `error` always carries a message in `error`; `empty` means the API answered
 * successfully with no sites. The two states are distinct so the UI never
 * mistakes a network failure for "no sites available".
 */
export type SiteLoadStatus = "idle" | "loading" | "success" | "empty" | "error";

interface SiteListResponse {
  sites: Site[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

interface SiteState {
  sites: Site[];
  currentSite: Site | null;
  status: SiteLoadStatus;
  isLoading: boolean;
  error: string | null;
  /** Number of attempts made in the current load cycle (1..MAX_SITE_FETCH_ATTEMPTS). */
  attempts: number;
  fetchSites: () => Promise<void>;
  retrySites: () => Promise<void>;
  setCurrentSite: (site: Site) => void;
  clearCurrentSite: () => void;
  /** Full teardown used on logout: invalidates in-flight loads, clears all
   * state and removes the persisted `current_site_id`. Idempotent. */
  reset: () => void;
}

const STORAGE_KEY = "current_site_id";

/** Automatic attempts per load cycle (initial request + retries before failing). */
export const MAX_SITE_FETCH_ATTEMPTS = 3;
/** Base backoff delay; each retry waits `SITE_RETRY_BACKOFF_MS * attempt`. */
export const SITE_RETRY_BACKOFF_MS = 800;

function retryDelay(attempt: number): number {
  return SITE_RETRY_BACKOFF_MS * attempt;
}

function loadPersistedSite(sites: Site[]): Site | null {
  const storedId = localStorage.getItem(STORAGE_KEY);
  if (!storedId) return null;
  return sites.find((s) => s.id === storedId) || null;
}

export const useSiteStore = create<SiteState>((set, get) => {
  /**
   * Monotonic generation counter. Every load() captures the current value at
   * start; whenever the captured generation differs from the current one the
   * request result is stale (e.g. the session was reset mid-flight) and is
   * discarded. reset() bumps it so no in-flight fetchSites can ever apply
   * results from a previous session.
   */
  let loadEpoch = 0;

  function applySites(sites: Site[]) {
    const persisted = loadPersistedSite(sites);

    if (persisted) {
      localStorage.setItem(STORAGE_KEY, persisted.id);
      set({ sites, currentSite: persisted });
    } else if (sites.length > 0) {
      const firstSite = sites[0];
      if (!firstSite) return;
      localStorage.setItem(STORAGE_KEY, firstSite.id);
      set({ sites, currentSite: firstSite });
    } else {
      localStorage.removeItem(STORAGE_KEY);
      set({ sites: [], currentSite: null });
    }
  }

  async function load(): Promise<void> {
    const epoch = loadEpoch;
    for (let attempt = 1; attempt <= MAX_SITE_FETCH_ATTEMPTS; attempt++) {
      if (epoch !== loadEpoch) return;
      set({ status: "loading", isLoading: true, error: null, attempts: attempt });
      try {
        const response = await api.get<SiteListResponse>("/sites");
        if (epoch !== loadEpoch) return;
        const sites = response.sites || [];
        applySites(sites);
        if (epoch !== loadEpoch) return;
        set({
          status: sites.length > 0 ? "success" : "empty",
          isLoading: false,
          error: null,
          attempts: attempt,
        });
        return;
      } catch (err) {
        if (epoch !== loadEpoch) return;
        if (attempt < MAX_SITE_FETCH_ATTEMPTS) {
          await new Promise((resolve) => setTimeout(resolve, retryDelay(attempt)));
          continue;
        }
        const message =
          err instanceof Error && err.message ? err.message : "Não foi possível carregar os sites";
        set({ status: "error", isLoading: false, error: message, attempts: attempt });
      }
    }
  }

  return {
    sites: [],
    currentSite: null,
    status: "idle",
    isLoading: false,
    error: null,
    attempts: 0,

    fetchSites: async () => {
      if (get().status === "loading") return;
      await load();
    },

    retrySites: async () => {
      await get().fetchSites();
    },

    setCurrentSite: (site: Site) => {
      localStorage.setItem(STORAGE_KEY, site.id);
      set({ currentSite: site });
    },

    clearCurrentSite: () => {
      localStorage.removeItem(STORAGE_KEY);
      set({ currentSite: null });
    },

    reset: () => {
      loadEpoch++;
      localStorage.removeItem(STORAGE_KEY);
      set({
        sites: [],
        currentSite: null,
        status: "idle",
        isLoading: false,
        error: null,
        attempts: 0,
      });
    },
  };
});