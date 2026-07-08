import { error } from '@sveltejs/kit';
import { GUILD_RESOURCES } from '$lib/data/bakery';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	// Guild existence/membership is already verified by [guild]/+layout.server.ts
	// before this ever runs — this load only needs to resolve the (still mocked
	// until Phase 03) app itself.
	const app = GUILD_RESOURCES[params.guild]?.apps.find((a) => a.id === params.appId);
	if (!app) error(404, 'Project not found');
};
