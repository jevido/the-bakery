import { error, json } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { repo, source } from '$lib/server/db/schema';
import { requireGuild } from '$lib/server/guild-context';
import { detectRemoteBuildFile } from '$lib/server/github/detect-remote-build-file';

/** Live Dockerfile/Containerfile detection preview for the "New app" form (task 10), backing task 07's detection logic. */
export const GET: RequestHandler = async (event) => {
	const [repoRow] = await db.select().from(repo).where(eq(repo.id, event.params.repoId));
	if (!repoRow) error(404, 'Repo not found');

	const [sourceRow] = await db.select().from(source).where(eq(source.id, repoRow.sourceId));
	if (!sourceRow?.githubInstallationId) error(404, 'Repo not found');

	await requireGuild(event, { permission: 'view_apps', organizationId: sourceRow.organizationId });

	const branch = event.url.searchParams.get('branch') || repoRow.defaultBranch;
	const buildContext = event.url.searchParams.get('buildContext') || '.';

	const detected = await detectRemoteBuildFile(
		sourceRow.githubInstallationId,
		repoRow.fullName,
		branch,
		buildContext
	).catch(() => null);

	return json({ detected });
};
