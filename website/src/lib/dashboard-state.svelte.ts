/** Shared dashboard state — editing mode, layout management.
 *
 * The widget layout is persisted per-user via the generic preference store
 * (see $lib/preferences). This module exposes a domain API on top of it:
 * edit toggle, revision counter (to force GridStack re-init on add/remove),
 * and named mutations.
 */

import type { GridLayoutItem } from "$lib/types/grid-layout";
import { createWidgetItem } from "$lib/widget-registry";
import { createPreferenceStore } from "$lib/preferences/store.svelte";
import { dashboardLayoutPref } from "$lib/preferences/registry";

const layoutStore = createPreferenceStore(dashboardLayoutPref);

let editingDashboard = $state(false);

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
        return layoutStore.value;
    },
    set layout(v: GridLayoutItem[]) {
        layoutStore.set(v);
    },

    get activeWidgetIds(): string[] {
        return layoutStore.value.map((w) => w.id);
    },

    get revision(): number {
        return revision;
    },

    get isLoading(): boolean {
        return layoutStore.isLoading;
    },

    /** Load the layout for the given user from the backend. */
    async hydrate(userId: number): Promise<void> {
        await layoutStore.hydrate(userId);
        // Force GridStack to re-init with the freshly loaded layout — its
        // internal DOM was built from whatever state existed at mount time.
        revision++;
    },

    addWidget(id: string): void {
        const current = layoutStore.value;
        if (current.some((w) => w.id === id)) return;
        const item = createWidgetItem(id);
        if (!item) return;
        item.x = 0;
        item.y = 0;
        layoutStore.set([...current, item]);
        revision++;
    },

    removeWidget(id: string): void {
        layoutStore.set(layoutStore.value.filter((w) => w.id !== id));
        revision++;
    },

    resetLayout(): void {
        void layoutStore.reset();
        revision++;
    },
};
