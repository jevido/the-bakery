<script lang="ts">
	import { page } from '$app/state';
	import AppRail from '$lib/components/bakery/AppRail.svelte';
	import GuildSidebar from '$lib/components/bakery/GuildSidebar.svelte';
	import TopBar from '$lib/components/bakery/TopBar.svelte';
	import DeployDialog from '$lib/components/bakery/DeployDialog.svelte';

	let { children } = $props();

	const guildId = $derived(page.params.guild ?? '');
	let deployOpen = $state(false);

	// Derive module and section from URL
	const pathParts = $derived(page.url.pathname.split('/').filter(Boolean));
	const module = $derived((pathParts[1] === 'planning' ? 'planning' : pathParts[1] === 'guild' ? 'guild' : 'deploy') as 'deploy' | 'planning' | 'guild');
	// section is the 3rd segment (e.g. 'overview', 'apps', 'hosts')
	// or for app detail it's still 'apps'
	const section = $derived(pathParts[2] ?? 'overview');
</script>

<div
	class="bakery-shell"
	style="
		display: flex;
		height: 100vh;
		width: 100vw;
		overflow: hidden;
		--rail: #0a0b10;
		--sidebar: #0c0d13;
		--main: #0a0b0f;
		--topbar: #0b0c11;
		--panel: #101119;
		--card: #12141b;
		--card-2: #191c25;
		--line: rgba(255,255,255,.06);
		--line-2: rgba(255,255,255,.11);
		--tx: #e7e9ee;
		--tx-2: #8f94a3;
		--tx-3: #5c6170;
		--grn: #3fb984;
		--grn-2: #52cc96;
		--grn-dim: rgba(63,185,132,.14);
		--grn-line: rgba(63,185,132,.28);
		--ok: #34d399;
		--ok-2: #5ee0ab;
		--amber: #f59e0b;
		--violet: #8b5cf6;
		color: var(--tx);
		font-family: 'Instrument Sans', system-ui, sans-serif;
		-webkit-font-smoothing: antialiased;
		background: var(--main);
	"
>
	<AppRail {guildId} />
	<GuildSidebar {guildId} {module} activeSection={section} />

	<div style="flex:1;min-width:0;display:flex;flex-direction:column;background:var(--main);">
		<TopBar {guildId} {module} {section} onDeploy={() => deployOpen = true} />
		<div style="flex:1;overflow-y:auto;">
			{@render children()}
		</div>
	</div>

	<DeployDialog bind:open={deployOpen} {guildId} />
</div>
