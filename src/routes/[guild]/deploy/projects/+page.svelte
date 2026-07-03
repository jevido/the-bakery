<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS, statusMeta } from '$lib/data/bakery';
	import StatusDot from '$lib/components/bakery/StatusDot.svelte';
	import AppIcon from '$lib/components/bakery/AppIcon.svelte';
	import { goto } from '$app/navigation';

	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
	const apps = $derived(guild?.apps ?? []);

	let filter = $state<'all' | 'running' | 'issues'>('all');

	const filtered = $derived(
		filter === 'all' ? apps
		: filter === 'running' ? apps.filter((a) => a.status === 'running')
		: apps.filter((a) => a.status === 'failed' || a.status === 'stopped')
	);

	const runCount = $derived(apps.filter((a) => a.status === 'running').length);
	const buildCount = $derived(apps.filter((a) => a.status === 'building').length);
	const hostCount = $derived(guild?.hosts?.length ?? 0);

	function filterStyle(f: typeof filter) {
		return f === filter
			? 'padding:5px 11px;font-size:12.5px;color:var(--tx);border-radius:6px;cursor:pointer;background:var(--card-2);'
			: 'padding:5px 11px;font-size:12.5px;color:var(--tx-2);border-radius:6px;cursor:pointer;';
	}
</script>

<div style="padding: 22px 28px;">
	<!-- Header -->
	<div style="display:flex;align-items:center;gap:14px;margin-bottom:18px;">
		<div>
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;letter-spacing:-.01em;">Projects</div>
			<div style="color:var(--tx-2);font-size:13px;margin-top:1px;">
				{apps.length} projects across {hostCount} hosts · {runCount} fresh · {buildCount} baking
			</div>
		</div>
		<div style="flex:1"></div>
		<!-- Filter tabs -->
		<div style="display:flex;gap:2px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:3px;">
			<button onclick={() => filter = 'all'} style={filterStyle('all')}>All</button>
			<button onclick={() => filter = 'running'} style={filterStyle('running')}>Running</button>
			<button onclick={() => filter = 'issues'} style={filterStyle('issues')}>Issues</button>
		</div>
	</div>

	<!-- Table -->
	<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
		<!-- Header row -->
		<div style="display:grid;grid-template-columns:2.2fr 1.1fr 1.3fr 1fr 90px;gap:14px;padding:11px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
			<div>PROJECT</div>
			<div>STATUS</div>
			<div>HOST · DOMAIN</div>
			<div>RESOURCES</div>
			<div style="text-align:right">DEPLOYED</div>
		</div>

		{#each filtered as a}
			{@const m = statusMeta(a.status)}
			<div
				onclick={() => goto(`/${guildId}/deploy/projects/${a.id}`)}
				onkeydown={(e) => e.key === 'Enter' && goto(`/${guildId}/deploy/projects/${a.id}`)}
				role="row"
				tabindex="0"
				style="padding:13px 18px;display:grid;grid-template-columns:2.2fr 1.1fr 1.3fr 1fr 90px;gap:14px;align-items:center;border-bottom:1px solid var(--line);cursor:pointer;"
			>
				<!-- App name + type -->
				<div style="display:flex;align-items:center;gap:12px;min-width:0;">
					<AppIcon initial={a.initial} status={a.status} size={34} radius={10} />
					<div style="min-width:0;">
						<div style="font-size:14px;font-weight:600;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{a.name}</div>
						<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--tx-3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{a.type}</div>
					</div>
				</div>

				<!-- Status -->
				<div style="display:flex;align-items:center;gap:8px;">
					<StatusDot status={a.status} />
					<div>
						<div style="font-size:12.5px;font-weight:600;color:{m.c};">{m.label}</div>
						<div style="font-size:10.5px;color:var(--tx-3);">{m.flavor}</div>
					</div>
				</div>

				<!-- Host + Domain -->
				<div style="min-width:0;">
					<div style="font-size:12.5px;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{a.host}</div>
					<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--grn-2);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{a.domain}</div>
				</div>

				<!-- Resources -->
				<div style="display:flex;flex-direction:column;gap:4px;">
					<div style="display:flex;align-items:center;gap:6px;">
						<span style="font-family:'JetBrains Mono',monospace;font-size:10.5px;color:var(--tx-3);width:26px;">CPU</span>
						<div style="flex:1;height:5px;border-radius:3px;background:rgba(255,255,255,.07);overflow:hidden;">
							<div style="width:{Math.max(4, a.cpu)}%;height:100%;background:{a.cpu > 70 ? '#e0a83e' : 'var(--grn)'};border-radius:3px;"></div>
						</div>
					</div>
					<div style="display:flex;align-items:center;gap:6px;">
						<span style="font-family:'JetBrains Mono',monospace;font-size:10.5px;color:var(--tx-3);width:26px;">MEM</span>
						<span style="font-family:'JetBrains Mono',monospace;font-size:10.5px;color:var(--tx-2);">{a.mem}</span>
					</div>
				</div>

				<!-- Deployed -->
				<div style="text-align:right;">
					<div style="font-size:12px;color:var(--tx-2);">{a.deployed}</div>
				</div>
			</div>
		{/each}

		{#if filtered.length === 0}
			<div style="padding:40px;text-align:center;color:var(--tx-3);font-size:13.5px;">No projects match the current filter.</div>
		{/if}
	</div>
</div>

<style>
	button {
		all: unset;
		box-sizing: border-box;
		cursor: pointer;
	}
	div[role="row"]:hover {
		background: rgba(255,255,255,.02);
	}
</style>
