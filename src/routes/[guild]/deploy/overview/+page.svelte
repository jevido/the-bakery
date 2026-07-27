<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { GUILD_RESOURCES, statusMeta } from '$lib/data/bakery';
	import { AreaChart } from 'layerchart';
	import { HOST_METRIC_RANGES } from '$lib/hosts/metric-ranges';
	import { formatBytes, formatRelativeTime } from '$lib/utils';

	const guildId = $derived(page.params.guild ?? '');
	const guildName = $derived(page.data.organization?.name ?? guildId);
	const resources = $derived(GUILD_RESOURCES[guildId]);
	const apps = $derived(resources?.apps ?? []);

	const primaryHost = $derived(page.data.primaryHost);
	const primaryHostLatestSample = $derived(page.data.primaryHostLatestSample);
	const primaryHostHistory = $derived(page.data.primaryHostHistory ?? []);

	type MetricSample = (typeof primaryHostHistory)[number];

	function toSeries(history: MetricSample[], field: 'cpuPct' | 'memPct') {
		return history.map((sample, i) => ({ i, value: sample[field] ?? 0 }));
	}

	// Combined rx+tx throughput in KB/s (Phase 20 task 01 — real agent
	// collection replaces what used to be a synthetic wobble() generator).
	function netKBs(sample: MetricSample): number {
		return ((sample.netRxBytesPerSec ?? 0) + (sample.netTxBytesPerSec ?? 0)) / 1024;
	}
	function toNetSeries(history: MetricSample[]) {
		return history.map((sample, i) => ({ i, value: netKBs(sample) }));
	}

	// First-vs-last sample in the loaded window — a real, if noisy, trend
	// indicator rather than a decorative fixed percentage.
	function trend(history: MetricSample[], field: 'cpuPct' | 'memPct') {
		if (history.length < 2) return null;
		const first = history[0][field] ?? 0;
		const last = history[history.length - 1][field] ?? 0;
		return last - first;
	}
	function netTrend(history: MetricSample[]) {
		if (history.length < 2) return null;
		return netKBs(history[history.length - 1]) - netKBs(history[0]);
	}
	function deltaLabel(delta: number | null, unit = '%') {
		if (delta === null) return '—';
		return `${delta >= 0 ? '▲' : '▼'} ${Math.abs(delta).toFixed(0)}${unit}`;
	}
	function deltaColor(delta: number | null) {
		if (delta === null) return '#5c6170';
		return delta >= 0 ? '#5ee0ab' : '#5c6170';
	}

	const cpuSeries = $derived(toSeries(primaryHostHistory, 'cpuPct'));
	const memSeries = $derived(toSeries(primaryHostHistory, 'memPct'));
	const netSeries = $derived(toNetSeries(primaryHostHistory));

	const cpuDelta = $derived(trend(primaryHostHistory, 'cpuPct'));
	const memDelta = $derived(trend(primaryHostHistory, 'memPct'));
	const netDelta = $derived(netTrend(primaryHostHistory));

	const metricCards = $derived([
		{
			label: 'CPU usage',
			big:
				primaryHostLatestSample?.cpuPct != null ? primaryHostLatestSample.cpuPct.toFixed(1) : '—',
			unit: '%',
			sub: primaryHost?.name ?? 'no hosts yet',
			delta: deltaLabel(cpuDelta),
			deltaColor: deltaColor(cpuDelta),
			color: '#3fb984',
			series: cpuSeries
		},
		{
			label: 'Memory',
			big:
				primaryHostLatestSample?.memPct != null ? primaryHostLatestSample.memPct.toFixed(1) : '—',
			unit: '%',
			sub: 'committed',
			delta: deltaLabel(memDelta),
			deltaColor: deltaColor(memDelta),
			color: '#52cc96',
			series: memSeries
		},
		{
			label: 'Network I/O',
			big: primaryHostLatestSample ? netKBs(primaryHostLatestSample).toFixed(0) : '—',
			unit: 'KB/s',
			sub: '↓ in · ↑ out combined',
			delta: deltaLabel(netDelta, ' KB/s'),
			deltaColor: deltaColor(netDelta),
			color: '#8b5cf6',
			series: netSeries
		}
	]);

	const statRow = $derived([
		{
			label: 'Apps',
			value: page.data.appsCount ?? 0,
			sub: 'total',
			icon: 'M3 3h8v8H3zM13 3h8v8h-8zM3 13h8v8H3zM13 13h8v8h-8z'
		},
		{
			label: 'Hosts online',
			value: page.data.hostsOnlineCount ?? 0,
			sub: `/ ${page.data.hostsTotalCount ?? 0} total`,
			icon: 'M12 3l8 4v10l-8 4-8-4V7z'
		},
		{
			label: 'Running',
			value: page.data.runningDeploymentsCount ?? 0,
			sub: 'deployments',
			icon: 'M4 4h16v6H4zM4 14h16v6H4z'
		},
		{
			label: 'Building now',
			value: page.data.activeBuildsCount ?? 0,
			sub: 'in progress',
			icon: 'M4 5h16v14H4zM4 15l5-5 4 4 3-3 4 4'
		},
		{
			label: 'Volumes',
			value: page.data.volumesCount ?? 0,
			sub: formatBytes(page.data.volumesTotalBytes ?? 0),
			icon: 'M4 6c0-1.6 3.6-3 8-3s8 1.4 8 3-3.6 3-8 3-8-1.4-8-3zM4 6v12c0 1.6 3.6 3 8 3s8-1.4 8-3V6'
		}
	]);

	const workloads = $derived(
		apps.slice(0, 5).map((a) => {
			const m = statusMeta(a.status);
			return { ...a, dot: m.dot };
		})
	);

	function openApp(id: string) {
		goto(`/${guildId}/deploy/projects/${id}`);
	}

	function selectRange(id: string) {
		goto(`?range=${id}`, { keepFocus: true, noScroll: true, replaceState: true });
	}

	function rangePillCls(id: string) {
		return id === page.data.range
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}
</script>

<div class="px-7 py-6">
	<!-- Page header -->
	<div class="flex items-start gap-3 mb-5">
		<div class="flex-1 min-w-0">
			<div class="font-heading font-bold text-[28px] tracking-[-0.01em]">{guildName} Dashboard</div>
			<div class="text-[var(--tx-2)] text-[13.5px] mt-[3px]">
				Live overview of all containers and hosts in {guildName}
			</div>
		</div>
		<span
			class="font-mono-jb text-[11.5px] font-semibold px-[10px] py-[5px] rounded-[7px] bg-[var(--grn-dim)] text-[var(--grn)]"
			>rootless</span
		>
		<span
			class="font-mono-jb text-[11.5px] px-[10px] py-[5px] rounded-[7px] bg-[var(--card)] border border-[var(--line)] text-[var(--tx-2)]"
			>podman 5.1.0</span
		>
	</div>

	<!-- Range picker -->
	<div class="flex items-center justify-end mb-[10px]">
		<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
			{#each HOST_METRIC_RANGES as r (r.id)}
				<button onclick={() => selectRange(r.id)} class={rangePillCls(r.id)}>{r.label}</button>
			{/each}
		</div>
	</div>

	<!-- Metric cards -->
	<div class="grid grid-cols-3 gap-[14px]">
		{#each metricCards as m (m.label)}
			<div
				class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] pt-4 px-[18px] overflow-hidden"
				style:color={m.color}
			>
				<div class="flex items-center justify-between">
					<span class="text-[13px] font-semibold text-[var(--tx-2)] whitespace-nowrap"
						>{m.label}</span
					>
					<span class="text-[11.5px] font-semibold" style:color={m.deltaColor}>{m.delta}</span>
				</div>
				<div class="flex items-baseline gap-[7px] mt-[10px]">
					<span class="font-heading font-bold text-[34px] leading-none text-[var(--tx)]"
						>{m.big}</span
					>
					<span class="text-[13px] text-[var(--tx-3)]">{m.unit}</span>
				</div>
				<div class="text-[11.5px] text-[var(--tx-3)] mt-[6px]">{m.sub}</div>
				<div class="h-[72px] mt-[10px] -mx-[18px] w-[calc(100%+36px)]">
					{#if m.series.length > 1}
						<AreaChart
							data={m.series}
							x="i"
							y="value"
							yDomain={[0, null]}
							axis={false}
							grid={false}
							legend={false}
							rule={false}
							tooltipContext={false}
							highlight={false}
							padding={{ top: 4, bottom: 4 }}
							props={{
								area: {
									fillOpacity: 0.13,
									fill: 'currentColor',
									line: { stroke: 'currentColor', 'stroke-width': 2 }
								}
							}}
						/>
					{:else}
						<div class="h-full flex items-center text-[11px] text-[var(--tx-3)]">
							Not enough history yet
						</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<!-- Stat row -->
	<div class="grid grid-cols-5 gap-3 mt-[14px]">
		{#each statRow as s (s.label)}
			<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[14px_15px]">
				<div class="flex items-center gap-2 mb-[10px]">
					<svg
						width="15"
						height="15"
						viewBox="0 0 24 24"
						fill="none"
						stroke="var(--tx-3)"
						stroke-width="1.7"
					>
						<path d={s.icon} />
					</svg>
					<span class="text-[12px] font-semibold text-[var(--tx-2)]">{s.label}</span>
				</div>
				<div class="flex items-baseline gap-[6px]">
					<span class="font-heading font-bold text-[24px] leading-none">{s.value}</span>
					<span class="text-[11.5px] text-[var(--tx-3)]">{s.sub}</span>
				</div>
			</div>
		{/each}
	</div>

	<!-- Bottom 2-col: workloads + activity -->
	<div class="grid grid-cols-[1.55fr_1fr] gap-[14px] mt-[14px]">
		<!-- Top workloads -->
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] overflow-hidden">
			<div class="flex items-center justify-between px-[18px] pt-[15px] pb-2">
				<span class="text-[14px] font-bold">Top workloads</span>
				<a
					href="/{guildId}/deploy/projects"
					class="text-[12.5px] text-[var(--grn-2)] no-underline font-semibold">View all →</a
				>
			</div>
			{#each workloads as w (w.id)}
				<div
					onclick={() => openApp(w.id)}
					onkeydown={(e) => e.key === 'Enter' && openApp(w.id)}
					role="row"
					tabindex="0"
					class="grid grid-cols-[1fr_90px_54px_80px] gap-3 items-center px-[18px] py-[11px] border-t border-t-[var(--line)] cursor-pointer hover:bg-white/[0.02]"
				>
					<div class="flex items-center gap-[10px] min-w-0">
						<div class="size-[9px] rounded-full shrink-0" style:background={w.dot}></div>
						<div class="min-w-0">
							<div
								class="text-[13px] font-semibold whitespace-nowrap overflow-hidden text-ellipsis"
							>
								{w.name}
							</div>
							<div
								class="font-mono-jb text-[10.5px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis"
							>
								ghcr.io/…/{w.name}:latest
							</div>
						</div>
					</div>
					<div class="flex items-center gap-1">
						<div class="flex-1 h-[5px] rounded-[3px] bg-white/[0.07] overflow-hidden">
							<div
								class="h-full bg-[var(--grn)] rounded-[3px]"
								style:width="{Math.max(4, w.cpu)}%"
							></div>
						</div>
					</div>
					<span class="font-mono-jb text-[12px] text-[var(--tx)] text-right">{w.cpu}%</span>
					<span class="font-mono-jb text-[12px] text-[var(--tx-2)] text-right">{w.mem}</span>
				</div>
			{/each}
		</div>

		<!-- Recent activity -->
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] px-1 pt-[6px] pb-3">
			<div class="px-4 pt-[13px] pb-2 text-[14px] font-bold">Recent activity</div>
			{#each page.data.activityEvents ?? [] as a (a.at + a.text)}
				<div class="flex items-start gap-[11px] px-4 py-2">
					<div class="size-2 rounded-full mt-[5px] shrink-0" style:background={a.dot}></div>
					<div class="flex-1 min-w-0">
						<div class="text-[12.5px] text-[var(--tx-2)]">
							<span class="text-[var(--tx)] font-semibold">{a.who}</span>{a.text}
						</div>
						<div class="text-[11px] text-[var(--tx-3)] mt-[1px]">
							{formatRelativeTime(new Date(a.at))}
						</div>
					</div>
				</div>
			{/each}
			{#if (page.data.activityEvents ?? []).length === 0}
				<div class="px-4 py-6 text-center text-[12.5px] text-[var(--tx-3)]">
					No recent activity yet.
				</div>
			{/if}
		</div>
	</div>
</div>
