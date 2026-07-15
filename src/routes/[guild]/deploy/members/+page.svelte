<script lang="ts">
	import { enhance } from '$app/forms';
	import { toast } from 'svelte-sonner';
	import { ROLES, roles as roleAc } from '$lib/auth/permissions';

	let { data } = $props();

	const myRole = $derived(data.member?.role as string | undefined);
	const canManage = $derived(
		myRole
			? (roleAc[myRole as keyof typeof roleAc]?.authorize({ guild: ['manage_members'] }).success ??
					false)
			: false
	);

	const ASSIGNABLE_ROLES = ROLES.filter((r) => r.id !== 'guild-master');

	const AVATAR_PALETTE = [
		{ bg: 'linear-gradient(140deg,#d98a4a,#b5632c)', fg: '#1a0f07' },
		{ bg: 'linear-gradient(140deg,#b98ce6,#8b5fd0)', fg: '#150a24' },
		{ bg: 'linear-gradient(140deg,#7aa6f5,#4f79d8)', fg: '#0a1226' },
		{ bg: 'linear-gradient(140deg,#52cc96,#2f9e6c)', fg: '#07130c' },
		{ bg: 'linear-gradient(140deg,#c7c34a,#9a962f)', fg: '#161503' },
		{ bg: 'linear-gradient(140deg,#8a9aa5,#5f6f7a)', fg: '#0a1013' }
	];

	function avatarStyle(name: string) {
		let hash = 0;
		for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0;
		return AVATAR_PALETTE[Math.abs(hash) % AVATAR_PALETTE.length];
	}

	function initials(name: string) {
		const parts = name.trim().split(/\s+/);
		return (parts[0]?.[0] ?? '').concat(parts[1]?.[0] ?? '').toUpperCase() || '?';
	}

	function roleMeta(roleId: string) {
		const r = ROLES.find((r) => r.id === roleId);
		return { name: r?.name ?? roleId, color: r?.color ?? 'var(--tx-3)' };
	}

	function formatJoined(createdAt: string | Date) {
		return new Date(createdAt).toLocaleDateString('en-US', { month: 'short', year: 'numeric' });
	}

	async function copyInviteCode() {
		const code = data.organization?.inviteCode;
		if (!code) return;
		await navigator.clipboard.writeText(code);
		toast.success('Invite code copied', { description: code });
	}
</script>

<div class="px-7 py-[22px]">
	<div class="flex items-center justify-between mb-[18px]">
		<div>
			<div class="font-heading font-bold text-[23px]">Members</div>
			<div class="text-[13px] text-[var(--tx-2)] mt-0.5">
				{data.members.length} members in this guild
			</div>
		</div>
		{#if canManage}
			<button
				onclick={copyInviteCode}
				class="flex items-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] px-4 py-[10px] text-[13.5px] font-bold cursor-pointer"
			>
				<svg
					width="15"
					height="15"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.3"
					><path d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2" /><circle cx="9" cy="7" r="4" /><path
						d="M22 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"
					/></svg
				>
				Copy invite code
			</button>
		{/if}
	</div>

	<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
		<div
			class="grid grid-cols-[2fr_1.3fr_1fr_140px] gap-[14px] px-[18px] py-[10px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]"
		>
			<div>MEMBER</div>
			<div>ROLE</div>
			<div>JOINED</div>
			<div></div>
		</div>

		{#each data.members as m (m.id)}
			{@const av = avatarStyle(m.user.name)}
			{@const rm = roleMeta(m.role)}
			{@const isMaster = m.role === 'guild-master'}
			<div
				class="grid grid-cols-[2fr_1.3fr_1fr_140px] gap-[14px] items-center px-[18px] py-[13px] border-b border-b-[var(--line)]"
			>
				<!-- Avatar + name -->
				<div class="flex items-center gap-3">
					<div
						class="size-9 rounded-[9px] flex items-center justify-center font-heading font-bold text-[14px] shrink-0"
						style:background={av.bg}
						style:color={av.fg}
					>
						{initials(m.user.name)}
					</div>
					<div>
						<div class="text-[14px] font-semibold text-[var(--tx)]">{m.user.name}</div>
						<div class="text-[11.5px] text-[var(--tx-3)]">{m.user.email}</div>
					</div>
				</div>

				<!-- Role badge -->
				<div>
					<span
						class="text-[12.5px] font-semibold px-[10px] py-1 rounded-[20px]"
						style:background="{rm.color}22"
						style:color={rm.color}>{rm.name}</span
					>
				</div>

				<!-- Joined -->
				<div class="text-[12.5px] text-[var(--tx-2)]">{formatJoined(m.createdAt)}</div>

				<!-- Actions -->
				<div class="flex justify-end gap-1">
					{#if canManage && !isMaster}
						<form method="post" action="?/updateRole" use:enhance>
							<input type="hidden" name="memberId" value={m.id} />
							<select
								name="role"
								value={m.role}
								onchange={(e) => e.currentTarget.form?.requestSubmit()}
								class="px-[8px] py-[6px] text-[12px] bg-[var(--card-2)] border border-[var(--line)] rounded-[7px] text-[var(--tx-2)] cursor-pointer font-semibold"
							>
								{#each ASSIGNABLE_ROLES as r (r.id)}
									<option value={r.id}>{r.name}</option>
								{/each}
							</select>
						</form>
						<form method="post" action="?/removeMember" use:enhance>
							<input type="hidden" name="memberId" value={m.id} />
							<button
								type="submit"
								class="px-[11px] py-[6px] text-[12px] bg-[var(--card-2)] border border-[var(--line)] rounded-[7px] text-[var(--tx-2)] cursor-pointer font-semibold"
								>Remove</button
							>
						</form>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>
