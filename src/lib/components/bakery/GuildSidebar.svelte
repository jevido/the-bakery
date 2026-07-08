<script lang="ts">
	import { GUILDS, MEMBERS } from '$lib/data/bakery';
	import { goto } from '$app/navigation';
	import type { Snippet } from 'svelte';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';

	let {
		guildId,
		module,
		activeSection,
		children,
	}: {
		guildId: string;
		module: 'deploy' | 'planning';
		activeSection: string;
		children?: Snippet;
	} = $props();

	const guild = $derived(GUILDS[guildId]);
	const members = $derived(MEMBERS[guildId] ?? []);
	const memberCount = $derived(members.length);
	const apps = $derived(guild?.apps ?? []);

	let guildMenuOpen = $state(false);
	let moduleMenuOpen = $state(false);
	let sidebarEl = $state<HTMLElement | null>(null);
	let leaveDialogOpen = $state(false);

	function toggleGuildMenu() { guildMenuOpen = !guildMenuOpen; moduleMenuOpen = false; }
	function toggleModuleMenu() { moduleMenuOpen = !moduleMenuOpen; guildMenuOpen = false; }
	function closeAll() { guildMenuOpen = false; moduleMenuOpen = false; }

	function handleWindowClick(e: MouseEvent) {
		if (sidebarEl && !sidebarEl.contains(e.target as Node)) closeAll();
	}

	function nav(path: string) {
		closeAll();
		goto(`/${guildId}/${path}`);
	}

	function isActive(section: string) { return activeSection === section; }

	const navBase = 'flex items-center gap-[11px] px-[10px] py-2 rounded-[8px] cursor-pointer text-[13.5px] font-medium text-[var(--tx-2)] mb-0.5 w-full text-left hover:bg-white/5';
	const navOn = 'flex items-center gap-[11px] px-[10px] py-2 rounded-[8px] cursor-pointer text-[13.5px] font-semibold text-[var(--grn)] mb-0.5 w-full text-left bg-[var(--grn-dim)]';
	function ns(s: string) { return isActive(s) ? navOn : navBase; }

	const moduleIcons: Record<string, string> = {
		deploy: 'M20 7l-8-4-8 4v10l8 4 8-4V7zM4 7l8 4 8-4M12 21V11',
		planning: 'M3 4h18v4H3zM3 12h18v4H3zM3 20h18',
	};
	const moduleNames: Record<string, string> = { deploy: 'Deploy', planning: 'Planning' };
	const moduleDescs: Record<string, string> = { deploy: 'Projects, sources, infra & guild', planning: 'Board, cycles & sprints' };

	const appCount = $derived(apps.length);
	const hostCount = $derived(guild?.hosts?.length ?? 0);

	const badgeStyle = 'text-[11px] font-bold px-[7px] py-[1px] rounded-[20px] bg-white/[0.07] text-[var(--tx-2)]';

	const guildMenuItems = [
		{ icon: 'M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z', label: 'Invite People', color: 'var(--tx)', action: () => nav('deploy/members') },
		{ icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z', label: 'Guild Settings', color: 'var(--tx)', action: () => nav('deploy/settings') },
		{ icon: 'M17 7l-1.41 1.41L18.17 11H8v2h10.17l-2.58 2.58L17 17l5-5zM4 5h8V3H4c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h8v-2H4V5z', label: 'Leave Guild', color: '#f0836b', action: () => { leaveDialogOpen = true; closeAll(); } },
	];
</script>

<svelte:window onclick={handleWindowClick} />

<AlertDialog.Root bind:open={leaveDialogOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Leave {guild?.name}?</AlertDialog.Title>
			<AlertDialog.Description>You will lose access to all projects and resources in this guild. You can only rejoin with an invite code.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action>Leave guild</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>

<div bind:this={sidebarEl} class="w-[248px] shrink-0 bg-[var(--sidebar)] flex flex-col border-r border-r-[var(--line)] z-[1] relative">
	<!-- Guild header -->
	<div class="relative shrink-0">
		<div
			onclick={toggleGuildMenu}
			onkeydown={(e) => e.key === 'Enter' && toggleGuildMenu()}
			role="button"
			tabindex="0"
			class="h-[52px] px-4 flex items-center justify-between border-b border-b-[var(--line)] cursor-pointer"
		>
			<div class="min-w-0">
				<div class="font-heading font-bold text-[15px] text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">
					{guild?.name ?? guildId}
				</div>
			</div>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="2">
				<path d={guildMenuOpen ? 'M18 15l-6-6-6 6' : 'M6 9l6 6 6-6'}/>
			</svg>
		</div>

		{#if guildMenuOpen}
			<div class="absolute left-[10px] right-[10px] top-[50px] z-[39] bg-[var(--panel)] border border-[var(--line-2)] rounded-[11px] p-[6px] shadow-[0_18px_46px_rgba(0,0,0,.55)]">
				{#each guildMenuItems as m (m.label)}
					<div
						onclick={m.action}
						onkeydown={(e) => e.key === 'Enter' && m.action()}
						role="button"
						tabindex="0"
						class="flex items-center gap-[10px] px-[10px] py-[9px] rounded-[8px] cursor-pointer hover:bg-white/5"
					>
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={m.color} stroke-width="1.8" class="shrink-0">
							<path d={m.icon}/>
						</svg>
						<span class="flex-1 text-[13px] font-semibold" style:color={m.color}>{m.label}</span>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Module dropdown -->
	<div class="px-3 pt-3 pb-[6px] relative">
		<div class="text-[10px] font-bold tracking-[.09em] text-[var(--tx-3)] px-[6px] pb-[7px]">MODULE</div>
		<div
			onclick={toggleModuleMenu}
			onkeydown={(e) => e.key === 'Enter' && toggleModuleMenu()}
			role="button"
			tabindex="0"
			class="flex items-center gap-[9px] px-[11px] py-[9px] rounded-[9px] bg-[var(--card)] border border-[var(--line-2)] cursor-pointer"
		>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="1.9">
				<path d={moduleIcons[module]}/>
			</svg>
			<div class="text-[13.5px] font-bold text-[var(--tx)] flex-1">{moduleNames[module]}</div>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="2">
				<path d="M8 9l4-4 4 4M8 15l4 4 4-4"/>
			</svg>
		</div>

		{#if moduleMenuOpen}
			<div class="absolute left-3 right-3 top-[64px] z-[30] bg-[var(--panel)] border border-[var(--line-2)] rounded-[11px] p-[6px] shadow-[0_16px_44px_rgba(0,0,0,.55)]">
				{#each (['deploy', 'planning'] as const) as m (m)}
					{@const defaultSection = m === 'deploy' ? 'overview' : 'board'}
					<div
						onclick={() => { nav(`${m}/${defaultSection}`); }}
						onkeydown={(e) => e.key === 'Enter' && nav(`${m}/${defaultSection}`)}
						role="button"
						tabindex="0"
						class="flex items-center gap-[10px] px-[11px] py-[10px] rounded-[8px] cursor-pointer"
						class:bg-[var(--grn-dim)]={module === m}
					>
						<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke={module === m ? 'var(--grn)' : 'var(--tx-2)'} stroke-width="1.8" class="shrink-0">
							<path d={moduleIcons[m]}/>
						</svg>
						<div class="flex-1 min-w-0">
							<div class="text-[13.5px] font-semibold" style:color={module === m ? 'var(--grn)' : 'var(--tx)'}>{moduleNames[m]}</div>
							<div class="text-[11px] text-[var(--tx-3)]">{moduleDescs[m]}</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Nav -->
	<div class="flex-1 overflow-y-auto px-3 pb-3 pt-[10px]">
		{#if module === 'deploy'}
			<button onclick={() => nav('deploy/overview')} class={ns('overview')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<rect x="3" y="3" width="7" height="9" rx="1.5"/>
					<rect x="14" y="3" width="7" height="5" rx="1.5"/>
					<rect x="14" y="12" width="7" height="9" rx="1.5"/>
					<rect x="3" y="16" width="7" height="5" rx="1.5"/>
				</svg>
				<span class="flex-1">Dashboard</span>
			</button>

			<button onclick={() => nav('deploy/projects')} class={ns('projects')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M4 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2z"/>
				</svg>
				<span class="flex-1">Projects</span>
			</button>

			<button onclick={() => nav('deploy/sources')} class={ns('sources')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<polyline points="16 18 22 12 16 6"/>
					<polyline points="8 6 2 12 8 18"/>
				</svg>
				<span class="flex-1">Sources</span>
			</button>

			<button onclick={() => nav('deploy/networks')} class={ns('networks')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="12" cy="5" r="2"/>
					<circle cx="5" cy="19" r="2"/>
					<circle cx="19" cy="19" r="2"/>
					<path d="M12 7v4M12 11l-5 6M12 11l5 6"/>
				</svg>
				<span class="flex-1">Networks</span>
			</button>

			<button onclick={() => nav('deploy/storage')} class={ns('storage')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<ellipse cx="12" cy="6" rx="7" ry="3"/>
					<path d="M5 6v12c0 1.6 3.1 3 7 3s7-1.4 7-3V6"/>
					<path d="M5 12c0 1.6 3.1 3 7 3s7-1.4 7-3"/>
				</svg>
				<span class="flex-1">Storage</span>
			</button>

			<button onclick={() => nav('deploy/hosts')} class={ns('hosts')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<rect x="2" y="4" width="20" height="7" rx="1.5"/>
					<rect x="2" y="13" width="20" height="7" rx="1.5"/>
					<circle cx="6" cy="7.5" r="1"/>
					<circle cx="6" cy="16.5" r="1"/>
				</svg>
				<span class="flex-1">Hosts</span>
				<span class={badgeStyle}>{hostCount}</span>
			</button>

			<div class="text-[10px] font-bold tracking-[.09em] text-[var(--tx-3)] px-[6px] pt-4 pb-[6px]">GUILD</div>

			<button onclick={() => nav('deploy/members')} class={ns('members')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="8" cy="9" r="3"/>
					<circle cx="16.5" cy="10" r="2.4"/>
					<path d="M2.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5M15 19c0-2 .8-3.4 2.2-4.2"/>
				</svg>
				<span class="flex-1">Members</span>
				<span class={badgeStyle}>{memberCount}</span>
			</button>

			<button onclick={() => nav('deploy/roles')} class={ns('roles')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M12 3l7 3v5c0 4.2-3 7.5-7 9-4-1.5-7-4.8-7-9V6l7-3z"/>
				</svg>
				<span class="flex-1">Roles</span>
			</button>

			<div class="text-[10px] font-bold tracking-[.09em] text-[var(--tx-3)] px-[6px] pt-4 pb-[6px]">SETTINGS</div>

			<button onclick={() => nav('deploy/settings')} class={ns('settings')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
					<circle cx="12" cy="12" r="3.2"/>
					<path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/>
				</svg>
				<span class="flex-1">Guild Settings</span>
			</button>

		{:else}
			<div class="text-[10px] font-bold tracking-[.09em] text-[var(--tx-3)] px-[6px] pt-2 pb-[6px]">PLANNING</div>

			<button onclick={() => nav('planning/board')} class={ns('board')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<rect x="3" y="4" width="18" height="16" rx="2"/>
					<path d="M9 4v16M15 4v16"/>
				</svg>
				<span class="flex-1">Board</span>
			</button>

			<button onclick={() => nav('planning/my-work')} class={ns('my-work')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="12" cy="8" r="3.2"/>
					<path d="M5 20c0-3.4 3.1-6 7-6s7 2.6 7 6"/>
				</svg>
				<span class="flex-1">My Work</span>
				<span class={badgeStyle}>4</span>
			</button>

			<button onclick={() => nav('planning/projects')} class={ns('projects')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M4 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2z"/>
				</svg>
				<span class="flex-1">Projects</span>
				<span class={badgeStyle}>3</span>
			</button>

			<button onclick={() => nav('planning/cycles')} class={ns('cycles')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M21 12a9 9 0 11-3-6.7L21 8"/>
					<path d="M21 3v5h-5"/>
				</svg>
				<span class="flex-1">Cycles</span>
			</button>

			<button onclick={() => nav('planning/views')} class={ns('views')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z"/>
					<circle cx="12" cy="12" r="3"/>
				</svg>
				<span class="flex-1">Views</span>
			</button>
		{/if}
	</div>

	<!-- User panel -->
	<div class="h-[56px] shrink-0 bg-black/[0.22] border-t border-t-[var(--line)] flex items-center gap-[9px] px-[10px]">
		<div class="relative size-[34px] shrink-0">
			<div class="size-[34px] rounded-[11px] bg-gradient-to-br from-[#d98a4a] to-[#b5632c] flex items-center justify-center font-bold text-[13px] text-[#1a0f07]">RY</div>
			<div class="absolute right-[-2px] bottom-[-2px] size-3 rounded-full bg-[var(--ok)] border-[2.5px] border-[var(--sidebar)]"></div>
		</div>
		<div class="min-w-0 flex-1">
			<div class="text-[13px] font-semibold text-[var(--tx)] flex items-center gap-[5px]">Rye
				<svg width="11" height="11" viewBox="0 0 24 24" fill="#e0a83e"><path d="M5 16L3 6l5 4 4-6 4 6 5-4-2 10z"/></svg>
			</div>
			<div class="text-[11px] text-[var(--tx-3)]">Guild Master</div>
		</div>
		<div class="flex gap-0.5 text-[var(--tx-2)]">
			<button
				onclick={() => nav('deploy/settings')}
				aria-label="Guild settings"
				class="size-7 flex items-center justify-center rounded-[7px] cursor-pointer hover:bg-white/5"
			>
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
					<circle cx="12" cy="12" r="3.2"/>
					<path d="M12 2v3M12 19v3M2 12h3M19 12h3"/>
				</svg>
			</button>
		</div>
	</div>
</div>
