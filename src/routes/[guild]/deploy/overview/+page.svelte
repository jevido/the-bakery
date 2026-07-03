<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS, ACTIVITY, statusMeta } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
	const apps = $derived(guild?.apps ?? []);
	const hosts = $derived(guild?.hosts ?? []);

	const runningCount = $derived(apps.filter((a) => a.status === 'running').length);
	const buildingCount = $derived(apps.filter((a) => a.status === 'building').length);
	const failedCount = $derived(apps.filter((a) => a.status === 'failed').length);
	const stoppedCount = $derived(apps.filter((a) => a.status === 'stopped').length);
	const primaryHost = $derived(hosts[0]);

	// Sparkline path helpers
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
		goto(`/${guildId}/deploy/apps/${id}`);
	}
</script>

<div style="padding: 24px 28px;">
	<!-- Page header -->
	<div style="display:flex;align-items:flex-start;gap:12px;margin-bottom:20px;">
		<div style="flex:1;min-width:0;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:28px;letter-spacing:-.01em;">{guild?.name} Dashboard</div>
			<div style="color:var(--tx-2);font-size:13.5px;margin-top:3px;">Live overview of all containers and hosts in {guild?.name}</div>
		</div>
		<span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;font-weight:600;padding:5px 10px;border-radius:7px;background:var(--grn-dim);color:var(--grn);">rootless</span>
		<span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;padding:5px 10px;border-radius:7px;background:var(--card);border:1px solid var(--line);color:var(--tx-2);">podman 5.1.0</span>
	</div>

	<!-- Metric cards -->
	<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:14px;">
		{#each metricCards as m}
			<div style="background:var(--card);border:1px solid var(--line);border-radius:14px;padding:16px 18px 0;overflow:hidden;color:{m.color};">
				<div style="display:flex;align-items:center;justify-content:space-between;">
					<span style="font-size:13px;font-weight:600;color:var(--tx-2);white-space:nowrap;">{m.label}</span>
					<span style="font-size:11.5px;font-weight:600;color:{m.deltaColor};">{m.delta}</span>
				</div>
				<div style="display:flex;align-items:baseline;gap:7px;margin-top:10px;">
					<span style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:34px;line-height:1;color:var(--tx);">{m.big}</span>
					<span style="font-size:13px;color:var(--tx-3);">{m.unit}</span>
				</div>
				<div style="font-size:11.5px;color:var(--tx-3);margin-top:6px;">{m.sub}</div>
				<svg viewBox="0 0 320 88" preserveAspectRatio="none" height="72" style="display:block;margin:10px -18px 0;width:calc(100% + 36px);">
					<path d={genArea(m.vals, 320, 88)} fill="currentColor" fill-opacity="0.13" stroke="none"/>
					<path d={genLine(m.vals, 320, 88)} fill="none" stroke="currentColor" stroke-width="2"/>
				</svg>
			</div>
		{/each}
	</div>

	<!-- Stat row -->
	<div style="display:grid;grid-template-columns:repeat(5,1fr);gap:12px;margin-top:14px;">
		{#each statRow as s}
			<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:14px 15px;">
				<div style="display:flex;align-items:center;gap:8px;margin-bottom:10px;">
					<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.7">
						<path d={s.icon}/>
					</svg>
					<span style="font-size:12px;font-weight:600;color:var(--tx-2);">{s.label}</span>
				</div>
				<div style="display:flex;align-items:baseline;gap:6px;">
					<span style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:24px;line-height:1;">{s.value}</span>
					<span style="font-size:11.5px;color:var(--tx-3);">{s.sub}</span>
				</div>
			</div>
		{/each}
	</div>

	<!-- Bottom 2-col: workloads + activity -->
	<div style="display:grid;grid-template-columns:1.55fr 1fr;gap:14px;margin-top:14px;">
		<!-- Top workloads -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:14px;overflow:hidden;">
			<div style="display:flex;align-items:center;justify-content:space-between;padding:15px 18px 8px;">
				<span style="font-size:14px;font-weight:700;">Top workloads</span>
				<a href="/{guildId}/deploy/apps" style="font-size:12.5px;color:var(--grn-2);text-decoration:none;font-weight:600;">View all →</a>
			</div>
			{#each workloads as w}
				<div
					onclick={() => openApp(w.id)}
					onkeydown={(e) => e.key === 'Enter' && openApp(w.id)}
					role="row"
					tabindex="0"
					style="display:grid;grid-template-columns:1fr 90px 54px 80px;gap:12px;align-items:center;padding:11px 18px;border-top:1px solid var(--line);cursor:pointer;"
				>
					<div style="display:flex;align-items:center;gap:10px;min-width:0;">
						<div style="width:9px;height:9px;border-radius:50%;background:{w.dot};flex:none;"></div>
						<div style="min-width:0;">
							<div style="font-size:13px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{w.name}</div>
							<div style="font-family:'JetBrains Mono',monospace;font-size:10.5px;color:var(--tx-3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">ghcr.io/…/{w.name}:latest</div>
						</div>
					</div>
					<div style="display:flex;align-items:center;gap:4px;">
						<div style="flex:1;height:5px;border-radius:3px;background:rgba(255,255,255,.07);overflow:hidden;">
							<div style="width:{Math.max(4, w.cpu)}%;height:100%;background:var(--grn);border-radius:3px;"></div>
						</div>
					</div>
					<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx);text-align:right;">{w.cpu}%</span>
					<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx-2);text-align:right;">{w.mem}</span>
				</div>
			{/each}
		</div>

		<!-- Recent activity -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:14px;padding:6px 4px 12px;">
			<div style="padding:13px 16px 8px;font-size:14px;font-weight:700;">Recent activity</div>
			{#each ACTIVITY as a}
				<div style="display:flex;align-items:flex-start;gap:11px;padding:8px 16px;">
					<div style="width:8px;height:8px;border-radius:50%;background:{a.dot};margin-top:5px;flex:none;"></div>
					<div style="flex:1;min-width:0;">
						<div style="font-size:12.5px;color:var(--tx-2);">
							<span style="color:var(--tx);font-weight:600;">{a.who}</span>{a.text}
						</div>
						<div style="font-size:11px;color:var(--tx-3);margin-top:1px;">{a.time}</div>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>
