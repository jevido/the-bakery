<script lang="ts">
	import { page } from '$app/state';
	import { setContext } from 'svelte';
	import { Toaster } from 'svelte-sonner';
	import AppRail from '$lib/components/bakery/AppRail.svelte';
	import GuildSidebar from '$lib/components/bakery/GuildSidebar.svelte';
	import TopBar from '$lib/components/bakery/TopBar.svelte';
	import DeployDialog from '$lib/components/bakery/DeployDialog.svelte';
	import GuildJourneyDialog from '$lib/components/bakery/GuildJourneyDialog.svelte';
	import NotFound from '$lib/components/bakery/NotFound.svelte';

	let { children, data } = $props();

	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(data.organization);
	const guildName = $derived(guild?.name ?? guildId);
	let deployOpen = $state(false);
	let guildJourneyOpen = $state(false);
	let ctaAction = $state<(() => void) | undefined>(undefined);

	setContext('bakery:cta', {
		register(fn: () => void) { ctaAction = fn; },
		unregister() { ctaAction = undefined; },
	});

	const pathParts = $derived(page.url.pathname.split('/').filter(Boolean));
	const section = $derived(pathParts[2] ?? 'overview');

	const sectionLabels: Record<string, string> = {
		overview: 'Dashboard', projects: 'Projects', hosts: 'Hosts',
		sources: 'Sources', networks: 'Networks', storage: 'Storage',
		members: 'Members', roles: 'Roles', settings: 'Guild Settings',
	};
	const crumb = $derived(sectionLabels[section] ?? section);
	const pageTitle = $derived(`${crumb} · ${guildName} — The Bakery`);

	const onDeploy = $derived(
		(section === 'overview' || section === 'projects') ? () => deployOpen = true : ctaAction
	);
</script>

<svelte:head><title>{pageTitle}</title></svelte:head>

<div class="bakery-shell flex h-screen w-screen overflow-hidden bg-[var(--main)] text-[var(--tx)] font-bakery antialiased">
	<AppRail {guildId} onOpenJourney={() => (guildJourneyOpen = true)} />
	<GuildSidebar {guildId} activeSection={section} />

	<div class="flex-1 min-w-0 flex flex-col bg-[var(--main)]">
		<TopBar {guildId} {section} {onDeploy} />
		<div class="flex-1 overflow-y-auto">
			{#if guild}
				{@render children()}
			{:else}
				<NotFound label="Guild not found" detail="There's no guild at /{guildId}" />
			{/if}
		</div>
	</div>

	<DeployDialog bind:open={deployOpen} {guildId} />
	<GuildJourneyDialog bind:open={guildJourneyOpen} />
</div>

<Toaster theme="dark" position="bottom-right" richColors />
