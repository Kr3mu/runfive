/**
 * User management API client.
 *
 * Provides typed fetch wrappers for /v1/users endpoints (owner-only).
 */

/** Linked provider info matching the backend ProviderInfo shape. */
interface UserProviders {
  cfx: { id: number; username: string; avatarUrl: string } | null;
  discord: { id: string; username: string; avatar: string } | null;
}

/** A user in the panel user list. */
export interface PanelUser {
  id: number;
  username: string;
  isOwner: boolean;
  hasPassword: boolean;
  providers: UserProviders;
  suspendedAt: string | null;
  createdAt: string;
}

/** Fetches all users. Owner-only. */
export async function fetchUsers(): Promise<PanelUser[]> {
  const res: Response = await fetch('/v1/users');
  if (!res.ok) throw new Error(`GET /v1/users failed: ${res.status}`);
  return (await res.json()) as PanelUser[];
}

/** Suspends a user, revoking all their sessions. */
export async function suspendUser(id: number): Promise<void> {
  const res: Response = await fetch(`/v1/users/${id}/suspend`, { method: 'POST' });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Suspend failed: ${res.status}`);
  }
}

/** Unsuspends a previously suspended user. */
export async function unsuspendUser(id: number): Promise<void> {
  const res: Response = await fetch(`/v1/users/${id}/unsuspend`, { method: 'POST' });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Unsuspend failed: ${res.status}`);
  }
}

/** Permanently deletes a user and all their sessions. */
export async function deleteUser(id: number): Promise<void> {
  const res: Response = await fetch(`/v1/users/${id}`, { method: 'DELETE' });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Delete failed: ${res.status}`);
  }
}
