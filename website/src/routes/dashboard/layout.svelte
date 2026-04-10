<script lang="ts">
    import type { Snippet } from "svelte";
    import Sidebar from "$lib/components/dashboard/sidebar.svelte";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import CircleAlert from "@lucide/svelte/icons/circle-alert";
    import X from "@lucide/svelte/icons/x";

    interface Props {
        children: Snippet;
    }

    let { children }: Props = $props();
    let sidebarCollapsed = $state(false);

    const dashboardErrors: Record<string, string> = {
        link_failed: "Failed to link your Cfx.re account. Please try again.",
    };

    const urlParams: URLSearchParams = new URLSearchParams(window.location.search);
    const urlError: string | null = urlParams.get("error");
    let dashboardError = $state<string | null>(
        urlError && dashboardErrors[urlError] ? dashboardErrors[urlError] : null,
    );

    if (urlError) {
        history.replaceState(null, "", window.location.pathname);
    }
</script>

<div class="flex h-dvh w-dvw overflow-hidden bg-background">
    <Sidebar
        bind:collapsed={sidebarCollapsed}
        activeWidgets={dashboardState.activeWidgetIds}
        onaddwidget={(id) => dashboardState.addWidget(id)}
    />

    <div class="relative flex-1 overflow-hidden border-l border-border">
        {#if dashboardError}
            <div class="absolute top-0 right-0 left-0 z-40 flex items-center gap-2.5 border-b border-destructive/20 bg-destructive/5 px-4 py-2.5 backdrop-blur-sm"
                 style="animation: slide-down 0.35s cubic-bezier(0.16, 1, 0.3, 1) both;"
            >
                <CircleAlert size={14} class="shrink-0 text-destructive" />
                <p class="flex-1 text-xs text-foreground">{dashboardError}</p>
                <button
                    onclick={() => (dashboardError = null)}
                    class="shrink-0 text-muted-foreground/40 transition-colors hover:text-foreground"
                    aria-label="Dismiss"
                >
                    <X size={13} />
                </button>
            </div>
        {/if}
        {@render children()}
    </div>
</div>

<style>
    @keyframes slide-down {
        from { opacity: 0; transform: translateY(-100%); }
        to { opacity: 1; transform: translateY(0); }
    }
</style>
