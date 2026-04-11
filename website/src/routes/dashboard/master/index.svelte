<!--
    Master Actions page — owner-only panel configuration.

    Houses settings that affect the entire panel (not per-server),
    such as configuring OAuth providers (Discord, etc.).
    Only accessible to the owner/master account.
-->
<script>
    import Discord from "$lib/components/dashboard/master/discord.svelte";
</script>

<div class="flex h-full flex-col overflow-y-auto">
    <div class="mx-auto w-full max-w-2xl px-6 py-8">
        <h1 class="mb-1 text-lg font-semibold text-foreground">
            Master Actions
        </h1>
        <p class="mb-8 text-sm text-muted-foreground">
            Manage all settings regarded to panel
        </p>

        <!-- Discord OAuth credential configuration -->
        <Discord />
    </div>
</div>
