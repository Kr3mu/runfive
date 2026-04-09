<script lang="ts">
    import Search from "@lucide/svelte/icons/search";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Download from "@lucide/svelte/icons/download";
    import ArrowRight from "@lucide/svelte/icons/arrow-right";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import X from "@lucide/svelte/icons/x";

    type LogLevel = "info" | "warn" | "error" | "debug" | "command";

    interface LogEntry {
        id: number;
        timestamp: string;
        level: LogLevel;
        source: string;
        message: string;
    }

    const mockLogs: LogEntry[] = [
        { id: 1, timestamp: "12:00:01", level: "info", source: "server", message: "Server started on port 30120" },
        { id: 2, timestamp: "12:00:01", level: "info", source: "server", message: "Loading resources..." },
        { id: 3, timestamp: "12:00:02", level: "info", source: "resource", message: "[es_extended] loaded successfully" },
        { id: 4, timestamp: "12:00:02", level: "info", source: "resource", message: "[qb-core] loaded successfully" },
        { id: 5, timestamp: "12:00:03", level: "warn", source: "resource", message: "[ox_inventory] deprecated function call in items.lua:142" },
        { id: 6, timestamp: "12:00:03", level: "info", source: "resource", message: "[ox_lib] v3.28.0 initialized" },
        { id: 7, timestamp: "12:00:04", level: "info", source: "server", message: "All 47 resources loaded" },
        { id: 8, timestamp: "12:00:05", level: "info", source: "server", message: "Server is ready, accepting connections" },
        { id: 9, timestamp: "12:01:12", level: "info", source: "player", message: "[Join] Kr3mu (steam:1100001abcdef12) connected" },
        { id: 10, timestamp: "12:01:14", level: "debug", source: "resource", message: "[qb-core] Player Kr3mu loaded, citizenid: ABC12345" },
        { id: 11, timestamp: "12:02:30", level: "info", source: "player", message: "[Join] Lananal (discord:123456789) connected" },
        { id: 12, timestamp: "12:03:45", level: "error", source: "resource", message: "[qb-banking] attempt to index nil value 'account' at banking.lua:89" },
        { id: 13, timestamp: "12:03:45", level: "error", source: "resource", message: "[qb-banking] stack traceback: banking.lua:89 > GetAccount" },
        { id: 14, timestamp: "12:04:01", level: "warn", source: "server", message: "Entity pool usage at 78% (4,680/6,000)" },
        { id: 15, timestamp: "12:05:22", level: "info", source: "player", message: "[Join] xXDarkRiderXx (license:abcdef123456) connected" },
        { id: 16, timestamp: "12:06:10", level: "command", source: "admin", message: "Kr3mu executed: /weather clear" },
        { id: 17, timestamp: "12:06:10", level: "info", source: "server", message: "Weather set to CLEAR" },
        { id: 18, timestamp: "12:07:44", level: "warn", source: "resource", message: "[esx_ambulancejob] player ped not found for source 3" },
        { id: 19, timestamp: "12:08:15", level: "info", source: "player", message: "[Leave] xXDarkRiderXx disconnected (timeout)" },
        { id: 20, timestamp: "12:09:30", level: "debug", source: "resource", message: "[ox_inventory] inventory refresh for Kr3mu, 24 items synced" },
        { id: 21, timestamp: "12:10:00", level: "info", source: "server", message: "Scheduled restart in 50 minutes" },
        { id: 22, timestamp: "12:10:05", level: "info", source: "txAdmin", message: "Heartbeat OK - uptime 10m, avg tick 8.2ms" },
    ];

    let logs = $state<LogEntry[]>([...mockLogs]);
    let searchQuery = $state("");
    let searching = $state(false);
    let commandInput = $state("");
    let autoScroll = $state(true);
    let consoleEl = $state<HTMLElement | null>(null);
    let wrapperEl = $state<HTMLElement | null>(null);
    let commandInputEl = $state<HTMLInputElement | null>(null);
    let searchInputEl = $state<HTMLInputElement | null>(null);

    const filteredLogs = $derived(
        searchQuery
            ? logs.filter(
                  (l) =>
                      l.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
                      l.source.toLowerCase().includes(searchQuery.toLowerCase()),
              )
            : logs,
    );

    const levelColors: Record<LogLevel, string> = {
        info: "text-blue-400",
        warn: "text-primary",
        error: "text-destructive",
        debug: "text-muted-foreground/50",
        command: "text-chart-2",
    };

    const levelTag: Record<LogLevel, { text: string; class: string }> = {
        info: { text: "INF", class: "text-blue-400/60" },
        warn: { text: "WRN", class: "text-primary/60" },
        error: { text: "ERR", class: "text-destructive/60" },
        debug: { text: "DBG", class: "text-muted-foreground/25" },
        command: { text: "CMD", class: "text-chart-2/60" },
    };

    function scrollToBottom(): void {
        if (consoleEl && autoScroll) {
            consoleEl.scrollTop = consoleEl.scrollHeight;
        }
    }

    function handleScroll(): void {
        if (!consoleEl) return;
        const { scrollTop, scrollHeight, clientHeight } = consoleEl;
        autoScroll = scrollHeight - scrollTop - clientHeight < 40;
    }

    function clearConsole(): void {
        logs = [];
    }

    function downloadLogs(): void {
        const content = logs
            .map((l) => `[${l.timestamp}] [${l.level.toUpperCase()}] [${l.source}] ${l.message}`)
            .join("\n");
        const blob = new Blob([content], { type: "text/plain" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `runfive-console-${new Date().toISOString().slice(0, 10)}.log`;
        a.click();
        URL.revokeObjectURL(url);
    }

    function openSearch(): void {
        searching = true;
        requestAnimationFrame(() => searchInputEl?.focus());
    }

    function closeSearch(): void {
        searching = false;
        searchQuery = "";
    }

    function sendCommand(): void {
        if (!commandInput.trim()) return;
        const entry: LogEntry = {
            id: Date.now(),
            timestamp: new Date().toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
            level: "command",
            source: "admin",
            message: `> ${commandInput}`,
        };
        logs = [...logs, entry];
        commandInput = "";
        autoScroll = true;
        requestAnimationFrame(scrollToBottom);
    }

    function handleKeydown(e: KeyboardEvent): void {
        // Ctrl+F / Cmd+F -> open search
        if ((e.ctrlKey || e.metaKey) && e.key === "f") {
            e.preventDefault();
            openSearch();
        }
        // Escape -> close search
        if (e.key === "Escape" && searching) {
            closeSearch();
            commandInputEl?.focus();
        }
    }

    $effect(() => {
        filteredLogs;
        requestAnimationFrame(scrollToBottom);
    });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
    bind:this={wrapperEl}
    onkeydown={handleKeydown}
    class="relative flex h-full flex-col overflow-hidden bg-background"
    tabindex="-1"
>
    <!-- Search overlay bar (slides down when active) -->
    {#if searching}
        <div class="absolute top-0 right-0 left-0 z-10 flex items-center gap-2 border-b border-border bg-card px-3 py-2">
            <Search size={14} class="shrink-0 text-muted-foreground/40" />
            <input
                bind:this={searchInputEl}
                type="text"
                bind:value={searchQuery}
                placeholder="Filter logs..."
                class="h-8 flex-1 rounded-md border border-border bg-background px-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/40 focus:border-primary focus:ring-1 focus:ring-primary"
            />
            <span class="shrink-0 text-xs text-muted-foreground/40">
                {filteredLogs.length} results
            </span>
            <button
                onclick={closeSearch}
                class="shrink-0 rounded-md p-1.5 text-muted-foreground/40 transition-colors hover:bg-muted hover:text-foreground"
            >
                <X size={14} />
            </button>
        </div>
    {/if}

    <!-- Log Output -->
    <div
        bind:this={consoleEl}
        onscroll={handleScroll}
        class="console-scroll flex-1 overflow-y-auto overflow-x-hidden font-mono text-xs leading-[1.7]"
    >
        {#each filteredLogs as entry, i (entry.id)}
            <div class="flex items-baseline border-b border-border/10 px-3 py-0.75 transition-colors hover:bg-muted/20">
                <span class="w-6 shrink-0 select-none text-right text-[11px] text-muted-foreground/25">{i + 1}</span>
                <span class="mx-2 shrink-0 select-none text-[11px] text-muted-foreground/35">{entry.timestamp}</span>
                <span class="mr-2 w-7 shrink-0 select-none text-right text-[11px] font-semibold {levelTag[entry.level].class}">{levelTag[entry.level].text}</span>
                <span class="{levelColors[entry.level]} break-all">{entry.message}</span>
            </div>
        {/each}
        {#if filteredLogs.length === 0}
            <div class="flex h-full items-center justify-center text-sm text-muted-foreground/20">
                {searchQuery ? "No matching entries" : "No output"}
            </div>
        {/if}
    </div>

    <!-- Scroll-to-bottom pill -->
    {#if !autoScroll}
        <button
            onclick={() => { autoScroll = true; scrollToBottom(); }}
            class="absolute bottom-14 left-1/2 z-10 flex -translate-x-1/2 items-center gap-1.5 rounded-full border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground shadow-lg transition-colors hover:text-foreground"
        >
            <ChevronDown size={12} />
            New logs below
        </button>
    {/if}

    <!-- Bottom Bar: Input + Actions -->
    <div class="shrink-0 border-t border-border bg-card">
        <form
            onsubmit={(e) => { e.preventDefault(); sendCommand(); }}
            class="flex items-center gap-2 px-3 py-2"
        >
            <!-- Command input -->
            <div class="relative flex-1">
                <span class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-xs font-bold text-primary/50 select-none">&gt;</span>
                <input
                    bind:this={commandInputEl}
                    type="text"
                    bind:value={commandInput}
                    placeholder="Enter command..."
                    class="h-9 w-full rounded-md border border-border bg-background pl-7 pr-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/30 focus:border-primary focus:ring-1 focus:ring-primary"
                />
            </div>

            <!-- Send -->
            <button
                type="submit"
                disabled={!commandInput.trim()}
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-30"
                title="Send command"
            >
                <ArrowRight size={16} />
            </button>

            <!-- Divider -->
            <div class="h-5 w-px shrink-0 bg-border"></div>

            <!-- Actions -->
            <button
                type="button"
                onclick={openSearch}
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title="Search (Ctrl+F)"
            >
                <Search size={15} />
            </button>
            <button
                type="button"
                onclick={downloadLogs}
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title="Download logs"
            >
                <Download size={15} />
            </button>
            <button
                type="button"
                onclick={clearConsole}
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                title="Clear console"
            >
                <Trash2 size={15} />
            </button>
        </form>
    </div>
</div>
