/**
 * Central registry of all user-scoped preferences.
 *
 * Each entry is a `PreferenceSpec<T>` that declares how to encode/decode
 * the value for transport and storage. To add a new preference:
 *
 *   1. Whitelist the key in `api/internal/preferences/keys.go`
 *   2. Export a new spec below
 *   3. Wrap it with `createPreferenceStore(spec)` wherever it's consumed
 *
 * The backend never inspects the value, so the format is chosen per-spec
 * (base62-packed bits, JSON, plain string, …).
 */

import type { GridLayoutItem } from "$lib/types/grid-layout";
import { encodeLayout, paramToLayout } from "$lib/layout-codec";
import type { PreferenceSpec } from "./store.svelte";

const DEFAULT_DASHBOARD_LAYOUT: GridLayoutItem[] = [
    { id: "console", x: 0, y: 0, w: 7, h: 6, minW: 4, minH: 3 },
    { id: "players", x: 7, y: 0, w: 5, h: 6, minW: 3, minH: 3 },
];

/**
 * Dashboard widget grid layout. Stored as the base62-packed code produced
 * by the existing share-layout codec — tiny payload and built-in schema
 * versioning (mismatches decode to `null` and fall back to defaults).
 */
export const dashboardLayoutPref: PreferenceSpec<GridLayoutItem[]> = {
    key: "dashboard-layout",
    defaultValue: DEFAULT_DASHBOARD_LAYOUT,
    encode: encodeLayout,
    decode: paramToLayout,
};
