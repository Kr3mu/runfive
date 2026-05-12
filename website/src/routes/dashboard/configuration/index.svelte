<script lang="ts">
    /**
     * Per-server configuration page: hosts the operator-owned custom.cfg
     * editor. The page is the dirty-state authority — the embedded
     * <CfgEditor> stays dumb about networking — and gates read/write
     * separately so a read-only role still sees the file.
     *
     * @see /v1/servers/:serverId/custom-cfg
     * @see CfgEditor for the underlying text-surface widget
     */
    import { createQuery } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import {
        fetchCustomCfg,
        serversQueryOptions,
        updateCustomCfg,
        type ServerCustomCfg,
    } from "$lib/api/servers";
    import { canServer } from "$lib/permissions.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import CfgEditor from "$lib/components/dashboard/cfg-editor.svelte";
    import { toast } from "svelte-sonner";

    import FileText from "@lucide/svelte/icons/file-text";
    import Save from "@lucide/svelte/icons/save";
    import RotateCcw from "@lucide/svelte/icons/rotate-ccw";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";
    import CircleCheck from "@lucide/svelte/icons/circle-check";
    import CircleAlert from "@lucide/svelte/icons/circle-alert";
    import Info from "@lucide/svelte/icons/info";

    type SaveState = "idle" | "saving" | "saved" | "error";

    const authQuery = createQuery(() => authQueryOptions());
    const serversQuery = createQuery(() => serversQueryOptions());

    const currentUser = $derived(authQuery.data);
    const servers = $derived(serversQuery.data ?? []);
    const selectedServer = $derived(serverState.resolve(servers));
    const canReadCfg = $derived(
        selectedServer
            ? canServer(currentUser, selectedServer.id, "settings", "read")
            : false,
    );
    const canEditCfg = $derived(
        selectedServer
            ? canServer(currentUser, selectedServer.id, "settings", "update")
            : false,
    );

    /** Buffer currently rendered in the editor. Owned by this page. */
    let buffer = $state("");
    /** Last server-confirmed snapshot — used for dirty tracking & discard. */
    let lastSaved = $state<ServerCustomCfg | null>(null);
    let loading = $state(false);
    let loadError = $state<string | null>(null);
    let saveState = $state<SaveState>("idle");
    /**
     * Sticky timestamp shown in the footer between saves. Re-renders every
     * minute so "2 min ago" stays accurate without a save cycle.
     */
    let now = $state<number>(Date.now());
    /**
     * Tracks the server we last hydrated against. Server switches reset the
     * buffer and re-fetch; we deliberately do NOT prompt about unsaved
     * edits because the per-server scope is the user's choice and they may
     * have meant to abandon the in-flight edit on the previous server.
     */
    let hydratedFor = $state<string | null>(null);

    const dirty = $derived((lastSaved?.content ?? "") !== buffer);
    const tooLarge = $derived(
        lastSaved ? buffer.length > lastSaved.maxBytes : false,
    );
    const canSave = $derived(canEditCfg && dirty && !tooLarge && saveState !== "saving");

    /**
     * Hydrates the editor whenever the selected server (or read permission)
     * changes. Errors surface as a load-error banner; the editor body is
     * hidden until we have a real snapshot, so partial state can't trick
     * the user into overwriting a missing file.
     */
    $effect((): void => {
        const sid = selectedServer?.id;
        const readable = canReadCfg;

        if (!sid || !readable) {
            lastSaved = null;
            buffer = "";
            hydratedFor = null;
            loadError = null;
            loading = false;
            return;
        }

        if (sid === hydratedFor) return;

        hydratedFor = sid;
        loading = true;
        loadError = null;

        void (async (): Promise<void> => {
            try {
                const snap: ServerCustomCfg = await fetchCustomCfg(sid);
                if (hydratedFor !== sid) return;
                lastSaved = snap;
                buffer = snap.content;
                saveState = "idle";
            } catch (err: unknown) {
                if (hydratedFor !== sid) return;
                loadError =
                    err instanceof Error
                        ? err.message
                        : "Failed to load custom.cfg";
            } finally {
                if (hydratedFor === sid) loading = false;
            }
        })();
    });

    /** One-minute heartbeat for the "last saved X ago" footer label. */
    $effect((): (() => void) => {
        const id = window.setInterval((): void => {
            now = Date.now();
        }, 60_000);
        return (): void => window.clearInterval(id);
    });

    /**
     * Guard against accidental tab close / reload while there are unsaved
     * edits. Standard beforeunload pattern — modern browsers ignore the
     * custom message but honour the prompt itself.
     */
    $effect((): (() => void) => {
        const onBeforeUnload = (e: BeforeUnloadEvent): void => {
            if (!dirty) return;
            e.preventDefault();
            e.returnValue = "";
        };
        window.addEventListener("beforeunload", onBeforeUnload);
        return (): void => window.removeEventListener("beforeunload", onBeforeUnload);
    });

    async function handleSave(): Promise<void> {
        if (!selectedServer || !canSave) return;
        saveState = "saving";
        try {
            const snap: ServerCustomCfg = await updateCustomCfg(
                selectedServer.id,
                buffer,
            );
            lastSaved = snap;
            saveState = "saved";
            toast.success("custom.cfg saved", {
                description: "Restart the server for the changes to take effect.",
            });
            // Drop the "saved" highlight after a beat so the button settles
            // back to its neutral state once the operator's eye moves on.
            window.setTimeout((): void => {
                if (saveState === "saved") saveState = "idle";
            }, 1500);
        } catch (err: unknown) {
            saveState = "error";
            const message: string =
                err instanceof Error ? err.message : "Failed to save custom.cfg";
            toast.error(message);
        }
    }

    function handleDiscard(): void {
        if (!lastSaved || !dirty) return;
        buffer = lastSaved.content;
        saveState = "idle";
    }

    /**
     * Human-readable "last saved" hint. Backend returns the zero time when
     * the file does not exist yet; we treat that as "never" so the UI
     * doesn't show a meaningless `01-01-0001` date.
     */
    const lastSavedLabel = $derived.by((): string => {
        const iso = lastSaved?.updatedAt;
        if (!iso) return "Never saved";
        const t = new Date(iso).getTime();
        if (!Number.isFinite(t) || t <= 0) return "Never saved";
        const seconds = Math.max(0, Math.floor((now - t) / 1000));
        if (seconds < 60) return "Saved just now";
        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) return `Saved ${minutes} min ago`;
        const hours = Math.floor(minutes / 60);
        if (hours < 24) return `Saved ${hours} h ago`;
        const days = Math.floor(hours / 24);
        return `Saved ${days} d ago`;
    });

    const sizeLabel = $derived.by((): string => {
        const max = lastSaved?.maxBytes ?? 0;
        if (max === 0) return `${buffer.length} bytes`;
        return `${buffer.length.toLocaleString()} / ${max.toLocaleString()} bytes`;
    });

    const sizeTone = $derived.by((): string => {
        const max = lastSaved?.maxBytes ?? 0;
        if (max === 0) return "text-muted-foreground/50";
        const ratio = buffer.length / max;
        if (ratio >= 1) return "text-destructive";
        if (ratio >= 0.85) return "text-amber-400";
        return "text-muted-foreground/50";
    });
</script>

<div class="flex h-full min-h-0 flex-col overflow-hidden">
    {#if !selectedServer}
        <div class="flex h-full items-center justify-center px-6 text-sm text-muted-foreground/50">
            Select a server to open its configuration.
        </div>
    {:else if !canReadCfg}
        <div class="flex h-full items-center justify-center px-6">
            <div class="max-w-sm rounded-lg border border-border bg-card p-6 text-center">
                <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                    <ShieldAlert size={20} class="text-destructive" />
                </div>
                <h2 class="font-heading text-base font-semibold text-foreground">
                    Configuration access required
                </h2>
                <p class="mt-2 text-sm text-muted-foreground">
                    This account can see the server, but it doesn't have permission to read its settings.
                </p>
            </div>
        </div>
    {:else}
        <!-- Header: title + status + actions -->
        <div class="shrink-0 border-b border-border bg-card/80 px-6 py-4 backdrop-blur-sm">
            <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="min-w-0">
                    <div class="flex items-center gap-2">
                        <FileText size={14} class="text-primary" />
                        <h1 class="font-heading text-base font-semibold text-foreground">
                            custom.cfg
                        </h1>
                        <span class="font-mono text-[11px] text-muted-foreground/40">
                            {selectedServer.name}
                        </span>
                    </div>
                    <p class="mt-1 max-w-2xl text-[12px] text-muted-foreground/70">
                        Hand-managed overrides that survive panel rewrites. Loaded by
                        <span class="font-mono text-foreground/70">server.cfg</span>
                        on launch via <span class="font-mono text-foreground/70">exec</span>.
                    </p>
                </div>

                {#if canEditCfg}
                    <div class="flex shrink-0 items-center gap-2">
                        <button
                            type="button"
                            onclick={handleDiscard}
                            disabled={!dirty || saveState === "saving"}
                            class="inline-flex h-8 items-center gap-1.5 rounded-md border border-border bg-background px-3 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-35"
                            title="Discard unsaved changes"
                        >
                            <RotateCcw size={13} />
                            Discard
                        </button>
                        <button
                            type="button"
                            onclick={handleSave}
                            disabled={!canSave}
                            class="inline-flex h-8 items-center gap-1.5 rounded-md border border-primary/30 bg-primary/10 px-3 text-xs font-semibold text-primary transition-colors hover:bg-primary/15 disabled:cursor-not-allowed disabled:opacity-35"
                            title="Save (Ctrl+S)"
                        >
                            {#if saveState === "saving"}
                                <LoaderCircle size={13} class="animate-spin" />
                                Saving…
                            {:else if saveState === "saved"}
                                <CircleCheck size={13} />
                                Saved
                            {:else}
                                <Save size={13} />
                                Save
                            {/if}
                        </button>
                    </div>
                {/if}
            </div>

            {#if dirty}
                <div class="mt-3 flex items-center gap-2 rounded-md border border-amber-400/25 bg-amber-400/5 px-3 py-1.5 text-[11.5px] text-amber-300/90">
                    <Info size={12} class="shrink-0 text-amber-400/80" />
                    <span>
                        Unsaved changes. Restart <span class="font-mono">{selectedServer.name}</span> after saving for the new directives to load.
                    </span>
                </div>
            {/if}

            {#if loadError}
                <div class="mt-3 flex items-center gap-2 rounded-md border border-destructive/25 bg-destructive/5 px-3 py-1.5 text-[11.5px] text-destructive">
                    <CircleAlert size={12} class="shrink-0" />
                    <span>{loadError}</span>
                </div>
            {/if}
        </div>

        <!-- Editor body -->
        <div class="relative flex-1 min-h-0 overflow-hidden p-4">
            {#if loading && lastSaved === null}
                <div class="flex h-full items-center justify-center">
                    <span class="inline-flex items-center gap-2 text-sm text-muted-foreground/60">
                        <LoaderCircle size={14} class="animate-spin" />
                        Loading configuration…
                    </span>
                </div>
            {:else}
                <CfgEditor
                    bind:value={buffer}
                    readonly={!canEditCfg}
                    onsave={handleSave}
                    placeholder={canEditCfg
                        ? "# This file is empty.\n# Add convars, ACEs or `start` lines that should run after the panel-managed directives.\n"
                        : ""}
                />
            {/if}
        </div>

        <!-- Footer: size counter + last-saved hint -->
        <div class="shrink-0 border-t border-border bg-card/60 px-6 py-2.5">
            <div class="flex flex-wrap items-center justify-between gap-2 text-[11px]">
                <div class="flex items-center gap-3">
                    <span class="font-mono tabular-nums {sizeTone}">{sizeLabel}</span>
                    {#if tooLarge}
                        <span class="font-semibold text-destructive">Exceeds size limit</span>
                    {/if}
                </div>
                <span class="text-muted-foreground/50">{lastSavedLabel}</span>
            </div>
        </div>
    {/if}
</div>
