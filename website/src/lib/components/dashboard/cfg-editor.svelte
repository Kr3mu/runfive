<script lang="ts">
    /**
     * Reusable cfg-style text editor with a gutter, line numbers, tab handling
     * and scroll sync. Kept dependency-free on purpose — a textarea is enough
     * for fxserver cfg syntax (no AST, no completion) and skipping a code-
     * editor package keeps the dashboard bundle small.
     *
     * The host owns the buffer: pass `value` and react to `onchange`. Save
     * orchestration (dirty tracking, debouncing, Ctrl+S, network call) lives
     * one level up because it depends on the network shape, not the widget.
     */

    interface Props {
        /** Current buffer contents. Two-way: emit changes via `onchange`. */
        value: string;
        /** Fired on every keystroke / paste with the new buffer body. */
        onchange?: (next: string) => void;
        /** Fired when the user hits Ctrl+S / Cmd+S inside the editor. */
        onsave?: () => void;
        /** Disables editing without changing the visual surface. */
        readonly?: boolean;
        /**
         * Optional placeholder shown when the buffer is empty. Rendered as
         * its own layer so it doesn't pollute the underlying value.
         */
        placeholder?: string;
        /**
         * Field name used by the textarea — meaningful only when this editor
         * is rendered inside a real `<form>`. Defaults to "content".
         */
        name?: string;
    }

    let {
        value = $bindable(""),
        onchange,
        onsave,
        readonly = false,
        placeholder = "",
        name = "content",
    }: Props = $props();

    let textareaEl = $state<HTMLTextAreaElement | null>(null);
    let gutterEl = $state<HTMLDivElement | null>(null);

    /**
     * Line count drives the gutter. We always show at least one number so
     * an empty buffer doesn't render a bare gutter strip.
     */
    const lineCount = $derived.by((): number => {
        if (value.length === 0) return 1;
        let n = 1;
        for (let i = 0; i < value.length; i += 1) {
            if (value.charCodeAt(i) === 10) n += 1;
        }
        return n;
    });

    /**
     * Pre-rendered numbers array. Kept reactive so it tracks line additions
     * without us pushing imperatively into the gutter DOM.
     */
    const lineNumbers = $derived.by((): number[] => {
        const arr = new Array<number>(lineCount);
        for (let i = 0; i < lineCount; i += 1) arr[i] = i + 1;
        return arr;
    });

    /**
     * Mirror the textarea's vertical scroll to the gutter on every frame
     * the user scrolls. Listening on the textarea's `scroll` event is
     * cheap; we do a single transform write per tick.
     */
    function handleScroll(): void {
        if (!textareaEl || !gutterEl) return;
        gutterEl.scrollTop = textareaEl.scrollTop;
    }

    function handleInput(event: Event): void {
        const target = event.currentTarget as HTMLTextAreaElement;
        value = target.value;
        onchange?.(target.value);
    }

    function handleKeydown(event: KeyboardEvent): void {
        // Ctrl+S / Cmd+S = save passthrough. Always intercept the browser
        // "save page" shortcut so it doesn't pop a Save Dialog on the
        // dashboard.
        if ((event.ctrlKey || event.metaKey) && event.key === "s") {
            event.preventDefault();
            if (!readonly) onsave?.();
            return;
        }

        if (readonly) return;

        // Tab inserts two spaces at the caret rather than moving focus —
        // fxserver cfg files aren't tab-indented in practice but it's the
        // expected behaviour in any editor surface that calls itself one.
        // setRangeText mutates the DOM input synchronously and parks the
        // caret after the insertion ("end" selection mode), which avoids
        // the value/selectionStart race a manual substring rewrite has.
        if (event.key === "Tab") {
            event.preventDefault();
            const ta = event.currentTarget as HTMLTextAreaElement;
            ta.setRangeText("  ", ta.selectionStart, ta.selectionEnd, "end");
            value = ta.value;
            onchange?.(ta.value);
        }
    }
</script>

<div
    class="cfg-editor relative flex h-full min-h-0 overflow-hidden rounded-md border border-border bg-background"
>
    <div
        bind:this={gutterEl}
        class="cfg-editor-gutter shrink-0 select-none overflow-hidden border-r border-border bg-muted/30 py-3 text-right font-mono text-[12.5px] leading-[1.55] tabular-nums text-muted-foreground/40"
        aria-hidden="true"
    >
        {#each lineNumbers as n}
            <div class="px-2.5">{n}</div>
        {/each}
    </div>

    <div class="relative flex-1 min-w-0">
        <textarea
            bind:this={textareaEl}
            {value}
            {name}
            {readonly}
            spellcheck="false"
            autocomplete="off"
            autocapitalize="off"
            oninput={handleInput}
            onkeydown={handleKeydown}
            onscroll={handleScroll}
            class="cfg-editor-textarea h-full w-full resize-none border-0 bg-transparent px-3 py-3 font-mono text-[12.5px] leading-[1.55] text-foreground outline-none placeholder:text-muted-foreground/30 focus:ring-0 disabled:opacity-50"
        ></textarea>

        {#if placeholder && value.length === 0}
            <div
                class="pointer-events-none absolute top-3 left-3 font-mono text-[12.5px] leading-[1.55] whitespace-pre text-muted-foreground/30"
                aria-hidden="true"
            >
                {placeholder}
            </div>
        {/if}
    </div>
</div>

<style>
    /*
     * White-space: pre is the load-bearing rule — without it the textarea
     * collapses indentation and the gutter line count drifts from what the
     * user sees. tab-size keeps `\t` characters visually consistent if a
     * paste includes them.
     */
    .cfg-editor-textarea {
        white-space: pre;
        tab-size: 2;
        -moz-tab-size: 2;
        overflow: auto;
        scrollbar-width: thin;
        scrollbar-color: rgba(148, 163, 184, 0.25) transparent;
    }
    .cfg-editor-textarea::-webkit-scrollbar {
        width: 9px;
        height: 9px;
    }
    .cfg-editor-textarea::-webkit-scrollbar-thumb {
        background-color: rgba(148, 163, 184, 0.25);
        border-radius: 4px;
    }
    .cfg-editor-textarea::-webkit-scrollbar-thumb:hover {
        background-color: rgba(148, 163, 184, 0.45);
    }
    /*
     * The gutter hides its own scrollbar — the editor textarea is the
     * single source of truth for scroll position and we mirror onto the
     * gutter via JS. A visible second scrollbar would just be confusing.
     */
    .cfg-editor-gutter {
        scrollbar-width: none;
        min-width: 3rem;
    }
    .cfg-editor-gutter::-webkit-scrollbar {
        display: none;
    }
</style>
