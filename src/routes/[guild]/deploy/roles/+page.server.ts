import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { auth } from '$lib/server/auth';
import { db } from '$lib/server/db';
import { member } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { APIError } from 'better-auth/api';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_guild' });

	const customRoles = await auth.api.listOrgRoles({
		headers: event.request.headers,
		query: { organizationId: organization.id }
	});

	const orgMembers = await db
		.select({ role: member.role })
		.from(member)
		.where(eq(member.organizationId, organization.id));

	const memberCounts: Record<string, number> = {};
	for (const m of orgMembers) {
		memberCounts[m.role] = (memberCounts[m.role] ?? 0) + 1;
	}

	return { customRoles, memberCounts, organizationId: organization.id };
};

export const actions: Actions = {
	createRole: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_roles' });
		const formData = await event.request.formData();
		const roleName = formData.get('roleName')?.toString().trim();
		const color = formData.get('color')?.toString() ?? '#5b8def';
		if (!roleName) return fail(400, { message: 'Role name is required' });

		try {
			await auth.api.createOrgRole({
				headers: event.request.headers,
				body: {
					organizationId: organization.id,
					role: roleName,
					permission: { guild: [], apps: [], hosts: [] },
					additionalFields: { color, note: '' }
				}
			});
		} catch (e) {
			if (e instanceof APIError)
				return fail(400, { message: e.message ?? 'Could not create role' });
			throw e;
		}
	},
	updateRole: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_roles' });
		const formData = await event.request.formData();
		const roleId = formData.get('roleId')?.toString();
		const permissionJson = formData.get('permission')?.toString();
		if (!roleId || !permissionJson)
			return fail(400, { message: 'roleId and permission are required' });

		let permission: Record<string, string[]>;
		try {
			permission = JSON.parse(permissionJson);
		} catch {
			return fail(400, { message: 'Invalid permission payload' });
		}

		try {
			await auth.api.updateOrgRole({
				headers: event.request.headers,
				body: { organizationId: organization.id, roleId, data: { permission } }
			});
		} catch (e) {
			if (e instanceof APIError)
				return fail(400, { message: e.message ?? 'Could not update role' });
			throw e;
		}
	},
	deleteRole: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_roles' });
		const formData = await event.request.formData();
		const roleId = formData.get('roleId')?.toString();
		if (!roleId) return fail(400, { message: 'roleId is required' });

		try {
			await auth.api.deleteOrgRole({
				headers: event.request.headers,
				body: { organizationId: organization.id, roleId }
			});
		} catch (e) {
			if (e instanceof APIError)
				return fail(400, { message: e.message ?? 'Could not delete role' });
			throw e;
		}
	}
};
