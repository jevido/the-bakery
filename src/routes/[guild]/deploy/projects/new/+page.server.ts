import { fail, redirect } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import { superValidate, message } from 'sveltekit-superforms';
import { zod } from '$lib/forms/zod4-adapter';
import { z } from 'zod';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { source, repo, app, domain } from '$lib/server/db/schema';
import { defaultSubdomain } from '$lib/server/deploy/subdomain';

// No `branch` field: the schema (task 02) scopes a build-triggering branch
// to the *repo*, not the app — `repo.defaultBranch` is what the webhook
// (task 04) and manual "Build now" (this task) both build from. The form
// still surfaces it (read-only, informational) once a repo is picked.
const newAppSchema = z.object({
	repoId: z.string().uuid('Pick a repository'),
	name: z.string().trim().min(1, 'App name is required').max(100),
	buildContext: z.string().trim().min(1).max(300).default('.')
});

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'create_apps' });

	const repos = await db
		.select({ id: repo.id, fullName: repo.fullName, defaultBranch: repo.defaultBranch })
		.from(repo)
		.innerJoin(source, eq(repo.sourceId, source.id))
		.where(eq(source.organizationId, organization.id));

	const form = await superValidate(zod(newAppSchema));

	return { form, repos };
};

export const actions: Actions = {
	create: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'create_apps' });
		const form = await superValidate(event, zod(newAppSchema));
		if (!form.valid) return fail(400, { form });

		const [repoRow] = await db
			.select({ id: repo.id, sourceId: repo.sourceId })
			.from(repo)
			.where(eq(repo.id, form.data.repoId));
		if (!repoRow) return message(form, 'That repository is no longer available', { status: 400 });

		const [sourceRow] = await db
			.select({ organizationId: source.organizationId })
			.from(source)
			.where(eq(source.id, repoRow.sourceId));
		if (!sourceRow || sourceRow.organizationId !== organization.id) {
			return message(form, 'That repository is no longer available', { status: 400 });
		}

		const [created] = await db
			.insert(app)
			.values({
				organizationId: organization.id,
				name: form.data.name,
				repoId: repoRow.id,
				buildContext: form.data.buildContext
			})
			.returning({ id: app.id });

		await db.insert(domain).values({
			appId: created.id,
			hostname: defaultSubdomain(form.data.name, organization.slug),
			isDefaultSubdomain: true
		});

		redirect(303, `/${organization.slug}/deploy/projects/${created.id}`);
	}
};
