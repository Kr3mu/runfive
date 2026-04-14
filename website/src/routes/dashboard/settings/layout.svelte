<script lang="ts">
    import type { Snippet } from "svelte";
    import { createQuery } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import { canGlobal } from "$lib/permissions.svelte";
    import { isActive, navigate } from "sv-router/generated";
    import User from "@lucide/svelte/icons/user";
    import ShieldCheck from "@lucide/svelte/icons/shield-check";
    import Users from "@lucide/svelte/icons/users";
    import Shield from "@lucide/svelte/icons/shield";
    import Plug from "@lucide/svelte/icons/plug";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";

    let { children }: { children: Snippet } = $props();

    const authQuery = createQuery(() => authQueryOptions());
    const user = $derived(authQuery.data);

    const canMembers = $derived(canGlobal(user, "users", "read"));
    const canRoles = $derived(canGlobal(user, "roles", "read"));
    const isOwner = $derived(user?.isOwner ?? false);

    // Permission guard: deep links to gated tabs bounce to /profile.
    $effect(() => {
        if (!user) return;
        if (isActive("/dashboard/settings/members" as any) && !canMembers) {
            void navigate("/dashboard/settings/profile" as any);
        } else if (isActive("/dashboard/settings/roles" as any) && !canRoles) {
            void navigate("/dashboard/settings/profile" as any);
        } else if (isActive("/dashboard/settings/providers" as any) && !isOwner) {
            void navigate("/dashboard/settings/profile" as any);
        }
    });

    interface Tab {
        href: string;
        label: string;
        icon: typeof User;
        group: "you" | "panel";
    }

    const tabs = $derived<Tab[]>([
        { href: "/dashboard/settings/profile", label: "Profile", icon: User, group: "you" },
        { href: "/dashboard/settings/sign-in", label: "Sign-in", icon: ShieldCheck, group: "you" },
        ...(canMembers
            ? [{ href: "/dashboard/settings/members", label: "Members", icon: Users, group: "panel" as const }]
            : []),
        ...(canRoles
            ? [{ href: "/dashboard/settings/roles", label: "Roles", icon: Shield, group: "panel" as const }]
            : []),
        ...(isOwner
            ? [{ href: "/dashboard/settings/providers", label: "Providers", icon: Plug, group: "panel" as const }]
            : []),
    ]);

    const hasPanelGroup = $derived(tabs.some((t) => t.group === "panel"));
</script>

{#if authQuery.isLoading || !user}
    <div class="flex h-full items-center justify-center">
        <LoaderCircle size={18} class="animate-spin text-muted-foreground/40" />
    </div>
{:else}
    <div class="flex h-full flex-col overflow-hidden">
        <!-- Pinned header -->
        <div class="settings-reveal shrink-0 px-8 pt-8 pb-5">
            <h1 class="mb-1 font-heading text-lg font-semibold text-foreground">
                Settings
            </h1>
            <p class="text-sm text-muted-foreground">
                Manage your account{#if hasPanelGroup} and panel configuration{/if}
            </p>
        </div>

        <!-- Pinned tab strip -->
        <nav class="settings-reveal delay-1 shrink-0 border-b border-border px-8">
            <div class="tab-scroller flex items-center gap-0.5 overflow-x-auto">
                {#each tabs as tab, i}
                    {#if i > 0 && tabs[i - 1].group === "you" && tab.group === "panel"}
                        <div
                            class="mx-2 h-4 w-px shrink-0 bg-border/60"
                            aria-hidden="true"
                        ></div>
                    {/if}
                    {@const active = isActive(tab.href as any)}
                    <a
                        href={tab.href}
                        class="group relative flex shrink-0 items-center gap-2 px-3.5 py-3 text-[13px] font-medium transition-colors
                            {active
                            ? 'text-foreground'
                            : 'text-muted-foreground/60 hover:text-foreground/80'}"
                    >
                        <tab.icon
                            size={14}
                            strokeWidth={active ? 2.2 : 1.8}
                            class={active
                                ? "text-primary"
                                : "text-muted-foreground/40 group-hover:text-muted-foreground/70"}
                        />
                        <span>{tab.label}</span>
                        {#if active}
                            <span
                                class="absolute inset-x-3 -bottom-px h-[2px] rounded-full bg-primary"
                            ></span>
                        {/if}
                    </a>
                {/each}
            </div>
        </nav>

        <!-- Scrollable content -->
        <div class="flex-1 overflow-y-auto px-8 py-8">
            {@render children()}
        </div>
    </div>
{/if}

<style>
    .tab-scroller {
        scrollbar-width: none;
    }
    .tab-scroller::-webkit-scrollbar {
        display: none;
    }

    .settings-reveal {
        animation: reveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }
    .delay-1 {
        animation-delay: 0.08s;
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
