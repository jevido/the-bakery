<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onMount, onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { GUILDS } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
	const hosts = $derived(guild?.hosts ?? []);
	const onlineCount = $derived(hosts.filter((h) => h.online).length);

	let filter = $state<'all' | 'online' | 'offline'>('all');

	const filtered = $derived(
		filter === 'all'      ? hosts
		: filter === 'online' ? hosts.filter((h) => h.online)
		: hosts.filter((h) => !h.online)
	);

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}

	function barColor(pct: number) {
		return pct > 70 ? '#e0a83e' : 'var(--grn)';
	}

	const cp = { cpu: 18, mem: 45, disk: 52, bakery: '0.9.2', podman: '5.1', uptime: '14d 3h', location: 'Hetzner CPX11 · Falkenstein', spec: 'AMD EPYC · 2 vCPU · 2 GB' };

	let panelOpen = $state(false);
	let formStep = $state<'form' | 'installing' | 'done'>('form');
	let hostAddr = $state('');
	let hostLabel = $state('');
	let hostPort = $state('22');
	let hostKey = $state('generate');
	let currentStep = $state(0);

	const installSteps = [
		'Connecting via SSH',
		'Downloading Bakery agent',
		'Starting agent service',
		'Verifying connectivity',
	];

	function openPanel() {
		panelOpen = true;
		formStep = 'form';
		hostAddr = '';
		hostLabel = '';
		hostPort = '22';
		hostKey = 'generate';
		currentStep = 0;
	}

	function closePanel() { panelOpen = false; }

	const cta = getContext<{ register(fn: () => void): void; unregister(): void }>('bakery:cta');
	onMount(() => cta?.register(openPanel));
	onDestroy(() => cta?.unregister());

	const canConnect = $derived(hostAddr.trim().length > 0 && hostLabel.trim().length > 0);

	async function addHost() {
		formStep = 'installing';
		for (let i = 0; i < installSteps.length; i++) {
			currentStep = i;
			await new Promise((r) => setTimeout(r, 700 + Math.random() * 500));
		}
		currentStep = installSteps.length;
		await new Promise((r) => setTimeout(r, 300));
		formStep = 'done';
		toast.success('Host registered', { description: `Bakery agent running on ${hostLabel}` });
	}

	const panelInputCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-mono-jb';
	const panelSelectCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-bakery appearance-none cursor-pointer';
</script>

<div class="px-7 py-[22px]">
	<div class="flex items-start justify-between mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Hosts</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{hosts.length} host{hosts.length !== 1 ? 's' : ''} · {onlineCount} online
			</div>
		</div>

		<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
			<button onclick={() => filter = 'all'} class={filterCls('all')}>All</button>
			<button onclick={() => filter = 'online'} class={filterCls('online')}>Online</button>
			<button onclick={() => filter = 'offline'} class={filterCls('offline')}>Offline</button>
		</div>
	</div>

	<div class="grid grid-cols-3 gap-5">
		<!-- Control plane — always visible -->
		<div class="bg-[var(--card)] border border-[var(--grn-line)] rounded-[14px] p-6">
			<div class="flex items-start justify-between mb-3">
				<div class="flex items-center gap-[10px]">
					<div class="size-[38px] rounded-[10px] bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center shrink-0">
						<span class="font-heading font-bold text-[17px] text-[var(--grn-2)]">B</span>
					</div>
					<div>
						<div class="flex items-center gap-[7px]">
							<span class="text-[14.5px] font-semibold text-[var(--tx)]">Control Plane</span>
							<span class="text-[10px] font-bold tracking-[.06em] px-[6px] py-[2px] rounded-[4px] bg-[var(--grn-dim)] text-[var(--grn-2)]">BAKERY</span>
						</div>
						<div class="text-[11px] text-[var(--tx-3)] mt-[1px]">{cp.location}</div>
					</div>
				</div>
				<div class="flex items-center gap-[5px] shrink-0">
					<div class="size-[7px] rounded-full bg-[var(--ok)]"></div>
					<span class="text-[12px] text-[var(--tx-2)]">online</span>
				</div>
			</div>

			<div class="text-[11.5px] text-[var(--tx-3)] mb-[16px]">{cp.spec}</div>

			<div class="flex flex-col gap-[9px] mb-[16px]">
				{#each [['CPU', cp.cpu], ['MEM', cp.mem], ['DISK', cp.disk]] as [label, pct] (label)}
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
				<div class="flex items-center gap-[6px]">
					<span class="text-[11px] font-semibold text-[var(--tx-3)]">Bakery</span>
					<span class="font-mono-jb text-[11.5px] text-[var(--grn-2)]">v{cp.bakery}</span>
				</div>
				<span class="text-[11px] text-[var(--tx-3)]">up {cp.uptime}</span>
			</div>
		</div>

		<!-- Host cards -->
		{#if hosts.length === 0}
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
				{formStep === 'form' ? 'Add host' : formStep === 'installing' ? 'Installing…' : 'Host connected'}
			</div>
			<div class="flex-1"></div>
			{#if formStep !== 'installing'}
				<button onclick={closePanel} aria-label="Close panel" class="text-[var(--tx-3)] cursor-pointer flex items-center">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
				</button>
			{/if}
		</div>

		<div class="flex-1 overflow-y-auto p-6">
			{#if formStep === 'form'}
				<div class="flex flex-col gap-[18px]">
					<div>
						<label for="host-addr" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">HOST ADDRESS</label>
						<input id="host-addr" bind:value={hostAddr} placeholder="192.168.1.10 or server.example.com" class={panelInputCls} />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">IP address or hostname reachable over SSH.</div>
					</div>

					<div>
						<label for="host-label" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">LABEL</label>
						<input id="host-label" bind:value={hostLabel} placeholder="oven-03" class={panelInputCls} />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">A short name used across the guild.</div>
					</div>

					<div>
						<label for="host-port" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">SSH PORT</label>
						<input id="host-port" bind:value={hostPort} placeholder="22" class={panelInputCls} />
					</div>

					<div>
						<label for="host-key" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">SSH KEY</label>
						<select id="host-key" bind:value={hostKey} class={panelSelectCls}>
							<option value="generate">Generate new key for this host</option>
							<option value="guild">Use guild shared key</option>
						</select>
						{#if hostKey === 'generate'}
							<div class="mt-2 bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] px-3 py-[10px] text-[12px] text-[var(--tx-3)] leading-[1.5]">
								A new Ed25519 key pair will be generated. Add the public key to <span class="font-mono-jb">~/.ssh/authorized_keys</span> on the host before connecting.
							</div>
						{/if}
					</div>

					<div class="bg-[var(--card-2)] border border-[var(--line)] rounded-[10px] px-4 py-[13px]">
						<div class="text-[12.5px] font-semibold text-[var(--tx)] mb-[4px]">Bakery agent will be installed</div>
						<div class="text-[12px] text-[var(--tx-3)] leading-[1.55]">
							Bakery connects over SSH and installs a lightweight agent daemon on the host. The agent manages Podman, reports metrics, and executes deployments on behalf of the guild.
						</div>
					</div>

					<button
						onclick={addHost}
						disabled={!canConnect}
						class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold shadow-[0_2px_12px_var(--grn-dim)] {!canConnect ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
					>
						Connect & install agent
					</button>
				</div>

			{:else if formStep === 'installing'}
				<div class="flex flex-col gap-[10px] pt-2">
					<div class="text-[13px] text-[var(--tx-3)] mb-4">
						Installing Bakery agent on <span class="font-mono-jb text-[var(--tx)]">{hostAddr}</span>
					</div>

					{#each installSteps as step, i (step)}
						<div class="flex items-center gap-[12px] py-[10px] px-[14px] rounded-[9px] bg-[var(--card-2)] border border-[var(--line)]">
							<div class="size-[20px] flex items-center justify-center shrink-0">
								{#if currentStep > i}
									<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--ok)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
								{:else if currentStep === i}
									<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="2" class="animate-[bk-spin_1s_linear_infinite]"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
								{:else}
									<div class="size-[6px] rounded-full bg-white/20"></div>
								{/if}
							</div>
							<span class="text-[13.5px] {currentStep >= i ? 'text-[var(--tx)]' : 'text-[var(--tx-3)]'} flex-1">{step}</span>
							{#if currentStep > i}
								<span class="text-[11px] text-[var(--ok)] font-semibold">done</span>
							{:else if currentStep === i}
								<span class="text-[11px] text-[var(--grn-2)]">running</span>
							{/if}
						</div>
					{/each}
				</div>

			{:else}
				<div class="text-center py-6">
					<div class="size-14 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mx-auto mb-4">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div class="font-heading text-[18px] font-bold text-[var(--tx)] mb-[6px]">Host connected!</div>
					<div class="text-[13px] text-[var(--tx-3)] mb-7">The Bakery agent is running and accepting deployments.</div>

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[10px] text-left overflow-hidden mb-6">
						{#each [
							{ label: 'Label',         value: hostLabel,                                        mono: true  },
							{ label: 'Address',        value: hostAddr,                                         mono: true  },
							{ label: 'SSH port',       value: hostPort,                                         mono: true  },
							{ label: 'Bakery agent',   value: '0.9.2',                                          mono: true  },
							{ label: 'SSH key',        value: hostKey === 'generate' ? 'New key generated' : 'Guild shared key', mono: false },
							{ label: 'Status',         value: 'online',                                         mono: false },
						] as row (row.label)}
							<div class="flex items-center justify-between px-4 py-[10px] border-b border-b-[var(--line)] last:border-b-0">
								<span class="text-[12px] text-[var(--tx-3)]">{row.label}</span>
								<span class="text-[12.5px] text-[var(--tx)] {row.mono ? 'font-mono-jb' : ''}">{row.value}</span>
							</div>
						{/each}
					</div>

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
