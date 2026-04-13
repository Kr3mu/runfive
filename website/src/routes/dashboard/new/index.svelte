<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { navigate } from "sv-router/generated";
    import { authQueryOptions } from "$lib/api/auth";
    import type { ManagedServer } from "$lib/api/servers";
    import CreateServerForm from "$lib/components/dashboard/create-server-form.svelte";
    import { canGlobal } from "$lib/permissions.svelte";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";

    const authQuery = createQuery(() => authQueryOptions());
    const currentUser = $derived(authQuery.data);
    const canCreateServers = $derived(canGlobal(currentUser, "servers", "create"));

    function handleCreated(_server: ManagedServer): void {
        void navigate("/dashboard");
    }
</script>

{#if authQuery.isLoading}
    <div class="flex h-full items-center justify-center">
        <LoaderCircle size={20} class="animate-spin text-muted-foreground" />
    </div>
{:else}
    <div class="flex h-full flex-col overflow-y-auto">
        <div class="mx-auto w-full max-w-2xl px-6 py-8">
            {#if canCreateServers}
                <CreateServerForm
                    heading="Create a new server"
                    subtitle="Give it a name and pick a FiveM build. Your existing servers stay untouched."
                    oncreated={handleCreated}
                />
            {:else}
                <div class="mx-auto max-w-md rounded-lg border border-border bg-card p-8 text-center">
                    <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                        <ShieldAlert size={20} class="text-destructive" />
                    </div>
                    <h1 class="font-heading text-lg font-semibold text-foreground">
                        Not allowed
                    </h1>
                    <p class="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
                        You don't have permission to create new servers. Ask an owner to grant you access.
                    </p>
                </div>
            {/if}
        </div>
    </div>
{/if}
