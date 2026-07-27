<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { formatDuration, formatRelativeTime } from '$lib/utils';
	import { thresholdColor, hostHealthColor } from '$lib/data/health-thresholds';
	import MetricChart from '$lib/components/bakery/MetricChart.svelte';
	import MetricRangePicker from '$lib/components/bakery/MetricRangePicker.svelte';
	import { provideMetricChartCrosshair } from '$lib/components/bakery/metric-chart-crosshair.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	provideMetricChartCrosshair();

	const guildId = $derived(page.params.guild ?? '');

	function selectRange(id: string) {
		goto(`?range=${id}`, { keepFocus: true, noScroll: true, replaceState: true });
	}

	type Sample = (typeof data.history)[number];
	type PctField = 'cpuPct' | 'memPct' | 'diskPct' | 'swapPct';
	type ThroughputField =
		'diskReadBytesPerSec' | 'diskWriteBytesPerSec' | 'netRxBytesPerSec' | 'netTxBytesPerSec';

	function series(field: PctField | 'loadAvg1') {
		return data.history.map((s) => ({ ts: s.ts, value: s[field] ?? 0 }));
	}

	// Disk read/write and network rx/tx are shown combined (summed) rather
	// than as separate series -- MetricChart renders a single value per
	// point, and this matches the same "combined throughput" convention
	// already established for the guild dashboard's Network I/O card.
	function combinedSeries(fieldA: ThroughputField, fieldB: ThroughputField) {
		return data.history.map((s: Sample) => ({
			ts: s.ts,
			value: ((s[fieldA] ?? 0) + (s[fieldB] ?? 0)) / 1024
		}));
	}
	function combinedLatestKBs(a: number | null | undefined, b: number | null | undefined): string {
		if (a == null && b == null) return '—';
		return (((a ?? 0) + (b ?? 0)) / 1024).toFixed(0);
	}

	const latest = $derived(data.latestSample);

	function fmtPct(v: number | null | undefined): string {
		return v != null ? v.toFixed(1) : '—';
	}

	const metricCards = $derived([
		{
			key: 'cpu',
			label: 'CPU',
			unit: '%',
			big: fmtPct(latest?.cpuPct),
			color: thresholdColor(latest?.cpuPct ?? 0),
			series: series('cpuPct')
		},
		{
			key: 'mem',
			label: 'Memory',
			unit: '%',
			big: fmtPct(latest?.memPct),
			color: thresholdColor(latest?.memPct ?? 0),
			series: series('memPct')
		},
		{
			key: 'disk',
			label: 'Disk',
			unit: '%',
			big: fmtPct(latest?.diskPct),
			color: thresholdColor(latest?.diskPct ?? 0),
			series: series('diskPct')
		},
		{
			key: 'swap',
			label: 'Swap',
			unit: '%',
			big: fmtPct(latest?.swapPct),
			color: thresholdColor(latest?.swapPct ?? 0),
			series: series('swapPct')
		},
		{
			key: 'load',
			label: 'Load average',
			unit: '',
			big: latest?.loadAvg1 != null ? latest.loadAvg1.toFixed(2) : '—',
			color: '#5b9dd9',
			series: series('loadAvg1')
		},
		{
			key: 'diskio',
			label: 'Disk I/O',
			unit: ' KB/s',
			big: combinedLatestKBs(latest?.diskReadBytesPerSec, latest?.diskWriteBytesPerSec),
			color: '#e0a83e',
			series: combinedSeries('diskReadBytesPerSec', 'diskWriteBytesPerSec')
		},
		{
			key: 'netio',
			label: 'Network I/O',
			unit: ' KB/s',
			big: combinedLatestKBs(latest?.netRxBytesPerSec, latest?.netTxBytesPerSec),
			color: '#8b5cf6',
			series: combinedSeries('netRxBytesPerSec', 'netTxBytesPerSec')
		}
	]);

	function formatUptime(seconds: number | null | undefined): string {
		return seconds != null ? formatDuration(seconds * 1000) : '—';
	}

	const statusDotColor = $derived(hostHealthColor(data.host.health));
</script>

<svelte:head><title>{data.host.name} · Hosts — The Bakery</title></svelte:head>

<div class="px-7 py-[22px]">
	<button
		onclick={() => goto(`/${guildId}/deploy/hosts`)}
		class="inline-flex items-center gap-[6px] text-[12.5px] text-[var(--tx-2)] cursor-pointer mb-[14px]"
	>
		<svg
			width="14"
			height="14"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2.2"><path d="M15 18l-6-6 6-6" /></svg
		>
		All hosts
	</button>

	<!-- Identity header -->
	<div class="flex items-start justify-between gap-4 mb-5">
		<div class="flex items-center gap-[14px] min-w-0">
			<div
				class="size-[46px] rounded-[12px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0"
			>
				<svg
					width="20"
					height="20"
					viewBox="0 0 24 24"
					fill="none"
					stroke="var(--tx-2)"
					stroke-width="1.6"
				>
					<rect x="2" y="4" width="20" height="7" rx="1.5" />
					<rect x="2" y="13" width="20" height="7" rx="1.5" />
					<circle cx="6" cy="7.5" r="1" />
					<circle cx="6" cy="16.5" r="1" />
				</svg>
			</div>
			<div class="min-w-0">
				<div class="flex items-center gap-[10px]">
					<div class="font-heading font-bold text-[22px] tracking-[-0.01em] truncate">
						{data.host.name}
					</div>
					<div class="flex items-center gap-[6px] px-[9px] py-[3px] rounded-[20px] bg-white/[0.04]">
						<div class="size-[7px] rounded-full" style:background={statusDotColor}></div>
						<span class="text-[11.5px] text-[var(--tx-2)]">{data.host.computedStatus}</span>
					</div>
				</div>
				<div class="text-[12.5px] text-[var(--tx-3)] mt-[3px]">
					{data.host.location ?? '—'} · {data.host.spec ?? '—'}
				</div>
			</div>
		</div>

		<div class="flex items-center gap-[10px] shrink-0">
			<div class="text-right">
				<div class="text-[11px] text-[var(--tx-3)]">Uptime</div>
				<div class="font-mono-jb text-[13px] text-[var(--tx)]">
					{formatUptime(data.latestSample?.uptimeSeconds)}
				</div>
			</div>
			<div class="w-px h-[26px] bg-[var(--line)]"></div>
			<div class="text-right">
				<div class="text-[11px] text-[var(--tx-3)]">Last seen</div>
				<div class="font-mono-jb text-[13px] text-[var(--tx)]">
					{data.host.lastSeenAt ? formatRelativeTime(new Date(data.host.lastSeenAt)) : 'never'}
				</div>
			</div>
			<div class="w-px h-[26px] bg-[var(--line)]"></div>
			<div class="text-right">
				<div class="text-[11px] text-[var(--tx-3)]">Agent</div>
				<div class="font-mono-jb text-[13px] text-[var(--tx)]">
					{data.host.agentVersion ? `v${data.host.agentVersion}` : '—'}
				</div>
			</div>
			<div class="w-px h-[26px] bg-[var(--line)]"></div>
			<div class="text-right">
				<div class="text-[11px] text-[var(--tx-3)]">Podman</div>
				<div class="font-mono-jb text-[13px] text-[var(--tx)]">
					{data.latestSample?.podmanVersion ? `v${data.latestSample.podmanVersion}` : '—'}
				</div>
			</div>
			<div class="w-px h-[26px] bg-[var(--line)]"></div>
			<div class="text-right">
				<div class="text-[11px] text-[var(--tx-3)]">Apps</div>
				<div class="font-mono-jb text-[13px] text-[var(--tx)]">{data.appsCount}</div>
			</div>
		</div>
	</div>

	<!-- Tab bar (Containers/Activity added in later tasks) -->
	<div class="flex gap-0.5 border-b border-b-[var(--line)] mb-[18px]">
		<span
			class="px-[14px] py-[10px] text-[13.5px] font-semibold text-[var(--grn)] border-b-2 border-b-[var(--grn)] -mb-px"
			>Overview</span
		>
	</div>

	<!-- Range picker -->
	<div class="flex items-center justify-end mb-[10px]">
		<MetricRangePicker value={data.range} onchange={selectRange} />
	</div>

	<!-- Metrics grid -->
	<div class="grid grid-cols-4 gap-[14px]">
		{#each metricCards as m (m.key)}
			<div
				class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] pt-4 px-[18px] overflow-hidden"
				style:color={m.color}
			>
				<div class="flex items-center justify-between">
					<span class="text-[13px] font-semibold text-[var(--tx-2)] whitespace-nowrap"
						>{m.label}</span
					>
				</div>
				<div class="flex items-baseline gap-[7px] mt-[10px]">
					<span class="font-heading font-bold text-[26px] leading-none text-[var(--tx)]"
						>{m.big}</span
					>
					<span class="text-[12.5px] text-[var(--tx-3)]">{m.unit}</span>
				</div>
				<div class="h-[62px] mt-[10px] -mx-[18px] w-[calc(100%+36px)]">
					<MetricChart series={m.series} color={m.color} label={m.label} unit={m.unit} />
				</div>
			</div>
		{/each}
	</div>
</div>
