import { desc, eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { deployment, build, hostCommand, app, host } from '$lib/server/db/schema';

export interface ClusterEvent {
	id: string;
	ts: Date;
	color: string;
	description: string;
	href: string;
}

// How many rows to pull per source table before merging/sorting — enough
// that the true N-most-recent-overall aren't missed just because one source
// happens to be chattier than another, same convention `recentActivity`
// already uses for its own (different, member/host-focused) feed.
const EVENTS_PER_SOURCE = 25;
const MAX_EVENTS = 40;

const DEPLOYMENT_COLOR: Record<string, string> = {
	running: '#52cc96',
	failed: '#f0836b',
	rolled_back: '#e0a83e',
	stopped: '#5c6170'
};
const BUILD_COLOR: Record<string, string> = {
	succeeded: '#52cc96',
	failed: '#f0836b',
	building: '#e0a83e',
	queued: '#5c6170'
};
const COMMAND_COLOR: Record<string, string> = {
	succeeded: '#52cc96',
	failed: '#f0836b',
	pending: '#e0a83e',
	delivered: '#e0a83e'
};

/**
 * Org-wide chronological feed of deployments, builds, and hostCommand
 * dispatches (Phase 20 task 12) — the k8s-"Events"-view equivalent this
 * project didn't have before. Distinct from `recentActivity` (member joins,
 * host add/revoke) — reuses the existing per-entity tables, no new unified
 * events table.
 */
export async function recentClusterEvents(
	organizationId: string,
	guildSlug: string
): Promise<ClusterEvent[]> {
	const [deployments, builds, commands] = await Promise.all([
		db
			.select({
				id: deployment.id,
				status: deployment.status,
				startedAt: deployment.startedAt,
				finishedAt: deployment.finishedAt,
				appId: app.id,
				appName: app.name,
				commitSha: build.commitSha
			})
			.from(deployment)
			.innerJoin(app, eq(app.id, deployment.appId))
			.innerJoin(build, eq(build.id, deployment.buildId))
			.where(eq(app.organizationId, organizationId))
			.orderBy(desc(deployment.startedAt))
			.limit(EVENTS_PER_SOURCE),
		db
			.select({
				id: build.id,
				status: build.status,
				startedAt: build.startedAt,
				finishedAt: build.finishedAt,
				appId: app.id,
				appName: app.name,
				commitSha: build.commitSha
			})
			.from(build)
			.innerJoin(app, eq(app.id, build.appId))
			.where(eq(app.organizationId, organizationId))
			.orderBy(desc(build.startedAt))
			.limit(EVENTS_PER_SOURCE),
		db
			.select({
				id: hostCommand.id,
				type: hostCommand.type,
				status: hostCommand.status,
				createdAt: hostCommand.createdAt,
				completedAt: hostCommand.completedAt,
				errorMessage: hostCommand.errorMessage,
				hostId: host.id,
				hostName: host.name
			})
			.from(hostCommand)
			.innerJoin(host, eq(host.id, hostCommand.hostId))
			.where(eq(host.organizationId, organizationId))
			.orderBy(desc(hostCommand.createdAt))
			.limit(EVENTS_PER_SOURCE)
	]);

	const events: ClusterEvent[] = [];

	for (const d of deployments) {
		events.push({
			id: `deployment-${d.id}`,
			ts: d.finishedAt ?? d.startedAt,
			color: DEPLOYMENT_COLOR[d.status] ?? '#e0a83e',
			description: `Deployed ${d.commitSha.slice(0, 7)} to ${d.appName} (${d.status})`,
			href: `/${guildSlug}/deploy/projects/${d.appId}?tab=deployments`
		});
	}

	for (const b of builds) {
		if (!b.startedAt) continue; // still queued -- nothing has "happened" yet
		events.push({
			id: `build-${b.id}`,
			ts: b.finishedAt ?? b.startedAt,
			color: BUILD_COLOR[b.status] ?? '#e0a83e',
			description:
				b.status === 'failed'
					? `Build failed for ${b.appName} (${b.commitSha.slice(0, 7)})`
					: `Build ${b.status} for ${b.appName} (${b.commitSha.slice(0, 7)})`,
			href: `/${guildSlug}/deploy/projects/${b.appId}?tab=deployments`
		});
	}

	for (const c of commands) {
		events.push({
			id: `command-${c.id}`,
			ts: c.completedAt ?? c.createdAt,
			color: COMMAND_COLOR[c.status] ?? '#e0a83e',
			description: c.errorMessage
				? `${c.type} command failed on ${c.hostName}: ${c.errorMessage}`
				: `${c.type} command ${c.status} on ${c.hostName}`,
			href: `/${guildSlug}/deploy/hosts/${c.hostId}?tab=activity`
		});
	}

	return events.sort((a, b) => b.ts.getTime() - a.ts.getTime()).slice(0, MAX_EVENTS);
}
