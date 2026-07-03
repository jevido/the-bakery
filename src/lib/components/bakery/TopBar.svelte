<script lang="ts">
	import { GUILDS } from '$lib/data/bakery';

	let {
		guildId,
		module,
		section,
		onDeploy,
	}: {
		guildId: string;
		module: string;
		section: string;
		onDeploy?: () => void;
	} = $props();

	const guild = $derived(GUILDS[guildId]);
	const guildName = $derived(guild?.name ?? guildId);
	const moduleName = $derived(module === 'deploy' ? 'Deploy' : module === 'guild' ? 'Guild' : 'Planning');

	const sectionLabels: Record<string, string> = {
		overview: 'Dashboard',
		projects: 'Projects',
		sources: 'Sources',
		networks: 'Networks',
		storage: 'Storage',
		members: 'Members',
		roles: 'Roles',
		settings: 'Guild Settings',
		board: 'Board',
		'my-work': 'My Work',
		cycles: 'Cycles',
		views: 'Views',
	};

	const ctaLabels: Record<string, string> = {
		overview: 'Deploy',
		projects: 'New project',
		sources: 'Add source',
		networks: 'Add network',
		storage: 'Add volume',
		members: 'Invite member',
		roles: 'Create role',
		board: 'New task',
	};

	const crumb = $derived(sectionLabels[section] ?? section);
	const ctaLabel = $derived(ctaLabels[section] ?? 'New');

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') onDeploy?.();
	}
</script>

<div class="h-[52px] shrink-0 border-b border-b-[var(--line)] flex items-center gap-[14px] px-5 bg-[var(--topbar)]">
	<div class="flex items-center gap-2 text-[14px] min-w-0">
		<span class="text-[var(--tx-3)]">{guildName}</span>
		<span class="text-[var(--tx-3)]">/</span>
		<span class="text-[var(--tx-2)]">{moduleName}</span>
		<span class="text-[var(--tx-3)]">/</span>
		<span class="text-[var(--tx)] font-semibold whitespace-nowrap overflow-hidden text-ellipsis">{crumb}</span>
	</div>

	<div class="flex-1"></div>

	<div class="flex items-center gap-2 bg-[var(--card)] border border-[var(--line)] rounded-[9px] px-[11px] py-[6px] w-[230px] text-[var(--tx-3)]">
		<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="11" cy="11" r="7"/>
			<path d="M20 20l-3.5-3.5"/>
		</svg>
		<span class="text-[13px]">Search apps, hosts…</span>
		<span class="ml-auto font-mono-jb text-[10.5px] bg-white/[0.06] px-[5px] py-[1px] rounded-[4px]">⌘K</span>
	</div>

	<button
		onclick={onDeploy}
		onkeydown={handleKeydown}
		class="flex items-center gap-[7px] bg-[var(--grn)] text-white rounded-[9px] px-[14px] py-2 text-[13.5px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]"
	>
		<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4">
			<path d="M12 5v14M5 12h14"/>
		</svg>
		{ctaLabel}
	</button>
</div>
