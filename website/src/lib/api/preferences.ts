/**
 * Raw fetch wrappers for the per-user preferences endpoints.
 *
 * The value is an opaque string — encoding/decoding is the caller's
 * responsibility (see $lib/preferences/store.svelte for the reactive
 * wrapper that handles this).
 */

interface PreferenceResponse {
    key: string;
    value: string;
}

/**
 * Fetches the current user's value for `key`.
 * Returns `null` when the user has no entry for this key (404).
 */
export async function fetchPreference(key: string): Promise<string | null> {
    const res = await fetch(`/v1/preferences/${encodeURIComponent(key)}`);
    if (res.status === 404) return null;
    if (!res.ok) throw new Error(`GET /v1/preferences/${key} failed: ${res.status}`);
    const body = (await res.json()) as PreferenceResponse;
    return body.value;
}

/** Upserts the value for `key` on the current user. */
export async function putPreference(key: string, value: string): Promise<void> {
    const res = await fetch(`/v1/preferences/${encodeURIComponent(key)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value }),
    });
    if (!res.ok) throw new Error(`PUT /v1/preferences/${key} failed: ${res.status}`);
}

/** Removes the stored value for `key` on the current user (idempotent). */
export async function deletePreference(key: string): Promise<void> {
    const res = await fetch(`/v1/preferences/${encodeURIComponent(key)}`, {
        method: "DELETE",
    });
    if (!res.ok) throw new Error(`DELETE /v1/preferences/${key} failed: ${res.status}`);
}
