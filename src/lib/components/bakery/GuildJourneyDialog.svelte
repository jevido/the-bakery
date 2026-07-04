<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Dialog from '$lib/components/ui/dialog';
	import { guildStore } from '$lib/stores/guilds.svelte';

	let { open = $bindable(false) }: { open: boolean } = $props();

	const PALETTE = [
		'#3fb984', '#5b8def', '#e05c4b', '#d97f2f',
		'#9b6ed4', '#2fc2c2', '#d4b44a', '#e06ba0',
	];

	type Mode = 'choice' | 'create' | 'join';

	let mode = $state<Mode>('choice');
	let name = $state('');
	let color = $state(PALETTE[0]);
	let inviteCode = $state('');
	let foundGuild = $state<ReturnType<typeof guildStore.findByInvite>>(null);
	let findError = $state('');

	const letter = $derived(name.trim()[0]?.toUpperCase() ?? '?');
	const canCreate = $derived(name.trim().length > 0);

	function resetForm() {
		mode = 'choice';
		name = '';
		color = PALETTE[0];
		inviteCode = '';
		foundGuild = null;
		findError = '';
	}

	function findGuild() {
		const found = guildStore.findByInvite(inviteCode.trim());
		foundGuild = found;
		findError = found ? '' : 'No guild found with that invite code.';
	}

	function createGuild() {
		if (!canCreate) return;
		const base =
			name
				.toLowerCase()
				.replace(/\s+/g, '-')
				.replace(/[^a-z0-9-]/g, '') || 'guild';
		const id = guildStore.guilds[base] ? `${base}-${Date.now().toString(36)}` : base;
		const invite = `${id}-${Math.floor(Math.random() * 90 + 10)}`;
		guildStore.add({ id, name: name.trim(), letter, color, invite, apps: [], hosts: [] });
		goto(`/${id}/deploy/overview`);
		open = false;
	}

	function joinGuild() {
		if (!foundGuild) return;
		goto(`/${foundGuild.id}/deploy/overview`);
		open = false;
	}
</script>

<Dialog.Root bind:open onOpenChange={(isOpen) => { if (isOpen) resetForm(); }}>
	<Dialog.Content
		class="sm:max-w-[480px] bg-[var(--panel)] ring-[var(--line-2)] p-0 overflow-hidden"
		portalProps={{ to: '.bakery-shell' }}
	>
		{#if mode === 'choice'}
			<div class="p-6">
				<p class="text-[11px] text-[var(--tx-3)] uppercase tracking-wider font-semibold mb-1">
					Guild
				</p>
				<h2 class="text-[18px] font-heading font-bold text-[var(--tx)] mb-5">Add a guild</h2>
				<div class="grid grid-cols-2 gap-3">
					<button
						onclick={() => (mode = 'create')}
						class="text-left p-4 rounded-[12px] bg-[var(--card-2)] border border-[var(--line)] hover:border-[var(--line-2)] hover:bg-white/[0.04] transition-colors cursor-pointer"
					>
						<div
							class="size-8 rounded-[8px] bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center text-[var(--grn-2)] mb-3"
						>
							<svg
								width="15"
								height="15"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.2"
							>
								<path d="M12 5v14M5 12h14" />
							</svg>
						</div>
						<p class="text-[13px] font-semibold text-[var(--tx)] mb-1">Create a guild</p>
						<p class="text-[12px] text-[var(--tx-3)] leading-relaxed">
							Start fresh with your own workspace
						</p>
					</button>

					<button
						onclick={() => (mode = 'join')}
						class="text-left p-4 rounded-[12px] bg-[var(--card-2)] border border-[var(--line)] hover:border-[var(--line-2)] hover:bg-white/[0.04] transition-colors cursor-pointer"
					>
						<div
							class="size-8 rounded-[8px] bg-white/[0.05] border border-[var(--line)] flex items-center justify-center text-[var(--tx-2)] mb-3"
						>
							<svg
								width="15"
								height="15"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
							>
								<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
								<circle cx="9" cy="7" r="4" />
								<path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" />
							</svg>
						</div>
						<p class="text-[13px] font-semibold text-[var(--tx)] mb-1">Join a guild</p>
						<p class="text-[12px] text-[var(--tx-3)] leading-relaxed">
							Enter an invite code to join a workspace
						</p>
					</button>
				</div>
			</div>
		{:else if mode === 'create'}
			<div class="p-6">
				<button
					onclick={() => (mode = 'choice')}
					class="flex items-center gap-1.5 text-[12px] text-[var(--tx-3)] hover:text-[var(--tx-2)] transition-colors mb-5"
				>
					<svg
						width="13"
						height="13"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2.2"
					>
						<path d="M19 12H5M12 5l-7 7 7 7" />
					</svg>
					Back
				</button>

				<h2 class="text-[17px] font-heading font-bold text-[var(--tx)] mb-5">Create a guild</h2>

				<label class="block mb-4">
					<span class="text-[12px] text-[var(--tx-2)] mb-1.5 block">Guild name</span>
					<input
						bind:value={name}
						placeholder="e.g. Rye & Co"
						class="w-full h-9 px-3 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] placeholder:text-[var(--tx-3)] outline-none focus:border-[var(--line-2)] transition-colors"
					/>
				</label>

				<div class="mb-5">
					<span class="text-[12px] text-[var(--tx-2)] mb-2 block">Color</span>
					<div class="flex gap-2 flex-wrap">
						{#each PALETTE as swatch (swatch)}
							<button
								onclick={() => (color = swatch)}
								class="size-6 rounded-full transition-transform hover:scale-110 relative flex-shrink-0"
								style:background={swatch}
								title={swatch}
							>
								{#if color === swatch}
									<span
										class="absolute inset-[-3px] rounded-full border-2 border-white/50 pointer-events-none"
									></span>
								{/if}
							</button>
						{/each}
					</div>
				</div>

				<div
					class="flex items-center gap-3 p-3 rounded-[10px] bg-[var(--card-2)] border border-[var(--line)] mb-5"
				>
					<div
						class="size-10 rounded-[12px] flex items-center justify-center font-heading font-bold text-[17px] shrink-0"
						style:background="{color}22"
						style:color
						style:border="1px solid {color}44"
					>{letter}</div>
					<div class="min-w-0">
						<p class="text-[13px] font-semibold text-[var(--tx)] truncate">
							{name.trim() || 'Your guild name'}
						</p>
						<p class="text-[11px] text-[var(--tx-3)]">New guild · 0 apps</p>
					</div>
				</div>

				<div class="flex justify-end">
					<button
						onclick={createGuild}
						disabled={!canCreate}
						class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
					>Create guild →</button>
				</div>
			</div>
		{:else}
			<div class="p-6">
				<button
					onclick={() => (mode = 'choice')}
					class="flex items-center gap-1.5 text-[12px] text-[var(--tx-3)] hover:text-[var(--tx-2)] transition-colors mb-5"
				>
					<svg
						width="13"
						height="13"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2.2"
					>
						<path d="M19 12H5M12 5l-7 7 7 7" />
					</svg>
					Back
				</button>

				<h2 class="text-[17px] font-heading font-bold text-[var(--tx)] mb-5">Join a guild</h2>

				<div class="mb-4">
					<span class="text-[12px] text-[var(--tx-2)] mb-1.5 block">Invite code</span>
					<div class="flex gap-2">
						<input
							bind:value={inviteCode}
							placeholder="e.g. warm-rye-42"
							onkeydown={(e) => e.key === 'Enter' && findGuild()}
							class="flex-1 h-9 px-3 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] placeholder:text-[var(--tx-3)] outline-none focus:border-[var(--line-2)] transition-colors min-w-0"
						/>
						<button
							onclick={findGuild}
							class="px-3 h-9 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx-2)] text-[13px] hover:border-[var(--line-2)] hover:text-[var(--tx)] transition-colors shrink-0"
						>Find</button>
					</div>
				</div>

				{#if foundGuild}
					<div
						class="flex items-center gap-3 p-3 rounded-[10px] bg-[var(--grn-dim)] border border-[var(--grn-line)] mb-5"
					>
						<div
							class="size-10 rounded-[12px] flex items-center justify-center font-heading font-bold text-[17px] shrink-0"
							style:background="{foundGuild.color}22"
							style:color={foundGuild.color}
							style:border="1px solid {foundGuild.color}44"
						>{foundGuild.letter}</div>
						<div class="min-w-0">
							<p class="text-[13px] font-semibold text-[var(--tx)] truncate">{foundGuild.name}</p>
							<p class="text-[11px] text-[var(--tx-3)]">
								{foundGuild.id} · {foundGuild.apps.length} apps
							</p>
						</div>
						<svg
							class="ml-auto text-[var(--ok)] shrink-0"
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2.5"
						>
							<path d="M20 6L9 17l-5-5" />
						</svg>
					</div>
				{:else if findError}
					<div class="p-3 rounded-[10px] bg-red-500/10 border border-red-500/20 mb-5">
						<p class="text-[12px] text-red-400">{findError}</p>
					</div>
				{/if}

				<div class="flex justify-end">
					<button
						onclick={joinGuild}
						disabled={!foundGuild}
						class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
					>Join guild →</button>
				</div>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
