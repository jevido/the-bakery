import { and, desc, eq, ne } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { app, build, deployment, envVar, host, hostCommand, volume } from '$lib/server/db/schema';
import {
	environmentFileContent,
	guildNetworkName,
	podmanVolumeName,
	publishedPort,
	quadletContent,
	versionedUnitName
} from './quadlet';
import { caddyfileContentForHost } from './proxy';

type App = typeof app.$inferSelect;
type Build = typeof build.$inferSelect;
type Host = typeof host.$inferSelect;
type Deployment = typeof deployment.$inferSelect;
type HostCommand = typeof hostCommand.$inferSelect;

/**
 * Kicks off the zero-downtime rollover state machine (task 06) for an app:
 * creates the `deployment` row and dispatches the first `deploy` command for
 * the new unit. The old unit (if any) is left running untouched — it's only
 * stopped once the new one reports healthy, in `handleCommandCompletion`.
 */
export async function startDeployment(params: {
	appRow: App;
	buildRow: Build;
	hostRow: Host;
	triggeredBy: string;
}): Promise<Deployment> {
	const { appRow, buildRow, hostRow, triggeredBy } = params;
	const imageRef = buildRow.imageRef;
	if (!imageRef) throw new Error('Build has no image to deploy');

	const [deploymentRow] = await db
		.insert(deployment)
		.values({
			appId: appRow.id,
			buildId: buildRow.id,
			hostId: hostRow.id,
			status: 'starting_new',
			triggeredBy
		})
		.returning();

	await dispatchNewUnitDeploy(deploymentRow, appRow, buildRow.commitSha, imageRef, hostRow.id);

	return deploymentRow;
}

async function dispatchNewUnitDeploy(
	deploymentRow: Deployment,
	appRow: App,
	commitSha: string,
	imageRef: string,
	hostId: string
) {
	const unitName = versionedUnitName(appRow.name, commitSha);
	const envVars = await db.select().from(envVar).where(eq(envVar.appId, appRow.id));
	const volumeRows = await db.select().from(volume).where(eq(volume.appId, appRow.id));

	await db.insert(hostCommand).values({
		hostId,
		deploymentId: deploymentRow.id,
		type: 'deploy',
		payload: {
			unitName,
			unitContent: quadletContent(appRow, imageRef, unitName, volumeRows),
			envFileContent: environmentFileContent(envVars),
			healthCheckPort: publishedPort(unitName),
			// The unit content already references this network (task 05), but
			// the agent still needs it out-of-band to ensure the network exists
			// *before* starting the unit — Podman doesn't auto-create a network
			// a container references, unlike the image it references.
			networkName: guildNetworkName(appRow.organizationId),
			// Same reasoning as networkName, task 07: the unit's `Volume=` lines
			// don't create the volumes they reference either.
			volumes: volumeRows.map((v) => ({
				name: podmanVolumeName(v.appId, v.name),
				mountPath: v.mountPath
			}))
		}
	});
}

/**
 * Advances a deployment's state machine in response to a hostCommand's
 * completion (called from the command-completion endpoint, task 04). Only
 * acts on completions that match the deployment's *current* status — a
 * stray/duplicate completion report for a step the deployment has already
 * moved past is a no-op rather than corrupting later state.
 */
export async function handleCommandCompletion(
	commandRow: HostCommand,
	outcome: 'succeeded' | 'failed'
): Promise<void> {
	const [deploymentRow] = await db
		.select()
		.from(deployment)
		.where(eq(deployment.id, commandRow.deploymentId));
	if (!deploymentRow) return;

	if (commandRow.type === 'deploy' && deploymentRow.status === 'starting_new') {
		if (outcome === 'failed') {
			await db
				.update(deployment)
				.set({ status: 'failed', finishedAt: new Date() })
				.where(eq(deployment.id, deploymentRow.id));
			// Best-effort cleanup in case the unit partially started before
			// failing (e.g. it started but failed its embedded health probe)
			// — the old version must never be touched, only the new one.
			if (isDeployPayload(commandRow.payload)) {
				await dispatchCleanupStop(deploymentRow, commandRow.payload.unitName);
			}
			return;
		}

		// Succeeded implies healthy too — the agent's `deploy` command only
		// reports success once its embedded health probe passes (task 05).
		// The proxy flip (Phase 05 task 03) has to land before the old unit is
		// touched, so nothing observes a gap where neither unit is reachable
		// at the app's hostname.
		if (!deploymentRow.hostId || !isDeployPayload(commandRow.payload)) return;

		await db
			.update(deployment)
			.set({ status: 'flipping_proxy' })
			.where(eq(deployment.id, deploymentRow.id));
		await dispatchConfigureProxy(
			deploymentRow,
			deploymentRow.hostId,
			deploymentRow.appId,
			commandRow.payload.unitName
		);
		return;
	}

	if (commandRow.type === 'configureProxy' && deploymentRow.status === 'flipping_proxy') {
		if (outcome === 'failed') {
			await db
				.update(deployment)
				.set({ status: 'failed', finishedAt: new Date() })
				.where(eq(deployment.id, deploymentRow.id));
			const newUnitName = await unitNameForDeployment(deploymentRow);
			if (newUnitName) await dispatchCleanupStop(deploymentRow, newUnitName);
			return;
		}

		const oldDeploymentRow = await findRunningDeployment(deploymentRow.appId, deploymentRow.id);
		const oldUnitName = oldDeploymentRow ? await unitNameForDeployment(oldDeploymentRow) : null;
		const newUnitName = await unitNameForDeployment(deploymentRow);

		// Redeploying the exact same build (same commit -> same deterministic
		// unit name, task 03) means the "old" unit *is* the unit that was just
		// (re)started — stopping it would tear down the deploy that just
		// succeeded rather than cleaning up a distinct previous version.
		if (!oldDeploymentRow || !oldUnitName || oldUnitName === newUnitName) {
			await db
				.update(deployment)
				.set({ status: 'running', finishedAt: new Date() })
				.where(eq(deployment.id, deploymentRow.id));
			return;
		}

		await db
			.update(deployment)
			.set({ status: 'stopping_old' })
			.where(eq(deployment.id, deploymentRow.id));
		await dispatchStopOld(deploymentRow, deploymentRow.hostId, oldUnitName, oldDeploymentRow.id);
		return;
	}

	if (commandRow.type === 'stop' && deploymentRow.status === 'stopping_old') {
		// The new unit is already healthy and serving on its own port
		// regardless of how cleanly the old one stopped, so a failure here
		// doesn't fail the deploy — it just leaves the old unit lingering
		// (Phase 16's reconciliation, not this handler, is what catches that
		// case now). Both writes land in one transaction: a crash between two
		// separate updates would otherwise strand the old row `running`
		// forever with nothing to ever notice, since reconciliation only acts
		// on state the DB itself already flags as wrong.
		await db.transaction(async (tx) => {
			await tx
				.update(deployment)
				.set({ status: 'running', finishedAt: new Date() })
				.where(eq(deployment.id, deploymentRow.id));

			const oldDeploymentId = isStopPayload(commandRow.payload)
				? commandRow.payload.oldDeploymentId
				: undefined;
			if (oldDeploymentId) {
				await tx
					.update(deployment)
					.set({ status: 'stopped', finishedAt: new Date() })
					.where(eq(deployment.id, oldDeploymentId));
			}
		});
		return;
	}

	// A `stop` completion for a deployment tagged directly with its own id
	// (Phase 16 task 04's reconciliation, Phase 17 task 01's manual stop) —
	// as opposed to the `stopping_old` branch above, which is a *new*
	// deployment's own transition driven by stopping a *different* old one.
	// Excludes `failed`: `dispatchCleanupStop` (used when a brand-new
	// deployment's own `deploy`/`configureProxy` step fails, above) also
	// dispatches a `stop` tagged to that same deployment's id, but only
	// after it's already `failed` — a deployment that never successfully
	// ran must stay `failed`, not get overwritten to `stopped`, which would
	// misrepresent what actually happened. Idempotent otherwise (a no-op if
	// already `stopped`), matching this handler's existing
	// optimistic-regardless-of-outcome philosophy.
	if (
		commandRow.type === 'stop' &&
		deploymentRow.status !== 'stopping_old' &&
		deploymentRow.status !== 'failed'
	) {
		await db
			.update(deployment)
			.set({ status: 'stopped', finishedAt: new Date() })
			.where(eq(deployment.id, deploymentRow.id));
		return;
	}
}

async function findRunningDeployment(
	appId: string,
	excludeDeploymentId?: string
): Promise<Deployment | undefined> {
	const conditions = [eq(deployment.appId, appId), eq(deployment.status, 'running')];
	if (excludeDeploymentId) conditions.push(ne(deployment.id, excludeDeploymentId));

	const [row] = await db
		.select()
		.from(deployment)
		.where(and(...conditions))
		.orderBy(desc(deployment.startedAt))
		.limit(1);
	return row;
}

async function unitNameForDeployment(deploymentRow: Deployment): Promise<string | null> {
	const [buildRow] = await db.select().from(build).where(eq(build.id, deploymentRow.buildId));
	const [appRow] = await db.select().from(app).where(eq(app.id, deploymentRow.appId));
	if (!buildRow || !appRow) return null;
	return versionedUnitName(appRow.name, buildRow.commitSha);
}

async function dispatchStopOld(
	newDeploymentRow: Deployment,
	hostId: string | null,
	unitName: string,
	oldDeploymentId: string
) {
	if (!hostId) return;

	await db.insert(hostCommand).values({
		hostId,
		deploymentId: newDeploymentRow.id,
		type: 'stop',
		// `oldDeploymentId` rides along so the `stop` completion handler above
		// can mark the *old* deployment's own row `stopped` once this is
		// confirmed done — the hostCommand itself stays tagged to the new
		// deployment (unchanged), since that's still what drives its own
		// `stopping_old` -> `running` transition. Safe to add: the agent's
		// payload decoding only reads `unitName` for a `stop` command and
		// silently ignores fields it doesn't recognize.
		payload: { unitName, oldDeploymentId }
	});
}

function isDeployPayload(payload: unknown): payload is { unitName: string } {
	return (
		typeof payload === 'object' &&
		payload !== null &&
		typeof (payload as { unitName?: unknown }).unitName === 'string'
	);
}

function isStopPayload(
	payload: unknown
): payload is { unitName: string; oldDeploymentId?: string } {
	return isDeployPayload(payload);
}

async function dispatchCleanupStop(deploymentRow: Deployment, unitName: string) {
	if (!deploymentRow.hostId) return;

	await db.insert(hostCommand).values({
		hostId: deploymentRow.hostId,
		deploymentId: deploymentRow.id,
		type: 'stop',
		payload: { unitName }
	});
}

/**
 * Dispatches the real traffic-flip step (Phase 05 task 03): the full desired
 * Caddyfile for the whole host, with this one app pointed at its new,
 * not-yet-`running` unit while every other app on the host keeps routing to
 * whatever it's already running.
 */
async function dispatchConfigureProxy(
	deploymentRow: Deployment,
	hostId: string,
	appId: string,
	newUnitName: string
) {
	const caddyfileContent = await caddyfileContentForHost(hostId, { appId, unitName: newUnitName });

	await db.insert(hostCommand).values({
		hostId,
		deploymentId: deploymentRow.id,
		type: 'configureProxy',
		payload: { caddyfileContent }
	});
}

/**
 * Re-pushes the host's proxy config as-is (no flip in progress, so no
 * override needed) — used when a domain is added/verified/removed outside
 * of a deploy (Phase 05 task 04). `hostCommand.deploymentId` is a hard
 * `NOT NULL` FK with no "not tied to a deployment" concept of its own, so
 * this rides on the app's current running deployment; `handleCommandCompletion`
 * only acts on a `configureProxy` completion when that deployment's status is
 * `flipping_proxy`, so tagging it onto an already-`running` deployment here
 * is inert as far as the rollover state machine is concerned. A no-op if the
 * app was never deployed or has nothing currently running — the domain
 * change still lands in Postgres and takes effect on the next real deploy.
 */
export async function refreshProxyConfigForApp(appId: string): Promise<void> {
	const [appRow] = await db.select().from(app).where(eq(app.id, appId));
	if (!appRow?.hostId) return;

	const runningDeploymentRow = await findRunningDeployment(appId);
	if (!runningDeploymentRow) return;

	const caddyfileContent = await caddyfileContentForHost(appRow.hostId);

	await db.insert(hostCommand).values({
		hostId: appRow.hostId,
		deploymentId: runningDeploymentRow.id,
		type: 'configureProxy',
		payload: { caddyfileContent }
	});
}
