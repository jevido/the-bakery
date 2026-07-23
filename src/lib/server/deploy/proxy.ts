import { and, desc, eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { app, build, deployment, domain } from '$lib/server/db/schema';
import { publishedPort, versionedUnitName } from './quadlet';

/**
 * Caddy site block routing one hostname to a local upstream port — matches
 * the app detail page's "Domains & Proxy" preview exactly, since this is now
 * the real content behind that preview rather than just a mock of it.
 */
export function caddySiteBlock(hostname: string, upstreamPort: number): string {
	return `${hostname} {
	reverse_proxy 127.0.0.1:${upstreamPort}
	encode zstd gzip
}`;
}

const CADDYFILE_HEADER = `# Managed by bakery-agent. Site blocks for deployed apps and custom
# domains are added here automatically — avoid hand-editing.`;

// Static site blocks for the control plane's own domain and the private
// registry's public domain (Phase 08 task 04) — only relevant on the one
// Bakery host that's *also* running the control plane itself (jevido.app).
// All four are optional/unset by default: nothing changes for every other
// host in the system, which have nothing listening on these local ports.
//
// Gated on `CONTROL_PLANE_HOST_ID` matching the `hostId` this function is
// generating for, not just on the domain/port vars being set — env vars are
// process-global, but `caddyfileContentForHost` runs for *every* Bakery
// host's regeneration, and a future second host must never also claim to
// serve jevido.app just because these vars happen to be set on the shared
// control-plane process.
// Matches compose.yaml's dev registry service *and* agent/registry.go's
// registryPort constant (the native production Quadlet unit, Phase 08 task
// 10) — fixed by convention across both, not meant to vary per install, so
// not worth its own env var.
const REGISTRY_LOCAL_PORT = 5050;

// Read fresh on every call, not cached at module load — matches this
// codebase's existing convention for env-driven config (e.g. `registry.ts`'s
// `requireEnv`), and these are process-lifetime-stable values anyway so
// there's no real cost to re-reading them.
function staticSiteBlocks(hostId: string): string[] {
	const controlPlaneHostId = process.env.CONTROL_PLANE_HOST_ID;
	if (!controlPlaneHostId || hostId !== controlPlaneHostId) return [];

	const blocks: string[] = [];
	const controlPlaneDomain = process.env.CONTROL_PLANE_DOMAIN;
	const controlPlanePort = process.env.CONTROL_PLANE_PORT;
	if (controlPlaneDomain && controlPlanePort) {
		blocks.push(caddySiteBlock(controlPlaneDomain, Number(controlPlanePort)));
	}
	const registryPublicDomain = process.env.BAKERY_REGISTRY_PUBLIC_DOMAIN;
	if (registryPublicDomain) {
		blocks.push(caddySiteBlock(registryPublicDomain, REGISTRY_LOCAL_PORT));
	}
	return blocks;
}

/**
 * Full desired Caddyfile for every app scheduled on `hostId`, regenerated
 * from scratch each time rather than patched incrementally — Postgres stays
 * the single source of truth for proxy config, with no drift possible
 * between it and what's actually written/loaded on the host.
 *
 * `flipOverride` lets a deploy in flight (task 06's `flipping_proxy` step)
 * point one specific app at its *new*, not-yet-`running` unit while every
 * other app on the host keeps routing to its last-known-running one.
 */
export async function caddyfileContentForHost(
	hostId: string,
	flipOverride?: { appId: string; unitName: string }
): Promise<string> {
	const apps = await db.select().from(app).where(eq(app.hostId, hostId));

	const blocks: string[] = [...staticSiteBlocks(hostId)];
	for (const appRow of apps) {
		const unitName =
			appRow.id === flipOverride?.appId
				? flipOverride.unitName
				: await runningUnitName(appRow.id, appRow.name);
		if (!unitName) continue; // never deployed, or nothing currently running

		const domainRows = await db.select().from(domain).where(eq(domain.appId, appRow.id));
		for (const domainRow of domainRows) {
			blocks.push(caddySiteBlock(domainRow.hostname, publishedPort(unitName)));
		}
	}

	return [CADDYFILE_HEADER, ...blocks].join('\n\n') + '\n';
}

/**
 * The unit name of an app's currently `running` deployment, or null if it's
 * never been deployed or has nothing running right now. Exported for reuse
 * by the check-in endpoint (Phase 06 task 01) to map reported per-container
 * stats back to the app they belong to.
 */
export async function runningUnitName(appId: string, appName: string): Promise<string | null> {
	const [row] = await db
		.select({ commitSha: build.commitSha })
		.from(deployment)
		.innerJoin(build, eq(build.id, deployment.buildId))
		.where(and(eq(deployment.appId, appId), eq(deployment.status, 'running')))
		.orderBy(desc(deployment.startedAt))
		.limit(1);
	if (!row) return null;

	return versionedUnitName(appName, row.commitSha);
}
