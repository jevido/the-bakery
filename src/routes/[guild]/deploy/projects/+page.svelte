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

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-center gap-[14px] mb-[18px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em]">Projects</div>
			<div class="text-[var(--tx-2)] text-[13px] mt-[1px]">
				{apps.length} projects across {hostCount} hosts · {runCount} fresh · {buildCount} baking
			</div>
		</div>
		<div class="flex-1"></div>
		<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
			<button onclick={() => filter = 'all'} class={filterCls('all')}>All</button>
			<button onclick={() => filter = 'running'} class={filterCls('running')}>Running</button>
			<button onclick={() => filter = 'issues'} class={filterCls('issues')}>Issues</button>
		</div>
	</div>

	<!-- Table -->
	<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
		<div class="grid grid-cols-[2.2fr_1.1fr_1.3fr_1fr_90px] gap-[14px] px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
			<div>PROJECT</div>
			<div>STATUS</div>
			<div>HOST · DOMAIN</div>
			<div>RESOURCES</div>
			<div class="text-right">DEPLOYED</div>
		</div>

		{#each filtered as a}
			{@const m = statusMeta(a.status)}
			<div
				onclick={() => goto(`/${guildId}/deploy/projects/${a.id}`)}
				onkeydown={(e) => e.key === 'Enter' && goto(`/${guildId}/deploy/projects/${a.id}`)}
				role="row"
				tabindex="0"
				class="px-[18px] py-[13px] grid grid-cols-[2.2fr_1.1fr_1.3fr_1fr_90px] gap-[14px] items-center border-b border-b-[var(--line)] cursor-pointer hover:bg-white/[0.02]"
			>
				<!-- App name + type -->
				<div class="flex items-center gap-3 min-w-0">
					<AppIcon initial={a.initial} status={a.status} size={34} radius={10} />
					<div class="min-w-0">
						<div class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{a.name}</div>
						<div class="font-mono-jb text-[11px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis">{a.type}</div>
					</div>
				</div>

				<!-- Status -->
				<div class="flex items-center gap-2">
					<StatusDot status={a.status} />
					<div>
						<div class="text-[12.5px] font-semibold" style:color={m.c}>{m.label}</div>
						<div class="text-[10.5px] text-[var(--tx-3)]">{m.flavor}</div>
					</div>
				</div>

				<!-- Host + Domain -->
				<div class="min-w-0">
					<div class="text-[12.5px] text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{a.host}</div>
					<div class="font-mono-jb text-[11px] text-[var(--grn-2)] whitespace-nowrap overflow-hidden text-ellipsis">{a.domain}</div>
				</div>

				<!-- Resources -->
				<div class="flex flex-col gap-1">
					<div class="flex items-center gap-[6px]">
						<span class="font-mono-jb text-[10.5px] text-[var(--tx-3)] w-[26px]">CPU</span>
						<div class="flex-1 h-[5px] rounded-[3px] bg-white/[0.07] overflow-hidden">
							<div class="h-full rounded-[3px]" style:width="{Math.max(4, a.cpu)}%" style:background={a.cpu > 70 ? '#e0a83e' : 'var(--grn)'}></div>
						</div>
					</div>
					<div class="flex items-center gap-[6px]">
						<span class="font-mono-jb text-[10.5px] text-[var(--tx-3)] w-[26px]">MEM</span>
						<span class="font-mono-jb text-[10.5px] text-[var(--tx-2)]">{a.mem}</span>
					</div>
				</div>

				<!-- Deployed -->
				<div class="text-right">
					<div class="text-[12px] text-[var(--tx-2)]">{a.deployed}</div>
				</div>
			</div>
		{/each}

		{#if filtered.length === 0}
			<div class="p-10 text-center text-[var(--tx-3)] text-[13.5px]">No projects match the current filter.</div>
		{/if}
	</div>
</div>
