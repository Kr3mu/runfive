/**
 * Authentication API client and TanStack Query options.
 *
 * Provides typed fetch wrappers for all /v1/auth endpoints and
 * query options for reactive auth state in Svelte components.
 *
 * @see MeResponse for the authenticated user shape
 */

import {
  queryOptions,
  type UndefinedInitialDataOptions,
} from '@tanstack/svelte-query';

/** Linked Cfx.re account information. */
interface CfxInfo {
  /** Discourse user ID on forum.cfx.re */
  id: number;
  /** Discourse username */
  username: string;
  /** Avatar URL template */
  avatarUrl: string;
}

/** Linked Discord account information (planned). */
interface DiscordInfo {
  /** Discord user ID (snowflake) */
  id: string;
  /** Discord username */
  username: string;
  /** Discord avatar hash */
  avatar: string;
}

/** Linked authentication provider details. */
interface ProviderInfo {
  /** Cfx.re account, null if not linked */
  cfx: CfxInfo | null;
  /** Discord account, null if not linked */
  discord: DiscordInfo | null;
}

/** Authenticated user profile returned by GET /v1/auth/me. */
// TODO: Add serverRoles field (Record<serverId, Role>) once RBAC lands.
// Server IDs come from the TOML directory names, not the DB.
export interface AuthUser {
  /** User database ID */
  id: number;
  /** Username */
  username: string;
  /** Whether this user is the owner (master account) */
  isOwner: boolean;
  /** Linked authentication providers */
  providers: ProviderInfo;
}

/** Single active session entry returned by GET /v1/auth/sessions. */
export interface SessionEntry {
  /** Session database ID */
  id: number;
  /** Client User-Agent */
  userAgent: string;
  /** ISO 8601 creation timestamp */
  createdAt: string;
  /** ISO 8601 last activity timestamp */
  lastSeenAt: string;
  /** Whether this is the current request's session */
  isCurrent: boolean;
}

/** Setup status returned by GET /v1/auth/setup-status. */
interface SetupStatus {
  /** True when no users exist and master account registration is required */
  needsSetup: boolean;
}

/** Discord OAuth credentials returned by GET /v1/auth/master/getdiscord. */
export interface DiscordAuthentication {
  /** Discord application client ID */
  clientId: string;
  /** Discord application client secret */
  clientSecret: string;
}

/**
 * Fetches the current authenticated user or returns null if not logged in.
 *
 * @returns AuthUser or null
 */
async function fetchMe(): Promise<AuthUser | null> {
  const res: Response = await fetch('/v1/auth/me');
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`GET /v1/auth/me failed: ${res.status}`);
  return (await res.json()) as AuthUser;
}

/** TanStack Query options for the authenticated user. */
export const authQueryOptions = (): UndefinedInitialDataOptions<
  AuthUser | null,
  Error,
  AuthUser | null,
  string[]
> =>
  queryOptions({
    queryKey: ['auth', 'me'],
    queryFn: fetchMe,
    staleTime: 1000 * 30,
    retry: false,
  });

/**
 * Checks whether the application needs initial setup (no users exist).
 *
 * @returns Setup status
 */
export async function fetchSetupStatus(): Promise<SetupStatus> {
  const res: Response = await fetch('/v1/auth/setup-status');
  if (!res.ok) throw new Error(`GET /v1/auth/setup-status failed: ${res.status}`);
  return (await res.json()) as SetupStatus;
}

export async function fetchDiscordStatus(): Promise<boolean> {
  const res: Response = await fetch('/v1/auth/discord-status');
  if (!res.ok) throw new Error(`GET /v1/auth/discord-status failed: ${res.status}`);
  return (await res.json())?.configured as boolean ?? false;
}

/**
 * Registers the master account. Only works when no users exist AND the
 * caller provides the setup code printed to the server console on first
 * startup.
 *
 * @param username - Desired username
 * @param password - Plaintext password (min 8 chars)
 * @param code - Formatted setup token ("xxxx-xxxx") from the server console
 * @returns Created user profile
 * @throws Error if registration fails (invalid code, setup already completed, etc.)
 */
export async function register(
  username: string,
  password: string,
  code: string,
): Promise<AuthUser> {
  const res: Response = await fetch('/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password, code }),
  });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Registration failed: ${res.status}`);
  }
  return (await res.json()) as AuthUser;
}

/**
 * Logs in with username and password.
 *
 * @param username - Account username
 * @param password - Plaintext password
 * @returns Authenticated user profile
 * @throws Error if credentials are invalid
 */
export async function login(username: string, password: string): Promise<AuthUser> {
  const res: Response = await fetch('/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Login failed: ${res.status}`);
  }
  return (await res.json()) as AuthUser;
}

/**
 * Saves Discord OAuth credentials. The backend validates them against
 * the Discord API before persisting, so this will throw on invalid credentials.
 *
 * @param clientId - Discord application client ID
 * @param clientSecret - Discord application client secret
 * @throws Error if credentials are invalid or saving fails
 */
export async function SaveDiscordAuthentication(clientId: string, clientSecret: string): Promise<void> {
  const res: Response = await fetch('/v1/auth/master/savediscord', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ clientId, clientSecret}),
  });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Saving Failed: ${res.status}`);
  }
}

/**
 * Fetches the currently configured Discord OAuth credentials.
 * Returns null if the user is not authenticated (401).
 *
 * @returns Discord credentials or null
 */
export async function GetDiscordAuthentication(): Promise<DiscordAuthentication | null> {
  const res: Response = await fetch('/v1/auth/master/getdiscord');
  if (res.status === 401) return null;
  if (!res.ok) throw new Error(`GET /v1/auth/master/getdiscord failed: ${res.status}`);
  return (await res.json()) as DiscordAuthentication;
}

/**
 * Logs out the current session.
 *
 * @throws Error if logout fails
 */
export async function logout(): Promise<void> {
  const res: Response = await fetch('/v1/auth/logout', { method: 'POST' });
  if (!res.ok && res.status !== 204) {
    throw new Error(`Logout failed: ${res.status}`);
  }
}

/**
 * Fetches all active sessions for the current user.
 *
 * @returns List of active sessions
 */
export async function fetchSessions(): Promise<SessionEntry[]> {
  const res: Response = await fetch('/v1/auth/sessions');
  if (!res.ok) throw new Error(`GET /v1/auth/sessions failed: ${res.status}`);
  return (await res.json()) as SessionEntry[];
}

/**
 * Revokes a specific session by its ID.
 *
 * @param sessionId - Database ID of the session to revoke
 */
export async function revokeSession(sessionId: number): Promise<void> {
  const res: Response = await fetch(`/v1/auth/sessions/${sessionId}`, {
    method: 'DELETE',
  });
  if (!res.ok && res.status !== 204) {
    throw new Error(`Revoke session failed: ${res.status}`);
  }
}
