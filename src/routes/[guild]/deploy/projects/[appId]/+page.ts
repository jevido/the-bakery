import { error } from '@sveltejs/kit';
import { GUILDS } from '$lib/data/bakery';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	const guild = GUILDS[params.guild];
	if (!guild) error(404, 'Guild not found');

	const app = guild.apps.find((a) => a.id === params.appId);
	if (!app) error(404, 'Project not found');
};
