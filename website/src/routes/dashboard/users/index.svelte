<script lang="ts">
    import { createQuery } from '@tanstack/svelte-query';
    import { authQueryOptions } from '$lib/api/auth';
    import { fetchUsers, suspendUser, unsuspendUser, deleteUser, type PanelUser } from '$lib/api/users';
    import Cfxre from '$lib/components/icons/cfxre.svelte';
    import DiscordIcon from '$lib/components/icons/discord.svelte';
    import Users from '@lucide/svelte/icons/users';
    import UserPlus from '@lucide/svelte/icons/user-plus';
    import Shield from '@lucide/svelte/icons/shield';
    import ShieldAlert from '@lucide/svelte/icons/shield-alert';
    import ShieldCheck from '@lucide/svelte/icons/shield-check';
    import KeyRound from '@lucide/svelte/icons/key-round';
    import Ban from '@lucide/svelte/icons/ban';
    import Trash2 from '@lucide/svelte/icons/trash-2';
    import Plus from '@lucide/svelte/icons/plus';
    import Link from '@lucide/svelte/icons/link';
    import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
    import LoaderCircle from '@lucide/svelte/icons/loader-circle';
    import Clock from '@lucide/svelte/icons/clock';

    const authQuery = createQuery(() => authQueryOptions());
    const currentUser = $derived(authQuery.data);
    const isOwner = $derived(currentUser?.isOwner ?? false);

    let users = $state<PanelUser[]>([]);
    let isLoadingUsers = $state(true);
    let pendingInvites = $state<{ id: number; token: string; createdAt: string; expiresAt: string }[]>([]);
    let isLoadingInvites = $state(true);

    // Action states
    let actionInProgress = $state<number | null>(null);
    let confirmDialog = $state<{ type: 'suspend' | 'delete'; userId: number } | null>(null);
    let inviteCreating = $state(false);
    let inviteCopiedId = $state<number | null>(null);

    $effect((): void => {
        if (!isOwner) return;
        fetchUsers()
            .then((u): void => { users = u; isLoadingUsers = false; })
            .catch((): void => { isLoadingUsers = false; });
        fetchInvites();
    });

    async function fetchInvites(): Promise<void> {
        try {
            const res = await fetch('/v1/invites');
            if (res.ok) pendingInvites = await res.json();
        } catch { /* ignore */ }
        isLoadingInvites = false;
    }

    async function handleSuspend(id: number): Promise<void> {
        actionInProgress = id;
        confirmDialog = null;
        try {
            await suspendUser(id);
            users = users.map((u) =>
                u.id === id ? { ...u, suspendedAt: new Date().toISOString() } : u,
            );
        } catch { /* ignore */ }
        actionInProgress = null;
    }

    async function handleUnsuspend(id: number): Promise<void> {
        actionInProgress = id;
        try {
            await unsuspendUser(id);
            users = users.map((u) =>
                u.id === id ? { ...u, suspendedAt: null } : u,
            );
        } catch { /* ignore */ }
        actionInProgress = null;
    }

    async function handleDelete(id: number): Promise<void> {
        actionInProgress = id;
        confirmDialog = null;
        try {
            await deleteUser(id);
            users = users.filter((u) => u.id !== id);
        } catch { /* ignore */ }
        actionInProgress = null;
    }

    async function createInvite(): Promise<void> {
        if (inviteCreating) return;
        inviteCreating = true;
        try {
            const res = await fetch('/v1/invites', { method: 'POST' });
            if (!res.ok) return;
            const data: { id: number; token: string; url: string; expiresAt: string } = await res.json();
            await navigator.clipboard.writeText(data.url);
            inviteCopiedId = data.id;
            setTimeout(() => (inviteCopiedId = null), 2000);
            await fetchInvites();
        } finally {
            inviteCreating = false;
        }
    }

    async function copyInviteLink(invite: { id: number; token: string }): Promise<void> {
        const url = `${window.location.origin}/invite/accept?token=${invite.token}`;
        await navigator.clipboard.writeText(url);
        inviteCopiedId = invite.id;
        setTimeout(() => (inviteCopiedId = null), 2000);
    }

    async function revokeInvite(id: number): Promise<void> {
        await fetch(`/v1/invites/${id}`, { method: 'DELETE' });
        pendingInvites = pendingInvites.filter((i) => i.id !== id);
    }

    function formatDate(iso: string): string {
        const d = new Date(iso);
        const now = new Date();
        const diff = now.getTime() - d.getTime();
        const mins = Math.floor(diff / 60000);
        if (mins < 1) return 'Just now';
        if (mins < 60) return `${mins}m ago`;
        const hours = Math.floor(mins / 60);
        if (hours < 24) return `${hours}h ago`;
        const days = Math.floor(hours / 24);
        return `${days}d ago`;
    }

    function formatTimeLeft(iso: string): string {
        const diff = new Date(iso).getTime() - Date.now();
        if (diff <= 0) return 'Expired';
        const h = Math.floor(diff / 3600000);
        const m = Math.floor((diff % 3600000) / 60000);
        return h > 0 ? `${h}h ${m}m left` : `${m}m left`;
    }
</script>

{#if !isOwner && !authQuery.isLoading}
    <!-- Access denied -->
    <div class="flex h-full items-center justify-center">
        <div class="text-center">
            <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
                <ShieldAlert size={24} class="text-destructive" />
            </div>
            <p class="font-heading text-lg font-semibold text-foreground">Access Denied</p>
            <p class="mt-1 text-sm text-muted-foreground">Only the owner can manage users.</p>
        </div>
    </div>
{:else}
    <div class="flex h-full flex-col overflow-y-auto">
        <div class="mx-auto w-full max-w-6xl px-6 py-8">
            <!-- Header -->
            <div class="users-reveal mb-8">
                <h1 class="mb-1 font-heading text-lg font-semibold text-foreground">Users</h1>
                <p class="text-sm text-muted-foreground">Manage panel access and invitations</p>
            </div>

            <!-- Users Section -->
            <section class="users-reveal mb-8 delay-1">
                <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                    <Users size={14} />
                    Panel Users
                </h2>

                <div class="rounded-lg border border-border bg-card">
                    <!-- Toolbar -->
                    <div class="flex items-center justify-between border-b border-border/50 px-4 py-2.5">
                        <span class="rounded bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-bold text-primary">
                            {users.length} {users.length === 1 ? 'user' : 'users'}
                        </span>
                    </div>

                    {#if isLoadingUsers}
                        <div class="flex items-center justify-center py-12">
                            <LoaderCircle size={18} class="animate-spin text-muted-foreground" />
                        </div>
                    {:else if users.length === 0}
                        <div class="px-4 py-12 text-center text-sm text-muted-foreground/40">
                            No users yet. Create an invite to get started.
                        </div>
                    {:else}
                        <table class="w-full">
                            <thead>
                                <tr class="border-b border-border/30 text-left">
                                    <th class="px-4 py-2 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">User</th>
                                    <th class="px-4 py-2 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">Auth</th>
                                    <th class="px-4 py-2 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">Status</th>
                                    <th class="hidden px-4 py-2 text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase lg:table-cell">Created</th>
                                    <th class="px-4 py-2 text-right text-[10px] font-semibold tracking-widest text-muted-foreground/40 uppercase">
                                        <span class="sr-only">Actions</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each users as u (u.id)}
                                    <tr
                                        class="group border-b border-border/20 last:border-0 transition-colors hover:bg-muted/30
                                            {u.id === currentUser?.id ? 'border-l-2 border-l-primary/40' : ''}
                                            {actionInProgress === u.id ? 'opacity-60 pointer-events-none' : ''}"
                                    >
                                        <!-- User -->
                                        <td class="px-4 py-3">
                                            <div class="flex items-center gap-3">
                                                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
                                                    <span class="text-xs font-bold">{u.username.charAt(0).toUpperCase()}</span>
                                                </div>
                                                <div class="min-w-0">
                                                    <div class="flex items-center gap-2">
                                                        <span class="truncate text-[13px] font-medium text-foreground">{u.username}</span>
                                                        {#if u.isOwner}
                                                            <span class="inline-flex shrink-0 items-center gap-1 rounded-full bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-500">
                                                                <Shield size={9} />
                                                                Owner
                                                            </span>
                                                        {/if}
                                                        {#if u.id === currentUser?.id}
                                                            <span class="shrink-0 rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                                                                You
                                                            </span>
                                                        {/if}
                                                    </div>
                                                </div>
                                            </div>
                                        </td>

                                        <!-- Auth methods -->
                                        <td class="px-4 py-3">
                                            <div class="flex items-center gap-1.5">
                                                {#if u.hasPassword}
                                                    <span class="inline-flex items-center rounded-full bg-foreground/5 px-1.5 py-0.5" title="Password">
                                                        <KeyRound size={11} class="text-foreground/40" />
                                                    </span>
                                                {/if}
                                                {#if u.providers.cfx}
                                                    <span class="inline-flex items-center rounded-full bg-[#F40552]/10 px-1.5 py-0.5" title="Cfx.re: {u.providers.cfx.username}">
                                                        <Cfxre class="h-2.5 w-auto text-[#F40552]" />
                                                    </span>
                                                {/if}
                                                {#if u.providers.discord}
                                                    <span class="inline-flex items-center rounded-full bg-[#5865F2]/10 px-1.5 py-0.5" title="Discord: {u.providers.discord.username}">
                                                        <DiscordIcon class="h-2.5 w-2.5 text-[#5865F2]" />
                                                    </span>
                                                {/if}
                                            </div>
                                        </td>

                                        <!-- Status -->
                                        <td class="px-4 py-3">
                                            {#if u.suspendedAt}
                                                <span class="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[10px] font-medium text-destructive">
                                                    Suspended
                                                </span>
                                            {:else}
                                                <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-500">
                                                    Active
                                                </span>
                                            {/if}
                                        </td>

                                        <!-- Created -->
                                        <td class="hidden px-4 py-3 lg:table-cell">
                                            <span class="text-xs text-muted-foreground/60">{formatDate(u.createdAt)}</span>
                                        </td>

                                        <!-- Actions -->
                                        <td class="px-4 py-3 text-right">
                                            {#if !u.isOwner && u.id !== currentUser?.id}
                                                {#if confirmDialog?.userId === u.id}
                                                    <!-- Inline confirm -->
                                                    <div class="inline-flex items-center gap-2">
                                                        <span class="text-[11px] text-muted-foreground">
                                                            {confirmDialog.type === 'delete' ? 'Delete?' : 'Suspend?'}
                                                        </span>
                                                        <button
                                                            onclick={() => confirmDialog?.type === 'delete' ? handleDelete(u.id) : handleSuspend(u.id)}
                                                            class="rounded-md px-2 py-0.5 text-[11px] font-medium bg-destructive/10 text-destructive transition-colors hover:bg-destructive/20"
                                                        >
                                                            Confirm
                                                        </button>
                                                        <button
                                                            onclick={() => (confirmDialog = null)}
                                                            class="rounded-md px-2 py-0.5 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
                                                        >
                                                            Cancel
                                                        </button>
                                                    </div>
                                                {:else}
                                                    <div class="inline-flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                                                        {#if actionInProgress === u.id}
                                                            <LoaderCircle size={14} class="animate-spin text-muted-foreground" />
                                                        {:else}
                                                            {#if u.suspendedAt}
                                                                <button
                                                                    onclick={() => handleUnsuspend(u.id)}
                                                                    class="rounded-md p-1.5 text-muted-foreground/40 transition-colors hover:bg-emerald-500/10 hover:text-emerald-500"
                                                                    title="Unsuspend"
                                                                >
                                                                    <ShieldCheck size={14} />
                                                                </button>
                                                            {:else}
                                                                <button
                                                                    onclick={() => (confirmDialog = { type: 'suspend', userId: u.id })}
                                                                    class="rounded-md p-1.5 text-muted-foreground/40 transition-colors hover:bg-amber-500/10 hover:text-amber-500"
                                                                    title="Suspend"
                                                                >
                                                                    <Ban size={14} />
                                                                </button>
                                                            {/if}
                                                            <button
                                                                onclick={() => (confirmDialog = { type: 'delete', userId: u.id })}
                                                                class="rounded-md p-1.5 text-muted-foreground/40 transition-colors hover:bg-destructive/10 hover:text-destructive"
                                                                title="Delete"
                                                            >
                                                                <Trash2 size={14} />
                                                            </button>
                                                        {/if}
                                                    </div>
                                                {/if}
                                            {/if}
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    {/if}
                </div>
            </section>

            <!-- Invites Section -->
            <section class="users-reveal mb-8 delay-2">
                <h2 class="mb-3 flex items-center gap-2 text-xs font-semibold tracking-widest text-muted-foreground/60 uppercase">
                    <UserPlus size={14} />
                    Invitations
                </h2>

                <div class="rounded-lg border border-border bg-card p-4">
                    <!-- Top row -->
                    <div class="mb-4 flex items-center justify-between">
                        <div class="flex items-center gap-2 text-xs text-muted-foreground/50">
                            <Clock size={12} />
                            Invite links expire after 24 hours
                        </div>
                        <button
                            onclick={createInvite}
                            disabled={inviteCreating}
                            class="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground transition-opacity hover:opacity-90 active:opacity-80 disabled:opacity-50"
                        >
                            {#if inviteCreating}
                                <LoaderCircle size={13} class="animate-spin" />
                                Creating...
                            {:else}
                                <Plus size={13} />
                                New Invite
                            {/if}
                        </button>
                    </div>

                    <!-- Pending invites -->
                    {#if isLoadingInvites}
                        <div class="flex items-center justify-center py-6">
                            <LoaderCircle size={16} class="animate-spin text-muted-foreground/40" />
                        </div>
                    {:else if pendingInvites.length === 0}
                        <p class="py-6 text-center text-sm text-muted-foreground/30">No pending invites</p>
                    {:else}
                        <div class="space-y-2">
                            {#each pendingInvites as invite (invite.id)}
                                <div class="flex items-center justify-between rounded-md border border-border/50 bg-background/50 px-3 py-2.5">
                                    <div class="flex items-center gap-3">
                                        <Link size={13} class="shrink-0 text-muted-foreground/30" />
                                        <div class="min-w-0">
                                            <p class="truncate font-mono text-xs text-foreground/70">{invite.token.slice(0, 20)}...</p>
                                            <p class="text-[10px] {formatTimeLeft(invite.expiresAt) === 'Expired' ? 'text-destructive/60' : 'text-muted-foreground/40'}">
                                                {formatTimeLeft(invite.expiresAt)}
                                            </p>
                                        </div>
                                    </div>
                                    <div class="flex items-center gap-1">
                                        <button
                                            onclick={() => copyInviteLink(invite)}
                                            class="rounded-md p-1.5 text-muted-foreground/30 transition-colors hover:text-primary {inviteCopiedId === invite.id ? 'text-emerald-500' : ''}"
                                            title="Copy invite link"
                                        >
                                            {#if inviteCopiedId === invite.id}
                                                <ClipboardCheck size={14} />
                                            {:else}
                                                <Link size={14} />
                                            {/if}
                                        </button>
                                        <button
                                            onclick={() => revokeInvite(invite.id)}
                                            class="rounded-md p-1.5 text-muted-foreground/30 transition-colors hover:bg-destructive/10 hover:text-destructive"
                                            title="Revoke invite"
                                        >
                                            <Trash2 size={14} />
                                        </button>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            </section>
        </div>
    </div>
{/if}

<style>
    .users-reveal {
        animation: reveal 0.6s cubic-bezier(0.16, 1, 0.3, 1) both;
    }

    .delay-1 { animation-delay: 0.1s; }
    .delay-2 { animation-delay: 0.2s; }

    @keyframes reveal {
        from {
            opacity: 0;
            transform: translateY(12px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
