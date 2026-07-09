import { json, error } from '@sveltejs/kit';
import { eq, and } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { hostCommand, deployment } from '$lib/server/db/schema';
import { authenticateHost } from '$lib/server/agent/auth';
import { commandCompletionSchema } from '$lib/server/agent/protocol';

/**
 * Maps a completed hostCommand back onto its deployment's state machine
 * (task 01). This is deliberately the simplest correct mapping — task 06's
 * zero-downtime rollover adds the intermediate `health_checking`/
 * `flipping_proxy`/`stopping_old` steps; this task only wires completion
 * reporting through to *some* terminal-ish status.
 */
function nextDeploymentStatus(
	type: (typeof hostCommand.$inferSelect)['type'],
	status: 'succeeded' | 'failed'
) {
	if (status === 'failed') return 'failed' as const;
	if (type === 'stop') return 'rolled_back' as const;
	return 'running' as const;
}

export const POST: RequestHandler = async (event) => {
	const matchedHost = await authenticateHost(event.request);

	const body = await event.request.json().catch(() => null);
	const parsed = commandCompletionSchema.safeParse(body);
	if (!parsed.success) error(400, 'Invalid completion payload');

	const [commandRow] = await db
		.select()
		.from(hostCommand)
		.where(and(eq(hostCommand.id, event.params.id), eq(hostCommand.hostId, matchedHost.id)));

	if (!commandRow) error(404, 'Command not found');

	const { status, errorMessage } = parsed.data;

	await db
		.update(hostCommand)
		.set({ status, completedAt: new Date(), errorMessage: errorMessage ?? null })
		.where(eq(hostCommand.id, commandRow.id));

	await db
		.update(deployment)
		.set({
			status: nextDeploymentStatus(commandRow.type, status),
			finishedAt: new Date()
		})
		.where(eq(deployment.id, commandRow.deploymentId));

	return json({ success: true });
};
