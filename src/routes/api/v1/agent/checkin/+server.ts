import { json, error } from '@sveltejs/kit';
import { eq, and } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import {
	host,
	hostMetricSample,
	hostCommand,
	volume,
	app,
	appMetricSample
} from '$lib/server/db/schema';
import { authenticateHost } from '$lib/server/agent/auth';
import { podmanVolumeName } from '$lib/server/deploy/quadlet';
import { runningUnitName } from '$lib/server/deploy/proxy';
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

	const {
		cpuPct,
		memPct,
		diskPct,
		podmanVersion,
		containerCount,
		agentVersion,
		volumes,
		containers
	} = parsed.data;

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

	// The agent reports real Podman volume names (task 07), not the
	// user-declared logical ones — recompute the expected name per row
	// (rather than storing it) to find which report belongs to which
	// `volume` row, same pattern the rest of the codebase uses for
	// derived-not-stored names (`guildNetworkName`).
	if (volumes.length > 0) {
		const sizeByName = new Map(volumes.map((v) => [v.name, v.sizeBytes]));
		const volumeRows = await db.select().from(volume).where(eq(volume.hostId, matchedHost.id));

		for (const row of volumeRows) {
			const reportedSize = sizeByName.get(podmanVolumeName(row.appId, row.name));
			if (reportedSize === undefined) continue;

			await db
				.update(volume)
				.set({ sizeBytes: reportedSize, lastReportedAt: new Date() })
				.where(eq(volume.id, row.id));
		}
	}

	// Reported containers are keyed by Quadlet unit name, not app id (task
	// 01) — recompute each app's currently expected running unit name and
	// match against it, same pattern as the volume matching above. An app
	// with nothing in `containers` (stopped/removed) simply gets no new
	// sample, no error.
	if (containers.length > 0) {
		const statByUnitName = new Map(containers.map((c) => [c.unitName, c]));
		const appRows = await db.select().from(app).where(eq(app.hostId, matchedHost.id));

		for (const appRow of appRows) {
			const unitName = await runningUnitName(appRow.id, appRow.name);
			if (!unitName) continue;

			const stat = statByUnitName.get(unitName);
			if (!stat) continue;

			await db.insert(appMetricSample).values({
				appId: appRow.id,
				hostId: matchedHost.id,
				cpuPct: stat.cpuPct,
				memBytes: stat.memBytes
			});
		}
	}

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
