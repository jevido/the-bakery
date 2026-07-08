<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import * as Dialog from '$lib/components/ui/dialog';

	let { open = $bindable(false) }: { open: boolean } = $props();

	const PALETTE = [
		'#3fb984', '#5b8def', '#e05c4b', '#d97f2f',
		'#9b6ed4', '#2fc2c2', '#d4b44a', '#e06ba0',
	];

	type Mode = 'choice' | 'create' | 'join';
	type FoundGuild = { name: string; slug: string; color: string };

	let mode = $state<Mode>('choice');
	let name = $state('');
	let color = $state(PALETTE[0]);
	let creating = $state(false);
	let createError = $state('');

	let inviteCode = $state('');
	let foundGuild = $state<FoundGuild | null>(null);
	let finding = $state(false);
	let joining = $state(false);
	let findError = $state('');

	const letter = $derived(name.trim()[0]?.toUpperCase() ?? '?');
	const canCreate = $derived(name.trim().length > 0 && !creating);

	function resetForm() {
		mode = 'choice';
		name = '';
		color = PALETTE[0];
		creating = false;
		createError = '';
		inviteCode = '';
		foundGuild = null;
		finding = false;
		joining = false;
		findError = '';
	}

	async function findGuild() {
		const code = inviteCode.trim();
		if (!code) return;
		finding = true;
		findError = '';
		foundGuild = null;
		try {
			const res = await fetch(`/api/guilds/join?code=${encodeURIComponent(code)}`);
			if (!res.ok) {
				const body = await res.json().catch(() => null);
				findError = body?.message ?? 'No guild found with that invite code.';
				return;
			}
			foundGuild = await res.json();
		} finally {
			finding = false;
		}
	}

	async function createGuild() {
		if (!canCreate) return;
		creating = true;
		createError = '';
		try {
			const res = await fetch('/api/guilds', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name: name.trim(), color })
			});
			if (!res.ok) {
				const body = await res.json().catch(() => null);
				createError = body?.message ?? 'Could not create guild.';
				return;
			}
			const { slug } = await res.json();
			await invalidateAll();
			await goto(`/${slug}/deploy/overview`);
			open = false;
		} finally {
			creating = false;
		}
	}

	async function joinGuild() {
		if (!foundGuild) return;
		joining = true;
		findError = '';
		try {
			const res = await fetch('/api/guilds/join', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ inviteCode: inviteCode.trim() })
			});
			if (!res.ok) {
				const body = await res.json().catch(() => null);
				findError = body?.message ?? 'Could not join guild.';
				return;
			}
			const { slug } = await res.json();
			await invalidateAll();
			await goto(`/${slug}/deploy/overview`);
			open = false;
		} finally {
			joining = false;
		}
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
						<p class="text-[11px] text-[var(--tx-3)]">New guild</p>
					</div>
				</div>

				{#if createError}
					<div class="p-3 rounded-[10px] bg-red-500/10 border border-red-500/20 mb-5">
						<p class="text-[12px] text-red-400">{createError}</p>
					</div>
				{/if}

				<div class="flex justify-end">
					<button
						onclick={createGuild}
						disabled={!canCreate}
						class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
					>{creating ? 'Creating…' : 'Create guild →'}</button>
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
							disabled={finding}
							class="px-3 h-9 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx-2)] text-[13px] hover:border-[var(--line-2)] hover:text-[var(--tx)] transition-colors shrink-0 disabled:opacity-40"
						>{finding ? '…' : 'Find'}</button>
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
						>{foundGuild.name.trim()[0]?.toUpperCase() ?? '?'}</div>
						<div class="min-w-0">
							<p class="text-[13px] font-semibold text-[var(--tx)] truncate">{foundGuild.name}</p>
							<p class="text-[11px] text-[var(--tx-3)]">{foundGuild.slug}</p>
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
						disabled={!foundGuild || joining}
						class="px-4 py-2 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
					>{joining ? 'Joining…' : 'Join guild →'}</button>
				</div>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
