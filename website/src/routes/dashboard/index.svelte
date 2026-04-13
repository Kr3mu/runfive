<script lang="ts">
    import { createQuery, useQueryClient } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import { artifactsQueryOptions } from "$lib/api/artifacts";
    import { createServer, serversQueryOptions } from "$lib/api/servers";
    import GridStack from "$lib/components/dashboard/grid-stack.svelte";
    import GridWidget from "$lib/components/dashboard/grid-widget.svelte";
    import Console from "$lib/components/dashboard/console.svelte";
    import PlayerList from "$lib/components/dashboard/player-list.svelte";
    import * as Select from "$lib/components/ui/select";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { canGlobal } from "$lib/permissions.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import { getWidgetDef } from "$lib/widget-registry";
    import type { GridLayoutItem } from "$lib/types/grid-layout";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Rocket from "@lucide/svelte/icons/rocket";
    import Server from "@lucide/svelte/icons/server";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";
    import Boxes from "@lucide/svelte/icons/boxes";
    import HardDriveDownload from "@lucide/svelte/icons/hard-drive-download";
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
    const upstreamOnlyArtifacts = $derived(
        (artifactsQuery.data?.available ?? []).filter((artifact) => !artifact.installed),
    );

    let serverName = $state("");
    let artifactVersion = $state("");
    let isCreatingServer = $state(false);

    const selectedArtifactLabel = $derived.by(() => {
        if (!artifactVersion) return "Choose an artifact version";
        if (installedArtifacts.find((artifact) => artifact.version === artifactVersion)) {
            return `${artifactVersion} · Installed locally`;
        }
        return `${artifactVersion} · Download on create`;
    });

    $effect((): void => {
        if (!shouldLoadArtifacts || artifactVersion || !artifactsQuery.data) return;
        artifactVersion =
            artifactsQuery.data.installed[0]?.version ??
            artifactsQuery.data.available[0]?.version ??
            "";
    });

    function handleLayoutChange(items: GridLayoutItem[]): void {
        // TODO: persist to backend/localStorage
    }

    function handleRemove(id: string): void {
        dashboardState.removeWidget(id);
    }

    async function handleCreateServer(): Promise<void> {
        if (isCreatingServer) return;
        if (!serverName.trim()) {
            toast.error("Enter a server name first");
            return;
        }
        if (!artifactVersion) {
            toast.error("Choose an artifact version first");
            return;
        }

        isCreatingServer = true;
        try {
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
        } catch (error) {
            const message =
                error instanceof Error ? error.message : "Failed to create server";
            toast.error(message);
        } finally {
            isCreatingServer = false;
        }
    }
</script>

{#if authQuery.isLoading || serversQuery.isPending}
    <div class="flex h-full items-center justify-center">
        <LoaderCircle size={20} class="animate-spin text-muted-foreground" />
    </div>
{:else if servers.length === 0}
    <div class="flex h-full items-center justify-center overflow-y-auto px-6 py-10">
        {#if canCreateServers}
            <section class="w-full max-w-3xl rounded-[2rem] border border-border/60 bg-card/90 p-6 shadow-[0_24px_80px_-40px_rgba(0,0,0,0.65)] backdrop-blur-sm">
                <div class="mb-8 flex flex-wrap items-start justify-between gap-4">
                    <div class="max-w-xl">
                        <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/8 px-3 py-1 text-[11px] font-semibold tracking-[0.18em] text-primary uppercase">
                            <Rocket size={12} />
                            First Server Setup
                        </div>
                        <h1 class="font-heading text-3xl font-semibold text-foreground">
                            Build the panel around your first FiveM server
                        </h1>
                        <p class="mt-2 text-sm leading-6 text-muted-foreground">
                            RunFive now reads real servers from the `servers` directory and shares artifact installs from the host cache. Start with a name and the artifact build you want this server to reference.
                        </p>
                    </div>

                    <div class="rounded-3xl border border-border/60 bg-background/70 px-4 py-3">
                        <div class="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground/50 uppercase">
                            Host Artifact Tree
                        </div>
                        <div class="mt-1 flex items-center gap-2 text-sm font-medium text-foreground">
                            <Boxes size={15} class="text-primary" />
                            {artifactsQuery.data?.os ?? "Loading..."}
                        </div>
                    </div>
                </div>

                <div class="grid gap-6 lg:grid-cols-[1.25fr_0.95fr]">
                    <div class="rounded-[1.5rem] border border-border/50 bg-background/60 p-5">
                        <label class="mb-2 block text-[11px] font-semibold tracking-[0.14em] text-muted-foreground/55 uppercase" for="server-name">
                            Server Name
                        </label>
                        <input
                            id="server-name"
                            bind:value={serverName}
                            type="text"
                            maxlength="64"
                            placeholder="RunFive RP"
                            class="h-11 w-full rounded-3xl border border-border/70 bg-background px-4 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground/35 focus:border-primary/40"
                        />
                        <p class="mt-2 text-xs text-muted-foreground/55">
                            The folder name is generated automatically and spaces are converted to `_`.
                        </p>

                        <div class="mt-5">
                            <p class="mb-2 block text-[11px] font-semibold tracking-[0.14em] text-muted-foreground/55 uppercase">
                                Artifact Version
                            </p>

                            {#if artifactsQuery.isPending}
                                <div class="flex h-11 items-center gap-2 rounded-3xl border border-border/60 bg-background px-4 text-sm text-muted-foreground/60">
                                    <LoaderCircle size={14} class="animate-spin" />
                                    Loading artifact catalog...
                                </div>
                            {:else if artifactsQuery.error}
                                <div class="rounded-3xl border border-destructive/20 bg-destructive/6 px-4 py-3 text-sm text-destructive/80">
                                    {artifactsQuery.error.message}
                                </div>
                            {:else}
                                <Select.Root type="single" bind:value={artifactVersion}>
                                    <Select.Trigger class="h-11 w-full rounded-3xl border border-border/70 bg-background px-4 text-left">
                                        <span class={artifactVersion ? "text-foreground" : "text-muted-foreground/45"}>
                                            {selectedArtifactLabel}
                                        </span>
                                    </Select.Trigger>
                                    <Select.Content class="max-h-80">
                                        {#if installedArtifacts.length > 0}
                                            <Select.Group>
                                                <Select.GroupHeading>Installed</Select.GroupHeading>
                                                {#each installedArtifacts as artifact (artifact.version)}
                                                    <Select.Item value={artifact.version}>
                                                        <div class="flex w-full items-center justify-between gap-3">
                                                            <span>{artifact.version}</span>
                                                            <span class="text-xs text-emerald-500/80">Ready</span>
                                                        </div>
                                                    </Select.Item>
                                                {/each}
                                            </Select.Group>
                                        {/if}

                                        {#if upstreamOnlyArtifacts.length > 0}
                                            <Select.Group>
                                                <Select.GroupHeading>Available Upstream</Select.GroupHeading>
                                                {#each upstreamOnlyArtifacts as artifact (artifact.version)}
                                                    <Select.Item value={artifact.version}>
                                                        <div class="flex w-full items-center justify-between gap-3">
                                                            <span>{artifact.version}</span>
                                                            <span class="text-xs text-muted-foreground/55">Download</span>
                                                        </div>
                                                    </Select.Item>
                                                {/each}
                                            </Select.Group>
                                        {/if}
                                    </Select.Content>
                                </Select.Root>
                            {/if}
                        </div>

                        <div class="mt-6 flex flex-wrap items-center gap-3">
                            <button
                                onclick={handleCreateServer}
                                disabled={isCreatingServer || artifactsQuery.isPending || !artifactVersion}
                                class="inline-flex h-11 items-center gap-2 rounded-3xl bg-primary px-5 text-sm font-semibold text-primary-foreground transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {#if isCreatingServer}
                                    <LoaderCircle size={14} class="animate-spin" />
                                    Creating server...
                                {:else}
                                    <Server size={15} />
                                    Create Server
                                {/if}
                            </button>
                            <span class="text-xs text-muted-foreground/55">
                                If the build is not installed yet, RunFive downloads it before writing `server.toml`.
                            </span>
                        </div>
                    </div>

                    <div class="rounded-[1.5rem] border border-border/50 bg-gradient-to-br from-background via-background to-primary/6 p-5">
                        <h2 class="flex items-center gap-2 text-sm font-semibold text-foreground">
                            <HardDriveDownload size={15} class="text-primary" />
                            Shared Artifacts
                        </h2>
                        <p class="mt-2 text-sm leading-6 text-muted-foreground">
                            Artifacts are extracted once under
                            <code>artifacts/{artifactsQuery.data?.os ?? "host"}/&lt;version&gt;</code>
                            and every managed server points at that shared install through <code>artifact_version</code>.
                        </p>

                        <div class="mt-5 space-y-3">
                            <div class="rounded-2xl border border-border/50 bg-background/75 p-4">
                                <div class="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground/45 uppercase">
                                    Installed Locally
                                </div>
                                <div class="mt-2 text-2xl font-semibold text-foreground">
                                    {installedArtifacts.length}
                                </div>
                            </div>
                            <div class="rounded-2xl border border-border/50 bg-background/75 p-4">
                                <div class="text-[10px] font-semibold tracking-[0.16em] text-muted-foreground/45 uppercase">
                                    Upstream Choices
                                </div>
                                <div class="mt-2 text-2xl font-semibold text-foreground">
                                    {artifactsQuery.data?.available.length ?? 0}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        {:else}
            <section class="w-full max-w-xl rounded-[2rem] border border-border/60 bg-card/90 p-8 text-center shadow-[0_24px_80px_-40px_rgba(0,0,0,0.65)] backdrop-blur-sm">
                <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
                    <ShieldAlert size={28} class="text-destructive" />
                </div>
                <h1 class="mt-5 font-heading text-2xl font-semibold text-foreground">
                    No servers are visible to this account
                </h1>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">
                    RunFive did not discover any accessible entries under the `servers` directory, and this account does not have permission to create a new one.
                </p>
            </section>
        {/if}
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
