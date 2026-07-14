import { desc, eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { deployment, app, member, user, host } from '$lib/server/db/schema';

export interface ActivityEvent {
	dot: string;
	who: string;
	text: string;
	at: Date;
}

// How many rows to pull per source table before merging/sorting — not a
// hard cap on what's shown (that's MAX_EVENTS), just enough of a window per
// source that the true N-most-recent-overall aren't missed because one
// source (e.g. deployments) is much chattier than another (e.g. hosts).
const EVENTS_PER_SOURCE = 8;
const MAX_EVENTS = 8;

const DEPLOYMENT_STATUS_DOT: Record<string, string> = {
	running: '#34d399',
	failed: '#f0836b',
	rolled_back: '#e0a83e'
};

/**
 * Recent-events feed for the overview dashboard (task 06), assembled from
 * existing tables rather than a dedicated activity-log table (the real
 * audit log is a deferred, heavier-weight backlog item — this is
 * explicitly not that). Covers deployment status changes, member joins,
 * and host add/revoke — the three sources named in the task.
 */
export async function recentActivity(organizationId: string): Promise<ActivityEvent[]> {
	const [deployments, allMembers, hosts] = await Promise.all([
		db
			.select({
				status: deployment.status,
				startedAt: deployment.startedAt,
				triggeredBy: deployment.triggeredBy,
				appName: app.name
			})
			.from(deployment)
			.innerJoin(app, eq(app.id, deployment.appId))
			.where(eq(app.organizationId, organizationId))
			.orderBy(desc(deployment.startedAt))
			.limit(EVENTS_PER_SOURCE),
		// Fetched in full (not limited) so a deployment triggered by a
		// long-standing member can still resolve a display name below, even
		// though only the EVENTS_PER_SOURCE most recent joins become their
		// own feed entries.
		db
			.select({ userId: member.userId, userName: user.name, createdAt: member.createdAt })
			.from(member)
			.innerJoin(user, eq(user.id, member.userId))
			.where(eq(member.organizationId, organizationId))
			.orderBy(desc(member.createdAt)),
		db
			.select({ name: host.name, createdAt: host.createdAt, revokedAt: host.revokedAt })
			.from(host)
			.where(eq(host.organizationId, organizationId))
			.orderBy(desc(host.createdAt))
			.limit(EVENTS_PER_SOURCE)
	]);

	const memberNameById = new Map(allMembers.map((m) => [m.userId, m.userName]));

	const events: ActivityEvent[] = [];

	for (const d of deployments) {
		const who =
			d.triggeredBy === 'webhook'
				? 'webhook'
				: (memberNameById.get(d.triggeredBy ?? '') ?? 'a guild member');
		events.push({
			dot: DEPLOYMENT_STATUS_DOT[d.status] ?? '#3fb984',
			who,
			text: ` deployed ${d.appName} (${d.status})`,
			at: d.startedAt
		});
	}

	for (const m of allMembers.slice(0, EVENTS_PER_SOURCE)) {
		events.push({ dot: '#34d399', who: m.userName, text: ' joined the guild', at: m.createdAt });
	}

	for (const h of hosts) {
		events.push({ dot: '#3fb984', who: 'system', text: ` added host ${h.name}`, at: h.createdAt });
		if (h.revokedAt) {
			events.push({
				dot: '#f0836b',
				who: 'system',
				text: ` revoked host ${h.name}`,
				at: h.revokedAt
			});
		}
	}

	return events.sort((a, b) => b.at.getTime() - a.at.getTime()).slice(0, MAX_EVENTS);
}
