import { fail } from '@sveltejs/kit';
import { eq, and, sql, desc } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { app, repo, source, build, deployment, domain } from '$lib/server/db/schema';
import { getLatestCommitSha } from '$lib/server/github/app-auth';
import { quadletContent, versionedUnitName } from '$lib/server/deploy/quadlet';
import { startDeployment } from '$lib/server/deploy/orchestrator';
import { resolveHostForApp } from '$lib/server/deploy/host-assignment';

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

	if (!appRow) return { realApp: null, repo: null, builds: [], domains: [] };

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

	// The Quadlet unit needs a real image to reference — only a succeeded
	// build has one (`build.imageRef` is set on push, task 08), so a still-
	// building or never-built app has no real unit content to preview yet.
	const latestSucceeded = builds.find((b) => b.status === 'succeeded' && b.imageRef);
	let realQuadletContent: string | null = null;
	if (latestSucceeded?.imageRef) {
		const unitName = versionedUnitName(appRow.name, latestSucceeded.commitSha);
		realQuadletContent = quadletContent(appRow, latestSucceeded.imageRef, unitName);
	}

	const deployments = await db
		.select({
			id: deployment.id,
			status: deployment.status,
			triggeredBy: deployment.triggeredBy,
			startedAt: deployment.startedAt,
			finishedAt: deployment.finishedAt,
			buildId: deployment.buildId,
			commitSha: build.commitSha,
			branch: build.branch
		})
		.from(deployment)
		.innerJoin(build, eq(build.id, deployment.buildId))
		.where(eq(deployment.appId, appRow.id))
		.orderBy(desc(deployment.startedAt));

	const domains = await db.select().from(domain).where(eq(domain.appId, appRow.id));

	return { realApp: appRow, repo: repoRow, builds, realQuadletContent, deployments, domains };
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
	},

	deploy: async (event) => {
		const { organization, userId } = await requireGuild(event, { permission: 'deploy_apps' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const requestedBuildId = formData.get('buildId');

		let buildRow;
		if (typeof requestedBuildId === 'string' && requestedBuildId) {
			[buildRow] = await db
				.select()
				.from(build)
				.where(
					and(
						eq(build.id, requestedBuildId),
						eq(build.appId, appRow.id),
						eq(build.status, 'succeeded')
					)
				);
			if (!buildRow) return fail(400, { message: 'Selected build is not available to deploy' });
		} else {
			[buildRow] = await db
				.select()
				.from(build)
				.where(and(eq(build.appId, appRow.id), eq(build.status, 'succeeded')))
				.orderBy(desc(build.finishedAt))
				.limit(1);
			if (!buildRow) return fail(400, { message: 'No successful build to deploy yet' });
		}

		const hostRow = await resolveHostForApp(appRow, organization.id);
		if (!hostRow) return fail(400, { message: 'No host available — add a host first' });

		await startDeployment({ appRow, buildRow, hostRow, triggeredBy: userId });

		return { success: true };
	},

	rollback: async (event) => {
		const { organization, userId } = await requireGuild(event, { permission: 'deploy_apps' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const targetDeploymentId = formData.get('deploymentId');
		if (typeof targetDeploymentId !== 'string' || !targetDeploymentId) {
			return fail(400, { message: 'Missing deployment to roll back to' });
		}

		const [targetDeployment] = await db
			.select()
			.from(deployment)
			.where(and(eq(deployment.id, targetDeploymentId), eq(deployment.appId, appRow.id)));
		if (!targetDeployment) return fail(404, { message: 'Deployment not found' });

		// A rollback reuses that deployment's already-built image — no new
		// build — but still runs the full zero-downtime state machine (task
		// 06), not a special-cased unsafe swap.
		const [buildRow] = await db
			.select()
			.from(build)
			.where(and(eq(build.id, targetDeployment.buildId), eq(build.status, 'succeeded')));
		if (!buildRow?.imageRef) return fail(400, { message: 'That build is no longer deployable' });

		const hostRow = await resolveHostForApp(appRow, organization.id);
		if (!hostRow) return fail(400, { message: 'No host available — add a host first' });

		await startDeployment({ appRow, buildRow, hostRow, triggeredBy: userId });

		return { success: true };
	}
};
