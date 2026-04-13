<script lang="ts">
    import { createQuery, useQueryClient } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import { artifactsQueryOptions, downloadArtifact } from "$lib/api/artifacts";
    import { createServer, serversQueryOptions } from "$lib/api/servers";
    import GridStack from "$lib/components/dashboard/grid-stack.svelte";
    import GridWidget from "$lib/components/dashboard/grid-widget.svelte";
    import Console from "$lib/components/dashboard/console.svelte";
    import PlayerList from "$lib/components/dashboard/player-list.svelte";
    import LinuxIcon from "$lib/components/icons/linux.svelte";
    import WindowsIcon from "$lib/components/icons/windows.svelte";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { canGlobal } from "$lib/permissions.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import { getWidgetDef } from "$lib/widget-registry";
    import type { GridLayoutItem } from "$lib/types/grid-layout";
    import type { AvailableArtifactVersion } from "$lib/api/artifacts";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Eye from "@lucide/svelte/icons/eye";
    import ListChecks from "@lucide/svelte/icons/list-checks";
    import Server from "@lucide/svelte/icons/server";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";
    import HardDriveDownload from "@lucide/svelte/icons/hard-drive-download";
    import Search from "@lucide/svelte/icons/search";
    import Check from "@lucide/svelte/icons/check";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import Sparkles from "@lucide/svelte/icons/sparkles";
    import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
    import { toast } from "svelte-sonner";

    const widgetMap: Record<string, { component: typeof Console; title: string }> = {
        console: { component: Console, title: "Console" },
        players: { component: PlayerList, title: "Players" },
    };

    const authQuery = createQuery(() => authQueryOptions());
    const serversQuery = createQuery(() => serversQueryOptions());
    const queryClient = useQueryClient();

    const currentUser = $derived(authQuery.data);
    const servers = $derived(serversQuery.data ?? []);
    const canCreateServers = $derived(canGlobal(currentUser, "servers", "create"));
    const shouldLoadArtifacts = $derived(servers.length === 0 && canCreateServers);

    const artifactsQuery = createQuery(() => ({
        ...artifactsQueryOptions(),
        enabled: shouldLoadArtifacts,
    }));

    const installedArtifacts = $derived(artifactsQuery.data?.installed ?? []);
    const hostOs = $derived(artifactsQuery.data?.os);
    const hostOsLabel = $derived.by((): string => {
        if (!hostOs) return "Loading...";
        if (hostOs === "windows") return "Windows";
        if (hostOs === "linux") return "Linux";
        return hostOs;
    });

    type CreationPhase = "idle" | "downloading" | "creating";

    let serverName = $state("");
    let artifactVersion = $state("");

    let creationPhase = $state<CreationPhase>("idle");
    let phaseStartedAt = $state<number | null>(null);
    let elapsedMs = $state(0);

    const isCreatingServer = $derived(creationPhase !== "idle");

    const creationPhaseLabel = $derived.by((): string => {
        if (creationPhase === "downloading") return `Downloading build ${artifactVersion}`;
        if (creationPhase === "creating") return "Setting up your server";
        return "";
    });

    const creationPhaseHint = $derived.by((): string => {
        if (creationPhase === "downloading") {
            return "Fetching the artifact archive from runtime.fivem.net. This can take a minute on slower connections.";
        }
        if (creationPhase === "creating") {
            return "Writing server.toml and registering the server with the panel.";
        }
        return "";
    });

    $effect((): (() => void) | void => {
        if (creationPhase === "idle" || phaseStartedAt === null) return;
        const start: number = phaseStartedAt;
        const interval: number = window.setInterval((): void => {
            elapsedMs = Date.now() - start;
        }, 250);
        return (): void => window.clearInterval(interval);
    });

    function formatElapsed(ms: number): string {
        const total: number = Math.max(0, Math.floor(ms / 1000));
        const minutes: number = Math.floor(total / 60);
        const seconds: number = total % 60;
        return `${minutes}:${seconds.toString().padStart(2, "0")}`;
    }

    let artifactPopoverOpen = $state(false);
    let artifactSearch = $state("");
    let artifactSearchInput = $state<HTMLInputElement | null>(null);

    const recommendedVersion = $derived(artifactsQuery.data?.recommended ?? "");

    const selectedArtifactEntry = $derived(
        artifactsQuery.data?.available.find((a: AvailableArtifactVersion): boolean => a.version === artifactVersion),
    );
    const selectedArtifactInstalled = $derived(
        Boolean(selectedArtifactEntry?.installed),
    );
    const selectedArtifactBroken = $derived(
        Boolean(selectedArtifactEntry?.brokenReason),
    );

    const selectedArtifactLabel = $derived.by((): string => {
        if (!artifactVersion) return "Choose a build";
        if (selectedArtifactInstalled) return `${artifactVersion} · Ready to use`;
        return `${artifactVersion} · Downloads on create`;
    });

    /**
     * Curated list shown when no search is active: the community-recommended
     * build first, then the newest non-broken upstream versions up to a small
     * cap. Keeps the default view small.
     */
    const curatedOptions = $derived.by((): AvailableArtifactVersion[] => {
        const all: AvailableArtifactVersion[] = artifactsQuery.data?.available ?? [];
        if (all.length === 0) return [];

        const result: AvailableArtifactVersion[] = [];
        const recommended: AvailableArtifactVersion | undefined = recommendedVersion
            ? all.find((a: AvailableArtifactVersion): boolean => a.version === recommendedVersion)
            : undefined;

        if (recommended) result.push(recommended);

        const maxCurated = 5;
        for (const entry of all) {
            if (result.length >= maxCurated) break;
            if (entry.brokenReason) continue;
            if (entry.version === recommendedVersion) continue;
            result.push(entry);
        }

        return result;
    });

    const searchResults = $derived.by((): AvailableArtifactVersion[] => {
        const term: string = artifactSearch.trim().toLowerCase();
        if (!term) return [];
        const all: AvailableArtifactVersion[] = artifactsQuery.data?.available ?? [];
        return all
            .filter((a: AvailableArtifactVersion): boolean => a.version.toLowerCase().includes(term))
            .slice(0, 50);
    });

    const displayedOptions = $derived(
        artifactSearch.trim() ? searchResults : curatedOptions,
    );

    function selectArtifact(version: string): void {
        artifactVersion = version;
        artifactSearch = "";
        artifactPopoverOpen = false;
    }

    function openArtifactPopover(): void {
        if (artifactPopoverOpen) return;
        artifactPopoverOpen = true;
        setTimeout((): void => artifactSearchInput?.focus(), 0);
    }

    function closeArtifactPopover(): void {
        artifactPopoverOpen = false;
        artifactSearch = "";
    }

    function handleSearchKeydown(event: KeyboardEvent): void {
        if (event.key === "Escape") {
            event.preventDefault();
            closeArtifactPopover();
            return;
        }
        if (event.key === "Enter") {
            event.preventDefault();
            const first: AvailableArtifactVersion | undefined = displayedOptions[0];
            if (first) selectArtifact(first.version);
        }
    }

    /**
     * Mirrors `sanitizeDirName` in api/internal/serverfs/registry.go so the
     * preview folder path matches what the backend will actually create.
     */
    const folderPreview = $derived.by((): string => {
        let slug: string = serverName.trim();
        slug = slug.replace(/\s+/g, "_");
        slug = slug.replace(/[^A-Za-z0-9_-]+/g, "_");
        slug = slug.replace(/_+/g, "_");
        slug = slug.replace(/^[_\-.]+|[_\-.]+$/g, "");
        return slug === "" ? "server" : slug;
    });

    $effect((): void => {
        if (!shouldLoadArtifacts || artifactVersion || !artifactsQuery.data) return;
        artifactVersion =
            artifactsQuery.data.recommended ??
            curatedOptions[0]?.version ??
            artifactsQuery.data.installed[0]?.version ??
            "";
    });

    function handleLayoutChange(items: GridLayoutItem[]): void {
        // TODO: persist to backend/localStorage
    }

    function handleRemove(id: string): void {
        dashboardState.removeWidget(id);
    }

    function beginPhase(phase: CreationPhase): void {
        creationPhase = phase;
        phaseStartedAt = Date.now();
        elapsedMs = 0;
    }

    function resetPhase(): void {
        creationPhase = "idle";
        phaseStartedAt = null;
        elapsedMs = 0;
    }

    async function handleCreateServer(): Promise<void> {
        if (creationPhase !== "idle") return;
        if (!serverName.trim()) {
            toast.error("Enter a server name first");
            return;
        }
        if (!artifactVersion) {
            toast.error("Choose an artifact build first");
            return;
        }

        try {
            if (!selectedArtifactInstalled) {
                beginPhase("downloading");
                await downloadArtifact(artifactVersion);
                await queryClient.invalidateQueries({ queryKey: ["artifacts"] });
            }

            beginPhase("creating");
            const created = await createServer({
                name: serverName.trim(),
                artifactVersion,
            });

            serverState.select(created.id);
            serverName = "";

            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["servers"] }),
                queryClient.invalidateQueries({ queryKey: ["artifacts"] }),
            ]);

            toast.success(`Server ${created.name} is ready`);
        } catch (error: unknown) {
            const message: string =
                error instanceof Error ? error.message : "Failed to create server";
            toast.error(message);
        } finally {
            resetPhase();
        }
    }
</script>

{#if authQuery.isLoading || serversQuery.isPending}
    <div class="flex h-full items-center justify-center">
        <LoaderCircle size={20} class="animate-spin text-muted-foreground" />
    </div>
{:else if servers.length === 0}
    <div class="flex h-full flex-col overflow-y-auto">
        <div class="mx-auto w-full max-w-5xl px-6 py-8">
            {#if canCreateServers}
                <h1 class="mb-1 text-lg font-semibold text-foreground">
                    Create your first server
                </h1>
                <p class="mb-8 text-sm text-muted-foreground">
                    Give it a name and pick a FiveM build. Everything else is wired up for you.
                </p>

                <div class="grid gap-8 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
                    <div>
                        <section class="mb-8">
                            <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                                <Server size={14} />
                                Server
                            </h2>
                            <div class="rounded-lg border border-border bg-card p-4">
                                <label for="server-name" class="mb-1.5 block text-xs font-medium text-muted-foreground">
                                    Name
                                </label>
                                <input
                                    id="server-name"
                                    bind:value={serverName}
                                    type="text"
                                    maxlength="64"
                                    placeholder="RunFive RP"
                                    class="h-10 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground/40 focus:border-primary/50"
                                />
                                <p class="mt-2 text-[11px] text-muted-foreground/60">
                                    Spaces become underscores in the folder name.
                                </p>
                            </div>
                        </section>

                        <section class="mb-8">
                            <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                                <HardDriveDownload size={14} />
                                Artifact Build
                            </h2>
                            <div class="rounded-lg border border-border bg-card p-4">
                                {#if artifactsQuery.isPending}
                                    <div class="flex h-10 items-center gap-2 text-sm text-muted-foreground/60">
                                        <LoaderCircle size={14} class="animate-spin" />
                                        Loading catalog...
                                    </div>
                                {:else if artifactsQuery.error}
                                    <div class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                                        {artifactsQuery.error.message}
                                    </div>
                                {:else}
                                    <div class="relative">
                                        <button
                                            type="button"
                                            onclick={openArtifactPopover}
                                            class="flex h-10 w-full items-center justify-between gap-2 rounded-md border border-border bg-background px-3 text-left text-sm transition-colors hover:border-primary/40 focus:border-primary/50 focus:outline-none"
                                        >
                                            <span class="flex min-w-0 flex-1 items-center gap-2">
                                                {#if selectedArtifactBroken}
                                                    <TriangleAlert size={13} class="shrink-0 text-destructive" />
                                                {/if}
                                                <span class="truncate {artifactVersion ? (selectedArtifactBroken ? 'text-destructive' : 'text-foreground') : 'text-muted-foreground/45'}">
                                                    {selectedArtifactLabel}
                                                </span>
                                            </span>
                                            <ChevronDown size={14} class="shrink-0 text-muted-foreground/60" />
                                        </button>

                                        {#if artifactPopoverOpen}
                                            <button
                                                type="button"
                                                class="fixed inset-0 z-40 cursor-default"
                                                aria-label="Close build picker"
                                                onclick={closeArtifactPopover}
                                            ></button>

                                            <div class="absolute top-full right-0 left-0 z-50 mt-2 overflow-hidden rounded-md border border-border bg-popover shadow-[0_16px_48px_-24px_rgba(0,0,0,0.65)]">
                                                <div class="border-b border-border p-2">
                                                    <div class="relative">
                                                        <Search size={13} class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground/50" />
                                                        <input
                                                            bind:this={artifactSearchInput}
                                                            bind:value={artifactSearch}
                                                            type="text"
                                                            placeholder="Search by build number..."
                                                            onkeydown={handleSearchKeydown}
                                                            class="h-8 w-full rounded bg-background pr-2 pl-7 text-sm text-foreground outline-none placeholder:text-muted-foreground/40"
                                                        />
                                                    </div>
                                                </div>

                                                <div class="max-h-72 overflow-y-auto py-1">
                                                    {#if displayedOptions.length === 0}
                                                        <div class="px-3 py-6 text-center text-xs text-muted-foreground/60">
                                                            {artifactSearch.trim() ? "No builds match" : "No builds available"}
                                                        </div>
                                                    {:else}
                                                        {#if !artifactSearch.trim()}
                                                            <div class="flex items-center gap-1.5 px-3 pt-1 pb-1.5 text-[10px] font-semibold tracking-wider text-muted-foreground/50 uppercase">
                                                                <Sparkles size={10} />
                                                                Curated
                                                            </div>
                                                        {/if}
                                                        {#each displayedOptions as option (option.version)}
                                                            {@const isRecommended = option.version === recommendedVersion}
                                                            {@const isSelected = option.version === artifactVersion}
                                                            <button
                                                                type="button"
                                                                onclick={() => selectArtifact(option.version)}
                                                                class="flex w-full items-start gap-2 px-3 py-2 text-left transition-colors hover:bg-accent {isSelected ? 'bg-accent/60' : ''}"
                                                            >
                                                                <div class="min-w-0 flex-1">
                                                                    <div class="flex flex-wrap items-center gap-1.5">
                                                                        <span class="font-mono text-sm {option.brokenReason ? 'text-destructive/80' : 'text-foreground'}">
                                                                            {option.version}
                                                                        </span>
                                                                        {#if isRecommended}
                                                                            <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[9px] font-semibold text-emerald-500">
                                                                                <Sparkles size={9} />
                                                                                Recommended
                                                                            </span>
                                                                        {/if}
                                                                        {#if option.installed}
                                                                            <span class="rounded-full bg-primary/10 px-1.5 py-0.5 text-[9px] font-semibold text-primary">
                                                                                Installed
                                                                            </span>
                                                                        {/if}
                                                                        {#if option.brokenReason}
                                                                            <span class="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[9px] font-semibold text-destructive">
                                                                                <TriangleAlert size={9} />
                                                                                Broken
                                                                            </span>
                                                                        {/if}
                                                                    </div>
                                                                    {#if option.brokenReason}
                                                                        <p class="mt-0.5 line-clamp-2 text-[11px] text-destructive/70">
                                                                            {option.brokenReason}
                                                                        </p>
                                                                    {/if}
                                                                </div>
                                                                {#if isSelected}
                                                                    <Check size={14} class="mt-1 shrink-0 text-primary" />
                                                                {/if}
                                                            </button>
                                                        {/each}
                                                    {/if}
                                                </div>

                                                <div class="flex items-center justify-between gap-2 border-t border-border bg-muted/30 px-3 py-1.5 text-[10px] text-muted-foreground/60">
                                                    {#if !artifactSearch.trim() && artifactsQuery.data}
                                                        <span>Type a number to search all {artifactsQuery.data.available.length} builds.</span>
                                                    {:else}
                                                        <span></span>
                                                    {/if}
                                                    <a
                                                        href="https://github.com/jgscripts/fivem-artifacts-db"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        class="shrink-0 text-muted-foreground/60 transition-colors hover:text-foreground"
                                                    >
                                                        Stability data by jgscripts
                                                    </a>
                                                </div>
                                            </div>
                                        {/if}
                                    </div>
                                {/if}

                                {#if selectedArtifactBroken && selectedArtifactEntry?.brokenReason}
                                    <div class="mt-3 flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-[11px] text-destructive">
                                        <TriangleAlert size={13} class="mt-0.5 shrink-0" />
                                        <span><span class="font-semibold">Known issue:</span> {selectedArtifactEntry.brokenReason}</span>
                                    </div>
                                {/if}

                                {#if artifactsQuery.data}
                                    <div class="mt-3 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground/60">
                                        <span>Builds for</span>
                                        {#if hostOs === "windows"}
                                            <WindowsIcon class="h-3 w-3 text-primary" />
                                        {:else if hostOs === "linux"}
                                            <LinuxIcon class="h-3 w-3 text-primary" />
                                        {/if}
                                        <span class="font-medium text-foreground/80">{hostOsLabel}</span>
                                        <span class="text-muted-foreground/40">·</span>
                                        <span>{installedArtifacts.length} installed</span>
                                        <span class="text-muted-foreground/40">·</span>
                                        <span>{artifactsQuery.data.available.length} upstream</span>
                                    </div>
                                {/if}
                            </div>
                        </section>

                        {#if isCreatingServer}
                            <div class="rounded-lg border border-border bg-card p-4">
                                <div class="flex items-center justify-between gap-3">
                                    <div class="flex min-w-0 items-center gap-2">
                                        <LoaderCircle size={14} class="shrink-0 animate-spin text-primary" />
                                        <span class="truncate text-sm font-medium text-foreground">
                                            {creationPhaseLabel}
                                        </span>
                                    </div>
                                    <span class="shrink-0 font-mono text-xs tabular-nums text-muted-foreground">
                                        {formatElapsed(elapsedMs)}
                                    </span>
                                </div>
                                <div class="relative mt-3 h-1 overflow-hidden rounded-full bg-muted">
                                    <div class="progress-indeterminate absolute inset-y-0 w-1/3 rounded-full bg-primary"></div>
                                </div>
                                <p class="mt-2 text-[11px] text-muted-foreground/60">
                                    {creationPhaseHint}
                                </p>
                            </div>
                        {:else}
                            <div class="flex flex-wrap items-center justify-between gap-3">
                                <p class="text-[11px] text-muted-foreground/60">
                                    Missing builds are downloaded before the server is created.
                                </p>
                                <button
                                    onclick={handleCreateServer}
                                    disabled={artifactsQuery.isPending || !serverName.trim() || !artifactVersion}
                                    class="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
                                >
                                    Create Server
                                </button>
                            </div>
                        {/if}
                    </div>

                    <aside>
                        <section class="mb-8">
                            <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                                <Eye size={14} />
                                Preview
                            </h2>
                            <div class="rounded-lg border border-border bg-card p-4">
                                <div class="mb-3 flex items-start justify-between gap-3">
                                    <div class="min-w-0 flex-1">
                                        <p class="truncate text-sm font-semibold text-foreground">
                                            {serverName.trim() || "Unnamed server"}
                                        </p>
                                        <p class="mt-0.5 truncate font-mono text-[11px] text-muted-foreground/60">
                                            servers/{folderPreview}/
                                        </p>
                                    </div>
                                    <span class="inline-flex shrink-0 items-center gap-1 rounded-full bg-muted-foreground/10 px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                                        Draft
                                    </span>
                                </div>

                                <div class="space-y-2 border-t border-border pt-3">
                                    <div class="flex items-center justify-between gap-3 text-xs">
                                        <span class="text-muted-foreground/60">Artifact</span>
                                        {#if artifactVersion}
                                            <span class="flex min-w-0 items-center gap-1.5">
                                                <span class="font-mono text-foreground">{artifactVersion}</span>
                                                {#if selectedArtifactInstalled}
                                                    <span class="rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[9px] font-semibold text-emerald-500">
                                                        Ready
                                                    </span>
                                                {:else}
                                                    <span class="rounded-full bg-muted-foreground/10 px-1.5 py-0.5 text-[9px] font-semibold text-muted-foreground">
                                                        Will download
                                                    </span>
                                                {/if}
                                            </span>
                                        {:else}
                                            <span class="text-muted-foreground/40">Not selected</span>
                                        {/if}
                                    </div>
                                    <div class="flex items-center justify-between gap-3 text-xs">
                                        <span class="text-muted-foreground/60">Host</span>
                                        <span class="inline-flex items-center gap-1.5 text-foreground">
                                            {#if hostOs === "windows"}
                                                <WindowsIcon class="h-3 w-3 text-primary" />
                                            {:else if hostOs === "linux"}
                                                <LinuxIcon class="h-3 w-3 text-primary" />
                                            {/if}
                                            {hostOsLabel}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </section>

                        <section>
                            <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                                <ListChecks size={14} />
                                What happens next
                            </h2>
                            <ol class="space-y-3">
                                <li class="flex gap-3">
                                    <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary">1</span>
                                    <div>
                                        <p class="text-xs font-medium text-foreground">Folder is created</p>
                                        <p class="mt-0.5 text-[11px] text-muted-foreground/60">
                                            A clean <code class="font-mono text-[10px]">server.toml</code> lands in the server folder.
                                        </p>
                                    </div>
                                </li>
                                <li class="flex gap-3">
                                    <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary">2</span>
                                    <div>
                                        <p class="text-xs font-medium text-foreground">Artifact is linked</p>
                                        <p class="mt-0.5 text-[11px] text-muted-foreground/60">
                                            {#if selectedArtifactInstalled}
                                                Already installed — instant link.
                                            {:else if artifactVersion}
                                                Downloaded from the FiveM runtime, then shared with every future server.
                                            {:else}
                                                Downloaded on first use if needed, shared across all servers.
                                            {/if}
                                        </p>
                                    </div>
                                </li>
                                <li class="flex gap-3">
                                    <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary">3</span>
                                    <div>
                                        <p class="text-xs font-medium text-foreground">You land on the dashboard</p>
                                        <p class="mt-0.5 text-[11px] text-muted-foreground/60">
                                            Console, players, resources — ready to configure.
                                        </p>
                                    </div>
                                </li>
                            </ol>
                        </section>
                    </aside>
                </div>
            {:else}
                <div class="mx-auto max-w-md rounded-lg border border-border bg-card p-8 text-center">
                    <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                        <ShieldAlert size={20} class="text-destructive" />
                    </div>
                    <h1 class="text-lg font-semibold text-foreground">
                        No servers available
                    </h1>
                    <p class="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
                        This account can't see any servers and doesn't have permission to create one. Ask an owner to grant you access.
                    </p>
                </div>
            {/if}
        </div>
    </div>
{:else}
    {#key dashboardState.revision}
        <GridStack
            bind:items={dashboardState.layout}
            columns={12}
            cellHeight={80}
            margin={4}
            onchange={handleLayoutChange}
        >
            {#each dashboardState.layout as item (item.id)}
                {@const widget = widgetMap[item.id]}
                {@const def = getWidgetDef(item.id)}
                {#if widget}
                    <GridWidget
                        id={item.id}
                        title={widget.title}
                        x={item.x}
                        y={item.y}
                        w={item.w}
                        h={item.h}
                        minW={def?.minW ?? 2}
                        minH={def?.minH ?? 2}
                        onremove={handleRemove}
                    >
                        <widget.component />
                    </GridWidget>
                {/if}
            {/each}
        </GridStack>
    {/key}
{/if}

<style>
    @keyframes progress-indeterminate {
        0% {
            left: -35%;
        }
        100% {
            left: 100%;
        }
    }

    :global(.progress-indeterminate) {
        animation: progress-indeterminate 1.4s cubic-bezier(0.65, 0, 0.35, 1) infinite;
    }
</style>
