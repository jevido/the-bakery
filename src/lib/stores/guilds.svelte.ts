import { page } from '$app/state';

export interface GuildSummary {
	id: string;
	name: string;
	slug: string;
	color: string;
}

/**
 * Thin reactive wrapper around the root layout's `organizations` load data
 * (src/routes/+layout.server.ts, real Better Auth organizations for the
 * current user). No local mutation — creating/joining a guild triggers a
 * reload via `invalidateAll()` instead.
 */
class GuildStore {
	get list(): GuildSummary[] {
		return (page.data.organizations ?? []) as GuildSummary[];
	}
}

export const guildStore = new GuildStore();
