<script lang="ts">
    import GridStack from "$lib/components/dashboard/grid-stack.svelte";
    import GridWidget from "$lib/components/dashboard/grid-widget.svelte";
    import Console from "$lib/components/dashboard/console.svelte";
    import PlayerList from "$lib/components/dashboard/player-list.svelte";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { getWidgetDef } from "$lib/widget-registry";
    import type { GridLayoutItem } from "$lib/types/grid-layout";

    const widgetMap: Record<string, { component: typeof Console; title: string }> = {
        console: { component: Console, title: "Console" },
        players: { component: PlayerList, title: "Players" },
    };

    function handleLayoutChange(items: GridLayoutItem[]): void {
        // TODO: persist to backend/localStorage
    }

    function handleRemove(id: string): void {
        dashboardState.removeWidget(id);
    }
</script>

{#key dashboardState.revision}
    <GridStack
        bind:items={dashboardState.layout}
        columns={12}
        rowHeight={80}
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
