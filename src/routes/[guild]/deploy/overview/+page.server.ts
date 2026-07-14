import { and, asc, desc, eq, gte, inArray, isNull } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample } from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import { downsampleToMinuteBuckets } from '$lib/server/hosts/metrics';
import {
	HOST_METRIC_RANGES,
	DEFAULT_HOST_METRIC_RANGE,
	isHostMetricRangeId
} from '$lib/hosts/metric-ranges';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const rangeParam = event.url.searchParams.get('range');
	const range = isHostMetricRangeId(rangeParam) ? rangeParam : DEFAULT_HOST_METRIC_RANGE;
	const windowMs = HOST_METRIC_RANGES.find((r) => r.id === range)!.windowMs;

	const hostRows = await db
		.select()
		.from(host)
		.where(and(eq(host.organizationId, organization.id), isNull(host.revokedAt)));

	const hosts = hostRows.map((h) => ({ ...h, computedStatus: computeHostStatus(h) }));
	const primaryHost = hosts[0] ?? null;

	const primaryHostHistoryRaw = primaryHost
		? await db
				.select()
				.from(hostMetricSample)
				.where(
					and(
						eq(hostMetricSample.hostId, primaryHost.id),
						gte(hostMetricSample.ts, new Date(Date.now() - windowMs))
					)
				)
				.orderBy(asc(hostMetricSample.ts))
		: [];
	const primaryHostHistory = downsampleToMinuteBuckets(primaryHostHistoryRaw);

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
		primaryHostHistory,
		range
	};
};
