import { eq } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { app, host } from '$lib/server/db/schema';
import { guildNetworkName } from '$lib/server/deploy/quadlet';

/**
 * One managed bridge network per host (task 05) — there's nothing to list
 * or create, so this is a plain derived view: every host in the guild that
 * has at least one app assigned to it has exactly one network, named and
 * scoped by organization id. A host with no apps assigned yet has no
 * network on it either (the agent only creates one on first deploy), so
 * it's simply left out rather than shown as an empty row.
 */
export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const hostRows = await db.select().from(host).where(eq(host.organizationId, organization.id));
	const appRows = await db.select().from(app).where(eq(app.organizationId, organization.id));

	const networkName = guildNetworkName(organization.id);
	const networks = hostRows
		.map((hostRow) => ({
			hostId: hostRow.id,
			hostName: hostRow.name,
			name: networkName,
			apps: appRows.filter((a) => a.hostId === hostRow.id).map((a) => a.name)
		}))
		.filter((n) => n.apps.length > 0);

	return { networks };
};
