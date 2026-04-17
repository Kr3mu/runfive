<script lang="ts">
    import { Dialog } from "bits-ui";
    import { useQueryClient } from "@tanstack/svelte-query";
    import { toast } from "svelte-sonner";
    import { slide } from "svelte/transition";
    import { cubicOut } from "svelte/easing";
    import { deleteServer, type ManagedServer, type ServerStatus } from "$lib/api/servers";
    import { serverState } from "$lib/server-state.svelte";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Archive from "@lucide/svelte/icons/archive";
    import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import CheckCircle2 from "@lucide/svelte/icons/check-circle-2";
    import X from "@lucide/svelte/icons/x";
    import ArrowRight from "@lucide/svelte/icons/arrow-right";

    interface Props {
        /** Two-way bound open state; parent controls visibility. */
        open: boolean;
        /** Server to delete. Null hides the dialog even when `open` is true. */
        server: ManagedServer | null;
        /**
         * Optional list of remaining servers used to auto-advance the selection
         * when the active server is the one being removed. Pass the full
         * current list; the dialog picks the first non-deleted entry.
         */
        fallbackServers?: ManagedServer[];
        /** Fired once a delete succeeds, after cache invalidation. */
        ondeleted?: (server: ManagedServer, trashed: boolean) => void;
    }

    let { open = $bindable(false), server, fallbackServers = [], ondeleted }: Props = $props();

    type Mode = "trash" | "permanent";

    let mode = $state<Mode>("trash");
    let confirmText = $state("");
    let isSubmitting = $state(false);
    /** Bound to the typed-confirm input so we can autofocus it when it appears. */
    let confirmInputEl = $state<HTMLInputElement | null>(null);

    // Reset transient state every time the dialog opens so the user sees the
    // safe default (trash) and a fresh confirm field — previous runs should
    // never pre-fill a destructive choice.
    $effect((): void => {
        if (open) {
            mode = "trash";
            confirmText = "";
            isSubmitting = false;
        }
    });

    // Pull focus into the typed-confirm input as soon as permanent mode
    // renders it. Uses a microtask so bits-ui's focus-trap doesn't fight us.
    $effect((): void => {
        if (mode === "permanent" && confirmInputEl) {
            queueMicrotask((): void => confirmInputEl?.focus());
        }
    });

    const isRunning = $derived(
        server !== null && (server.status === "running" || server.status === "starting"),
    );

    // Typing the server name gates permanent delete — a trash move is
    // recoverable, but os.RemoveAll is not. Defense against muscle memory.
    const requiresTypedConfirm = $derived(mode === "permanent");
    const typedConfirmMatches = $derived(
        !requiresTypedConfirm || confirmText.trim() === (server?.name ?? ""),
    );
    const typedConfirmHasInput = $derived(confirmText.trim().length > 0);

    const confirmDisabled = $derived(
        isSubmitting || isRunning || server === null || !typedConfirmMatches,
    );

    /** Status-dot colour matches the server switcher so the preview feels native. */
    const statusDot: Record<ServerStatus, string> = {
        running: "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]",
        starting: "bg-amber-400 shadow-[0_0_8px_rgba(251,191,36,0.6)] animate-pulse",
        stopped: "bg-muted-foreground/30",
        crashed: "bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]",
    };

    const statusLabel: Record<ServerStatus, string> = {
        running: "Running",
        starting: "Starting",
        stopped: "Stopped",
        crashed: "Crashed",
    };

    const queryClient = useQueryClient();

    async function handleConfirm(): Promise<void> {
        if (confirmDisabled || server === null) return;
        const target: ManagedServer = server;

        isSubmitting = true;
        try {
            await deleteServer(target.id, { trash: mode === "trash" });

            // Move the active-server pointer off the deleted entry so the
            // dashboard doesn't land on a phantom server after the query
            // invalidation resolves.
            if (serverState.selectedId === target.id) {
                const next = fallbackServers.find((s: ManagedServer): boolean => s.id !== target.id);
                if (next) serverState.select(next.id);
            }

            await queryClient.invalidateQueries({ queryKey: ["servers"] });

            toast.success(
                mode === "trash"
                    ? `${target.name} moved to trash`
                    : `${target.name} deleted permanently`,
            );
            ondeleted?.(target, mode === "trash");
            open = false;
        } catch (err: unknown) {
            const message: string = err instanceof Error ? err.message : "Delete failed";
            toast.error(message);
        } finally {
            isSubmitting = false;
        }
    }
</script>

<Dialog.Root bind:open>
    <Dialog.Portal>
        <Dialog.Overlay
            class="fixed inset-0 z-50 bg-black/55 backdrop-blur-sm data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0"
        />
        <Dialog.Content
            class="fixed top-1/2 left-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-xl border border-border bg-popover shadow-2xl shadow-black/40 ring-1 ring-foreground/5 outline-none data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95"
        >
            {#if server}
                <!-- Header: icon + title stacked with subtle top accent bar
                     that shifts colour based on the destructive-ness of the
                     selected mode. Close button floats top-right. -->
                <div class="relative border-b border-border/60 px-5 pt-5 pb-4">
                    <div
                        class="absolute inset-x-0 top-0 h-px transition-colors {mode ===
                        'permanent'
                            ? 'bg-gradient-to-r from-transparent via-destructive/60 to-transparent'
                            : 'bg-gradient-to-r from-transparent via-primary/50 to-transparent'}"
                    ></div>

                    <Dialog.Close
                        class="absolute top-3 right-3 flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground/60 transition-colors hover:bg-muted/70 hover:text-foreground focus-visible:bg-muted/70 focus-visible:text-foreground focus-visible:outline-none"
                        aria-label="Close"
                    >
                        <X size={14} />
                    </Dialog.Close>

                    <div class="flex items-center gap-2.5">
                        <div
                            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors {mode ===
                            'permanent'
                                ? 'bg-destructive/10 text-destructive'
                                : 'bg-primary/10 text-primary'}"
                        >
                            {#if mode === "permanent"}
                                <TriangleAlert size={15} />
                            {:else}
                                <Archive size={15} />
                            {/if}
                        </div>
                        <div class="min-w-0">
                            <Dialog.Title
                                class="font-heading text-[15px] leading-tight font-semibold text-foreground"
                            >
                                Delete server
                            </Dialog.Title>
                            <Dialog.Description
                                class="mt-0.5 text-[12px] leading-snug text-muted-foreground/70"
                            >
                                Choose how this server should be removed.
                            </Dialog.Description>
                        </div>
                    </div>
                </div>

                <div class="space-y-4 px-5 py-4">
                    <!-- Server preview card: gives the operator a second
                         chance to confirm they're acting on the right server
                         by echoing the same visual cues as the switcher. -->
                    <div
                        class="rounded-lg border border-border/70 bg-background/60 p-3"
                    >
                        <div class="flex items-center gap-2.5">
                            <span
                                class="h-2 w-2 shrink-0 rounded-full {statusDot[
                                    server.status
                                ]}"
                                aria-hidden="true"
                            ></span>
                            <span
                                class="min-w-0 flex-1 truncate font-heading text-[13px] font-semibold text-foreground"
                            >
                                {server.name}
                            </span>
                            <span
                                class="shrink-0 rounded-full border border-border/60 bg-muted/30 px-1.5 py-0.5 font-mono text-[9.5px] tracking-wide text-muted-foreground/80 uppercase"
                            >
                                {statusLabel[server.status]}
                            </span>
                        </div>
                        <div
                            class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 pl-4.5 font-mono text-[10.5px] text-muted-foreground/70"
                        >
                            <span>id <span class="text-foreground/80">{server.id}</span></span>
                            <span class="text-muted-foreground/25">·</span>
                            <span>:{server.port}</span>
                            <span class="text-muted-foreground/25">·</span>
                            <span>{server.playerCount}/{server.maxPlayers} slots</span>
                            {#if server.artifactVersion}
                                <span class="text-muted-foreground/25">·</span>
                                <span>build {server.artifactVersion}</span>
                            {/if}
                        </div>
                    </div>

                    {#if isRunning}
                        <div
                            class="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2.5 text-[11.5px] leading-snug text-amber-700 dark:text-amber-400"
                        >
                            <TriangleAlert size={12} class="mt-0.5 shrink-0" />
                            <span>
                                Stop the server before removing it — the
                                backend rejects the delete while a live
                                process still references this directory.
                            </span>
                        </div>
                    {/if}

                    <!-- Mode cards: selection shifts the whole dialog's accent
                         colour so the commit button, header ring and card
                         border line up into a single visual language. -->
                    <div class="space-y-2">
                        <button
                            type="button"
                            onclick={() => (mode = "trash")}
                            aria-pressed={mode === "trash"}
                            class="group flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-all
                                {mode === 'trash'
                                ? 'border-primary/60 bg-primary/5 ring-1 ring-primary/30'
                                : 'border-border hover:border-primary/30 hover:bg-muted/30'}"
                        >
                            <div
                                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md transition-colors
                                    {mode === 'trash'
                                    ? 'bg-primary/15 text-primary'
                                    : 'bg-muted/50 text-muted-foreground/60 group-hover:text-foreground'}"
                            >
                                <Archive size={14} />
                            </div>
                            <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-2">
                                    <span
                                        class="min-w-0 flex-1 truncate text-[13px] font-semibold text-foreground"
                                        >Move to trash</span
                                    >
                                    <span
                                        class="shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[9.5px] font-medium tracking-wide text-emerald-600 uppercase dark:text-emerald-400"
                                        >Recoverable</span
                                    >
                                </div>
                                <p
                                    class="mt-0.5 text-[11.5px] leading-snug text-muted-foreground/75"
                                >
                                    Files stay on disk so an operator can
                                    restore them later.
                                </p>
                                <div
                                    class="mt-1.5 flex items-center gap-1 font-mono text-[10px] text-muted-foreground/55"
                                >
                                    <ArrowRight size={10} class="shrink-0" />
                                    <code class="truncate"
                                        >servers/.trash/{server.id}.&lt;timestamp&gt;/</code
                                    >
                                </div>
                            </div>
                        </button>

                        <button
                            type="button"
                            onclick={() => (mode = "permanent")}
                            aria-pressed={mode === "permanent"}
                            class="group flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-all
                                {mode === 'permanent'
                                ? 'border-destructive/60 bg-destructive/5 ring-1 ring-destructive/30'
                                : 'border-border hover:border-destructive/30 hover:bg-muted/30'}"
                        >
                            <div
                                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md transition-colors
                                    {mode === 'permanent'
                                    ? 'bg-destructive/15 text-destructive'
                                    : 'bg-muted/50 text-muted-foreground/60 group-hover:text-destructive/80'}"
                            >
                                <Trash2 size={14} />
                            </div>
                            <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-2">
                                    <span
                                        class="min-w-0 flex-1 truncate text-[13px] font-semibold text-foreground"
                                        >Delete permanently</span
                                    >
                                    <span
                                        class="shrink-0 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[9.5px] font-medium tracking-wide text-destructive uppercase"
                                        >Cannot be undone</span
                                    >
                                </div>
                                <p
                                    class="mt-0.5 text-[11.5px] leading-snug text-muted-foreground/75"
                                >
                                    Removes the directory from disk. Config,
                                    console history and data are gone.
                                </p>
                            </div>
                        </button>
                    </div>

                    {#if requiresTypedConfirm}
                        <div
                            transition:slide={{ duration: 160, easing: cubicOut }}
                            class="rounded-lg border border-destructive/30 bg-destructive/5 p-3"
                        >
                            <label
                                for="delete-confirm-input"
                                class="flex items-center justify-between gap-2 text-[11.5px] font-medium text-foreground"
                            >
                                <span>
                                    Type
                                    <code
                                        class="mx-0.5 rounded bg-background px-1 py-0.5 font-mono text-[11px] text-foreground"
                                        >{server.name}</code
                                    >
                                    to confirm
                                </span>
                                <span
                                    class="text-[10px] font-normal text-muted-foreground/60"
                                    >case sensitive</span
                                >
                            </label>
                            <div class="relative mt-2">
                                <input
                                    bind:this={confirmInputEl}
                                    bind:value={confirmText}
                                    id="delete-confirm-input"
                                    type="text"
                                    autocomplete="off"
                                    spellcheck="false"
                                    class="h-9 w-full rounded-md border bg-background px-3 pr-9 font-mono text-sm text-foreground outline-none transition-colors
                                        {typedConfirmMatches && typedConfirmHasInput
                                        ? 'border-emerald-500/60'
                                        : typedConfirmHasInput
                                          ? 'border-destructive/60'
                                          : 'border-border focus:border-destructive/60'}"
                                />
                                {#if typedConfirmMatches && typedConfirmHasInput}
                                    <CheckCircle2
                                        size={14}
                                        class="absolute top-1/2 right-2.5 -translate-y-1/2 text-emerald-500"
                                    />
                                {:else if typedConfirmHasInput}
                                    <X
                                        size={14}
                                        class="absolute top-1/2 right-2.5 -translate-y-1/2 text-destructive/70"
                                    />
                                {/if}
                            </div>
                        </div>
                    {/if}
                </div>

                <!-- Footer: ESC hint on the left, action buttons on the right.
                     The accent colour of the submit button tracks the selected
                     mode so there's never a moment the dialog disagrees with
                     itself about what's about to happen. -->
                <div
                    class="flex items-center justify-between gap-3 border-t border-border/60 bg-muted/20 px-5 py-3"
                >
                    <span class="flex items-center gap-1.5 text-[10.5px] text-muted-foreground/60">
                        <kbd
                            class="rounded border border-border/80 bg-background px-1.5 py-0.5 font-mono text-[10px] text-foreground/80 shadow-[0_1px_0_0_hsl(var(--border))]"
                            >Esc</kbd
                        >
                        to cancel
                    </span>
                    <div class="flex items-center gap-2">
                        <Dialog.Close
                            class="inline-flex h-9 items-center rounded-md border border-border bg-background px-3 text-[12.5px] font-medium text-foreground/80 transition-colors hover:bg-muted"
                        >
                            Cancel
                        </Dialog.Close>
                        <button
                            type="button"
                            onclick={handleConfirm}
                            disabled={confirmDisabled}
                            class="inline-flex h-9 min-w-[9rem] items-center justify-center gap-2 rounded-md px-3 text-[12.5px] font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-40
                                {mode === 'permanent'
                                ? 'bg-destructive text-destructive-foreground shadow-[0_6px_20px_-10px_hsl(var(--destructive)/0.75)] hover:opacity-90'
                                : 'bg-primary text-primary-foreground shadow-[0_6px_20px_-10px_hsl(var(--primary)/0.75)] hover:opacity-90'}"
                        >
                            {#if isSubmitting}
                                <LoaderCircle size={13} class="animate-spin" />
                                <span>Working…</span>
                            {:else if mode === "permanent"}
                                <Trash2 size={13} />
                                <span>Delete permanently</span>
                            {:else}
                                <Archive size={13} />
                                <span>Move to trash</span>
                            {/if}
                        </button>
                    </div>
                </div>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
