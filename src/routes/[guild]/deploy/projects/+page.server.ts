import { eq, and, inArray, desc } from 'drizzle-orm';
import type { PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { app, build, deployment, domain, host, appMetricSample } from '$lib/server/db/schema';
import { formatRelativeTime } from '$lib/utils';
import type { App as MockApp, AppStatus } from '$lib/data/bakery';

/**
 * Real apps for this guild, shaped to match `$lib/data/bakery`'s mock `App`
 * interface so the existing template (AppIcon/StatusDot/statusMeta, the
 * whole grid layout) renders them with zero changes — same "map real rows
 * into the mock shape" pattern `[appId]/+page.server.ts`'s `realAppDisplay`
 * already uses for a single app, applied here across the guild's full list.
 *
 * This page previously rendered `GUILD_RESOURCES` mock data only — every
 * other real-data page (overview, hosts, the app detail page itself) was
 * wired up in its own phase task, but this specific list view never was,
 * so a genuinely deployed app (including Bakery's own self-hosted
 * deployment, Phase 08) had no way to be discovered through it even though
 * its detail page worked correctly.
 */
export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_apps' });

	const appRows = await db.select().from(app).where(eq(app.organizationId, organization.id));
	if (appRows.length === 0) {
		return { realApps: [] as MockApp[] };
	}
	const appIds = appRows.map((a) => a.id);

	const [hostRows, domainRows, buildRows, runningDeployments, metricRows] = await Promise.all([
		db.select().from(host).where(eq(host.organizationId, organization.id)),
		db
			.select()
			.from(domain)
			.where(inArray(domain.appId, appIds))
			.orderBy(desc(domain.isDefaultSubdomain), domain.createdAt),
		db.select().from(build).where(inArray(build.appId, appIds)).orderBy(desc(build.startedAt)),
		db
			.select({ appId: deployment.appId })
			.from(deployment)
			.where(and(inArray(deployment.appId, appIds), eq(deployment.status, 'running'))),
		db
			.select()
			.from(appMetricSample)
			.where(inArray(appMetricSample.appId, appIds))
			.orderBy(desc(appMetricSample.ts))
	]);

	const hostNameById = new Map(hostRows.map((h) => [h.id, h.name]));
	const runningAppIds = new Set(runningDeployments.map((d) => d.appId));

	// First occurrence per appId wins for both — rows are already ordered
	// newest-first, same "fetch then take the first per group in JS" style
	// the overview dashboard and the app detail page both already use
	// rather than a per-group SQL query.
	const defaultDomainByAppId = new Map<string, string>();
	for (const d of domainRows) {
		if (!defaultDomainByAppId.has(d.appId)) defaultDomainByAppId.set(d.appId, d.hostname);
	}
	const latestBuildByAppId = new Map<string, (typeof buildRows)[number]>();
	for (const b of buildRows) {
		if (!latestBuildByAppId.has(b.appId)) latestBuildByAppId.set(b.appId, b);
	}
	const latestMetricByAppId = new Map<string, (typeof metricRows)[number]>();
	for (const m of metricRows) {
		if (!latestMetricByAppId.has(m.appId)) latestMetricByAppId.set(m.appId, m);
	}

	const realApps: MockApp[] = appRows.map((appRow) => {
		const latestBuild = latestBuildByAppId.get(appRow.id);
		const latestMetric = latestMetricByAppId.get(appRow.id);

		let status: AppStatus = 'stopped';
		if (runningAppIds.has(appRow.id)) status = 'running';
		else if (latestBuild?.status === 'building' || latestBuild?.status === 'queued')
			status = 'building';
		else if (latestBuild?.status === 'failed') status = 'failed';

		return {
			id: appRow.id,
			name: appRow.name,
			type: appRow.repoId ? 'app' : 'app · no repo',
			status,
			host: (appRow.hostId && hostNameById.get(appRow.hostId)) || '— unassigned —',
			domain: defaultDomainByAppId.get(appRow.id) ?? '— internal —',
			cpu: latestMetric?.cpuPct ?? 0,
			mem: latestMetric?.memBytes ? `${Math.round(latestMetric.memBytes / 1024 / 1024)} MB` : '—',
			deployed: latestBuild?.finishedAt
				? formatRelativeTime(latestBuild.finishedAt)
				: latestBuild
					? 'building'
					: 'never built',
			commitSha: latestBuild?.commitSha,
			branch: latestBuild?.branch,
			port: '—',
			initial: appRow.name.charAt(0).toUpperCase() || 'A',
			unit: `${appRow.name}.container`,
			quadletPath: `deploy/${appRow.name}.container`
		};
	});

	return { realApps };
};
