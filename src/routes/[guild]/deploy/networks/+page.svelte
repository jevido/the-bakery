<script lang="ts">
	import { page } from '$app/state';
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

	function filterStyle(f: typeof filter) {
		return f === filter
			? 'padding:5px 11px;font-size:12.5px;color:var(--tx);border-radius:6px;cursor:pointer;background:var(--card-2);'
			: 'padding:5px 11px;font-size:12.5px;color:var(--tx-2);border-radius:6px;cursor:pointer;';
	}

	// ── Add network panel ─────────────────────────────────────────────
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

	function closePanel() {
		panelOpen = false;
	}

	const canCreate = $derived(netName.trim().length > 0 && netHost.length > 0);

	async function createNetwork() {
		creating = true;
		await new Promise((r) => setTimeout(r, 800));
		creating = false;
		formStep = 'done';
	}

	const resolvedSubnet = $derived(netSubnet.trim() || '10.88.0.0/16');
</script>

<div style="padding: 22px 28px;">
	<!-- Header -->
	<div style="display:flex;align-items:flex-start;gap:14px;margin-bottom:22px;">
		<div>
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;letter-spacing:-.01em;margin-bottom:3px;">Networks</div>
			<div style="font-size:13px;color:var(--tx-2);">
				{#if networks.length > 0}
					{networks.length} network{networks.length !== 1 ? 's' : ''} · {totalApps} container{totalApps !== 1 ? 's' : ''} attached
				{:else}
					Podman networks for container-to-container connectivity.
				{/if}
			</div>
		</div>
		<div style="flex:1"></div>
		<button onclick={openPanel} style="display:flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border-radius:9px;padding:8px 14px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
			Add network
		</button>
	</div>

	{#if networks.length === 0}
		<!-- Empty / onboarding state -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;padding:40px 32px;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);margin-bottom:6px;">No networks defined</div>
			<div style="font-size:13px;color:var(--tx-3);margin-bottom:32px;">Create a Podman network to isolate or connect your containers across a host.</div>

			<!-- 3-step guide -->
			<div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:16px;margin-bottom:32px;">
				{#each [
					{ n: '1', title: 'Choose a host', desc: 'Select which machine the network lives on. Networks are scoped to a single host.' },
					{ n: '2', title: 'Configure the network', desc: 'Give it a name, pick a driver (bridge is the default), and optionally set a CIDR range.' },
					{ n: '3', title: 'Attach apps', desc: 'Connect deployments at deploy time or add them from the app detail page.' },
				] as guideStep (guideStep.n)}
					<div style="background:var(--card-2);border:1px solid var(--line);border-radius:10px;padding:18px 16px;">
						<div style="width:28px;height:28px;border-radius:50%;background:var(--grn-dim);border:1px solid var(--grn-line);display:flex;align-items:center;justify-content:center;margin-bottom:12px;">
							<span style="font-family:'JetBrains Mono',monospace;font-size:12px;font-weight:700;color:var(--grn-2);">{guideStep.n}</span>
						</div>
						<div style="font-size:13.5px;font-weight:600;color:var(--tx);margin-bottom:5px;">{guideStep.title}</div>
						<div style="font-size:12px;color:var(--tx-3);line-height:1.55;">{guideStep.desc}</div>
					</div>
				{/each}
			</div>

			<!-- Driver pills -->
			<div style="display:flex;flex-wrap:wrap;gap:8px;margin-bottom:28px;">
				{#each [
					{ label: 'bridge',  color: '#7aa6f5', desc: 'Default — software bridge on host' },
					{ label: 'macvlan', color: '#c09bf0', desc: 'Physical device appearance' },
					{ label: 'ipvlan',  color: '#efc060', desc: 'Shares parent MAC address' },
				] as d (d.label)}
					<div style="display:flex;align-items:center;gap:8px;background:var(--card-2);border:1px solid var(--line);border-radius:7px;padding:6px 12px;">
						<div style="width:8px;height:8px;border-radius:50%;background:{d.color};"></div>
						<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx-2);">{d.label}</span>
						<span style="font-size:12px;color:var(--tx-3);">{d.desc}</span>
					</div>
				{/each}
			</div>

			<button onclick={openPanel} style="display:inline-flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border-radius:9px;padding:9px 18px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
				Create your first network
			</button>
		</div>

	{:else}
		<!-- Filter tabs -->
		<div style="display:flex;align-items:center;gap:14px;margin-bottom:14px;">
			<div style="display:flex;gap:2px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:3px;">
				<button onclick={() => filter = 'all'} style={filterStyle('all')}>All</button>
				<button onclick={() => filter = 'bridge'} style={filterStyle('bridge')}>Bridge</button>
				<button onclick={() => filter = 'internal'} style={filterStyle('internal')}>Internal</button>
			</div>
		</div>

		<!-- Networks table -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
			<!-- Header row -->
			<div style="display:grid;grid-template-columns:1.6fr 100px 1.3fr 1.4fr 120px 90px 80px;gap:12px;padding:11px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
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
				<div class="net-row" style="padding:13px 18px;display:grid;grid-template-columns:1.6fr 100px 1.3fr 1.4fr 120px 90px 80px;gap:12px;align-items:center;border-bottom:1px solid var(--line);">
					<!-- Name -->
					<div style="display:flex;align-items:center;gap:11px;min-width:0;">
						<div style="width:34px;height:34px;border-radius:9px;background:rgba(255,255,255,.05);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;flex:none;">
							{@render NetworkIcon()}
						</div>
						<div style="min-width:0;">
							<div style="font-size:14px;font-weight:600;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{net.name}</div>
							<div style="font-size:11px;color:var(--tx-3);">{net.internal ? 'internal · isolated' : 'external · routable'}</div>
						</div>
					</div>

					<!-- Driver badge -->
					<div>
						<span style="font-size:12px;font-weight:600;color:{dm.color};background:{dm.bg};border-radius:5px;padding:3px 8px;">{dm.label}</span>
					</div>

					<!-- Subnet + gateway -->
					<div>
						<div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx-2);">{net.subnet}</div>
						<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--tx-3);">gw {net.gateway}</div>
					</div>

					<!-- Apps -->
					<div style="display:flex;flex-wrap:wrap;gap:4px;min-width:0;">
						{#each net.connectedApps.slice(0, 3) as appId (appId)}
							<span style="font-size:11px;color:var(--tx-2);background:var(--card-2);border:1px solid var(--line);border-radius:4px;padding:2px 6px;white-space:nowrap;">{appId}</span>
						{/each}
						{#if net.connectedApps.length > 3}
							<span style="font-size:11px;color:var(--tx-3);background:var(--card-2);border:1px solid var(--line);border-radius:4px;padding:2px 6px;">+{net.connectedApps.length - 3} more</span>
						{/if}
					</div>

					<!-- Host -->
					<div style="font-size:12.5px;color:var(--tx-2);">{net.host}</div>

					<!-- Created -->
					<div style="font-size:12px;color:var(--tx-3);">{net.created}</div>

					<!-- Inspect -->
					<div style="text-align:right;">
						<button class="inspect-btn" style="font-size:12.5px;color:var(--tx-3);cursor:pointer;padding:4px 8px;border-radius:6px;">Inspect →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div style="padding:40px;text-align:center;color:var(--tx-3);font-size:13.5px;">No networks match the current filter.</div>
			{/if}
		</div>
	{/if}
</div>

<!-- ── Add Network Panel ──────────────────────────────────────────────── -->
{#if panelOpen}
	<!-- Backdrop -->
	<div
		role="button"
		tabindex="-1"
		onclick={closePanel}
		onkeydown={(e) => e.key === 'Escape' && closePanel()}
		style="position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:200;"
	></div>

	<!-- Panel -->
	<div style="position:fixed;top:0;right:0;width:480px;height:100vh;background:var(--panel);border-left:1px solid var(--line);z-index:201;display:flex;flex-direction:column;overflow:hidden;">
		<!-- Header -->
		<div style="padding:20px 24px 16px;border-bottom:1px solid var(--line);display:flex;align-items:center;gap:12px;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);">
				{formStep === 'form' ? 'Add network' : 'Network created'}
			</div>
			<div style="flex:1"></div>
			<button onclick={closePanel} aria-label="Close panel" style="color:var(--tx-3);cursor:pointer;display:flex;align-items:center;">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<!-- Body -->
		<div style="flex:1;overflow-y:auto;padding:24px;">

			{#if formStep === 'form'}
				<div style="display:flex;flex-direction:column;gap:18px;">

					<!-- Name -->
					<div>
						<label for="net-name" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">NETWORK NAME</label>
						<input
							id="net-name"
							bind:value={netName}
							placeholder="bakery-net"
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
						/>
					</div>

					<!-- Host -->
					<div>
						<label for="net-host" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">HOST</label>
						<select
							id="net-host"
							bind:value={netHost}
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;appearance:none;cursor:pointer;"
						>
							{#each hosts as h (h.name)}
								<option value={h.name}>{h.name} — {h.location}</option>
							{/each}
						</select>
						<div style="font-size:11.5px;color:var(--tx-3);margin-top:5px;">Networks are scoped to a single host.</div>
					</div>

					<!-- Driver -->
					<div>
						<label for="net-driver" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">DRIVER</label>
						<select
							id="net-driver"
							bind:value={netDriver}
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;appearance:none;cursor:pointer;"
						>
							<option value="bridge">bridge — default software bridge</option>
							<option value="macvlan">macvlan — appear as physical device</option>
							<option value="ipvlan">ipvlan — shares parent MAC</option>
						</select>
						{#if netDriver !== 'bridge'}
							<div style="margin-top:8px;background:var(--card-2);border:1px solid var(--line);border-radius:8px;padding:10px 12px;font-size:12px;color:var(--tx-3);line-height:1.5;">
								{networkDriverMeta(netDriver).desc}
							</div>
						{/if}
					</div>

					<!-- Subnet -->
					<div>
						<label for="net-subnet" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">SUBNET <span style="font-weight:400;color:var(--tx-3);">— optional</span></label>
						<input
							id="net-subnet"
							bind:value={netSubnet}
							placeholder="10.88.0.0/16"
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
						/>
						<div style="font-size:11.5px;color:var(--tx-3);margin-top:5px;">Leave blank to let Podman assign automatically.</div>
					</div>

					<!-- Internal toggle -->
					<div>
						<div style="font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:8px;letter-spacing:.03em;">CONNECTIVITY</div>
						<button
							onclick={() => netInternal = !netInternal}
							style="display:flex;align-items:center;gap:12px;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:10px 14px;width:100%;cursor:pointer;text-align:left;"
						>
							<!-- Toggle -->
							<div style="width:36px;height:20px;border-radius:10px;background:{netInternal ? 'var(--grn)' : 'rgba(255,255,255,.1)'};position:relative;transition:background 0.2s;flex:none;">
								<div style="position:absolute;top:3px;left:{netInternal ? '19px' : '3px'};width:14px;height:14px;border-radius:50%;background:#fff;transition:left 0.2s;"></div>
							</div>
							<div>
								<div style="font-size:13px;font-weight:600;color:var(--tx);">{netInternal ? 'Internal (isolated)' : 'External'}</div>
								<div style="font-size:12px;color:var(--tx-3);">{netInternal ? 'Containers cannot reach the internet.' : 'Containers can reach external hosts.'}</div>
							</div>
						</button>
					</div>

					<!-- Create button -->
					<button
						onclick={createNetwork}
						disabled={!canCreate || creating}
						style="width:100%;display:flex;align-items:center;justify-content:center;gap:8px;background:var(--grn);color:#07130c;border-radius:9px;padding:11px;font-size:14px;font-weight:700;box-shadow:0 2px 12px var(--grn-dim);{(!canCreate || creating) ? 'opacity:.4;cursor:not-allowed;' : 'cursor:pointer;'}"
					>
						{#if creating}
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation:bk-spin 1s linear infinite;"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
							Creating…
						{:else}
							Create network
						{/if}
					</button>
				</div>

			{:else}
				<!-- Success -->
				<div style="text-align:center;padding:24px 0;">
					<div style="width:56px;height:56px;border-radius:50%;background:var(--grn-dim);border:1px solid var(--grn-line);display:flex;align-items:center;justify-content:center;margin:0 auto 16px;">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:18px;font-weight:700;color:var(--tx);margin-bottom:6px;">Network created!</div>
					<div style="font-size:13px;color:var(--tx-3);margin-bottom:28px;">
						Your network is ready. Attach containers by selecting it at deploy time.
					</div>

					<!-- Network summary card -->
					<div style="background:var(--card);border:1px solid var(--line);border-radius:10px;text-align:left;overflow:hidden;margin-bottom:24px;">
						{#each [
							{ label: 'Name',    value: netName || 'my-network', mono: true },
							{ label: 'Driver',  value: netDriver, mono: true },
							{ label: 'Subnet',  value: resolvedSubnet, mono: true },
							{ label: 'Host',    value: netHost, mono: false },
							{ label: 'Internal', value: netInternal ? 'Yes' : 'No', mono: false },
						] as row (row.label)}
							<div style="display:flex;align-items:center;justify-content:space-between;padding:10px 16px;border-bottom:1px solid var(--line);">
								<span style="font-size:12px;color:var(--tx-3);">{row.label}</span>
								<span style="font-size:12.5px;color:var(--tx);{row.mono ? 'font-family:\'JetBrains Mono\',monospace;' : ''}">{row.value}</span>
							</div>
						{/each}
					</div>

					<button onclick={closePanel} style="width:100%;display:flex;align-items:center;justify-content:center;background:var(--grn);color:#07130c;border-radius:9px;padding:11px;font-size:14px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
						Done
					</button>
				</div>
			{/if}

		</div>
	</div>
{/if}

<!-- ── Snippets ────────────────────────────────────────────────────────── -->
{#snippet NetworkIcon()}
	<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6">
		<circle cx="12" cy="5" r="2"/>
		<circle cx="5" cy="19" r="2"/>
		<circle cx="19" cy="19" r="2"/>
		<path d="M12 7v4M12 11l-5 6M12 11l5 6"/>
	</svg>
{/snippet}

<style>
	button {
		all: unset;
		box-sizing: border-box;
		cursor: pointer;
	}
	input, select {
		box-sizing: border-box;
	}
	input:focus, select:focus {
		border-color: var(--grn) !important;
	}
	.net-row:hover {
		background: rgba(255,255,255,.02);
	}
	.net-row:last-child {
		border-bottom: none;
	}
	.inspect-btn {
		opacity: 0;
		transition: opacity 0.15s;
	}
	.net-row:hover .inspect-btn {
		opacity: 1;
	}
</style>
