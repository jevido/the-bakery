import { redirect } from '@sveltejs/kit';
import type { Actions } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { createGithubInstallState } from '$lib/server/github/install-state';

export const actions: Actions = {
	connectGithub: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });

		const state = createGithubInstallState(organization.id);
		const slug = process.env.GITHUB_APP_SLUG;
		const installUrl = new URL(`https://github.com/apps/${slug}/installations/new`);
		installUrl.searchParams.set('state', state);

		redirect(302, installUrl.toString(), { external: ['https://github.com'] });
	}
};
