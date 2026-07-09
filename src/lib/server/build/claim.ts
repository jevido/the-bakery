import { sql } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { build } from '$lib/server/db/schema';

export type Build = typeof build.$inferSelect;

/**
 * Atomically claims the oldest `queued` build for this worker to process.
 *
 * A single `UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED)` runs
 * as one implicitly-committed statement, so the row lock is only held for
 * the instant of the update rather than for the whole build — unlike
 * wrapping the claim + build in one long transaction, which would hold a
 * lock (and an idle-in-transaction connection) for as long as the build
 * takes. `SKIP LOCKED` is what lets two worker instances poll concurrently
 * without either blocking on, or double-claiming, the same row.
 */
export async function claimNextBuild(): Promise<Build | null> {
	// The WHERE clause's subquery is raw SQL (drizzle's query builder can't
	// express FOR UPDATE SKIP LOCKED on a correlated subquery), but the
	// UPDATE/RETURNING goes through the normal query builder so results get
	// drizzle's usual snake_case -> camelCase column mapping — unlike
	// db.execute(), which returns raw driver rows keyed by SQL column name.
	const [claimed] = await db
		.update(build)
		.set({ status: 'building', startedAt: new Date() })
		.where(
			sql`
			${build.id} = (
				SELECT id FROM ${build}
				WHERE status = 'queued'
				ORDER BY started_at NULLS FIRST, id
				FOR UPDATE SKIP LOCKED
				LIMIT 1
			)
		`
		)
		.returning();
	return claimed ?? null;
}
