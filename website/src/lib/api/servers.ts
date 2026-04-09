/**
 * Managed-server list API client and TanStack Query options.
 *
 * The panel manages multiple FiveM server instances. Each entry in the
 * returned list represents one directory under servers/ — the id is the
 * directory name (stable across renames), the rest is runtime telemetry
 * that the backend will eventually stream from the server process.
 *
 * Backend endpoint does not exist yet; this module returns mock data so
 * the UI can be built and iterated on independently.
 */

import { queryOptions } from "@tanstack/svelte-query";

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
}

// Mock data — replace with GET /v1/servers once the backend exists.
const mockServers: ManagedServer[] = [
    {
        id: "runfive-dev",
        name: "RunFive Dev",
        status: "online",
        address: "127.0.0.1:30120",
        playerCount: 42,
        maxPlayers: 64,
        cpu: 23,
        ramMB: 4200,
        tickMs: 8.2,
    },
    {
        id: "runfive-staging",
        name: "Staging",
        status: "online",
        address: "10.0.0.14:30120",
        playerCount: 7,
        maxPlayers: 32,
        cpu: 11,
        ramMB: 2100,
        tickMs: 5.4,
    },
    {
        id: "runfive-eu-prod",
        name: "EU · Prod",
        status: "stopped",
        address: "5.45.102.19:30120",
        playerCount: 0,
        maxPlayers: 128,
        cpu: 0,
        ramMB: 0,
        tickMs: 0,
    },
];

async function fetchServers(): Promise<ManagedServer[]> {
    // TODO: replace with actual API call
    // return (await fetch("/v1/servers")).json();
    return mockServers;
}

export const serversQueryOptions = () =>
    queryOptions({
        queryKey: ["servers"],
        queryFn: fetchServers,
        refetchInterval: 1000 * 5, // poll every 5s — volatile telemetry
        refetchIntervalInBackground: false,
    });
