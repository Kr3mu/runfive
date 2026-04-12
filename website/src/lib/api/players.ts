import { queryOptions } from "@tanstack/svelte-query";
import type { UndefinedInitialDataOptions } from "@tanstack/svelte-query";

export interface Player {
    id: number;
    name: string;
    discord: string;
    license: string;
    connectedSince: string;
    allTimeConnected: string;
    ping: number;
    source: number;
}

// Mock data — replace with actual API call when backend is ready
const mockPlayers: Player[] = [
    { id: 1, name: "Kr3mu", discord: "kr3mu", license: "license:abcd1234ef56", connectedSince: "2h 14m", allTimeConnected: "342h", ping: 32, source: 1 },
    { id: 2, name: "Lananal", discord: "lananal", license: "license:7890abcd1234", connectedSince: "1h 52m", allTimeConnected: "281h", ping: 28, source: 2 },
    { id: 3, name: "xXDarkRiderXx", discord: "darkrider99", license: "license:ef567890abcd", connectedSince: "45m", allTimeConnected: "89h", ping: 54, source: 3 },
    { id: 4, name: "TurboNinja", discord: "turboninja", license: "license:1234ef5678ab", connectedSince: "3h 01m", allTimeConnected: "512h", ping: 41, source: 4 },
    { id: 5, name: "CoolGuy_2k", discord: "coolguy2k", license: "license:5678ab1234ef", connectedSince: "22m", allTimeConnected: "15h", ping: 67, source: 5 },
    { id: 6, name: "SilentSniper", discord: "silentsniper", license: "license:abcdef123456", connectedSince: "1h 10m", allTimeConnected: "198h", ping: 38, source: 6 },
    { id: 7, name: "PixelDrifter", discord: "pixeldrift", license: "license:654321fedcba", connectedSince: "4h 33m", allTimeConnected: "723h", ping: 22, source: 7 },
    { id: 8, name: "NeonBlade", discord: "neonblade", license: "license:aabb11cc22dd", connectedSince: "15m", allTimeConnected: "42h", ping: 88, source: 8 },
    { id: 9, name: "ChefRamsay_RP", discord: "cheframsay", license: "license:dd22cc11bbaa", connectedSince: "2h 45m", allTimeConnected: "456h", ping: 35, source: 9 },
    { id: 10, name: "MidnightFox", discord: "midnightfox", license: "license:112233aabbcc", connectedSince: "58m", allTimeConnected: "67h", ping: 44, source: 10 },
];

async function fetchPlayers(): Promise<Player[]> {
    // TODO: replace with actual API call
    // return (await fetch("/api/v1/players")).json();
    return mockPlayers;
}

export const playersQueryOptions = (): UndefinedInitialDataOptions<
    Player[],
    Error,
    Player[],
    string[]
> =>
    queryOptions({
        queryKey: ["players"],
        queryFn: fetchPlayers,
        refetchInterval: 1000 * 5, // poll every 5s — volatile data
        refetchIntervalInBackground: false,
    });
