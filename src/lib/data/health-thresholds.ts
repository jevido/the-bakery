/**
 * Shared health-threshold and status-color helpers for the hardware
 * dashboard (Phase 20) — built once so the Host Detail page's charts (task
 * 06), the Hosts list card coloring (task 09), and the dashboard's alerts
 * panel (task 11) all agree on what "warning" and "critical" mean for the
 * same metric value, reusing this codebase's existing status-dot color
 * palette (`statusMeta` in `./bakery.ts`) rather than inventing new colors.
 * Client-safe (no `$lib/server` imports) — usable directly from components.
 */

export const WARNING_THRESHOLD = 80;
export const CRITICAL_THRESHOLD = 95;

const HEALTHY_COLOR = '#3fb984'; // matches statusMeta.running's dot
const WARNING_COLOR = '#e0a83e'; // matches statusMeta.building's dot
const CRITICAL_COLOR = '#e5654b'; // matches statusMeta.failed's dot

/** Maps a percentage-based metric value (CPU%, mem%, disk%, swap%) to this dashboard's shared status color. */
export function thresholdColor(value: number): string {
	if (value >= CRITICAL_THRESHOLD) return CRITICAL_COLOR;
	if (value >= WARNING_THRESHOLD) return WARNING_COLOR;
	return HEALTHY_COLOR;
}

/**
 * Finer-grained than `computeHostStatus`'s online/offline/pending/revoked
 * (`$lib/server/hosts/status.ts`) — computed there (server-only, since it
 * depends on that module) and passed down as this plain classification for
 * components to color consistently.
 */
export type HostHealth = 'healthy' | 'stale' | 'offline';

export function hostHealthColor(health: HostHealth): string {
	switch (health) {
		case 'offline':
			return CRITICAL_COLOR;
		case 'stale':
			return WARNING_COLOR;
		case 'healthy':
			return HEALTHY_COLOR;
	}
}
