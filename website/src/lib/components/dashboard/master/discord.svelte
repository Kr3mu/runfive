<script lang="ts">
    import Discord from "$lib/components/icons/discord.svelte";
    import Check from "@lucide/svelte/icons/check";
    import Eye from "@lucide/svelte/icons/eye";
    import EyeOff from "@lucide/svelte/icons/eye-off";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import X from "@lucide/svelte/icons/x";
    import SquareArrowOutUpRight from "@lucide/svelte/icons/square-arrow-out-up-right";
    import { SaveDiscordAuthentication } from "$lib/api/auth";

    type DiscordState = "idle" | "loading" | "success" | "transition" | "error";

    let showDiscordSecret = $state(false);
    let discordSecret = $state("");
    let discordState = $state<DiscordState>("idle");
    let discordClient = $state("");
    let errorMessage = $state("");

    let isSubmitting = $state(false);

    function handleSave(e: SubmitEvent): void {
        e.preventDefault();
        if (isSubmitting) return;

        discordState = "loading";
        errorMessage = "";

        SaveDiscordAuthentication(discordClient, discordSecret)
            .then((): void => {
                discordState = "success";
                isSubmitting = false;
            })
            .catch((err: unknown): void => {
                discordState = "error";
                errorMessage =
                    err instanceof Error ? err.message : "Saving failed";
            });

        setTimeout((): void => {
            discordState = "idle";
        }, 4000);
    }
</script>

<section class="mb-8">
    <h2
        class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
    >
        <Discord class="size-3.5" />
        Discord Authentication
    </h2>
    <form onsubmit={handleSave} class="flex flex-col gap-4">
        <a
            href="https://discord.com/developers/home"
            class="text-sm flex items-center gap-2 italic underline underline-offset-4 text-neutral-400 leading-none"
            target="_blank"
        >
            <SquareArrowOutUpRight class="mt-1" size={14} /> Open Discord Developer
            Dashboard</a
        >
        <div class="flex flex-col gap-1.5">
            <label
                for="clientid"
                class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
            >
                Client Id
            </label>
            <input
                id="clientid"
                type="text"
                bind:value={discordClient}
                disabled={isSubmitting}
                class="h-10 w-full rounded-md border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {discordState ===
                'error'
                    ? 'border-destructive'
                    : 'border-border'}"
                placeholder="1491511953331036393"
            />
        </div>
        <div class="flex flex-col gap-1.5">
            <label
                for="discord-client-secret"
                class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
            >
                Client Secret
            </label>
            <div class="relative">
                <input
                    id="discord-client-secret"
                    type={showDiscordSecret ? "text" : "password"}
                    bind:value={discordSecret}
                    disabled={isSubmitting}
                    class="h-10 w-full rounded-md border bg-background pr-10 pl-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {discordState ===
                    'error'
                        ? 'border-destructive'
                        : 'border-border'}"
                    placeholder="••••••••••••••••••••••••"
                />
                <button
                    type="button"
                    onclick={() => (showDiscordSecret = !showDiscordSecret)}
                    class="absolute top-1/2 right-3 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                    aria-label={showDiscordSecret
                        ? "Hide password"
                        : "Show password"}
                >
                    {#if showDiscordSecret}
                        <EyeOff size={16} />
                    {:else}
                        <Eye size={16} />
                    {/if}
                </button>
            </div>
        </div>
        <button
            type="submit"
            disabled={isSubmitting}
            class="mt-2 flex h-10 w-full items-center justify-center gap-2 rounded-md font-heading text-sm font-semibold tracking-wide transition-all disabled:cursor-not-allowed {discordState ===
            'success'
                ? 'bg-emerald-500 text-white'
                : discordState === 'error'
                  ? 'animate-shake bg-red-600 text-white'
                  : 'bg-primary text-primary-foreground hover:opacity-90 active:opacity-80'}"
        >
            {#if discordState === "loading"}
                <LoaderCircle size={16} class="animate-spin" />
                Saving configuration
            {:else if discordState === "success"}
                <Check size={16} />
                Successfully saved configuration
            {:else if discordState === "error"}
                <X size={16} />
                {errorMessage}
            {:else}
                Save configuration
            {/if}
        </button>
    </form>
</section>
