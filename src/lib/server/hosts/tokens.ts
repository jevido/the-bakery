import { randomBytes, createHash } from 'node:crypto';

const TOKEN_PREFIX = 'bkry_host_';

/**
 * Opaque per-host bearer token. Not tied to a user password, so a fast hash
 * (SHA-256) is appropriate for storage — see hashToken.
 */
export function generateHostToken(): string {
	return TOKEN_PREFIX + randomBytes(32).toString('base64url');
}

/**
 * One-way, deterministic digest for storage. Tokens are high-entropy random
 * values (not user-chosen secrets), so SHA-256 is sufficient — no need for a
 * slow password hash like bcrypt/argon2.
 */
export function hashToken(token: string): string {
	return createHash('sha256').update(token).digest('hex');
}

export function lastFour(token: string): string {
	return token.slice(-4);
}

export interface IssuedHostToken {
	plaintext: string;
	hash: string;
	lastFour: string;
}

/**
 * Returns the plaintext exactly once — callers must show it to the user
 * immediately and persist only `hash`/`lastFour`.
 */
export function issueHostToken(): IssuedHostToken {
	const plaintext = generateHostToken();
	return {
		plaintext,
		hash: hashToken(plaintext),
		lastFour: lastFour(plaintext)
	};
}
