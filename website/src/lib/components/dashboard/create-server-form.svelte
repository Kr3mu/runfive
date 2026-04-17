<script lang="ts">
    import { createQuery, useQueryClient } from "@tanstack/svelte-query";
    import {
        artifactsQueryOptions,
        downloadArtifact,
        type AvailableArtifactVersion,
    } from "$lib/api/artifacts";
    import {
        createServer,
        serversQueryOptions,
        type ManagedServer,
    } from "$lib/api/servers";
    import LinuxIcon from "$lib/components/icons/linux.svelte";
    import WindowsIcon from "$lib/components/icons/windows.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Search from "@lucide/svelte/icons/search";
    import Check from "@lucide/svelte/icons/check";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import Sparkles from "@lucide/svelte/icons/sparkles";
    import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
    import ExternalLink from "@lucide/svelte/icons/external-link";
    import { toast } from "svelte-sonner";

    interface Props {
        /** Heading rendered above the form. Defaults to first-time onboarding copy. */
        heading?: string;
        /** Subtitle rendered under the heading. Defaults to first-time onboarding copy. */
        subtitle?: string;
        /** Fired after a server has been created successfully. Parent decides what
         *  to do next — e.g. navigate back to the main dashboard. */
        oncreated?: (server: ManagedServer) => void;
    }

    let {
        heading = "Create your first server",
        subtitle = "Give it a name and pick a FiveM build. Everything else is wired up for you.",
        oncreated,
    }: Props = $props();

    type CreationPhase = "idle" | "downloading" | "creating";

    /** Port range mirrors the backend allow-list in serverfs/registry.go. */
    const MIN_PORT = 1024;
    const MAX_PORT = 65535;
    const DEFAULT_BASE_PORT = 30120;
    /** Slot-count bounds mirror resolveMaxPlayers in serverfs/registry.go. */
    const MIN_MAX_PLAYERS = 1;
    const MAX_MAX_PLAYERS = 2048;
    const DEFAULT_MAX_PLAYERS = 32;

    const queryClient = useQueryClient();
    const artifactsQuery = createQuery(() => artifactsQueryOptions());
    const serversQuery = createQuery(() => serversQueryOptions());

    const installedArtifacts = $derived(artifactsQuery.data?.installed ?? []);
    const hostOs = $derived(artifactsQuery.data?.os);
    const hostOsLabel = $derived.by((): string => {
        if (!hostOs) return "Loading...";
        if (hostOs === "windows") return "Windows";
        if (hostOs === "linux") return "Linux";
        return hostOs;
    });

    let serverName = $state("");
    let artifactVersion = $state("");
    let licenseKey = $state("");
    /** Raw input string so the user can type freely; coerced on submit. */
    let portInput = $state("");
    let maxPlayersInput = $state(String(DEFAULT_MAX_PLAYERS));

    const licenseKeyInvalid = $derived(
        licenseKey.trim().length > 0 && !licenseKey.trim().startsWith("cfxk_"),
    );

    /** Map of currently-claimed ports → server name, for conflict detection. */
    const portOwners = $derived.by((): Map<number, string> => {
        const entries: ManagedServer[] = serversQuery.data ?? [];
        const map = new Map<number, string>();
        for (const server of entries) {
            if (server.port > 0) map.set(server.port, server.name);
        }
        return map;
    });

    /** First port ≥ DEFAULT_BASE_PORT that is not in portOwners. */
    const nextFreePort = $derived.by((): number => {
        for (let p: number = DEFAULT_BASE_PORT; p <= MAX_PORT; p += 1) {
            if (!portOwners.has(p)) return p;
        }
        return DEFAULT_BASE_PORT;
    });

    /** Numeric view of the user's input. NaN when empty/unparseable. */
    const portValue = $derived.by((): number => {
        const trimmed: string = portInput.trim();
        if (trimmed === "") return Number.NaN;
        const parsed: number = Number(trimmed);
        return Number.isInteger(parsed) ? parsed : Number.NaN;
    });

    const portOutOfRange = $derived(
        !Number.isNaN(portValue) && (portValue < MIN_PORT || portValue > MAX_PORT),
    );
    const portConflictOwner = $derived.by((): string | null => {
        if (Number.isNaN(portValue)) return null;
        return portOwners.get(portValue) ?? null;
    });
    const portInvalid = $derived(portOutOfRange || portConflictOwner !== null);

    /** Numeric view of the max-players input; NaN when empty/unparseable. */
    const maxPlayersValue = $derived.by((): number => {
        const trimmed: string = maxPlayersInput.trim();
        if (trimmed === "") return Number.NaN;
        const parsed: number = Number(trimmed);
        return Number.isInteger(parsed) ? parsed : Number.NaN;
    });

    const maxPlayersOutOfRange = $derived(
        !Number.isNaN(maxPlayersValue) &&
            (maxPlayersValue < MIN_MAX_PLAYERS || maxPlayersValue > MAX_MAX_PLAYERS),
    );
    const maxPlayersInvalid = $derived(
        Number.isNaN(maxPlayersValue) || maxPlayersOutOfRange,
    );

    /** Pre-fill the port once the servers query resolves; user edits win. */
    $effect((): void => {
        if (portInput !== "") return;
        if (serversQuery.data === undefined) return;
        portInput = String(nextFreePort);
    });

    /**
     * Per-field visual state for the status dots next to each group label.
     * `warning` is an amber state that lets submit proceed after explicit
     * user acknowledgement (used for port collisions where the operator
     * might genuinely want to override).
     */
    type FieldState = "empty" | "valid" | "invalid" | "pending" | "warning";

    const nameState = $derived<FieldState>(serverName.trim() ? "valid" : "empty");

    const licenseState = $derived<FieldState>(
        licenseKeyInvalid
            ? "invalid"
            : licenseKey.trim()
              ? "valid"
              : "empty",
    );

    const portState = $derived<FieldState>(
        portOutOfRange || Number.isNaN(portValue)
            ? "invalid"
            : portConflictOwner
              ? "warning"
              : "valid",
    );

    const networkState = $derived<FieldState>(
        portState === "invalid" || maxPlayersInvalid
            ? "invalid"
            : portState === "warning"
              ? "warning"
              : "valid",
    );

    const artifactState = $derived<FieldState>(
        selectedArtifactBroken
            ? "invalid"
            : !artifactVersion
              ? "empty"
              : selectedArtifactInstalled
                ? "valid"
                : "pending",
    );

    /** Tailwind classes for the small status dot per field state. */
    const dotClass: Record<FieldState, string> = {
        empty: "bg-muted-foreground/30",
        valid: "bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]",
        invalid: "bg-destructive shadow-[0_0_6px_rgba(239,68,68,0.55)]",
        pending: "bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.55)]",
        warning: "bg-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.6)]",
    };

    /** Hard errors — submit is completely blocked while any of these are true. */
    const hasBlockingError = $derived(
        nameState !== "valid" ||
            licenseState === "invalid" ||
            networkState === "invalid" ||
            artifactState === "invalid" ||
            artifactState === "empty",
    );

    /** Soft warning — submit is allowed on a second, deliberate click. */
    const hasWarning = $derived(networkState === "warning");

    /** Everything fully green — just for the bottom-row "Ready" status copy. */
    const allValid = $derived(!hasBlockingError && !hasWarning);

    /**
     * Tracks whether the user has seen one shake-rejection and clicked again.
     * Reset to false whenever the port changes so a fresh conflict has to be
     * acknowledged again.
     */
    let portConflictAcknowledged = $state(false);
    let submitShaking = $state(false);

    $effect((): void => {
        portInput;
        portConflictAcknowledged = false;
    });

    let creationPhase = $state<CreationPhase>("idle");
    let phaseStartedAt = $state<number | null>(null);
    let elapsedMs = $state(0);

    const isCreatingServer = $derived(creationPhase !== "idle");

    const creationPhaseLabel = $derived.by((): string => {
        if (creationPhase === "downloading") return `Downloading build ${artifactVersion}`;
        if (creationPhase === "creating") return "Setting up your server";
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
        if (artifactVersion || !artifactsQuery.data) return;
        artifactVersion =
            artifactsQuery.data.recommended ??
            curatedOptions[0]?.version ??
            artifactsQuery.data.installed[0]?.version ??
            "";
    });

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
        if (licenseKeyInvalid) {
            toast.error("License key must start with cfxk_");
            return;
        }
        if (Number.isNaN(portValue)) {
            toast.error("Enter a port number");
            return;
        }
        if (portOutOfRange) {
            toast.error(`Port must be between ${MIN_PORT} and ${MAX_PORT}`);
            return;
        }
        if (Number.isNaN(maxPlayersValue)) {
            toast.error("Enter a max player count");
            return;
        }
        if (maxPlayersOutOfRange) {
            toast.error(
                `Max players must be between ${MIN_MAX_PLAYERS} and ${MAX_MAX_PLAYERS}`,
            );
            return;
        }
        // Soft warning: port is already taken. First click shakes to confirm
        // the user sees it; second click within the same edit proceeds.
        if (portConflictOwner && !portConflictAcknowledged) {
            submitShaking = true;
            window.setTimeout((): void => {
                submitShaking = false;
            }, 450);
            portConflictAcknowledged = true;
            toast.warning(
                `Port ${portValue} is used by ${portConflictOwner}. Click Create again to proceed anyway.`,
            );
            return;
        }

        try {
            if (!selectedArtifactInstalled) {
                beginPhase("downloading");
                await downloadArtifact(artifactVersion);
                await queryClient.invalidateQueries({ queryKey: ["artifacts"] });
            }

            beginPhase("creating");
            const trimmedKey: string = licenseKey.trim();
            const created: ManagedServer = await createServer({
                name: serverName.trim(),
                artifactVersion,
                port: portValue,
                maxPlayers: maxPlayersValue,
                ...(trimmedKey ? { licenseKey: trimmedKey } : {}),
            });

            serverState.select(created.id);
            serverName = "";
            licenseKey = "";
            portInput = "";
            maxPlayersInput = String(DEFAULT_MAX_PLAYERS);

            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["servers"] }),
                queryClient.invalidateQueries({ queryKey: ["artifacts"] }),
            ]);

            toast.success(`Server ${created.name} is ready`);
            oncreated?.(created);
        } catch (error: unknown) {
            const message: string =
                error instanceof Error ? error.message : "Failed to create server";
            toast.error(message);
        } finally {
            resetPhase();
        }
    }
</script>

<div class="mb-8">
    <h1 class="mb-1 font-heading text-lg font-semibold text-foreground">
        {heading}
    </h1>
    <p class="text-sm text-muted-foreground">
        {subtitle}
    </p>
</div>

<!-- Unified config card: four field groups, two-column layout with status dots. -->
<div class="mb-6 rounded-xl border border-border bg-card">
    <!-- Server name -->
    <div class="grid grid-cols-1 gap-5 p-5 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <div class="flex items-start gap-2.5">
            <span class="mt-[7px] h-2 w-2 shrink-0 rounded-full {dotClass[nameState]}" aria-hidden="true"></span>
            <div class="min-w-0">
                <label for="server-name" class="block text-sm font-medium text-foreground">
                    Server name
                </label>
                <p class="mt-1 text-[12px] leading-snug text-muted-foreground/70">
                    Shows up in the panel and becomes the folder slug on disk.
                </p>
            </div>
        </div>
        <div>
            <input
                id="server-name"
                bind:value={serverName}
                type="text"
                maxlength="64"
                placeholder="RunFive RP"
                class="h-10 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground/40 focus:border-primary/50"
            />
            <p class="mt-1.5 flex items-center gap-1.5 text-[11px] text-muted-foreground/60">
                <span>Folder</span>
                <code class="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] text-foreground/70">servers/{folderPreview}/</code>
            </p>
        </div>
    </div>

    <div class="border-t border-border/60"></div>

    <!-- License key -->
    <div class="grid grid-cols-1 gap-5 p-5 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <div class="flex items-start gap-2.5">
            <span class="mt-[7px] h-2 w-2 shrink-0 rounded-full {dotClass[licenseState]}" aria-hidden="true"></span>
            <div class="min-w-0">
                <div class="flex items-baseline gap-2">
                    <label for="license-key" class="text-sm font-medium text-foreground">
                        License key
                    </label>
                    <span class="text-[11px] text-muted-foreground/60">Optional</span>
                </div>
                <p class="mt-1 text-[12px] leading-snug text-muted-foreground/70">
                    Cfx.re key, encrypted by the panel before it touches disk.
                </p>
            </div>
        </div>
        <div>
            <input
                id="license-key"
                bind:value={licenseKey}
                type="text"
                autocomplete="off"
                spellcheck="false"
                placeholder="cfxk_..."
                class="h-10 w-full rounded-md border bg-background px-3 font-mono text-sm text-foreground outline-none transition-colors placeholder:font-sans placeholder:text-muted-foreground/40 focus:border-primary/50 {licenseKeyInvalid
                    ? 'border-destructive/50'
                    : 'border-border'}"
            />
            <div class="mt-1.5 flex items-center justify-between gap-2 text-[11px]">
                {#if licenseKeyInvalid}
                    <p class="flex items-center gap-1.5 text-destructive/80">
                        <TriangleAlert size={11} class="shrink-0" />
                        <span>Must start with <code class="rounded bg-destructive/10 px-1 font-mono text-[10px]">cfxk_</code></span>
                    </p>
                {:else}
                    <p class="text-muted-foreground/60">Leave empty to add later.</p>
                {/if}
                <a
                    href="https://keymaster.fivem.net/"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex shrink-0 items-center gap-1 text-muted-foreground/60 transition-colors hover:text-foreground"
                >
                    Open keymaster
                    <ExternalLink size={10} />
                </a>
            </div>
        </div>
    </div>

    <div class="border-t border-border/60"></div>

    <!-- Network -->
    <div class="grid grid-cols-1 gap-5 p-5 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <div class="flex items-start gap-2.5">
            <span class="mt-[7px] h-2 w-2 shrink-0 rounded-full {dotClass[networkState]}" aria-hidden="true"></span>
            <div class="min-w-0">
                <div class="block text-sm font-medium text-foreground">Network</div>
                <p class="mt-1 text-[12px] leading-snug text-muted-foreground/70">
                    TCP/UDP endpoint port and server slot count.
                </p>
            </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
            <div>
                <label for="server-port" class="mb-1 block text-[11px] font-medium text-muted-foreground/70">
                    Port
                </label>
                <input
                    id="server-port"
                    bind:value={portInput}
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    placeholder={String(nextFreePort)}
                    class="no-spin h-10 w-full rounded-md border bg-background px-3 font-mono text-sm text-foreground outline-none transition-colors placeholder:font-sans placeholder:text-muted-foreground/40 focus:border-primary/50 {portOutOfRange ||
                    Number.isNaN(portValue)
                        ? 'border-destructive/50'
                        : portConflictOwner
                          ? 'border-amber-500/60'
                          : 'border-border'}"
                />
                {#if portConflictOwner}
                    <p class="mt-1.5 flex items-center gap-1 text-[11px] text-amber-600 dark:text-amber-500">
                        <TriangleAlert size={10} class="shrink-0" />
                        <span class="truncate">Used by <span class="font-medium">{portConflictOwner}</span></span>
                    </p>
                {:else if portOutOfRange}
                    <p class="mt-1.5 flex items-center gap-1 text-[11px] text-destructive/80">
                        <TriangleAlert size={10} class="shrink-0" />
                        <span>Out of range</span>
                    </p>
                {:else}
                    <p class="mt-1.5 text-[11px] text-muted-foreground/60">TCP + UDP</p>
                {/if}
            </div>
            <div>
                <label for="server-max-players" class="mb-1 block text-[11px] font-medium text-muted-foreground/70">
                    Max players
                </label>
                <input
                    id="server-max-players"
                    bind:value={maxPlayersInput}
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    placeholder={String(DEFAULT_MAX_PLAYERS)}
                    class="no-spin h-10 w-full rounded-md border bg-background px-3 font-mono text-sm text-foreground outline-none transition-colors placeholder:font-sans placeholder:text-muted-foreground/40 focus:border-primary/50 {maxPlayersInvalid
                        ? 'border-destructive/50'
                        : 'border-border'}"
                />
                {#if maxPlayersOutOfRange}
                    <p class="mt-1.5 flex items-center gap-1 text-[11px] text-destructive/80">
                        <TriangleAlert size={10} class="shrink-0" />
                        <span>Out of range</span>
                    </p>
                {:else}
                    <p class="mt-1.5 text-[11px] text-muted-foreground/60">sv_maxclients</p>
                {/if}
            </div>
        </div>
    </div>

    <div class="border-t border-border/60"></div>

    <!-- Artifact build -->
    <div class="grid grid-cols-1 gap-5 p-5 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <div class="flex items-start gap-2.5">
            <span class="mt-[7px] h-2 w-2 shrink-0 rounded-full {dotClass[artifactState]}" aria-hidden="true"></span>
            <div class="min-w-0">
                <div class="block text-sm font-medium text-foreground">Artifact build</div>
                <p class="mt-1 text-[12px] leading-snug text-muted-foreground/70">
                    fxserver runtime. Missing builds are downloaded before the server is created.
                </p>
                {#if artifactsQuery.data}
                    <p class="mt-2 flex items-center gap-1.5 text-[11px] text-muted-foreground/60">
                        {#if hostOs === "windows"}
                            <WindowsIcon class="h-3 w-3 text-primary" />
                        {:else if hostOs === "linux"}
                            <LinuxIcon class="h-3 w-3 text-primary" />
                        {/if}
                        <span class="font-medium text-foreground/80">{hostOsLabel}</span>
                        <span class="text-muted-foreground/30">·</span>
                        <span>{installedArtifacts.length} installed</span>
                    </p>
                {/if}
            </div>
        </div>
        <div>
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
                        {:else if artifactVersion && artifactVersion === recommendedVersion}
                            <Sparkles size={12} class="shrink-0 text-emerald-500" />
                        {/if}
                        {#if artifactVersion}
                            <span class="truncate font-mono {selectedArtifactBroken ? 'text-destructive' : 'text-foreground'}">
                                {artifactVersion}
                            </span>
                            <span class="shrink-0 text-[11px] text-muted-foreground/50">
                                {#if selectedArtifactInstalled}
                                    · Ready
                                {:else}
                                    · Will download
                                {/if}
                            </span>
                        {:else}
                            <span class="text-muted-foreground/45">Choose a build</span>
                        {/if}
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

                    <div class="absolute top-full right-0 left-0 z-50 mt-2 overflow-hidden rounded-lg border border-border bg-popover shadow-[0_16px_48px_-24px_rgba(0,0,0,0.65)]">
                        <div class="border-b border-border p-2">
                            <div class="relative">
                                <Search size={13} class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted-foreground/50" />
                                <input
                                    bind:this={artifactSearchInput}
                                    bind:value={artifactSearch}
                                    type="text"
                                    placeholder="Search by build number..."
                                    onkeydown={handleSearchKeydown}
                                    class="h-8 w-full rounded-md bg-background pr-2 pl-7 text-sm text-foreground outline-none placeholder:text-muted-foreground/40"
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
                                    <div class="flex items-center gap-1.5 px-3 pt-1 pb-1.5 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">
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
                                                    <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-500">
                                                        <Sparkles size={9} />
                                                        Recommended
                                                    </span>
                                                {/if}
                                                {#if option.installed}
                                                    <span class="rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                                                        Installed
                                                    </span>
                                                {/if}
                                                {#if option.brokenReason}
                                                    <span class="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[10px] font-medium text-destructive">
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
                                Stability by jgscripts
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
        </div>
    </div>
</div>

<!-- Action row / progress panel -->
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
    </div>
{:else}
    <div class="flex items-center justify-between gap-3">
        <p class="flex items-center gap-2 text-[11px] text-muted-foreground/70">
            <span
                class="h-1.5 w-1.5 rounded-full {hasBlockingError
                    ? 'bg-muted-foreground/30'
                    : hasWarning
                      ? 'bg-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.6)]'
                      : 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]'}"
                aria-hidden="true"
            ></span>
            {hasBlockingError
                ? "Complete the fields above."
                : hasWarning
                  ? portConflictAcknowledged
                      ? "Port conflict acknowledged."
                      : `Port ${portValue} is in use.`
                  : "Ready to create."}
        </p>
        <button
            onclick={handleCreateServer}
            disabled={hasBlockingError || artifactsQuery.isPending}
            class="inline-flex shrink-0 items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground transition-all hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40 {!hasBlockingError &&
            !hasWarning
                ? 'shadow-[0_8px_24px_-12px_hsl(var(--primary)/0.6)]'
                : ''} {submitShaking ? 'btn-shake' : ''}"
        >
            Create Server
        </button>
    </div>
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

    /* Hide the browser's number-input spin buttons so the field matches the
       other monospace inputs in this form. */
    .no-spin::-webkit-outer-spin-button,
    .no-spin::-webkit-inner-spin-button {
        appearance: none;
        margin: 0;
    }
    .no-spin {
        -moz-appearance: textfield;
    }

    /* Confirmation shake for soft-warning submits (e.g. port already in use).
       One cycle, then the button returns to rest and the second click goes
       through. */
    @keyframes btn-shake {
        0%,
        100% {
            transform: translateX(0);
        }
        15% {
            transform: translateX(-5px);
        }
        30% {
            transform: translateX(5px);
        }
        45% {
            transform: translateX(-4px);
        }
        60% {
            transform: translateX(4px);
        }
        75% {
            transform: translateX(-2px);
        }
        90% {
            transform: translateX(2px);
        }
    }

    :global(.btn-shake) {
        animation: btn-shake 0.45s cubic-bezier(0.36, 0.07, 0.19, 0.97);
    }
</style>
