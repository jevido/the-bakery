<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onMount, onDestroy } from 'svelte';
	import { invalidateAll } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { superForm } from 'sveltekit-superforms';
	import { zodClient } from '$lib/forms/zod4-adapter';
	import { Field, Control, Label, FieldErrors } from 'formsnap';
	import { z } from 'zod';
	import { GUILD_RESOURCES } from '$lib/data/bakery';

	let { data } = $props();

	const guildId = $derived(page.params.guild ?? '');
	const mockHosts = $derived(GUILD_RESOURCES[guildId]?.hosts ?? []);
	const onlineCount = $derived(mockHosts.filter((h) => h.online).length);

	let filter = $state<'all' | 'online' | 'offline'>('all');

	const filtered = $derived(
		filter === 'all'      ? mockHosts
		: filter === 'online' ? mockHosts.filter((h) => h.online)
		: mockHosts.filter((h) => !h.online)
	);

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}

	function barColor(pct: number) {
		return pct > 70 ? '#e0a83e' : 'var(--grn)';
	}

	const addHostSchema = z.object({
		name: z.string().trim().min(1, 'Host name is required').max(100),
		location: z.string().trim().max(200).optional(),
		spec: z.string().trim().max(200).optional()
	});

	let panelOpen = $state(false);
	let formStep = $state<'form' | 'waiting' | 'done'>('form');
	let installCommand = $state('');
	let createdHostId = $state<string | null>(null);
	let createdHostName = $state('');

	const hostForm = superForm(data.form, {
		validators: zodClient(addHostSchema),
		resetForm: true,
		onResult: ({ result }) => {
			if (result.type === 'success' && result.data?.installCommand) {
				const created = result.data.host as { id: string; name: string };
				installCommand = result.data.installCommand as string;
				createdHostId = created.id;
				createdHostName = created.name;
				formStep = 'waiting';
			}
		}
	});
	const { form: formData, enhance, submitting } = hostForm;

	// Real hosts from the DB (task 03's load) — used only to watch the
	// just-created host's status here; the grid above still renders mock
	// data until task 05 switches it over.
	const realHosts = $derived(data.hosts ?? []);
	const createdHost = $derived(realHosts.find((h) => h.id === createdHostId));

	$effect(() => {
		if (formStep !== 'waiting') return;
		const interval = setInterval(() => invalidateAll(), 4000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		if (formStep === 'waiting' && createdHost?.status === 'online') {
			formStep = 'done';
			toast.success('Host connected', { description: `Bakery agent checked in on ${createdHostName}` });
		}
	});

	function openPanel() {
		panelOpen = true;
		formStep = 'form';
		installCommand = '';
		createdHostId = null;
		createdHostName = '';
	}

	function closePanel() {
		panelOpen = false;
	}

	const cta = getContext<{ register(fn: () => void): void; unregister(): void }>('bakery:cta');
	onMount(() => cta?.register(openPanel));
	onDestroy(() => cta?.unregister());

	async function copyInstallCommand() {
		await navigator.clipboard.writeText(installCommand);
		toast.success('Install command copied');
	}

	const panelInputCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-mono-jb';
</script>

<div class="px-7 py-[22px]">
	<div class="flex items-start justify-between mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Hosts</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{mockHosts.length} host{mockHosts.length !== 1 ? 's' : ''} · {onlineCount} online
			</div>
		</div>

		<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
			<button onclick={() => filter = 'all'} class={filterCls('all')}>All</button>
			<button onclick={() => filter = 'online'} class={filterCls('online')}>Online</button>
			<button onclick={() => filter = 'offline'} class={filterCls('offline')}>Offline</button>
		</div>
	</div>

	<div class="grid grid-cols-3 gap-5">
		{#if mockHosts.length === 0}
			<button
				onclick={openPanel}
				class="bg-[var(--card)] border border-dashed border-[var(--line-2)] rounded-[14px] p-6 flex flex-col items-center justify-center gap-3 cursor-pointer hover:bg-white/[0.02] hover:border-[var(--grn-line)] transition-colors group"
			>
				<div class="size-10 rounded-full bg-white/5 flex items-center justify-center group-hover:bg-[var(--grn-dim)] transition-colors">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="2" class="group-hover:stroke-[var(--grn-2)]"><path d="M12 5v14M5 12h14"/></svg>
				</div>
				<div class="text-center">
					<div class="text-[13.5px] font-semibold text-[var(--tx-2)] group-hover:text-[var(--tx)] mb-[3px]">Add your first host</div>
					<div class="text-[12px] text-[var(--tx-3)] leading-[1.5]">Install the Bakery agent on any<br>machine and connect it here</div>
				</div>
			</button>
		{:else}
			{#each filtered as host (host.name)}
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[14px] p-6 group cursor-default">
					<div class="flex items-start justify-between mb-3">
						<div class="flex items-center gap-[10px] min-w-0">
							<div class="size-[38px] rounded-[10px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0">
								{@render HostIcon(host.online)}
							</div>
							<div class="min-w-0">
								<div class="text-[14.5px] font-semibold text-[var(--tx)] truncate">{host.name}</div>
								<div class="text-[11px] text-[var(--tx-3)] truncate">{host.location}</div>
							</div>
						</div>
						<div class="flex items-center gap-[5px] shrink-0 ml-2">
							<div class="size-[7px] rounded-full" style:background={host.online ? 'var(--ok)' : 'var(--err)'}></div>
							<span class="text-[12px] text-[var(--tx-2)]">{host.online ? 'online' : 'offline'}</span>
						</div>
					</div>

					<div class="text-[11.5px] text-[var(--tx-3)] mb-[16px]">{host.spec}</div>

					<div class="flex flex-col gap-[9px] mb-[16px]">
						{#each [['CPU', host.cpu], ['MEM', host.mem], ['DISK', host.disk]] as [label, pct] (label)}
							<div class="flex items-center gap-[10px]">
								<span class="text-[10.5px] font-bold tracking-[.05em] text-[var(--tx-3)] w-[30px]">{label}</span>
								<div class="flex-1 h-[5px] rounded-[3px] bg-white/[0.07] overflow-hidden">
									<div class="h-full rounded-[3px]" style:width="{pct}%" style:background={barColor(pct as number)}></div>
								</div>
								<span class="font-mono-jb text-[11.5px] text-[var(--tx-3)] w-[28px] text-right">{pct}%</span>
							</div>
						{/each}
					</div>

					<div class="flex items-center justify-between pt-[13px] border-t border-t-[var(--line)]">
						<span class="text-[12px] text-[var(--tx-3)]">{host.apps} app{host.apps !== 1 ? 's' : ''}</span>
						<div class="flex items-center gap-[5px]">
							<span class="text-[11px] text-[var(--tx-3)]">Podman</span>
							<span class="font-mono-jb text-[11.5px] text-[var(--tx-2)]">v{host.podman}</span>
						</div>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div class="col-span-2 flex items-center justify-center rounded-[14px] border border-dashed border-[var(--line-2)] p-10 text-[var(--tx-3)] text-[13.5px]">
					No hosts match the current filter.
				</div>
			{/if}
		{/if}
	</div>
</div>

{#if panelOpen}
	<div
		role="button"
		tabindex="-1"
		onclick={closePanel}
		onkeydown={(e) => e.key === 'Escape' && closePanel()}
		class="fixed inset-0 bg-black/55 z-[200]"
	></div>

	<div class="fixed top-0 right-0 w-[480px] h-screen bg-[var(--panel)] border-l border-l-[var(--line)] z-[201] flex flex-col overflow-hidden">
		<div class="px-6 pt-5 pb-4 border-b border-b-[var(--line)] flex items-center gap-3">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)]">
				{formStep === 'form' ? 'Add host' : formStep === 'waiting' ? 'Waiting for check-in…' : 'Host connected'}
			</div>
			<div class="flex-1"></div>
			<button onclick={closePanel} aria-label="Close panel" class="text-[var(--tx-3)] cursor-pointer flex items-center">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto p-6">
			{#if formStep === 'form'}
				<form method="post" action="?/addHost" use:enhance class="flex flex-col gap-[18px]">
					<Field form={hostForm} name="name">
						<Control>
							{#snippet children({ props })}
								<Label class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">NAME</Label>
								<input {...props} bind:value={$formData.name} placeholder="oven-03" class={panelInputCls} />
							{/snippet}
						</Control>
						<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">A short name used across the guild.</div>
					</Field>

					<Field form={hostForm} name="location">
						<Control>
							{#snippet children({ props })}
								<Label class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">LOCATION</Label>
								<input {...props} bind:value={$formData.location} placeholder="Hetzner CPX21 · Falkenstein" class={panelInputCls} />
							{/snippet}
						</Control>
						<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Optional — where this machine lives.</div>
					</Field>

					<div class="bg-[var(--card-2)] border border-[var(--line)] rounded-[10px] px-4 py-[13px]">
						<div class="text-[12.5px] font-semibold text-[var(--tx)] mb-[4px]">You'll get a copy-paste install command</div>
						<div class="text-[12px] text-[var(--tx-3)] leading-[1.55]">
							Run it on the target machine as any user with systemd --user support. The Bakery agent installs as a rootless service, checks in over outbound HTTP, and reports metrics — no inbound access to your host required.
						</div>
					</div>

					<button
						type="submit"
						disabled={$submitting}
						class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold shadow-[0_2px_12px_var(--grn-dim)] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
					>
						{$submitting ? 'Creating host…' : 'Generate install command'}
					</button>
				</form>

			{:else if formStep === 'waiting'}
				<div class="flex flex-col gap-[16px]">
					<div class="text-[13px] text-[var(--tx-3)]">
						Run this on <span class="font-mono-jb text-[var(--tx)]">{createdHostName}</span>:
					</div>

					<div class="bg-[var(--card-2)] border border-[var(--line)] rounded-[9px] px-3 py-[12px] font-mono-jb text-[12.5px] text-[var(--tx)] break-all">
						{installCommand}
					</div>

					<button
						onclick={copyInstallCommand}
						class="w-full flex items-center justify-center gap-2 bg-[var(--card-2)] border border-[var(--line-2)] rounded-[9px] py-[10px] text-[13.5px] font-semibold text-[var(--tx)] cursor-pointer"
					>
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
						Copy install command
					</button>

					<div class="flex items-center gap-[10px] py-[10px] px-[14px] rounded-[9px] bg-[var(--card-2)] border border-[var(--line)]">
						<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="2" class="animate-[bk-spin_1s_linear_infinite] shrink-0"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
						<span class="text-[13px] text-[var(--tx-2)]">Waiting for the agent's first check-in…</span>
					</div>
				</div>

			{:else}
				<div class="text-center py-6">
					<div class="size-14 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mx-auto mb-4">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div class="font-heading text-[18px] font-bold text-[var(--tx)] mb-[6px]">Host connected!</div>
					<div class="text-[13px] text-[var(--tx-3)] mb-7">The Bakery agent is running and accepting deployments.</div>

					<button onclick={closePanel} class="w-full flex items-center justify-center bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">Done</button>
				</div>
			{/if}
		</div>
	</div>
{/if}

{#snippet HostIcon(online: boolean)}
	<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke={online ? 'var(--tx-2)' : 'var(--tx-3)'} stroke-width="1.6">
		<rect x="2" y="4" width="20" height="7" rx="1.5"/>
		<rect x="2" y="13" width="20" height="7" rx="1.5"/>
		<circle cx="6" cy="7.5" r="1"/>
		<circle cx="6" cy="16.5" r="1"/>
	</svg>
{/snippet}
