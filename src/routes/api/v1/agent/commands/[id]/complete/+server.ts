import { json, error } from '@sveltejs/kit';
import { eq, and } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { hostCommand } from '$lib/server/db/schema';
import { authenticateHost } from '$lib/server/agent/auth';
import { commandCompletionSchema } from '$lib/server/agent/protocol';
import { handleCommandCompletion } from '$lib/server/deploy/orchestrator';

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

	// Zero-downtime rollover state machine (task 06) — advances the linked
	// deployment based on which step just completed.
	await handleCommandCompletion(commandRow, status);

	return json({ success: true });
};
