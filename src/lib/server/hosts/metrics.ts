import type { hostMetricSample } from '$lib/server/db/schema';

type HostMetricSample = typeof hostMetricSample.$inferSelect;

/**
 * Buckets samples into 1-minute-aligned averages (Phase 06 task 03). The
 * agent checks in every ~45s (`CHECKIN_INTERVAL_MS`), so at any range this
 * lands close to a no-op — one raw sample per bucket, roughly — its real
 * job is capping point count at wide ranges (48h of raw ~45s samples is
 * ~3800 points) without a separate "should I downsample" branch per range.
 */
export function downsampleToMinuteBuckets(samples: HostMetricSample[]): HostMetricSample[] {
	const buckets = new Map<number, HostMetricSample[]>();
	for (const sample of samples) {
		const bucketKey = Math.floor(sample.ts.getTime() / 60_000);
		const bucket = buckets.get(bucketKey);
		if (bucket) bucket.push(sample);
		else buckets.set(bucketKey, [sample]);
	}

	return [...buckets.entries()]
		.sort(([a], [b]) => a - b)
		.map(([bucketKey, bucketSamples]) => averageBucket(bucketKey, bucketSamples));
}

function average(values: Array<number | null>): number | null {
	const nums = values.filter((v): v is number => v != null);
	if (nums.length === 0) return null;
	return nums.reduce((a, b) => a + b, 0) / nums.length;
}

// Non-numeric fields (podmanVersion, containerCount, id) come from the
// bucket's last sample, same as `uptimeSeconds` (Phase 20 task 01) — a
// monotonically increasing counter, so "last observed" is the meaningful
// value, not an average. Every other numeric field is a rate/intensity
// metric that's actually charted, so those are averaged.
function averageBucket(bucketKey: number, bucketSamples: HostMetricSample[]): HostMetricSample {
	const last = bucketSamples[bucketSamples.length - 1];
	return {
		...last,
		ts: new Date(bucketKey * 60_000),
		cpuPct: average(bucketSamples.map((s) => s.cpuPct)),
		memPct: average(bucketSamples.map((s) => s.memPct)),
		diskPct: average(bucketSamples.map((s) => s.diskPct)),
		swapPct: average(bucketSamples.map((s) => s.swapPct)),
		loadAvg1: average(bucketSamples.map((s) => s.loadAvg1)),
		diskReadBytesPerSec: average(bucketSamples.map((s) => s.diskReadBytesPerSec)),
		diskWriteBytesPerSec: average(bucketSamples.map((s) => s.diskWriteBytesPerSec)),
		netRxBytesPerSec: average(bucketSamples.map((s) => s.netRxBytesPerSec)),
		netTxBytesPerSec: average(bucketSamples.map((s) => s.netTxBytesPerSec))
	};
}
