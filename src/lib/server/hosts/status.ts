/**
 * No background sweeper exists yet, so a host's effective status is computed
 * at read time from lastSeenAt rather than trusting the stored `status`
 * column, which is only ever updated by the check-in endpoint and would
 * otherwise go stale the moment an agent stops checking in. Revisit if/when
 * a real background job system is introduced.
 */
export const CHECKIN_INTERVAL_MS = 45_000;
const MISSED_INTERVALS_THRESHOLD = 3;
const OFFLINE_THRESHOLD_MS = CHECKIN_INTERVAL_MS * MISSED_INTERVALS_THRESHOLD;

export type HostStatus = 'revoked' | 'pending' | 'offline' | 'online';

export interface HostForStatus {
	revokedAt: Date | null;
	lastSeenAt: Date | null;
}

export function computeHostStatus(host: HostForStatus): HostStatus {
	if (host.revokedAt) return 'revoked';
	if (!host.lastSeenAt) return 'pending';
	if (Date.now() - host.lastSeenAt.getTime() > OFFLINE_THRESHOLD_MS) return 'offline';
	return 'online';
}
