<script lang="ts">
    import { theme } from "$lib/theme.svelte";

    const logoSrc = $derived(
        theme.value === "dark" ? "/logo.webp" : "/logo-light.webp",
    );
</script>

<main
    class="flex min-h-screen flex-col items-center justify-center bg-background"
>
    <div class="flex flex-col items-center gap-8">
        <img src={logoSrc} alt="runfive" class="w-105" />
        <p class="text-lg text-muted-foreground">Admin Panel</p>
        <div class="flex gap-4">
            <a href="/about" class="text-primary hover:underline">About</a>
            <button
                onclick={theme.toggle}
                class="text-muted-foreground hover:text-foreground"
            >
                {theme.value === "dark" ? "Light Mode" : "Dark Mode"}
            </button>
        </div>
    </div>
</main>
