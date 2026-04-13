/**
 * Managed-server list API client and TanStack Query options.
 */

import { queryOptions } from "@tanstack/svelte-query";
import type { UndefinedInitialDataOptions } from "@tanstack/svelte-query";

/** Lifecycle state of a managed server instance. */
export type ServerStatus = "online" | "starting" | "stopped" | "crashed";

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
    name: string;
    artifactVersion: string;
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
