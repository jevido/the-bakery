<script lang="ts">
	let { data } = $props();
</script>

{#if data.hasNoGuilds}
	<div
		class="flex h-screen w-screen items-center justify-center bg-[var(--main)] text-[var(--tx)] font-bakery"
	>
		<div class="text-center max-w-sm">
			<h1 class="text-[18px] font-heading font-bold mb-2">No guilds yet</h1>
			<p class="text-[13px] text-[var(--tx-3)]">
				You're not a member of any guild. Guild creation is coming in a later phase — for now,
				ask someone to add you to theirs.
			</p>
		</div>
	</div>
{/if}
