<script lang="ts">
	import { ROLES } from '$lib/auth/permissions';
	import { initialsFrom } from '$lib/utils';

	let { data } = $props();

	const user = $derived(data.user);
	const userInitials = $derived(initialsFrom(user.name));
	const roleId = $derived(data.member?.role as string | undefined);
	const roleLabel = $derived(ROLES.find((r) => r.id === roleId)?.name ?? roleId ?? '—');
	const guildName = $derived(data.organization?.name ?? 'this guild');
</script>

<svelte:head><title>Account · {guildName} — The Bakery</title></svelte:head>

<div class="px-7 py-[22px] max-w-[560px]">
	<div class="font-heading font-bold text-[23px] mb-1">Account</div>
	<div class="text-[13px] text-[var(--tx-2)] mb-[22px]">Your account details.</div>

	<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
		<div class="p-[18px] flex items-center gap-[14px]">
			{#if user.image}
				<img
					src={user.image}
					alt={user.name}
					class="size-[52px] rounded-[14px] object-cover shrink-0"
				/>
			{:else}
				<div
					class="size-[52px] rounded-[14px] bg-gradient-to-br from-[#d98a4a] to-[#b5632c] flex items-center justify-center font-bold text-[18px] text-[#1a0f07] shrink-0"
				>
					{userInitials}
				</div>
			{/if}
			<div class="min-w-0">
				<div class="text-[16px] font-semibold text-[var(--tx)]">{user.name}</div>
				<div class="text-[13px] text-[var(--tx-2)]">{user.email}</div>
			</div>
		</div>

		<div
			class="border-t border-t-[var(--line)] px-[18px] py-[14px] flex justify-between gap-[10px]"
		>
			<span class="text-[12.5px] text-[var(--tx-2)]">Role in {guildName}</span>
			<span class="text-[12.5px] font-semibold text-[var(--tx)]">{roleLabel}</span>
		</div>
	</div>
</div>
