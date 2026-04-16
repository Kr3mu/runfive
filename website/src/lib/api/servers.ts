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
