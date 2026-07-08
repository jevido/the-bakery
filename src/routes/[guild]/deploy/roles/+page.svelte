<script lang="ts">
	import { getContext, onMount, onDestroy } from 'svelte';
	import { enhance } from '$app/forms';
	import { toast } from 'svelte-sonner';
	import {
		ROLES,
		PERM_GROUPS,
		roles as roleAc,
		staticRolePermissionIds,
		PERMISSION_RESOURCE,
		CUSTOM_ROLE_RESOURCES
	} from '$lib/auth/permissions';
	import * as Dialog from '$lib/components/ui/dialog';

	let { data } = $props();

	type CustomRole = (typeof data)['customRoles'][number];

	type RoleRow = {
		key: string;
		dbId: string | null;
		name: string;
		color: string;
		note: string;
		master: boolean;
		isCustom: boolean;
		count: number;
		perms: Set<string>;
	};

	function customRolePerms(cr: CustomRole): Set<string> {
		return new Set(CUSTOM_ROLE_RESOURCES.flatMap((res) => cr.permission[res] ?? []));
	}

	const roleRows = $derived<RoleRow[]>([
		...ROLES.map((r) => ({
			key: r.id,
			dbId: null,
			name: r.name,
			color: r.color,
			note: r.note,
			master: !!r.master,
			isCustom: false,
			count: data.memberCounts[r.id] ?? 0,
			perms: staticRolePermissionIds(r.id as keyof typeof roleAc)
		})),
		...data.customRoles.map((cr) => ({
			key: cr.role,
			dbId: cr.id,
			name: cr.role,
			color: (cr as { color?: string }).color ?? '#5b8def',
			note: (cr as { note?: string }).note ?? '',
			master: false,
			isCustom: true,
			count: data.memberCounts[cr.role] ?? 0,
			perms: customRolePerms(cr)
		}))
	]);

	function rolePerms(roleKey: string): Set<string> {
		return roleRows.find((r) => r.key === roleKey)?.perms ?? new Set();
	}

	const myRole = $derived(data.member?.role as string | undefined);
	const canManageRoles = $derived(myRole ? rolePerms(myRole).has('manage_roles') : false);

	const ALL_PERMISSION_IDS = PERM_GROUPS.flatMap((g) => g.perms.map((p) => p.id)).filter((id) => id !== 'administrator');

	let selectedRoleKey = $state(ROLES[0].id);
	const selectedRole = $derived(roleRows.find((r) => r.key === selectedRoleKey) ?? roleRows[0]);
	const hasAdministrator = $derived(ALL_PERMISSION_IDS.every((id) => selectedRole.perms.has(id)));

	let updateRoleForm = $state<HTMLFormElement>();
	let updateRoleIdInput = $state<HTMLInputElement>();
	let updateRolePermissionInput = $state<HTMLInputElement>();

	function togglePermission(permId: string) {
		if (!selectedRole.isCustom || !canManageRoles || !selectedRole.dbId) return;
		const newPerms = new Set(selectedRole.perms);
		if (newPerms.has(permId)) newPerms.delete(permId);
		else newPerms.add(permId);

		const permission: Record<string, string[]> = { guild: [], apps: [], hosts: [] };
		for (const id of newPerms) {
			const resource = PERMISSION_RESOURCE[id];
			if (resource) permission[resource].push(id);
		}

		if (updateRoleIdInput && updateRolePermissionInput && updateRoleForm) {
			updateRoleIdInput.value = selectedRole.dbId;
			updateRolePermissionInput.value = JSON.stringify(permission);
			updateRoleForm.requestSubmit();
		}
	}

	// New-role dialog, opened via the TopBar's "Create role" CTA.
	let createOpen = $state(false);
	let newRoleName = $state('');
	const PALETTE = ['#3fb984', '#5b8def', '#e05c4b', '#d97f2f', '#9b6ed4', '#2fc2c2', '#d4b44a', '#e06ba0'];
	let newRoleColor = $state(PALETTE[0]);

	function openCreateDialog() {
		newRoleName = '';
		newRoleColor = PALETTE[0];
		createOpen = true;
	}

	const cta = getContext<{ register(fn: () => void): void; unregister(): void } | undefined>('bakery:cta');
	onMount(() => cta?.register(openCreateDialog));
	onDestroy(() => cta?.unregister());
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
			{#each roleRows as r (r.key)}
				{@const active = r.key === selectedRole.key}
				<button
					onclick={() => (selectedRoleKey = r.key)}
					class="flex items-center gap-3 w-full px-4 py-[13px] border-b border-b-[var(--line)] cursor-pointer border-l-[3px] text-left {active ? 'bg-[rgba(63,185,132,.08)] border-l-[var(--grn)]' : 'bg-transparent border-l-transparent'}"
				>
					<div class="size-[10px] rounded-full shrink-0" style:background={r.color}></div>
					<div class="flex-1 min-w-0">
						<div class="text-[13.5px] {active ? 'font-bold text-[var(--tx)]' : 'font-semibold text-[var(--tx-2)]'}">{r.name}</div>
						<div class="text-[11.5px] text-[var(--tx-3)]">{r.count} member{r.count !== 1 ? 's' : ''}</div>
					</div>
					{#if r.master}
						<span class="text-[10px] font-bold px-[7px] py-[2px] rounded-[5px] bg-[rgba(224,168,62,.14)] text-[#e0a83e]">OWNER</span>
					{:else if r.isCustom}
						<span class="text-[10px] font-bold px-[7px] py-[2px] rounded-[5px] bg-white/[0.06] text-[var(--tx-3)]">CUSTOM</span>
					{/if}
				</button>
			{/each}
			{#if canManageRoles}
				<button
					onclick={openCreateDialog}
					class="flex items-center gap-2 w-full px-4 py-[13px] cursor-pointer text-left text-[13px] font-semibold text-[var(--grn-2)] hover:bg-white/5"
				>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M12 5v14M5 12h14"/></svg>
					New role
				</button>
			{/if}
		</div>

		<!-- Right: permissions panel -->
		<div>
			<!-- Role header -->
			<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[16px_18px] mb-3">
				<div class="flex items-center justify-between mb-2">
					<div class="flex items-center gap-[11px]">
						<div class="size-3 rounded-full" style:background={selectedRole.color}></div>
						<div class="text-[16px] font-bold">{selectedRole.name}</div>
					</div>
					{#if selectedRole.isCustom && canManageRoles}
						<form method="post" action="?/deleteRole" use:enhance>
							<input type="hidden" name="roleId" value={selectedRole.dbId} />
							<button
								type="submit"
								class="px-3 py-[6px] text-[12px] bg-[rgba(229,101,75,.1)] border border-[rgba(229,101,75,.25)] rounded-[7px] text-[#f0836b] cursor-pointer font-semibold"
							>Delete role</button>
						</form>
					{/if}
				</div>
				<div class="text-[13px] text-[var(--tx-2)] leading-[1.5]">
					{selectedRole.isCustom ? (selectedRole.note || 'Custom position with configurable permissions.') : selectedRole.note}
				</div>
			</div>

			<!-- Hidden form driving permission toggles for custom roles -->
			<form
				bind:this={updateRoleForm}
				method="post"
				action="?/updateRole"
				use:enhance={() => {
					return async ({ update }) => {
						await update();
					};
				}}
				class="hidden"
			>
				<input bind:this={updateRoleIdInput} type="hidden" name="roleId" />
				<input bind:this={updateRolePermissionInput} type="hidden" name="permission" />
			</form>

			<!-- Perm groups -->
			{#each PERM_GROUPS as g (g.name)}
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden mb-[10px]">
					<div class="px-[18px] py-[10px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.06em] text-[var(--tx-3)]">{g.name}</div>
					{#each g.perms as p (p.id)}
						{@const has = p.id === 'administrator' ? hasAdministrator : selectedRole.perms.has(p.id)}
						{@const editable = p.id !== 'administrator' && selectedRole.isCustom && canManageRoles}
						<div class="flex items-center gap-[13px] px-[18px] py-3 border-b border-b-[var(--line)]">
							<!-- Toggle -->
							<button
								type="button"
								disabled={!editable}
								onclick={() => togglePermission(p.id)}
								aria-label="Toggle {p.label}"
								aria-pressed={has}
								class="w-[34px] h-5 rounded-[10px] shrink-0 relative border-0 p-0 {has ? 'bg-[var(--grn)]' : 'bg-white/[0.08]'} {editable ? 'cursor-pointer' : 'cursor-not-allowed'}"
							>
								<div
									class="size-[14px] rounded-full bg-white absolute top-[3px] transition-[left] duration-150"
									style:left={has ? '17px' : '3px'}
								></div>
							</button>
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

<Dialog.Root bind:open={createOpen}>
	<Dialog.Content class="sm:max-w-[420px] bg-[var(--panel)] ring-[var(--line-2)] p-0 overflow-hidden">
		<div class="p-6">
			<h2 class="text-[17px] font-heading font-bold text-[var(--tx)] mb-5">New role</h2>

			<form
				method="post"
				action="?/createRole"
				use:enhance={() => {
					return async ({ result, update }) => {
						await update();
						if (result.type === 'success') {
							toast.success('Role created', { description: newRoleName });
							createOpen = false;
						}
					};
				}}
			>
				<label class="block mb-4">
					<span class="text-[12px] text-[var(--tx-2)] mb-1.5 block">Role name</span>
					<input
						name="roleName"
						bind:value={newRoleName}
						placeholder="e.g. Auditor"
						class="w-full h-9 px-3 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] placeholder:text-[var(--tx-3)] outline-none focus:border-[var(--line-2)] transition-colors"
					/>
				</label>

				<div class="mb-5">
					<span class="text-[12px] text-[var(--tx-2)] mb-2 block">Color</span>
					<div class="flex gap-2 flex-wrap">
						{#each PALETTE as swatch (swatch)}
							<button
								type="button"
								onclick={() => (newRoleColor = swatch)}
								class="size-6 rounded-full transition-transform hover:scale-110 relative flex-shrink-0"
								style:background={swatch}
							>
								{#if newRoleColor === swatch}
									<span class="absolute inset-[-3px] rounded-full border-2 border-white/50 pointer-events-none"></span>
								{/if}
							</button>
						{/each}
					</div>
					<input type="hidden" name="color" value={newRoleColor} />
				</div>

				<div class="flex justify-end">
					<button
						type="submit"
						disabled={!newRoleName.trim()}
						class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
					>Create role →</button>
				</div>
			</form>
		</div>
	</Dialog.Content>
</Dialog.Root>
