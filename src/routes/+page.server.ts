import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { firstGuildPath } from '$lib/server/auth/post-auth-redirect';

export const load: PageServerLoad = async (event) => {
	if (!event.locals.user) {
		redirect(302, '/login');
	}

	const path = await firstGuildPath(event);
	if (path) {
		redirect(302, path);
	}

	return { hasNoGuilds: true as const };
};
