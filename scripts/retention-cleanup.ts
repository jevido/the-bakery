/**
 * Deletes rows older than their table's retention window from the metric
 * and log tables that would otherwise grow unbounded — no in-app job
 * scheduler exists yet (consistent with the "computed on read" pattern used
 * elsewhere), so this is meant to be invoked by a host-level cron entry or
 * systemd timer, e.g. daily:
 *
 *   0 3 * * * cd /path/to/bakery && bun scripts/retention-cleanup.ts
 *
 * Config knobs (env, both optional):
 *   RETENTION_DAYS_METRICS — hostMetricSample/appMetricSample cutoff, default 30
 *   RETENTION_DAYS_LOGS    — buildLogLine (and appLogLine, once task 05 adds
 *                            it) cutoff, default 30
 *
 * Run with `bun scripts/retention-cleanup.ts`.
 */
import { lt, inArray, type AnyColumn, type Table } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { hostMetricSample, appMetricSample, buildLogLine } from '$lib/server/db/schema';

const RETENTION_DAYS_METRICS = Number(process.env.RETENTION_DAYS_METRICS ?? 30);
const RETENTION_DAYS_LOGS = Number(process.env.RETENTION_DAYS_LOGS ?? 30);

// Caps how many rows a single DELETE touches — Postgres has no native
// `DELETE ... LIMIT`, so each batch is a `select id ... limit N` followed by
// a `delete where id in (...)`, looped until nothing's left past the
// cutoff. Keeps any one statement's lock/duration bounded regardless of how
// far retention has fallen behind (e.g. after downtime).
const BATCH_SIZE = 1000;

interface RetentionTarget {
	label: string;
	table: Table & { id: AnyColumn; ts: AnyColumn };
	retentionDays: number;
}

const TARGETS: RetentionTarget[] = [
	{ label: 'host_metric_sample', table: hostMetricSample, retentionDays: RETENTION_DAYS_METRICS },
	{ label: 'app_metric_sample', table: appMetricSample, retentionDays: RETENTION_DAYS_METRICS },
	{ label: 'build_log_line', table: buildLogLine, retentionDays: RETENTION_DAYS_LOGS }
	// appLogLine (task 05) belongs here too, once that table exists.
];

async function deleteOlderThan(target: RetentionTarget): Promise<number> {
	const cutoff = new Date(Date.now() - target.retentionDays * 24 * 60 * 60 * 1000);
	let totalDeleted = 0;

	while (true) {
		const rows = await db
			.select({ id: target.table.id })
			.from(target.table)
			.where(lt(target.table.ts, cutoff))
			.limit(BATCH_SIZE);
		if (rows.length === 0) break;

		await db.delete(target.table).where(
			inArray(
				target.table.id,
				rows.map((r) => r.id)
			)
		);
		totalDeleted += rows.length;

		if (rows.length < BATCH_SIZE) break;
	}

	return totalDeleted;
}

async function main() {
	console.log(
		`[retention-cleanup] started (metrics: ${RETENTION_DAYS_METRICS}d, logs: ${RETENTION_DAYS_LOGS}d)`
	);

	for (const target of TARGETS) {
		const deleted = await deleteOlderThan(target);
		console.log(`[retention-cleanup] ${target.label}: deleted ${deleted} row(s)`);
	}

	console.log('[retention-cleanup] done');
}

main()
	.then(() => process.exit(0))
	.catch((err) => {
		console.error('[retention-cleanup] failed:', err);
		process.exit(1);
	});
