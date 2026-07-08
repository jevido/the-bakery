import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { auth } from '$lib/server/auth';
import { APIError } from 'better-auth/api';

function slugify(name: string): string {
	return name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '') || 'guild';
}

export const POST: RequestHandler = async (event) => {
	if (!event.locals.user) {
		error(401, 'Not authenticated');
	}

	const { name, color } = await event.request.json();
	if (typeof name !== 'string' || !name.trim() || typeof color !== 'string') {
		error(400, 'name and color are required');
	}

	const base = slugify(name);
	let slug = base;
	let attempt = 0;

	// createOrganization rejects a taken slug; retry with a random suffix rather
	// than pre-checking (avoids a race between the check and the create call).
	for (;;) {
		try {
			const inviteCode = `${slug}-${Math.floor(Math.random() * 90 + 10)}`;
			const org = await auth.api.createOrganization({
				headers: event.request.headers,
				body: { name: name.trim(), slug, color, inviteCode }
			});
			return json({ slug: org?.slug });
		} catch (e) {
			if (e instanceof APIError && e.body?.message?.includes('already exists') && attempt < 5) {
				attempt += 1;
				slug = `${base}-${Math.random().toString(36).slice(2, 6)}`;
				continue;
			}
			if (e instanceof APIError) {
				error(400, e.message ?? 'Could not create guild');
			}
			throw e;
		}
	}
};
