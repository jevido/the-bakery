import { error, redirect } from '@sveltejs/kit';
import type { RequestEvent } from '@sveltejs/kit';
import { auth } from '$lib/server/auth';
import { roles, PERMISSION_RESOURCE } from '$lib/auth/permissions';
import { db } from '$lib/server/db';
import { organizationRole } from '$lib/server/db/schema';
import { eq, and } from 'drizzle-orm';

/**
 * Every guild-scoped `load`/action/`+server.ts` handler (hosts, apps, sources,
 * env vars, etc. — everything from Phase 02 onward) MUST resolve the current
 * guild through this helper and filter all of its queries by the returned
 * `organization.id`. This is the single enforcement point for strict
 * guild-level data isolation across the whole app — do not re-implement the
 * slug → organization lookup ad hoc in individual routes.
 */

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
		const staticRole = roles[member.role as keyof typeof roles];

		let authorized = resource
			? (staticRole?.authorize({ [resource]: [opts.permission] } as never).success ?? false)
			: false;

		// member.role isn't one of the 4 static defaults — it's a custom role
		// created via dynamicAccessControl (Phase 01 task 14). Those aren't in
		// the static `roles` map at all, so check the real organizationRole
		// grant directly rather than always denying.
		if (!authorized && !staticRole && resource) {
			const [customRole] = await db
				.select()
				.from(organizationRole)
				.where(and(eq(organizationRole.organizationId, organization.id), eq(organizationRole.role, member.role)));
			if (customRole) {
				const permission = JSON.parse(customRole.permission) as Record<string, string[]>;
				authorized = (permission[resource] ?? []).includes(opts.permission);
			}
		}

		if (!authorized) {
			error(403, 'You do not have permission to do that.');
		}
	}

	return {
		organization,
		member,
		userId: event.locals.user.id
	};
}
