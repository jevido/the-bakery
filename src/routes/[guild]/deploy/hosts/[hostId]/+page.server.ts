import { error } from '@sveltejs/kit';
import { and, asc, desc, eq, gte, inArray } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import {
	host,
	hostMetricSample,
	app,
	deployment,
	build,
	appMetricSample,
	hostCommand
} from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import { classifyHostHealth } from '$lib/server/hosts/health';
import { downsampleToMinuteBuckets } from '$lib/server/hosts/metrics';
import { DEFAULT_METRIC_RANGE, isMetricRangeId, rangeStartDate } from '$lib/hosts/metric-ranges';
import { versionedUnitName, guildNetworkName } from '$lib/server/deploy/quadlet';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	// Explicit column list, not `select()` — `tokenHash` must never reach the
	// client (Phase 07 task 05's encryption-at-rest review), same convention
	// as the Hosts list page.
	const [hostRow] = await db
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
		.where(and(eq(host.id, event.params.hostId), eq(host.organizationId, organization.id)));
	if (!hostRow) error(404, 'Host not found');

	const rangeParam = event.url.searchParams.get('range');
	const range = isMetricRangeId(rangeParam) ? rangeParam : DEFAULT_METRIC_RANGE;

	const [historyRaw, latestSampleRows, appsOnHost] = await Promise.all([
		db
			.select()
			.from(hostMetricSample)
			.where(
				and(
					eq(hostMetricSample.hostId, hostRow.id),
					gte(hostMetricSample.ts, rangeStartDate(range))
				)
			)
			.orderBy(asc(hostMetricSample.ts)),
		// Fetched independently of the selected range so the identity header
		// (podman/agent version, uptime, last-seen) always reflects the real
		// latest check-in, not whatever the chart range happens to cover.
		db
			.select()
			.from(hostMetricSample)
			.where(eq(hostMetricSample.hostId, hostRow.id))
			.orderBy(desc(hostMetricSample.ts))
			.limit(1),
		db.select({ id: app.id, name: app.name }).from(app).where(eq(app.hostId, hostRow.id))
	]);

	const appIds = appsOnHost.map((a) => a.id);

	// A "container" here mirrors the App detail page's own Containers tab
	// model (Phase 11): exactly one container per app, which exists only
	// while that app has a `running` deployment — not literal agent-reported
	// container data at load time.
	const runningDeployments = appIds.length
		? await db
				.select({ appId: deployment.appId, commitSha: build.commitSha, imageRef: build.imageRef })
				.from(deployment)
				.innerJoin(build, eq(build.id, deployment.buildId))
				.where(and(inArray(deployment.appId, appIds), eq(deployment.status, 'running')))
		: [];
	const runningByAppId = new Map(runningDeployments.map((d) => [d.appId, d]));

	const latestMetricRows = appIds.length
		? await db
				.select()
				.from(appMetricSample)
				.where(inArray(appMetricSample.appId, appIds))
				.orderBy(desc(appMetricSample.ts))
		: [];
	const latestMetricByAppId = new Map<string, (typeof latestMetricRows)[number]>();
	for (const s of latestMetricRows) {
		if (!latestMetricByAppId.has(s.appId)) latestMetricByAppId.set(s.appId, s);
	}

	const containers = appsOnHost
		.map((a) => {
			const running = runningByAppId.get(a.id);
			const metric = latestMetricByAppId.get(a.id) ?? null;
			return {
				appId: a.id,
				name: a.name,
				status: (running ? 'running' : 'stopped') as 'running' | 'stopped',
				image: running?.imageRef ?? '—',
				unit: running ? versionedUnitName(a.name, running.commitSha) : null,
				nets: running ? guildNetworkName(organization.id) : '—',
				cpuPct: metric?.cpuPct ?? null,
				memBytes: metric?.memBytes ?? null
			};
		})
		// Heaviest first, matching the same "top containers" intent task 13
		// applies org-wide -- this is the per-host slice of that view.
		.sort((a, b) => (b.cpuPct ?? -1) - (a.cpuPct ?? -1));

	// Activity feed: hostCommand dispatches (deploy/stop/restart/
	// configureProxy) and the deployments that ran here, merged into one
	// chronological list. `build` has no hostId of its own -- a build
	// compiles centrally, not on a specific host -- so "builds that ran
	// here" is represented via each deployment's own build (commit/branch),
	// not as a separate event kind.
	const ACTIVITY_LIMIT = 75;
	const [hostCommandRows, deploymentRows] = await Promise.all([
		db
			.select({
				id: hostCommand.id,
				type: hostCommand.type,
				status: hostCommand.status,
				createdAt: hostCommand.createdAt,
				completedAt: hostCommand.completedAt,
				errorMessage: hostCommand.errorMessage,
				appId: app.id,
				appName: app.name
			})
			.from(hostCommand)
			.innerJoin(deployment, eq(deployment.id, hostCommand.deploymentId))
			.innerJoin(app, eq(app.id, deployment.appId))
			.where(eq(hostCommand.hostId, hostRow.id))
			.orderBy(desc(hostCommand.createdAt))
			.limit(ACTIVITY_LIMIT),
		db
			.select({
				id: deployment.id,
				status: deployment.status,
				startedAt: deployment.startedAt,
				finishedAt: deployment.finishedAt,
				triggeredBy: deployment.triggeredBy,
				commitSha: build.commitSha,
				appId: app.id,
				appName: app.name
			})
			.from(deployment)
			.innerJoin(build, eq(build.id, deployment.buildId))
			.innerJoin(app, eq(app.id, deployment.appId))
			.where(eq(deployment.hostId, hostRow.id))
			.orderBy(desc(deployment.startedAt))
			.limit(ACTIVITY_LIMIT)
	]);

	const activity = [
		...hostCommandRows.map((c) => ({
			id: c.id,
			ts: c.completedAt ?? c.createdAt,
			kind: 'command' as const,
			status: c.status,
			description: `${c.type} command ${c.status} for ${c.appName}`,
			errorMessage: c.errorMessage,
			appId: c.appId
		})),
		...deploymentRows.map((d) => ({
			id: d.id,
			ts: d.finishedAt ?? d.startedAt,
			kind: 'deployment' as const,
			status: d.status,
			description: `Deployed ${d.commitSha.slice(0, 7)} to ${d.appName}${d.triggeredBy ? ` · triggered by ${d.triggeredBy}` : ''}`,
			errorMessage: null as string | null,
			appId: d.appId
		}))
	]
		.sort((a, b) => b.ts.getTime() - a.ts.getTime())
		.slice(0, ACTIVITY_LIMIT);

	return {
		host: {
			...hostRow,
			computedStatus: computeHostStatus(hostRow),
			health: classifyHostHealth(hostRow)
		},
		range,
		history: downsampleToMinuteBuckets(historyRaw),
		latestSample: latestSampleRows[0] ?? null,
		appsCount: appsOnHost.length,
		containers,
		activity
	};
};
