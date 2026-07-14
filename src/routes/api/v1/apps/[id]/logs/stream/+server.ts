import { error } from '@sveltejs/kit';
import { and, asc, desc, eq, gte } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { app, appLogLine } from '$lib/server/db/schema';
import { requireGuild } from '$lib/server/guild-context';

const POLL_INTERVAL_MS = 1500;
// How much backlog a freshly-opened viewer sees before it starts following
// new lines — a runtime container can accumulate days of log history (per
// its retention window), unlike a build's short-lived log, so sending
// everything on connect isn't reasonable here.
const RECENT_LINES_LIMIT = 500;

/**
 * SSE stream of an app's `appLogLine` rows (task 05) — same polling
 * approach as the build-log stream (Phase 03 task 09), but never closes on
 * its own: a running container's log has no terminal state, so the stream
 * stays open (following new lines) until the client disconnects. Viewing a
 * stopped/removed app's logs still works — it's just a query against
 * `appLogLine`, not a check on whether anything is currently running.
 */
export const GET: RequestHandler = async (event) => {
	const appId = event.params.id;

	const [appRow] = await db
		.select({ organizationId: app.organizationId })
		.from(app)
		.where(eq(app.id, appId));
	if (!appRow) error(404, 'App not found');

	await requireGuild(event, { permission: 'view_apps', organizationId: appRow.organizationId });

	const encoder = new TextEncoder();
	let cursor: Date | null = null;
	// Postgres `timestamp` has microsecond precision; a JS `Date` read from
	// it truncates to milliseconds, so a plain `ts > cursor` comparison can
	// re-match a row already sent (its real stored value is a few
	// microseconds past the truncated cursor). Using `>=` plus this id set
	// (cleared whenever the cursor actually advances to a later timestamp)
	// re-fetches same-timestamp rows without re-emitting ones already sent.
	let seenAtCursor = new Set<string>();
	let timer: ReturnType<typeof setInterval> | undefined;
	let closed = false;

	const stream = new ReadableStream({
		start(controller) {
			const finish = () => {
				if (closed) return;
				closed = true;
				clearInterval(timer);
				try {
					controller.close();
				} catch {
					// already closed by the client disconnecting
				}
			};

			const poll = async () => {
				if (closed) return;

				const rows = cursor
					? await db
							.select({ id: appLogLine.id, ts: appLogLine.ts, message: appLogLine.message })
							.from(appLogLine)
							.where(and(eq(appLogLine.appId, appId), gte(appLogLine.ts, cursor)))
							.orderBy(asc(appLogLine.ts))
					: (
							await db
								.select({ id: appLogLine.id, ts: appLogLine.ts, message: appLogLine.message })
								.from(appLogLine)
								.where(eq(appLogLine.appId, appId))
								.orderBy(desc(appLogLine.ts))
								.limit(RECENT_LINES_LIMIT)
						).reverse();
				if (closed) return;

				for (const row of rows) {
					if (seenAtCursor.has(row.id)) continue;
					controller.enqueue(encoder.encode(`data: ${JSON.stringify(row.message)}\n\n`));
				}

				if (rows.length > 0) {
					const newCursor = rows[rows.length - 1].ts;
					if (cursor === null || newCursor.getTime() !== cursor.getTime()) {
						seenAtCursor = new Set(
							rows.filter((r) => r.ts.getTime() === newCursor.getTime()).map((r) => r.id)
						);
					} else {
						for (const row of rows) seenAtCursor.add(row.id);
					}
					cursor = newCursor;
				}
			};

			void poll();
			timer = setInterval(poll, POLL_INTERVAL_MS);
			event.request.signal.addEventListener('abort', finish);
		},
		cancel() {
			closed = true;
			clearInterval(timer);
		}
	});

	return new Response(stream, {
		headers: {
			'Content-Type': 'text/event-stream',
			'Cache-Control': 'no-cache',
			Connection: 'keep-alive'
		}
	});
};
