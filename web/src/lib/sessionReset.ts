import { queryClient } from "@/lib/queryClient";
import { useSiteStore } from "@/stores/site";

/**
 * Central session teardown. Called by the explicit logout (auth store) and by
 * forceLogout() (expired session), guaranteeing that no user data, site state
 * or cached query survives a session boundary.
 *
 * Idempotent: calling it repeatedly is safe (clearing an empty cache and
 * resetting an already-reset store are no-ops).
 */
export function resetSession(): void {
  queryClient.clear();
  useSiteStore.getState().reset();
}