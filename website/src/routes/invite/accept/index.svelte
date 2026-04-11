<script lang="ts">
    import { theme } from '$lib/theme.svelte';
    import Logo from '$lib/components/logo.svelte';
    import Cfxre from '$lib/components/icons/cfxre.svelte';
    import Sun from '@lucide/svelte/icons/sun';
    import Moon from '@lucide/svelte/icons/moon';
    import Eye from '@lucide/svelte/icons/eye';
    import EyeOff from '@lucide/svelte/icons/eye-off';
    import LoaderCircle from '@lucide/svelte/icons/loader-circle';
    import Check from '@lucide/svelte/icons/check';
    import X from '@lucide/svelte/icons/x';
    import ShieldAlert from '@lucide/svelte/icons/shield-alert';
    import Clock from '@lucide/svelte/icons/clock';
    import { validateInvite, acceptInvite } from '$lib/api/invites';

    type PageState = 'validating' | 'invalid' | 'ready' | 'loading' | 'success' | 'transition' | 'error';

    let pageState = $state<PageState>('validating');
    let username = $state('');
    let password = $state('');
    let showPassword = $state(false);
    let errorMessage = $state('');
    let expiresAt = $state('');
    let logoEl = $state<HTMLElement | null>(null);
    let logoRect = $state<{ top: number; left: number; width: number } | null>(null);

    const token: string = new URLSearchParams(window.location.search).get('token') ?? '';
    const urlError: string | null = new URLSearchParams(window.location.search).get('error');

    const isSubmitting = $derived(
        pageState === 'loading' || pageState === 'success' || pageState === 'transition',
    );

    function formatExpiry(iso: string): string {
        const diff: number = new Date(iso).getTime() - Date.now();
        if (diff <= 0) return 'expired';
        const hours: number = Math.floor(diff / 3600000);
        const minutes: number = Math.floor((diff % 3600000) / 60000);
        if (hours > 0) return `${hours}h ${minutes}m remaining`;
        return `${minutes}m remaining`;
    }

    $effect((): void => {
        if (!token) {
            pageState = 'invalid';
            return;
        }
        if (urlError) {
            pageState = 'invalid';
            errorMessage = urlError === 'invalid_invite'
                ? 'This invite link is no longer valid.'
                : urlError === 'registration_failed'
                    ? 'Registration failed. The username may already be taken.'
                    : 'Something went wrong. Please try again.';
            return;
        }
        validateInvite(token).then((result): void => {
            if (result.valid) {
                pageState = 'ready';
                expiresAt = result.expiresAt ?? '';
            } else {
                pageState = 'invalid';
            }
        }).catch((): void => {
            pageState = 'invalid';
        });
    });

    function handleSubmit(e: SubmitEvent): void {
        e.preventDefault();
        if (isSubmitting || pageState !== 'ready') return;

        pageState = 'loading';
        errorMessage = '';

        acceptInvite(token, username, password)
            .then((): void => {
                pageState = 'success';
                setTimeout((): void => {
                    if (logoEl) {
                        const rect: DOMRect = logoEl.getBoundingClientRect();
                        logoRect = { top: rect.top, left: rect.left, width: rect.width };
                    }
                    pageState = 'transition';
                    setTimeout((): void => {
                        window.location.href = '/dashboard';
                    }, 1200);
                }, 800);
            })
            .catch((err: unknown): void => {
                pageState = 'error';
                errorMessage = err instanceof Error ? err.message : 'Registration failed';
                setTimeout((): void => {
                    pageState = 'ready';
                }, 2000);
            });
    }

    function handleCfxRegister(): void {
        if (isSubmitting) return;
        pageState = 'loading';
        window.location.href = `/v1/auth/cfx?invite=${encodeURIComponent(token)}`;
    }
</script>

{#if pageState === 'transition' && logoRect}
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
    class="relative flex flex-1 items-center justify-center bg-background px-4 transition-opacity duration-500"
    class:opacity-0={pageState === 'transition'}
>
    <button
        onclick={theme.toggle}
        class="absolute top-6 right-6 text-muted-foreground transition-colors hover:text-foreground"
        aria-label="Toggle theme"
    >
        {#if theme.value === 'dark'}
            <Sun size={18} />
        {:else}
            <Moon size={18} />
        {/if}
    </button>

    <div class="w-full max-w-sm">
        <!-- Logo -->
        <div class="invite-reveal mb-10 flex flex-col items-center">
            <div
                bind:this={logoEl}
                class="mb-3 w-52"
                class:animate-pulse={pageState === 'loading' || pageState === 'validating'}
            >
                <Logo class="w-full" />
            </div>
        </div>

        {#if pageState === 'validating'}
            <!-- Validating token -->
            <div class="invite-reveal flex flex-col items-center gap-3 delay-1">
                <LoaderCircle size={20} class="animate-spin text-muted-foreground" />
                <p class="text-sm text-muted-foreground">Validating invite...</p>
            </div>

        {:else if pageState === 'invalid'}
            <!-- Invalid / expired token -->
            <div class="invite-reveal flex flex-col items-center gap-4 delay-1">
                <div class="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                    <ShieldAlert size={22} class="text-destructive" />
                </div>
                <div class="text-center">
                    <p class="font-heading text-lg font-semibold tracking-tight text-foreground">
                        Invite not valid
                    </p>
                    <p class="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                        {#if errorMessage}
                            {errorMessage}
                        {:else}
                            This link has expired or has already been used.<br />
                            Ask the server owner for a new invite.
                        {/if}
                    </p>
                </div>
                <a
                    href="/"
                    class="mt-2 text-xs tracking-wide text-muted-foreground/50 transition-colors hover:text-foreground"
                >
                    Back to login
                </a>
            </div>

        {:else}
            <!-- Valid token — registration form -->
            <div class="invite-reveal mb-8 text-center delay-1">
                <p class="font-heading text-xl font-semibold tracking-tight text-foreground">
                    You've been invited
                </p>
                <p class="mt-1.5 text-sm text-muted-foreground">
                    Choose how you'd like to create your account.
                </p>
                {#if expiresAt}
                    <p class="mt-3 inline-flex items-center gap-1.5 text-[11px] text-muted-foreground/50">
                        <Clock size={11} />
                        {formatExpiry(expiresAt)}
                    </p>
                {/if}
            </div>

            <form onsubmit={handleSubmit} class="invite-reveal flex flex-col gap-4 delay-2">
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
                        class="h-10 w-full rounded-md border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {pageState === 'error' ? 'border-destructive' : 'border-border'}"
                        placeholder="your username"
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
                            type={showPassword ? 'text' : 'password'}
                            bind:value={password}
                            autocomplete="new-password"
                            disabled={isSubmitting}
                            class="h-10 w-full rounded-md border bg-background pr-10 pl-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {pageState === 'error' ? 'border-destructive' : 'border-border'}"
                            placeholder="••••••••"
                        />
                        <button
                            type="button"
                            onclick={() => (showPassword = !showPassword)}
                            class="absolute top-1/2 right-3 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                            aria-label={showPassword ? 'Hide password' : 'Show password'}
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
                    disabled={isSubmitting || username.length < 3 || password.length < 8}
                    class="mt-2 flex h-10 w-full items-center justify-center gap-2 rounded-md font-heading text-sm font-semibold tracking-wide transition-all disabled:cursor-not-allowed {pageState === 'success' ? 'bg-emerald-500 text-white' : pageState === 'error' ? 'animate-shake bg-red-600 text-white' : 'bg-primary text-primary-foreground hover:opacity-90 active:opacity-80'}"
                >
                    {#if pageState === 'loading'}
                        <LoaderCircle size={16} class="animate-spin" />
                        Creating account...
                    {:else if pageState === 'success'}
                        <Check size={16} />
                        Welcome aboard
                    {:else if pageState === 'error'}
                        <X size={16} />
                        {errorMessage}
                    {:else}
                        Create Account
                    {/if}
                </button>
            </form>

            <div class="invite-reveal my-6 flex items-center gap-3 delay-3">
                <div class="h-px flex-1 bg-border"></div>
                <span class="text-xs text-muted-foreground/50">or</span>
                <div class="h-px flex-1 bg-border"></div>
            </div>

            <button
                type="button"
                disabled={isSubmitting}
                onclick={handleCfxRegister}
                class="invite-reveal flex h-10 w-full items-center justify-center gap-2.5 rounded-md bg-[#F40552] text-sm font-semibold text-white transition-opacity delay-3 hover:opacity-90 active:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
            >
                <Cfxre class="h-4 w-auto" />
                Register with Cfx.re
            </button>

            <p class="invite-reveal mt-8 text-center text-xs text-muted-foreground/50 delay-4">
                Already have an account?
                <a href="/" class="text-muted-foreground transition-colors hover:text-foreground">Sign in</a>
            </p>
        {/if}
    </div>
</main>

<style>
    @keyframes shake {
        0%,
        100% {
            transform: translateX(0);
        }
        10%,
        30%,
        50%,
        70%,
        90% {
            transform: translateX(-4px);
        }
        20%,
        40%,
        60%,
        80% {
            transform: translateX(4px);
        }
    }

    :global(.animate-shake) {
        animation: shake 0.5s ease-in-out;
    }

    .invite-reveal {
        animation: reveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }

    .delay-1 { animation-delay: 0.08s; }
    .delay-2 { animation-delay: 0.16s; }
    .delay-3 { animation-delay: 0.24s; }
    .delay-4 { animation-delay: 0.32s; }

    @keyframes reveal {
        from {
            opacity: 0;
            transform: translateY(12px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
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
