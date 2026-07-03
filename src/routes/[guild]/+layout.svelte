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

<div class="bakery-shell flex h-screen w-screen overflow-hidden bg-[var(--main)] text-[var(--tx)] font-bakery antialiased">
	<AppRail {guildId} />
	<GuildSidebar {guildId} {module} activeSection={section} />

	<div class="flex-1 min-w-0 flex flex-col bg-[var(--main)]">
		<TopBar {guildId} {module} {section} onDeploy={() => deployOpen = true} />
		<div class="flex-1 overflow-y-auto">
			{@render children()}
		</div>
	</div>

	<DeployDialog bind:open={deployOpen} {guildId} />
</div>
