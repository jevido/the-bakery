<script lang="ts">
	import { page } from '$app/state';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	// Full settings form wiring (name edit, invite regenerate, delete) is
	// Phase 01 task 15 — this only sources the invite code from the real
	// organization instead of the mock, per task 09.
	const guild = $derived(page.data.organization as { name: string; inviteCode: string } | undefined);
	let deleteDialogOpen = $state(false);
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
				<input id="invite-code" value={guild?.inviteCode ?? ''} readonly class="flex-1 bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] font-mono-jb text-[13px] text-[var(--tx-2)]" />
				<button class="px-[14px] py-[10px] bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] text-[13px] text-[var(--tx-2)] cursor-pointer font-semibold">Regenerate</button>
			</div>
		</div>
	</div>

	<div class="bg-[rgba(229,101,75,.08)] border border-[rgba(229,101,75,.2)] rounded-[13px] p-[18px]">
		<div class="text-[14px] font-bold text-[#f0836b] mb-1">Danger zone</div>
		<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">These actions are irreversible. Please be certain before proceeding.</div>
		<button
			onclick={() => (deleteDialogOpen = true)}
			class="px-4 py-[9px] bg-[rgba(229,101,75,.14)] border border-[rgba(229,101,75,.3)] rounded-[8px] text-[#f0836b] text-[13px] font-bold cursor-pointer"
		>Delete guild…</button>
	</div>
</div>

<AlertDialog.Root bind:open={deleteDialogOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Delete {guild?.name}?</AlertDialog.Title>
			<AlertDialog.Description>This action is permanent and cannot be undone. All apps, hosts, members, and settings will be irreversibly removed.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action>Delete guild</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
