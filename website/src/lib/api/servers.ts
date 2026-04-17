/**
 * Managed-server lifecycle API client and TanStack Query options.
 */

import { queryOptions } from "@tanstack/svelte-query";
import type { UndefinedInitialDataOptions } from "@tanstack/svelte-query";

/** Lifecycle state of a managed server instance. */
export type ServerStatus = "running" | "starting" | "stopped" | "crashed";

/** A managed FiveM server instance as surfaced in the panel. */
export interface ManagedServer {
    /** Directory name under servers/ — matches the server.toml parent dir */
    id: string;
    /** Display name from server.toml */
    name: string;
    /** Current lifecycle state */
    status: ServerStatus;
    /** Public "host:port" shown in the switcher */
    address: string;
    /** Configured TCP/UDP endpoint port from server.toml */
    port: number;
    /** Connected players right now */
    playerCount: number;
    /** Configured max slots */
    maxPlayers: number;
    /** CPU utilization (0–100) */
    cpu: number;
    /** Resident memory in MB */
    ramMB: number;
    /** Server tick time in milliseconds */
    tickMs: number;
    /** Shared artifact version referenced by server.toml */
    artifactVersion: string;
}

export interface CreateServerRequest {
    /** Human-readable server name. */
    name: string;
    /** FiveM artifact build the server will launch against. */
    artifactVersion: string;
    /** Optional Cfx.re license key (cfxk_...). Encrypted server-side. */
    licenseKey?: string;
    /** TCP/UDP endpoint port. Omit or 0 = server-side auto-allocation. */
    port?: number;
    /** sv_maxclients slot count. Omit or 0 = server-side default (32). */
    maxPlayers?: number;
}

/**
 * Partial mutation body for {@link updateServer}. Every field is optional and
 * omitted fields leave the stored value untouched. For the three fields that
 * carry a meaningful cleared state (`licenseKey`, `enforceGameBuild`,
 * `onesync`) sending an empty string clears the value on disk.
 */
export interface UpdateServerRequest {
    /** Display name (sv_hostname). Cannot be cleared. */
    name?: string;
    /** fxserver build identifier. The server installs it on the caller's behalf. */
    artifactVersion?: string;
    /** Cfx.re key rotation. "" clears, "cfxk_..." rotates, omitted keeps. */
    licenseKey?: string;
    /** TCP/UDP endpoint port. 0 triggers server-side re-allocation. */
    port?: number;
    /** sv_maxclients. 0 resets to the panel default. */
    maxPlayers?: number;
    /** sv_enforceGameBuild. Empty string clears. */
    enforceGameBuild?: string;
    /** gameplay.onesync: "on" | "legacy" | "infinity" | "off" | "" (disabled). */
    onesync?: string;
    /** Full replacement for the resources.ensure list. */
    resourcesEnsure?: string[];
}

export interface ServerProcessStatus {
    id: string;
    status: ServerStatus;
    pid?: number;
    exitCode?: number;
    exitReason?: string;
    updatedAt: string;
}

export interface ServerLogLine {
    id: number;
    timestamp: string;
    stream: "stdout" | "stderr" | "stdin" | "system" | string;
    message: string;
}

export interface ServerLogsResponse {
    lines: ServerLogLine[];
}

export interface ServerConsoleEvent {
    type: "snapshot" | "status" | "line" | "error";
    status?: ServerProcessStatus;
    lines?: ServerLogLine[];
    line?: ServerLogLine;
    error?: string;
}

async function fetchServers(): Promise<ManagedServer[]> {
    const res: Response = await fetch("/v1/servers");
    if (!res.ok) throw new Error(`GET /v1/servers failed: ${res.status}`);
    return (await res.json()) as ManagedServer[];
}

export async function createServer(body: CreateServerRequest): Promise<ManagedServer> {
    const res: Response = await fetch("/v1/servers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });

    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `POST /v1/servers failed: ${res.status}`);
    }

    return (await res.json()) as ManagedServer;
}

/**
 * Apply a partial update to an existing server's on-disk config.
 *
 * The backend mirrors the create-path validation (port ranges, slot bounds,
 * onesync allow-list, license-key prefix) and re-emits the generated
 * server.cfg synchronously, so a successful response means the next launch
 * will pick up the change.
 *
 * @param serverId - Directory ID (stable across renames).
 * @param body - Fields to change. Omitted fields leave the stored value alone.
 * @returns The updated managed server as it appears in the panel list.
 */
export async function updateServer(
    serverId: string,
    body: UpdateServerRequest,
): Promise<ManagedServer> {
    const res: Response = await fetch(`/v1/servers/${encodeURIComponent(serverId)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });

    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `PUT /v1/servers/${serverId} failed: ${res.status}`);
    }

    return (await res.json()) as ManagedServer;
}

/**
 * Remove a server from the panel.
 *
 * Soft-deletes by default — the directory is moved under
 * `servers/.trash/<id>.<timestamp>/` so fat-finger deletes can be rolled back
 * by an operator. Pass `{ trash: false }` to permanently remove the directory.
 *
 * The backend rejects with 409 while the launcher still owns a live process,
 * so callers should stop the server first (or let the UI's confirm dialog do
 * that) before calling this.
 *
 * @param serverId - Directory ID of the server to remove.
 * @param options.trash - Defaults to true (soft-delete). Set to false for permanent removal.
 */
export async function deleteServer(
    serverId: string,
    options: { trash?: boolean } = {},
): Promise<void> {
    const url = new URL(`/v1/servers/${encodeURIComponent(serverId)}`, window.location.origin);
    if (options.trash === false) url.searchParams.set("trash", "false");

    const res: Response = await fetch(url.toString(), { method: "DELETE" });
    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `DELETE /v1/servers/${serverId} failed: ${res.status}`);
    }
}

export async function fetchServerStatus(serverId: string): Promise<ServerProcessStatus> {
    const res: Response = await fetch(`/v1/servers/${encodeURIComponent(serverId)}/status`);
    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `GET /v1/servers/${serverId}/status failed: ${res.status}`);
    }
    return (await res.json()) as ServerProcessStatus;
}

export async function startServer(serverId: string): Promise<ServerProcessStatus> {
    const res: Response = await fetch(`/v1/servers/${encodeURIComponent(serverId)}/start`, {
        method: "POST",
    });
    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `POST /v1/servers/${serverId}/start failed: ${res.status}`);
    }
    return (await res.json()) as ServerProcessStatus;
}

export async function stopServer(serverId: string): Promise<ServerProcessStatus> {
    const res: Response = await fetch(`/v1/servers/${encodeURIComponent(serverId)}/stop`, {
        method: "POST",
    });
    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `POST /v1/servers/${serverId}/stop failed: ${res.status}`);
    }
    return (await res.json()) as ServerProcessStatus;
}

export async function fetchServerLogs(serverId: string, n = 200): Promise<ServerLogLine[]> {
    const url = new URL(`/v1/servers/${encodeURIComponent(serverId)}/logs`, window.location.origin);
    url.searchParams.set("n", String(n));

    const res: Response = await fetch(url.toString());
    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `GET /v1/servers/${serverId}/logs failed: ${res.status}`);
    }

    return ((await res.json()) as ServerLogsResponse).lines;
}

export function serverLogsWebSocketURL(serverId: string): string {
    const url = new URL(`/v1/servers/${encodeURIComponent(serverId)}/logs/ws`, window.location.origin);
    url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return url.toString();
}

export const serversQueryOptions = (): UndefinedInitialDataOptions<
    ManagedServer[],
    Error,
    ManagedServer[],
    string[]
> =>
    queryOptions({
        queryKey: ["servers"],
        queryFn: fetchServers,
        refetchInterval: 1000 * 5,
        refetchIntervalInBackground: false,
    });
