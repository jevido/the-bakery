import { eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { build, buildLogLine } from '$lib/server/db/schema';
import { redactSecrets, secretValuesForApp } from '$lib/server/secrets/redact';

// Cached per build (not re-queried per line) since a build streams many
// lines from its own subprocess output — cleared via
// `clearBuildLogSecretsCache` once the build finishes (`run-build.ts`'s
// `finally` block) so this doesn't grow for the life of the worker process.
const secretsCache = new Map<string, Promise<string[]>>();

async function secretsForBuild(buildId: string): Promise<string[]> {
	let cached = secretsCache.get(buildId);
	if (!cached) {
		cached = (async () => {
			const [buildRow] = await db
				.select({ appId: build.appId })
				.from(build)
				.where(eq(build.id, buildId));
			return buildRow ? secretValuesForApp(buildRow.appId) : [];
		})();
		secretsCache.set(buildId, cached);
	}
	return cached;
}

export function clearBuildLogSecretsCache(buildId: string): void {
	secretsCache.delete(buildId);
}

/**
 * Inserted per line rather than batched so the SSE stream (task 09) can pick
 * up new rows as the build progresses instead of only after it finishes.
 * Redacted against the app's own declared secret env values (task 03,
 * Phase 07) before ever reaching the database — covers a build tool or
 * cloned repo's build output that happens to echo one back, even though no
 * build-time secret injection exists yet to make that likely.
 */
export async function appendBuildLog(buildId: string, line: string): Promise<void> {
	const secrets = await secretsForBuild(buildId);
	await db.insert(buildLogLine).values({ buildId, line: redactSecrets(line, secrets) });
}
