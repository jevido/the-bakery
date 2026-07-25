import { fail } from '@sveltejs/kit';
import { eq, and, sql, desc, asc, gte } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import {
	app,
	repo,
	source,
	build,
	deployment,
	domain,
	volume,
	appMetricSample
} from '$lib/server/db/schema';
import { z } from 'zod';
import { getLatestCommitSha } from '$lib/server/github/app-auth';
import { quadletContent, versionedUnitName, guildNetworkName } from '$lib/server/deploy/quadlet';
import {
	startDeployment,
	refreshProxyConfigForApp,
	stopDeployment
} from '$lib/server/deploy/orchestrator';
import { resolveHostForApp } from '$lib/server/deploy/host-assignment';
import { verifyCnameTarget } from '$lib/server/deploy/custom-domain';

// Loose but real: each label alphanumeric (hyphens allowed mid-label), at
// least two labels. DNS resolution during verification is the actual
// authority — this is just enough to reject obvious typos in the form.
const HOSTNAME_PATTERN =
	/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$/;

// Same character set Podman itself accepts for a volume name, minus
// characters that'd be awkward in `podmanVolumeName()`'s `-`-joined suffix.
const VOLUME_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,31}$/;
// Must be an absolute, non-root path — `Volume=<name>:/` would shadow the
// entire container filesystem.
const MOUNT_PATH_PATTERN = /^\/[a-zA-Z0-9_./-]*[a-zA-Z0-9_-]$/;

// Matches the overview page's history window (Phase 02 task 11) — an hour
// of minute-level samples is enough to show a real, if short, trend without
// the chart looking sparse for a just-deployed app.
const APP_METRIC_HISTORY_WINDOW_MS = 60 * 60 * 1000;

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

	if (!appRow) {
		return {
			realApp: null,
			repo: null,
			connectableRepos: [],
			builds: [],
			domains: [],
			volumes: [],
			appMetricHistory: [],
			latestAppMetricSample: null,
			realContainers: []
		};
	}

	// Only queried for a repo-less app (created via `bakery bootstrap`, task
	// 12, Phase 08, or one whose `repo` row was removed via `onDelete: 'set
	// null'` after a GitHub App uninstall) — every normal app already picked
	// its repo at creation time (`projects/new`) and never needs this list.
	const connectableRepos = appRow.repoId
		? []
		: await db
				.select({ id: repo.id, fullName: repo.fullName, defaultBranch: repo.defaultBranch })
				.from(repo)
				.innerJoin(source, eq(repo.sourceId, source.id))
				.where(eq(source.organizationId, organization.id));

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

	const volumes = await db
		.select()
		.from(volume)
		.where(eq(volume.appId, appRow.id))
		.orderBy(volume.createdAt);

	// The Quadlet unit needs a real image to reference — only a succeeded
	// build has one (`build.imageRef` is set on push, task 08), so a still-
	// building or never-built app has no real unit content to preview yet.
	const latestSucceeded = builds.find((b) => b.status === 'succeeded' && b.imageRef);
	let realQuadletContent: string | null = null;
	if (latestSucceeded?.imageRef) {
		const unitName = versionedUnitName(appRow.name, latestSucceeded.commitSha);
		realQuadletContent = quadletContent(appRow, latestSucceeded.imageRef, unitName, volumes);
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
			branch: build.branch,
			imageRef: build.imageRef
		})
		.from(deployment)
		.innerJoin(build, eq(build.id, deployment.buildId))
		.where(eq(deployment.appId, appRow.id))
		.orderBy(desc(deployment.startedAt));

	// Bakery's real deploy model is exactly one container per app (no sidecar
	// concept anywhere in the schema or `quadletContent()`), so the Containers
	// tab (Phase 11) is inherently a 0-or-1-row list: the one deployment
	// that's actually serving traffic right now, same "find the `running`
	// one" logic the Rollouts list already uses client-side for
	// `currentDeploymentId` — done here instead since `versionedUnitName`/
	// `guildNetworkName` are server-only.
	const currentDeployment = deployments.find((d) => d.status === 'running');
	const realContainers = currentDeployment
		? [
				{
					name: appRow.name,
					unit: versionedUnitName(appRow.name, currentDeployment.commitSha),
					image: currentDeployment.imageRef ?? 'unknown',
					nets: guildNetworkName(organization.id)
				}
			]
		: [];

	const domains = await db
		.select()
		.from(domain)
		.where(eq(domain.appId, appRow.id))
		.orderBy(desc(domain.isDefaultSubdomain), domain.createdAt);

	// Per-container CPU/mem history (Phase 06 task 01) — replaces the app
	// detail page's sine-wave-driven sparkbars with real data.
	const appMetricHistory = await db
		.select()
		.from(appMetricSample)
		.where(
			and(
				eq(appMetricSample.appId, appRow.id),
				gte(appMetricSample.ts, new Date(Date.now() - APP_METRIC_HISTORY_WINDOW_MS))
			)
		)
		.orderBy(asc(appMetricSample.ts));
	const latestAppMetricSample = appMetricHistory[appMetricHistory.length - 1] ?? null;

	return {
		realApp: appRow,
		repo: repoRow,
		connectableRepos,
		builds,
		realQuadletContent,
		deployments,
		domains,
		volumes,
		appMetricHistory,
		latestAppMetricSample,
		realContainers
	};
};

const connectRepoSchema = z.object({ repoId: z.string().uuid() });

export const actions: Actions = {
	// Points a repo-less app at one of the guild's already-synced repos —
	// the counterpart to `projects/new`'s creation-time repo pick, for an
	// app that either never had one (`bakery bootstrap`, task 12, Phase 08)
	// or lost it (`app.repoId`'s `onDelete: 'set null'`, e.g. a GitHub App
	// uninstall). `buildContext` isn't asked for here — it already defaults
	// to `.` at the schema level, same as `projects/new`, and an operator
	// can adjust it via a later action if this app really does need a
	// non-root build context.
	connectRepo: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'deploy_apps' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const parsed = connectRepoSchema.safeParse({ repoId: formData.get('repoId') });
		if (!parsed.success) return fail(400, { message: 'Pick a repository' });

		const [repoRow] = await db
			.select({ id: repo.id, sourceId: repo.sourceId })
			.from(repo)
			.where(eq(repo.id, parsed.data.repoId));
		if (!repoRow) return fail(400, { message: 'That repository is no longer available' });

		const [sourceRow] = await db
			.select({ organizationId: source.organizationId })
			.from(source)
			.where(eq(source.id, repoRow.sourceId));
		if (!sourceRow || sourceRow.organizationId !== organization.id) {
			return fail(400, { message: 'That repository is no longer available' });
		}

		await db.update(app).set({ repoId: repoRow.id }).where(eq(app.id, appRow.id));

		return { success: true };
	},
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
	},

	stop: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'deploy_apps' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const targetDeploymentId = formData.get('deploymentId');
		if (typeof targetDeploymentId !== 'string' || !targetDeploymentId) {
			return fail(400, { message: 'Missing deployment to stop' });
		}

		const [targetDeployment] = await db
			.select()
			.from(deployment)
			.where(and(eq(deployment.id, targetDeploymentId), eq(deployment.appId, appRow.id)));
		if (!targetDeployment) return fail(404, { message: 'Deployment not found' });

		// Stopping only makes sense for the deployment actually serving traffic
		// right now — anything else either isn't running (nothing to stop) or
		// is mid-rollout (stopping it here would fight the state machine).
		if (targetDeployment.status !== 'running') {
			return fail(400, { message: 'Only the currently running deployment can be stopped' });
		}

		await stopDeployment(targetDeployment);

		return { success: true };
	},

	addCustomDomain: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_domains' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const hostname = formData.get('hostname')?.toString().trim().toLowerCase();
		if (!hostname || !HOSTNAME_PATTERN.test(hostname)) {
			return fail(400, { message: 'Enter a valid domain name (e.g. app.yourcompany.com)' });
		}

		try {
			await db.insert(domain).values({ appId: appRow.id, hostname, isDefaultSubdomain: false });
		} catch {
			// Unique constraint on `domain.hostname` — already attached (to this
			// app or another one entirely).
			return fail(400, { message: 'That domain is already in use' });
		}

		return { success: true };
	},

	verifyCustomDomain: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_domains' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const domainId = formData.get('domainId');
		if (typeof domainId !== 'string' || !domainId) return fail(400, { message: 'Missing domain' });

		const [domainRow] = await db
			.select()
			.from(domain)
			.where(and(eq(domain.id, domainId), eq(domain.appId, appRow.id)));
		if (!domainRow || domainRow.isDefaultSubdomain)
			return fail(404, { message: 'Domain not found' });

		const [defaultDomainRow] = await db
			.select()
			.from(domain)
			.where(and(eq(domain.appId, appRow.id), eq(domain.isDefaultSubdomain, true)));
		if (!defaultDomainRow) {
			return fail(400, { message: 'App has no default subdomain to verify against' });
		}

		const verified = await verifyCnameTarget(domainRow.hostname, defaultDomainRow.hostname);
		if (!verified) {
			return fail(400, {
				message: `DNS not verified yet — add a CNAME record for ${domainRow.hostname} pointing to ${defaultDomainRow.hostname}`
			});
		}

		await db.update(domain).set({ verifiedAt: new Date() }).where(eq(domain.id, domainRow.id));
		await refreshProxyConfigForApp(appRow.id);

		return { success: true };
	},

	removeCustomDomain: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_domains' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const domainId = formData.get('domainId');
		if (typeof domainId !== 'string' || !domainId) return fail(400, { message: 'Missing domain' });

		const [domainRow] = await db
			.select()
			.from(domain)
			.where(and(eq(domain.id, domainId), eq(domain.appId, appRow.id)));
		if (!domainRow || domainRow.isDefaultSubdomain) {
			return fail(400, { message: 'Cannot remove the default subdomain' });
		}

		await db.delete(domain).where(eq(domain.id, domainRow.id));
		await refreshProxyConfigForApp(appRow.id);

		return { success: true };
	},

	attachVolume: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_env' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const name = formData.get('name')?.toString().trim().toLowerCase();
		const mountPath = formData.get('mountPath')?.toString().trim();

		if (!name || !VOLUME_NAME_PATTERN.test(name)) {
			return fail(400, {
				message: 'Volume name must be lowercase letters, numbers, and hyphens (max 32 chars)'
			});
		}
		if (!mountPath || !MOUNT_PATH_PATTERN.test(mountPath)) {
			return fail(400, { message: 'Enter an absolute mount path (e.g. /data)' });
		}

		// Volumes are host-scoped (task 07's Notes) — attaching one commits
		// the app to a host the same way a deploy does, via the same
		// fallback-to-first-host assignment `deploy` already uses.
		const hostRow = await resolveHostForApp(appRow, organization.id);
		if (!hostRow) return fail(400, { message: 'No host available — add a host first' });

		try {
			await db.insert(volume).values({ appId: appRow.id, hostId: hostRow.id, name, mountPath });
		} catch {
			// Unique constraint on (appId, name).
			return fail(400, { message: 'This app already has a volume with that name' });
		}

		return { success: true };
	},

	removeVolume: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_env' });

		const [appRow] = await db
			.select()
			.from(app)
			.where(and(eq(app.id, event.params.appId), eq(app.organizationId, organization.id)));
		if (!appRow) return fail(404, { message: 'App not found' });

		const formData = await event.request.formData();
		const volumeId = formData.get('volumeId');
		if (typeof volumeId !== 'string' || !volumeId) return fail(400, { message: 'Missing volume' });

		const [volumeRow] = await db
			.select()
			.from(volume)
			.where(and(eq(volume.id, volumeId), eq(volume.appId, appRow.id)));
		if (!volumeRow) return fail(404, { message: 'Volume not found' });

		// Detaches the declaration only — the underlying Podman volume and
		// its data are left alone on the host (task 07's Notes: no backup or
		// destructive lifecycle tooling in scope here).
		await db.delete(volume).where(eq(volume.id, volumeRow.id));

		return { success: true };
	}
};
