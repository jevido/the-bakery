import { and, asc, desc, eq, gte, inArray, isNull } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample, app, volume, deployment, build } from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import { downsampleToMinuteBuckets } from '$lib/server/hosts/metrics';
import { recentActivity } from '$lib/server/overview/activity';
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

	// Real summary-tile aggregates (task 06) — deployment/volume/build aren't
	// directly organization-scoped, so appIds is the join key for all three,
	// same "fetch the rows, count/sum in JS" style the rest of this
	// dashboard (and hosts/+page.server.ts) already uses rather than
	// introducing SQL aggregate functions used nowhere else in the codebase.
	const appRows = await db.select().from(app).where(eq(app.organizationId, organization.id));
	const appIds = appRows.map((a) => a.id);

	const [volumeRows, runningDeployments, activeBuilds, activityEvents] = await Promise.all([
		appIds.length
			? db.select().from(volume).where(inArray(volume.appId, appIds))
			: Promise.resolve([]),
		appIds.length
			? db
					.select({ id: deployment.id })
					.from(deployment)
					.where(and(inArray(deployment.appId, appIds), eq(deployment.status, 'running')))
			: Promise.resolve([]),
		appIds.length
			? db
					.select({ id: build.id })
					.from(build)
					.where(and(inArray(build.appId, appIds), inArray(build.status, ['queued', 'building'])))
			: Promise.resolve([]),
		recentActivity(organization.id)
	]);

	return {
		hosts,
		primaryHost,
		primaryHostLatestSample: primaryHost ? (latestSampleByHost.get(primaryHost.id) ?? null) : null,
		primaryHostHistory,
		range,
		appsCount: appRows.length,
		hostsOnlineCount: hosts.filter((h) => h.computedStatus === 'online').length,
		hostsTotalCount: hosts.length,
		runningDeploymentsCount: runningDeployments.length,
		activeBuildsCount: activeBuilds.length,
		volumesCount: volumeRows.length,
		volumesTotalBytes: volumeRows.reduce((sum, v) => sum + v.sizeBytes, 0),
		activityEvents
	};
};
