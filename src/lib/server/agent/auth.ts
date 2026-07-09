import { error } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { host } from '$lib/server/db/schema';
import { hashToken } from '$lib/server/hosts/tokens';

/** Bearer-token host auth shared by every agent-facing endpoint (check-in, command completion, ...). */
export async function authenticateHost(request: Request) {
	const token = request.headers.get('authorization')?.match(/^Bearer\s+(.+)$/i)?.[1];
	if (!token) error(401, 'Missing bearer token');

	const [matchedHost] = await db
		.select()
		.from(host)
		.where(eq(host.tokenHash, hashToken(token)));

	if (!matchedHost || matchedHost.revokedAt) error(401, 'Invalid or revoked token');

	return matchedHost;
}
