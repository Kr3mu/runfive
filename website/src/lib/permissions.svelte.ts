/**
 * RBAC permission helpers for the frontend.
 *
 * Provides pure functions to check global and server-scoped permissions
 * from the AuthUser object returned by GET /v1/auth/me.
 * Components use these in $derived blocks for reactive permission checks.
 *
 * @see AuthUser for the permission data shape
 */

import type { AuthUser } from '$lib/api/auth';

/** Supported RBAC actions, including server-scoped sub-actions. */
type Action = 'create' | 'read' | 'update' | 'delete' | 'kick' | 'warn' | 'execute';

/**
 * Checks whether a user has a specific global (panel-wide) permission.
 *
 * @param user - The authenticated user from the auth query
 * @param resource - Global resource name (e.g. "users", "roles", "settings")
 * @param action - CRUD action to check
 * @returns True if the user has the permission or is the owner
 */
export function canGlobal(user: AuthUser | null | undefined, resource: string, action: Action): boolean {
  if (!user) return false;
  if (user.isOwner) return true;
  return user.globalPermissions?.[resource]?.[action] === true;
}

/**
 * Checks whether a user has a specific per-server permission.
 *
 * @param user - The authenticated user from the auth query
 * @param serverId - The server directory name
 * @param resource - Server resource name (e.g. "players", "console", "bans")
 * @param action - CRUD action to check
 * @returns True if the user has the permission on this server or is the owner
 */
export function canServer(
  user: AuthUser | null | undefined,
  serverId: string | null | undefined,
  resource: string,
  action: Action,
): boolean {
  if (!user || !serverId) return false;
  if (user.isOwner) return true;
  return user.serverPermissions?.[serverId]?.permissions?.[resource]?.[action] === true;
}

/**
 * Returns the list of server IDs the user has any access to.
 *
 * @param user - The authenticated user from the auth query
 * @returns Array of server ID strings
 */
export function accessibleServerIds(user: AuthUser | null | undefined): string[] {
  if (!user) return [];
  if (user.isOwner) return []; // owner sees all, handled separately
  return Object.keys(user.serverPermissions ?? {});
}

/**
 * Checks whether a user has any permission on a given server.
 *
 * @param user - The authenticated user from the auth query
 * @param serverId - The server directory name
 * @returns True if the user has at least one permission on this server
 */
export function hasAnyServerAccess(user: AuthUser | null | undefined, serverId: string): boolean {
  if (!user) return false;
  if (user.isOwner) return true;
  return serverId in (user.serverPermissions ?? {});
}

/**
 * Checks whether a user has any global permission at all (for showing panel nav).
 *
 * @param user - The authenticated user from the auth query
 * @returns True if the user has at least one global permission
 */
export function hasAnyGlobalAccess(user: AuthUser | null | undefined): boolean {
  if (!user) return false;
  if (user.isOwner) return true;
  const perms = user.globalPermissions;
  if (!perms) return false;
  for (const resource of Object.values(perms)) {
    for (const granted of Object.values(resource)) {
      if (granted) return true;
    }
  }
  return false;
}
