<script lang="ts">
	import { guildStore } from '$lib/stores/guilds.svelte';
	import { goto } from '$app/navigation';

	let { label = 'Not found', detail = '' }: { label?: string; detail?: string } = $props();

	const firstGuild = $derived(guildStore.list[0]);
</script>

<div class="flex flex-col items-center justify-center h-full gap-6 text-center px-8 select-none">
	<div class="relative">
		<div class="font-heading font-extrabold text-[120px] leading-none tracking-[-0.04em] text-white/[0.04] pointer-events-none">
			404
		</div>
		<div class="absolute inset-0 flex items-center justify-center">
			<div class="size-16 rounded-[20px] bg-[var(--card)] border border-[var(--line)] flex items-center justify-center shadow-[0_8px_32px_rgba(0,0,0,.4)]">
				<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.5">
					<path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
					<line x1="12" y1="9" x2="12" y2="13"/>
					<line x1="12" y1="17" x2="12.01" y2="17"/>
				</svg>
			</div>
		</div>
	</div>

	<div>
		<div class="font-heading font-bold text-[22px] tracking-[-0.01em] text-[var(--tx)]">{label}</div>
		{#if detail}
			<div class="font-mono-jb text-[12px] text-[var(--tx-3)] mt-[6px]">{detail}</div>
		{/if}
	</div>

	{#if firstGuild}
		<button
			onclick={() => goto(`/${firstGuild.id}/deploy/overview`)}
			class="flex items-center gap-2 px-4 py-2 rounded-[9px] bg-[var(--card)] border border-[var(--line)] text-[13px] text-[var(--tx-2)] hover:text-[var(--tx)] hover:border-[var(--line-2)] transition-colors cursor-pointer"
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M19 12H5M12 5l-7 7 7 7"/>
			</svg>
			Back to {firstGuild.name}
		</button>
	{/if}
</div>
