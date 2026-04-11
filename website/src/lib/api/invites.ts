/**
 * Invite API client.
 *
 * Provides typed fetch wrappers for /v1/invites endpoints.
 */

import type { AuthUser } from './auth';

/** Validation result for an invite token. */
export interface InviteValidation {
  valid: boolean;
  expiresAt?: string;
}

/** Validates an invite token without consuming it. */
export async function validateInvite(token: string): Promise<InviteValidation> {
  const res: Response = await fetch(`/v1/invites/${encodeURIComponent(token)}/validate`);
  if (!res.ok) return { valid: false };
  return (await res.json()) as InviteValidation;
}

/** Accepts an invite with username and password, creating a new account. */
export async function acceptInvite(
  token: string,
  username: string,
  password: string,
): Promise<AuthUser> {
  const res: Response = await fetch(`/v1/invites/${encodeURIComponent(token)}/accept`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!res.ok) {
    const body: { error: string } = (await res.json()) as { error: string };
    throw new Error(body.error ?? `Registration failed: ${res.status}`);
  }
  return (await res.json()) as AuthUser;
}
