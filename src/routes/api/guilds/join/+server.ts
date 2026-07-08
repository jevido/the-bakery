import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { auth } from '$lib/server/auth';
import { db } from '$lib/server/db';
import { organization } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { APIError } from 'better-auth/api';

async function findByInviteCode(code: string) {
	const [org] = await db.select().from(organization).where(eq(organization.inviteCode, code));
	return org;
}

/** Preview a guild by invite code without joining it, for the "join" dialog step. */
export const GET: RequestHandler = async (event) => {
	const code = event.url.searchParams.get('code')?.trim();
	if (!code) {
		error(400, 'code is required');
	}

	const org = await findByInviteCode(code);
	if (!org) {
		error(404, 'No guild found with that invite code.');
	}

	return json({ name: org.name, slug: org.slug, color: org.color });
};

export const POST: RequestHandler = async (event) => {
	if (!event.locals.user) {
		error(401, 'Not authenticated');
	}

	const { inviteCode } = await event.request.json();
	if (typeof inviteCode !== 'string' || !inviteCode.trim()) {
		error(400, 'inviteCode is required');
	}

	const org = await findByInviteCode(inviteCode.trim());
	if (!org) {
		error(404, 'No guild found with that invite code.');
	}

	try {
		await auth.api.addMember({
			headers: event.request.headers,
			body: { userId: event.locals.user.id, organizationId: org.id, role: 'apprentice' }
		});
	} catch (e) {
		if (e instanceof APIError && e.body?.message?.includes('already a member')) {
			// Already in — treat as success, just take them there.
			return json({ slug: org.slug });
		}
		if (e instanceof APIError) {
			error(400, e.message ?? 'Could not join guild');
		}
		throw e;
	}

	return json({ slug: org.slug });
};
