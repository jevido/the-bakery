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

<div style="
	height: 52px;
	flex: none;
	border-bottom: 1px solid var(--line);
	display: flex;
	align-items: center;
	gap: 14px;
	padding: 0 20px;
	background: var(--topbar);
">
	<div style="display:flex;align-items:center;gap:8px;font-size:14px;min-width:0;">
		<span style="color:var(--tx-3)">{guildName}</span>
		<span style="color:var(--tx-3)">/</span>
		<span style="color:var(--tx-2)">{moduleName}</span>
		<span style="color:var(--tx-3)">/</span>
		<span style="color:var(--tx);font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{crumb}</span>
	</div>

	<div style="flex:1"></div>

	<div style="display:flex;align-items:center;gap:8px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:6px 11px;width:230px;color:var(--tx-3);">
		<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="11" cy="11" r="7"/>
			<path d="M20 20l-3.5-3.5"/>
		</svg>
		<span style="font-size:13px">Search apps, hosts…</span>
		<span style="margin-left:auto;font-family:'JetBrains Mono',monospace;font-size:10.5px;background:rgba(255,255,255,.06);padding:1px 5px;border-radius:4px;">⌘K</span>
	</div>

	<button
		onclick={onDeploy}
		onkeydown={handleKeydown}
		style="
			display:flex;align-items:center;gap:7px;
			background:var(--grn);color:#fff;border:none;border-radius:9px;
			padding:8px 14px;font-size:13.5px;font-weight:700;cursor:pointer;
			box-shadow:0 2px 12px var(--grn-dim);
		"
	>
		<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4">
			<path d="M12 5v14M5 12h14"/>
		</svg>
		{ctaLabel}
	</button>
</div>
