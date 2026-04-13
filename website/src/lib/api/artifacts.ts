import { queryOptions } from "@tanstack/svelte-query";
import type { UndefinedInitialDataOptions } from "@tanstack/svelte-query";

export interface InstalledArtifact {
    os: string;
    version: string;
    path: string;
}

export interface AvailableArtifactVersion {
    version: string;
    installed: boolean;
}

export interface ArtifactListResponse {
    os: string;
    installed: InstalledArtifact[];
    available: AvailableArtifactVersion[];
}

async function fetchArtifacts(): Promise<ArtifactListResponse> {
    const res: Response = await fetch("/v1/artifacts");
    if (!res.ok) throw new Error(`GET /v1/artifacts failed: ${res.status}`);
    return (await res.json()) as ArtifactListResponse;
}

export async function downloadArtifact(version: string): Promise<InstalledArtifact> {
    const res: Response = await fetch("/v1/artifacts/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ version }),
    });

    if (!res.ok) {
        const payload: { error?: string } = (await res.json()) as { error?: string };
        throw new Error(payload.error ?? `POST /v1/artifacts/download failed: ${res.status}`);
    }

    return (await res.json()) as InstalledArtifact;
}

export const artifactsQueryOptions = (): UndefinedInitialDataOptions<
    ArtifactListResponse,
    Error,
    ArtifactListResponse,
    string[]
> =>
    queryOptions({
        queryKey: ["artifacts"],
        queryFn: fetchArtifacts,
        staleTime: 1000 * 30,
        retry: false,
    });
