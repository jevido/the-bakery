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

export function average(values: Array<number | null>): number | null {
	const nums = values.filter((v): v is number => v != null);
	if (nums.length === 0) return null;
	return nums.reduce((a, b) => a + b, 0) / nums.length;
}

export interface AggregatedMetricSample {
	ts: Date;
	cpuPct: number | null;
	memPct: number | null;
	netKBs: number | null;
}

/**
 * Combines samples across every host in an org into one fleet-wide series
 * (Phase 20 task 10's "All hosts" dashboard view). Buckets by minute, then
 * by host *within* each bucket first, so a host that happens to report more
 * samples in a given minute than another doesn't skew the fleet average
 * just by being chattier.
 *
 * Percentages (cpu/mem) are averaged across hosts — "how loaded is the
 * fleet on average" is the useful read; summing would blow past 100% with
 * more than one host. Network throughput is summed — it's additive across
 * machines (total fleet bandwidth), not something that makes sense to
 * average.
 */
export function aggregateAcrossHosts(samples: HostMetricSample[]): AggregatedMetricSample[] {
	const byBucket = new Map<number, Map<string, HostMetricSample[]>>();
	for (const sample of samples) {
		const bucketKey = Math.floor(sample.ts.getTime() / 60_000);
		let byHost = byBucket.get(bucketKey);
		if (!byHost) {
			byHost = new Map();
			byBucket.set(bucketKey, byHost);
		}
		const bucket = byHost.get(sample.hostId);
		if (bucket) bucket.push(sample);
		else byHost.set(sample.hostId, [sample]);
	}

	return [...byBucket.entries()]
		.sort(([a], [b]) => a - b)
		.map(([bucketKey, byHost]) => {
			const perHost = [...byHost.values()].map((hostSamples) => ({
				cpuPct: average(hostSamples.map((s) => s.cpuPct)),
				memPct: average(hostSamples.map((s) => s.memPct)),
				netKBs: average(
					hostSamples.map((s) => ((s.netRxBytesPerSec ?? 0) + (s.netTxBytesPerSec ?? 0)) / 1024)
				)
			}));

			return {
				ts: new Date(bucketKey * 60_000),
				cpuPct: average(perHost.map((h) => h.cpuPct)),
				memPct: average(perHost.map((h) => h.memPct)),
				netKBs: perHost.reduce((sum, h) => sum + (h.netKBs ?? 0), 0)
			};
		});
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
