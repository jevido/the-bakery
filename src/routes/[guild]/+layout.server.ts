import { redirect, error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { auth } from '$lib/server/auth';

export const load: LayoutServerLoad = async (event) => {
	if (!event.locals.user || !event.locals.session) {
		redirect(302, '/login');
	}

	const organization = await auth.api
		.getFullOrganization({
			headers: event.request.headers,
			query: { organizationSlug: event.params.guild }
		})
		.catch(() => null);

	// Same 404 for "guild doesn't exist" and "you're not a member of it" —
	// don't leak a guild's existence to people who aren't in it.
	const member = organization?.members.find((m) => m.userId === event.locals.user!.id);
	if (!organization || !member) {
		error(404, 'Guild not found');
	}

	return {
		user: event.locals.user,
		session: event.locals.session,
		organization,
		member
	};
};
