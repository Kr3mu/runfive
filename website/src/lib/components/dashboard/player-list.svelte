<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { playersQueryOptions, type Player } from "$lib/api/players";
    import { serverState } from "$lib/server-state.svelte";
    import Search from "@lucide/svelte/icons/search";
    import ArrowUpDown from "@lucide/svelte/icons/arrow-up-down";
    import ArrowUp from "@lucide/svelte/icons/arrow-up";
    import ArrowDown from "@lucide/svelte/icons/arrow-down";
    import Ban from "@lucide/svelte/icons/ban";
    import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
    import UserX from "@lucide/svelte/icons/user-x";
    import X from "@lucide/svelte/icons/x";
    import RefreshCw from "@lucide/svelte/icons/refresh-cw";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";

    const selectedServerId = $derived(serverState.selectedId);
    const players = createQuery(() => playersQueryOptions(selectedServerId));

    type SortKey = "name" | "ping";
    type SortDir = "asc" | "desc";

    let searchQuery = $state("");
    let sortKey = $state<SortKey>("name");
    let sortDir = $state<SortDir>("asc");
    let activeAction = $state<{ playerId: number; action: string } | null>(
        null,
    );

    const filteredPlayers = $derived(() => {
        const data: Player[] = players.data ?? [];
        let result = [...data];

        if (searchQuery) {
            const q = searchQuery.toLowerCase();
            result = result.filter(
                (p) =>
                    p.name.toLowerCase().includes(q) ||
                    p.discord.toLowerCase().includes(q) ||
                    p.license.toLowerCase().includes(q),
            );
        }

        result.sort((a, b) => {
            let cmp = 0;
            if (sortKey === "name") cmp = a.name.localeCompare(b.name);
            else if (sortKey === "ping") cmp = a.ping - b.ping;
            return sortDir === "asc" ? cmp : -cmp;
        });

        return result;
    });

    function toggleSort(key: SortKey): void {
        if (sortKey === key) {
            sortDir = sortDir === "asc" ? "desc" : "asc";
        } else {
            sortKey = key;
            sortDir = "asc";
        }
    }

    function handleAction(playerId: number, action: string): void {
        activeAction = { playerId, action };
        setTimeout(() => (activeAction = null), 2000);
    }

    function getSortIcon(key: SortKey) {
        if (sortKey !== key) return ArrowUpDown;
        return sortDir === "asc" ? ArrowUp : ArrowDown;
    }

    function getPingColor(ping: number): string {
        if (ping < 40) return "text-emerald-400";
        if (ping < 70) return "text-amber-400";
        return "text-red-400";
    }
</script>

{#snippet sortIcon(key: SortKey)}
    {@const Icon = getSortIcon(key)}<Icon size={10} />
{/snippet}

<div class="flex h-full flex-col overflow-hidden">
    <!-- Toolbar -->
    <div
        class="flex h-7 shrink-0 items-center justify-between border-b border-border/50 bg-card px-2"
    >
        <div class="flex items-center gap-1.5">
            <span
                class="rounded bg-primary/10 px-1.5 py-px font-mono text-[9px] font-bold text-primary"
            >
                {players.data?.length ?? 0} online
            </span>
            {#if players.isFetching}
                <LoaderCircle
                    size={10}
                    class="animate-spin text-muted-foreground/30"
                />
            {/if}
        </div>
        <div class="flex items-center gap-1">
            <div class="relative flex items-center">
                <Search
                    size={10}
                    class="pointer-events-none absolute left-1.5 text-muted-foreground/30"
                />
                <input
                    type="text"
                    bind:value={searchQuery}
                    placeholder="Search..."
                    class="h-5 w-32 rounded-md border border-border/50 bg-background pl-5 pr-5 text-[10px] text-foreground outline-none placeholder:text-muted-foreground/20 focus:border-primary/40 focus:ring-1 focus:ring-primary/20"
                />
                {#if searchQuery}
                    <button
                        onclick={() => (searchQuery = "")}
                        class="absolute right-1 text-muted-foreground/25 hover:text-foreground"
                    >
                        <X size={10} />
                    </button>
                {/if}
            </div>
            <button
                onclick={() => players.refetch()}
                class="rounded-md p-1 text-muted-foreground/25 transition-colors hover:bg-muted hover:text-foreground"
                title="Refresh"
            >
                <RefreshCw size={11} />
            </button>
        </div>
    </div>

    <!-- Content -->
    {#if players.isPending}
        <div class="flex flex-1 items-center justify-center">
            <LoaderCircle
                size={20}
                class="animate-spin text-muted-foreground/30"
            />
        </div>
    {:else if players.isError}
        <div class="flex flex-1 flex-col items-center justify-center gap-2">
            <p class="text-sm text-destructive/70">Failed to load players</p>
            <button
                onclick={() => players.refetch()}
                class="rounded-md border border-border bg-background px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
                Retry
            </button>
        </div>
    {:else}
        <div class="flex-1 overflow-auto bg-card">
            <table class="w-full">
                <thead class="sticky top-0 z-10">
                    <tr class="border-b border-border bg-card text-left">
                        <th class="py-2 pl-3 pr-0">
                            <span
                                class="inline-flex min-w-6 justify-center px-1.5 text-[11px] font-semibold text-muted-foreground/40"
                                >#</span
                            >
                        </th>
                        <th class="py-2 pl-2 pr-3">
                            <button
                                onclick={() => toggleSort("name")}
                                class="flex items-center gap-1 text-[11px] font-semibold tracking-wider text-muted-foreground/60 uppercase transition-colors hover:text-foreground"
                            >
                                Name
                                {@render sortIcon("name")}
                            </button>
                        </th>
                        <th class="px-3 py-2">
                            <span
                                class="text-[11px] font-semibold tracking-wider text-muted-foreground/60 uppercase"
                                >Discord</span
                            >
                        </th>
                        <th class="hidden px-3 py-2 xl:table-cell">
                            <span
                                class="text-[11px] font-semibold tracking-wider text-muted-foreground/60 uppercase"
                                >License</span
                            >
                        </th>
                        <th class="px-3 py-2">
                            <button
                                onclick={() => toggleSort("ping")}
                                class="flex items-center gap-1 text-[11px] font-semibold tracking-wider text-muted-foreground/60 uppercase transition-colors hover:text-foreground"
                            >
                                Ping
                                {@render sortIcon("ping")}
                            </button>
                        </th>
                        <th class="px-3 py-2 text-right">
                            <span
                                class="text-[11px] font-semibold tracking-wider text-muted-foreground/60 uppercase"
                                >Actions</span
                            >
                        </th>
                    </tr>
                </thead>
                <tbody>
                    {#each filteredPlayers() as player (player.id)}
                        {@const isActioned =
                            activeAction?.playerId === player.id}
                        <tr
                            class="group border-b border-border/30 transition-colors hover:bg-muted/20"
                        >
                            <td class="py-1.5 pl-3 pr-0">
                                <span
                                    class="inline-flex h-6 min-w-6 items-center justify-center rounded bg-primary/8 px-1.5 font-mono text-[11px] font-bold text-primary/70"
                                >
                                    {player.id}
                                </span>
                            </td>
                            <td class="py-1.5 pl-2 pr-3">
                                <span
                                    class="text-[13px] leading-6 font-medium text-foreground"
                                    >{player.name}</span
                                >
                            </td>
                            <td class="px-3 py-1.5">
                                <span class="text-xs text-muted-foreground/70"
                                    >{player.discord || "—"}</span
                                >
                            </td>
                            <td class="hidden px-3 py-1.5 xl:table-cell">
                                {#if player.license}
                                    <code
                                        class="rounded bg-muted/50 px-1.5 py-px font-mono text-[11px] text-muted-foreground/50"
                                    >
                                        {player.license.slice(0, 22)}{player
                                            .license.length > 22
                                            ? "…"
                                            : ""}
                                    </code>
                                {:else}
                                    <span
                                        class="text-xs text-muted-foreground/30"
                                        >—</span
                                    >
                                {/if}
                            </td>
                            <td class="px-3 py-1.5">
                                <span
                                    class="font-mono text-xs font-medium {getPingColor(
                                        player.ping,
                                    )}"
                                    >{player.ping}<span
                                        class="text-[10px] opacity-50">ms</span
                                    ></span
                                >
                            </td>
                            <td class="px-3 py-1.5">
                                {#if isActioned}
                                    <div class="flex justify-end">
                                        <span
                                            class="rounded bg-primary/12 px-1.5 py-0.5 font-mono text-[10px] font-bold text-primary uppercase"
                                        >
                                            {activeAction?.action}
                                        </span>
                                    </div>
                                {:else}
                                    <div
                                        class="flex items-center justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
                                    >
                                        <button
                                            onclick={() =>
                                                handleAction(
                                                    player.id,
                                                    "warned",
                                                )}
                                            class="rounded p-1 text-amber-500/40 transition-colors hover:bg-amber-500/10 hover:text-amber-400"
                                            title="Warn {player.name}"
                                        >
                                            <TriangleAlert size={13} />
                                        </button>
                                        <button
                                            onclick={() =>
                                                handleAction(
                                                    player.id,
                                                    "kicked",
                                                )}
                                            class="rounded p-1 text-orange-500/40 transition-colors hover:bg-orange-500/10 hover:text-orange-400"
                                            title="Kick {player.name}"
                                        >
                                            <UserX size={13} />
                                        </button>
                                        <button
                                            onclick={() =>
                                                handleAction(
                                                    player.id,
                                                    "banned",
                                                )}
                                            class="rounded p-1 text-red-500/40 transition-colors hover:bg-red-500/10 hover:text-red-400"
                                            title="Ban {player.name}"
                                        >
                                            <Ban size={13} />
                                        </button>
                                    </div>
                                {/if}
                            </td>
                        </tr>
                    {/each}
                    {#if filteredPlayers().length === 0}
                        <tr>
                            <td
                                colspan="6"
                                class="py-8 text-center text-xs text-muted-foreground/30"
                            >
                                No players found
                            </td>
                        </tr>
                    {/if}
                </tbody>
            </table>
        </div>
    {/if}
</div>
