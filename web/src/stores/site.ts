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
  isLoading: boolean;
  error: string | null;
  fetchSites: () => Promise<void>;
  setCurrentSite: (site: Site) => void;
  clearCurrentSite: () => void;
}

const STORAGE_KEY = "current_site_id";

function loadPersistedSite(sites: Site[]): Site | null {
  const storedId = localStorage.getItem(STORAGE_KEY);
  if (!storedId) return null;
  return sites.find((s) => s.id === storedId) || null;
}

export const useSiteStore = create<SiteState>((set, get) => ({
  sites: [],
  currentSite: null,
  isLoading: false,
  error: null,

  fetchSites: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.get<SiteListResponse>("/sites");
      const sites = response.sites || [];
      const persisted = loadPersistedSite(sites);
      set({
        sites,
        currentSite: persisted || (sites.length > 0 ? sites[0] : null),
        isLoading: false,
      });
      if (persisted) {
        localStorage.setItem(STORAGE_KEY, persisted.id);
      } else if (sites.length > 0) {
        localStorage.setItem(STORAGE_KEY, sites[0].id);
      }
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Failed to load sites",
        isLoading: false,
      });
    }
  },

  setCurrentSite: (site: Site) => {
    localStorage.setItem(STORAGE_KEY, site.id);
    set({ currentSite: site });
  },

  clearCurrentSite: () => {
    localStorage.removeItem(STORAGE_KEY);
    set({ currentSite: null });
  },
}));
