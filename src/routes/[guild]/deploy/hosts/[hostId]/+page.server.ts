import { error } from '@sveltejs/kit';
import { and, asc, desc, eq, gte } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample, app } from '$lib/server/db/schema';
import { computeHostStatus } from '$lib/server/hosts/status';
import { classifyHostHealth } from '$lib/server/hosts/health';
import { downsampleToMinuteBuckets } from '$lib/server/hosts/metrics';
import { DEFAULT_METRIC_RANGE, isMetricRangeId, rangeStartDate } from '$lib/hosts/metric-ranges';

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
		db.select({ id: app.id }).from(app).where(eq(app.hostId, hostRow.id))
	]);

	return {
		host: {
			...hostRow,
			computedStatus: computeHostStatus(hostRow),
			health: classifyHostHealth(hostRow)
		},
		range,
		history: downsampleToMinuteBuckets(historyRaw),
		latestSample: latestSampleRows[0] ?? null,
		appsCount: appsOnHost.length
	};
};
