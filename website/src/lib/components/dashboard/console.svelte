<script lang="ts">
    import { Terminal, type ITheme } from "@xterm/xterm";
    import { FitAddon } from "@xterm/addon-fit";
    import { SearchAddon } from "@xterm/addon-search";
    import { WebLinksAddon } from "@xterm/addon-web-links";
    import "@xterm/xterm/css/xterm.css";

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
    import { toast } from "svelte-sonner";

    import Search from "@lucide/svelte/icons/search";
    import Trash2 from "@lucide/svelte/icons/trash-2";
    import Download from "@lucide/svelte/icons/download";
    import ArrowRight from "@lucide/svelte/icons/arrow-right";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import ChevronUp from "@lucide/svelte/icons/chevron-up";
    import X from "@lucide/svelte/icons/x";
    import Play from "@lucide/svelte/icons/play";
    import Square from "@lucide/svelte/icons/square";
    import LoaderCircle from "@lucide/svelte/icons/loader-circle";
    import ShieldAlert from "@lucide/svelte/icons/shield-alert";
    import ServerCrash from "@lucide/svelte/icons/server-crash";

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
    let loading = $state(false);
    let socketState = $state<SocketState>("idle");
    let actionState = $state<ActionState>(null);
    let searching = $state(false);
    let searchQuery = $state("");
    let searchHits = $state(0);
    let commandInput = $state("");
    let autoScroll = $state(true);
    /**
     * Raw log-line buffer retained alongside the xterm instance so the
     * download action can export a clean text file and clear() can repopulate
     * from state instead of from the terminal's internal buffer. Capped to
     * match the terminal scrollback so memory stays bounded.
     */
    let rawLines = $state<ServerLogLine[]>([]);
    /**
     * Watermark for the clear-console action. After clearing, this holds the
     * highest backend log id we'd already shown — every subsequent push or
     * snapshot drops lines with id ≤ floor before they reach xterm. Without
     * this, a websocket reconnect or any other replay would hand the backend
     * tail buffer's last 200 lines straight back into a "cleared" terminal.
     * Local system lines use negative ids (nextLocalLogID) and bypass the
     * gate so error toasts / connection notes survive a clear. Reset to zero
     * on every server switch so a fresh server starts unfiltered.
     */
    let floorLogId = 0;

    /** Bound to the host <div> that xterm mounts into. */
    let termHostEl = $state<HTMLDivElement | null>(null);
    /** Bound to the command <input> so the keyboard shortcut can focus it. */
    let commandInputEl = $state<HTMLInputElement | null>(null);
    let searchInputEl = $state<HTMLInputElement | null>(null);

    /** Live xterm instance. Recreated when the mount point changes, never per-server. */
    let term: Terminal | null = null;
    let fitAddon: FitAddon | null = null;
    let searchAddon: SearchAddon | null = null;
    /**
     * Bumps every time a new Terminal is mounted. Pending write-callback
     * chains capture the generation they were started under and bail if it
     * no longer matches — without this, a chain that fired after dispose()
     * would hit a torn-down buffer/renderer and crash the whole widget.
     */
    let termGen = 0;
    /**
     * Bumps every time replaceBuffer is called, so a new bulk replay
     * supersedes any chain still walking through the previous batch.
     */
    let replayGen = 0;

    function termAlive(capturedGen: number): boolean {
        return term !== null && termGen === capturedGen;
    }

    function replayValid(capturedTermGen: number, capturedReplayGen: number): boolean {
        return termAlive(capturedTermGen) && replayGen === capturedReplayGen;
    }

    /**
     * scrollToBottom can hit the renderer before it has measured the
     * terminal for the first time (e.g. during initial mount or a tab
     * that was hidden when the component first rendered), which throws
     * on the internal `dimensions` getter. The autoscroll state survives
     * the throw so the next write naturally retries.
     */
    function safeScrollToBottom(t: Terminal): void {
        try {
            t.scrollToBottom();
        } catch {
            // Renderer not ready yet; skip this tick.
        }
    }

    let socketRef: WebSocket | null = null;
    let localLogID = 0;

    /**
     * Hard cap on the client-side scrollback / raw buffer. Matches the xterm
     * scrollback so users see exactly as much history as is selectable.
     */
    const SCROLLBACK_LINES = 2000;

    const activeStatus = $derived.by((): ServerProcessStatus | null => {
        if (runtimeStatus) return runtimeStatus;
        if (!selectedServer) return null;
        return {
            id: selectedServer.id,
            status: selectedServer.status,
            updatedAt: new Date().toISOString(),
        };
    });

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

    /**
     * xterm theme pulled from the panel's palette. Kept in sync with the
     * Tailwind tokens in app.css so dark / light mode flips come along for
     * free via the parent theme switcher.
     */
    function buildTheme(): ITheme {
        return {
            background: "#00000000",
            foreground: "rgb(226, 232, 240)",
            // Transparent cursor — the terminal is strictly a viewer here
            // (disableStdin is true, input goes through the form below),
            // so the blinking caret xterm draws on focus would just be
            // visual noise. CSS below belt-and-suspenders hides any
            // cursor-layer DOM node the renderer adds.
            cursor: "#00000000",
            cursorAccent: "#00000000",
            selectionBackground: "rgba(253, 224, 71, 0.35)",
            selectionForeground: undefined,
            selectionInactiveBackground: "rgba(148, 163, 184, 0.25)",

            black: "#0f172a",
            red: "#f87171",
            green: "#4ade80",
            yellow: "#fbbf24",
            blue: "#60a5fa",
            magenta: "#c084fc",
            cyan: "#22d3ee",
            white: "rgb(226, 232, 240)",

            brightBlack: "#475569",
            brightRed: "#fca5a5",
            brightGreen: "#86efac",
            brightYellow: "#fde68a",
            brightBlue: "#93c5fd",
            brightMagenta: "#d8b4fe",
            brightCyan: "#67e8f9",
            brightWhite: "#f8fafc",
        };
    }

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

    function formatTimestamp(isoTimestamp: string): string {
        return new Date(isoTimestamp).toLocaleTimeString("en-GB", {
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
        });
    }

    /**
     * Writes a log line to xterm and attaches a timestamp decoration to its
     * starting row in the left gutter.
     *
     * xterm's write pipeline is asynchronous: multiple write() calls in a
     * single synchronous block all queue against the same "current" cursor
     * position, so registering a marker before a write ends up anchoring
     * every marker to the same row. The fix is to snapshot the starting row
     * inside the write callback, once the parser has advanced the cursor,
     * and then `registerMarker(-rowsUsed)` to walk back to the first visual
     * row of the just-written message (which matters when a line wrapped
     * across the viewport width).
     *
     * The decoration element is mounted inside xterm's own decoration
     * layer, so it scrolls with the content. `user-select: none` + the
     * fact that xterm's selection model reads from the cell grid (not the
     * DOM) means drag-to-copy never captures the timestamp — which is the
     * whole reason we can show it visibly without breaking the copy flow.
     */
    /**
     * Attaches a timestamp decoration at the first visual row of a just-
     * written message. Called from a write callback so the buffer reflects
     * the post-write cursor position; walks back by the actual number of
     * rows the write consumed so wrapped lines still anchor at the top.
     */
    function attachDecoration(
        t: Terminal,
        timestamp: string,
        startY: number,
    ): void {
        const endY = t.buffer.active.baseY + t.buffer.active.cursorY;
        const rowsUsed = Math.max(1, endY - startY);
        const marker = t.registerMarker(-rowsUsed);
        if (!marker) return;

        const decoration = t.registerDecoration({
            marker,
            anchor: "left",
            x: 0,
            width: 1,
            height: 1,
            layer: "top",
        });
        if (!decoration) return;

        decoration.onRender((el: HTMLElement): void => {
            el.classList.add("console-timestamp-gutter");
            el.textContent = formatTimestamp(timestamp);
        });

        // Drop the decoration once the marker scrolls out of the scrollback
        // ring so xterm's DOM node pool stays bounded.
        marker.onDispose((): void => decoration.dispose());
    }

    function writeLine(line: ServerLogLine): void {
        if (!term) return;
        const t = term;
        const gen = termGen;

        // Empty-write barrier captures the starting Y after any prior
        // queued writes have been parsed, so markers don't all pile up on
        // the same row during a burst.
        t.write("", (): void => {
            if (!termAlive(gen)) return;
            const startY = t.buffer.active.baseY + t.buffer.active.cursorY;
            t.write(line.message + "\r\n", (): void => {
                if (!termAlive(gen)) return;
                attachDecoration(t, line.timestamp, startY);
                if (autoScroll) safeScrollToBottom(t);
            });
        });
    }

    /**
     * Gate for both single-line pushes and bulk replays. Backend ids
     * monotonically increase, so anything ≤ floor was already on screen
     * before the operator cleared the console. Local system lines use
     * negative ids and bypass the gate so error / connection notes still
     * render after a clear.
     */
    function passesFloor(line: ServerLogLine): boolean {
        return line.id <= 0 || line.id > floorLogId;
    }

    function replaceBuffer(lines: ServerLogLine[]): void {
        if (!term) return;
        const t = term;
        const gen = termGen;
        const replay = ++replayGen;

        t.clear();
        rawLines = [];
        // Filter against the clear-watermark BEFORE trimming so a snapshot
        // that ends entirely below the floor produces an empty buffer
        // instead of silently re-flooding from the SCROLLBACK_LINES tail.
        const trimmed = lines.filter(passesFloor).slice(-SCROLLBACK_LINES);
        for (const line of trimmed) {
            rawLines.push(line);
        }

        if (trimmed.length === 0) {
            autoScroll = true;
            return;
        }

        // One bulk write instead of 200 serial callback chains: chaining
        // through xterm's async write scheduler per-line would queue
        // hundreds of callbacks and stall the main thread for most of a
        // page load. A single write flushes in one parse pass; we then
        // walk the buffer backwards to find each message's starting row
        // and attach one decoration per log line in a synchronous loop.
        //
        // isWrapped on IBufferLine tells us which rows are continuations
        // of a wrapped message vs. fresh message starts, so wrapped lines
        // still anchor their timestamp at the first visual row without
        // per-line tracking during the write.
        const bulk = trimmed.map((l: ServerLogLine): string => l.message).join("\r\n") + "\r\n";

        t.write(bulk, (): void => {
            if (!replayValid(gen, replay)) return;

            const buffer = t.buffer.active;
            const cursorAbs = buffer.baseY + buffer.cursorY;
            const startsAbs: number[] = [];
            const scanStart = Math.max(0, cursorAbs - SCROLLBACK_LINES * 4);
            for (let y = scanStart; y < cursorAbs; y += 1) {
                const bl = buffer.getLine(y);
                if (!bl) continue;
                if (bl.isWrapped) continue;
                startsAbs.push(y);
            }
            // Collapse to exactly one start per message. The buffer may
            // have one extra "fresh" row at the cursor position — drop
            // it if the scan picked up more starts than messages.
            while (startsAbs.length > trimmed.length) startsAbs.shift();

            for (let i = 0; i < startsAbs.length; i += 1) {
                const line = trimmed[i];
                const absTarget = startsAbs[i];
                const offset = absTarget - cursorAbs;
                const marker = t.registerMarker(offset);
                if (!marker) continue;
                const decoration = t.registerDecoration({
                    marker,
                    anchor: "left",
                    x: 0,
                    width: 1,
                    height: 1,
                    layer: "top",
                });
                if (!decoration) continue;
                const ts = line.timestamp;
                decoration.onRender((el: HTMLElement): void => {
                    el.classList.add("console-timestamp-gutter");
                    el.textContent = formatTimestamp(ts);
                });
                marker.onDispose((): void => decoration.dispose());
            }

            safeScrollToBottom(t);
            autoScroll = true;
        });
    }

    function pushLine(line: ServerLogLine): void {
        if (!passesFloor(line)) return;
        rawLines = [...rawLines.slice(-(SCROLLBACK_LINES - 1)), line];
        writeLine(line);
    }

    function clearConsole(): void {
        if (!term) return;
        // Lift the floor to the highest backend id we've already rendered.
        // A reconnect snapshot will hand us those same lines back (the
        // backend tail buffer isn't aware of this client's clear), and
        // without the floor they'd all reappear — defeating the trash-can.
        // Local system lines have negative ids and don't contribute.
        let maxId = floorLogId;
        for (const line of rawLines) {
            if (line.id > maxId) maxId = line.id;
        }
        floorLogId = maxId;
        term.clear();
        rawLines = [];
    }

    /** Strips ANSI SGR sequences so the downloaded .log is plain text. */
    function stripAnsi(s: string): string {
        // Match the minimal set of SGR / CSI sequences fxserver and the
        // panel emit — colour, formatting, cursor moves aren't used in logs.
        return s.replace(/\x1b\[[0-9;]*[A-Za-z]/g, "");
    }

    function downloadLogs(): void {
        const content = rawLines
            .map(
                (line: ServerLogLine): string =>
                    `[${formatTimestamp(line.timestamp)}] ${stripAnsi(line.message)}`,
            )
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
        requestAnimationFrame((): void => searchInputEl?.focus());
    }

    function closeSearch(): void {
        searching = false;
        searchQuery = "";
        searchHits = 0;
        searchAddon?.clearDecorations();
    }

    function runSearch(next: boolean): void {
        if (!searchAddon) return;
        const q = searchQuery.trim();
        if (!q) {
            searchAddon.clearDecorations();
            searchHits = 0;
            return;
        }
        const opts = {
            caseSensitive: false,
            wholeWord: false,
            regex: false,
            decorations: {
                matchBackground: "rgba(253, 224, 71, 0.35)",
                matchOverviewRuler: "rgba(253, 224, 71, 0.55)",
                activeMatchBackground: "rgba(253, 224, 71, 0.6)",
                activeMatchColorOverviewRuler: "rgba(253, 224, 71, 0.9)",
            },
        };
        if (next) {
            searchAddon.findNext(q, opts);
        } else {
            searchAddon.findPrevious(q, opts);
        }
    }

    $effect((): void => {
        // When the query changes, jump to the first match. Reactive so Svelte
        // fires this on every keystroke without a manual handler on the input.
        searchQuery;
        if (!searching) return;
        runSearch(true);
    });

    function openSocket(serverId: string, reconnect: () => void): void {
        socketRef = new WebSocket(serverLogsWebSocketURL(serverId));
        socketState = "connecting";

        socketRef.addEventListener("open", (): void => {
            socketState = "open";
        });

        socketRef.addEventListener("message", (raw: MessageEvent<string>): void => {
            const payload = JSON.parse(raw.data) as ServerConsoleEvent;
            if (payload.type === "snapshot") {
                if (payload.status) runtimeStatus = payload.status;
                if (payload.lines) replaceBuffer(payload.lines);
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

        socketRef.addEventListener("error", (): void => {
            socketState = "closed";
        });

        socketRef.addEventListener("close", (): void => {
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
        } catch (err: unknown) {
            const message: string = err instanceof Error ? err.message : "Failed to start server";
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
        } catch (err: unknown) {
            const message: string = err instanceof Error ? err.message : "Failed to stop server";
            toast.error(message);
            pushLine(makeLocalLine("system", message));
        } finally {
            actionState = null;
        }
    }

    function sendCommand(): void {
        const command: string = commandInput.trim();
        if (!command || !socketRef || socketState !== "open") return;
        socketRef.send(JSON.stringify({ type: "command", command }));
        commandInput = "";
        autoScroll = true;
        if (term) safeScrollToBottom(term);
    }

    function handleKeydown(e: KeyboardEvent): void {
        if ((e.ctrlKey || e.metaKey) && e.key === "f") {
            e.preventDefault();
            openSearch();
            return;
        }
        if (!searching) return;
        if (e.key === "Escape") {
            closeSearch();
            commandInputEl?.focus();
        } else if (e.key === "Enter") {
            e.preventDefault();
            runSearch(!e.shiftKey);
        }
    }

    // Terminal lifecycle: recreate when the mount node appears. Disposed on
    // unmount so a page-level transition doesn't leak an xterm instance.
    $effect((): (() => void) | void => {
        if (!termHostEl) return;

        const t = new Terminal({
            scrollback: SCROLLBACK_LINES,
            fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, "JetBrains Mono", "Fira Code", "Liberation Mono", monospace',
            fontSize: 12,
            lineHeight: 1.35,
            letterSpacing: 0,
            cursorBlink: false,
            cursorStyle: "block",
            disableStdin: true,
            convertEol: true,
            allowProposedApi: true,
            allowTransparency: true,
            theme: buildTheme(),
            rightClickSelectsWord: true,
            scrollOnUserInput: false,
        });

        const fit = new FitAddon();
        const search = new SearchAddon();
        const links = new WebLinksAddon();
        t.loadAddon(fit);
        t.loadAddon(search);
        t.loadAddon(links);
        t.open(termHostEl);
        // Run the initial fit before any writes so the renderer's
        // dimensions are populated — otherwise the first scrollToBottom
        // hits an undefined `dimensions` and throws.
        try {
            fit.fit();
        } catch {
            // Host not yet laid out; the ResizeObserver below will retry.
        }

        // Sync autoscroll to the user's viewport: if they scroll up we stop
        // pinning to the bottom; the "jump to latest" pill re-arms it.
        t.onScroll((): void => {
            const viewportY = t.buffer.active.viewportY;
            const base = t.buffer.active.baseY;
            autoScroll = viewportY >= base;
        });

        const ro = new ResizeObserver((): void => {
            try {
                fit.fit();
            } catch {
                // Terminal not visible yet (tab hidden, display:none, etc.).
                // Fit throws instead of returning — swallow and retry on the
                // next observer tick.
            }
        });
        ro.observe(termHostEl);

        term = t;
        fitAddon = fit;
        searchAddon = search;
        termGen += 1;

        // NB: no replay of rawLines here. Reading rawLines inside this
        // effect creates a reactive dependency, and replaceBuffer mutates
        // rawLines by design — every push would re-trigger this effect,
        // tear the terminal down mid-write and recreate it, which is
        // exactly the hang we used to see on initial load. HMR / tab
        // switches just show an empty terminal until the next snapshot.

        return (): void => {
            // Bump the generation BEFORE disposing so any in-flight write
            // callback that fires during/after teardown sees the mismatch
            // and aborts instead of touching xterm internals.
            termGen += 1;
            term = null;
            fitAddon = null;
            searchAddon = null;
            ro.disconnect();
            try {
                t.dispose();
            } catch {
                // Already torn down mid-callback chain — safe to ignore.
            }
        };
    });

    // Per-server lifecycle: wipe buffer, fetch tail, open websocket, close on
    // server switch / permission change.
    $effect((): (() => void) | void => {
        const serverId: string | undefined = selectedServer?.id;
        const readAllowed: boolean = canReadConsole;

        if (typeof window === "undefined") return;

        if (!serverId || !readAllowed) {
            socketRef?.close(1000, "switch");
            socketRef = null;
            runtimeStatus = null;
            floorLogId = 0;
            replaceBuffer([]);
            loading = false;
            socketState = "idle";
            return;
        }

        let disposed = false;
        let reconnectTimer: number | null = null;

        floorLogId = 0;
        replaceBuffer([]);
        runtimeStatus = null;
        loading = true;

        const connect = (): void => {
            if (disposed) return;
            if (reconnectTimer !== null) {
                window.clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
            openSocket(serverId, (): void => {
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
                replaceBuffer(initialLogs);
            } catch (err: unknown) {
                if (disposed) return;
                const message: string = err instanceof Error ? err.message : "Failed to load console";
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
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
    onkeydown={handleKeydown}
    class="relative flex h-full flex-col overflow-hidden bg-background"
    tabindex="-1"
    role="region"
    aria-label="Server console"
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
        <!-- Header -->
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

        <!-- Search bar (floats above the terminal when open) -->
        {#if searching}
            <div class="absolute top-[58px] right-2 left-2 z-20 flex items-center gap-2 rounded-md border border-border bg-card/95 px-2 py-1.5 shadow-lg backdrop-blur-sm">
                <Search size={13} class="shrink-0 text-muted-foreground/50" />
                <input
                    bind:this={searchInputEl}
                    type="text"
                    bind:value={searchQuery}
                    placeholder="Find in console…"
                    class="h-7 flex-1 rounded-sm bg-transparent text-[12.5px] text-foreground outline-none placeholder:text-muted-foreground/40"
                />
                <div class="flex items-center gap-0.5">
                    <button
                        type="button"
                        onclick={() => runSearch(false)}
                        class="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground/60 hover:bg-muted hover:text-foreground"
                        aria-label="Previous match"
                        title="Previous match (Shift+Enter)"
                    >
                        <ChevronUp size={13} />
                    </button>
                    <button
                        type="button"
                        onclick={() => runSearch(true)}
                        class="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground/60 hover:bg-muted hover:text-foreground"
                        aria-label="Next match"
                        title="Next match (Enter)"
                    >
                        <ChevronDown size={13} />
                    </button>
                </div>
                <button
                    onclick={closeSearch}
                    class="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground/40 hover:bg-muted hover:text-foreground"
                    aria-label="Close search"
                >
                    <X size={13} />
                </button>
            </div>
        {/if}

        <!-- Terminal body -->
        <div class="relative flex-1 overflow-hidden">
            <div bind:this={termHostEl} class="console-host absolute inset-0"></div>

            {#if loading}
                <div class="absolute inset-0 flex items-center justify-center bg-background/60 backdrop-blur-sm">
                    <span class="inline-flex items-center gap-2 text-sm text-muted-foreground/60">
                        <LoaderCircle size={14} class="animate-spin" />
                        Loading console…
                    </span>
                </div>
            {:else if activeStatus?.status === "crashed" && rawLines.length === 0}
                <div class="pointer-events-none absolute inset-0 flex items-center justify-center">
                    <span class="inline-flex items-center gap-2 text-sm text-muted-foreground/40">
                        <ServerCrash size={14} />
                        Server crashed. Start it again to continue streaming logs.
                    </span>
                </div>
            {:else if rawLines.length === 0}
                <div class="pointer-events-none absolute inset-0 flex items-center justify-center">
                    <span class="text-sm text-muted-foreground/30">No output yet</span>
                </div>
            {/if}

            {#if !autoScroll}
                <button
                    onclick={() => {
                        autoScroll = true;
                        if (term) safeScrollToBottom(term);
                    }}
                    class="absolute bottom-3 left-1/2 z-10 flex -translate-x-1/2 items-center gap-1.5 rounded-full border border-border bg-card/95 px-3 py-1.5 text-xs text-muted-foreground shadow-lg backdrop-blur-sm transition-colors hover:text-foreground"
                >
                    <ChevronDown size={12} />
                    Jump to latest
                </button>
            {/if}
        </div>

        <!-- Footer: command input + action toolbar -->
        <div class="shrink-0 border-t border-border bg-card">
            <form
                onsubmit={(e: SubmitEvent): void => {
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
                        placeholder={canExecuteConsole ? "Enter command…" : "Console input requires execute permission"}
                        disabled={commandDisabled}
                        autocomplete="off"
                        spellcheck="false"
                        class="h-9 w-full rounded-md border border-border bg-background pr-3 pl-7 font-mono text-sm text-foreground outline-none transition-all placeholder:text-muted-foreground/30 focus:border-primary focus:ring-1 focus:ring-primary disabled:cursor-not-allowed disabled:opacity-50"
                    />
                </div>

                <button
                    type="submit"
                    disabled={!commandInput.trim() || commandDisabled}
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-30"
                    title="Send command (Enter)"
                >
                    <ArrowRight size={16} />
                </button>

                <div class="h-5 w-px shrink-0 bg-border"></div>

                <button
                    type="button"
                    onclick={openSearch}
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                    title="Find in console (Ctrl+F)"
                >
                    <Search size={15} />
                </button>
                <button
                    type="button"
                    onclick={downloadLogs}
                    disabled={rawLines.length === 0}
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:cursor-not-allowed disabled:opacity-35"
                    title="Download current log buffer"
                >
                    <Download size={15} />
                </button>
                <button
                    type="button"
                    onclick={clearConsole}
                    disabled={rawLines.length === 0}
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:cursor-not-allowed disabled:opacity-35"
                    title="Clear console"
                >
                    <Trash2 size={15} />
                </button>
            </form>
        </div>
    {/if}
</div>

<style>
    /*
     * Make room on the left side of the terminal for the timestamp gutter.
     * xterm renders its canvas into an absolutely positioned child, so we
     * apply the padding on the host and let the canvas inherit the indent.
     */
    .console-host :global(.xterm) {
        padding: 4px 4px 4px 72px;
    }
    .console-host :global(.xterm-viewport) {
        background-color: transparent !important;
        /* Neutralise xterm's default scrollbar styling so it matches the
           rest of the panel (subtle, on-hover emphasis). */
        scrollbar-width: thin;
        scrollbar-color: rgba(148, 163, 184, 0.25) transparent;
    }
    .console-host :global(.xterm-viewport::-webkit-scrollbar) {
        width: 8px;
    }
    .console-host :global(.xterm-viewport::-webkit-scrollbar-thumb) {
        background-color: rgba(148, 163, 184, 0.25);
        border-radius: 4px;
    }
    .console-host :global(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
        background-color: rgba(148, 163, 184, 0.45);
    }

    /*
     * Hide the xterm cursor in every renderer variant (DOM + canvas).
     * The transparent theme colour above already zeroes it out on the
     * canvas renderer, but the DOM renderer paints a .xterm-cursor node
     * whose presence alone reserves layout space and reads as a visible
     * caret on focus. Nothing gets typed here — stdin is disabled at the
     * Terminal options level — so the cursor serves no purpose.
     */
    .console-host :global(.xterm-cursor),
    .console-host :global(.xterm-cursor-layer) {
        display: none !important;
    }
    /*
     * Timestamp gutter decoration.
     *
     * xterm mounts decorations inside .xterm-decoration-container which sits
     * over the canvas area at the cell-grid's origin. We translate the element
     * into the padding-left strip via a negative X offset and lock the width
     * so every row's timestamp is aligned regardless of message length.
     *
     * user-select:none is the important part: the gutter element is part of
     * the DOM but xterm's selection model draws from the cell grid only, so
     * text copies never include these glyphs. That's the key reason this
     * layout solves the copy/selection pain of the previous DOM renderer.
     */
    :global(.console-timestamp-gutter) {
        position: absolute;
        left: -68px;
        top: 0;
        width: 64px;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        padding-right: 6px;
        font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, "JetBrains Mono",
            "Fira Code", "Liberation Mono", monospace;
        font-size: 10.5px;
        color: rgba(148, 163, 184, 0.45);
        pointer-events: none;
        user-select: none;
        -webkit-user-select: none;
        white-space: nowrap;
    }
</style>
