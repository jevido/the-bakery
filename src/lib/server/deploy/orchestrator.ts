import { and, desc, eq, ne } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { app, build, deployment, envVar, host, hostCommand } from '$lib/server/db/schema';
import {
	environmentFileContent,
	publishedPort,
	quadletContent,
	versionedUnitName
} from './quadlet';

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

	await db.insert(hostCommand).values({
		hostId,
		deploymentId: deploymentRow.id,
		type: 'deploy',
		payload: {
			unitName,
			unitContent: quadletContent(appRow, imageRef, unitName),
			envFileContent: environmentFileContent(envVars),
			healthCheckPort: publishedPort(unitName)
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
			await dispatchCleanupStop(deploymentRow, commandRow);
			return;
		}

		// Succeeded implies healthy too — the agent's `deploy` command only
		// reports success once its embedded health probe passes (task 05).
		const oldDeploymentRow = await findRunningDeployment(deploymentRow.appId, deploymentRow.id);
		if (!oldDeploymentRow) {
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
		await dispatchStopOld(deploymentRow, oldDeploymentRow);
		return;
	}

	if (commandRow.type === 'stop' && deploymentRow.status === 'stopping_old') {
		// The new unit is already healthy and serving on its own port
		// regardless of how cleanly the old one stopped, so a failure here
		// doesn't fail the deploy — it just leaves the old unit lingering,
		// which task 08's history view can surface for manual cleanup.
		await db
			.update(deployment)
			.set({ status: 'running', finishedAt: new Date() })
			.where(eq(deployment.id, deploymentRow.id));
		return;
	}
}

async function findRunningDeployment(
	appId: string,
	excludeDeploymentId: string
): Promise<Deployment | undefined> {
	const [row] = await db
		.select()
		.from(deployment)
		.where(
			and(
				eq(deployment.appId, appId),
				eq(deployment.status, 'running'),
				ne(deployment.id, excludeDeploymentId)
			)
		)
		.orderBy(desc(deployment.startedAt))
		.limit(1);
	return row;
}

async function dispatchStopOld(newDeploymentRow: Deployment, oldDeploymentRow: Deployment) {
	if (!newDeploymentRow.hostId) return;

	const [oldBuildRow] = await db.select().from(build).where(eq(build.id, oldDeploymentRow.buildId));
	const [appRow] = await db.select().from(app).where(eq(app.id, newDeploymentRow.appId));
	if (!oldBuildRow || !appRow) return;

	const oldUnitName = versionedUnitName(appRow.name, oldBuildRow.commitSha);
	await db.insert(hostCommand).values({
		hostId: newDeploymentRow.hostId,
		deploymentId: newDeploymentRow.id,
		type: 'stop',
		payload: { unitName: oldUnitName }
	});
}

function isDeployPayload(payload: unknown): payload is { unitName: string } {
	return (
		typeof payload === 'object' &&
		payload !== null &&
		typeof (payload as { unitName?: unknown }).unitName === 'string'
	);
}

async function dispatchCleanupStop(deploymentRow: Deployment, failedDeployCommand: HostCommand) {
	if (!deploymentRow.hostId) return;
	if (!isDeployPayload(failedDeployCommand.payload)) return;

	await db.insert(hostCommand).values({
		hostId: deploymentRow.hostId,
		deploymentId: deploymentRow.id,
		type: 'stop',
		payload: { unitName: failedDeployCommand.payload.unitName }
	});
}
