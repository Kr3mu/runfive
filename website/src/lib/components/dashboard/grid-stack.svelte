<script lang="ts" module>
    import type { GridStack as GridStackType, GridStackWidget } from "gridstack";
    export type { GridLayoutItem } from "$lib/types/grid-layout";
    export type { GridStackType, GridStackWidget };
</script>

<script lang="ts">
    import { onMount, tick, type Snippet } from "svelte";
    import "gridstack/dist/gridstack.min.css";
    import { dashboardState } from "$lib/dashboard-state.svelte";

    interface Props {
        items: GridLayoutItem[];
        columns?: number;
        rows?: number;
        margin?: number;
        children: Snippet;
        onchange?: (items: GridLayoutItem[]) => void;
    }

    let {
        items = $bindable(),
        columns = 12,
        rows = 6,
        margin = 4,
        children,
        onchange,
    }: Props = $props();

    let containerEl = $state<HTMLElement | null>(null);
    let grid: GridStackType | null = null;
    let GridStackClass: typeof GridStackType | null = null;

    function serializeLayout(): GridLayoutItem[] {
        if (!grid) return items;
        const saved = grid.save(false) as GridStackWidget[];
        return saved.map((w) => ({
            id: w.id ?? "",
            x: w.x ?? 0,
            y: w.y ?? 0,
            w: w.w ?? 1,
            h: w.h ?? 1,
            minW: w.minW,
            minH: w.minH,
            maxW: w.maxW,
            maxH: w.maxH,
        }));
    }

    function computeCellHeight(): number {
        if (!containerEl) return 80;
        const available = containerEl.clientHeight;
        const totalMargin = rows * margin * 2;
        return Math.floor((available - totalMargin) / rows);
    }

    function initGrid(): void {
        if (!GridStackClass || !containerEl) return;

        // Destroy previous instance if any
        if (grid) {
            grid.offAll();
            grid.destroy(false);
            grid = null;
        }

        grid = GridStackClass.init(
            {
                column: columns,
                cellHeight: computeCellHeight(),
                margin,
                maxRow: rows,
                animate: true,
                float: false,
                disableDrag: false,
                disableResize: false,
                draggable: {
                    handle: ".gs-drag-handle",
                },
                resizable: {
                    handles: "e,se,s,sw,w",
                },
            },
            containerEl,
        );

        // Apply static mode after init so handles are created first
        grid.setStatic(!dashboardState.editing);

        grid.on("change", () => {
            items = serializeLayout();
            onchange?.(items);
        });
    }

    onMount(async () => {
        const gs = await import("gridstack");
        GridStackClass = gs.GridStack;

        await tick();
        initGrid();

        const onResize = () => {
            grid?.cellHeight(computeCellHeight());
        };
        window.addEventListener("resize", onResize);

        return () => {
            window.removeEventListener("resize", onResize);
            if (grid) {
                grid.offAll();
                grid.destroy(false);
                grid = null;
            }
        };
    });

    $effect(() => {
        grid?.setStatic(!dashboardState.editing);
    });
</script>

<div bind:this={containerEl} class="grid-stack h-full" class:gs-editing={dashboardState.editing}>
    {@render children()}
</div>

<style>
    :global(.grid-stack) {
        min-height: 100%;
    }

    :global(.grid-stack-item-content) {
        overflow: hidden;
        border-radius: var(--radius);
        border: 1px solid var(--border);
        background: var(--card);
    }

    :global(.grid-stack-placeholder > .placeholder-content) {
        border: 2px dashed var(--primary) !important;
        border-radius: var(--radius) !important;
        background: oklch(from var(--primary) l c h / 8%) !important;
    }

    :global(.grid-stack-item > .ui-resizable-handle) {
        opacity: 0;
        pointer-events: none;
        transition: opacity 0.15s ease;
    }

    :global(.gs-editing .grid-stack-item:hover > .ui-resizable-handle) {
        opacity: 1;
        pointer-events: auto;
    }

    :global(.ui-resizable-se) {
        width: 14px !important;
        height: 14px !important;
        bottom: 2px !important;
        right: 2px !important;
        background: none !important;
        border-right: 2px solid var(--primary);
        border-bottom: 2px solid var(--primary);
        border-radius: 0 0 3px 0;
    }

    :global(.ui-resizable-e) {
        right: 0 !important;
        width: 6px !important;
        cursor: e-resize;
    }

    :global(.ui-resizable-w) {
        left: 0 !important;
        width: 6px !important;
        cursor: w-resize;
    }

    :global(.ui-resizable-s) {
        bottom: 0 !important;
        height: 6px !important;
        cursor: s-resize;
    }

    :global(.ui-resizable-sw) {
        width: 14px !important;
        height: 14px !important;
        bottom: 2px !important;
        left: 2px !important;
        background: none !important;
        border-left: 2px solid var(--primary);
        border-bottom: 2px solid var(--primary);
        border-radius: 0 0 0 3px;
    }

    :global(.gs-editing .grid-stack-item-content) {
        border-style: dashed;
        border-color: var(--primary);
        border-width: 1px;
    }

    :global(.gs-editing .gs-drag-handle) {
        cursor: grab;
    }

    :global(.gs-editing .gs-drag-handle:active) {
        cursor: grabbing;
    }

    :global(.grid-stack-animate .grid-stack-item) {
        transition:
            left 0.2s ease,
            top 0.2s ease,
            width 0.2s ease,
            height 0.2s ease;
    }
</style>
