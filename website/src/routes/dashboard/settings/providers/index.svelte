<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import Plug from "@lucide/svelte/icons/plug";
    import DiscordProvider from "./discord-provider.svelte";

    const authQuery = createQuery(() => authQueryOptions());
    const user = $derived(authQuery.data);
    const isOwner = $derived(user?.isOwner ?? false);
</script>

{#if isOwner}
    <section class="tab-reveal max-w-3xl">
        <h2
            class="mb-1 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
        >
            <Plug size={14} />
            Authentication Providers
        </h2>
        <p class="mb-3 text-[11px] text-muted-foreground/50">
            External OAuth providers members can link to their account.
            Credentials are encrypted at rest and validated against the
            provider before saving.
        </p>

        <DiscordProvider />
    </section>
{/if}

<style>
    .tab-reveal {
        animation: reveal 0.5s cubic-bezier(0.16, 1, 0.3, 1) both;
        animation-delay: 0.12s;
    }
    @keyframes reveal {
        from {
            opacity: 0;
            transform: translateY(8px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
