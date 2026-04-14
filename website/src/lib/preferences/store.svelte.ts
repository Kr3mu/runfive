/**
 * Generic per-user preference store.
 *
 * Each preference is declared once as a `PreferenceSpec<T>` (see registry.ts)
 * and turned into a reactive store via `createPreferenceStore(spec)`. The
 * store handles:
 *
 *  - Reactive `$state<T>` initialized to the spec default
 *  - Synchronous localStorage cache keyed per-user for instant first paint
 *  - Background fetch from the backend on `hydrate(userId)`
 *  - Debounced PUT on every `set(...)` (default 500 ms)
 *  - Graceful fallback: fetch/parse errors leave the local state intact
 *
 * Adding a new preference = declare its spec in registry.ts and whitelist
 * the key on the backend. No new store class, no new HTTP wiring.
 */

import { fetchPreference, putPreference, deletePreference } from "$lib/api/preferences";

export interface PreferenceSpec<T> {
    /** Preference key, must match the backend whitelist. */
    key: string;
    /** Value used before hydration and after failed decode. */
    defaultValue: T;
    /** Serializes the value to an opaque string for storage. */
    encode: (value: T) => string;
    /** Deserializes a stored string; return null to fall back to defaults. */
    decode: (raw: string) => T | null;
    /** Debounce window for saves; defaults to 500 ms. */
    debounceMs?: number;
}

export interface PreferenceStore<T> {
    /** Current reactive value. */
    readonly value: T;
    /** True while the first server fetch is in flight. */
    readonly isLoading: boolean;
    /** Replace the value, write through to localStorage and schedule a debounced PUT. */
    set(value: T): void;
    /** Delete the stored value on the server and reset to the spec default. */
    reset(): Promise<void>;
    /**
     * Load the server value for `userId`. Must be called once the authed
     * user is known (typically from a `$effect` in a layout). Safe to call
     * again when the user changes — the store rescopes its cache.
     */
    hydrate(userId: number): Promise<void>;
}

function localStorageKey(userId: number, key: string): string {
    return `runfive:pref:${userId}:${key}`;
}

export function createPreferenceStore<T>(spec: PreferenceSpec<T>): PreferenceStore<T> {
    const debounceMs = spec.debounceMs ?? 500;

    let state = $state<T>(spec.defaultValue);
    let loading = $state(false);
    let currentUserId: number | null = null;
    let saveTimer: ReturnType<typeof setTimeout> | null = null;
    let hydrateSeq = 0;

    function readLocal(userId: number): T | null {
        try {
            const raw = localStorage.getItem(localStorageKey(userId, spec.key));
            if (raw === null) return null;
            return spec.decode(raw);
        } catch {
            return null;
        }
    }

    function writeLocal(userId: number, encoded: string): void {
        try {
            localStorage.setItem(localStorageKey(userId, spec.key), encoded);
        } catch {
            // localStorage full or unavailable — silently ignore; server is source of truth.
        }
    }

    function clearLocal(userId: number): void {
        try {
            localStorage.removeItem(localStorageKey(userId, spec.key));
        } catch {
            // ignore
        }
    }

    function scheduleSave(value: T): void {
        if (currentUserId === null) return;
        if (saveTimer !== null) clearTimeout(saveTimer);
        const encoded = spec.encode(value);
        saveTimer = setTimeout(() => {
            saveTimer = null;
            void putPreference(spec.key, encoded).catch(() => {
                // swallow — next mutation will retry; local state stays authoritative.
            });
        }, debounceMs);
    }

    return {
        get value(): T {
            return state;
        },
        get isLoading(): boolean {
            return loading;
        },
        set(value: T): void {
            try {
                state = value;
                if (currentUserId !== null) {
                    writeLocal(currentUserId, spec.encode(value));
                }
                scheduleSave(value);
            } catch (err) {
                console.error(`[pref:${spec.key}] set failed`, err);
            }
        },
        async reset(): Promise<void> {
            if (saveTimer !== null) {
                clearTimeout(saveTimer);
                saveTimer = null;
            }
            state = spec.defaultValue;
            if (currentUserId !== null) {
                clearLocal(currentUserId);
                try {
                    await deletePreference(spec.key);
                } catch {
                    // ignore; local state already reset.
                }
            }
        },
        async hydrate(userId: number): Promise<void> {
            // Rescope if the user changed — discard timers and stale state.
            if (currentUserId !== null && currentUserId !== userId) {
                if (saveTimer !== null) {
                    clearTimeout(saveTimer);
                    saveTimer = null;
                }
                state = spec.defaultValue;
            }
            currentUserId = userId;

            const cached = readLocal(userId);
            if (cached !== null) {
                state = cached;
            }

            const seq = ++hydrateSeq;
            loading = true;
            try {
                const raw = await fetchPreference(spec.key);
                if (seq !== hydrateSeq) return;
                if (raw === null) return;
                const decoded = spec.decode(raw);
                if (decoded !== null) {
                    state = decoded;
                    writeLocal(userId, raw);
                }
            } catch (err) {
                console.error(`[pref:${spec.key}] hydrate failed`, err);
            } finally {
                if (seq === hydrateSeq) loading = false;
            }
        },
    };
}
