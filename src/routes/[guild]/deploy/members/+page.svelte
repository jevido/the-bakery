<script lang="ts">
	import { page } from '$app/state';
	import { MEMBERS, ROLES } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const members = $derived(MEMBERS[guildId] ?? []);

	function roleColor(roleName: string) {
		const r = ROLES.find(r => r.name === roleName);
		return r?.color ?? 'var(--tx-3)';
	}
</script>

<div class="px-7 py-[22px]">
	<div class="flex items-center justify-between mb-[18px]">
		<div>
			<div class="font-heading font-bold text-[23px]">Members</div>
			<div class="text-[13px] text-[var(--tx-2)] mt-0.5">{members.length} members in this guild</div>
		</div>
		<button class="flex items-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] px-4 py-[10px] text-[13.5px] font-bold cursor-pointer">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"/></svg>
			Invite member
		</button>
	</div>

	<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
		<div class="grid grid-cols-[2fr_1.3fr_1fr_80px] gap-[14px] px-[18px] py-[10px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
			<div>MEMBER</div>
			<div>ROLE</div>
			<div>JOINED</div>
			<div></div>
		</div>

		{#each members as m}
			{@const rc = roleColor(m.role)}
			<div class="grid grid-cols-[2fr_1.3fr_1fr_80px] gap-[14px] items-center px-[18px] py-[13px] border-b border-b-[var(--line)]">
				<!-- Avatar + name -->
				<div class="flex items-center gap-3">
					<div class="size-9 rounded-[9px] flex items-center justify-center font-heading font-bold text-[14px] shrink-0" style:background={m.avBg} style:color={m.avColor}>{m.initials}</div>
					<div>
						<div class="text-[14px] font-semibold text-[var(--tx)]">{m.name}</div>
						<div class="text-[11.5px] text-[var(--tx-3)]">{m.handle}</div>
					</div>
				</div>

				<!-- Role badge -->
				<div>
					<span class="text-[12.5px] font-semibold px-[10px] py-1 rounded-[20px]" style:background="{rc}22" style:color={rc}>{m.role}</span>
				</div>

				<!-- Joined -->
				<div class="text-[12.5px] text-[var(--tx-2)]">{m.joinedAt ?? '—'}</div>

				<!-- Actions -->
				<div class="flex justify-end gap-1">
					{#if !m.master}
						<button class="px-[11px] py-[6px] text-[12px] bg-[var(--card-2)] border border-[var(--line)] rounded-[7px] text-[var(--tx-2)] cursor-pointer font-semibold">Edit</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>
