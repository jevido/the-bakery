import { error } from '@sveltejs/kit';
import { eq, asc } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { build, app, buildLogLine } from '$lib/server/db/schema';
import { requireGuild } from '$lib/server/guild-context';

const POLL_INTERVAL_MS = 750;
const TERMINAL_STATUSES = new Set(['succeeded', 'failed']);

/**
 * SSE stream of a build's `buildLogLine` rows. Polls rather than
 * LISTEN/NOTIFY — simpler than a dedicated Postgres listener connection per
 * open stream, and build logs are small/short-lived enough that re-fetching
 * on an interval is not a real cost for this v1.
 */
export const GET: RequestHandler = async (event) => {
	const buildId = event.params.id;

	const [buildRow] = await db.select().from(build).where(eq(build.id, buildId));
	if (!buildRow) error(404, 'Build not found');

	const [appRow] = await db
		.select({ organizationId: app.organizationId })
		.from(app)
		.where(eq(app.id, buildRow.appId));
	if (!appRow) error(404, 'Build not found');

	await requireGuild(event, { permission: 'view_apps', organizationId: appRow.organizationId });

	const encoder = new TextEncoder();
	let sentCount = 0;
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
				const [lines, [current]] = await Promise.all([
					db
						.select({ line: buildLogLine.line })
						.from(buildLogLine)
						.where(eq(buildLogLine.buildId, buildId))
						.orderBy(asc(buildLogLine.ts), asc(buildLogLine.id)),
					db.select({ status: build.status }).from(build).where(eq(build.id, buildId))
				]);
				if (closed) return;

				for (const { line } of lines.slice(sentCount)) {
					controller.enqueue(encoder.encode(`data: ${JSON.stringify(line)}\n\n`));
				}
				sentCount = lines.length;

				if (!current || TERMINAL_STATUSES.has(current.status)) finish();
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
