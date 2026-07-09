import { and, asc, desc, eq, gte, inArray, isNull } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample } from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';

const HISTORY_WINDOW_MS = 60 * 60 * 1000;

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const hostRows = await db
		.select()
		.from(host)
		.where(and(eq(host.organizationId, organization.id), isNull(host.revokedAt)));

	const hosts = hostRows.map((h) => ({ ...h, computedStatus: computeHostStatus(h) }));
	const primaryHost = hosts[0] ?? null;

	const primaryHostHistory = primaryHost
		? await db
				.select()
				.from(hostMetricSample)
				.where(
					and(
						eq(hostMetricSample.hostId, primaryHost.id),
						gte(hostMetricSample.ts, new Date(Date.now() - HISTORY_WINDOW_MS))
					)
				)
				.orderBy(asc(hostMetricSample.ts))
		: [];

	const hostIds = hosts.map((h) => h.id);
	const latestSamples = hostIds.length
		? await db
				.select()
				.from(hostMetricSample)
				.where(inArray(hostMetricSample.hostId, hostIds))
				.orderBy(desc(hostMetricSample.ts))
		: [];
	const latestSampleByHost = new Map<string, (typeof latestSamples)[number]>();
	for (const sample of latestSamples) {
		if (!latestSampleByHost.has(sample.hostId)) latestSampleByHost.set(sample.hostId, sample);
	}

	return {
		hosts,
		primaryHost,
		primaryHostLatestSample: primaryHost ? (latestSampleByHost.get(primaryHost.id) ?? null) : null,
		primaryHostHistory
	};
};
