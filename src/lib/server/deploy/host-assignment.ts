import { and, eq, isNull } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { app, host } from '$lib/server/db/schema';

type App = typeof app.$inferSelect;
type Host = typeof host.$inferSelect;

/**
 * Returns the app's assigned host, or — since there's no host-assignment UI
 * yet (Phase 05 may add real scheduling) — falls back to the org's oldest
 * non-revoked host and persists that as the app's assignment, so the Deploy
 * button works end-to-end without a blocking prerequisite feature. Returns
 * null only when the org has no host at all.
 */
export async function resolveHostForApp(appRow: App, organizationId: string): Promise<Host | null> {
	if (appRow.hostId) {
		const [assigned] = await db.select().from(host).where(eq(host.id, appRow.hostId));
		if (assigned) return assigned;
	}

	const [firstHost] = await db
		.select()
		.from(host)
		.where(and(eq(host.organizationId, organizationId), isNull(host.revokedAt)))
		.orderBy(host.createdAt)
		.limit(1);
	if (!firstHost) return null;

	await db.update(app).set({ hostId: firstHost.id }).where(eq(app.id, appRow.id));
	return firstHost;
}
