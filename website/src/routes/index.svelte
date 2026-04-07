<script lang="ts">
    import { theme } from "$lib/theme.svelte";
    import Logo from "$lib/components/logo.svelte";
    import Sun from "@lucide/svelte/icons/sun";
    import Moon from "@lucide/svelte/icons/moon";
    import Eye from "@lucide/svelte/icons/eye";
    import EyeOff from "@lucide/svelte/icons/eye-off";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Check from "@lucide/svelte/icons/check";
    import X from "@lucide/svelte/icons/x";

    type LoginState = "idle" | "loading" | "success" | "transition" | "error";

    let username = $state("");
    let password = $state("");
    let showPassword = $state(false);
    let loginState = $state<LoginState>("idle");
    let errorMessage = $state("");
    let logoEl = $state<HTMLElement | null>(null);
    let logoRect = $state<{ top: number; left: number; width: number } | null>(null);

    const isSubmitting = $derived(
        loginState === "loading" || loginState === "success" || loginState === "transition",
    );

    function handleLogin(e: SubmitEvent): void {
        e.preventDefault();
        if (isSubmitting) return;

        loginState = "loading";
        errorMessage = "";

        // TODO: replace with actual auth call
        setTimeout((): void => {
            if (username === "admin" && password === "admin") {
                loginState = "success";

                setTimeout((): void => {
                    if (logoEl) {
                        const rect = logoEl.getBoundingClientRect();
                        logoRect = { top: rect.top, left: rect.left, width: rect.width };
                    }
                    loginState = "transition";
                    // TODO: navigate to dashboard after animation
                }, 800);
            } else {
                loginState = "error";
                errorMessage = "Invalid username or password";
                setTimeout((): void => {
                    loginState = "idle";
                }, 2000);
            }
        }, 1500);
    }

    function handleCfxLogin(): void {
        if (isSubmitting) return;
        loginState = "loading";
        // TODO: redirect to Cfx.re OAuth
    }
</script>

{#if loginState === "transition" && logoRect}
    <div class="fixed inset-0 z-50 bg-background">
        <div
            class="logo-transition"
            style="
                --start-top: {logoRect.top}px;
                --start-left: {logoRect.left}px;
                --start-width: {logoRect.width}px;
            "
        >
            <Logo class="w-full" />
        </div>
    </div>
{/if}

<main
    class="relative flex min-h-svh items-center justify-center bg-background px-4 transition-opacity duration-500"
    class:opacity-0={loginState === "transition"}
>
    <button
        onclick={theme.toggle}
        class="absolute top-6 right-6 text-muted-foreground transition-colors hover:text-foreground"
        aria-label="Toggle theme"
    >
        {#if theme.value === "dark"}
            <Sun size={18} />
        {:else}
            <Moon size={18} />
        {/if}
    </button>

    <div class="w-full max-w-sm">
        <div class="mb-10 flex flex-col items-center">
            <div
                bind:this={logoEl}
                class="mb-3 w-52"
                class:animate-pulse={loginState === "loading"}
            >
                <Logo class="w-full" />
            </div>
            <p class="text-sm tracking-wide text-muted-foreground">
                Server Management
            </p>
        </div>

        <form onsubmit={handleLogin} class="flex flex-col gap-4">
            <div class="flex flex-col gap-1.5">
                <label
                    for="username"
                    class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
                >
                    Username
                </label>
                <input
                    id="username"
                    type="text"
                    bind:value={username}
                    autocomplete="username"
                    disabled={isSubmitting}
                    class="h-10 w-full rounded-md border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {loginState === 'error' ? 'border-destructive' : 'border-border'}"
                    placeholder="admin"
                />
            </div>

            <div class="flex flex-col gap-1.5">
                <label
                    for="password"
                    class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
                >
                    Password
                </label>
                <div class="relative">
                    <input
                        id="password"
                        type={showPassword ? "text" : "password"}
                        bind:value={password}
                        autocomplete="current-password"
                        disabled={isSubmitting}
                        class="h-10 w-full rounded-md border bg-background pr-10 pl-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {loginState === 'error' ? 'border-destructive' : 'border-border'}"
                        placeholder="••••••••"
                    />
                    <button
                        type="button"
                        onclick={() => (showPassword = !showPassword)}
                        class="absolute top-1/2 right-3 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                        aria-label={showPassword
                            ? "Hide password"
                            : "Show password"}
                    >
                        {#if showPassword}
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
                class="mt-2 flex h-10 w-full items-center justify-center gap-2 rounded-md font-heading text-sm font-semibold tracking-wide transition-all disabled:cursor-not-allowed {loginState === 'success' ? 'bg-emerald-500 text-white' : loginState === 'error' ? 'animate-shake bg-red-600 text-white' : 'bg-primary text-primary-foreground hover:opacity-90 active:opacity-80'}"
            >
                {#if loginState === "loading"}
                    <LoaderCircle size={16} class="animate-spin" />
                    Signing in...
                {:else if loginState === "success"}
                    <Check size={16} />
                    Welcome back
                {:else if loginState === "error"}
                    <X size={16} />
                    {errorMessage}
                {:else}
                    Sign in
                {/if}
            </button>
        </form>

        <div class="my-6 flex items-center gap-3">
            <div class="h-px flex-1 bg-border"></div>
            <span class="text-xs text-muted-foreground/50">or</span>
            <div class="h-px flex-1 bg-border"></div>
        </div>

        <button
            type="button"
            disabled={isSubmitting}
            onclick={handleCfxLogin}
            class="flex h-10 w-full items-center justify-center gap-2.5 rounded-md bg-[#F40552] text-sm font-semibold text-white transition-opacity hover:opacity-90 active:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
        >
            <svg viewBox="0 0 95 64" fill="none" class="h-4 w-auto">
                <path fill="currentColor" d="M64.4992 0H52.0201L53.054 12.8535H41.6446L42.6785 0H30.3064L0 64H37.4908L38.7209 48.7933H55.9599L57.19 64H94.7877L64.4813 0H64.4992ZM39.5944 38.2039L41.2167 18.2017H53.4997L55.122 38.2039H39.5944Z"/>
            </svg>
            Sign in with Cfx.re
        </button>

        <p class="mt-8 text-center text-xs text-muted-foreground/50">
            runfive &middot; open source &middot; done right
        </p>
    </div>
</main>

<style>
    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        10%, 30%, 50%, 70%, 90% { transform: translateX(-4px); }
        20%, 40%, 60%, 80% { transform: translateX(4px); }
    }

    :global(.animate-shake) {
        animation: shake 0.5s ease-in-out;
    }

    .logo-transition {
        position: absolute;
        top: var(--start-top);
        left: var(--start-left);
        width: var(--start-width);
        animation: logo-fly 1.2s cubic-bezier(0.4, 0, 0.2, 1) forwards;
    }

    @keyframes logo-fly {
        0% {
            top: var(--start-top);
            left: var(--start-left);
            width: var(--start-width);
            transform: translate(0, 0);
            opacity: 1;
        }
        40% {
            top: 50%;
            left: 50%;
            width: 280px;
            transform: translate(-50%, -50%) scale(1.15);
            opacity: 1;
        }
        65% {
            top: 50%;
            left: 50%;
            width: 280px;
            transform: translate(-50%, -50%) scale(1.1);
            opacity: 1;
        }
        100% {
            top: 50%;
            left: 120%;
            width: 280px;
            transform: translate(-50%, -50%) scale(1.1);
            opacity: 0;
        }
    }
</style>
