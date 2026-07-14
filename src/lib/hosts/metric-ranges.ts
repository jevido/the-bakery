/**
 * The historical range picker for host metric charts (Phase 06 task 03).
 * Shared between the overview page's server `load` (window calculation) and
 * its client component (range picker labels/active state) — a single
 * source of truth for the range ids so a client-sent `range` query param
 * always matches something the server actually knows how to compute.
 */
export interface HostMetricRange {
	id: '1h' | '6h' | '24h' | '48h';
	label: string;
	windowMs: number;
}

export const HOST_METRIC_RANGES: HostMetricRange[] = [
	{ id: '1h', label: '1h', windowMs: 60 * 60 * 1000 },
	{ id: '6h', label: '6h', windowMs: 6 * 60 * 60 * 1000 },
	{ id: '24h', label: '24h', windowMs: 24 * 60 * 60 * 1000 },
	{ id: '48h', label: '48h', windowMs: 48 * 60 * 60 * 1000 }
];

export const DEFAULT_HOST_METRIC_RANGE: HostMetricRange['id'] = '1h';

export function isHostMetricRangeId(value: string | null): value is HostMetricRange['id'] {
	return HOST_METRIC_RANGES.some((r) => r.id === value);
}
