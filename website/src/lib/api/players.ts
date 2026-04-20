import { queryOptions } from "@tanstack/svelte-query";
import type { UndefinedInitialDataOptions } from "@tanstack/svelte-query";

export interface Player {
    id: number;
    name: string;
    ping: number;
    license: string;
    discord: string;
}

async function fetchPlayers(serverId: string): Promise<Player[]> {
    const res: Response = await fetch(`/v1/servers/${encodeURIComponent(serverId)}/players`);
    if (!res.ok) {
        const payload: { error?: string } = await res.json().catch(() => ({}));
        throw new Error(payload.error ?? `GET /v1/servers/${serverId}/players failed: ${res.status}`);
    }
    const players = (await res.json()) as Player[];
    return players.map((p) => ({
        id: p.id,
        name: p.name,
        ping: p.ping,
        license: p.license ?? "",
        discord: p.discord ?? "",
    }));
}

export const playersQueryOptions = (
    serverId: string | null,
): UndefinedInitialDataOptions<Player[], Error, Player[], (string | null)[]> =>
    queryOptions({
        queryKey: ["players", serverId],
        queryFn: () => {
            if (!serverId) return Promise.resolve([]);
            return fetchPlayers(serverId);
        },
        enabled: serverId !== null,
        refetchInterval: 1000 * 5, // poll every 5s — volatile data
        refetchIntervalInBackground: false,
    });
