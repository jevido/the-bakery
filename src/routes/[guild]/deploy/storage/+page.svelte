<script lang="ts">
	import { page } from '$app/state';
	import { VOLUMES, GUILDS, volumeDriverMeta, type VolumeDriver } from '$lib/data/bakery';

	const guildId  = $derived(page.params.guild ?? '');
	const volumes  = $derived(VOLUMES[guildId] ?? []);
	const hosts    = $derived(GUILDS[guildId]?.hosts ?? []);
	const totalSize = $derived(volumes.map((v) => v.size).join(' · '));

	let filter = $state<'all' | 'local' | 'nfs' | 'tmpfs'>('all');

	const filtered = $derived(
		filter === 'all' ? volumes : volumes.filter((v) => v.driver === filter)
	);

	function filterStyle(f: typeof filter) {
		return f === filter
			? 'padding:5px 11px;font-size:12.5px;color:var(--tx);border-radius:6px;cursor:pointer;background:var(--card-2);'
			: 'padding:5px 11px;font-size:12.5px;color:var(--tx-2);border-radius:6px;cursor:pointer;';
	}

	// ── Add volume panel ──────────────────────────────────────────────
	let panelOpen  = $state(false);
	let formStep   = $state<'form' | 'done'>('form');
	let volName    = $state('');
	let volHost    = $state('');
	let volDriver  = $state<VolumeDriver>('local');
	let nfsServer  = $state('');
	let nfsPath    = $state('');
	let tmpfsSize  = $state('');
	let creating   = $state(false);

	const canCreate = $derived(volName.trim().length > 0 && volHost.length > 0);

	function openPanel() {
		panelOpen = true;
		formStep  = 'form';
		volName   = '';
		volHost   = hosts[0]?.name ?? '';
		volDriver = 'local';
		nfsServer = '';
		nfsPath   = '';
		tmpfsSize = '';
		creating  = false;
	}

	function closePanel() { panelOpen = false; }

	async function createVolume() {
		creating = true;
		await new Promise((r) => setTimeout(r, 800));
		creating = false;
		formStep = 'done';
	}
</script>

<div style="padding: 22px 28px;">
	<!-- Header -->
	<div style="display:flex;align-items:flex-start;gap:14px;margin-bottom:22px;">
		<div>
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;letter-spacing:-.01em;margin-bottom:3px;">Storage</div>
			<div style="font-size:13px;color:var(--tx-2);">
				{#if volumes.length > 0}
					{volumes.length} volume{volumes.length !== 1 ? 's' : ''} · {totalSize}
				{:else}
					Persistent volumes for your container deployments.
				{/if}
			</div>
		</div>
		<div style="flex:1"></div>
		<button onclick={openPanel} style="display:flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border-radius:9px;padding:8px 14px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
			Add volume
		</button>
	</div>

	{#if volumes.length === 0}
		<!-- Empty / onboarding -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;padding:40px 32px;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);margin-bottom:6px;">No volumes created</div>
			<div style="font-size:13px;color:var(--tx-3);margin-bottom:32px;">Create a Podman volume to persist data across container restarts.</div>

			<!-- 3-step guide -->
			<div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:16px;margin-bottom:32px;">
				{#each [
					{ n: '1', title: 'Choose a host', desc: 'Pick which machine holds the volume. Volumes are scoped to a single host.' },
					{ n: '2', title: 'Name the volume', desc: 'Give it a unique name. Podman manages the path on disk automatically.' },
					{ n: '3', title: 'Mount it', desc: 'Attach the volume to apps at deploy time or from the app detail page.' },
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
					{ label: 'local', color: '#7aa6f5', desc: 'Host filesystem (default)' },
					{ label: 'NFS',   color: '#c09bf0', desc: 'Remote network share' },
					{ label: 'tmpfs', color: '#efc060', desc: 'In-memory, ephemeral' },
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
				Create your first volume
			</button>
		</div>

	{:else}
		<!-- Filter tabs -->
		<div style="display:flex;align-items:center;gap:14px;margin-bottom:14px;">
			<div style="display:flex;gap:2px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:3px;">
				<button onclick={() => filter = 'all'}   style={filterStyle('all')}>All</button>
				<button onclick={() => filter = 'local'} style={filterStyle('local')}>Local</button>
				<button onclick={() => filter = 'nfs'}   style={filterStyle('nfs')}>NFS</button>
				<button onclick={() => filter = 'tmpfs'} style={filterStyle('tmpfs')}>tmpfs</button>
			</div>
		</div>

		<!-- Volumes table -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
			<div style="display:grid;grid-template-columns:1.4fr 90px 90px 2fr 1fr 100px 80px;gap:12px;padding:11px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
				<div>VOLUME</div>
				<div>DRIVER</div>
				<div>SIZE</div>
				<div>MOUNT PATH</div>
				<div>APPS</div>
				<div>HOST</div>
				<div></div>
			</div>

			{#each filtered as vol (vol.id)}
				{@const dm = volumeDriverMeta(vol.driver)}
				<div class="vol-row" style="padding:13px 18px;display:grid;grid-template-columns:1.4fr 90px 90px 2fr 1fr 100px 80px;gap:12px;align-items:center;border-bottom:1px solid var(--line);">
					<!-- Name -->
					<div style="display:flex;align-items:center;gap:11px;min-width:0;">
						<div style="width:34px;height:34px;border-radius:9px;background:rgba(255,255,255,.05);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;flex:none;">
							{@render VolumeIcon()}
						</div>
						<div style="font-size:14px;font-weight:600;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{vol.name}</div>
					</div>

					<!-- Driver -->
					<div>
						<span style="font-size:12px;font-weight:600;color:{dm.color};background:{dm.bg};border-radius:5px;padding:3px 8px;">{dm.label}</span>
					</div>

					<!-- Size -->
					<div style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:var(--tx-2);">{vol.size}</div>

					<!-- Mount path -->
					<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--tx-3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;" title={vol.mountpoint}>{vol.mountpoint}</div>

					<!-- Apps -->
					<div style="display:flex;flex-wrap:wrap;gap:4px;min-width:0;">
						{#each vol.attachedTo.slice(0, 2) as appId (appId)}
							<span style="font-size:11px;color:var(--tx-2);background:var(--card-2);border:1px solid var(--line);border-radius:4px;padding:2px 6px;white-space:nowrap;">{appId}</span>
						{/each}
						{#if vol.attachedTo.length > 2}
							<span style="font-size:11px;color:var(--tx-3);background:var(--card-2);border:1px solid var(--line);border-radius:4px;padding:2px 6px;">+{vol.attachedTo.length - 2} more</span>
						{/if}
						{#if vol.attachedTo.length === 0}
							<span style="font-size:11px;color:var(--tx-3);">—</span>
						{/if}
					</div>

					<!-- Host -->
					<div style="font-size:12.5px;color:var(--tx-2);">{vol.host}</div>

					<!-- Inspect -->
					<div style="text-align:right;">
						<button class="inspect-btn" style="font-size:12.5px;color:var(--tx-3);cursor:pointer;padding:4px 8px;border-radius:6px;">Inspect →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div style="padding:40px;text-align:center;color:var(--tx-3);font-size:13.5px;">No volumes match the current filter.</div>
			{/if}
		</div>
	{/if}
</div>

<!-- ── Add Volume Panel ───────────────────────────────────────────────── -->
{#if panelOpen}
	<div
		role="button"
		tabindex="-1"
		onclick={closePanel}
		onkeydown={(e) => e.key === 'Escape' && closePanel()}
		style="position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:200;"
	></div>

	<div style="position:fixed;top:0;right:0;width:480px;height:100vh;background:var(--panel);border-left:1px solid var(--line);z-index:201;display:flex;flex-direction:column;overflow:hidden;">
		<!-- Panel header -->
		<div style="padding:20px 24px 16px;border-bottom:1px solid var(--line);display:flex;align-items:center;gap:12px;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);">
				{formStep === 'form' ? 'Add volume' : 'Volume created'}
			</div>
			<div style="flex:1"></div>
			<button onclick={closePanel} aria-label="Close panel" style="color:var(--tx-3);cursor:pointer;display:flex;align-items:center;">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<!-- Panel body -->
		<div style="flex:1;overflow-y:auto;padding:24px;">

			{#if formStep === 'form'}
				<div style="display:flex;flex-direction:column;gap:18px;">

					<!-- Volume name -->
					<div>
						<label for="vol-name" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">VOLUME NAME</label>
						<input
							id="vol-name"
							bind:value={volName}
							placeholder="my-volume"
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
						/>
					</div>

					<!-- Host -->
					<div>
						<label for="vol-host" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">HOST</label>
						<select
							id="vol-host"
							bind:value={volHost}
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;appearance:none;cursor:pointer;"
						>
							{#each hosts as h (h.name)}
								<option value={h.name}>{h.name} — {h.location}</option>
							{/each}
						</select>
						<div style="font-size:11.5px;color:var(--tx-3);margin-top:5px;">Volumes are scoped to a single host.</div>
					</div>

					<!-- Driver -->
					<div>
						<label for="vol-driver" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">DRIVER</label>
						<select
							id="vol-driver"
							bind:value={volDriver}
							style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;appearance:none;cursor:pointer;"
						>
							<option value="local">local — host filesystem (default)</option>
							<option value="nfs">NFS — remote network share</option>
							<option value="tmpfs">tmpfs — in-memory, ephemeral</option>
						</select>

						{#if volDriver === 'local'}
							<div style="margin-top:8px;font-size:12px;color:var(--tx-3);line-height:1.5;">Podman manages the path automatically. Data persists under <code style="font-family:'JetBrains Mono',monospace;background:rgba(255,255,255,.06);padding:1px 4px;border-radius:3px;">/var/lib/containers/storage/volumes/</code></div>
						{/if}

						{#if volDriver === 'nfs'}
							<div style="margin-top:10px;display:flex;flex-direction:column;gap:10px;">
								<div>
									<label for="vol-nfs-server" style="display:block;font-size:11.5px;font-weight:600;color:var(--tx-3);margin-bottom:5px;letter-spacing:.03em;">NFS SERVER</label>
									<input
										id="vol-nfs-server"
										bind:value={nfsServer}
										placeholder="192.168.1.10"
										style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:8px 12px;font-size:13px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
									/>
								</div>
								<div>
									<label for="vol-nfs-path" style="display:block;font-size:11.5px;font-weight:600;color:var(--tx-3);margin-bottom:5px;letter-spacing:.03em;">EXPORT PATH</label>
									<input
										id="vol-nfs-path"
										bind:value={nfsPath}
										placeholder="/exports/volumes/my-volume"
										style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:8px 12px;font-size:13px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
									/>
								</div>
							</div>
						{/if}

						{#if volDriver === 'tmpfs'}
							<div style="margin-top:10px;">
								<label for="vol-tmpfs-size" style="display:block;font-size:11.5px;font-weight:600;color:var(--tx-3);margin-bottom:5px;letter-spacing:.03em;">SIZE LIMIT <span style="font-weight:400;">— optional</span></label>
								<input
									id="vol-tmpfs-size"
									bind:value={tmpfsSize}
									placeholder="256m"
									style="width:100%;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:8px 12px;font-size:13px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
								/>
								<div style="font-size:11.5px;color:var(--tx-3);margin-top:5px;">Data is lost when the container stops. Useful for caches and scratch space.</div>
							</div>

							<div style="margin-top:8px;background:rgba(224,168,62,.08);border:1px solid rgba(224,168,62,.2);border-radius:8px;padding:10px 12px;display:flex;gap:8px;">
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#efc060" stroke-width="2" style="flex:none;margin-top:1px;"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
								<div style="font-size:12px;color:#efc060;line-height:1.5;">tmpfs volumes are not persistent. Do not use for databases or uploaded files.</div>
							</div>
						{/if}
					</div>

					<!-- Create button -->
					<button
						onclick={createVolume}
						disabled={!canCreate || creating}
						style="width:100%;display:flex;align-items:center;justify-content:center;gap:8px;background:var(--grn);color:#07130c;border-radius:9px;padding:11px;font-size:14px;font-weight:700;box-shadow:0 2px 12px var(--grn-dim);{(!canCreate || creating) ? 'opacity:.4;cursor:not-allowed;' : 'cursor:pointer;'}"
					>
						{#if creating}
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation:bk-spin 1s linear infinite;"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
							Creating…
						{:else}
							Create volume
						{/if}
					</button>
				</div>

			{:else}
				<!-- Success -->
				<div style="text-align:center;padding:24px 0;">
					<div style="width:56px;height:56px;border-radius:50%;background:var(--grn-dim);border:1px solid var(--grn-line);display:flex;align-items:center;justify-content:center;margin:0 auto 16px;">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:18px;font-weight:700;color:var(--tx);margin-bottom:6px;">Volume created!</div>
					<div style="font-size:13px;color:var(--tx-3);margin-bottom:28px;">Mount it to an app at deploy time or from the app's environment tab.</div>

					<div style="background:var(--card);border:1px solid var(--line);border-radius:10px;text-align:left;overflow:hidden;margin-bottom:24px;">
						{#each [
							{ label: 'Name',   value: volName || 'my-volume', mono: true },
							{ label: 'Driver', value: volDriver, mono: true },
							{ label: 'Host',   value: volHost, mono: false },
							...(volDriver === 'tmpfs' ? [{ label: 'Size limit', value: tmpfsSize || 'unlimited', mono: true }] : []),
							...(volDriver === 'nfs'   ? [{ label: 'NFS server', value: nfsServer || '—', mono: true }, { label: 'Export path', value: nfsPath || '—', mono: true }] : []),
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
{#snippet VolumeIcon()}
	<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6">
		<ellipse cx="12" cy="6" rx="7" ry="3"/>
		<path d="M5 6v12c0 1.6 3.1 3 7 3s7-1.4 7-3V6"/>
		<path d="M5 12c0 1.6 3.1 3 7 3s7-1.4 7-3"/>
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
	.vol-row:hover {
		background: rgba(255,255,255,.02);
	}
	.vol-row:last-child {
		border-bottom: none;
	}
	.inspect-btn {
		opacity: 0;
		transition: opacity 0.15s;
	}
	.vol-row:hover .inspect-btn {
		opacity: 1;
	}
</style>
