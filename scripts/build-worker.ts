/**
 * Long-lived worker process for the central build service (as opposed to
 * spawning one process per build): a single warm process means no repeated
 * Node/DB-connection startup cost per build, and a plain polling loop keeps
 * the claim query (see claimNextBuild) as the only coordination point, so
 * running N instances of this script is just "more capacity" with no other
 * setup. Run with `bun scripts/build-worker.ts`.
 */
import { claimNextBuild } from '$lib/server/build/claim';
import { processBuild } from '$lib/server/build/run-build';

const POLL_INTERVAL_MS = 2000;

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

async function main() {
	console.log('[build-worker] started, polling for queued builds...');
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
