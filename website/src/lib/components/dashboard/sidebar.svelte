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
    import PanelLeftClose from "@lucide/svelte/icons/panel-left-close";
    import PanelLeftOpen from "@lucide/svelte/icons/panel-left-open";
    import LogOut from "@lucide/svelte/icons/log-out";
    import Cpu from "@lucide/svelte/icons/cpu";
    import Activity from "@lucide/svelte/icons/activity";
    import HardDrive from "@lucide/svelte/icons/hard-drive";
    import Wifi from "@lucide/svelte/icons/wifi";
    import Github from "$lib/components/icons/github.svelte";
    import Discord from "$lib/components/icons/discord.svelte";
    import Info from "@lucide/svelte/icons/info";
    import Pencil from "@lucide/svelte/icons/pencil";
    import Check from "@lucide/svelte/icons/check";
    import Plus from "@lucide/svelte/icons/plus";
    import Share2 from "@lucide/svelte/icons/share-2";
    import ClipboardCheck from "@lucide/svelte/icons/clipboard-check";
    import { dashboardState } from "$lib/dashboard-state.svelte";
    import { widgetRegistry } from "$lib/widget-registry";
    import { encodeLayout } from "$lib/layout-codec";

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

    let { collapsed = $bindable(false), activeWidgets = [], onaddwidget }: Props = $props();

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

    const serverName = "RunFive Dev";
    const playerCount = 42;
    const maxPlayers = 64;
    const playerPercent = Math.round((playerCount / maxPlayers) * 100);

    const navItems = [
        { icon: LayoutDashboard, label: "Dashboard", href: "/dashboard", active: true },
        { icon: Users, label: "Players", href: "/dashboard/players", active: false },
        { icon: Terminal, label: "Console", href: "/dashboard/console", active: false },
        { icon: ShieldBan, label: "Bans", href: "/dashboard/bans", active: false },
        { icon: Settings, label: "Settings", href: "/dashboard/settings", active: false },
    ];

    const stats = [
        { icon: Cpu, label: "CPU", value: "23%", color: "text-emerald-400" },
        { icon: HardDrive, label: "RAM", value: "4.2G", color: "text-blue-400" },
        { icon: Activity, label: "Tick", value: "8.2ms", color: "text-primary" },
    ];
</script>

<aside
    class="group/sidebar flex h-full flex-col bg-sidebar transition-all duration-300 ease-out
        {collapsed ? 'w-[52px]' : 'w-[220px]'}"
>
    <!-- Logo Row -->
    <div class="flex h-12 shrink-0 items-center {collapsed ? 'justify-center px-0' : 'justify-between px-4'}">
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

    <!-- Server Status Block -->
    <div class="shrink-0 {collapsed ? 'px-1.5 py-2' : 'px-3 pb-3'}">
        {#if collapsed}
            <div class="flex flex-col items-center gap-2">
                <div class="h-2 w-2 rounded-full bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.6)]"></div>
                <span class="text-[9px] font-bold text-primary">{playerCount}</span>
            </div>
        {:else}
            <div class="rounded-lg border border-border/50 bg-background/50 p-3">
                <div class="mb-2 flex items-center justify-between">
                    <div class="flex items-center gap-2">
                        <div class="h-2 w-2 rounded-full bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.6)]"></div>
                        <span class="font-heading text-[11px] font-semibold text-foreground">{serverName}</span>
                    </div>
                    <Wifi size={12} class="text-emerald-500" />
                </div>

                <!-- Player bar -->
                <div class="mb-2.5">
                    <div class="mb-1 flex items-baseline justify-between">
                        <span class="text-[10px] text-muted-foreground">Players</span>
                        <span class="font-mono text-[10px] font-semibold text-foreground">
                            <span class="text-primary">{playerCount}</span><span class="text-muted-foreground">/{maxPlayers}</span>
                        </span>
                    </div>
                    <div class="h-1 overflow-hidden rounded-full bg-muted">
                        <div
                            class="h-full rounded-full bg-primary transition-all duration-500"
                            style="width: {playerPercent}%"
                        ></div>
                    </div>
                </div>

                <!-- Quick Stats -->
                <div class="flex justify-between">
                    {#each stats as stat}
                        <div class="flex items-center gap-1">
                            <stat.icon size={10} class="text-muted-foreground/50" />
                            <span class="font-mono text-[9px] font-medium {stat.color}">{stat.value}</span>
                        </div>
                    {/each}
                </div>
            </div>
        {/if}
    </div>

    <!-- Divider -->
    <div class="mx-3 h-px bg-border/50"></div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto {collapsed ? 'px-1.5' : 'px-2'} py-2">
        {#each navItems as item}
            <a
                href={item.href}
                data-view-transition
                class="group flex items-center rounded-md transition-all duration-150
                    {collapsed ? 'mb-1 justify-center p-2' : 'mb-0.5 gap-2.5 px-2.5 py-[7px]'}
                    {item.active
                        ? 'bg-primary/12 text-primary'
                        : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
                title={collapsed ? item.label : undefined}
            >
                <item.icon
                    size={collapsed ? 17 : 15}
                    strokeWidth={item.active ? 2.2 : 1.8}
                    class="shrink-0 {item.active ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground/70'}"
                />
                {#if !collapsed}
                    <span class="text-[12.5px] font-medium {item.active ? 'font-semibold' : ''}">{item.label}</span>
                    {#if item.active}
                        <div class="ml-auto h-1 w-1 rounded-full bg-primary"></div>
                    {/if}
                {/if}
            </a>
        {/each}
    </nav>

    <!-- Widget Picker (edit mode only) -->
    {#if dashboardState.editing && availableWidgets.length > 0 && !collapsed}
        <div class="shrink-0 px-2 pb-2">
            <div class="mx-0.5 mb-2 h-px bg-border/50"></div>
            <p class="mb-1.5 px-2.5 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">
                Add Widget
            </p>
            {#each availableWidgets as widget}
                <button
                    onclick={() => onaddwidget?.(widget.id)}
                    class="group flex w-full items-center gap-2.5 rounded-md px-2.5 py-[7px] text-muted-foreground/50 transition-colors hover:bg-primary/8 hover:text-primary"
                >
                    {#if widgetIconMap[widget.icon]}
                        <svelte:component this={widgetIconMap[widget.icon]} size={15} strokeWidth={1.8} class="shrink-0" />
                    {:else}
                        <Plus size={15} strokeWidth={1.8} class="shrink-0" />
                    {/if}
                    <span class="text-[12.5px]">{widget.label}</span>
                    <Plus size={12} class="ml-auto opacity-0 transition-opacity group-hover:opacity-100" />
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

    <!-- Bottom -->
    <div class="shrink-0 {collapsed ? 'px-1.5' : 'px-2'} pb-2">
        <!-- Edit + Share -->
        <button
            onclick={() => dashboardState.toggle()}
            class="mb-0.5 flex w-full items-center rounded-md transition-all
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-[7px]'}
                {dashboardState.editing
                    ? 'bg-primary/12 text-primary'
                    : 'text-muted-foreground/40 hover:bg-muted/50 hover:text-muted-foreground'}"
            title={collapsed ? (dashboardState.editing ? "Done editing" : "Edit Dashboard") : undefined}
        >
            {#if dashboardState.editing}
                <Check size={collapsed ? 17 : 15} strokeWidth={2.2} class="shrink-0" />
                {#if !collapsed}<span class="text-[12.5px] font-semibold">Done Editing</span>{/if}
            {:else}
                <Pencil size={collapsed ? 17 : 15} strokeWidth={1.8} class="shrink-0" />
                {#if !collapsed}<span class="text-[12.5px]">Edit Dashboard</span>{/if}
            {/if}
        </button>
        {#if dashboardState.editing}
            <button
                onclick={shareDashboard}
                class="mb-1 flex w-full items-center rounded-md transition-all
                    {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-[7px]'}
                    {shareState === 'copied'
                        ? 'text-emerald-500'
                        : 'text-muted-foreground/40 hover:bg-muted/50 hover:text-muted-foreground'}"
                title={collapsed ? "Share layout" : undefined}
            >
                {#if shareState === "copied"}
                    <ClipboardCheck size={collapsed ? 17 : 15} strokeWidth={2.2} class="shrink-0" />
                    {#if !collapsed}<span class="text-[12.5px] font-semibold">Copied!</span>{/if}
                {:else}
                    <Share2 size={collapsed ? 17 : 15} strokeWidth={1.8} class="shrink-0" />
                    {#if !collapsed}<span class="text-[12.5px]">Share Layout</span>{/if}
                {/if}
            </button>
        {/if}

        <div class="mb-1 {collapsed ? '' : 'mx-0.5'} h-px bg-border/50"></div>
        <button
            onclick={theme.toggle}
            class="flex w-full items-center rounded-md text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-[7px]'}"
            title={collapsed ? "Toggle theme" : undefined}
        >
            {#if theme.value === "dark"}
                <Sun size={collapsed ? 17 : 15} strokeWidth={1.8} class="text-muted-foreground/60" />
                {#if !collapsed}<span class="text-[12.5px]">Light Mode</span>{/if}
            {:else}
                <Moon size={collapsed ? 17 : 15} strokeWidth={1.8} class="text-muted-foreground/60" />
                {#if !collapsed}<span class="text-[12.5px]">Dark Mode</span>{/if}
            {/if}
        </button>
        <button
            class="flex w-full items-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive
                {collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-[7px]'}"
            title={collapsed ? "Sign out" : undefined}
        >
            <LogOut size={collapsed ? 17 : 15} strokeWidth={1.8} class="text-muted-foreground/60" />
            {#if !collapsed}<span class="text-[12.5px]">Sign Out</span>{/if}
        </button>

        <!-- Mini footer links -->
        <div class="mt-1 {collapsed ? '' : 'mx-0.5'} h-px bg-border/30"></div>
        <div class="mt-1.5 flex items-center {collapsed ? 'flex-col gap-1.5' : 'justify-center gap-3'}">
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
