import { fail } from '@sveltejs/kit';
import { eq, and, sql } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { app, repo, source, build } from '$lib/server/db/schema';
import { getLatestCommitSha } from '$lib/server/github/app-auth';

/**
 * `app.id` may be either a real UUID (created via task 10's "New app" flow)
 * or one of the still-mock `GUILD_RESOURCES` string ids (e.g. "crumb-api")
 * — the latter has no matching row here, so `realApp` is null and the page
 * falls back to fully-mock rendering exactly as before this task.
 */
export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_apps' });

	const [appRow] = await db
		.select()
		.from(app)
		.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));

	if (!appRow) return { realApp: null, repo: null, builds: [] };

	const [repoRow] = appRow.repoId
		? await db.select().from(repo).where(eq(repo.id, appRow.repoId))
		: [null];

	// No `createdAt`/`queuedAt` column on `build` (task 02) — `startedAt` is
	// only set once a worker claims the row, so a still-queued build has no
	// timestamp to sort by yet. Sorting nulls first surfaces exactly the
	// build a "Build now" click just queued, which is what you want to see.
	const builds = await db
		.select()
		.from(build)
		.where(eq(build.appId, appRow.id))
		.orderBy(sql`${build.startedAt} desc nulls first`);

	return { realApp: appRow, repo: repoRow, builds };
};

export const actions: Actions = {
	buildNow: async (event) => {
		const { organization, userId } = await requireGuild(event, { permission: 'deploy_apps' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow?.repoId) return fail(404, { message: 'App not found' });

		const [repoRow] = await db.select().from(repo).where(eq(repo.id, appRow.repoId));
		if (!repoRow) return fail(404, { message: 'App not found' });

		const [sourceRow] = await db.select().from(source).where(eq(source.id, repoRow.sourceId));
		if (!sourceRow?.githubInstallationId) return fail(400, { message: 'Source is not connected' });

		let commitSha: string;
		try {
			commitSha = await getLatestCommitSha(
				sourceRow.githubInstallationId,
				repoRow.fullName,
				repoRow.defaultBranch
			);
		} catch {
			return fail(502, { message: 'Could not reach GitHub to resolve the latest commit' });
		}

		await db.insert(build).values({
			appId: appRow.id,
			repoId: repoRow.id,
			commitSha,
			branch: repoRow.defaultBranch,
			triggeredBy: userId,
			status: 'queued'
		});

		return { success: true };
	}
};
