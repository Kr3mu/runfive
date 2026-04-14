<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import { useQueryClient } from "@tanstack/svelte-query";
    import {
        authQueryOptions,
        fetchDiscordStatus,
        fetchSessions,
        revokeSession,
        unlinkDiscord,
        type SessionEntry,
    } from "$lib/api/auth";
    import Cfxre from "$lib/components/icons/cfxre.svelte";
    import DiscordIcon from "$lib/components/icons/discord.svelte";
    import LinkIcon from "@lucide/svelte/icons/link";
    import Monitor from "@lucide/svelte/icons/monitor";
    import Smartphone from "@lucide/svelte/icons/smartphone";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import { toast } from "svelte-sonner";

    const authQuery = createQuery(() => authQueryOptions());
    const queryClient = useQueryClient();
    const user = $derived(authQuery.data);

    let sessions = $state<SessionEntry[]>([]);
    let isLoadingSessions = $state(true);
    let revokingId = $state<number | null>(null);
    let discordConfigured = $state(false);
    let isUnlinkingDiscord = $state(false);

    $effect((): void => {
        fetchSessions()
            .then((s: SessionEntry[]): void => {
                sessions = s;
                isLoadingSessions = false;
            })
            .catch((): void => {
                isLoadingSessions = false;
            });

        fetchDiscordStatus()
            .then((configured) => {
                discordConfigured = configured;
            })
            .catch(() => {
                toast.error("Failed to fetch Discord login status");
            });
    });

    function handleLinkCfx(): void {
        window.location.href = "/v1/auth/cfx";
    }

    function handleLinkDiscord(): void {
        window.location.href = "/v1/auth/discord";
    }

    async function handleUnlinkDiscord(): Promise<void> {
        isUnlinkingDiscord = true;
        try {
            await unlinkDiscord();
            await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
            toast.success("Discord account disconnected");
        } catch {
            toast.error("Failed to disconnect Discord");
        }
        isUnlinkingDiscord = false;
    }

    async function handleRevoke(id: number): Promise<void> {
        revokingId = id;
        try {
            await revokeSession(id);
            sessions = sessions.filter((s: SessionEntry): boolean => s.id !== id);
        } catch {
            // silently fail
        }
        revokingId = null;
    }

    function parseUserAgent(ua: string): string {
        if (ua.includes("Firefox")) return "Firefox";
        if (ua.includes("Edg/")) return "Edge";
        if (ua.includes("Chrome")) return "Chrome";
        if (ua.includes("Safari")) return "Safari";
        return "Browser";
    }

    function formatDate(iso: string): string {
        const d: Date = new Date(iso);
        const diff: number = Date.now() - d.getTime();
        const mins: number = Math.floor(diff / 60000);
        if (mins < 1) return "Just now";
        if (mins < 60) return `${mins}m ago`;
        const hours: number = Math.floor(mins / 60);
        if (hours < 24) return `${hours}h ago`;
        const days: number = Math.floor(hours / 24);
        return `${days}d ago`;
    }

    function isMobile(ua: string): boolean {
        return /mobile|android|iphone|ipad/i.test(ua);
    }
</script>

{#if user}
    <div class="flex max-w-3xl flex-col gap-8">
        <!-- Sign-in methods -->
        <section class="tab-reveal">
            <h2
                class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
            >
                <LinkIcon size={14} />
                Sign-in Methods
            </h2>
            <div class="space-y-2">
                <!-- Cfx.re -->
                <div
                    class="flex items-center justify-between rounded-lg border border-border bg-card p-4"
                >
                    <div class="flex items-center gap-3">
                        <div
                            class="flex h-9 w-9 items-center justify-center rounded-lg bg-[#F40552]/10"
                        >
                            <Cfxre class="h-4 w-auto text-[#F40552]" />
                        </div>
                        <div>
                            <p class="text-sm font-medium text-foreground">
                                Cfx.re
                            </p>
                            {#if user.providers.cfx}
                                <p class="text-xs text-muted-foreground">
                                    {user.providers.cfx.username}
                                </p>
                            {:else}
                                <p class="text-xs text-muted-foreground/50">
                                    Not connected
                                </p>
                            {/if}
                        </div>
                    </div>
                    {#if user.providers.cfx}
                        <span
                            class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-1 text-[11px] font-medium text-emerald-500"
                        >
                            <LinkIcon size={11} />
                            Linked
                        </span>
                    {:else}
                        <button
                            onclick={handleLinkCfx}
                            class="inline-flex items-center gap-1.5 rounded-md bg-[#F40552] px-3 py-1.5 text-xs font-semibold text-white transition-opacity hover:opacity-90"
                        >
                            <LinkIcon size={12} />
                            Connect
                        </button>
                    {/if}
                </div>

                <!-- Discord -->
                <div
                    class="flex items-center justify-between rounded-lg border border-border bg-card p-4"
                >
                    <div class="flex items-center gap-3">
                        <div
                            class="flex h-9 w-9 items-center justify-center rounded-lg bg-[#5865F2]/10"
                        >
                            <DiscordIcon class="h-4 w-4 text-[#5865F2]" />
                        </div>
                        <div>
                            <p class="text-sm font-medium text-foreground">
                                Discord
                            </p>
                            {#if user.providers.discord}
                                <p class="text-xs text-muted-foreground">
                                    {user.providers.discord.username}
                                </p>
                            {:else}
                                <p class="text-xs text-muted-foreground/50">
                                    Not connected
                                </p>
                            {/if}
                        </div>
                    </div>
                    {#if user.providers.discord}
                        <button
                            disabled={isUnlinkingDiscord}
                            onclick={handleUnlinkDiscord}
                            class="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-semibold text-foreground transition-colors hover:bg-muted disabled:opacity-50"
                            title="Disconnect Discord"
                        >
                            {#if isUnlinkingDiscord}
                                <LoaderCircle size={12} class="animate-spin" />
                            {:else}
                                <Trash2 size={12} />
                            {/if}
                            Disconnect
                        </button>
                    {:else}
                        <button
                            disabled={!discordConfigured}
                            onclick={handleLinkDiscord}
                            class="inline-flex items-center gap-1.5 rounded-md bg-[#5865F2] px-3 py-1.5 text-xs font-semibold text-white transition-opacity hover:opacity-90 disabled:opacity-40"
                            title={discordConfigured
                                ? "Connect Discord"
                                : "Discord OAuth not configured"}
                        >
                            <LinkIcon size={12} />
                            Connect
                        </button>
                    {/if}
                </div>
            </div>
        </section>

        <!-- Active Sessions -->
        <section class="tab-reveal delay-1">
            <h2
                class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
            >
                <Monitor size={14} />
                Active Sessions
            </h2>
            {#if isLoadingSessions}
                <div
                    class="flex items-center justify-center rounded-lg border border-border bg-card p-8"
                >
                    <LoaderCircle
                        size={18}
                        class="animate-spin text-muted-foreground"
                    />
                </div>
            {:else if sessions.length === 0}
                <div
                    class="rounded-lg border border-border bg-card p-6 text-center text-sm text-muted-foreground"
                >
                    No active sessions
                </div>
            {:else}
                <div class="space-y-2">
                    {#each sessions as session (session.id)}
                        <div
                            class="flex items-center justify-between rounded-lg border bg-card p-3 {session.isCurrent
                                ? 'border-primary/30'
                                : 'border-border'}"
                        >
                            <div class="flex items-center gap-3">
                                <div class="text-muted-foreground/40">
                                    {#if isMobile(session.userAgent)}
                                        <Smartphone size={16} />
                                    {:else}
                                        <Monitor size={16} />
                                    {/if}
                                </div>
                                <div>
                                    <div class="flex items-center gap-2">
                                        <p
                                            class="text-xs font-medium text-foreground"
                                        >
                                            {parseUserAgent(session.userAgent)}
                                        </p>
                                        {#if session.isCurrent}
                                            <span
                                                class="rounded-full bg-primary/10 px-1.5 py-0.5 text-[9px] font-semibold text-primary"
                                            >
                                                This device
                                            </span>
                                        {/if}
                                    </div>
                                    <p
                                        class="text-[11px] text-muted-foreground/60"
                                    >
                                        {formatDate(session.lastSeenAt)}
                                    </p>
                                </div>
                            </div>
                            {#if !session.isCurrent}
                                <button
                                    onclick={() => handleRevoke(session.id)}
                                    disabled={revokingId === session.id}
                                    class="rounded-md p-1.5 text-muted-foreground/40 transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
                                    title="Revoke session"
                                >
                                    {#if revokingId === session.id}
                                        <LoaderCircle
                                            size={14}
                                            class="animate-spin"
                                        />
                                    {:else}
                                        <Trash2 size={14} />
                                    {/if}
                                </button>
                            {/if}
                        </div>
                    {/each}
                </div>
            {/if}
        </section>
    </div>
{/if}

<style>
    .tab-reveal {
        animation: reveal 0.5s cubic-bezier(0.16, 1, 0.3, 1) both;
        animation-delay: 0.12s;
    }
    .delay-1 {
        animation-delay: 0.22s;
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
