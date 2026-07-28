import { and, asc, desc, eq, gte, inArray, isNull } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import {
	host,
	hostMetricSample,
	app,
	volume,
	deployment,
	build,
	appMetricSample
} from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import { classifyHostHealth } from '$lib/server/hosts/health';
import {
	downsampleToMinuteBuckets,
	aggregateAcrossHosts,
	average
} from '$lib/server/hosts/metrics';
import { recentActivity } from '$lib/server/overview/activity';
import { recentClusterEvents } from '$lib/server/overview/events';
import { DEFAULT_METRIC_RANGE, isMetricRangeId, rangeStartDate } from '$lib/hosts/metric-ranges';
import { WARNING_THRESHOLD, CRITICAL_THRESHOLD } from '$lib/data/health-thresholds';

// How far back a failed deployment/build still counts as an active issue —
// bounds the alerts panel's query so it doesn't grow unbounded over the
// org's entire history.
const ALERT_LOOKBACK_MS = 7 * 24 * 60 * 60 * 1000;
const ALERTS_LIMIT = 20;

export interface DashboardAlert {
	id: string;
	severity: 'critical' | 'warning';
	description: string;
	href: string;
	ts: Date;
}

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

	const [
		volumeRows,
		runningDeployments,
		activeBuilds,
		activityEvents,
		clusterEvents,
		appMetricRows
	] = await Promise.all([
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
		recentActivity(organization.id),
		recentClusterEvents(organization.id, event.params.guild ?? ''),
		appIds.length
			? db
					.select()
					.from(appMetricSample)
					.where(inArray(appMetricSample.appId, appIds))
					.orderBy(desc(appMetricSample.ts))
			: Promise.resolve([])
	]);

	// Capacity views (task 13) — "where is my capacity going," grouped into
	// three compact panels rather than three separate metric systems.
	const latestMetricByApp = new Map<string, (typeof appMetricRows)[number]>();
	for (const m of appMetricRows) {
		if (!latestMetricByApp.has(m.appId)) latestMetricByApp.set(m.appId, m);
	}

	const TOP_CONTAINERS_LIMIT = 8;
	const topContainers = appRows
		.map((a) => {
			const metric = latestMetricByApp.get(a.id);
			return {
				appId: a.id,
				appName: a.name,
				cpuPct: metric?.cpuPct ?? null,
				memBytes: metric?.memBytes ?? null
			};
		})
		.filter((c) => c.cpuPct != null || c.memBytes != null)
		.sort((a, b) => (b.cpuPct ?? 0) - (a.cpuPct ?? 0))
		.slice(0, TOP_CONTAINERS_LIMIT);

	const appCountByHost = new Map<string, number>();
	for (const a of appRows) {
		if (!a.hostId) continue;
		appCountByHost.set(a.hostId, (appCountByHost.get(a.hostId) ?? 0) + 1);
	}
	const hostCapacity = hosts
		.map((h) => ({ hostId: h.id, hostName: h.name, appCount: appCountByHost.get(h.id) ?? 0 }))
		.sort((a, b) => b.appCount - a.appCount);

	const storageBytesByHost = new Map<string, number>();
	for (const v of volumeRows) {
		storageBytesByHost.set(v.hostId, (storageBytesByHost.get(v.hostId) ?? 0) + v.sizeBytes);
	}
	const storageByHost = hosts
		.map((h) => ({ hostId: h.id, hostName: h.name, totalBytes: storageBytesByHost.get(h.id) ?? 0 }))
		.filter((s) => s.totalBytes > 0)
		.sort((a, b) => b.totalBytes - a.totalBytes);

	// Alerts panel (task 11) — org-wide surfacing of anything that needs
	// attention: hosts over a resource threshold, hosts that have gone
	// stale/offline, and recent failed deployments/builds. No alerting
	// existed anywhere in this codebase before this.
	const alertCutoff = new Date(Date.now() - ALERT_LOOKBACK_MS);
	const alerts: DashboardAlert[] = [];

	for (const h of hosts) {
		const sample = latestSampleByHost.get(h.id);
		const metricChecks: Array<[string, number | null | undefined]> = [
			['CPU', sample?.cpuPct],
			['memory', sample?.memPct],
			['disk', sample?.diskPct],
			['swap', sample?.swapPct]
		];
		for (const [label, value] of metricChecks) {
			if (value == null) continue;
			if (value >= WARNING_THRESHOLD) {
				alerts.push({
					id: `host-${h.id}-${label}`,
					severity: value >= CRITICAL_THRESHOLD ? 'critical' : 'warning',
					description: `${h.name}: ${label} at ${value.toFixed(0)}%`,
					href: `/${event.params.guild}/deploy/hosts/${h.id}`,
					ts: sample?.ts ?? h.createdAt
				});
			}
		}

		const health = classifyHostHealth(h);
		if (health === 'offline') {
			alerts.push({
				id: `host-${h.id}-offline`,
				severity: 'critical',
				description: `${h.name} is offline`,
				href: `/${event.params.guild}/deploy/hosts/${h.id}`,
				ts: h.lastSeenAt ?? h.createdAt
			});
		} else if (health === 'stale') {
			alerts.push({
				id: `host-${h.id}-stale`,
				severity: 'warning',
				description: `${h.name} hasn't checked in recently`,
				href: `/${event.params.guild}/deploy/hosts/${h.id}`,
				ts: h.lastSeenAt ?? h.createdAt
			});
		}
	}

	const [failedDeploymentRows, failedBuildRows] = await Promise.all([
		appIds.length
			? db
					.select({
						id: deployment.id,
						appId: app.id,
						appName: app.name,
						commitSha: build.commitSha,
						startedAt: deployment.startedAt,
						finishedAt: deployment.finishedAt
					})
					.from(deployment)
					.innerJoin(build, eq(build.id, deployment.buildId))
					.innerJoin(app, eq(app.id, deployment.appId))
					.where(
						and(
							inArray(deployment.appId, appIds),
							eq(deployment.status, 'failed'),
							gte(deployment.startedAt, alertCutoff)
						)
					)
					.orderBy(desc(deployment.startedAt))
					.limit(ALERTS_LIMIT)
			: Promise.resolve([]),
		appIds.length
			? db
					.select({
						id: build.id,
						appId: app.id,
						appName: app.name,
						commitSha: build.commitSha,
						finishedAt: build.finishedAt
					})
					.from(build)
					.innerJoin(app, eq(app.id, build.appId))
					.where(
						and(
							inArray(build.appId, appIds),
							eq(build.status, 'failed'),
							gte(build.finishedAt, alertCutoff)
						)
					)
					.orderBy(desc(build.finishedAt))
					.limit(ALERTS_LIMIT)
			: Promise.resolve([])
	]);

	for (const d of failedDeploymentRows) {
		alerts.push({
			id: `deployment-${d.id}`,
			severity: 'critical',
			description: `Deployment of ${d.commitSha.slice(0, 7)} to ${d.appName} failed`,
			href: `/${event.params.guild}/deploy/projects/${d.appId}?tab=deployments`,
			ts: d.finishedAt ?? d.startedAt
		});
	}
	for (const b of failedBuildRows) {
		alerts.push({
			id: `build-${b.id}`,
			severity: 'critical',
			description: `Build of ${b.commitSha.slice(0, 7)} for ${b.appName} failed`,
			href: `/${event.params.guild}/deploy/projects/${b.appId}?tab=deployments`,
			ts: b.finishedAt ?? alertCutoff
		});
	}

	alerts.sort((a, b) => {
		if (a.severity !== b.severity) return a.severity === 'critical' ? -1 : 1;
		return b.ts.getTime() - a.ts.getTime();
	});

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
		activityEvents,
		clusterEvents,
		alerts: alerts.slice(0, ALERTS_LIMIT),
		topContainers,
		hostCapacity,
		storageByHost
	};
};
