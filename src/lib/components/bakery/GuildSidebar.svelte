<script lang="ts">
	import { GUILDS, MEMBERS } from '$lib/data/bakery';
	import { goto } from '$app/navigation';
	import type { Snippet } from 'svelte';

	let {
		guildId,
		module,
		activeSection,
		children,
	}: {
		guildId: string;
		module: 'deploy' | 'planning' | 'guild';
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

	const navBase = 'display:flex;align-items:center;gap:11px;padding:8px 10px;border-radius:8px;cursor:pointer;font-size:13.5px;font-weight:500;color:var(--tx-2);margin-bottom:2px;';
	const navOn = 'display:flex;align-items:center;gap:11px;padding:8px 10px;border-radius:8px;cursor:pointer;font-size:13.5px;font-weight:600;color:var(--grn);margin-bottom:2px;background:var(--grn-dim);';
	function ns(s: string) { return isActive(s) ? navOn : navBase; }

	const moduleIcons: Record<string, string> = {
		deploy: 'M20 7l-8-4-8 4v10l8 4 8-4V7zM4 7l8 4 8-4M12 21V11',
		planning: 'M3 4h18v4H3zM3 12h18v4H3zM3 20h18',
		guild: 'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75',
	};
	const moduleNames: Record<string, string> = { deploy: 'Deploy', planning: 'Planning', guild: 'Guild' };
	const moduleDescs: Record<string, string> = { deploy: 'Projects, sources & infra', planning: 'Board, cycles & sprints', guild: 'Members, roles & settings' };

	const appCount = $derived(apps.length);
	const hostCount = $derived(guild?.hosts?.length ?? 0);

	const badgeStyle = 'font-size:11px;font-weight:700;padding:1px 7px;border-radius:20px;background:rgba(255,255,255,.07);color:var(--tx-2);';

	const menuHover = 'background:rgba(255,255,255,.05);';
	const guildMenuItems = [
		{ icon: 'M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5c-1.66 0-3 1.34-3 3s1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z', label: 'Invite People', color: 'var(--tx)' },
		{ icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z', label: 'Guild Settings', color: 'var(--tx)' },
		{ icon: 'M17 7l-1.41 1.41L18.17 11H8v2h10.17l-2.58 2.58L17 17l5-5zM4 5h8V3H4c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h8v-2H4V5z', label: 'Leave Guild', color: '#f0836b' },
	];
</script>

<svelte:window onclick={handleWindowClick} />

<div bind:this={sidebarEl} style="
	width: 248px;
	flex: none;
	background: var(--sidebar);
	display: flex;
	flex-direction: column;
	border-right: 1px solid var(--line);
	z-index: 1;
	position: relative;
">
	<!-- Guild header -->
	<div style="position:relative;flex:none;">
		<div
			onclick={toggleGuildMenu}
			onkeydown={(e) => e.key === 'Enter' && toggleGuildMenu()}
			role="button"
			tabindex="0"
			style="height:52px;padding:0 16px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid var(--line);cursor:pointer;"
		>
			<div style="min-width:0;">
				<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:15px;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">
					{guild?.name ?? guildId}
				</div>
			</div>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="2">
				<path d={guildMenuOpen ? 'M18 15l-6-6-6 6' : 'M6 9l6 6 6-6'}/>
			</svg>
		</div>

		{#if guildMenuOpen}
			<div style="position:absolute;left:10px;right:10px;top:50px;z-index:39;background:var(--panel);border:1px solid var(--line-2);border-radius:11px;padding:6px;box-shadow:0 18px 46px rgba(0,0,0,.55);">
				{#each guildMenuItems as m}
					<div
						onclick={closeAll}
						onkeydown={(e) => e.key === 'Enter' && closeAll()}
						role="button"
						tabindex="0"
						style="display:flex;align-items:center;gap:10px;padding:9px 10px;border-radius:8px;cursor:pointer;"
					>
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={m.color} stroke-width="1.8" style="flex:none">
							<path d={m.icon}/>
						</svg>
						<span style="flex:1;font-size:13px;font-weight:600;color:{m.color}">{m.label}</span>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Module dropdown -->
	<div style="padding:12px 12px 6px;position:relative;">
		<div style="font-size:10px;font-weight:700;letter-spacing:.09em;color:var(--tx-3);padding:0 6px 7px">MODULE</div>
		<div
			onclick={toggleModuleMenu}
			onkeydown={(e) => e.key === 'Enter' && toggleModuleMenu()}
			role="button"
			tabindex="0"
			style="display:flex;align-items:center;gap:9px;padding:9px 11px;border-radius:9px;background:var(--card);border:1px solid var(--line-2);cursor:pointer;"
		>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="1.9">
				<path d={moduleIcons[module]}/>
			</svg>
			<div style="font-size:13.5px;font-weight:700;color:var(--tx);flex:1">{moduleNames[module]}</div>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="2">
				<path d="M8 9l4-4 4 4M8 15l4 4 4-4"/>
			</svg>
		</div>

		{#if moduleMenuOpen}
			<div style="position:absolute;left:12px;right:12px;top:64px;z-index:30;background:var(--panel);border:1px solid var(--line-2);border-radius:11px;padding:6px;box-shadow:0 16px 44px rgba(0,0,0,.55);">
				{#each (['deploy', 'planning', 'guild'] as const) as m}
					{@const defaultSection = m === 'deploy' ? 'overview' : m === 'planning' ? 'board' : 'members'}
					<div
						onclick={() => { nav(`${m}/${defaultSection}`); }}
						onkeydown={(e) => e.key === 'Enter' && nav(`${m}/${defaultSection}`)}
						role="button"
						tabindex="0"
						style="display:flex;align-items:center;gap:10px;padding:10px 11px;border-radius:8px;cursor:pointer;{module === m ? 'background:var(--grn-dim);' : ''}"
					>
						<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="{module === m ? 'var(--grn)' : 'var(--tx-2)'}" stroke-width="1.8" style="flex:none">
							<path d={moduleIcons[m]}/>
						</svg>
						<div style="flex:1;min-width:0;">
							<div style="font-size:13.5px;font-weight:600;color:{module === m ? 'var(--grn)' : 'var(--tx)'}">{moduleNames[m]}</div>
							<div style="font-size:11px;color:var(--tx-3)">{moduleDescs[m]}</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Nav -->
	<div style="flex:1;overflow-y:auto;padding:10px 12px 12px;">
		{#if module === 'deploy'}
			<button onclick={() => nav('deploy/overview')} style={ns('overview')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<rect x="3" y="3" width="7" height="9" rx="1.5"/>
					<rect x="14" y="3" width="7" height="5" rx="1.5"/>
					<rect x="14" y="12" width="7" height="9" rx="1.5"/>
					<rect x="3" y="16" width="7" height="5" rx="1.5"/>
				</svg>
				<span style="flex:1">Dashboard</span>
			</button>

			<button onclick={() => nav('deploy/projects')} style={ns('projects')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M4 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2z"/>
				</svg>
				<span style="flex:1">Projects</span>
			</button>

			<button onclick={() => nav('deploy/sources')} style={ns('sources')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<polyline points="16 18 22 12 16 6"/>
					<polyline points="8 6 2 12 8 18"/>
				</svg>
				<span style="flex:1">Sources</span>
			</button>

			<button onclick={() => nav('deploy/networks')} style={ns('networks')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="12" cy="5" r="2"/>
					<circle cx="5" cy="19" r="2"/>
					<circle cx="19" cy="19" r="2"/>
					<path d="M12 7v4M12 11l-5 6M12 11l5 6"/>
				</svg>
				<span style="flex:1">Networks</span>
			</button>

			<button onclick={() => nav('deploy/storage')} style={ns('storage')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<ellipse cx="12" cy="6" rx="7" ry="3"/>
					<path d="M5 6v12c0 1.6 3.1 3 7 3s7-1.4 7-3V6"/>
					<path d="M5 12c0 1.6 3.1 3 7 3s7-1.4 7-3"/>
				</svg>
				<span style="flex:1">Storage</span>
			</button>

		{:else if module === 'guild'}
			<div style="font-size:10px;font-weight:700;letter-spacing:.09em;color:var(--tx-3);padding:8px 6px 6px">GUILD</div>

			<button onclick={() => nav('guild/members')} style={ns('members')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="8" cy="9" r="3"/>
					<circle cx="16.5" cy="10" r="2.4"/>
					<path d="M2.5 19c0-3 2.5-5 5.5-5s5.5 2 5.5 5M15 19c0-2 .8-3.4 2.2-4.2"/>
				</svg>
				<span style="flex:1">Members</span>
				<span style={badgeStyle}>{memberCount}</span>
			</button>

			<button onclick={() => nav('guild/roles')} style={ns('roles')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M12 3l7 3v5c0 4.2-3 7.5-7 9-4-1.5-7-4.8-7-9V6l7-3z"/>
				</svg>
				<span style="flex:1">Roles</span>
			</button>

			<div style="font-size:10px;font-weight:700;letter-spacing:.09em;color:var(--tx-3);padding:16px 6px 6px">SETTINGS</div>

			<button onclick={() => nav('guild/settings')} style={ns('settings')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
					<circle cx="12" cy="12" r="3.2"/>
					<path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/>
				</svg>
				<span style="flex:1">Guild Settings</span>
			</button>

		{:else}
			<div style="font-size:10px;font-weight:700;letter-spacing:.09em;color:var(--tx-3);padding:8px 6px 6px">PLANNING</div>

			<button onclick={() => nav('planning/board')} style={ns('board')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<rect x="3" y="4" width="18" height="16" rx="2"/>
					<path d="M9 4v16M15 4v16"/>
				</svg>
				<span style="flex:1">Board</span>
			</button>

			<button onclick={() => nav('planning/my-work')} style={ns('my-work')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<circle cx="12" cy="8" r="3.2"/>
					<path d="M5 20c0-3.4 3.1-6 7-6s7 2.6 7 6"/>
				</svg>
				<span style="flex:1">My Work</span>
				<span style={badgeStyle}>4</span>
			</button>

			<button onclick={() => nav('planning/projects')} style={ns('projects')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M4 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v9a2 2 0 01-2 2H6a2 2 0 01-2-2z"/>
				</svg>
				<span style="flex:1">Projects</span>
				<span style={badgeStyle}>3</span>
			</button>

			<button onclick={() => nav('planning/cycles')} style={ns('cycles')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M21 12a9 9 0 11-3-6.7L21 8"/>
					<path d="M21 3v5h-5"/>
				</svg>
				<span style="flex:1">Cycles</span>
			</button>

			<button onclick={() => nav('planning/views')} style={ns('views')}>
				<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
					<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z"/>
					<circle cx="12" cy="12" r="3"/>
				</svg>
				<span style="flex:1">Views</span>
			</button>
		{/if}
	</div>

	<!-- User panel -->
	<div style="height:56px;flex:none;background:rgba(0,0,0,.22);border-top:1px solid var(--line);display:flex;align-items:center;gap:9px;padding:0 10px;">
		<div style="position:relative;width:34px;height:34px;flex:none;">
			<div style="width:34px;height:34px;border-radius:11px;background:linear-gradient(140deg,#d98a4a,#b5632c);display:flex;align-items:center;justify-content:center;font-weight:700;font-size:13px;color:#1a0f07">RY</div>
			<div style="position:absolute;right:-2px;bottom:-2px;width:12px;height:12px;border-radius:50%;background:var(--ok);border:2.5px solid var(--sidebar);"></div>
		</div>
		<div style="min-width:0;flex:1;">
			<div style="font-size:13px;font-weight:600;color:var(--tx);display:flex;align-items:center;gap:5px;">Rye
				<svg width="11" height="11" viewBox="0 0 24 24" fill="#e0a83e"><path d="M5 16L3 6l5 4 4-6 4 6 5-4-2 10z"/></svg>
			</div>
			<div style="font-size:11px;color:var(--tx-3);">Guild Master</div>
		</div>
		<div style="display:flex;gap:2px;color:var(--tx-2);">
			<div style="width:28px;height:28px;display:flex;align-items:center;justify-content:center;border-radius:7px;cursor:pointer;">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
					<circle cx="12" cy="12" r="3.2"/>
					<path d="M12 2v3M12 19v3M2 12h3M19 12h3"/>
				</svg>
			</div>
		</div>
	</div>
</div>

<style>
	button {
		all: unset;
		box-sizing: border-box;
		width: 100%;
		text-align: left;
	}
	button:hover {
		background: rgba(255,255,255,.05);
		border-radius: 8px;
	}
</style>
