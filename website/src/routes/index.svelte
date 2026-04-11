<script lang="ts">
    import { theme } from "$lib/theme.svelte";
    import Logo from "$lib/components/logo.svelte";
    import Cfxre from "$lib/components/icons/cfxre.svelte";
    import Sun from "@lucide/svelte/icons/sun";
    import Moon from "@lucide/svelte/icons/moon";
    import Eye from "@lucide/svelte/icons/eye";
    import EyeOff from "@lucide/svelte/icons/eye-off";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import Check from "@lucide/svelte/icons/check";
    import X from "@lucide/svelte/icons/x";
    import {
        login,
        register,
        fetchSetupStatus,
        fetchDiscordStatus,
    } from "$lib/api/auth";
    import Discord from "$lib/components/icons/discord.svelte";
    import { toast } from "svelte-sonner";

    type LoginState = "idle" | "loading" | "success" | "transition" | "error";

    let username = $state("");
    let password = $state("");
    let showPassword = $state(false);
    let loginState = $state<LoginState>("idle");
    let errorMessage = $state("");
    let logoEl = $state<HTMLElement | null>(null);
    let logoRect = $state<{ top: number; left: number; width: number } | null>(
        null,
    );
    let needsSetup = $state<boolean | null>(null);
    let discordStatus = $state(false);

    /** Eight individual hex characters for the setup code boxes. */
    let codeBoxes = $state<string[]>(['', '', '', '', '', '', '', '']);
    /** References to the eight box inputs for focus management. */
    const codeInputs: (HTMLInputElement | null)[] = $state([
        null, null, null, null, null, null, null, null,
    ]);

    /** True when every box holds a hex character. */
    const isCodeComplete = $derived(codeBoxes.every((c: string): boolean => c.length === 1));
    /** Backend wire format: "xxxx-xxxx". */
    const codeFormatted = $derived(
        `${codeBoxes.slice(0, 4).join('')}-${codeBoxes.slice(4).join('')}`,
    );

    const isSubmitting = $derived(
        loginState === "loading" ||
            loginState === "success" ||
            loginState === "transition",
    );

    const buttonLabel = $derived(needsSetup ? 'Create Account' : 'Sign in');
    const canSubmit = $derived(
        needsSetup === false || (needsSetup === true && isCodeComplete),
    );

    $effect((): void => {
        fetchSetupStatus()
            .then((status): void => {
                needsSetup = status.needsSetup;
                if (status.needsSetup) {
                    const params: URLSearchParams = new URLSearchParams(window.location.search);
                    const fromUrl: string | null = params.get('setup');
                    if (fromUrl) {
                        fillCodeFromString(fromUrl);
                    }
                }
            })
            .catch((): void => {
                needsSetup = false;
            });

        fetchDiscordStatus()
            .then((status) => {
                discordStatus = status;
            })
            .catch(() => {
                toast.error("Failed to fetch discord login status");
            });
    });

    /**
     * Normalises a raw string to uppercase hex chars and writes up to 8 of
     * them into the box state, leaving any leftover boxes blank. Accepts
     * both upper- and lowercase input; the backend is case-insensitive.
     */
    function fillCodeFromString(raw: string): void {
        const cleaned: string = raw.toUpperCase().replace(/[^0-9A-F]/g, '').slice(0, 8);
        for (let i = 0; i < 8; i += 1) {
            codeBoxes[i] = cleaned[i] ?? '';
        }
    }

    /** Handle a single character typed into box `index`. */
    function handleCodeInput(index: number, event: Event): void {
        const target: HTMLInputElement = event.target as HTMLInputElement;
        const cleaned: string = target.value.toUpperCase().replace(/[^0-9A-F]/g, '');
        if (cleaned.length === 0) {
            codeBoxes[index] = '';
            target.value = '';
            return;
        }
        const next: string = cleaned.slice(-1);
        codeBoxes[index] = next;
        target.value = next;
        if (index < 7) {
            codeInputs[index + 1]?.focus();
            codeInputs[index + 1]?.select();
        }
    }

    /** Handle navigation / deletion keys inside a code box. */
    function handleCodeKeydown(index: number, event: KeyboardEvent): void {
        if (event.key === 'Backspace' && codeBoxes[index] === '' && index > 0) {
            event.preventDefault();
            codeBoxes[index - 1] = '';
            codeInputs[index - 1]?.focus();
            return;
        }
        if (event.key === 'ArrowLeft' && index > 0) {
            event.preventDefault();
            codeInputs[index - 1]?.focus();
            codeInputs[index - 1]?.select();
            return;
        }
        if (event.key === 'ArrowRight' && index < 7) {
            event.preventDefault();
            codeInputs[index + 1]?.focus();
            codeInputs[index + 1]?.select();
        }
    }

    /** Distribute a pasted string across all boxes. */
    function handleCodePaste(event: ClipboardEvent): void {
        event.preventDefault();
        const text: string = event.clipboardData?.getData('text') ?? '';
        fillCodeFromString(text);
        const firstEmpty: number = codeBoxes.findIndex((c: string): boolean => c === '');
        const focusIdx: number = firstEmpty === -1 ? 7 : firstEmpty;
        codeInputs[focusIdx]?.focus();
    }

    function handleLogin(e: SubmitEvent): void {
        e.preventDefault();
        if (isSubmitting) return;
        if (needsSetup === null) return;
        if (needsSetup && !isCodeComplete) return;

        loginState = "loading";
        errorMessage = "";

        const action: Promise<unknown> = needsSetup
            ? register(username, password, codeFormatted)
            : login(username, password);

        action
            .then((): void => {
                loginState = "success";
                setTimeout((): void => {
                    if (logoEl) {
                        const rect: DOMRect = logoEl.getBoundingClientRect();
                        logoRect = {
                            top: rect.top,
                            left: rect.left,
                            width: rect.width,
                        };
                    }
                    loginState = "transition";
                    setTimeout((): void => {
                        window.location.href = "/dashboard";
                    }, 1200);
                }, 800);
            })
            .catch((err: unknown): void => {
                loginState = "error";
                errorMessage =
                    err instanceof Error
                        ? err.message
                        : "Authentication failed";
                setTimeout((): void => {
                    loginState = "idle";
                }, 2000);
            });
    }

    function handleCfxLogin(): void {
        if (isSubmitting) return;
        loginState = "loading";
        window.location.href = "/v1/auth/cfx";
    }

    function handleDiscordLogin(): void {
        if (isSubmitting) return;
        loginState = "loading";
        window.location.href = "/v1/auth/discord";
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
    class="relative flex flex-1 items-center justify-center bg-background px-4 transition-opacity duration-500"
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
                {#if needsSetup}
                    Initial Setup
                {:else}
                    Server Management
                {/if}
            </p>
        </div>

        <form onsubmit={handleLogin} class="flex flex-col gap-4">
            {#if needsSetup}
                <div class="flex flex-col gap-1.5">
                    <label
                        for="code-0"
                        class="text-xs font-medium tracking-wide text-muted-foreground uppercase"
                    >
                        Setup Code
                    </label>
                    <div class="flex items-center justify-between gap-1.5">
                        {#each codeBoxes as _box, i (i)}
                            <input
                                id={`code-${i}`}
                                bind:this={codeInputs[i]}
                                type="text"
                                inputmode="text"
                                autocomplete="off"
                                autocapitalize="off"
                                spellcheck="false"
                                maxlength="1"
                                value={codeBoxes[i]}
                                disabled={isSubmitting}
                                oninput={(e: Event): void => handleCodeInput(i, e)}
                                onkeydown={(e: KeyboardEvent): void => handleCodeKeydown(i, e)}
                                onpaste={handleCodePaste}
                                onfocus={(e: FocusEvent): void => {
                                    (e.target as HTMLInputElement).select();
                                }}
                                class="h-11 w-full min-w-0 rounded-md border bg-background text-center font-mono text-base font-semibold text-foreground uppercase outline-none transition-all focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {loginState === 'error' ? 'border-destructive' : 'border-border'}"
                            />
                            {#if i === 3}
                                <span class="select-none text-muted-foreground/60">&ndash;</span>
                            {/if}
                        {/each}
                    </div>
                    <p class="text-[11px] text-muted-foreground/60">
                        Printed to the server console at first startup.
                    </p>
                </div>
            {/if}
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
                    class="h-10 w-full rounded-md border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {loginState ===
                    'error'
                        ? 'border-destructive'
                        : 'border-border'}"
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
                        autocomplete={needsSetup
                            ? "new-password"
                            : "current-password"}
                        disabled={isSubmitting}
                        class="h-10 w-full rounded-md border bg-background pr-10 pl-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary disabled:opacity-50 {loginState ===
                        'error'
                            ? 'border-destructive'
                            : 'border-border'}"
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
                disabled={isSubmitting || needsSetup === null || !canSubmit}
                class="mt-2 flex h-10 w-full items-center justify-center gap-2 rounded-md font-heading text-sm font-semibold tracking-wide transition-all disabled:cursor-not-allowed {loginState === 'success' ? 'bg-emerald-500 text-white' : loginState === 'error' ? 'animate-shake bg-red-600 text-white' : 'bg-primary text-primary-foreground hover:opacity-90 active:opacity-80'}"
            >
                {#if loginState === "loading"}
                    <LoaderCircle size={16} class="animate-spin" />
                    {needsSetup ? "Creating account..." : "Signing in..."}
                {:else if loginState === "success"}
                    <Check size={16} />
                    {needsSetup ? "Account created" : "Welcome back"}
                {:else if loginState === "error"}
                    <X size={16} />
                    {errorMessage}
                {:else}
                    {buttonLabel}
                {/if}
            </button>
        </form>

        {#if !needsSetup}
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
                <Cfxre class="h-4 w-auto" />
                Sign in with Cfx.re
            </button>
            {#if discordStatus}
                <button
                    type="button"
                    disabled={isSubmitting}
                    onclick={handleDiscordLogin}
                    class="flex h-10 w-full mt-4 items-center justify-center gap-2.5 rounded-md bg-[#7289DA] text-sm font-semibold text-white transition-opacity hover:opacity-90 active:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
                >
                    <Discord class="h-4 w-auto" />
                    Sign in with Discord
                </button>
            {/if}
        {/if}

        <p class="mt-8 text-center text-xs text-muted-foreground/50">
            runfive &middot; open source &middot; done right
        </p>
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
