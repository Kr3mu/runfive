/**
 * Role management API client.
 *
 * Provides typed fetch wrappers for all /v1/roles and role-assignment
 * endpoints used by the RBAC management UI.
 *
 * @see RoleListItem for the role shape
 */

import type { RoleInfo } from './auth';

/** Permission map: resource -> action -> granted. */
type PermissionMap = Record<string, Record<string, boolean>>;

/** Single role entry returned by GET /v1/roles. */
export interface RoleListItem {
  /** Role database ID */
  id: number;
  /** Role display name */
  name: string;
  /** Optional description */
  description: string;
  /** Hex color for UI badges */
  color: string;
  /** Panel-wide permission map */
  globalPerms: PermissionMap;
  /** Server resource permission map */
  serverPerms: PermissionMap;
  /** Whether this is a system role (cannot be deleted) */
  isSystem: boolean;
  /** Display order */
  position: number;
  /** Number of users assigned to this role */
  assignedUsers: number;
}

/** Body for creating a new role. */
interface CreateRoleBody {
  /** Role name */
  name: string;
  /** Optional description */
  description?: string;
  /** Hex color */
  color: string;
  /** Panel-wide permissions */
  globalPerms: PermissionMap;
  /** Server resource permissions */
  serverPerms: PermissionMap;
}

/** Body for updating an existing role. */
interface UpdateRoleBody {
  /** New name */
  name?: string;
  /** New description */
  description?: string;
  /** New color */
  color?: string;
  /** New global permissions */
  globalPerms?: PermissionMap;
  /** New server permissions */
  serverPerms?: PermissionMap;
  /** New display position */
  position?: number;
}

/** Action schema for a single resource. */
export interface ResourceActionSchema {
  /** CRUD actions available for this resource */
  crud: string[];
  /** Resource-specific sub-actions (e.g. kick, warn, execute) */
  sub?: string[];
}

/** Permission schema returned by GET /v1/permissions/schema. */
export interface PermissionSchema {
  /** Global resource action schemas */
  global: Record<string, ResourceActionSchema>;
  /** Server resource action schemas */
  server: Record<string, ResourceActionSchema>;
}

/** Server role assignment for a user. */
export interface UserServerRoleEntry {
  /** Server directory name */
  serverId: string;
  /** Assigned role */
  role: RoleInfo;
}

/**
 * Fetches all roles.
 *
 * @returns List of roles with assignment counts
 */
export async function fetchRoles(): Promise<RoleListItem[]> {
  const res: Response = await fetch('/v1/roles');
  if (!res.ok) throw new Error(`GET /v1/roles failed: ${res.status}`);
  return (await res.json()) as RoleListItem[];
}

/**
 * Fetches a single role by ID.
 *
 * @param id - Role database ID
 * @returns Role details
 */
export async function fetchRole(id: number): Promise<RoleListItem> {
  const res: Response = await fetch(`/v1/roles/${id}`);
  if (!res.ok) throw new Error(`GET /v1/roles/${id} failed: ${res.status}`);
  return (await res.json()) as RoleListItem;
}

/**
 * Creates a new role.
 *
 * @param body - Role data
 * @returns Created role
 */
export async function createRole(body: CreateRoleBody): Promise<RoleListItem> {
  const res: Response = await fetch('/v1/roles', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Create role failed: ${res.status}`);
  }
  return (await res.json()) as RoleListItem;
}

/**
 * Updates an existing role.
 *
 * @param id - Role database ID
 * @param body - Fields to update
 */
export async function updateRole(id: number, body: UpdateRoleBody): Promise<void> {
  const res: Response = await fetch(`/v1/roles/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Update role failed: ${res.status}`);
  }
}

/**
 * Deletes a role. Fails if it is a system role or assigned to any user.
 *
 * @param id - Role database ID
 */
export async function deleteRole(id: number): Promise<void> {
  const res: Response = await fetch(`/v1/roles/${id}`, { method: 'DELETE' });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Delete role failed: ${res.status}`);
  }
}

/**
 * Fetches the permission schema (canonical resource + action list).
 *
 * @returns Global and server resource actions
 */
export async function fetchPermissionSchema(): Promise<PermissionSchema> {
  const res: Response = await fetch('/v1/permissions/schema');
  if (!res.ok) throw new Error(`GET /v1/permissions/schema failed: ${res.status}`);
  return (await res.json()) as PermissionSchema;
}

/**
 * Sets or clears the global role for a user.
 *
 * @param userId - User database ID
 * @param roleId - Role ID to assign, or null to remove
 */
export async function setGlobalRole(userId: number, roleId: number | null): Promise<void> {
  const res: Response = await fetch(`/v1/users/${userId}/global-role`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ roleId }),
  });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Set global role failed: ${res.status}`);
  }
}

/**
 * Fetches all server role assignments for a user.
 *
 * @param userId - User database ID
 * @returns List of server role entries
 */
export async function fetchUserServerRoles(userId: number): Promise<UserServerRoleEntry[]> {
  const res: Response = await fetch(`/v1/users/${userId}/server-roles`);
  if (!res.ok) throw new Error(`GET /v1/users/${userId}/server-roles failed: ${res.status}`);
  return (await res.json()) as UserServerRoleEntry[];
}

/**
 * Assigns a role to a user on a specific server.
 *
 * @param userId - User database ID
 * @param serverId - Server directory name
 * @param roleId - Role ID to assign
 */
export async function setServerRole(userId: number, serverId: string, roleId: number): Promise<void> {
  const res: Response = await fetch(`/v1/users/${userId}/server-roles/${serverId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ roleId }),
  });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Set server role failed: ${res.status}`);
  }
}

/**
 * Removes a user's role assignment on a specific server.
 *
 * @param userId - User database ID
 * @param serverId - Server directory name
 */
export async function removeServerRole(userId: number, serverId: string): Promise<void> {
  const res: Response = await fetch(`/v1/users/${userId}/server-roles/${serverId}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const err: { error: string } = (await res.json()) as { error: string };
    throw new Error(err.error ?? `Remove server role failed: ${res.status}`);
  }
}
