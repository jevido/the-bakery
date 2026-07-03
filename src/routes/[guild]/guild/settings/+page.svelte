<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS } from '$lib/data/bakery';
	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
</script>

<div class="px-7 py-[22px] max-w-[560px]">
	<div class="font-heading font-bold text-[23px] mb-1">Guild Settings</div>
	<div class="text-[13px] text-[var(--tx-2)] mb-[22px]">Manage settings for {guild?.name ?? 'this guild'}.</div>

	<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden mb-[14px]">
		<div class="px-[18px] py-3 border-b border-b-[var(--line)] text-[12px] font-bold tracking-[.05em] text-[var(--tx-3)]">GENERAL</div>
		<div class="p-[18px]">
			<label for="guild-name" class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]">Guild name</label>
			<input id="guild-name" value={guild?.name ?? ''} class="w-full bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] text-[14px] text-[var(--tx)]" />
			<label for="invite-code" class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px] mt-[14px]">Invite code</label>
			<div class="flex gap-2">
				<input id="invite-code" value={guild?.invite ?? ''} readonly class="flex-1 bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] font-mono-jb text-[13px] text-[var(--tx-2)]" />
				<button class="px-[14px] py-[10px] bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] text-[13px] text-[var(--tx-2)] cursor-pointer font-semibold">Regenerate</button>
			</div>
		</div>
	</div>

	<div class="bg-[rgba(229,101,75,.08)] border border-[rgba(229,101,75,.2)] rounded-[13px] p-[18px]">
		<div class="text-[14px] font-bold text-[#f0836b] mb-1">Danger zone</div>
		<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">These actions are irreversible. Please be certain before proceeding.</div>
		<button class="px-4 py-[9px] bg-[rgba(229,101,75,.14)] border border-[rgba(229,101,75,.3)] rounded-[8px] text-[#f0836b] text-[13px] font-bold cursor-pointer">Delete guild…</button>
	</div>
</div>
