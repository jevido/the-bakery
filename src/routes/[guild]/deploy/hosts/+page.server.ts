import { fail } from '@sveltejs/kit';
import { superValidate } from 'sveltekit-superforms';
import { zod } from '$lib/forms/zod4-adapter';
import { z } from 'zod';
import { and, eq, isNull } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host } from '$lib/server/db/schema';
import { issueHostToken } from '$lib/server/hosts/tokens';

const addHostSchema = z.object({
	name: z.string().trim().min(1, 'Host name is required').max(100),
	location: z.string().trim().max(200).optional(),
	spec: z.string().trim().max(200).optional()
});

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const hosts = await db
		.select()
		.from(host)
		.where(and(eq(host.organizationId, organization.id), isNull(host.revokedAt)));

	const form = await superValidate(zod(addHostSchema));

	return { hosts, form };
};

export const actions: Actions = {
	addHost: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });
		const form = await superValidate(event, zod(addHostSchema));
		if (!form.valid) return fail(400, { form });

		const issued = issueHostToken();

		const [created] = await db
			.insert(host)
			.values({
				organizationId: organization.id,
				name: form.data.name,
				location: form.data.location || null,
				spec: form.data.spec || null,
				tokenHash: issued.hash,
				tokenLastFour: issued.lastFour,
				status: 'pending'
			})
			.returning();

		// bakery-domain: prefer the configured public origin, fall back to the
		// request's own origin so this works before ORIGIN is set in dev.
		const bakeryDomain = process.env.ORIGIN ?? event.url.origin;
		const installCommand = `curl -fsSL ${bakeryDomain}/install.sh | sh -s -- --token=${issued.plaintext} --url=${bakeryDomain}`;

		return { form, host: created, installCommand };
	}
};
