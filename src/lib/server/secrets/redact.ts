import { and, eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { envVar } from '$lib/server/db/schema';
import { decrypt } from './crypto';

const REDACTED_PLACEHOLDER = '***REDACTED***';

/**
 * Replaces every exact occurrence of a known secret value in `line` with a
 * placeholder. Applied at the point a log line is captured/persisted (build
 * worker, check-in app-log ingestion) rather than at display time, so the
 * plaintext is never at rest in `buildLogLine`/`appLogLine` in the first
 * place — the goal named in this task.
 */
export function redactSecrets(line: string, secretValues: string[]): string {
	let redacted = line;
	for (const value of secretValues) {
		// A blank secret value would match (and mangle) every line — skip it
		// rather than ever redacting against one.
		if (!value.trim()) continue;
		redacted = redacted.split(value).join(REDACTED_PLACEHOLDER);
	}
	return redacted;
}

/** Every decrypted `isSecret` envVar value for an app — the redaction set for its build/app logs. */
export async function secretValuesForApp(appId: string): Promise<string[]> {
	const rows = await db
		.select({ valueCiphertext: envVar.valueCiphertext })
		.from(envVar)
		.where(and(eq(envVar.appId, appId), eq(envVar.isSecret, true)));
	return rows.map((r) => decrypt(r.valueCiphertext));
}
