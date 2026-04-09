/**
 * Persistent selected-server state for the multi-server panel.
 *
 * The canonical server list lives in the TanStack query cache
 * (see $lib/api/servers) — this module only tracks *which* one the user
 * currently has open. The selected ID is stored in localStorage so the
 * choice survives reloads and is shared across tabs of the same origin.
 */

import type { ManagedServer } from "$lib/api/servers";

const STORAGE_KEY = "runfive:selected-server";

function loadInitial(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(STORAGE_KEY);
}

let selectedId = $state<string | null>(loadInitial());

export const serverState = {
    get selectedId(): string | null {
        return selectedId;
    },

    /**
     * Resolve the active server from a freshly-fetched list.
     *
     * Falls back to the first server when no selection exists or when the
     * stored ID no longer matches anything in the list (e.g. the user
     * removed that server on another device).
     */
    resolve(servers: ManagedServer[]): ManagedServer | null {
        if (servers.length === 0) return null;
        const match = servers.find((s) => s.id === selectedId);
        return match ?? servers[0];
    },

    /** Switch to a different server and persist the choice. */
    select(id: string): void {
        selectedId = id;
        if (typeof window !== "undefined") {
            localStorage.setItem(STORAGE_KEY, id);
        }
    },
};
