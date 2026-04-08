<script lang="ts">
    import type { Snippet } from "svelte";
    import Logo from "$lib/components/logo.svelte";
    import Github from "$lib/components/icons/github.svelte";
    import Discord from "$lib/components/icons/discord.svelte";

    interface Props {
        children: Snippet;
    }

    let { children }: Props = $props();

    let pathname = $state(window.location.pathname);

    $effect(() => {
        const update = () => (pathname = window.location.pathname);
        window.addEventListener("popstate", update);
        // MutationObserver to catch SPA navigations that don't fire popstate
        const observer = new MutationObserver(update);
        observer.observe(document.querySelector("head title") ?? document.head, { childList: true, subtree: true, characterData: true });
        return () => {
            window.removeEventListener("popstate", update);
            observer.disconnect();
        };
    });

    const isDashboard = $derived(pathname.startsWith("/dashboard"));
</script>

{#if isDashboard}
    {@render children()}
{:else}
    <div class="flex min-h-svh flex-col">
        <div class="flex flex-1 flex-col">
            {@render children()}
        </div>

        <footer class="px-6 py-6" style="view-transition-name: footer;">
            <div class="mx-auto flex max-w-4xl flex-col items-center gap-4 md:flex-row md:justify-between">
                <a href="/" data-view-transition>
                    <Logo class="w-16 opacity-30 transition-opacity hover:opacity-60" />
                </a>

                <div class="flex items-center gap-4">
                    <a
                        href="https://github.com/Kr3mu/runfive"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-muted-foreground/40 transition-colors hover:text-foreground"
                        aria-label="GitHub"
                    >
                        <Github class="h-4 w-4" />
                    </a>
                    <a
                        href="https://discord.gg"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-muted-foreground/40 transition-colors hover:text-foreground"
                        aria-label="Discord"
                    >
                        <Discord class="h-4 w-4" />
                    </a>
                </div>

                <a
                    href="/about"
                    data-view-transition
                    class="text-xs text-muted-foreground/40 transition-colors hover:text-foreground"
                >
                    About
                </a>
            </div>
        </footer>
    </div>
{/if}
