/**
 * Long-lived worker process for the central build service (as opposed to
 * spawning one process per build): a single warm process means no repeated
 * Node/DB-connection startup cost per build, and a plain polling loop keeps
 * the claim query (see claimNextBuild) as the only coordination point, so
 * running N instances of this script is just "more capacity" with no other
 * setup. Run with `bun scripts/build-worker.ts`.
 */
import { and, eq, lt } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { build } from '$lib/server/db/schema';
import { claimNextBuild } from '$lib/server/build/claim';
import { processBuild } from '$lib/server/build/run-build';
import { appendBuildLog } from '$lib/server/build/log';

const POLL_INTERVAL_MS = 2000;

// Comfortably longer than run-build.ts's own SUBPROCESS_TIMEOUT_MS (15
// minutes) so this only catches a row genuinely orphaned by a crashed
// worker process -- not a build that's merely slow and would have been
// caught by that timeout instead.
const STALE_BUILDING_THRESHOLD_MS = 30 * 60 * 1000;

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Recovers `build` rows left at `status: 'building'` by a worker process
 * that crashed mid-build (OOM, SIGKILL) rather than a subprocess hanging --
 * `processBuild`'s own `try/catch`/`finally` never runs in that case, since
 * the process itself is gone, so nothing else ever revisits the row.
 * Single check on startup is sufficient for this project's actual
 * single-worker deployment model (one supervised systemd unit).
 */
async function recoverOrphanedBuilds(): Promise<void> {
	const staleBuilds = await db
		.select()
		.from(build)
		.where(
			and(
				eq(build.status, 'building'),
				lt(build.startedAt, new Date(Date.now() - STALE_BUILDING_THRESHOLD_MS))
			)
		);

	for (const staleBuild of staleBuilds) {
		console.log(`[build-worker] recovering orphaned build ${staleBuild.id}`);
		await appendBuildLog(
			staleBuild.id,
			'Build worker restarted while this build was still in progress — marking failed. Retry via Build now.'
		);
		await db
			.update(build)
			.set({ status: 'failed', finishedAt: new Date() })
			.where(eq(build.id, staleBuild.id));
	}
}

async function main() {
	console.log('[build-worker] started, polling for queued builds...');
	await recoverOrphanedBuilds();
	while (true) {
		const claimed = await claimNextBuild().catch((err) => {
			console.error('[build-worker] failed to claim next build:', err);
			return null;
		});

		if (!claimed) {
			await sleep(POLL_INTERVAL_MS);
			continue;
		}

		console.log(`[build-worker] claimed build ${claimed.id} (${claimed.commitSha})`);
		await processBuild(claimed);
		console.log(`[build-worker] finished build ${claimed.id}`);
	}
}

main();
