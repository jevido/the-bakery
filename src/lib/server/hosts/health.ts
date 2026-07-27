import { CHECKIN_INTERVAL_MS, computeHostStatus, type HostForStatus } from './status';
import type { HostHealth } from '$lib/data/health-thresholds';

// A host that's merely a check-in or two late isn't necessarily down --
// `computeHostStatus` only flips to 'offline' after missing 3 check-ins,
// which is the right bar for the Hosts list's binary online/offline badge.
// This finer 'stale' tier fires after just 1-2 missed intervals, early
// enough for the alerts panel (task 11) to flag a host that's *about to* go
// offline before it actually does.
const STALE_THRESHOLD_MS = CHECKIN_INTERVAL_MS * 1.5;

export function classifyHostHealth(host: HostForStatus): HostHealth {
	const status = computeHostStatus(host);
	if (status === 'offline' || status === 'revoked') return 'offline';
	if (status === 'pending' || !host.lastSeenAt) return 'healthy';
	return Date.now() - host.lastSeenAt.getTime() > STALE_THRESHOLD_MS ? 'stale' : 'healthy';
}
