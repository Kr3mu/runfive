<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { Popover } from "bits-ui";
    import Cpu from "@lucide/svelte/icons/cpu";
    import HardDrive from "@lucide/svelte/icons/hard-drive";
    import Activity from "@lucide/svelte/icons/activity";
    import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
    import Check from "@lucide/svelte/icons/check";
    import ServerCog from "@lucide/svelte/icons/server-cog";
    import {
        serversQueryOptions,
        type ManagedServer,
        type ServerStatus,
    } from "$lib/api/servers";
    import { serverState } from "$lib/server-state.svelte";

    interface Props {
        /** Collapsed sidebar shows only a status dot + player count */
        collapsed?: boolean;
    }

    let { collapsed = false }: Props = $props();

    let open = $state(false);

    const servers = createQuery(() => serversQueryOptions());

    const list = $derived(servers.data ?? []);
    const selected = $derived(serverState.resolve(list));

    function playerPercent(s: ManagedServer): number {
        if (s.maxPlayers === 0) return 0;
        return Math.round((s.playerCount / s.maxPlayers) * 100);
    }

    function ramLabel(mb: number): string {
        if (mb <= 0) return "—";
        if (mb < 1024) return `${mb}M`;
        return `${(mb / 1024).toFixed(1)}G`;
    }

    function tickLabel(ms: number): string {
        if (ms <= 0) return "—";
        return `${ms.toFixed(1)}ms`;
    }

    /** Dot colour + glow for a lifecycle state. */
    const statusDot: Record<ServerStatus, string> = {
        online: "bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]",
        starting: "bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.55)] animate-pulse",
        stopped: "bg-muted-foreground/30",
        crashed: "bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.55)]",
    };

    /** Sub-line text colour matching the dot. */
    const statusText: Record<ServerStatus, string> = {
        online: "text-emerald-400/70",
        starting: "text-amber-400/80",
        stopped: "text-muted-foreground/50",
        crashed: "text-red-400/80",
    };

    const statusLabel: Record<ServerStatus, string> = {
        online: "Online",
        starting: "Starting",
        stopped: "Offline",
        crashed: "Crashed",
    };

    function handleSelect(id: string): void {
        serverState.select(id);
        open = false;
    }
</script>

<Popover.Root bind:open>
    <Popover.Trigger
        class="w-full cursor-pointer rounded-lg text-left outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        title={collapsed ? (selected?.name ?? "No server selected") : undefined}
    >
        {#if collapsed}
            <div class="flex flex-col items-center gap-2 py-1">
                {#if selected}
                    <div class="h-2 w-2 rounded-full {statusDot[selected.status]}"></div>
                    <span class="font-mono text-[9px] font-bold tabular-nums text-primary">{selected.playerCount}</span>
                {:else}
                    <div class="h-2 w-2 rounded-full bg-muted-foreground/30"></div>
                {/if}
            </div>
        {:else if !selected}
            <div class="animate-pulse rounded-lg border border-border/50 bg-background/50 p-3">
                <div class="flex items-center gap-2">
                    <div class="h-2 w-2 rounded-full bg-muted-foreground/20"></div>
                    <div class="h-3 w-24 rounded bg-muted-foreground/10"></div>
                </div>
                <div class="mt-3 h-1 w-full rounded-full bg-muted-foreground/10"></div>
            </div>
        {:else}
            <div
                class="rounded-lg border border-border/50 bg-background/50 p-3 transition-all duration-200 hover:border-primary/30 hover:bg-background/80 hover:shadow-sm data-[state=open]:border-primary/40 data-[state=open]:bg-background/80"
            >
                <div class="mb-2 flex items-center justify-between">
                    <div class="flex min-w-0 items-center gap-2">
                        <div class="h-2 w-2 shrink-0 rounded-full {statusDot[selected.status]}"></div>
                        <span class="truncate font-heading text-[12px] font-semibold text-foreground">{selected.name}</span>
                    </div>
                    <ChevronsUpDown size={11} class="shrink-0 text-muted-foreground/40" />
                </div>

                <!-- Player bar -->
                <div class="mb-2.5">
                    <div class="mb-1 flex items-baseline justify-between">
                        <span class="text-[10px] text-muted-foreground">Players</span>
                        <span class="font-mono text-[10px] font-semibold tabular-nums text-foreground">
                            <span class="text-primary">{selected.playerCount}</span><span class="text-muted-foreground">/{selected.maxPlayers}</span>
                        </span>
                    </div>
                    <div class="h-1.25 overflow-hidden rounded-full bg-muted">
                        <div
                            class="h-full rounded-full bg-primary transition-all duration-500"
                            style="width: {playerPercent(selected)}%"
                        ></div>
                    </div>
                </div>

                <!-- Quick stats -->
                <div class="flex justify-between">
                    <div class="flex items-center gap-1">
                        <Cpu size={10} class="text-muted-foreground/50" />
                        <span class="font-mono text-[9px] font-medium tabular-nums text-emerald-400">{selected.cpu}%</span>
                    </div>
                    <div class="flex items-center gap-1">
                        <HardDrive size={10} class="text-muted-foreground/50" />
                        <span class="font-mono text-[9px] font-medium tabular-nums text-blue-400">{ramLabel(selected.ramMB)}</span>
                    </div>
                    <div class="flex items-center gap-1">
                        <Activity size={10} class="text-muted-foreground/50" />
                        <span class="font-mono text-[9px] font-medium tabular-nums text-primary">{tickLabel(selected.tickMs)}</span>
                    </div>
                </div>
            </div>
        {/if}
    </Popover.Trigger>

    <Popover.Content
        side="right"
        align="start"
        sideOffset={8}
        class="z-50 w-74rounded-xl border border-border/60 bg-popover/95 p-1.5 shadow-2xl shadow-black/30 ring-1 ring-foreground/5 outline-none backdrop-blur-md data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 data-[side=right]:slide-in-from-left-1"
    >
        <!-- Header -->
        <div class="flex items-center justify-between px-2 pt-1.5 pb-2">
            <span class="text-[9px] font-semibold tracking-[0.14em] text-muted-foreground/40 uppercase">
                Your Servers
            </span>
            <span class="font-mono text-[10px] tabular-nums text-muted-foreground/30">
                {list.length}
            </span>
        </div>

        <div class="mx-1 mb-1 h-px bg-border/40"></div>

        <!-- Server list -->
        <div class="flex max-h-80 flex-col gap-0.5 overflow-y-auto px-1">
            {#each list as server (server.id)}
                {@const isActive = selected?.id === server.id}
                <button
                    onclick={() => handleSelect(server.id)}
                    class="group relative w-full rounded-lg px-2 py-2 text-left transition-colors
                        {isActive ? 'bg-primary/8' : 'hover:bg-muted/40'}"
                >
                    {#if isActive}
                        <span class="absolute top-1/2 left-0 h-5 w-0.5 -translate-y-1/2 rounded-r-full bg-primary"></span>
                    {/if}

                    <!-- Row 1: at-a-glance -->
                    <div class="flex items-center gap-2.5">
                        <div class="h-2 w-2 shrink-0 rounded-full {statusDot[server.status]}"></div>

                        <span
                            class="flex-1 truncate font-heading text-[12.5px] font-semibold
                                {isActive ? 'text-foreground' : 'text-foreground/90'}"
                        >
                            {server.name}
                        </span>

                        <span class="shrink-0 font-mono text-[10.5px] tabular-nums">
                            <span class={isActive ? 'font-semibold text-primary' : 'font-medium text-foreground/70'}>
                                {server.playerCount}
                            </span><span class="text-muted-foreground/40">/{server.maxPlayers}</span>
                        </span>

                        <div class="flex h-3.5 w-3.5 shrink-0 items-center justify-center">
                            {#if isActive}
                                <Check size={12} class="text-primary" strokeWidth={2.5} />
                            {/if}
                        </div>
                    </div>

                    <!-- Row 2: sub-line status + address, aligned under name -->
                    <div class="mt-1 flex items-center gap-1.5 pl-4.5">
                        <span class="text-[9.5px] font-medium {statusText[server.status]}">
                            {statusLabel[server.status]}
                        </span>
                        <span class="text-[9px] text-muted-foreground/25">•</span>
                        <span class="truncate font-mono text-[9.5px] text-muted-foreground/40">
                            {server.address}
                        </span>
                    </div>
                </button>
            {/each}

            {#if list.length === 0 && !servers.isPending}
                <div class="px-3 py-8 text-center text-[11px] text-muted-foreground/40">
                    No servers configured
                </div>
            {/if}
        </div>

        <div class="mx-1 mt-1 h-px bg-border/40"></div>

        <!-- Footer -->
        <a
            href="/dashboard/settings"
            data-view-transition
            onclick={() => (open = false)}
            class="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-muted-foreground/60 transition-colors hover:bg-muted/40 hover:text-foreground"
        >
            <ServerCog size={13} strokeWidth={1.8} />
            <span class="text-[11px]">Manage servers</span>
        </a>
    </Popover.Content>
</Popover.Root>
