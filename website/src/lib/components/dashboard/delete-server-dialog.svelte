<script lang="ts">
    import { Dialog } from "bits-ui";
    import { useQueryClient } from "@tanstack/svelte-query";
    import { toast } from "svelte-sonner";
    import { deleteServer, type ManagedServer } from "$lib/api/servers";
    import { serverState } from "$lib/server-state.svelte";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Archive from "@lucide/svelte/icons/archive";
    import TriangleAlert from "@lucide/svelte/icons/triangle-alert";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";

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

    const isRunning = $derived(
        server !== null && (server.status === "running" || server.status === "starting"),
    );

    // Typing the server name gates permanent delete — a trash move is
    // recoverable, but os.RemoveAll is not. Defense against muscle memory.
    const requiresTypedConfirm = $derived(mode === "permanent");
    const typedConfirmMatches = $derived(
        !requiresTypedConfirm || confirmText.trim() === (server?.name ?? ""),
    );

    const confirmDisabled = $derived(
        isSubmitting || isRunning || server === null || !typedConfirmMatches,
    );

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
        <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0" />
        <Dialog.Content class="fixed top-1/2 left-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-popover p-5 shadow-2xl shadow-black/40 ring-1 ring-foreground/5 outline-none data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95">
            {#if server}
                <Dialog.Title class="flex items-center gap-2 font-heading text-base font-semibold text-foreground">
                    <TriangleAlert size={16} class="text-destructive" />
                    Delete server
                </Dialog.Title>
                <Dialog.Description class="mt-1.5 text-[13px] leading-relaxed text-muted-foreground">
                    About to remove
                    <span class="font-semibold text-foreground">{server.name}</span>
                    <span class="text-muted-foreground/60">({server.id})</span>
                    from the panel.
                </Dialog.Description>

                {#if isRunning}
                    <div class="mt-4 flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-[12px] text-amber-600 dark:text-amber-500">
                        <TriangleAlert size={13} class="mt-0.5 shrink-0" />
                        <span>
                            This server is still running. Stop it first — the
                            backend will reject the delete request otherwise.
                        </span>
                    </div>
                {/if}

                <div class="mt-4 space-y-2">
                    <button
                        type="button"
                        onclick={() => (mode = "trash")}
                        class="flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-colors
                            {mode === 'trash'
                            ? 'border-primary/60 bg-primary/5'
                            : 'border-border hover:border-primary/30'}"
                    >
                        <Archive size={16} class="mt-0.5 shrink-0 {mode === 'trash' ? 'text-primary' : 'text-muted-foreground/60'}" />
                        <div class="min-w-0 flex-1">
                            <div class="text-[13px] font-semibold text-foreground">Move to trash</div>
                            <p class="mt-0.5 text-[12px] text-muted-foreground/70">
                                Keeps the files under
                                <code class="rounded bg-muted px-1 font-mono text-[10px]">servers/.trash/</code>
                                so you can restore them later.
                            </p>
                        </div>
                    </button>

                    <button
                        type="button"
                        onclick={() => (mode = "permanent")}
                        class="flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-colors
                            {mode === 'permanent'
                            ? 'border-destructive/60 bg-destructive/5'
                            : 'border-border hover:border-destructive/30'}"
                    >
                        <Trash2 size={16} class="mt-0.5 shrink-0 {mode === 'permanent' ? 'text-destructive' : 'text-muted-foreground/60'}" />
                        <div class="min-w-0 flex-1">
                            <div class="text-[13px] font-semibold text-foreground">Delete permanently</div>
                            <p class="mt-0.5 text-[12px] text-muted-foreground/70">
                                Removes the directory from disk. No undo.
                            </p>
                        </div>
                    </button>
                </div>

                {#if requiresTypedConfirm}
                    <div class="mt-4">
                        <label for="delete-confirm-input" class="block text-[12px] font-medium text-foreground">
                            Type
                            <span class="font-mono text-foreground/90">{server.name}</span>
                            to confirm
                        </label>
                        <input
                            id="delete-confirm-input"
                            bind:value={confirmText}
                            type="text"
                            autocomplete="off"
                            spellcheck="false"
                            class="mt-1.5 h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-sm text-foreground outline-none focus:border-destructive/60"
                        />
                    </div>
                {/if}

                <div class="mt-5 flex items-center justify-end gap-2">
                    <Dialog.Close
                        class="inline-flex h-9 items-center rounded-md border border-border bg-background px-3 text-[13px] font-medium text-foreground/80 transition-colors hover:bg-muted"
                    >
                        Cancel
                    </Dialog.Close>
                    <button
                        type="button"
                        onclick={handleConfirm}
                        disabled={confirmDisabled}
                        class="inline-flex h-9 items-center gap-2 rounded-md px-3 text-[13px] font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-40
                            {mode === 'permanent'
                            ? 'bg-destructive text-destructive-foreground hover:opacity-90'
                            : 'bg-primary text-primary-foreground hover:opacity-90'}"
                    >
                        {#if isSubmitting}
                            <LoaderCircle size={13} class="animate-spin" />
                        {:else if mode === "permanent"}
                            <Trash2 size={13} />
                        {:else}
                            <Archive size={13} />
                        {/if}
                        {mode === "permanent" ? "Delete permanently" : "Move to trash"}
                    </button>
                </div>
            {/if}
        </Dialog.Content>
    </Dialog.Portal>
</Dialog.Root>
