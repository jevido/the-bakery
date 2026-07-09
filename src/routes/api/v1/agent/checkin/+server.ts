import { json, error } from '@sveltejs/kit';
import { eq, and } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { host, hostMetricSample, hostCommand } from '$lib/server/db/schema';
import { authenticateHost } from '$lib/server/agent/auth';
import {
	checkinPayloadSchema,
	type CheckinResponse,
	type PendingCommand
} from '$lib/server/agent/protocol';

export const POST: RequestHandler = async (event) => {
	const matchedHost = await authenticateHost(event.request);

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

	// Atomically claim every still-pending command for this host in one
	// UPDATE ... RETURNING — avoids a second check-in racing this one from
	// delivering (and the agent double-executing) the same command.
	const delivered = await db
		.update(hostCommand)
		.set({ status: 'delivered', deliveredAt: new Date() })
		.where(and(eq(hostCommand.hostId, matchedHost.id), eq(hostCommand.status, 'pending')))
		.returning();

	const pendingCommands: PendingCommand[] = delivered.map((c) => ({
		id: c.id,
		type: c.type,
		payload: c.payload
	}));

	return json({ pendingCommands } satisfies CheckinResponse);
};
