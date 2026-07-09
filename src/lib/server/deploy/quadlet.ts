import { app, envVar } from '$lib/server/db/schema';
import { decrypt } from '$lib/server/secrets/crypto';

type App = typeof app.$inferSelect;
type EnvVar = typeof envVar.$inferSelect;

/**
 * Bakery apps all listen on this port inside the container. There's no
 * per-app port config yet (Phase 05 may add one alongside the reverse
 * proxy work) — this is a fixed v1 convention, matching what the app
 * detail page's mock already assumed.
 */
export const APP_CONTAINER_PORT = 3000;

/**
 * Zero-downtime deploys (task 06) briefly run the old and new units side by
 * side, so each deploy needs its own unit name rather than reusing
 * `<app>.container` — keying off the build's short commit SHA keeps two
 * consecutive deploys of the same app from colliding on the same unit name.
 */
export function versionedUnitName(appName: string, commitSha: string): string {
	return `${appName}-${commitSha.slice(0, 7)}`;
}

export function environmentFilePath(unitName: string): string {
	return `/etc/bakery/${unitName}.env`;
}

/**
 * `.env`-format decrypted key=value pairs. Sent to the agent alongside the
 * unit content (task 04/05) and written to `environmentFilePath` on the
 * host — never written to disk or logged on the control plane.
 */
export function environmentFileContent(envVars: EnvVar[]): string {
	return envVars.map((e) => `${e.key}=${decrypt(e.valueCiphertext)}`).join('\n');
}

/**
 * Real Quadlet `.container` unit content for a deploy — the actual payload
 * sent to and executed by agents (task 05), replacing `bakery.ts`'s mock
 * `quadletContent()`.
 */
export function quadletContent(appRow: App, imageRef: string, unitName: string): string {
	const port = APP_CONTAINER_PORT;
	return `[Unit]
Description=${appRow.name}
After=network-online.target

[Container]
Image=${imageRef}
PublishPort=${port}:${port}
EnvironmentFile=${environmentFilePath(unitName)}
AutoUpdate=registry

[Service]
Restart=always
TimeoutStartSec=90

[Install]
WantedBy=default.target`;
}
