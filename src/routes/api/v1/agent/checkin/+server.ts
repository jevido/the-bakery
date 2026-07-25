import { json, error } from '@sveltejs/kit';
import { eq, and, inArray, lt } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import {
	host,
	hostMetricSample,
	hostCommand,
	volume,
	app,
	appMetricSample,
	appLogLine,
	deployment,
	build
} from '$lib/server/db/schema';
import { authenticateHost } from '$lib/server/agent/auth';
import { podmanVolumeName, versionedUnitName } from '$lib/server/deploy/quadlet';
import { runningUnitName, activeDeploymentUnitNames } from '$lib/server/deploy/proxy';
import { handleCommandCompletion } from '$lib/server/deploy/orchestrator';
import { redactSecrets, secretValuesForApp } from '$lib/server/secrets/redact';
import {
	checkinPayloadSchema,
	type CheckinResponse,
	type PendingCommand
} from '$lib/server/agent/protocol';

// Long enough that a command genuinely still in flight (agent picked it up,
// hasn't finished yet) is never mistaken for stuck; short enough that a
// truly orphaned one (agent crashed after claiming it, never calls back)
// doesn't block retries for too long.
const STALE_DELIVERED_COMMAND_MS = 10 * 60 * 1000;

export const POST: RequestHandler = async (event) => {
	const matchedHost = await authenticateHost(event.request);

	const body = await event.request.json().catch(() => null);
	const parsed = checkinPayloadSchema.safeParse(body);
	if (!parsed.success) {
		error(400, 'Invalid check-in payload');
	}

	const {
		cpuPct,
		memPct,
		diskPct,
		podmanVersion,
		containerCount,
		agentVersion,
		volumes,
		containers,
		logs
	} = parsed.data;

	await db.insert(hostMetricSample).values({
		hostId: matchedHost.id,
		cpuPct,
		memPct,
		diskPct,
		podmanVersion,
		containerCount
	});

	await db
		.update(host)
		.set({ lastSeenAt: new Date(), status: 'online', agentVersion })
		.where(eq(host.id, matchedHost.id));

	// Stale `delivered`-command reaper (Phase 16 task 06): a command the
	// agent claimed but never confirmed complete (crash, lost network) would
	// otherwise sit in `delivered` forever — nothing else ever revisits it.
	// Pre-existing gap (would already stall a normal deploy's own
	// `stopping_old` step if it happened), and more important now that
	// reconciliation (below) treats an outstanding `pending`/`delivered` stop
	// as "already being handled," which a permanently-stuck one would make
	// permanently true. Runs before reconciliation in this same check-in so
	// a command reaped just now can be retried immediately, not next cycle.
	const staleCommands = await db
		.select()
		.from(hostCommand)
		.where(
			and(
				eq(hostCommand.hostId, matchedHost.id),
				eq(hostCommand.status, 'delivered'),
				lt(hostCommand.deliveredAt, new Date(Date.now() - STALE_DELIVERED_COMMAND_MS))
			)
		);
	for (const stale of staleCommands) {
		await db
			.update(hostCommand)
			.set({
				status: 'failed',
				completedAt: new Date(),
				errorMessage: 'timed out waiting for agent to report completion'
			})
			.where(eq(hostCommand.id, stale.id));
		await handleCommandCompletion(stale, 'failed');
	}

	// The agent reports real Podman volume names (task 07), not the
	// user-declared logical ones — recompute the expected name per row
	// (rather than storing it) to find which report belongs to which
	// `volume` row, same pattern the rest of the codebase uses for
	// derived-not-stored names (`guildNetworkName`).
	if (volumes.length > 0) {
		const sizeByName = new Map(volumes.map((v) => [v.name, v.sizeBytes]));
		const volumeRows = await db.select().from(volume).where(eq(volume.hostId, matchedHost.id));

		for (const row of volumeRows) {
			const reportedSize = sizeByName.get(podmanVolumeName(row.appId, row.name));
			if (reportedSize === undefined) continue;

			await db
				.update(volume)
				.set({ sizeBytes: reportedSize, lastReportedAt: new Date() })
				.where(eq(volume.id, row.id));
		}
	}

	// Reported containers/logs are keyed by Quadlet unit name, not app id
	// (task 01/05) — recompute each app's currently expected running unit
	// name once and match both against it, same pattern as the volume
	// matching above. An app with nothing reported (stopped/removed) simply
	// gets no new rows, no error.
	if (containers.length > 0 || logs.length > 0) {
		const statByUnitName = new Map(containers.map((c) => [c.unitName, c]));
		const logsByUnitName = new Map<string, typeof logs>();
		for (const line of logs) {
			const bucket = logsByUnitName.get(line.unitName);
			if (bucket) bucket.push(line);
			else logsByUnitName.set(line.unitName, [line]);
		}

		const appRows = await db.select().from(app).where(eq(app.hostId, matchedHost.id));

		for (const appRow of appRows) {
			const unitName = await runningUnitName(appRow.id, appRow.name);
			if (unitName) {
				const stat = statByUnitName.get(unitName);
				if (stat) {
					await db.insert(appMetricSample).values({
						appId: appRow.id,
						hostId: matchedHost.id,
						cpuPct: stat.cpuPct,
						memBytes: stat.memBytes
					});
				}
			}

			// Logs are matched against *every* unit that could still be alive
			// for this app (Phase 09 task 02) — not just the single `running`
			// one metrics above use — so a new deployment's output (still
			// `starting_new`/`flipping_proxy`, before cutover) and an old one's
			// tail (still `stopping_old`, before its `stop` command completes)
			// both get captured instead of silently dropped just because
			// neither is *the* currently-running unit.
			const activeUnits = await activeDeploymentUnitNames(appRow.id, appRow.name);
			const matchedLines = [...activeUnits].flatMap(([activeUnitName, deploymentId]) =>
				(logsByUnitName.get(activeUnitName) ?? []).map((l) => ({ ...l, deploymentId }))
			);
			if (matchedLines.length) {
				// `ts` defaults to `now()`, which is constant for every row in one
				// INSERT (transaction-time, not per-row) — the live-tail viewer's
				// polling cursor (task 05's SSE endpoint) needs distinct
				// timestamps to preserve this batch's actual order, so stamp each
				// line 1ms apart explicitly instead of relying on the default.
				const baseTs = Date.now();
				// Redacted against the app's own declared secret env values (task
				// 03, Phase 07) before ever reaching the database — this is the
				// customer's own container output, which Bakery can't fully
				// control, but a value it handed the app itself is a known
				// quantity worth scrubbing on the way in.
				const secrets = await secretValuesForApp(appRow.id);
				await db.insert(appLogLine).values(
					matchedLines.map((l, i) => ({
						appId: appRow.id,
						hostId: matchedHost.id,
						deploymentId: l.deploymentId,
						ts: new Date(baseTs + i),
						message: redactSecrets(l.message, secrets)
					}))
				);
			}

			// Reconciliation (Phase 16 task 04): a container the agent reports
			// as genuinely still alive, but whose deployment the DB already
			// believes is `stopped`/`failed` — a stop that silently failed
			// earlier (no retry exists anywhere in the agent's own stop
			// execution). Deliberately narrow to just those two terminal
			// statuses: every other non-`running` status (`starting_new`,
			// `health_checking`, `flipping_proxy`, `stopping_old`) legitimately
			// has a live container as part of a normal, healthy rollout —
			// treating those as orphans would actively sabotage an in-progress
			// deploy, not fix anything.
			const appDeployments = await db
				.select({ id: deployment.id, status: deployment.status, commitSha: build.commitSha })
				.from(deployment)
				.innerJoin(build, eq(build.id, deployment.buildId))
				.where(eq(deployment.appId, appRow.id));

			for (const d of appDeployments) {
				if (d.status !== 'stopped' && d.status !== 'failed') continue;

				const orphanUnitName = versionedUnitName(appRow.name, d.commitSha);
				if (!statByUnitName.has(orphanUnitName)) continue;

				// Avoid re-dispatching every check-in cycle while a previous
				// attempt is still in flight — a failed stop leaves the
				// pending/delivered set immediately (the completion endpoint
				// updates hostCommand.status before handleCommandCompletion
				// runs), so this naturally retries on the very next check-in
				// rather than needing its own separate backoff logic.
				const [existingStop] = await db
					.select({ id: hostCommand.id })
					.from(hostCommand)
					.where(
						and(
							eq(hostCommand.deploymentId, d.id),
							eq(hostCommand.type, 'stop'),
							inArray(hostCommand.status, ['pending', 'delivered'])
						)
					)
					.limit(1);
				if (existingStop) continue;

				await db.insert(hostCommand).values({
					hostId: matchedHost.id,
					deploymentId: d.id,
					type: 'stop',
					payload: { unitName: orphanUnitName }
				});
			}
		}
	}

	// Atomically claim every still-pending command for this host in one
	// UPDATE ... RETURNING — avoids a second check-in racing this one from
	// delivering (and the agent double-executing) the same command.
	const delivered = await db
		.update(hostCommand)
		.set({ status: 'delivered', deliveredAt: new Date() })
		.where(and(eq(hostCommand.hostId, matchedHost.id), eq(hostCommand.status, 'pending')))
		.returning();

	const pendingCommands: PendingCommand[] = delivered.map((c) => ({
		id: c.id,
		type: c.type,
		payload: c.payload
	}));

	// Optional — absent whenever any of the three vars isn't set, same as
	// every other optional instance-level env var in this codebase. Without
	// this, an agent has no way to authenticate its own image pulls, which a
	// real (non-anonymous) registry requires.
	const registryHost = process.env.BAKERY_REGISTRY_HOST;
	const registryPullUsername = process.env.BAKERY_REGISTRY_PULL_USERNAME;
	const registryPullPassword = process.env.BAKERY_REGISTRY_PULL_PASSWORD;
	const registryAuth =
		registryHost && registryPullUsername && registryPullPassword
			? { host: registryHost, username: registryPullUsername, password: registryPullPassword }
			: undefined;

	return json({ pendingCommands, registryAuth } satisfies CheckinResponse);
};
