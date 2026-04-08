import type { Component } from "svelte";
import type { GridLayoutItem } from "$lib/components/dashboard/grid-stack.svelte";

export interface WidgetDefinition {
    id: string;
    label: string;
    icon: string;
    minW: number;
    minH: number;
    defaultW: number;
    defaultH: number;
}

/** All available widget types. Icons reference lucide icon names. */
export const widgetRegistry: WidgetDefinition[] = [
    { id: "console", label: "Console", icon: "terminal", minW: 4, minH: 3, defaultW: 7, defaultH: 6 },
    { id: "players", label: "Players", icon: "users", minW: 3, minH: 3, defaultW: 5, defaultH: 6 },
    // Future widgets:
    // { id: "stats", label: "Server Stats", icon: "activity", minW: 2, minH: 2, defaultW: 4, defaultH: 3 },
    // { id: "bans", label: "Recent Bans", icon: "shield-ban", minW: 3, minH: 2, defaultW: 4, defaultH: 3 },
    // { id: "resources", label: "Resources", icon: "blocks", minW: 2, minH: 2, defaultW: 3, defaultH: 3 },
    // { id: "chat", label: "Chat", icon: "message-square", minW: 3, minH: 3, defaultW: 4, defaultH: 4 },
];

export function getWidgetDef(id: string): WidgetDefinition | undefined {
    return widgetRegistry.find((w) => w.id === id);
}

/** Create a layout item for a widget, auto-positioned */
export function createWidgetItem(id: string): GridLayoutItem | null {
    const def = getWidgetDef(id);
    if (!def) return null;
    return {
        id: def.id,
        x: 0,
        y: 0,
        w: def.defaultW,
        h: def.defaultH,
        minW: def.minW,
        minH: def.minH,
    };
}
