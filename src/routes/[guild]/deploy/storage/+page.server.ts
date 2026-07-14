import { eq, desc } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { app, host, volume } from '$lib/server/db/schema';

/**
 * Real, host-backed Podman volumes (task 07) — unlike the mock this
 * replaces, there's nothing to create from this page; volumes are declared
 * per-app from the app detail page's Storage tab (mirroring how domains are
 * managed from the app page, not a guild-wide one). This is a read-only
 * rollup across every app in the guild, same pattern the Networks page
 * (task 06) established.
 */
export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const rows = await db
		.select({
			id: volume.id,
			name: volume.name,
			mountPath: volume.mountPath,
			sizeBytes: volume.sizeBytes,
			lastReportedAt: volume.lastReportedAt,
			appId: volume.appId,
			appName: app.name,
			hostName: host.name
		})
		.from(volume)
		.innerJoin(app, eq(volume.appId, app.id))
		.innerJoin(host, eq(volume.hostId, host.id))
		.where(eq(app.organizationId, organization.id))
		.orderBy(desc(volume.createdAt));

	return { volumes: rows };
};
