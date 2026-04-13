<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import { serversQueryOptions } from "$lib/api/servers";
    import CreateServerForm from "$lib/components/dashboard/create-server-form.svelte";
    import GridStack from "$lib/components/dashboard/grid-stack.svelte";
    import GridWidget from "$lib/components/dashboard/grid-widget.svelte";
    import Console from "$lib/components/dashboard/console.svelte";
    import PlayerList from "$lib/components/dashboard/player-list.svelte";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { canGlobal } from "$lib/permissions.svelte";
    import { getWidgetDef } from "$lib/widget-registry";
    import type { GridLayoutItem } from "$lib/types/grid-layout";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";

    const widgetMap: Record<string, { component: typeof Console; title: string }> = {
        console: { component: Console, title: "Console" },
        players: { component: PlayerList, title: "Players" },
    };

    const authQuery = createQuery(() => authQueryOptions());
    const serversQuery = createQuery(() => serversQueryOptions());

    const currentUser = $derived(authQuery.data);
    const servers = $derived(serversQuery.data ?? []);
    const canCreateServers = $derived(canGlobal(currentUser, "servers", "create"));

    function handleLayoutChange(_items: GridLayoutItem[]): void {
        // TODO: persist to backend/localStorage
    }

    function handleRemove(id: string): void {
        dashboardState.removeWidget(id);
    }
</script>

{#if authQuery.isLoading || serversQuery.isPending}
    <div class="flex h-full items-center justify-center">
        <LoaderCircle size={20} class="animate-spin text-muted-foreground" />
    </div>
{:else if servers.length === 0}
    <div class="flex h-full flex-col overflow-y-auto">
        <div class="mx-auto w-full max-w-2xl px-6 py-8">
            {#if canCreateServers}
                <CreateServerForm />
            {:else}
                <div class="mx-auto max-w-md rounded-lg border border-border bg-card p-8 text-center">
                    <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                        <ShieldAlert size={20} class="text-destructive" />
                    </div>
                    <h1 class="font-heading text-lg font-semibold text-foreground">
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
