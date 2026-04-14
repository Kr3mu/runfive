<script lang="ts">
    import type { Snippet } from "svelte";
    import { createQuery } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import Logo from "$lib/components/logo.svelte";
    import Github from "$lib/components/icons/github.svelte";
    import Discord from "$lib/components/icons/discord.svelte";
    import { Toaster } from "$lib/components/ui/sonner";

    let { children }: { children: Snippet } = $props();

    let pathname = $state(window.location.pathname);

    $effect((): (() => void) => {
        const update = (): void => {
            pathname = window.location.pathname;
        };
        window.addEventListener("popstate", update);
        const observer = new MutationObserver(update);
        observer.observe(
            document.querySelector("head title") ?? document.head,
            { childList: true, subtree: true, characterData: true },
        );
        return (): void => {
            window.removeEventListener("popstate", update);
            observer.disconnect();
        };
    });

    const isDashboard = $derived(pathname.startsWith("/dashboard"));
    const isLoginPage = $derived(pathname === "/" || pathname === "");

    const authQuery = createQuery(() => authQueryOptions());

    const isAuthenticated = $derived(
        authQuery.data !== undefined && authQuery.data !== null,
    );
    const isAuthLoading = $derived(authQuery.isLoading);

    // TODO: Once RBAC is implemented, add per-server permission checks to
    // dashboard routes. Currently any authenticated user sees everything.
    // Redirect users to a "no access" page if they lack permission for
    // the requested server.

    $effect((): void => {
        if (isAuthLoading) return;
        if (isDashboard && !isAuthenticated) {
            window.location.href = "/";
        }
        if (isLoginPage && isAuthenticated) {
            window.location.href = "/dashboard";
        }
    });
</script>

<Toaster />
{#if isAuthLoading}
    <div class="flex min-h-svh items-center justify-center bg-background">
        <Logo class="w-24 animate-pulse opacity-50" />
    </div>
{:else if isDashboard}
    {#if isAuthenticated}
        {@render children()}
    {/if}
{:else}
    <div class="flex min-h-svh flex-col">
        <div class="flex flex-1 flex-col">
            {@render children()}
        </div>

        <footer class="px-6 py-6" style="view-transition-name: footer;">
            <div
                class="mx-auto flex max-w-4xl flex-col items-center gap-4 md:flex-row md:justify-between"
            >
                <a href="/" data-view-transition>
                    <Logo
                        class="w-16 opacity-30 transition-opacity hover:opacity-60"
                    />
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
                        href="https://discord.gg/APvag5Ze5D"
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
