<script lang="ts">
    import { theme } from "$lib/theme.svelte";
    import Logo from "$lib/components/logo.svelte";
    import Sun from "@lucide/svelte/icons/sun";
    import Moon from "@lucide/svelte/icons/moon";
    import LayoutDashboard from "@lucide/svelte/icons/layout-dashboard";
    import Users from "@lucide/svelte/icons/users";
    import Terminal from "@lucide/svelte/icons/terminal";
    import ShieldBan from "@lucide/svelte/icons/shield-ban";
    import Settings from "@lucide/svelte/icons/settings";
    import Shield from "@lucide/svelte/icons/shield";
    import PanelLeftClose from "@lucide/svelte/icons/panel-left-close";
    import PanelLeftOpen from "@lucide/svelte/icons/panel-left-open";
    import LogOut from "@lucide/svelte/icons/log-out";
    import Activity from "@lucide/svelte/icons/activity";
    import Github from "$lib/components/icons/github.svelte";
    import Discord from "$lib/components/icons/discord.svelte";
    import Info from "@lucide/svelte/icons/info";
    import Pencil from "@lucide/svelte/icons/pencil";
    import Check from "@lucide/svelte/icons/check";
    import Plus from "@lucide/svelte/icons/plus";
    import Share2 from "@lucide/svelte/icons/share-2";
    import ClipboardCheck from "@lucide/svelte/icons/clipboard-check";
    import ServerSwitcher from "./server-switcher.svelte";
    import GraduationCap from "@lucide/svelte/icons/graduation-cap";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { widgetRegistry } from "$lib/widget-registry";
    import { encodeLayout } from "$lib/layout-codec";
    import { authQueryOptions, logout } from "$lib/api/auth";
    import { canGlobal, canServer } from "$lib/permissions.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import { createQuery } from "@tanstack/svelte-query";
    import { isActive } from "sv-router/generated";

    let isLoggingOut = $state(false);

    let pathname = $state(window.location.pathname);
    $effect((): (() => void) => {
        const update = (): void => { pathname = window.location.pathname; };
        window.addEventListener("popstate", update);
        const observer = new MutationObserver(update);
        observer.observe(document.querySelector("head title") ?? document.head, { childList: true, subtree: true, characterData: true });
        return (): void => { window.removeEventListener("popstate", update); observer.disconnect(); };
    });
    const isUsersPage = $derived(pathname.startsWith("/dashboard/users"));

    function handleLogout(): void {
        if (isLoggingOut) return;
        isLoggingOut = true;
        logout()
            .then((): void => {
                window.location.href = "/";
            })
            .catch((): void => {
                isLoggingOut = false;
            });
    }

    let shareState = $state<"idle" | "copied">("idle");

    function shareDashboard(): void {
        try {
            const code = encodeLayout(dashboardState.layout);
            navigator.clipboard.writeText(code);
            shareState = "copied";
            setTimeout(() => (shareState = "idle"), 2000);
        } catch {
            // fallback: prompt
            const code = encodeLayout(dashboardState.layout);
            prompt("Share code:", code);
        }
    }

    interface Props {
        collapsed?: boolean;
        activeWidgets?: string[];
        onaddwidget?: (id: string) => void;
    }

    let {
        collapsed = $bindable(false),
        activeWidgets = [],
        onaddwidget,
    }: Props = $props();

    const availableWidgets = $derived(
        widgetRegistry.filter((w) => !activeWidgets.includes(w.id)),
    );

    // Map widget icon names to imported components
    const widgetIconMap: Record<string, typeof Terminal> = {
        terminal: Terminal,
        users: Users,
        activity: Activity,
        "shield-ban": ShieldBan,
    };

    const allNavItems = [
        { icon: LayoutDashboard, label: "Dashboard", href: "/dashboard", resource: "dashboard" },
        { icon: Users, label: "Players", href: "/dashboard/players", resource: "players" },
        { icon: Terminal, label: "Console", href: "/dashboard/console", resource: "console" },
        { icon: ShieldBan, label: "Bans", href: "/dashboard/bans", resource: "bans" },
    ];

    const authQuery = createQuery(() => authQueryOptions());

    const user = $derived(authQuery.data);
    const isOwner = $derived(user?.isOwner ?? false);
    const currentServerId = $derived(serverState.selectedId);

    const navItems = $derived(
        allNavItems.filter((item) => canServer(user, currentServerId, item.resource, "read")),
    );
    const canViewUsers = $derived(canGlobal(user, "users", "read"));
    const canViewRoles = $derived(canGlobal(user, "roles", "read"));
</script>

<aside
    class="group/sidebar flex h-full flex-col bg-sidebar transition-all duration-300 ease-out
        {collapsed ? 'w-13' : 'w-55'}"
>
    <!-- Logo Row -->
    <div
        class="flex h-12 shrink-0 items-center {collapsed
            ? 'justify-center px-0'
            : 'justify-between px-4'}"
    >
        {#if !collapsed}
            <a href="/dashboard" class="flex items-center">
                <Logo class="w-20" />
            </a>
        {/if}
        <button
            onclick={() => (collapsed = !collapsed)}
            class="rounded-md p-1 text-muted-foreground/40 transition-colors hover:text-foreground"
        >
            {#if collapsed}
                <PanelLeftOpen size={16} />
            {:else}
                <PanelLeftClose size={16} />
            {/if}
        </button>
    </div>

    <!-- Server Switcher (active server preview + dropdown to change) -->
    <div class="shrink-0 {collapsed ? 'px-1.5 py-2' : 'px-3 pb-3'}">
        <ServerSwitcher {collapsed} />
    </div>

    <!-- Divider -->
    <div class="mx-3 h-px bg-border/50"></div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto {collapsed ? 'px-1.5' : 'px-2'} py-2">
        {#each navItems as item}
            <!-- ignore type missmatch (as any) -->
            {@const active = isActive(item.href as any)}
            <a
                href={item.href}
                data-view-transition
                class="group flex items-center rounded-md transition-all duration-150
                    {collapsed
                    ? 'mb-1 justify-center p-2'
                    : 'mb-0.5 gap-2.5 px-2.5 py-1.75'}
                    {active
                    ? 'bg-primary/12 text-primary'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
                title={collapsed ? item.label : undefined}
            >
                <item.icon
                    size={collapsed ? 17 : 15}
                    strokeWidth={active ? 2.2 : 1.8}
                    class="shrink-0 {active
                        ? 'text-primary'
                        : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
                />
                {#if !collapsed}
                    <span
                        class="text-[12.5px] font-medium {active
                            ? 'font-semibold'
                            : ''}">{item.label}</span
                    >
                    {#if active}
                        <div
                            class="ml-auto h-1 w-1 rounded-full bg-primary"
                        ></div>
                    {/if}
                {/if}
            </a>
        {/each}
    </nav>

    <!-- Widget Picker (edit mode only) -->
    {#if dashboardState.editing && availableWidgets.length > 0 && !collapsed}
        <div class="shrink-0 px-2 pb-2">
            <div class="mx-0.5 mb-2 h-px bg-border/50"></div>
            <p
                class="mb-1.5 px-2.5 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase"
            >
                Add Widget
            </p>
            {#each availableWidgets as widget}
                <button
                    onclick={() => onaddwidget?.(widget.id)}
                    class="group flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.75 text-muted-foreground/50 transition-colors hover:bg-primary/8 hover:text-primary"
                >
                    {#if widgetIconMap[widget.icon]}
                        {@const WidgetIcon =
                            widgetIconMap[widget.icon]}<WidgetIcon
                            size={15}
                            strokeWidth={1.8}
                            class="shrink-0"
                        />
                    {:else}
                        <Plus size={15} strokeWidth={1.8} class="shrink-0" />
                    {/if}
                    <span class="text-[12.5px]">{widget.label}</span>
                    <Plus
                        size={12}
                        class="ml-auto opacity-0 transition-opacity group-hover:opacity-100"
                    />
                </button>
            {/each}
        </div>
    {/if}

    {#if dashboardState.editing && availableWidgets.length > 0 && collapsed}
        <div class="shrink-0 px-1.5 pb-2">
            <div class="mb-1 h-px bg-border/50"></div>
            {#each availableWidgets as widget}
                <button
                    onclick={() => onaddwidget?.(widget.id)}
                    class="flex w-full items-center justify-center rounded-md p-2 text-muted-foreground/40 transition-colors hover:bg-primary/8 hover:text-primary"
                    title="Add {widget.label}"
                >
                    <Plus size={17} strokeWidth={1.8} />
                </button>
            {/each}
        </div>
    {/if}

    <!-- Panel section (permission-based) -->
    {#if canViewUsers || canViewRoles}
        <div class="shrink-0 {collapsed ? 'px-1.5' : 'px-2'} pb-1">
            <div class="{collapsed ? '' : 'mx-0.5'} mb-2 h-px bg-border/50"></div>
            {#if !collapsed}
                <p class="mb-1.5 px-2.5 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">
                    Panel
                </p>
            {/if}
            {#if canViewUsers}
                <a
                    href="/dashboard/users"
                    data-view-transition
                    class="group flex items-center rounded-md transition-all duration-150
                        {collapsed ? 'mb-1 justify-center p-2' : 'mb-0.5 gap-2.5 px-2.5 py-[7px]'}
                        {isUsersPage
                            ? 'bg-primary/12 text-primary'
                            : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
                    title={collapsed ? "Users" : undefined}
                >
                    <Users
                        size={collapsed ? 17 : 15}
                        strokeWidth={isUsersPage ? 2.2 : 1.8}
                        class="shrink-0 {isUsersPage ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
                    />
                    {#if !collapsed}
                        <span class="text-[12.5px] font-medium {isUsersPage ? 'font-semibold' : ''}">Users</span>
                        {#if isUsersPage}
                            <div class="ml-auto h-1 w-1 rounded-full bg-primary"></div>
                        {/if}
                    {/if}
                </a>
            {/if}
            {#if canViewRoles}
                {@const isRolesPage = pathname.startsWith("/dashboard/roles")}
                <a
                    href="/dashboard/roles"
                    data-view-transition
                    class="group flex items-center rounded-md transition-all duration-150
                        {collapsed ? 'mb-1 justify-center p-2' : 'mb-0.5 gap-2.5 px-2.5 py-[7px]'}
                        {isRolesPage
                            ? 'bg-primary/12 text-primary'
                            : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
                    title={collapsed ? "Roles" : undefined}
                >
                    <Shield
                        size={collapsed ? 17 : 15}
                        strokeWidth={isRolesPage ? 2.2 : 1.8}
                        class="shrink-0 {isRolesPage ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
                    />
                    {#if !collapsed}
                        <span class="text-[12.5px] font-medium {isRolesPage ? 'font-semibold' : ''}">Roles</span>
                        {#if isRolesPage}
                            <div class="ml-auto h-1 w-1 rounded-full bg-primary"></div>
                        {/if}
                    {/if}
                </a>
            {/if}
        </div>
    {/if}

    <!-- Bottom -->
    <div class="shrink-0 {collapsed ? 'px-1.5' : 'px-2'} pb-2">
        <!-- Edit + Share -->
        <button
            onclick={() => dashboardState.toggle()}
            class="mb-0.5 flex w-full items-center rounded-md transition-all
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-1.75'}
                {dashboardState.editing
                ? 'bg-primary/12 text-primary'
                : 'text-muted-foreground/40 hover:bg-muted/50 hover:text-muted-foreground'}"
            title={collapsed
                ? dashboardState.editing
                    ? "Done editing"
                    : "Edit Dashboard"
                : undefined}
        >
            {#if dashboardState.editing}
                <Check
                    size={collapsed ? 17 : 15}
                    strokeWidth={2.2}
                    class="shrink-0"
                />
                {#if !collapsed}<span class="text-[12.5px] font-semibold"
                        >Done Editing</span
                    >{/if}
            {:else}
                <Pencil
                    size={collapsed ? 17 : 15}
                    strokeWidth={1.8}
                    class="shrink-0"
                />
                {#if !collapsed}<span class="text-[12.5px]">Edit Dashboard</span
                    >{/if}
            {/if}
        </button>
        {#if dashboardState.editing}
            <button
                onclick={shareDashboard}
                class="mb-1 flex w-full items-center rounded-md transition-all
                    {collapsed
                    ? 'justify-center p-2'
                    : 'gap-2.5 px-2.5 py-1.75'}
                    {shareState === 'copied'
                    ? 'text-emerald-500'
                    : 'text-muted-foreground/40 hover:bg-muted/50 hover:text-muted-foreground'}"
                title={collapsed ? "Share layout" : undefined}
            >
                {#if shareState === "copied"}
                    <ClipboardCheck
                        size={collapsed ? 17 : 15}
                        strokeWidth={2.2}
                        class="shrink-0"
                    />
                    {#if !collapsed}<span class="text-[12.5px] font-semibold"
                            >Copied!</span
                        >{/if}
                {:else}
                    <Share2
                        size={collapsed ? 17 : 15}
                        strokeWidth={1.8}
                        class="shrink-0"
                    />
                    {#if !collapsed}<span class="text-[12.5px]"
                            >Share Layout</span
                        >{/if}
                {/if}
            </button>
        {/if}

        <div class="mb-1 {collapsed ? '' : 'mx-0.5'} h-px bg-border/50"></div>
        <!-- Master Actions -->
        {#if isOwner}
            <a
                href="/dashboard/master"
                data-view-transition
                class="group flex items-center rounded-md transition-all duration-150
                    {collapsed
                    ? 'mb-1 justify-center p-2'
                    : 'mb-0.5 gap-2.5 px-2.5 py-1.75'}                
                    {isActive('/dashboard/master')
                    ? 'bg-primary/12 text-primary'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
                title={collapsed ? "Master Actions" : undefined}
            >
                <GraduationCap
                    size={collapsed ? 17 : 15}
                    strokeWidth={isActive("/dashboard/master") ? 2.2 : 1.8}
                    class="shrink-0 {isActive('/dashboard/master')
                        ? 'text-primary'
                        : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
                />
                {#if !collapsed}
                    <span
                        class="text-[12.5px] font-medium {isActive(
                            '/dashboard/master',
                        )
                            ? 'font-semibold'
                            : ''}">Master Actions</span
                    >
                    {#if isActive("/dashboard/master")}
                        <div
                            class="ml-auto h-1 w-1 rounded-full bg-primary"
                        ></div>
                    {/if}
                {/if}
            </a>
        {/if}
        <!-- Account Settings -->
        <a
            href="/dashboard/settings"
            data-view-transition
            class="group flex items-center rounded-md transition-all duration-150
                {collapsed
                ? 'mb-1 justify-center p-2'
                : 'mb-0.5 gap-2.5 px-2.5 py-1.75'}                
                {isActive('/dashboard/settings')
                ? 'bg-primary/12 text-primary'
                : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
            title={collapsed ? "Account Settings" : undefined}
        >
            <Settings
                size={collapsed ? 17 : 15}
                strokeWidth={isActive("/dashboard/settings") ? 2.2 : 1.8}
                class="shrink-0 {isActive('/dashboard/settings')
                    ? 'text-primary'
                    : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
            />
            {#if !collapsed}
                <span
                    class="text-[12.5px] font-medium {isActive(
                        '/dashboard/settings',
                    )
                        ? 'font-semibold'
                        : ''}">Account Settings</span
                >
                {#if isActive("/dashboard/settings")}
                    <div class="ml-auto h-1 w-1 rounded-full bg-primary"></div>
                {/if}
            {/if}
        </a>

        <button
            onclick={theme.toggle}
            class="flex w-full items-center rounded-md text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-1.75'}"
            title={collapsed ? "Toggle theme" : undefined}
        >
            {#if theme.value === "dark"}
                <Sun
                    size={collapsed ? 17 : 15}
                    strokeWidth={1.8}
                    class="text-muted-foreground/60"
                />
                {#if !collapsed}<span class="text-[12.5px]">Light Mode</span
                    >{/if}
            {:else}
                <Moon
                    size={collapsed ? 17 : 15}
                    strokeWidth={1.8}
                    class="text-muted-foreground/60"
                />
                {#if !collapsed}<span class="text-[12.5px]">Dark Mode</span
                    >{/if}
            {/if}
        </button>
        <button
            onclick={handleLogout}
            disabled={isLoggingOut}
            class="flex w-full items-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:opacity-50
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-1.75'}"
            title={collapsed ? "Sign out" : undefined}
        >
            <LogOut
                size={collapsed ? 17 : 15}
                strokeWidth={1.8}
                class="text-muted-foreground/60"
            />
            {#if !collapsed}<span class="text-[12.5px]"
                    >{isLoggingOut ? "Signing out..." : "Sign Out"}</span
                >{/if}
        </button>

        <!-- Mini footer links -->
        <div class="mt-1 {collapsed ? '' : 'mx-0.5'} h-px bg-border/30"></div>
        <div
            class="mt-1.5 flex items-center {collapsed
                ? 'flex-col gap-1.5'
                : 'justify-center gap-3'}"
        >
            <a
                href="https://github.com/Kr3mu/runfive"
                target="_blank"
                rel="noopener noreferrer"
                class="text-muted-foreground/20 transition-colors hover:text-foreground/60"
                title="GitHub"
            >
                <Github class="h-3 w-3" />
            </a>
            <a
                href="https://discord.gg"
                target="_blank"
                rel="noopener noreferrer"
                class="text-muted-foreground/20 transition-colors hover:text-foreground/60"
                title="Discord"
            >
                <Discord class="h-3 w-3" />
            </a>
            <a
                href="/about"
                data-view-transition
                class="text-muted-foreground/20 transition-colors hover:text-foreground/60"
                title="About"
            >
                <Info size={12} />
            </a>
        </div>
    </div>
</aside>
