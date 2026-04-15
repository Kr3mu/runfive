<script lang="ts">
    import { createQuery, useQueryClient } from "@tanstack/svelte-query";
    import { authQueryOptions } from "$lib/api/auth";
    import {
        fetchServerLogs,
        fetchServerStatus,
        serverLogsWebSocketURL,
        serversQueryOptions,
        startServer,
        stopServer,
        type ServerConsoleEvent,
        type ServerLogLine,
        type ServerProcessStatus,
    } from "$lib/api/servers";
    import { canServer } from "$lib/permissions.svelte";
    import { serverState } from "$lib/server-state.svelte";
    import Search from "@lucide/svelte/icons/search";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Download from "@lucide/svelte/icons/download";
    import ArrowRight from "@lucide/svelte/icons/arrow-right";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import X from "@lucide/svelte/icons/x";
    import Play from "@lucide/svelte/icons/play";
    import Square from "@lucide/svelte/icons/square";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";
    import ServerCrash from "@lucide/svelte/icons/server-crash";
    import { toast } from "svelte-sonner";

    type LogLevel = "info" | "warn" | "error" | "debug" | "command";
    type SocketState = "idle" | "connecting" | "open" | "closed";
    type ActionState = "start" | "stop" | null;

    const queryClient = useQueryClient();
    const authQuery = createQuery(() => authQueryOptions());
    const serversQuery = createQuery(() => serversQueryOptions());

    const currentUser = $derived(authQuery.data);
    const servers = $derived(serversQuery.data ?? []);
    const selectedServer = $derived(serverState.resolve(servers));
    const canReadConsole = $derived(
        selectedServer ? canServer(currentUser, selectedServer.id, "console", "read") : false,
    );
    const canExecuteConsole = $derived(
        selectedServer ? canServer(currentUser, selectedServer.id, "console", "execute") : false,
    );

    let runtimeStatus = $state<ServerProcessStatus | null>(null);
    let logs = $state<ServerLogLine[]>([]);
    let loading = $state(false);
    let socketState = $state<SocketState>("idle");
    let actionState = $state<ActionState>(null);
    let searchQuery = $state("");
    let searching = $state(false);
    let commandInput = $state("");
    let autoScroll = $state(true);
    let consoleEl = $state<HTMLElement | null>(null);
    let wrapperEl = $state<HTMLElement | null>(null);
    let commandInputEl = $state<HTMLInputElement | null>(null);
    let searchInputEl = $state<HTMLInputElement | null>(null);
    let socketRef = $state<WebSocket | null>(null);
    let localLogID = 0;

    const activeStatus = $derived.by((): ServerProcessStatus | null => {
        if (runtimeStatus) return runtimeStatus;
        if (!selectedServer) return null;
        return {
            id: selectedServer.id,
            status: selectedServer.status,
            updatedAt: new Date().toISOString(),
        };
    });

    const filteredLogs = $derived(
        searchQuery
            ? logs.filter(
                  (line) =>
                      line.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
                      line.stream.toLowerCase().includes(searchQuery.toLowerCase()),
              )
            : logs,
    );

    const statusLabel = {
        running: "Running",
        starting: "Starting",
        stopped: "Stopped",
        crashed: "Crashed",
    } as const;

    const statusPill = {
        running: "border-emerald-500/30 bg-emerald-500/10 text-emerald-400",
        starting: "border-amber-400/30 bg-amber-400/10 text-amber-300",
        stopped: "border-border bg-muted/40 text-muted-foreground",
        crashed: "border-destructive/30 bg-destructive/10 text-destructive",
    } as const;

    const levelColors: Record<LogLevel, string> = {
        info: "text-blue-400",
        warn: "text-primary",
        error: "text-destructive",
        debug: "text-muted-foreground/60",
        command: "text-chart-2",
    };

    const levelTag: Record<LogLevel, { text: string; class: string }> = {
        info: { text: "INF", class: "text-blue-400/60" },
        warn: { text: "WRN", class: "text-primary/60" },
        error: { text: "ERR", class: "text-destructive/60" },
        debug: { text: "SYS", class: "text-muted-foreground/35" },
        command: { text: "CMD", class: "text-chart-2/60" },
    };

    const canStart = $derived(
        Boolean(selectedServer) &&
            canExecuteConsole &&
            (activeStatus?.status === "stopped" || activeStatus?.status === "crashed") &&
            actionState === null,
    );

    const canStop = $derived(
        Boolean(selectedServer) &&
            canExecuteConsole &&
            (activeStatus?.status === "starting" || activeStatus?.status === "running") &&
            actionState === null,
    );

    const commandDisabled = $derived(
        !selectedServer ||
            !canExecuteConsole ||
            socketState !== "open" ||
            (activeStatus?.status !== "starting" && activeStatus?.status !== "running"),
    );

    function nextLocalLogID(): number {
        localLogID -= 1;
        return localLogID;
    }

    function makeLocalLine(stream: ServerLogLine["stream"], message: string): ServerLogLine {
        return {
            id: nextLocalLogID(),
            timestamp: new Date().toISOString(),
            stream,
            message,
        };
    }

    function pushLine(line: ServerLogLine): void {
        logs = [...logs.slice(-2999), line];
        requestAnimationFrame(scrollToBottom);
    }

    function mergeLines(lines: ServerLogLine[]): void {
        logs = lines.slice(-3000);
        requestAnimationFrame(scrollToBottom);
    }

    function lineLevel(line: ServerLogLine): LogLevel {
        if (line.stream === "stderr") return "error";
        if (line.stream === "stdin") return "command";
        if (line.stream === "system") return "debug";
        return "info";
    }

    function lineTimestamp(line: ServerLogLine): string {
        return new Date(line.timestamp).toLocaleTimeString("en-GB", {
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
        });
    }

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
            .map((line) => `[${lineTimestamp(line)}] [${line.stream.toUpperCase()}] ${line.message}`)
            .join("\n");
        const blob = new Blob([content], { type: "text/plain" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${selectedServer?.id ?? "runfive"}-console-${new Date().toISOString().slice(0, 10)}.log`;
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

    function openSocket(serverId: string, reconnect: () => void): void {
        socketRef = new WebSocket(serverLogsWebSocketURL(serverId));
        socketState = "connecting";

        socketRef.addEventListener("open", () => {
            socketState = "open";
        });

        socketRef.addEventListener("message", (raw: MessageEvent<string>) => {
            const payload = JSON.parse(raw.data) as ServerConsoleEvent;
            if (payload.type === "snapshot") {
                if (payload.status) runtimeStatus = payload.status;
                if (payload.lines) mergeLines(payload.lines);
                loading = false;
                return;
            }

            if (payload.type === "status" && payload.status) {
                runtimeStatus = payload.status;
                void queryClient.invalidateQueries({ queryKey: ["servers"] });
                return;
            }

            if (payload.type === "line" && payload.line) {
                pushLine(payload.line);
                return;
            }

            if (payload.type === "error" && payload.error) {
                pushLine(makeLocalLine("system", payload.error));
            }
        });

        socketRef.addEventListener("error", () => {
            socketState = "closed";
        });

        socketRef.addEventListener("close", () => {
            socketState = "closed";
            socketRef = null;
            reconnect();
        });
    }

    async function handleStart(): Promise<void> {
        if (!selectedServer || !canStart) return;

        actionState = "start";
        try {
            runtimeStatus = await startServer(selectedServer.id);
            await queryClient.invalidateQueries({ queryKey: ["servers"] });
            toast.success(`Starting ${selectedServer.name}`);
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : "Failed to start server";
            toast.error(message);
            pushLine(makeLocalLine("system", message));
        } finally {
            actionState = null;
        }
    }

    async function handleStop(): Promise<void> {
        if (!selectedServer || !canStop) return;

        actionState = "stop";
        try {
            runtimeStatus = await stopServer(selectedServer.id);
            await queryClient.invalidateQueries({ queryKey: ["servers"] });
            toast.success(`Stopping ${selectedServer.name}`);
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : "Failed to stop server";
            toast.error(message);
            pushLine(makeLocalLine("system", message));
        } finally {
            actionState = null;
        }
    }

    function sendCommand(): void {
        const command = commandInput.trim();
        if (!command || !socketRef || socketState !== "open") return;
        socketRef.send(JSON.stringify({ type: "command", command }));
        commandInput = "";
        autoScroll = true;
    }

    function handleKeydown(e: KeyboardEvent): void {
        if ((e.ctrlKey || e.metaKey) && e.key === "f") {
            e.preventDefault();
            openSearch();
        }
        if (e.key === "Escape" && searching) {
            closeSearch();
            commandInputEl?.focus();
        }
    }

    $effect(() => {
        filteredLogs;
        requestAnimationFrame(scrollToBottom);
    });

    $effect(() => {
        const serverId = selectedServer?.id;
        const readAllowed = canReadConsole;

        if (typeof window === "undefined") return;

        if (!serverId || !readAllowed) {
            socketRef?.close(1000, "switch");
            socketRef = null;
            runtimeStatus = null;
            logs = [];
            loading = false;
            socketState = "idle";
            return;
        }

        let disposed = false;
        let reconnectTimer: number | null = null;

        logs = [];
        runtimeStatus = null;
        loading = true;

        const connect = (): void => {
            if (disposed) return;
            if (reconnectTimer !== null) {
                window.clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
            openSocket(serverId, () => {
                if (disposed) return;
                reconnectTimer = window.setTimeout(connect, 1500);
            });
        };

        (async (): Promise<void> => {
            try {
                const [status, initialLogs] = await Promise.all([
                    fetchServerStatus(serverId),
                    fetchServerLogs(serverId, 200),
                ]);
                if (disposed) return;
                runtimeStatus = status;
                mergeLines(initialLogs);
            } catch (error: unknown) {
                if (disposed) return;
                const message = error instanceof Error ? error.message : "Failed to load console";
                pushLine(makeLocalLine("system", message));
            } finally {
                if (disposed) return;
                loading = false;
                connect();
            }
        })();

        return (): void => {
            disposed = true;
            if (reconnectTimer !== null) {
                window.clearTimeout(reconnectTimer);
            }
            socketRef?.close(1000, "switch");
            socketRef = null;
            socketState = "idle";
        };
    });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
    bind:this={wrapperEl}
    onkeydown={handleKeydown}
    class="relative flex h-full flex-col overflow-hidden bg-background"
    tabindex="-1"
>
    {#if !selectedServer}
        <div class="flex h-full items-center justify-center px-6 text-sm text-muted-foreground/50">
            Select a server to open its console.
        </div>
    {:else if !canReadConsole}
        <div class="flex h-full items-center justify-center px-6">
            <div class="max-w-sm rounded-lg border border-border bg-card p-6 text-center">
                <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
                    <ShieldAlert size={20} class="text-destructive" />
                </div>
                <h2 class="font-heading text-base font-semibold text-foreground">Console access required</h2>
                <p class="mt-2 text-sm text-muted-foreground">
                    This account can see the server, but it doesn’t have permission to read the console stream.
                </p>
            </div>
        </div>
    {:else}
        {#if searching}
            <div class="absolute top-0 right-0 left-0 z-20 flex items-center gap-2 border-b border-border bg-card px-3 py-2">
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

        <div class="shrink-0 border-b border-border bg-card/80 px-3 py-2 backdrop-blur-sm">
            <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="min-w-0">
                    <div class="flex items-center gap-2">
                        <span class="truncate font-heading text-sm font-semibold text-foreground">
                            {selectedServer.name}
                        </span>
                        {#if activeStatus}
                            <span class="rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] {statusPill[activeStatus.status]}">
                                {statusLabel[activeStatus.status]}
                            </span>
                        {/if}
                        {#if socketState === "connecting"}
                            <span class="text-[10px] text-muted-foreground/60">Connecting…</span>
                        {:else if socketState === "closed"}
                            <span class="text-[10px] text-muted-foreground/60">Reconnecting…</span>
                        {/if}
                    </div>
                    <div class="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground/60">
                        {#if activeStatus?.pid}
                            <span>PID {activeStatus.pid}</span>
                        {/if}
                        {#if activeStatus?.exitReason}
                            <span class={activeStatus.status === "crashed" ? "text-destructive/80" : ""}>
                                {activeStatus.exitReason}
                            </span>
                        {:else}
                            <span>{selectedServer.artifactVersion}</span>
                        {/if}
                    </div>
                </div>

                {#if canExecuteConsole}
                    <div class="flex items-center gap-2">
                        <button
                            type="button"
                            onclick={handleStart}
                            disabled={!canStart}
                            class="inline-flex h-8 items-center gap-1.5 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 text-xs font-semibold text-emerald-400 transition-colors hover:bg-emerald-500/15 disabled:cursor-not-allowed disabled:opacity-35"
                        >
                            {#if actionState === "start"}
                                <LoaderCircle size={13} class="animate-spin" />
                            {:else}
                                <Play size={13} />
                            {/if}
                            Start
                        </button>
                        <button
                            type="button"
                            onclick={handleStop}
                            disabled={!canStop}
                            class="inline-flex h-8 items-center gap-1.5 rounded-md border border-destructive/30 bg-destructive/10 px-3 text-xs font-semibold text-destructive transition-colors hover:bg-destructive/15 disabled:cursor-not-allowed disabled:opacity-35"
                        >
                            {#if actionState === "stop"}
                                <LoaderCircle size={13} class="animate-spin" />
                            {:else}
                                <Square size={12} />
                            {/if}
                            Stop
                        </button>
                    </div>
                {/if}
            </div>
        </div>

        <div
            bind:this={consoleEl}
            onscroll={handleScroll}
            class="console-scroll flex-1 overflow-y-auto overflow-x-hidden font-mono text-xs leading-[1.7]"
        >
            {#each filteredLogs as line, i (line.id)}
                {@const level = lineLevel(line)}
                <div class="flex items-baseline border-b border-border/10 px-3 py-0.75 transition-colors hover:bg-muted/20">
                    <span class="w-6 shrink-0 select-none text-right text-[11px] text-muted-foreground/25">{i + 1}</span>
                    <span class="mx-2 shrink-0 select-none text-[11px] text-muted-foreground/35">{lineTimestamp(line)}</span>
                    <span class="mr-2 w-7 shrink-0 select-none text-right text-[11px] font-semibold {levelTag[level].class}">{levelTag[level].text}</span>
                    <span class="{levelColors[level]} break-all">{line.message}</span>
                </div>
            {/each}

            {#if filteredLogs.length === 0}
                <div class="flex h-full items-center justify-center px-6 text-center text-sm text-muted-foreground/25">
                    {#if loading}
                        <span class="inline-flex items-center gap-2">
                            <LoaderCircle size={14} class="animate-spin" />
                            Loading console…
                        </span>
                    {:else if activeStatus?.status === "crashed"}
                        <span class="inline-flex items-center gap-2">
                            <ServerCrash size={14} />
                            Server crashed. Start it again to continue streaming logs.
                        </span>
                    {:else if searchQuery}
                        <span>No matching entries</span>
                    {:else}
                        <span>No output yet</span>
                    {/if}
                </div>
            {/if}
        </div>

        {#if !autoScroll}
            <button
                onclick={() => {
                    autoScroll = true;
                    scrollToBottom();
                }}
                class="absolute bottom-14 left-1/2 z-10 flex -translate-x-1/2 items-center gap-1.5 rounded-full border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground shadow-lg transition-colors hover:text-foreground"
            >
                <ChevronDown size={12} />
                New logs below
            </button>
        {/if}

        <div class="shrink-0 border-t border-border bg-card">
            <form
                onsubmit={(e) => {
                    e.preventDefault();
                    sendCommand();
                }}
                class="flex items-center gap-2 px-3 py-2"
            >
                <div class="relative flex-1">
                    <span class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 select-none text-xs font-bold text-primary/50">&gt;</span>
                    <input
                        bind:this={commandInputEl}
                        type="text"
                        bind:value={commandInput}
                        placeholder={canExecuteConsole ? "Enter command..." : "Console input requires execute permission"}
                        disabled={commandDisabled}
                        class="h-9 w-full rounded-md border border-border bg-background pl-7 pr-3 text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/30 focus:border-primary focus:ring-1 focus:ring-primary disabled:cursor-not-allowed disabled:opacity-50"
                    />
                </div>

                <button
                    type="submit"
                    disabled={!commandInput.trim() || commandDisabled}
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-30"
                    title="Send command"
                >
                    <ArrowRight size={16} />
                </button>

                <div class="h-5 w-px shrink-0 bg-border"></div>

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
    {/if}
</div>
