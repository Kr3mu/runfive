<script lang="ts">
    import type { Snippet } from "svelte";
    import Sidebar from "$lib/components/dashboard/sidebar.svelte";
    import { dashboardState } from "$lib/dashboard-state.svelte";

    interface Props {
        children: Snippet;
    }

    let { children }: Props = $props();
    let sidebarCollapsed = $state(false);
</script>

<div class="flex h-dvh w-dvw overflow-hidden bg-background">
    <Sidebar
        bind:collapsed={sidebarCollapsed}
        activeWidgets={dashboardState.activeWidgetIds}
        onaddwidget={(id) => dashboardState.addWidget(id)}
    />

    <div class="relative flex-1 overflow-hidden border-l border-border">
        {@render children()}
    </div>
</div>
