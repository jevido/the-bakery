<script lang="ts">
	import GuildJourneyDialog from '$lib/components/bakery/GuildJourneyDialog.svelte';

	let { data } = $props();
	let journeyOpen = $state(false);
</script>

{#if data.hasNoGuilds}
	<div
		class="bakery-shell flex h-screen w-screen items-center justify-center bg-[var(--main)] text-[var(--tx)] font-bakery"
	>
		<div class="text-center max-w-sm">
			<h1 class="text-[18px] font-heading font-bold mb-2">No guilds yet</h1>
			<p class="text-[13px] text-[var(--tx-3)] mb-5">
				You're not a member of any guild yet — create one, or join an existing one with an invite
				code.
			</p>
			<button
				onclick={() => (journeyOpen = true)}
				class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors cursor-pointer"
			>
				Create or join a guild →
			</button>
		</div>

		<GuildJourneyDialog bind:open={journeyOpen} />
	</div>
{/if}
