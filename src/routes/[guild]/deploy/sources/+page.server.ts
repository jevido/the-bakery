import { fail, redirect } from '@sveltejs/kit';
import { and, eq, inArray } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { source, repo } from '$lib/server/db/schema';
import { createGithubInstallState } from '$lib/server/github/install-state';
import { syncReposForSource } from '$lib/server/github/sync-repos';

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const sourceRows = await db.select().from(source).where(eq(source.organizationId, organization.id));

	const sourceIds = sourceRows.map((s) => s.id);
	const repoRows = sourceIds.length
		? await db.select().from(repo).where(inArray(repo.sourceId, sourceIds))
		: [];

	const reposBySource = new Map<string, typeof repoRows>();
	for (const r of repoRows) {
		const list = reposBySource.get(r.sourceId) ?? [];
		list.push(r);
		reposBySource.set(r.sourceId, list);
	}

	const sources = sourceRows.map((s) => ({ ...s, repos: reposBySource.get(s.id) ?? [] }));

	return { sources };
};

export const actions: Actions = {
	connectGithub: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });

		const state = createGithubInstallState(organization.id);
		const slug = process.env.GITHUB_APP_SLUG;
		const installUrl = new URL(`https://github.com/apps/${slug}/installations/new`);
		installUrl.searchParams.set('state', state);

		redirect(302, installUrl.toString(), { external: ['https://github.com'] });
	},

	refreshRepos: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });

		const formData = await event.request.formData();
		const sourceId = formData.get('sourceId');
		if (typeof sourceId !== 'string' || !sourceId) {
			return fail(400, { message: 'Missing source id' });
		}

		const [sourceRow] = await db
			.select()
			.from(source)
			.where(and(eq(source.id, sourceId), eq(source.organizationId, organization.id)));
		if (!sourceRow || !sourceRow.githubInstallationId) {
			return fail(404, { message: 'Source not found' });
		}

		try {
			const count = await syncReposForSource(sourceRow.id, sourceRow.githubInstallationId);
			return { refreshed: true, count };
		} catch (e) {
			return fail(502, { message: e instanceof Error ? e.message : 'Failed to refresh repos from GitHub' });
		}
	}
};
