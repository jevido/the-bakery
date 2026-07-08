import { fail } from '@sveltejs/kit';
import { superValidate, message } from 'sveltekit-superforms';
import { zod } from '$lib/forms/zod4-adapter';
import { z } from 'zod';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { auth } from '$lib/server/auth';
import { APIError } from 'better-auth/api';

const settingsSchema = z.object({
	name: z.string().trim().min(1, 'Guild name is required').max(100),
	color: z
		.string()
		.regex(/^#[0-9a-f]{6}$/i, 'Must be a hex color, e.g. #3fb984')
});

type OrgWithFields = { name: string; color?: string; inviteCode?: string };

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'manage_guild' });
	const org = organization as OrgWithFields;

	const form = await superValidate(
		{ name: org.name, color: org.color ?? '#3fb984' },
		zod(settingsSchema)
	);

	return { form };
};

export const actions: Actions = {
	update: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_guild' });
		const form = await superValidate(event, zod(settingsSchema));
		if (!form.valid) return fail(400, { form });

		try {
			await auth.api.updateOrganization({
				headers: event.request.headers,
				body: {
					organizationId: organization.id,
					data: { name: form.data.name, color: form.data.color }
				}
			});
		} catch (e) {
			if (e instanceof APIError) return message(form, e.message ?? 'Could not update guild', { status: 400 });
			throw e;
		}

		return message(form, 'Guild settings updated');
	},
	regenerateInvite: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_guild' });
		const newCode = `${organization.slug}-${Math.floor(Math.random() * 90 + 10)}`;

		try {
			await auth.api.updateOrganization({
				headers: event.request.headers,
				body: { organizationId: organization.id, data: { inviteCode: newCode } }
			});
		} catch (e) {
			if (e instanceof APIError) return fail(400, { message: e.message ?? 'Could not regenerate invite code' });
			throw e;
		}
	}
};
