<script lang="ts">
	import { goto } from '$app/navigation';
	import { guildStore } from '$lib/stores/guilds.svelte';

	let { guildId, onOpenJourney }: { guildId: string; onOpenJourney: () => void } = $props();

	function selectGuild(id: string) {
		goto(`/${id}/deploy/overview`);
	}
</script>

<div class="w-[76px] shrink-0 bg-[var(--rail)] flex flex-col items-center pt-3 pb-[14px] gap-2 border-r border-r-black/40 z-[2]">
	<!-- Logo -->
	<div
		title="The Bakery"
		class="size-12 rounded-[15px] bg-gradient-to-br from-[var(--grn-2)] to-[var(--grn)] flex items-center justify-center font-heading font-extrabold text-2xl text-white shadow-[0_4px_16px_var(--grn-dim)] cursor-pointer"
		role="button"
		tabindex="0"
		onclick={() => selectGuild(guildId)}
		onkeydown={(e) => e.key === 'Enter' && selectGuild(guildId)}
	>B</div>

	<div class="w-8 h-0.5 bg-white/[0.09] rounded-sm my-0.5"></div>

	<!-- Guild list -->
	{#each guildStore.list as gd (gd.id)}
		{@const active = gd.slug === guildId}
		<div
			title={gd.name}
			onclick={() => selectGuild(gd.slug)}
			onkeydown={(e) => e.key === 'Enter' && selectGuild(gd.slug)}
			role="button"
			tabindex="0"
			class="relative size-12 cursor-pointer flex items-center justify-center"
		>
			<!-- Active pill -->
			<div
				class="absolute left-[-14px] w-1 rounded-r-[4px] bg-[var(--tx)] transition-all duration-150"
				style:height={active ? '32px' : '0px'}
			></div>
			<!-- Avatar -->
			<div
				class="size-12 flex items-center justify-center font-heading font-bold text-[19px] transition-all duration-150"
				class:rounded-[15px]={active}
				class:rounded-[24px]={!active}
				style:background={active ? 'linear-gradient(150deg, var(--grn-2), var(--grn))' : 'var(--card-2)'}
				style:color={active ? '#07130c' : 'var(--tx)'}
				style:border={active ? 'none' : '1px solid var(--line)'}
			>{gd.name.trim()[0]?.toUpperCase() ?? '?'}</div>
		</div>
	{/each}

	<!-- Add guild -->
	<div
		title="Create or join a guild"
		role="button"
		tabindex="0"
		onclick={onOpenJourney}
		onkeydown={(e) => e.key === 'Enter' && onOpenJourney()}
		class="size-12 rounded-[15px] bg-white/5 text-[var(--grn-2)] flex items-center justify-center text-[26px] font-normal cursor-pointer hover:bg-white/[0.08] transition-colors"
	>+</div>

	<!-- Discover -->
	<div
		title="Discover"
		class="size-12 rounded-[15px] bg-white/5 text-[var(--tx-2)] flex items-center justify-center cursor-pointer"
	>
		<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
			<circle cx="12" cy="12" r="9"/>
			<path d="M15 9l-2.5 5L9 15l2.5-5z" fill="currentColor" stroke="none"/>
		</svg>
	</div>

	<!-- Settings -->
	<div
		title="Settings"
		class="mt-auto size-10 rounded-[12px] bg-white/[0.04] text-[var(--tx-2)] flex items-center justify-center cursor-pointer"
	>
		<svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
			<circle cx="12" cy="12" r="3.2"/>
			<path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/>
		</svg>
	</div>
</div>
