import { error, redirect } from '@sveltejs/kit';
import type { RequestEvent } from '@sveltejs/kit';
import { auth } from '$lib/server/auth';
import { roles } from '$lib/auth/permissions';

/**
 * Every guild-scoped `load`/action/`+server.ts` handler (hosts, apps, sources,
 * env vars, etc. — everything from Phase 02 onward) MUST resolve the current
 * guild through this helper and filter all of its queries by the returned
 * `organization.id`. This is the single enforcement point for strict
 * guild-level data isolation across the whole app — do not re-implement the
 * slug → organization lookup ad hoc in individual routes.
 */

type PermissionResource = 'guild' | 'apps' | 'hosts';

/** Maps each permission id from the mock PERM_GROUPS catalogue to the
 * access-control resource it lives under in src/lib/auth/permissions.ts. */
const PERMISSION_RESOURCE: Record<string, PermissionResource> = {
	view_guild: 'guild',
	manage_guild: 'guild',
	manage_roles: 'guild',
	manage_members: 'guild',
	audit_log: 'guild',
	view_apps: 'apps',
	create_apps: 'apps',
	deploy_apps: 'apps',
	manage_env: 'apps',
	view_secrets: 'apps',
	delete_apps: 'apps',
	view_hosts: 'hosts',
	manage_hosts: 'hosts',
	manage_domains: 'hosts'
};

export async function requireGuild(event: RequestEvent, opts?: { permission?: string }) {
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

	if (opts?.permission) {
		const resource = PERMISSION_RESOURCE[opts.permission];
		const role = roles[member.role as keyof typeof roles];
		const authorized = resource && role?.authorize({ [resource]: [opts.permission] } as never);
		if (!authorized?.success) {
			error(403, 'You do not have permission to do that.');
		}
	}

	return {
		organization,
		member,
		userId: event.locals.user.id
	};
}
