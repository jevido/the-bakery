import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { auth } from '$lib/server/auth';
import { APIError } from 'better-auth/api';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_guild' });

	const { members } = await auth.api.listMembers({
		headers: event.request.headers,
		query: { organizationId: organization.id }
	});

	return { members };
};

export const actions: Actions = {
	removeMember: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_members' });
		const formData = await event.request.formData();
		const memberId = formData.get('memberId')?.toString();
		if (!memberId) return fail(400, { message: 'memberId is required' });

		try {
			await auth.api.removeMember({
				headers: event.request.headers,
				body: { memberIdOrEmail: memberId, organizationId: organization.id }
			});
		} catch (e) {
			if (e instanceof APIError) return fail(400, { message: e.message ?? 'Could not remove member' });
			throw e;
		}
	},
	updateRole: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_members' });
		const formData = await event.request.formData();
		const memberId = formData.get('memberId')?.toString();
		const role = formData.get('role')?.toString();
		if (!memberId || !role) return fail(400, { message: 'memberId and role are required' });

		try {
			await auth.api.updateMemberRole({
				headers: event.request.headers,
				body: { memberId, role, organizationId: organization.id }
			});
		} catch (e) {
			if (e instanceof APIError) return fail(400, { message: e.message ?? 'Could not update role' });
			throw e;
		}
	}
};
