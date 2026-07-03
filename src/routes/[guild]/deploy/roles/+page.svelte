<script lang="ts">
	import { ROLES, PERM_GROUPS, ROLE_PERMS } from '$lib/data/bakery';

	let selectedRole = $state(ROLES[0]);
	const rolePerms = $derived(new Set(ROLE_PERMS[selectedRole.id] ?? []));
</script>

<div class="px-7 py-[22px]">
	<div class="mb-[18px]">
		<div class="font-heading font-bold text-[23px]">Roles & Positions</div>
		<div class="text-[13px] text-[var(--tx-2)] mt-0.5">Configure what each position can do in the guild. Positions are ranked — a baker cannot edit roles above their own.</div>
	</div>

	<div class="grid grid-cols-[260px_1fr] gap-4 items-start">
		<!-- Left: role list -->
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div class="px-4 py-3 border-b border-b-[var(--line)] text-[12px] font-bold tracking-[.05em] text-[var(--tx-3)]">POSITIONS</div>
			{#each ROLES as r}
				{@const active = r.id === selectedRole.id}
				<button
					onclick={() => selectedRole = r}
					class="flex items-center gap-3 w-full px-4 py-[13px] border-b border-b-[var(--line)] cursor-pointer border-l-[3px] text-left {active ? 'bg-[rgba(63,185,132,.08)] border-l-[var(--grn)]' : 'bg-transparent border-l-transparent'}"
				>
					<div class="size-[10px] rounded-full shrink-0" style:background={r.color}></div>
					<div class="flex-1 min-w-0">
						<div class="text-[13.5px] {active ? 'font-bold text-[var(--tx)]' : 'font-semibold text-[var(--tx-2)]'}">{r.name}</div>
						<div class="text-[11.5px] text-[var(--tx-3)]">{r.count} member{r.count !== 1 ? 's' : ''}</div>
					</div>
					{#if r.master}
						<span class="text-[10px] font-bold px-[7px] py-[2px] rounded-[5px] bg-[rgba(224,168,62,.14)] text-[#e0a83e]">OWNER</span>
					{/if}
				</button>
			{/each}
		</div>

		<!-- Right: permissions panel -->
		<div>
			<!-- Role header -->
			<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[16px_18px] mb-3">
				<div class="flex items-center gap-[11px] mb-2">
					<div class="size-3 rounded-full" style:background={selectedRole.color}></div>
					<div class="text-[16px] font-bold">{selectedRole.name}</div>
				</div>
				<div class="text-[13px] text-[var(--tx-2)] leading-[1.5]">{selectedRole.note}</div>
			</div>

			<!-- Perm groups -->
			{#each PERM_GROUPS as g}
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden mb-[10px]">
					<div class="px-[18px] py-[10px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.06em] text-[var(--tx-3)]">{g.name}</div>
					{#each g.perms as p}
						{@const has = rolePerms.has(p.id) || rolePerms.has('administrator')}
						<div class="flex items-center gap-[13px] px-[18px] py-3 border-b border-b-[var(--line)]">
							<!-- Toggle -->
							<div
								class="w-[34px] h-5 rounded-[10px] shrink-0 relative {has ? 'bg-[var(--grn)]' : 'bg-white/[0.08]'} {selectedRole.master ? 'cursor-not-allowed' : 'cursor-pointer'}"
							>
								<div
									class="size-[14px] rounded-full bg-white absolute top-[3px] transition-[left] duration-150"
									style:left={has ? '17px' : '3px'}
								></div>
							</div>
							<div class="flex-1 min-w-0">
								<div class="text-[13.5px] font-semibold" style:color={p.danger ? '#f0836b' : 'var(--tx)'}>{p.label}</div>
								<div class="text-[12px] text-[var(--tx-3)] mt-[1px]">{p.desc}</div>
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	</div>
</div>
