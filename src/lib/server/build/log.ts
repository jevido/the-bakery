import { db } from '$lib/server/db';
import { buildLogLine } from '$lib/server/db/schema';

/**
 * Inserted per line rather than batched so the SSE stream (task 09) can pick
 * up new rows as the build progresses instead of only after it finishes.
 */
export async function appendBuildLog(buildId: string, line: string): Promise<void> {
	await db.insert(buildLogLine).values({ buildId, line });
}
