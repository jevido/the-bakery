import type { LayoutServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';

export const load: LayoutServerLoad = async (event) => {
	const { organization, member } = await requireGuild(event);

	return {
		user: event.locals.user!,
		session: event.locals.session!,
		organization,
		member
	};
};
