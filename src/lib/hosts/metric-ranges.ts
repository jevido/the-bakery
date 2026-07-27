/**
 * The historical range picker for metric charts (Phase 06 task 03, extended
 * to the full 1h/6h/24h/7d/30d set in Phase 20 task 04 once retention was
 * bumped to 30 days to make the longer ranges meaningful). Shared between
 * every page's server `load` (window calculation) and its client component
 * (range picker labels/active state) — a single source of truth for the
 * range ids so a client-sent `range` query param always matches something
 * the server actually knows how to compute. Not host-specific despite the
 * module's path — used for both `hostMetricSample` and `appMetricSample`
 * queries.
 */
export interface MetricRange {
	id: '1h' | '6h' | '24h' | '7d' | '30d';
	label: string;
	windowMs: number;
}

const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;

export const METRIC_RANGES: MetricRange[] = [
	{ id: '1h', label: '1h', windowMs: HOUR_MS },
	{ id: '6h', label: '6h', windowMs: 6 * HOUR_MS },
	{ id: '24h', label: '24h', windowMs: 24 * HOUR_MS },
	{ id: '7d', label: '7d', windowMs: 7 * DAY_MS },
	{ id: '30d', label: '30d', windowMs: 30 * DAY_MS }
];

export const DEFAULT_METRIC_RANGE: MetricRange['id'] = '1h';

export function isMetricRangeId(value: string | null): value is MetricRange['id'] {
	return METRIC_RANGES.some((r) => r.id === value);
}

/** The earliest `ts` a query should include for the given range, computed from `Date.now()` at call time. */
export function rangeStartDate(id: MetricRange['id']): Date {
	const range = METRIC_RANGES.find((r) => r.id === id)!;
	return new Date(Date.now() - range.windowMs);
}
