import type { LayoutServerLoad } from './$types';
import { auth } from '$lib/server/auth';

export const load: LayoutServerLoad = async (event) => {
	if (!event.locals.user) {
		return { organizations: [] };
	}

	const organizations = await auth.api.listOrganizations({ headers: event.request.headers });
	return { organizations };
};
