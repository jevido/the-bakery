import { and, asc, desc, eq, gte, inArray, isNull } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample, app, volume, deployment, build } from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import {
	downsampleToMinuteBuckets,
	aggregateAcrossHosts,
	average
} from '$lib/server/hosts/metrics';
import { recentActivity } from '$lib/server/overview/activity';
import { DEFAULT_METRIC_RANGE, isMetricRangeId, rangeStartDate } from '$lib/hosts/metric-ranges';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const rangeParam = event.url.searchParams.get('range');
	const range = isMetricRangeId(rangeParam) ? rangeParam : DEFAULT_METRIC_RANGE;
	const rangeStart = rangeStartDate(range);

	// Explicit column list, not `select()` — `tokenHash` must never reach the
	// client (Phase 07 task 05's encryption-at-rest review): `hosts` below is
	// returned straight through to the page, so an unqualified `select()`
	// would ship it in every load.
	const hostRows = await db
		.select({
			id: host.id,
			organizationId: host.organizationId,
			name: host.name,
			location: host.location,
			spec: host.spec,
			tokenLastFour: host.tokenLastFour,
			status: host.status,
			agentVersion: host.agentVersion,
			createdAt: host.createdAt,
			lastSeenAt: host.lastSeenAt,
			revokedAt: host.revokedAt
		})
		.from(host)
		.where(and(eq(host.organizationId, organization.id), isNull(host.revokedAt)));

	const hosts = hostRows.map((h) => ({ ...h, computedStatus: computeHostStatus(h) }));
	const hostIds = hosts.map((h) => h.id);

	// `host=all` (default) or a specific host id from this org (task 10's
	// switcher) — replaces the old hardcoded `primaryHost = hosts[0]`.
	const hostParam = event.url.searchParams.get('host');
	const selectedHostId = hostParam && hosts.some((h) => h.id === hostParam) ? hostParam : 'all';

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

	interface DisplaySample {
		ts: Date;
		cpuPct: number | null;
		memPct: number | null;
		netKBs: number | null;
	}
	let displayHistory: DisplaySample[];
	let displayLatest: DisplaySample | null;

	if (selectedHostId === 'all') {
		const allHistoryRaw = hostIds.length
			? await db
					.select()
					.from(hostMetricSample)
					.where(
						and(inArray(hostMetricSample.hostId, hostIds), gte(hostMetricSample.ts, rangeStart))
					)
					.orderBy(asc(hostMetricSample.ts))
			: [];
		displayHistory = aggregateAcrossHosts(allHistoryRaw);

		const latestPerHost = hostIds
			.map((id) => latestSampleByHost.get(id))
			.filter((s): s is NonNullable<typeof s> => s != null);
		displayLatest = latestPerHost.length
			? {
					ts: new Date(),
					cpuPct: average(latestPerHost.map((s) => s.cpuPct)),
					memPct: average(latestPerHost.map((s) => s.memPct)),
					netKBs: latestPerHost.reduce(
						(sum, s) => sum + ((s.netRxBytesPerSec ?? 0) + (s.netTxBytesPerSec ?? 0)) / 1024,
						0
					)
				}
			: null;
	} else {
		const hostHistoryRaw = await db
			.select()
			.from(hostMetricSample)
			.where(and(eq(hostMetricSample.hostId, selectedHostId), gte(hostMetricSample.ts, rangeStart)))
			.orderBy(asc(hostMetricSample.ts));
		displayHistory = downsampleToMinuteBuckets(hostHistoryRaw).map((s) => ({
			ts: s.ts,
			cpuPct: s.cpuPct,
			memPct: s.memPct,
			netKBs:
				s.netRxBytesPerSec != null || s.netTxBytesPerSec != null
					? ((s.netRxBytesPerSec ?? 0) + (s.netTxBytesPerSec ?? 0)) / 1024
					: null
		}));

		const latest = latestSampleByHost.get(selectedHostId) ?? null;
		displayLatest = latest
			? {
					ts: latest.ts,
					cpuPct: latest.cpuPct,
					memPct: latest.memPct,
					netKBs: ((latest.netRxBytesPerSec ?? 0) + (latest.netTxBytesPerSec ?? 0)) / 1024
				}
			: null;
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
		selectedHostId,
		displayHistory,
		displayLatest,
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
