<script lang="ts">
    import type { Snippet } from "svelte";
    import GripVertical from "@lucide/svelte/icons/grip-vertical";
    import X from "@lucide/svelte/icons/x";

    interface Props {
        id: string;
        x: number;
        y: number;
        w: number;
        h: number;
        minW?: number;
        minH?: number;
        maxW?: number;
        maxH?: number;
        title: string;
        noResize?: boolean;
        noMove?: boolean;
        children: Snippet;
        headerActions?: Snippet;
        onremove?: (id: string) => void;
    }

    let {
        id,
        x,
        y,
        w,
        h,
        minW = 2,
        minH = 2,
        maxW,
        maxH,
        title,
        noResize = false,
        noMove = false,
        children,
        headerActions,
        onremove,
    }: Props = $props();

    const gridAttrs = $derived({
        "gs-id": id,
        "gs-x": x,
        "gs-y": y,
        "gs-w": w,
        "gs-h": h,
        "gs-min-w": minW,
        "gs-min-h": minH,
        "gs-max-w": maxW,
        "gs-max-h": maxH,
        "gs-no-resize": noResize || undefined,
        "gs-no-move": noMove || undefined,
    });
</script>

<div
    class="grid-stack-item"
    {...gridAttrs}
>
    <div class="grid-stack-item-content">
        <div class="flex h-full flex-col overflow-hidden">
            <!-- Widget header / drag handle -->
            <div class="gs-drag-handle flex h-8 shrink-0 items-center justify-between border-b border-border bg-card px-2">
                <div class="flex items-center gap-1.5">
                    <GripVertical size={12} class="gs-grip-icon text-muted-foreground/25" />
                    <span class="font-heading text-[10px] font-semibold tracking-widest text-muted-foreground/50 uppercase select-none">
                        {title}
                    </span>
                </div>
                <div class="flex items-center gap-1">
                    {#if headerActions}
                        {@render headerActions()}
                    {/if}
                    {#if onremove}
                        <button
                            onclick={(e) => { e.stopPropagation(); onremove(id); }}
                            class="gs-remove-btn rounded-md p-0.5 text-muted-foreground/30 transition-colors hover:bg-destructive/10 hover:text-destructive"
                            title="Remove widget"
                        >
                            <X size={14} />
                        </button>
                    {/if}
                </div>
            </div>

            <!-- Widget body -->
            <div class="flex-1 overflow-hidden">
                {@render children()}
            </div>
        </div>
    </div>
</div>

<style>
    :global(.gs-grip-icon) {
        opacity: 0;
        width: 0;
        transition: opacity 0.15s ease, width 0.15s ease;
    }

    :global(.gs-editing .gs-grip-icon) {
        opacity: 1;
        width: 12px;
    }

    :global(.gs-remove-btn) {
        opacity: 0;
        pointer-events: none;
        transition: opacity 0.15s ease;
    }

    :global(.gs-editing .gs-remove-btn) {
        opacity: 1;
        pointer-events: auto;
    }
</style>
