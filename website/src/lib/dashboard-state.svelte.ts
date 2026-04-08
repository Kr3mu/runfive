/** Shared dashboard state — editing mode, layout management. */

import type { GridLayoutItem } from "$lib/components/dashboard/grid-stack.svelte";
import { createWidgetItem } from "$lib/widget-registry";

const DEFAULT_LAYOUT: GridLayoutItem[] = [
    { id: "console", x: 0, y: 0, w: 7, h: 6, minW: 4, minH: 3 },
    { id: "players", x: 7, y: 0, w: 5, h: 6, minW: 3, minH: 3 },
];

let editingDashboard = $state(false);
let layout = $state<GridLayoutItem[]>([...DEFAULT_LAYOUT]);

/** Incremented on add/remove to force GridStack re-init */
let revision = $state(0);

export const dashboardState = {
    get editing(): boolean {
        return editingDashboard;
    },
    set editing(v: boolean) {
        editingDashboard = v;
    },
    toggle(): void {
        editingDashboard = !editingDashboard;
    },

    get layout(): GridLayoutItem[] {
        return layout;
    },
    set layout(v: GridLayoutItem[]) {
        layout = v;
    },

    get activeWidgetIds(): string[] {
        return layout.map((w) => w.id);
    },

    get revision(): number {
        return revision;
    },

    addWidget(id: string): void {
        if (layout.some((w) => w.id === id)) return;
        const item = createWidgetItem(id);
        if (!item) return;
        item.x = 0;
        item.y = 0;
        layout = [...layout, item];
        revision++;
    },

    removeWidget(id: string): void {
        layout = layout.filter((w) => w.id !== id);
        revision++;
    },

    resetLayout(): void {
        layout = [...DEFAULT_LAYOUT];
        revision++;
    },
};
