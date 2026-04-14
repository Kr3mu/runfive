<script lang="ts" module>
    import type { GridStack as GridStackType, GridStackWidget } from "gridstack";
    export type { GridLayoutItem } from "$lib/types/grid-layout";
    export type { GridStackType, GridStackWidget };
</script>

<script lang="ts">
    import { onMount, tick, type Snippet } from "svelte";
    import "gridstack/dist/gridstack.min.css";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import type { GridLayoutItem } from "$lib/types/grid-layout";

    interface Props {
        items: GridLayoutItem[];
        columns?: number;
        margin?: number;
        children: Snippet;
        onchange?: (items: GridLayoutItem[]) => void;
    }

    let {
        items = $bindable(),
        columns = 12,
        margin = 4,
        children,
        onchange,
    }: Props = $props();

    let wrapperEl = $state<HTMLElement | null>(null);
    let containerEl = $state<HTMLElement | null>(null);
    let grid: GridStackType | null = null;
    let GridStackClass: typeof GridStackType | null = null;
    let ready = $state(false);

    /**
     * Fixed grid height — row 16 is always the bottom edge of the viewport.
     * Matrix coordinates stay portable across different screen sizes,
     * and `maxRow` on the grid prevents widgets from being dragged past it.
     */
    const VISIBLE_ROWS = 16;
    const MIN_CELL_HEIGHT = 24;

    function fitCellHeightToViewport(): void {
        if (!grid || !wrapperEl) return;
        const wrapperH = wrapperEl.clientHeight;
        if (wrapperH <= 0) return;
        // GridStack's row step is exactly `cellHeight` — the `margin` option
        // is rendered as an inset inside each item, not as extra row spacing.
        const newCellH = Math.max(MIN_CELL_HEIGHT, Math.floor(wrapperH / VISIBLE_ROWS));
        grid.cellHeight(newCellH);
        // Mirror GridStack's CSS variables onto the wrapper so the matrix
        // overlay (sibling of .grid-stack, not a child) can resolve them.
        wrapperEl.style.setProperty("--gs-cell-height", `${newCellH}px`);
        wrapperEl.style.setProperty("--gs-column-width", `${100 / columns}%`);
    }

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

    function initGrid(): void {
        if (!GridStackClass || !containerEl) return;

        if (grid) {
            grid.offAll();
            grid.destroy(false);
            grid = null;
        }

        grid = GridStackClass.init(
            {
                column: columns,
                maxRow: VISIBLE_ROWS,
                cellHeight: MIN_CELL_HEIGHT, // overridden by fitCellHeightToViewport
                margin,
                animate: true,
                float: true,
                disableDrag: false,
                disableResize: false,
                alwaysShowResizeHandle: true,
                // Enables GridStack's internal ResizeObserver — without
                // columnOpts (or cellHeight:'auto', or sizeToContent) the
                // grid never reflows when its container width changes.
                columnOpts: { columnMax: columns },
                draggable: {
                    handle: ".gs-drag-handle",
                },
                resizable: {
                    handles: "e,se,s,sw,w",
                },
            },
            containerEl,
        );

        if (grid) {
            fitCellHeightToViewport();
            grid.on("change", () => {
                const serialized = serializeLayout();
                onchange?.(serialized);
            });
            // Reveal + animate only after GridStack has finished placing
            // items, otherwise the animation runs while they're still
            // being positioned.
            requestAnimationFrame(() => {
                ready = true;
            });
        }
    }

    function onWindowResize(): void {
        fitCellHeightToViewport();
    }

    onMount(() => {
        window.addEventListener("resize", onWindowResize);
        void (async (): Promise<void> => {
            const gs = await import("gridstack");
            GridStackClass = gs.GridStack;
            await tick();
            initGrid();
        })();

        return () => {
            window.removeEventListener("resize", onWindowResize);
            if (grid) {
                grid.offAll();
                grid.destroy(false);
                grid = null;
            }
        };
    });

</script>

<div bind:this={wrapperEl} class="relative h-full overflow-hidden">
    <div
        bind:this={containerEl}
        class="grid-stack"
        class:gs-editing={dashboardState.editing}
        class:gs-ready={ready}
    >
        {@render children()}
    </div>
    {#if dashboardState.editing}
        <div class="matrix-overlay" aria-hidden="true">
            {#each Array.from({ length: VISIBLE_ROWS + 1 }, (_, i) => i) as row}
                <div class="matrix-row-line" style="top: calc({row} * var(--gs-cell-height, 80px))">
                    {#if row < VISIBLE_ROWS}
                        <span class="matrix-row-label">{row}</span>
                    {/if}
                </div>
            {/each}
            {#each Array.from({ length: columns + 1 }, (_, i) => i) as col}
                <div class="matrix-col-line" style="left: calc({col} * var(--gs-column-width, 8.333%))">
                    {#if col < columns}
                        <span class="matrix-col-label">{col}</span>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .matrix-overlay {
        position: absolute;
        inset: 0;
        pointer-events: none;
        z-index: 0;
    }
    .matrix-row-line {
        position: absolute;
        left: 0;
        right: 0;
        height: 0;
        border-top: 1px dashed oklch(from var(--primary) l c h / 18%);
    }
    .matrix-col-line {
        position: absolute;
        top: 0;
        bottom: 0;
        width: 0;
        border-left: 1px dashed oklch(from var(--primary) l c h / 18%);
    }
    .matrix-row-label {
        position: absolute;
        left: 3px;
        top: 2px;
        font-family: var(--font-mono, monospace);
        font-size: 9px;
        line-height: 1;
        color: oklch(from var(--primary) l c h / 55%);
        letter-spacing: 0.05em;
    }
    .matrix-col-label {
        position: absolute;
        top: 3px;
        left: 3px;
        font-family: var(--font-mono, monospace);
        font-size: 9px;
        line-height: 1;
        color: oklch(from var(--primary) l c h / 55%);
        letter-spacing: 0.05em;
    }

    :global(.grid-stack) {
        min-height: 100%;
        position: relative;
        z-index: 1;
    }

    :global(.grid-stack-item-content) {
        overflow: hidden;
        border-radius: var(--radius);
        border: 1px solid var(--border);
        background: var(--card);
    }
    /* Keep items invisible until GridStack has finished positioning them. */
    :global(.grid-stack:not(.gs-ready) .grid-stack-item-content) {
        opacity: 0;
    }
    :global(.gs-ready .grid-stack-item-content) {
        animation: widget-enter 0.45s cubic-bezier(0.16, 1, 0.3, 1) both;
    }
    :global(.gs-ready .grid-stack-item:nth-child(1) .grid-stack-item-content) { animation-delay: 0ms; }
    :global(.gs-ready .grid-stack-item:nth-child(2) .grid-stack-item-content) { animation-delay: 55ms; }
    :global(.gs-ready .grid-stack-item:nth-child(3) .grid-stack-item-content) { animation-delay: 110ms; }
    :global(.gs-ready .grid-stack-item:nth-child(4) .grid-stack-item-content) { animation-delay: 165ms; }
    :global(.gs-ready .grid-stack-item:nth-child(5) .grid-stack-item-content) { animation-delay: 220ms; }
    :global(.gs-ready .grid-stack-item:nth-child(6) .grid-stack-item-content) { animation-delay: 275ms; }
    :global(.gs-ready .grid-stack-item:nth-child(7) .grid-stack-item-content) { animation-delay: 330ms; }
    :global(.gs-ready .grid-stack-item:nth-child(8) .grid-stack-item-content) { animation-delay: 385ms; }
    @keyframes widget-enter {
        from {
            opacity: 0;
            transform: translateY(8px) scale(0.985);
            filter: blur(2px);
        }
        to {
            opacity: 1;
            transform: translateY(0) scale(1);
            filter: blur(0);
        }
    }
    @media (prefers-reduced-motion: reduce) {
        :global(.grid-stack-item-content) {
            animation: none;
        }
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

    :global(.grid-stack .gs-drag-handle) {
        pointer-events: none;
    }

    :global(.gs-editing .gs-drag-handle) {
        pointer-events: auto;
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
