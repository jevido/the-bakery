<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onMount, onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { NETWORKS, GUILDS, networkDriverMeta, type NetworkDriver } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const networks = $derived(NETWORKS[guildId] ?? []);
	const hosts = $derived(GUILDS[guildId]?.hosts ?? []);
	const totalApps = $derived(
		networks.reduce((n, net) => n + net.connectedApps.length, 0)
	);

	let filter = $state<'all' | 'bridge' | 'internal'>('all');

	const filtered = $derived(
		filter === 'all'      ? networks
		: filter === 'bridge' ? networks.filter((n) => n.driver === 'bridge')
		: networks.filter((n) => n.internal)
	);

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}

	let panelOpen = $state(false);
	let formStep = $state<'form' | 'done'>('form');
	let netName = $state('');
	let netHost = $state('');
	let netDriver = $state<NetworkDriver>('bridge');
	let netSubnet = $state('');
	let netInternal = $state(false);
	let creating = $state(false);

	function openPanel() {
		panelOpen = true;
		formStep = 'form';
		netName = '';
		netHost = hosts[0]?.name ?? '';
		netDriver = 'bridge';
		netSubnet = '';
		netInternal = false;
		creating = false;
	}

	function closePanel() { panelOpen = false; }

	const cta = getContext<{ register(fn: () => void): void; unregister(): void }>('bakery:cta');
	onMount(() => cta?.register(openPanel));
	onDestroy(() => cta?.unregister());

	const canCreate = $derived(netName.trim().length > 0 && netHost.length > 0);

	async function createNetwork() {
		creating = true;
		await new Promise((r) => setTimeout(r, 800));
		creating = false;
		formStep = 'done';
		toast.success('Network created', { description: `${netName} is ready on ${netHost}` });
	}

	const resolvedSubnet = $derived(netSubnet.trim() || '10.88.0.0/16');

	const panelInputCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-mono-jb';
	const panelSelectCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-bakery appearance-none cursor-pointer';
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-start gap-[14px] mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Networks</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{#if networks.length > 0}
					{networks.length} network{networks.length !== 1 ? 's' : ''} · {totalApps} container{totalApps !== 1 ? 's' : ''} attached
				{:else}
					Podman networks for container-to-container connectivity.
				{/if}
			</div>
		</div>
	</div>

	{#if networks.length === 0}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[40px_32px]">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)] mb-[6px]">No networks defined</div>
			<div class="text-[13px] text-[var(--tx-3)] mb-8">Create a Podman network to isolate or connect your containers across a host.</div>

			<div class="grid grid-cols-3 gap-4 mb-8">
				{#each [
					{ n: '1', title: 'Choose a host', desc: 'Select which machine the network lives on. Networks are scoped to a single host.' },
					{ n: '2', title: 'Configure the network', desc: 'Give it a name, pick a driver (bridge is the default), and optionally set a CIDR range.' },
					{ n: '3', title: 'Attach apps', desc: 'Connect deployments at deploy time or add them from the app detail page.' },
				] as guideStep (guideStep.n)}
					<div class="bg-[var(--card-2)] border border-[var(--line)] rounded-[10px] p-[18px_16px]">
						<div class="size-7 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mb-3">
							<span class="font-mono-jb text-[12px] font-bold text-[var(--grn-2)]">{guideStep.n}</span>
						</div>
						<div class="text-[13.5px] font-semibold text-[var(--tx)] mb-[5px]">{guideStep.title}</div>
						<div class="text-[12px] text-[var(--tx-3)] leading-[1.55]">{guideStep.desc}</div>
					</div>
				{/each}
			</div>

			<div class="flex flex-wrap gap-2 mb-7">
				{#each [
					{ label: 'bridge',  color: '#7aa6f5', desc: 'Default — software bridge on host' },
					{ label: 'macvlan', color: '#c09bf0', desc: 'Physical device appearance' },
					{ label: 'ipvlan',  color: '#efc060', desc: 'Shares parent MAC address' },
				] as d (d.label)}
					<div class="flex items-center gap-2 bg-[var(--card-2)] border border-[var(--line)] rounded-[7px] px-3 py-[6px]">
						<div class="size-2 rounded-full" style:background={d.color}></div>
						<span class="font-mono-jb text-[12px] text-[var(--tx-2)]">{d.label}</span>
						<span class="text-[12px] text-[var(--tx-3)]">{d.desc}</span>
					</div>
				{/each}
			</div>

			<button onclick={openPanel} class="inline-flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[9px] px-[18px] py-[9px] text-[13.5px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
				Create your first network
			</button>
		</div>

	{:else}
		<div class="flex items-center gap-[14px] mb-[14px]">
			<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
				<button onclick={() => filter = 'all'} class={filterCls('all')}>All</button>
				<button onclick={() => filter = 'bridge'} class={filterCls('bridge')}>Bridge</button>
				<button onclick={() => filter = 'internal'} class={filterCls('internal')}>Internal</button>
			</div>
		</div>

		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div class="grid grid-cols-[1.6fr_100px_1.3fr_1.4fr_120px_90px_80px] gap-3 px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
				<div>NETWORK</div>
				<div>DRIVER</div>
				<div>SUBNET</div>
				<div>APPS</div>
				<div>HOST</div>
				<div>CREATED</div>
				<div></div>
			</div>

			{#each filtered as net (net.id)}
				{@const dm = networkDriverMeta(net.driver)}
				<div class="group px-[18px] py-[13px] grid grid-cols-[1.6fr_100px_1.3fr_1.4fr_120px_90px_80px] gap-3 items-center border-b border-b-[var(--line)] last:border-b-0 hover:bg-white/[0.02]">
					<div class="flex items-center gap-[11px] min-w-0">
						<div class="size-[34px] rounded-[9px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0">
							{@render NetworkIcon()}
						</div>
						<div class="min-w-0">
							<div class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{net.name}</div>
							<div class="text-[11px] text-[var(--tx-3)]">{net.internal ? 'internal · isolated' : 'external · routable'}</div>
						</div>
					</div>

					<div>
						<span class="text-[12px] font-semibold rounded-[5px] px-2 py-[3px]" style:color={dm.color} style:background={dm.bg}>{dm.label}</span>
					</div>

					<div>
						<div class="font-mono-jb text-[12px] text-[var(--tx-2)]">{net.subnet}</div>
						<div class="font-mono-jb text-[11px] text-[var(--tx-3)]">gw {net.gateway}</div>
					</div>

					<div class="flex flex-wrap gap-1 min-w-0">
						{#each net.connectedApps.slice(0, 3) as appId (appId)}
							<span class="text-[11px] text-[var(--tx-2)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px] whitespace-nowrap">{appId}</span>
						{/each}
						{#if net.connectedApps.length > 3}
							<span class="text-[11px] text-[var(--tx-3)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px]">+{net.connectedApps.length - 3} more</span>
						{/if}
					</div>

					<div class="text-[12.5px] text-[var(--tx-2)]">{net.host}</div>
					<div class="text-[12px] text-[var(--tx-3)]">{net.created}</div>

					<div class="text-right">
						<button class="opacity-0 transition-opacity duration-150 group-hover:opacity-100 text-[12.5px] text-[var(--tx-3)] cursor-pointer px-2 py-1 rounded-[6px]">Inspect →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div class="p-10 text-center text-[var(--tx-3)] text-[13.5px]">No networks match the current filter.</div>
			{/if}
		</div>
	{/if}
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
				{formStep === 'form' ? 'Add network' : 'Network created'}
			</div>
			<div class="flex-1"></div>
			<button onclick={closePanel} aria-label="Close panel" class="text-[var(--tx-3)] cursor-pointer flex items-center">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto p-6">
			{#if formStep === 'form'}
				<div class="flex flex-col gap-[18px]">
					<div>
						<label for="net-name" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">NETWORK NAME</label>
						<input id="net-name" bind:value={netName} placeholder="bakery-net" class={panelInputCls} />
					</div>

					<div>
						<label for="net-host" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">HOST</label>
						<select id="net-host" bind:value={netHost} class={panelSelectCls}>
							{#each hosts as h (h.name)}
								<option value={h.name}>{h.name} — {h.location}</option>
							{/each}
						</select>
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Networks are scoped to a single host.</div>
					</div>

					<div>
						<label for="net-driver" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">DRIVER</label>
						<select id="net-driver" bind:value={netDriver} class={panelSelectCls}>
							<option value="bridge">bridge — default software bridge</option>
							<option value="macvlan">macvlan — appear as physical device</option>
							<option value="ipvlan">ipvlan — shares parent MAC</option>
						</select>
						{#if netDriver !== 'bridge'}
							<div class="mt-2 bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] px-3 py-[10px] text-[12px] text-[var(--tx-3)] leading-[1.5]">
								{networkDriverMeta(netDriver).desc}
							</div>
						{/if}
					</div>

					<div>
						<label for="net-subnet" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">SUBNET <span class="font-normal text-[var(--tx-3)]">— optional</span></label>
						<input id="net-subnet" bind:value={netSubnet} placeholder="10.88.0.0/16" class={panelInputCls} />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Leave blank to let Podman assign automatically.</div>
					</div>

					<div>
						<div class="text-[12px] font-semibold text-[var(--tx-2)] mb-2 tracking-[.03em]">CONNECTIVITY</div>
						<button
							onclick={() => netInternal = !netInternal}
							class="flex items-center gap-3 bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-[14px] py-[10px] w-full cursor-pointer text-left"
						>
							<div class="w-9 h-5 rounded-[10px] relative shrink-0 transition-[background] duration-200 {netInternal ? 'bg-[var(--grn)]' : 'bg-white/10'}">
								<div class="absolute top-[3px] size-[14px] rounded-full bg-white transition-[left] duration-200" style:left={netInternal ? '19px' : '3px'}></div>
							</div>
							<div>
								<div class="text-[13px] font-semibold text-[var(--tx)]">{netInternal ? 'Internal (isolated)' : 'External'}</div>
								<div class="text-[12px] text-[var(--tx-3)]">{netInternal ? 'Containers cannot reach the internet.' : 'Containers can reach external hosts.'}</div>
							</div>
						</button>
					</div>

					<button
						onclick={createNetwork}
						disabled={!canCreate || creating}
						class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold shadow-[0_2px_12px_var(--grn-dim)] {(!canCreate || creating) ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
					>
						{#if creating}
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="animate-[bk-spin_1s_linear_infinite]"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
							Creating…
						{:else}
							Create network
						{/if}
					</button>
				</div>

			{:else}
				<div class="text-center py-6">
					<div class="size-14 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mx-auto mb-4">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div class="font-heading text-[18px] font-bold text-[var(--tx)] mb-[6px]">Network created!</div>
					<div class="text-[13px] text-[var(--tx-3)] mb-7">Your network is ready. Attach containers by selecting it at deploy time.</div>

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[10px] text-left overflow-hidden mb-6">
						{#each [
							{ label: 'Name',     value: netName || 'my-network', mono: true },
							{ label: 'Driver',   value: netDriver, mono: true },
							{ label: 'Subnet',   value: resolvedSubnet, mono: true },
							{ label: 'Host',     value: netHost, mono: false },
							{ label: 'Internal', value: netInternal ? 'Yes' : 'No', mono: false },
						] as row (row.label)}
							<div class="flex items-center justify-between px-4 py-[10px] border-b border-b-[var(--line)]">
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

{#snippet NetworkIcon()}
	<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6">
		<circle cx="12" cy="5" r="2"/>
		<circle cx="5" cy="19" r="2"/>
		<circle cx="19" cy="19" r="2"/>
		<path d="M12 7v4M12 11l-5 6M12 11l5 6"/>
	</svg>
{/snippet}
