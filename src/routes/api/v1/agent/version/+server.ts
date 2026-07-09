import { json, error } from '@sveltejs/kit';
import { desc, eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { agentRelease } from '$lib/server/db/schema';

// Release metadata is global, not guild-scoped — no auth required, same as
// the public install.sh / static release binaries this endpoint points at.
export const GET: RequestHandler = async (event) => {
	const platform = event.url.searchParams.get('platform');
	if (!platform) {
		error(400, 'Missing platform query parameter');
	}

	const [release] = await db
		.select()
		.from(agentRelease)
		.where(eq(agentRelease.platform, platform))
		.orderBy(desc(agentRelease.releasedAt))
		.limit(1);

	if (!release) {
		error(404, 'No release available for this platform');
	}

	return json({
		version: release.version,
		downloadUrl: release.downloadUrl,
		sha256: release.sha256
	});
};
