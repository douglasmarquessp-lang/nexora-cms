/**
 * Temporary site-selection restriction for the Admin.
 *
 * This is the SINGLE place where the Admin decides which sites may be
 * shown/selected. By default the Admin shows every site the API returns;
 * when this list is non-empty, only the listed site IDs are kept.
 *
 * Current state: only AIWorkSimple is available for now. Other sites are NOT
 * deleted — they remain in the database and can be re-enabled here later.
 */
export const ADMIN_ALLOWED_SITE_IDS: string[] = [
  "a64d7d72-b97f-4f31-96fd-8aeb15f6184c", // AIWorkSimple (aiworksimple.com)
];
