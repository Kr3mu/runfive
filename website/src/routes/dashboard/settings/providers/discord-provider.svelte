<script lang="ts">
    import {
        SaveDiscordAuthentication,
        GetDiscordAuthentication,
    } from "$lib/api/auth";
    import Discord from "$lib/components/icons/discord.svelte";
    import Eye from "@lucide/svelte/icons/eye";
    import EyeOff from "@lucide/svelte/icons/eye-off";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Check from "@lucide/svelte/icons/check";
    import ExternalLink from "@lucide/svelte/icons/external-link";
    import { onMount } from "svelte";
    import { toast } from "svelte-sonner";

    let clientId = $state("");
    let clientSecret = $state("");
    let showSecret = $state(false);
    let isSubmitting = $state(false);
    let justSaved = $state(false);

    const isConfigured = $derived(
        clientId.trim() !== "" && clientSecret.trim() !== "",
    );

    onMount((): void => {
        GetDiscordAuthentication()
            .then((data) => {
                if (!data) return;
                clientId = data.clientId;
                clientSecret = data.clientSecret;
            })
            .catch((err: unknown) => {
                toast.error(
                    err instanceof Error
                        ? err.message
                        : "Failed to fetch Discord credentials",
                );
            });
    });

    async function handleSave(e: SubmitEvent): Promise<void> {
        e.preventDefault();
        if (isSubmitting) return;
        isSubmitting = true;
        try {
            await SaveDiscordAuthentication(clientId, clientSecret);
            justSaved = true;
            toast.success("Discord credentials saved");
            setTimeout((): boolean => (justSaved = false), 2500);
        } catch (err: unknown) {
            toast.error(
                err instanceof Error ? err.message : "Failed to save",
            );
        }
        isSubmitting = false;
    }

    const canSubmit = $derived(
        !isSubmitting && clientId.trim() !== "" && clientSecret.trim() !== "",
    );
</script>

<form
    onsubmit={handleSave}
    class="overflow-hidden rounded-lg border border-border bg-card"
>
    <!-- Header row -->
    <div
        class="flex items-center gap-3 border-b border-border/50 px-4 py-3"
    >
        <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[#5865F2]/10"
        >
            <Discord class="h-4 w-4 text-[#5865F2]" />
        </div>
        <div class="min-w-0 flex-1">
            <p class="text-[13px] font-medium text-foreground">Discord</p>
            <p class="text-[11px] text-muted-foreground/60">
                OAuth2 for member account linking
            </p>
        </div>
        <div class="shrink-0">
            {#if isConfigured}
                <span
                    class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-500"
                >
                    <span
                        class="h-1.5 w-1.5 rounded-full bg-emerald-500"
                    ></span>
                    Configured
                </span>
            {:else}
                <span
                    class="inline-flex items-center gap-1.5 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground/60"
                >
                    <span
                        class="h-1.5 w-1.5 rounded-full bg-muted-foreground/30"
                    ></span>
                    Not configured
                </span>
            {/if}
        </div>
    </div>

    <!-- Form body -->
    <div class="space-y-4 px-4 py-4">
        <a
            href="https://discord.com/developers/applications"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 text-[11px] text-muted-foreground/50 transition-colors hover:text-foreground"
        >
            <ExternalLink size={11} />
            Open Discord Developer Portal
        </a>

        <div>
            <label
                for="discord-client-id"
                class="mb-1.5 block text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase"
            >
                Client ID
            </label>
            <input
                id="discord-client-id"
                type="text"
                bind:value={clientId}
                disabled={isSubmitting}
                placeholder="1491511953331036393"
                autocomplete="off"
                spellcheck="false"
                class="h-9 w-full rounded-md border border-border bg-background px-3 font-mono text-xs text-foreground transition-colors placeholder:text-muted-foreground/25 focus:border-primary/50 focus:outline-none disabled:opacity-50"
            />
        </div>

        <div>
            <label
                for="discord-client-secret"
                class="mb-1.5 block text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase"
            >
                Client Secret
            </label>
            <div class="relative">
                <input
                    id="discord-client-secret"
                    type={showSecret ? "text" : "password"}
                    bind:value={clientSecret}
                    disabled={isSubmitting}
                    placeholder="••••••••••••••••••••••••"
                    autocomplete="off"
                    spellcheck="false"
                    class="h-9 w-full rounded-md border border-border bg-background px-3 pr-9 font-mono text-xs text-foreground transition-colors placeholder:text-muted-foreground/25 focus:border-primary/50 focus:outline-none disabled:opacity-50"
                />
                <button
                    type="button"
                    onclick={() => (showSecret = !showSecret)}
                    class="absolute top-1/2 right-2 -translate-y-1/2 rounded p-1 text-muted-foreground/40 transition-colors hover:text-foreground"
                    aria-label={showSecret ? "Hide secret" : "Show secret"}
                >
                    {#if showSecret}
                        <EyeOff size={13} />
                    {:else}
                        <Eye size={13} />
                    {/if}
                </button>
            </div>
        </div>
    </div>

    <!-- Footer -->
    <div
        class="flex items-center justify-end border-t border-border/30 px-4 py-3"
    >
        <button
            type="submit"
            disabled={!canSubmit}
            class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-50
                {justSaved
                ? 'bg-emerald-500/15 text-emerald-500'
                : 'bg-primary text-primary-foreground hover:opacity-90 active:opacity-80'}"
        >
            {#if isSubmitting}
                <LoaderCircle size={12} class="animate-spin" />
                Saving
            {:else if justSaved}
                <Check size={12} strokeWidth={2.5} />
                Saved
            {:else}
                Save configuration
            {/if}
        </button>
    </div>
</form>
