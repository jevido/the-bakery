import type { RequestEvent } from '@sveltejs/kit';
import { auth } from '$lib/server/auth';

/** Path to the user's first guild's dashboard, or null if they have none yet. */
export async function firstGuildPath(event: RequestEvent): Promise<string | null> {
	const orgs = await auth.api.listOrganizations({ headers: event.request.headers });
	const first = orgs[0];
	return first ? `/${first.slug}/deploy/overview` : null;
}
