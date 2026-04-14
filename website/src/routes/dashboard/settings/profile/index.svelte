<script lang="ts">
    import { createQuery } from "@tanstack/svelte-query";
    import {
        authQueryOptions,
        fetchSessions,
        type SessionEntry,
    } from "$lib/api/auth";
    import { serversQueryOptions } from "$lib/api/servers";
    import Cfxre from "$lib/components/icons/cfxre.svelte";
    import DiscordIcon from "$lib/components/icons/discord.svelte";
    import Shield from "@lucide/svelte/icons/shield";
    import User from "@lucide/svelte/icons/user";
    import Globe from "@lucide/svelte/icons/globe";
    import ServerIcon from "@lucide/svelte/icons/server";
    import Monitor from "@lucide/svelte/icons/monitor";
    import Smartphone from "@lucide/svelte/icons/smartphone";
    import LinkIcon from "@lucide/svelte/icons/link";
    import ArrowRight from "@lucide/svelte/icons/arrow-right";

    const authQuery = createQuery(() => authQueryOptions());
    const user = $derived(authQuery.data);

    const serversQuery = createQuery(() => serversQueryOptions());
    const allServers = $derived(serversQuery.data ?? []);

    let sessions = $state<SessionEntry[]>([]);
    let isLoadingSessions = $state(true);

    $effect((): void => {
        fetchSessions()
            .then((s: SessionEntry[]): void => {
                sessions = s;
                isLoadingSessions = false;
            })
            .catch((): void => {
                isLoadingSessions = false;
            });
    });

    const currentSession = $derived(sessions.find((s) => s.isCurrent));
    const sessionCount = $derived(sessions.length);

    const accessSummary = $derived.by((): string => {
        if (!user) return "";
        const parts: string[] = [];
        if (user.isOwner) {
            parts.push("Full panel access");
        } else {
            const n = Object.keys(user.serverPermissions ?? {}).length;
            if (n > 0) parts.push(`${n} server${n === 1 ? "" : "s"}`);
            if (user.globalRole)
                parts.push(`${user.globalRole.name} panel role`);
            if (parts.length === 0) parts.push("Member");
        }
        const methods: string[] = [];
        if (user.providers.cfx) methods.push("Cfx.re");
        if (user.providers.discord) methods.push("Discord");
        if (methods.length > 0) parts.push(`${methods.join(" + ")} linked`);
        return parts.join(" · ");
    });

    interface ServerAccessRow {
        id: string;
        name: string;
        roleName: string;
        roleColor: string;
    }

    const serverAccessRows = $derived.by<ServerAccessRow[]>(() => {
        if (!user) return [];
        if (user.isOwner) {
            return allServers.map((s) => ({
                id: s.id,
                name: s.name,
                roleName: "Owner",
                roleColor: "#f59e0b",
            }));
        }
        return Object.entries(user.serverPermissions ?? {}).map(
            ([id, entry]) => ({
                id,
                name: allServers.find((s) => s.id === id)?.name ?? id,
                roleName: entry.role.name,
                roleColor: entry.role.color,
            }),
        );
    });

    const hasAccessInfo = $derived(
        (user?.isOwner ?? false) ||
            (user?.globalRole ?? null) !== null ||
            serverAccessRows.length > 0,
    );

    function parseUserAgent(ua: string): string {
        if (ua.includes("Firefox")) return "Firefox";
        if (ua.includes("Edg/")) return "Edge";
        if (ua.includes("Chrome")) return "Chrome";
        if (ua.includes("Safari")) return "Safari";
        return "Browser";
    }

    function isMobile(ua: string): boolean {
        return /mobile|android|iphone|ipad/i.test(ua);
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
</script>

{#if user}
    <div class="flex max-w-4xl flex-col gap-6">
        <!-- Identity -->
        <section class="tab-reveal">
            <h2
                class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
            >
                <User size={14} />
                Identity
            </h2>
            <div class="rounded-lg border border-border bg-card p-5">
                <div class="flex items-center gap-4">
                    <div
                        class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary"
                    >
                        <span class="text-xl font-bold"
                            >{user.username.charAt(0).toUpperCase()}</span
                        >
                    </div>
                    <div class="min-w-0 flex-1">
                        <div class="flex flex-wrap items-center gap-2">
                            <p
                                class="text-lg font-semibold text-foreground leading-tight"
                            >
                                {user.username}
                            </p>
                            {#if user.isOwner}
                                <span
                                    class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-500"
                                >
                                    <Shield size={10} />
                                    Owner
                                </span>
                            {:else if user.globalRole}
                                <span
                                    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium"
                                    style="background-color: {user.globalRole
                                        .color}20; color: {user.globalRole
                                        .color}"
                                >
                                    {user.globalRole.name}
                                </span>
                            {/if}
                        </div>
                        {#if accessSummary}
                            <p
                                class="mt-1 text-[12px] text-muted-foreground/70"
                            >
                                {accessSummary}
                            </p>
                        {/if}
                    </div>
                </div>
            </div>
        </section>

        <!-- Access Scope -->
        {#if hasAccessInfo}
            <section class="tab-reveal delay-1">
                <h2
                    class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
                >
                    <Shield size={14} />
                    Access Scope
                </h2>
                <div
                    class="divide-y divide-border/40 overflow-hidden rounded-lg border border-border bg-card"
                >
                    <!-- Global row -->
                    <div class="flex items-center gap-4 px-4 py-3">
                        <div
                            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary/10"
                        >
                            <Globe size={15} class="text-primary" />
                        </div>
                        <div class="min-w-0 flex-1">
                            <p class="text-[13px] font-medium text-foreground">
                                Panel
                            </p>
                            <p class="text-[11px] text-muted-foreground/60">
                                {#if user.isOwner}
                                    Full access to all panel resources
                                {:else if user.globalRole}
                                    Panel-wide role applied across all servers
                                {:else}
                                    No panel-wide permissions
                                {/if}
                            </p>
                        </div>
                        <div class="shrink-0">
                            {#if user.isOwner}
                                <span
                                    class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/10 px-2 py-1 text-[10px] font-medium text-amber-500"
                                >
                                    <span
                                        class="h-1.5 w-1.5 rounded-full bg-amber-500"
                                    ></span>
                                    Owner
                                </span>
                            {:else if user.globalRole}
                                <span
                                    class="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[10px] font-medium"
                                    style="background-color: {user.globalRole
                                        .color}20; color: {user.globalRole
                                        .color}"
                                >
                                    <span
                                        class="h-1.5 w-1.5 rounded-full"
                                        style="background-color: {user
                                            .globalRole.color}"
                                    ></span>
                                    {user.globalRole.name}
                                </span>
                            {:else}
                                <span
                                    class="text-[10px] text-muted-foreground/30"
                                    >None</span
                                >
                            {/if}
                        </div>
                    </div>

                    <!-- Per-server rows -->
                    {#each serverAccessRows as srv (srv.id)}
                        <div class="flex items-center gap-4 px-4 py-3">
                            <div
                                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted/60"
                            >
                                <ServerIcon
                                    size={14}
                                    class="text-muted-foreground/60"
                                />
                            </div>
                            <div class="min-w-0 flex-1">
                                <p
                                    class="truncate text-[13px] font-medium text-foreground"
                                >
                                    {srv.name}
                                </p>
                                {#if srv.name !== srv.id}
                                    <p
                                        class="truncate font-mono text-[10px] text-muted-foreground/40"
                                    >
                                        {srv.id}
                                    </p>
                                {/if}
                            </div>
                            <div class="shrink-0">
                                <span
                                    class="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[10px] font-medium"
                                    style="background-color: {srv.roleColor}20; color: {srv.roleColor}"
                                >
                                    <span
                                        class="h-1.5 w-1.5 rounded-full"
                                        style="background-color: {srv.roleColor}"
                                    ></span>
                                    {srv.roleName}
                                </span>
                            </div>
                        </div>
                    {/each}

                    {#if serverAccessRows.length === 0 && user.isOwner}
                        <div
                            class="px-4 py-4 text-center text-[11px] text-muted-foreground/40"
                        >
                            No servers configured yet
                        </div>
                    {/if}
                </div>
            </section>
        {/if}

        <!-- Sign-in methods + Sessions (2 col grid) -->
        <section class="tab-reveal delay-2">
            <div class="grid gap-4 md:grid-cols-2">
                <!-- Sign-in methods -->
                <div class="flex flex-col rounded-lg border border-border bg-card">
                    <div
                        class="flex items-center justify-between border-b border-border/50 px-4 py-2.5"
                    >
                        <h3
                            class="flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
                        >
                            <LinkIcon size={13} />
                            Sign-in
                        </h3>
                    </div>
                    <div class="flex-1 divide-y divide-border/40">
                        <!-- Cfx.re -->
                        <div class="flex items-center gap-3 px-4 py-3">
                            <div
                                class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-[#F40552]/10"
                            >
                                <Cfxre class="h-3 w-auto text-[#F40552]" />
                            </div>
                            <div class="min-w-0 flex-1">
                                <p
                                    class="text-[12px] font-medium text-foreground"
                                >
                                    Cfx.re
                                </p>
                                {#if user.providers.cfx}
                                    <p
                                        class="truncate text-[10px] text-muted-foreground/60"
                                    >
                                        {user.providers.cfx.username}
                                    </p>
                                {:else}
                                    <p
                                        class="text-[10px] text-muted-foreground/40"
                                    >
                                        Not linked
                                    </p>
                                {/if}
                            </div>
                            {#if user.providers.cfx}
                                <span
                                    class="shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-500"
                                >
                                    Linked
                                </span>
                            {/if}
                        </div>
                        <!-- Discord -->
                        <div class="flex items-center gap-3 px-4 py-3">
                            <div
                                class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-[#5865F2]/10"
                            >
                                <DiscordIcon
                                    class="h-3 w-3 text-[#5865F2]"
                                />
                            </div>
                            <div class="min-w-0 flex-1">
                                <p
                                    class="text-[12px] font-medium text-foreground"
                                >
                                    Discord
                                </p>
                                {#if user.providers.discord}
                                    <p
                                        class="truncate text-[10px] text-muted-foreground/60"
                                    >
                                        {user.providers.discord.username}
                                    </p>
                                {:else}
                                    <p
                                        class="text-[10px] text-muted-foreground/40"
                                    >
                                        Not linked
                                    </p>
                                {/if}
                            </div>
                            {#if user.providers.discord}
                                <span
                                    class="shrink-0 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-500"
                                >
                                    Linked
                                </span>
                            {/if}
                        </div>
                    </div>
                    <a
                        href="/dashboard/settings/sign-in"
                        data-view-transition
                        class="flex items-center justify-between border-t border-border/50 px-4 py-2.5 text-[11px] font-medium text-muted-foreground/60 transition-colors hover:bg-muted/30 hover:text-foreground"
                    >
                        <span>Manage sign-in methods</span>
                        <ArrowRight size={12} />
                    </a>
                </div>

                <!-- Sessions -->
                <div class="flex flex-col rounded-lg border border-border bg-card">
                    <div
                        class="flex items-center justify-between border-b border-border/50 px-4 py-2.5"
                    >
                        <h3
                            class="flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase"
                        >
                            <Monitor size={13} />
                            Sessions
                        </h3>
                        {#if !isLoadingSessions}
                            <span
                                class="rounded bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-bold text-primary tabular-nums"
                            >
                                {sessionCount}
                            </span>
                        {/if}
                    </div>
                    <div class="flex-1 px-4 py-3">
                        {#if isLoadingSessions}
                            <div class="flex items-center gap-2 text-[11px] text-muted-foreground/40">
                                <span class="h-7 w-7 animate-pulse rounded-md bg-muted/60"></span>
                                <span class="flex-1 animate-pulse">
                                    <span class="block h-2 w-24 rounded bg-muted/60"></span>
                                    <span class="mt-1.5 block h-2 w-16 rounded bg-muted/40"></span>
                                </span>
                            </div>
                        {:else if currentSession}
                            <div class="flex items-center gap-3">
                                <div
                                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary"
                                >
                                    {#if isMobile(currentSession.userAgent)}
                                        <Smartphone size={13} />
                                    {:else}
                                        <Monitor size={13} />
                                    {/if}
                                </div>
                                <div class="min-w-0 flex-1">
                                    <div class="flex items-center gap-1.5">
                                        <p
                                            class="text-[12px] font-medium text-foreground"
                                        >
                                            {parseUserAgent(
                                                currentSession.userAgent,
                                            )}
                                        </p>
                                        <span
                                            class="rounded-full bg-primary/10 px-1.5 py-0.5 text-[9px] font-semibold text-primary"
                                        >
                                            This device
                                        </span>
                                    </div>
                                    <p
                                        class="text-[10px] text-muted-foreground/60"
                                    >
                                        Active {formatDate(
                                            currentSession.lastSeenAt,
                                        )}
                                    </p>
                                </div>
                            </div>
                        {:else}
                            <p class="text-[11px] text-muted-foreground/40">
                                No active session data
                            </p>
                        {/if}
                    </div>
                    <a
                        href="/dashboard/settings/sign-in"
                        data-view-transition
                        class="flex items-center justify-between border-t border-border/50 px-4 py-2.5 text-[11px] font-medium text-muted-foreground/60 transition-colors hover:bg-muted/30 hover:text-foreground"
                    >
                        <span>Manage sessions</span>
                        <ArrowRight size={12} />
                    </a>
                </div>
            </div>
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
    .delay-2 {
        animation-delay: 0.32s;
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
