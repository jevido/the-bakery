<script lang="ts">
	import { page } from '$app/state';
	import { setContext } from 'svelte';
	import { Toaster } from 'svelte-sonner';
	import AppRail from '$lib/components/bakery/AppRail.svelte';
	import GuildSidebar from '$lib/components/bakery/GuildSidebar.svelte';
	import TopBar from '$lib/components/bakery/TopBar.svelte';
	import DeployDialog from '$lib/components/bakery/DeployDialog.svelte';
	import GuildJourneyDialog from '$lib/components/bakery/GuildJourneyDialog.svelte';

	let { children } = $props();

	const guildId = $derived(page.params.guild ?? '');
	let deployOpen = $state(false);
	let guildJourneyOpen = $state(false);
	let ctaAction = $state<(() => void) | undefined>(undefined);

	setContext('bakery:cta', {
		register(fn: () => void) { ctaAction = fn; },
		unregister() { ctaAction = undefined; },
	});

	// Derive module and section from URL
	const pathParts = $derived(page.url.pathname.split('/').filter(Boolean));
	const module = $derived((pathParts[1] === 'planning' ? 'planning' : pathParts[1] === 'guild' ? 'guild' : 'deploy') as 'deploy' | 'planning' | 'guild');
	const section = $derived(pathParts[2] ?? 'overview');

	const onDeploy = $derived(
		module !== 'deploy' ? undefined :
		(section === 'overview' || section === 'projects') ? () => deployOpen = true :
		ctaAction
	);
</script>

<div class="bakery-shell flex h-screen w-screen overflow-hidden bg-[var(--main)] text-[var(--tx)] font-bakery antialiased">
	<AppRail {guildId} onOpenJourney={() => (guildJourneyOpen = true)} />
	<GuildSidebar {guildId} {module} activeSection={section} />

	<div class="flex-1 min-w-0 flex flex-col bg-[var(--main)]">
		<TopBar {guildId} {module} {section} {onDeploy} />
		<div class="flex-1 overflow-y-auto">
			{@render children()}
		</div>
	</div>

	<DeployDialog bind:open={deployOpen} {guildId} />
	<GuildJourneyDialog bind:open={guildJourneyOpen} />
</div>

<Toaster theme="dark" position="bottom-right" richColors />
