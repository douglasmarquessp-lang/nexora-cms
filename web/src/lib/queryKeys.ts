import { useSiteStore } from "@/stores/site";

/**
 * Placeholder used in site-scoped query keys when no current site is selected.
 * A disabled query still registers a cache entry; this keeps the key shape
 * stable and predictable without colliding with real site ids (UUIDs).
 */
export const NO_SITE_KEY = "__no_site__";

/**
 * Reactive id of the currently selected site (or null when none is set).
 * Components subscribe to it via Zustand so that a site switch re-renders
 * them and, because the site id is part of the query key, TanStack Query
 * automatically fetches the new site's data.
 */
export function useCurrentSiteId(): string | null {
  return useSiteStore((s) => s.currentSite?.id ?? null);
}

/**
 * Builds a site-scoped query key with a stable, predictable shape:
 * [resourceName, siteId, ...rest].
 *
 * The resource name stays the first element so prefix invalidations such as
 * `invalidateQueries({ queryKey: ["media"] })` keep working unchanged, while
 * the site id in position 1 guarantees Site A and Site B never share cache
 * entries.
 */
export function siteQueryKey(
  parts: Array<string | number | null | undefined>,
  siteId: string | null | undefined,
): Array<string | number | null | undefined> {
  return [parts[0], siteId ?? NO_SITE_KEY, ...parts.slice(1)];
}
