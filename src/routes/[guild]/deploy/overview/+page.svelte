<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS, ACTIVITY, statusMeta } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
	const apps = $derived(guild?.apps ?? []);
	const hosts = $derived(guild?.hosts ?? []);

	const primaryHost = $derived(hosts[0]);

	function wobble(min: number, max: number, points: number, seed: number) {
		return Array.from({ length: points }, (_, i) => {
			return min + ((max - min) * Math.abs(Math.sin(i * 0.7 + seed) + Math.cos(i * 0.4 + seed))) / 2;
		});
	}
	function genLine(vals: number[], w: number, h: number) {
		const min = Math.min(...vals), max = Math.max(...vals);
		const pts = vals.map((v, i) => {
			const x = (i / (vals.length - 1)) * w;
			const y = h - ((v - min) / (max - min || 1)) * (h - 8) - 4;
			return `${x},${y}`;
		});
		return 'M' + pts.join('L');
	}
	function genArea(vals: number[], w: number, h: number) {
		const line = genLine(vals, w, h);
		return line + `L${w},${h}L0,${h}Z`;
	}

	const cpuVals = $derived(wobble(25, primaryHost?.cpu ?? 40, 24, 1));
	const memVals = $derived(wobble(20, primaryHost?.mem ?? 60, 24, 3));
	const netVals = $derived(wobble(150, 500, 24, 7));

	const metricCards = $derived([
		{ label: 'CPU usage', big: String(primaryHost?.cpu ?? '—'), unit: '%', sub: `rootless · ${primaryHost?.spec?.split('·')[1]?.trim() ?? ''}`, delta: '▲ 4%', deltaColor: '#5ee0ab', color: '#3fb984', vals: cpuVals },
		{ label: 'Memory', big: ((primaryHost?.mem ?? 0) / 10).toFixed(1), unit: '/ 16 GiB', sub: `${primaryHost?.mem ?? 0}% committed`, delta: '▼ 1%', deltaColor: '#5c6170', color: '#52cc96', vals: memVals },
		{ label: 'Network I/O', big: '443', unit: 'KB/s', sub: '↓ in · ↑ out combined', delta: '▲ 12%', deltaColor: '#5ee0ab', color: '#8b5cf6', vals: netVals },
	]);

	const statRow = $derived([
		{ label: 'Apps', value: apps.length, sub: `/ ${apps.length + 2}`, icon: 'M3 3h8v8H3zM13 3h8v8h-8zM3 13h8v8H3zM13 13h8v8h-8z' },
		{ label: 'Pods', value: 5, sub: 'active', icon: 'M12 3l8 4v10l-8 4-8-4V7z' },
		{ label: 'Quadlets', value: 10, sub: 'units', icon: 'M4 4h16v6H4zM4 14h16v6H4z' },
		{ label: 'Images', value: 9, sub: '254 MB unused', icon: 'M4 5h16v14H4zM4 15l5-5 4 4 3-3 4 4' },
		{ label: 'Volumes', value: 6, sub: '17.1 GB', icon: 'M4 6c0-1.6 3.6-3 8-3s8 1.4 8 3-3.6 3-8 3-8-1.4-8-3zM4 6v12c0 1.6 3.6 3 8 3s8-1.4 8-3V6' },
	]);

	const workloads = $derived(apps.slice(0, 5).map((a) => {
		const m = statusMeta(a.status);
		return { ...a, dot: m.dot };
	}));

	import { goto } from '$app/navigation';
	function openApp(id: string) {
		goto(`/${guildId}/deploy/projects/${id}`);
	}
</script>

<div class="px-7 py-6">
	<!-- Page header -->
	<div class="flex items-start gap-3 mb-5">
		<div class="flex-1 min-w-0">
			<div class="font-heading font-bold text-[28px] tracking-[-0.01em]">{guild?.name} Dashboard</div>
			<div class="text-[var(--tx-2)] text-[13.5px] mt-[3px]">Live overview of all containers and hosts in {guild?.name}</div>
		</div>
		<span class="font-mono-jb text-[11.5px] font-semibold px-[10px] py-[5px] rounded-[7px] bg-[var(--grn-dim)] text-[var(--grn)]">rootless</span>
		<span class="font-mono-jb text-[11.5px] px-[10px] py-[5px] rounded-[7px] bg-[var(--card)] border border-[var(--line)] text-[var(--tx-2)]">podman 5.1.0</span>
	</div>

	<!-- Metric cards -->
	<div class="grid grid-cols-3 gap-[14px]">
		{#each metricCards as m}
			<div class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] pt-4 px-[18px] overflow-hidden" style:color={m.color}>
				<div class="flex items-center justify-between">
					<span class="text-[13px] font-semibold text-[var(--tx-2)] whitespace-nowrap">{m.label}</span>
					<span class="text-[11.5px] font-semibold" style:color={m.deltaColor}>{m.delta}</span>
				</div>
				<div class="flex items-baseline gap-[7px] mt-[10px]">
					<span class="font-heading font-bold text-[34px] leading-none text-[var(--tx)]">{m.big}</span>
					<span class="text-[13px] text-[var(--tx-3)]">{m.unit}</span>
				</div>
				<div class="text-[11.5px] text-[var(--tx-3)] mt-[6px]">{m.sub}</div>
				<svg viewBox="0 0 320 88" preserveAspectRatio="none" height="72" class="block mt-[10px] -mx-[18px] w-[calc(100%+36px)]">
					<path d={genArea(m.vals, 320, 88)} fill="currentColor" fill-opacity="0.13" stroke="none"/>
					<path d={genLine(m.vals, 320, 88)} fill="none" stroke="currentColor" stroke-width="2"/>
				</svg>
			</div>
		{/each}
	</div>

	<!-- Stat row -->
	<div class="grid grid-cols-5 gap-3 mt-[14px]">
		{#each statRow as s}
			<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[14px_15px]">
				<div class="flex items-center gap-2 mb-[10px]">
					<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.7">
						<path d={s.icon}/>
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
				<a href="/{guildId}/deploy/projects" class="text-[12.5px] text-[var(--grn-2)] no-underline font-semibold">View all →</a>
			</div>
			{#each workloads as w}
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
							<div class="text-[13px] font-semibold whitespace-nowrap overflow-hidden text-ellipsis">{w.name}</div>
							<div class="font-mono-jb text-[10.5px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis">ghcr.io/…/{w.name}:latest</div>
						</div>
					</div>
					<div class="flex items-center gap-1">
						<div class="flex-1 h-[5px] rounded-[3px] bg-white/[0.07] overflow-hidden">
							<div class="h-full bg-[var(--grn)] rounded-[3px]" style:width="{Math.max(4, w.cpu)}%"></div>
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
			{#each ACTIVITY as a}
				<div class="flex items-start gap-[11px] px-4 py-2">
					<div class="size-2 rounded-full mt-[5px] shrink-0" style:background={a.dot}></div>
					<div class="flex-1 min-w-0">
						<div class="text-[12.5px] text-[var(--tx-2)]">
							<span class="text-[var(--tx)] font-semibold">{a.who}</span>{a.text}
						</div>
						<div class="text-[11px] text-[var(--tx-3)] mt-[1px]">{a.time}</div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>
