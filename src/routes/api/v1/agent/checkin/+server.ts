import { json, error } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { host, hostMetricSample } from '$lib/server/db/schema';
import { hashToken } from '$lib/server/hosts/tokens';
import { checkinPayloadSchema, type CheckinResponse } from '$lib/server/agent/protocol';

export const POST: RequestHandler = async (event) => {
	const token = event.request.headers.get('authorization')?.match(/^Bearer\s+(.+)$/i)?.[1];
	if (!token) {
		error(401, 'Missing bearer token');
	}

	const [matchedHost] = await db
		.select()
		.from(host)
		.where(eq(host.tokenHash, hashToken(token)));

	if (!matchedHost || matchedHost.revokedAt) {
		error(401, 'Invalid or revoked token');
	}

	const body = await event.request.json().catch(() => null);
	const parsed = checkinPayloadSchema.safeParse(body);
	if (!parsed.success) {
		error(400, 'Invalid check-in payload');
	}

	const { cpuPct, memPct, diskPct, podmanVersion, containerCount, agentVersion } = parsed.data;

	await db.insert(hostMetricSample).values({
		hostId: matchedHost.id,
		cpuPct,
		memPct,
		diskPct,
		podmanVersion,
		containerCount
	});

	await db
		.update(host)
		.set({ lastSeenAt: new Date(), status: 'online', agentVersion })
		.where(eq(host.id, matchedHost.id));

	return json({ pendingCommands: [] } satisfies CheckinResponse);
};
