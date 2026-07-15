<script lang="ts">
	import { page } from '$app/state';
	import { superForm } from 'sveltekit-superforms';
	import { zodClient } from '$lib/forms/zod4-adapter';
	import { Field, Control, Label, FieldErrors } from 'formsnap';
	import { toast } from 'svelte-sonner';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import { z } from 'zod';

	let { data } = $props();

	const settingsSchema = z.object({
		name: z.string().trim().min(1, 'Guild name is required').max(100),
		color: z.string().regex(/^#[0-9a-f]{6}$/i, 'Must be a hex color, e.g. #3fb984')
	});

	const form = superForm(data.form, {
		validators: zodClient(settingsSchema),
		onUpdated: ({ form }) => {
			if (form.message) toast.success(form.message);
		}
	});
	const { form: formData, enhance, submitting } = form;

	const guildName = $derived(page.data.organization?.name ?? 'this guild');
	const inviteCode = $derived(page.data.organization?.inviteCode as string | undefined);

	let deleteDialogOpen = $state(false);
	let regenerating = $state(false);

	async function regenerateInvite() {
		regenerating = true;
		try {
			const res = await fetch('?/regenerateInvite', { method: 'POST', body: new FormData() });
			if (res.ok) {
				toast.success('Invite code regenerated');
				location.reload();
			} else {
				toast.error('Could not regenerate invite code');
			}
		} finally {
			regenerating = false;
		}
	}
</script>

<div class="px-7 py-[22px] max-w-[560px]">
	<div class="font-heading font-bold text-[23px] mb-1">Guild Settings</div>
	<div class="text-[13px] text-[var(--tx-2)] mb-[22px]">Manage settings for {guildName}.</div>

	<form method="post" action="?/update" use:enhance>
		<div
			class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden mb-[14px]"
		>
			<div
				class="px-[18px] py-3 border-b border-b-[var(--line)] text-[12px] font-bold tracking-[.05em] text-[var(--tx-3)]"
			>
				GENERAL
			</div>
			<div class="p-[18px]">
				<Field {form} name="name">
					<Control>
						{#snippet children({ props })}
							<Label class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]"
								>Guild name</Label
							>
							<input
								{...props}
								bind:value={$formData.name}
								class="w-full bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] text-[14px] text-[var(--tx)]"
							/>
						{/snippet}
					</Control>
					<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
				</Field>

				<div class="mt-[14px]">
					<Field {form} name="color">
						<Control>
							{#snippet children({ props })}
								<Label class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]"
									>Color</Label
								>
								<div class="flex items-center gap-2">
									<input
										{...props}
										type="color"
										bind:value={$formData.color}
										class="size-9 rounded-[8px] border border-[var(--line-2)] bg-[var(--card-2)] cursor-pointer"
									/>
									<span class="text-[13px] text-[var(--tx-2)] font-mono-jb">{$formData.color}</span>
								</div>
							{/snippet}
						</Control>
						<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
					</Field>
				</div>

				<label
					for="invite-code"
					class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px] mt-[14px]"
					>Invite code</label
				>
				<div class="flex gap-2">
					<input
						id="invite-code"
						value={inviteCode ?? ''}
						readonly
						class="flex-1 bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] font-mono-jb text-[13px] text-[var(--tx-2)]"
					/>
					<button
						type="button"
						onclick={regenerateInvite}
						disabled={regenerating}
						class="px-[14px] py-[10px] bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] text-[13px] text-[var(--tx-2)] cursor-pointer font-semibold disabled:opacity-40 disabled:cursor-not-allowed"
						>{regenerating ? 'Regenerating…' : 'Regenerate'}</button
					>
				</div>

				<div class="flex justify-end mt-[16px]">
					<button
						type="submit"
						disabled={$submitting}
						class="px-4 py-[10px] rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13.5px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
						>{$submitting ? 'Saving…' : 'Save changes'}</button
					>
				</div>
			</div>
		</div>
	</form>

	<div
		class="bg-[rgba(229,101,75,.08)] border border-[rgba(229,101,75,.2)] rounded-[13px] p-[18px]"
	>
		<div class="text-[14px] font-bold text-[#f0836b] mb-1">Danger zone</div>
		<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">
			These actions are irreversible. Please be certain before proceeding.
		</div>
		<button
			onclick={() => (deleteDialogOpen = true)}
			class="px-4 py-[9px] bg-[rgba(229,101,75,.14)] border border-[rgba(229,101,75,.3)] rounded-[8px] text-[#f0836b] text-[13px] font-bold cursor-pointer"
			>Delete guild…</button
		>
	</div>
</div>

<AlertDialog.Root bind:open={deleteDialogOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Delete {guildName}?</AlertDialog.Title>
			<AlertDialog.Description
				>This action is permanent and cannot be undone. All apps, hosts, members, and settings will
				be irreversibly removed.</AlertDialog.Description
			>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action>Delete guild</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
