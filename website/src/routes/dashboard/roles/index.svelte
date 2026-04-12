<script lang="ts">
    import { createQuery } from '@tanstack/svelte-query';
    import { authQueryOptions } from '$lib/api/auth';
    import {
        fetchRoles,
        fetchPermissionSchema,
        createRole,
        updateRole,
        deleteRole,
        type RoleListItem,
        type PermissionSchema,
        type ResourceActionSchema,
    } from '$lib/api/roles';
    import { canGlobal } from '$lib/permissions.svelte';
    import { toast } from 'svelte-sonner';
    import Shield from '@lucide/svelte/icons/shield';
    import ShieldAlert from '@lucide/svelte/icons/shield-alert';
    import Plus from '@lucide/svelte/icons/plus';
    import Pencil from '@lucide/svelte/icons/pencil';
    import Trash2 from '@lucide/svelte/icons/trash-2';
    import Users from '@lucide/svelte/icons/users';
    import Check from '@lucide/svelte/icons/check';
    import X from '@lucide/svelte/icons/x';
    import ChevronDown from '@lucide/svelte/icons/chevron-down';
    import ChevronRight from '@lucide/svelte/icons/chevron-right';
    import LoaderCircle from '@lucide/svelte/icons/loader-circle';

    const authQuery = createQuery(() => authQueryOptions());
    const currentUser = $derived(authQuery.data);
    const canRead = $derived(canGlobal(currentUser, 'roles', 'read'));
    const canCreateRole = $derived(canGlobal(currentUser, 'roles', 'create'));
    const canUpdate = $derived(canGlobal(currentUser, 'roles', 'update'));
    const canDeleteRole = $derived(canGlobal(currentUser, 'roles', 'delete'));

    let roles = $state<RoleListItem[]>([]);
    let schema = $state<PermissionSchema | null>(null);
    let isLoading = $state(true);

    let expandedRoleId = $state<number | null>(null);
    let expandedSubRows = $state<Set<string>>(new Set());
    let editingRoleId = $state<number | null>(null);
    let deleteConfirm = $state<number | null>(null);
    let actionInProgress = $state<number | null>(null);

    let showCreateForm = $state(false);
    let newRoleName = $state('');
    let newRoleDescription = $state('');
    let newRoleColor = $state('#6b7280');
    let newGlobalPerms = $state<Record<string, Record<string, boolean>>>({});
    let newServerPerms = $state<Record<string, Record<string, boolean>>>({});
    let isCreating = $state(false);

    let editName = $state('');
    let editDescription = $state('');
    let editColor = $state('');
    let editGlobalPerms = $state<Record<string, Record<string, boolean>>>({});
    let editServerPerms = $state<Record<string, Record<string, boolean>>>({});
    let origGlobalPerms = $state<Record<string, Record<string, boolean>>>({});
    let origServerPerms = $state<Record<string, Record<string, boolean>>>({});
    let isSaving = $state(false);

    function isCellDirty(
        current: Record<string, Record<string, boolean>>,
        original: Record<string, Record<string, boolean>>,
        resource: string,
        action: string,
    ): boolean {
        const cur: boolean = current[resource]?.[action] === true;
        const orig: boolean = original[resource]?.[action] === true;
        return cur !== orig;
    }

    const ROLE_COLORS: string[] = [
        '#ef4444', '#f59e0b', '#22c55e', '#3b82f6', '#8b5cf6',
        '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#6b7280',
    ];

    const RESOURCE_LABELS: Record<string, string> = {
        users: 'Users', roles: 'Roles', servers: 'Servers', settings: 'Settings',
        dashboard: 'Dashboard', players: 'Players', console: 'Console', bans: 'Bans',
    };

    const ACTION_LABELS: Record<string, string> = {
        create: 'Create', read: 'Read', update: 'Update', delete: 'Delete',
        kick: 'Kick', warn: 'Warn', execute: 'Execute',
    };

    $effect((): void => {
        if (!canRead) return;
        Promise.all([fetchRoles(), fetchPermissionSchema()])
            .then(([r, s]): void => { roles = r; schema = s; isLoading = false; })
            .catch((): void => { isLoading = false; });
    });

    function toggleExpand(id: number): void {
        if (expandedRoleId === id) { expandedRoleId = null; editingRoleId = null; }
        else { expandedRoleId = id; editingRoleId = null; }
    }

    function allActionsForResource(rs: ResourceActionSchema): string[] {
        return [...rs.crud, ...(rs.sub ?? [])];
    }

    function hydratePerms(
        saved: Record<string, Record<string, boolean>>,
        schemaSection: Record<string, ResourceActionSchema>,
    ): Record<string, Record<string, boolean>> {
        const result: Record<string, Record<string, boolean>> = {};
        for (const [resource, rs] of Object.entries(schemaSection)) {
            result[resource] = {};
            for (const action of allActionsForResource(rs)) {
                result[resource][action] = saved[resource]?.[action] === true;
            }
        }
        return result;
    }

    function startEdit(role: RoleListItem): void {
        if (!schema) return;
        expandedRoleId = role.id;
        editingRoleId = role.id;
        editName = role.name;
        editDescription = role.description;
        editColor = role.color;
        editGlobalPerms = hydratePerms(role.globalPerms, schema.global);
        editServerPerms = hydratePerms(role.serverPerms, schema.server);
        origGlobalPerms = hydratePerms(role.globalPerms, schema.global);
        origServerPerms = hydratePerms(role.serverPerms, schema.server);
    }

    async function saveEdit(role: RoleListItem): Promise<void> {
        isSaving = true;
        try {
            await updateRole(role.id, {
                name: editName, description: editDescription, color: editColor,
                globalPerms: editGlobalPerms, serverPerms: editServerPerms,
            });
            roles = roles.map((r) => r.id === role.id
                ? { ...r, name: editName, description: editDescription, color: editColor, globalPerms: editGlobalPerms, serverPerms: editServerPerms }
                : r,
            );
            editingRoleId = null;
            toast.success('Role updated');
        } catch (err: unknown) { toast.error(err instanceof Error ? err.message : 'Failed to save'); }
        isSaving = false;
    }

    async function handleDelete(id: number): Promise<void> {
        actionInProgress = id;
        deleteConfirm = null;
        try {
            await deleteRole(id);
            roles = roles.filter((r) => r.id !== id);
            if (expandedRoleId === id) expandedRoleId = null;
            toast.success('Role deleted');
        } catch (err: unknown) { toast.error(err instanceof Error ? err.message : 'Failed to delete'); }
        actionInProgress = null;
    }

    function initEmptyPerms(): void {
        if (!schema) return;
        newGlobalPerms = {};
        for (const [resource, rs] of Object.entries(schema.global)) {
            newGlobalPerms[resource] = {};
            for (const a of allActionsForResource(rs)) newGlobalPerms[resource][a] = false;
        }
        newServerPerms = {};
        for (const [resource, rs] of Object.entries(schema.server)) {
            newServerPerms[resource] = {};
            for (const a of allActionsForResource(rs)) newServerPerms[resource][a] = false;
        }
    }

    function openCreateForm(): void {
        newRoleName = ''; newRoleDescription = ''; newRoleColor = '#6b7280';
        initEmptyPerms();
        showCreateForm = true;
        expandedRoleId = null; editingRoleId = null;
    }

    async function handleCreate(): Promise<void> {
        if (!newRoleName.trim()) { toast.error('Role name is required'); return; }
        isCreating = true;
        try {
            const created: RoleListItem = await createRole({
                name: newRoleName.trim(), description: newRoleDescription.trim(),
                color: newRoleColor, globalPerms: newGlobalPerms, serverPerms: newServerPerms,
            });
            roles = [...roles, { ...created, globalPerms: newGlobalPerms, serverPerms: newServerPerms, assignedUsers: 0 }];
            showCreateForm = false;
            toast.success('Role created');
        } catch (err: unknown) { toast.error(err instanceof Error ? err.message : 'Failed to create'); }
        isCreating = false;
    }

    function countGranted(perms: Record<string, Record<string, boolean>>): number {
        let c = 0;
        for (const actions of Object.values(perms)) for (const v of Object.values(actions)) if (v) c++;
        return c;
    }

    function toggleCell(perms: Record<string, Record<string, boolean>>, resource: string, action: string): void {
        if (!perms[resource]) perms[resource] = {};
        perms[resource][action] = !perms[resource][action];
    }

    function toggleRow(perms: Record<string, Record<string, boolean>>, resource: string, actions: string[]): void {
        const allOn: boolean = actions.every((a) => perms[resource]?.[a]);
        if (!perms[resource]) perms[resource] = {};
        for (const a of actions) perms[resource][a] = !allOn;
    }

    function toggleCol(perms: Record<string, Record<string, boolean>>, schemaEntries: [string, string[]][], action: string): void {
        const allOn: boolean = schemaEntries.every(([res]) => perms[res]?.[action]);
        for (const [res] of schemaEntries) {
            if (!perms[res]) perms[res] = {};
            perms[res][action] = !allOn;
        }
    }
</script>

{#if !canRead && !authQuery.isLoading}
    <div class="flex h-full items-center justify-center">
        <div class="text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
                <ShieldAlert size={24} class="text-destructive" />
            </div>
            <p class="font-heading text-lg font-semibold text-foreground">Access Denied</p>
            <p class="mt-1 text-sm text-muted-foreground">You don't have permission to manage roles.</p>
        </div>
    </div>
{:else}
    <div class="flex h-full flex-col overflow-y-auto">
        <div class="mx-auto w-full max-w-6xl px-6 py-8">
            <!-- Header -->
            <div class="roles-reveal mb-8">
                <h1 class="mb-1 font-heading text-lg font-semibold text-foreground">Roles</h1>
                <p class="text-sm text-muted-foreground">Define permission sets and assign them to users per server</p>
            </div>

            <!-- Roles Table -->
            <section class="roles-reveal delay-1">
                <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                    <Shield size={14} />
                    Permission Roles
                </h2>

                <div class="rounded-lg border border-border bg-card">
                    <!-- Toolbar -->
                    <div class="flex items-center justify-between border-b border-border/50 px-4 py-2.5">
                        <span class="rounded bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-bold text-primary">
                            {roles.length} {roles.length === 1 ? 'role' : 'roles'}
                        </span>
                        {#if canCreateRole}
                            <button
                                onclick={openCreateForm}
                                disabled={showCreateForm}
                                class="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground transition-opacity hover:opacity-90 active:opacity-80 disabled:opacity-50"
                            >
                                <Plus size={13} />
                                New Role
                            </button>
                        {/if}
                    </div>

                    {#if isLoading}
                        <div class="flex items-center justify-center py-12">
                            <LoaderCircle size={18} class="animate-spin text-muted-foreground" />
                        </div>
                    {:else}
                        <!-- Create Form -->
                        {#if showCreateForm && schema}
                            <div class="border-b border-primary/20 bg-primary/[0.02]">
                                {@render roleForm(
                                    newRoleName, (v: string) => (newRoleName = v),
                                    newRoleDescription, (v: string) => (newRoleDescription = v),
                                    newRoleColor, (v: string) => (newRoleColor = v),
                                    newGlobalPerms, newServerPerms, schema, true,
                                )}
                                <div class="flex items-center justify-end gap-2 border-t border-border/30 px-4 py-3">
                                    <button onclick={() => (showCreateForm = false)} class="rounded-md px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground">Cancel</button>
                                    <button
                                        onclick={handleCreate}
                                        disabled={isCreating || !newRoleName.trim()}
                                        class="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground transition-opacity hover:opacity-90 disabled:opacity-50"
                                    >
                                        {#if isCreating}<LoaderCircle size={12} class="animate-spin" />{:else}<Check size={12} />{/if}
                                        Create Role
                                    </button>
                                </div>
                            </div>
                        {/if}

                        <!-- Role List -->
                        {#if roles.length === 0 && !showCreateForm}
                            <div class="px-4 py-12 text-center text-sm text-muted-foreground/40">No roles defined yet.</div>
                        {:else}
                            {#each roles as role (role.id)}
                                {@const isExpanded = expandedRoleId === role.id}
                                {@const isEditing = editingRoleId === role.id}
                                <div class="border-b border-border/20 last:border-0 {actionInProgress === role.id ? 'opacity-50 pointer-events-none' : ''}">
                                    <!-- Row -->
                                    <div class="group flex items-center px-4 py-3 transition-colors hover:bg-muted/20">
                                        <!-- Expand toggle -->
                                        <button onclick={() => toggleExpand(role.id)} class="mr-2 rounded p-0.5 text-muted-foreground/25 transition-colors hover:text-muted-foreground/60">
                                            {#if isExpanded}<ChevronDown size={14} />{:else}<ChevronRight size={14} />{/if}
                                        </button>

                                        <!-- Color + Name (clickable to expand) -->
                                        <button onclick={() => toggleExpand(role.id)} class="flex min-w-0 flex-1 items-center gap-2.5 text-left">
                                            <div class="h-2.5 w-2.5 shrink-0 rounded-full" style="background-color: {role.color}"></div>
                                            <span class="truncate text-[13px] font-medium text-foreground">{role.name}</span>
                                            {#if role.isSystem}
                                                <span class="shrink-0 rounded-full bg-muted px-1.5 py-0.5 text-[9px] font-medium text-muted-foreground">System</span>
                                            {/if}
                                            {#if role.description}
                                                <span class="hidden truncate text-[11px] text-muted-foreground/35 xl:inline">{role.description}</span>
                                            {/if}
                                        </button>

                                        <!-- Stats -->
                                        <div class="mr-3 hidden items-center gap-5 sm:flex">
                                            <span class="flex items-center gap-1.5 text-[11px] text-muted-foreground/35" title="{role.assignedUsers} user(s) assigned">
                                                <Users size={11} />
                                                <span class="tabular-nums">{role.assignedUsers}</span>
                                            </span>
                                            <span class="text-[11px] tabular-nums text-muted-foreground/35" title="Permissions granted">
                                                {countGranted(role.globalPerms) + countGranted(role.serverPerms)} perms
                                            </span>
                                        </div>

                                        <!-- Actions -->
                                        <div class="flex items-center gap-0.5" onclick={(e: MouseEvent) => e.stopPropagation()}>
                                            {#if deleteConfirm === role.id}
                                                <span class="mr-1 text-[11px] text-muted-foreground">Delete?</span>
                                                <button onclick={() => handleDelete(role.id)} class="rounded-md px-2 py-0.5 text-[11px] font-medium bg-destructive/10 text-destructive hover:bg-destructive/20">Confirm</button>
                                                <button onclick={() => (deleteConfirm = null)} class="rounded-md px-2 py-0.5 text-[11px] text-muted-foreground hover:text-foreground">Cancel</button>
                                            {:else}
                                                {#if canUpdate}
                                                    <button onclick={() => startEdit(role)} class="rounded-md p-1.5 text-muted-foreground/25 transition-colors hover:bg-muted/50 hover:text-foreground opacity-0 group-hover:opacity-100" title="Edit">
                                                        <Pencil size={13} />
                                                    </button>
                                                {/if}
                                                {#if canDeleteRole && !role.isSystem}
                                                    <button onclick={() => (deleteConfirm = role.id)} class="rounded-md p-1.5 text-muted-foreground/25 transition-colors hover:bg-destructive/10 hover:text-destructive opacity-0 group-hover:opacity-100" title="Delete">
                                                        <Trash2 size={13} />
                                                    </button>
                                                {/if}
                                            {/if}
                                        </div>
                                    </div>

                                    <!-- Expanded Panel -->
                                    {#if isExpanded && schema}
                                        <div class="border-t border-border/15 bg-muted/[0.04] px-4 pb-4 pt-3">
                                            {#if isEditing}
                                                {@render roleForm(
                                                    editName, (v: string) => (editName = v),
                                                    editDescription, (v: string) => (editDescription = v),
                                                    editColor, (v: string) => (editColor = v),
                                                    editGlobalPerms, editServerPerms, schema, true,
                                                    origGlobalPerms, origServerPerms,
                                                )}
                                                <div class="mt-4 flex items-center justify-end gap-2">
                                                    <button onclick={() => (editingRoleId = null)} class="rounded-md px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground">Cancel</button>
                                                    <button
                                                        onclick={() => saveEdit(role)}
                                                        disabled={isSaving}
                                                        class="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90 disabled:opacity-50"
                                                    >
                                                        {#if isSaving}<LoaderCircle size={12} class="animate-spin" />{:else}<Check size={12} />{/if}
                                                        Save
                                                    </button>
                                                </div>
                                            {:else}
                                                {#if role.description}
                                                    <p class="mb-3 text-xs text-muted-foreground/40">{role.description}</p>
                                                {/if}
                                                {@render permGrid('Global Permissions', role.globalPerms, schema.global, false)}
                                                <div class="my-3"></div>
                                                {@render permGrid('Server Permissions', role.serverPerms, schema.server, false)}
                                            {/if}
                                        </div>
                                    {/if}
                                </div>
                            {/each}
                        {/if}
                    {/if}
                </div>
            </section>
        </div>
    </div>
{/if}

<!-- Role form (name, desc, color, permission grids) -->
{#snippet roleForm(
    name: string, setName: (v: string) => void,
    desc: string, setDesc: (v: string) => void,
    color: string, setColor: (v: string) => void,
    globalPerms: Record<string, Record<string, boolean>>,
    serverPerms: Record<string, Record<string, boolean>>,
    s: PermissionSchema,
    editable: boolean,
    origGlobal?: Record<string, Record<string, boolean>>,
    origServer?: Record<string, Record<string, boolean>>,
)}
    <div class="px-4 pt-4">
        <div class="mb-5 grid grid-cols-1 gap-3 sm:grid-cols-[1fr_1fr_auto]">
            <div>
                <label class="mb-1 block text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">Name</label>
                <input
                    type="text"
                    value={name}
                    oninput={(e: Event) => setName((e.target as HTMLInputElement).value)}
                    placeholder="e.g. Support"
                    class="w-full rounded-md border border-border bg-background px-2.5 py-1.5 text-xs text-foreground placeholder:text-muted-foreground/25 focus:border-primary/50 focus:outline-none"
                />
            </div>
            <div>
                <label class="mb-1 block text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">Description</label>
                <input
                    type="text"
                    value={desc}
                    oninput={(e: Event) => setDesc((e.target as HTMLInputElement).value)}
                    placeholder="Optional"
                    class="w-full rounded-md border border-border bg-background px-2.5 py-1.5 text-xs text-foreground placeholder:text-muted-foreground/25 focus:border-primary/50 focus:outline-none"
                />
            </div>
            <div>
                <label class="mb-1 block text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">Color</label>
                <div class="flex items-center gap-1 pt-0.5">
                    {#each ROLE_COLORS as c}
                        <button
                            onclick={() => setColor(c)}
                            class="h-[22px] w-[22px] rounded-full border-2 transition-all
                                {color === c ? 'border-foreground/70 ring-1 ring-foreground/20 scale-110' : 'border-transparent hover:scale-110'}"
                            style="background-color: {c}"
                        ></button>
                    {/each}
                </div>
            </div>
        </div>

        {@render permGrid('Global Permissions', globalPerms, s.global, editable, origGlobal)}
        <div class="my-3"></div>
        {@render permGrid('Server Permissions', serverPerms, s.server, editable, origServer)}
    </div>
{/snippet}

<!-- Permission Grid Matrix -->
{#snippet permGrid(
    label: string,
    perms: Record<string, Record<string, boolean>>,
    schemaSection: Record<string, ResourceActionSchema>,
    editable: boolean,
    originalPerms?: Record<string, Record<string, boolean>>,
)}
    {@const entries = Object.entries(schemaSection)}
    {@const crudCols = ['create', 'read', 'update', 'delete'] as const}

    <div class="rounded-md border border-border/30 overflow-hidden">
        <table class="w-full">
            <thead>
                <tr class="bg-muted/40">
                    <th class="px-3 py-2 text-left">
                        <span class="text-[10px] font-semibold tracking-widest text-muted-foreground/50 uppercase">{label}</span>
                    </th>
                    {#each crudCols as action}
                        <th class="w-[80px] px-2 py-2 text-center">
                            {#if editable}
                                <button
                                    onclick={() => toggleCol(perms, entries.map(([r, rs]) => [r, rs.crud] as [string, string[]]), action)}
                                    class="text-[10px] font-semibold tracking-wider uppercase transition-colors hover:text-primary
                                        {entries.every(([r, rs]) => !rs.crud.includes(action) || perms[r]?.[action]) ? 'text-emerald-500' : 'text-muted-foreground/40'}"
                                >
                                    {action.charAt(0).toUpperCase() + action.slice(1)}
                                </button>
                            {:else}
                                <span class="text-[10px] font-semibold tracking-wider text-muted-foreground/40 uppercase">
                                    {action.charAt(0).toUpperCase() + action.slice(1)}
                                </span>
                            {/if}
                        </th>
                    {/each}
                </tr>
            </thead>
            <tbody>
                {#each entries as [resource, rs]}
                    {@const hasSub = (rs.sub?.length ?? 0) > 0}
                    {@const subKey = `${label}:${resource}`}
                    {@const isSubExpanded = expandedSubRows.has(subKey)}
                    {@const allCrudGranted = rs.crud.every((a) => perms[resource]?.[a])}

                    <!-- Main CRUD row -->
                    <tr class="border-t border-border/15 transition-colors {editable ? 'hover:bg-muted/20' : ''}">
                        <td class="px-3 py-1.5">
                            <div class="flex items-center gap-1.5">
                                {#if hasSub}
                                    <button
                                        onclick={() => {
                                            const next = new Set(expandedSubRows);
                                            if (next.has(subKey)) next.delete(subKey); else next.add(subKey);
                                            expandedSubRows = next;
                                        }}
                                        class="rounded p-0.5 text-muted-foreground/25 transition-colors hover:text-muted-foreground/60"
                                    >
                                        {#if isSubExpanded}<ChevronDown size={11} />{:else}<ChevronRight size={11} />{/if}
                                    </button>
                                {/if}
                                {#if editable}
                                    <button
                                        onclick={() => toggleRow(perms, resource, allActionsForResource(rs))}
                                        class="text-[12px] font-medium text-foreground transition-colors hover:text-primary"
                                    >
                                        {RESOURCE_LABELS[resource] ?? resource}
                                    </button>
                                {:else}
                                    <span class="text-[12px] font-medium {allCrudGranted ? 'text-foreground' : 'text-muted-foreground/50'}">
                                        {RESOURCE_LABELS[resource] ?? resource}
                                    </span>
                                {/if}
                                {#if hasSub && !isSubExpanded}
                                    {@const subGranted = (rs.sub ?? []).filter((a) => perms[resource]?.[a] === true).length}
                                    <span class="rounded bg-muted/60 px-1 py-0.5 text-[9px] tabular-nums text-muted-foreground/30">
                                        +{rs.sub?.length}
                                    </span>
                                {/if}
                            </div>
                        </td>
                        {#each crudCols as action}
                            {@const hasAction = rs.crud.includes(action)}
                            {@const granted = hasAction && perms[resource]?.[action] === true}
                            {@const dirty = hasAction && editable && originalPerms != null && isCellDirty(perms, originalPerms, resource, action)}
                            <td class="px-2 py-1.5 text-center">
                                {@render permCell(hasAction, granted, dirty, editable, perms, resource, action)}
                            </td>
                        {/each}
                    </tr>

                    <!-- Sub-action rows (expanded) -->
                    {#if hasSub && isSubExpanded}
                        {#each rs.sub ?? [] as subAction}
                            {@const granted = perms[resource]?.[subAction] === true}
                            {@const dirty = editable && originalPerms != null && isCellDirty(perms, originalPerms, resource, subAction)}
                            <tr class="border-t border-border/10 bg-muted/[0.03] transition-colors {editable ? 'hover:bg-muted/15' : ''}">
                                <td class="py-1.5 pl-9 pr-3" colspan="1">
                                    <span class="text-[11px] text-muted-foreground/50">{ACTION_LABELS[subAction] ?? subAction}</span>
                                </td>
                                <td colspan="4" class="px-2 py-1.5">
                                    <div class="flex items-center gap-2">
                                        {#if editable}
                                            <button
                                                onclick={() => toggleCell(perms, resource, subAction)}
                                                class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[10px] font-medium transition-all
                                                    {dirty
                                                        ? granted
                                                            ? 'bg-amber-400/20 text-amber-500 ring-2 ring-amber-400/40'
                                                            : 'bg-amber-400/10 text-amber-400/50 ring-2 ring-amber-400/40'
                                                        : granted
                                                            ? 'bg-emerald-500/15 text-emerald-500 hover:bg-emerald-500/25'
                                                            : 'bg-muted/40 text-muted-foreground/20 hover:bg-muted/70 hover:text-muted-foreground/40'}"
                                            >
                                                {#if granted}<Check size={10} strokeWidth={2.5} />{:else}<X size={10} strokeWidth={1.5} />{/if}
                                                {granted ? 'Allowed' : 'Denied'}
                                            </button>
                                        {:else}
                                            <span class="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[10px] font-medium
                                                {granted ? 'bg-emerald-500/15 text-emerald-500' : 'text-muted-foreground/15'}">
                                                {#if granted}<Check size={10} strokeWidth={2.5} />{:else}<X size={10} strokeWidth={1.5} />{/if}
                                                {granted ? 'Allowed' : 'Denied'}
                                            </span>
                                        {/if}
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    {/if}
                {/each}
            </tbody>
        </table>
    </div>
{/snippet}

<!-- Single permission cell (CRUD column) -->
{#snippet permCell(
    hasAction: boolean,
    granted: boolean,
    dirty: boolean,
    editable: boolean,
    perms: Record<string, Record<string, boolean>>,
    resource: string,
    action: string,
)}
    {#if !hasAction}
        <span class="inline-block h-4 w-4 cursor-default text-muted-foreground/10" title="{RESOURCE_LABELS[resource] ?? resource} does not support {action}">—</span>
    {:else if editable}
        <button
            onclick={() => toggleCell(perms, resource, action)}
            class="inline-flex h-6 w-6 items-center justify-center rounded transition-all
                {dirty
                    ? granted
                        ? 'bg-amber-400/20 text-amber-500 ring-2 ring-amber-400/40 hover:bg-amber-400/30'
                        : 'bg-amber-400/10 text-amber-400/50 ring-2 ring-amber-400/40 hover:bg-amber-400/20'
                    : granted
                        ? 'bg-emerald-500/15 text-emerald-500 hover:bg-emerald-500/25'
                        : 'bg-muted/40 text-muted-foreground/15 hover:bg-muted/70 hover:text-muted-foreground/30'}"
        >
            {#if granted}
                <Check size={12} strokeWidth={2.5} />
            {:else}
                <X size={12} strokeWidth={1.5} />
            {/if}
        </button>
    {:else}
        <span
            class="inline-flex h-6 w-6 items-center justify-center rounded
                {granted ? 'bg-emerald-500/15 text-emerald-500' : 'text-muted-foreground/10'}"
        >
            {#if granted}
                <Check size={12} strokeWidth={2.5} />
            {:else}
                <X size={12} strokeWidth={1.5} />
            {/if}
        </span>
    {/if}
{/snippet}

<style>
    .roles-reveal {
        animation: reveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }
    .delay-1 { animation-delay: 0.1s; }

    @keyframes reveal {
        from { opacity: 0; transform: translateY(12px); }
        to { opacity: 1; transform: translateY(0); }
    }
</style>
