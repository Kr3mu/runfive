<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { Popover } from "bits-ui";
    import { slide } from "svelte/transition";
    import { cubicOut } from "svelte/easing";
    import Cpu from "@lucide/svelte/icons/cpu";
    import HardDrive from "@lucide/svelte/icons/hard-drive";
    import Activity from "@lucide/svelte/icons/activity";
    import ChevronsUpDown from "@lucide/svelte/icons/chevrons-up-down";
    import Check from "@lucide/svelte/icons/check";
    import Search from "@lucide/svelte/icons/search";
    import X from "@lucide/svelte/icons/x";
    import ServerCog from "@lucide/svelte/icons/server-cog";
    import {
        serversQueryOptions,
        type ManagedServer,
        type ServerStatus,
    } from "$lib/api/servers";
    import { serverState } from "$lib/server-state.svelte";

    interface Props {
        /** Collapsed sidebar shows a status dot and keeps the popover switcher. */
        collapsed?: boolean;
    }

    let { collapsed = false }: Props = $props();

    /** Popover open state — only used when `collapsed`. */
    let popoverOpen = $state(false);
    /** Inline expansion state — only used when the sidebar is not collapsed. */
    let inlineOpen = $state(false);
    let searchQuery = $state("");

    const servers = createQuery(() => serversQueryOptions());

    const list = $derived(servers.data ?? []);
    const selected = $derived(serverState.resolve(list));

    /** Servers appearing before the active one in list order (≤3 case). */
    const beforeActive = $derived(() => {
        if (!selected) return [];
        const idx = list.findIndex((s) => s.id === selected.id);
        return idx > 0 ? list.slice(0, idx) : [];
    });

    /** Servers appearing after the active one in list order (≤3 case). */
    const afterActive = $derived(() => {
        if (!selected) return [];
        const idx = list.findIndex((s) => s.id === selected.id);
        return idx >= 0 ? list.slice(idx + 1) : [];
    });

    /** >3 managed servers switches to search + scrollable list layout. */
    const needsSearch = $derived(list.length > 3);

    /** Everything except the active server — used by both the popover and the
     *  search/scrollable inline layout. */
    const others = $derived(list.filter((s) => s.id !== selected?.id));

    const filteredOthers = $derived(() => {
        const q = searchQuery.trim().toLowerCase();
        if (!q) return others;
        return others.filter(
            (s) =>
                s.name.toLowerCase().includes(q) ||
                s.address.toLowerCase().includes(q),
        );
    });

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

    const statusDot: Record<ServerStatus, string> = {
        online: "bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.55)]",
        starting: "bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.55)] animate-pulse",
        stopped: "bg-muted-foreground/30",
        crashed: "bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.55)]",
    };

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

    function toggleInline(): void {
        inlineOpen = !inlineOpen;
        if (!inlineOpen) searchQuery = "";
    }

    function handleSelect(id: string): void {
        serverState.select(id);
        inlineOpen = false;
        popoverOpen = false;
        searchQuery = "";
    }

    // Escape closes the inline expansion.
    $effect((): (() => void) => {
        if (!inlineOpen) return () => {};
        const onKey = (e: KeyboardEvent): void => {
            if (e.key === "Escape") {
                inlineOpen = false;
                searchQuery = "";
            }
        };
        window.addEventListener("keydown", onKey);
        return (): void => window.removeEventListener("keydown", onKey);
    });
</script>

{#snippet serverRow(server: ManagedServer)}
    <button
        onclick={() => handleSelect(server.id)}
        class="group w-full rounded-lg px-2 py-2 text-left transition-colors hover:bg-muted/40"
    >
        <div class="flex items-center gap-2">
            <div class="h-2 w-2 shrink-0 rounded-full {statusDot[server.status]}"></div>
            <span class="flex-1 truncate font-heading text-[11px] font-semibold text-foreground/90">
                {server.name}
            </span>
            <span class="shrink-0 font-mono text-[10px] tabular-nums">
                <span class="font-medium text-foreground/70">{server.playerCount}</span><span class="text-muted-foreground/40">/{server.maxPlayers}</span>
            </span>
        </div>
        <div class="mt-0.5 flex items-center gap-1.5 pl-[16px]">
            <span class="text-[9px] font-medium {statusText[server.status]}">
                {statusLabel[server.status]}
            </span>
            <span class="text-[8px] text-muted-foreground/25">•</span>
            <span class="truncate font-mono text-[9px] text-muted-foreground/40">
                {server.address}
            </span>
        </div>
    </button>
{/snippet}

{#snippet triggerCard()}
    {#if !selected}
        <div class="animate-pulse rounded-lg border border-border/50 bg-background/50 p-3">
            <div class="flex items-center gap-2">
                <div class="h-2 w-2 rounded-full bg-muted-foreground/20"></div>
                <div class="h-3 w-24 rounded bg-muted-foreground/10"></div>
            </div>
            <div class="mt-3 h-1 w-full rounded-full bg-muted-foreground/10"></div>
        </div>
    {:else}
        <div
            class="rounded-lg border bg-background/50 p-3 transition-all duration-200
                {inlineOpen
                    ? 'border-primary/40 bg-background/80 shadow-sm'
                    : 'border-border/50 hover:border-primary/30 hover:bg-background/80 hover:shadow-sm'}"
        >
            <div class="mb-2 flex items-center justify-between">
                <div class="flex min-w-0 items-center gap-2">
                    <div class="h-2 w-2 shrink-0 rounded-full {statusDot[selected.status]}"></div>
                    <span class="truncate font-heading text-[12px] font-semibold text-foreground">{selected.name}</span>
                </div>
                <ChevronsUpDown
                    size={11}
                    class="shrink-0 transition-colors {inlineOpen ? 'text-primary/70' : 'text-muted-foreground/40'}"
                />
            </div>

            <div class="mb-2.5">
                <div class="mb-1 flex items-baseline justify-between">
                    <span class="text-[10px] text-muted-foreground">Players</span>
                    <span class="font-mono text-[10px] font-semibold tabular-nums text-foreground">
                        <span class="text-primary">{selected.playerCount}</span><span class="text-muted-foreground">/{selected.maxPlayers}</span>
                    </span>
                </div>
                <div class="h-[5px] overflow-hidden rounded-full bg-muted">
                    <div
                        class="h-full rounded-full bg-primary transition-all duration-500"
                        style="width: {playerPercent(selected)}%"
                    ></div>
                </div>
            </div>

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
{/snippet}

{#if collapsed}
    <!-- Collapsed sidebar: popover switcher (unchanged behaviour) -->
    <Popover.Root bind:open={popoverOpen}>
        <Popover.Trigger
            class="w-full cursor-pointer rounded-lg text-left outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
            title={selected?.name ?? "No server selected"}
        >
            <div class="flex flex-col items-center gap-2 py-1">
                {#if selected}
                    <div class="h-2 w-2 rounded-full {statusDot[selected.status]}"></div>
                    <span class="font-mono text-[9px] font-bold tabular-nums text-primary">{selected.playerCount}</span>
                {:else}
                    <div class="h-2 w-2 rounded-full bg-muted-foreground/30"></div>
                {/if}
            </div>
        </Popover.Trigger>

        <Popover.Content
            side="right"
            align="start"
            sideOffset={8}
            class="z-50 w-[296px] rounded-xl border border-border/60 bg-popover/95 p-1.5 shadow-2xl shadow-black/30 ring-1 ring-foreground/5 outline-none backdrop-blur-md data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 data-[side=right]:slide-in-from-left-1"
        >
            <div class="flex items-center justify-between px-2 pt-1.5 pb-2">
                <span class="text-[9px] font-semibold tracking-[0.14em] text-muted-foreground/40 uppercase">
                    Your Servers
                </span>
                <span class="font-mono text-[10px] tabular-nums text-muted-foreground/30">{list.length}</span>
            </div>

            <div class="mx-1 mb-1 h-px bg-border/40"></div>

            <div class="flex max-h-[320px] flex-col gap-0.5 overflow-y-auto px-1">
                {#each list as server (server.id)}
                    {@const isActive = selected?.id === server.id}
                    <button
                        onclick={() => handleSelect(server.id)}
                        class="group relative w-full rounded-lg px-2 py-2 text-left transition-colors
                            {isActive ? 'bg-primary/8' : 'hover:bg-muted/40'}"
                    >
                        {#if isActive}
                            <span class="absolute top-1/2 left-0 h-5 w-[2px] -translate-y-1/2 rounded-r-full bg-primary"></span>
                        {/if}

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

                        <div class="mt-1 flex items-center gap-1.5 pl-[18px]">
                            <span class="text-[9.5px] font-medium {statusText[server.status]}">{statusLabel[server.status]}</span>
                            <span class="text-[9px] text-muted-foreground/25">•</span>
                            <span class="truncate font-mono text-[9.5px] text-muted-foreground/40">{server.address}</span>
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

            <a
                href="/dashboard/settings"
                data-view-transition
                onclick={() => (popoverOpen = false)}
                class="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-muted-foreground/60 transition-colors hover:bg-muted/40 hover:text-foreground"
            >
                <ServerCog size={13} strokeWidth={1.8} />
                <span class="text-[11px]">Manage servers</span>
            </a>
        </Popover.Content>
    </Popover.Root>
{:else}
    <!-- Expanded sidebar: inline switcher.
         The active card stays as the visual anchor; other servers fan out
         above/below it for ≤3, or collapse into a searchable scroll list
         when there are more. -->
    <div class="flex flex-col">
        <!-- Items before the active one (≤3 case) -->
        {#if inlineOpen && selected && !needsSearch && beforeActive().length > 0}
            <div
                transition:slide={{ duration: 180, easing: cubicOut }}
                class="flex flex-col gap-0.5 pb-2"
            >
                {#each beforeActive() as server (server.id)}
                    {@render serverRow(server)}
                {/each}
            </div>
        {/if}

        <!-- Active card / trigger -->
        <button
            onclick={toggleInline}
            class="w-full cursor-pointer rounded-lg text-left outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
            aria-expanded={inlineOpen}
            aria-haspopup="listbox"
        >
            {@render triggerCard()}
        </button>

        <!-- Items after the active one (≤3 case) -->
        {#if inlineOpen && selected && !needsSearch && afterActive().length > 0}
            <div
                transition:slide={{ duration: 180, easing: cubicOut }}
                class="flex flex-col gap-0.5 pt-2"
            >
                {#each afterActive() as server (server.id)}
                    {@render serverRow(server)}
                {/each}
            </div>
        {/if}

        <!-- Search + scrollable list (>3 case) -->
        {#if inlineOpen && selected && needsSearch}
            <div
                transition:slide={{ duration: 180, easing: cubicOut }}
                class="pt-2"
            >
                <div class="relative mb-1.5">
                    <Search size={11} class="pointer-events-none absolute top-1/2 left-2 -translate-y-1/2 text-muted-foreground/40" />
                    <input
                        bind:value={searchQuery}
                        type="text"
                        placeholder="Search servers..."
                        class="h-7 w-full rounded-md border border-border/50 bg-background/50 pr-6 pl-7 text-[11px] text-foreground outline-none placeholder:text-muted-foreground/40 focus:border-primary/40 focus:ring-1 focus:ring-primary/20"
                    />
                    {#if searchQuery}
                        <button
                            onclick={() => (searchQuery = "")}
                            class="absolute top-1/2 right-1.5 -translate-y-1/2 text-muted-foreground/40 hover:text-foreground"
                            aria-label="Clear search"
                        >
                            <X size={11} />
                        </button>
                    {/if}
                </div>

                <div class="flex max-h-[240px] flex-col gap-0.5 overflow-y-auto">
                    {#each filteredOthers() as server (server.id)}
                        {@render serverRow(server)}
                    {/each}

                    {#if filteredOthers().length === 0}
                        <div class="px-3 py-4 text-center text-[10px] text-muted-foreground/40">
                            No matches
                        </div>
                    {/if}
                </div>
            </div>
        {/if}
    </div>
{/if}
