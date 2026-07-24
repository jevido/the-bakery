import { error } from '@sveltejs/kit';
import { and, asc, desc, eq, gte } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { deployment, app, appLogLine } from '$lib/server/db/schema';
import { requireGuild } from '$lib/server/guild-context';

const POLL_INTERVAL_MS = 1500;
// Same backlog size as the app-level stream (`apps/[id]/logs/stream`) — a
// deployment's log history is the same kind of unbounded-length runtime
// output, just scoped narrower.
const RECENT_LINES_LIMIT = 500;

/**
 * SSE stream of one deployment's `appLogLine` rows (Phase 09 task 03) —
 * identical polling approach to the app-level log stream (Phase 06 task
 * 05), just filtered by `deploymentId` instead of `appId`. Never closes on
 * its own: the currently-running deployment's container can keep producing
 * lines, and a finished one simply has nothing new arrive, which needs no
 * special-casing — same "viewing a stopped app's logs still works" behavior
 * the app-level stream already relies on.
 */
export const GET: RequestHandler = async (event) => {
	const deploymentId = event.params.id;

	const [deploymentRow] = await db
		.select({ appId: deployment.appId })
		.from(deployment)
		.where(eq(deployment.id, deploymentId));
	if (!deploymentRow) error(404, 'Deployment not found');

	const [appRow] = await db
		.select({ organizationId: app.organizationId })
		.from(app)
		.where(eq(app.id, deploymentRow.appId));
	if (!appRow) error(404, 'Deployment not found');

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
							.where(and(eq(appLogLine.deploymentId, deploymentId), gte(appLogLine.ts, cursor)))
							.orderBy(asc(appLogLine.ts))
					: (
							await db
								.select({ id: appLogLine.id, ts: appLogLine.ts, message: appLogLine.message })
								.from(appLogLine)
								.where(eq(appLogLine.deploymentId, deploymentId))
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
