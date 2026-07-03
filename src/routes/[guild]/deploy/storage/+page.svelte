<script lang="ts">
	import { page } from '$app/state';
	import { VOLUMES, GUILDS, volumeDriverMeta, type VolumeDriver } from '$lib/data/bakery';

	const guildId   = $derived(page.params.guild ?? '');
	const volumes   = $derived(VOLUMES[guildId] ?? []);
	const hosts     = $derived(GUILDS[guildId]?.hosts ?? []);
	const totalSize = $derived(volumes.map((v) => v.size).join(' · '));

	let filter = $state<'all' | 'local' | 'nfs' | 'tmpfs'>('all');

	const filtered = $derived(
		filter === 'all' ? volumes : volumes.filter((v) => v.driver === filter)
	);

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}

	let panelOpen = $state(false);
	let formStep  = $state<'form' | 'done'>('form');
	let volName   = $state('');
	let volHost   = $state('');
	let volDriver = $state<VolumeDriver>('local');
	let nfsServer = $state('');
	let nfsPath   = $state('');
	let tmpfsSize = $state('');
	let creating  = $state(false);

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

	const panelInputCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-mono-jb';
	const panelInputSmCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-2 text-[13px] text-[var(--tx)] font-mono-jb';
	const panelSelectCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-bakery appearance-none cursor-pointer';
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-start gap-[14px] mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Storage</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{#if volumes.length > 0}
					{volumes.length} volume{volumes.length !== 1 ? 's' : ''} · {totalSize}
				{:else}
					Persistent volumes for your container deployments.
				{/if}
			</div>
		</div>
		<div class="flex-1"></div>
		<button onclick={openPanel} class="flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[9px] px-[14px] py-2 text-[13.5px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
			Add volume
		</button>
	</div>

	{#if volumes.length === 0}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[40px_32px]">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)] mb-[6px]">No volumes created</div>
			<div class="text-[13px] text-[var(--tx-3)] mb-8">Create a Podman volume to persist data across container restarts.</div>

			<div class="grid grid-cols-3 gap-4 mb-8">
				{#each [
					{ n: '1', title: 'Choose a host', desc: 'Pick which machine holds the volume. Volumes are scoped to a single host.' },
					{ n: '2', title: 'Name the volume', desc: 'Give it a unique name. Podman manages the path on disk automatically.' },
					{ n: '3', title: 'Mount it', desc: 'Attach the volume to apps at deploy time or from the app detail page.' },
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
					{ label: 'local', color: '#7aa6f5', desc: 'Host filesystem (default)' },
					{ label: 'NFS',   color: '#c09bf0', desc: 'Remote network share' },
					{ label: 'tmpfs', color: '#efc060', desc: 'In-memory, ephemeral' },
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
				Create your first volume
			</button>
		</div>

	{:else}
		<div class="flex items-center gap-[14px] mb-[14px]">
			<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
				<button onclick={() => filter = 'all'}   class={filterCls('all')}>All</button>
				<button onclick={() => filter = 'local'} class={filterCls('local')}>Local</button>
				<button onclick={() => filter = 'nfs'}   class={filterCls('nfs')}>NFS</button>
				<button onclick={() => filter = 'tmpfs'} class={filterCls('tmpfs')}>tmpfs</button>
			</div>
		</div>

		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div class="grid grid-cols-[1.4fr_90px_90px_2fr_1fr_100px_80px] gap-3 px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
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
				<div class="group px-[18px] py-[13px] grid grid-cols-[1.4fr_90px_90px_2fr_1fr_100px_80px] gap-3 items-center border-b border-b-[var(--line)] last:border-b-0 hover:bg-white/[0.02]">
					<div class="flex items-center gap-[11px] min-w-0">
						<div class="size-[34px] rounded-[9px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0">
							{@render VolumeIcon()}
						</div>
						<div class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{vol.name}</div>
					</div>

					<div>
						<span class="text-[12px] font-semibold rounded-[5px] px-2 py-[3px]" style:color={dm.color} style:background={dm.bg}>{dm.label}</span>
					</div>

					<div class="font-mono-jb text-[12.5px] text-[var(--tx-2)]">{vol.size}</div>

					<div class="font-mono-jb text-[11px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis" title={vol.mountpoint}>{vol.mountpoint}</div>

					<div class="flex flex-wrap gap-1 min-w-0">
						{#each vol.attachedTo.slice(0, 2) as appId (appId)}
							<span class="text-[11px] text-[var(--tx-2)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px] whitespace-nowrap">{appId}</span>
						{/each}
						{#if vol.attachedTo.length > 2}
							<span class="text-[11px] text-[var(--tx-3)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px]">+{vol.attachedTo.length - 2} more</span>
						{/if}
						{#if vol.attachedTo.length === 0}
							<span class="text-[11px] text-[var(--tx-3)]">—</span>
						{/if}
					</div>

					<div class="text-[12.5px] text-[var(--tx-2)]">{vol.host}</div>

					<div class="text-right">
						<button class="opacity-0 transition-opacity duration-150 group-hover:opacity-100 text-[12.5px] text-[var(--tx-3)] cursor-pointer px-2 py-1 rounded-[6px]">Inspect →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div class="p-10 text-center text-[var(--tx-3)] text-[13.5px]">No volumes match the current filter.</div>
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
				{formStep === 'form' ? 'Add volume' : 'Volume created'}
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
						<label for="vol-name" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">VOLUME NAME</label>
						<input id="vol-name" bind:value={volName} placeholder="my-volume" class={panelInputCls} />
					</div>

					<div>
						<label for="vol-host" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">HOST</label>
						<select id="vol-host" bind:value={volHost} class={panelSelectCls}>
							{#each hosts as h (h.name)}
								<option value={h.name}>{h.name} — {h.location}</option>
							{/each}
						</select>
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Volumes are scoped to a single host.</div>
					</div>

					<div>
						<label for="vol-driver" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">DRIVER</label>
						<select id="vol-driver" bind:value={volDriver} class={panelSelectCls}>
							<option value="local">local — host filesystem (default)</option>
							<option value="nfs">NFS — remote network share</option>
							<option value="tmpfs">tmpfs — in-memory, ephemeral</option>
						</select>

						{#if volDriver === 'local'}
							<div class="mt-2 text-[12px] text-[var(--tx-3)] leading-[1.5]">Podman manages the path automatically. Data persists under <code class="font-mono-jb bg-white/[.06] px-1 py-[1px] rounded-[3px]">/var/lib/containers/storage/volumes/</code></div>
						{/if}

						{#if volDriver === 'nfs'}
							<div class="mt-[10px] flex flex-col gap-[10px]">
								<div>
									<label for="vol-nfs-server" class="block text-[11.5px] font-semibold text-[var(--tx-3)] mb-[5px] tracking-[.03em]">NFS SERVER</label>
									<input id="vol-nfs-server" bind:value={nfsServer} placeholder="192.168.1.10" class={panelInputSmCls} />
								</div>
								<div>
									<label for="vol-nfs-path" class="block text-[11.5px] font-semibold text-[var(--tx-3)] mb-[5px] tracking-[.03em]">EXPORT PATH</label>
									<input id="vol-nfs-path" bind:value={nfsPath} placeholder="/exports/volumes/my-volume" class={panelInputSmCls} />
								</div>
							</div>
						{/if}

						{#if volDriver === 'tmpfs'}
							<div class="mt-[10px]">
								<label for="vol-tmpfs-size" class="block text-[11.5px] font-semibold text-[var(--tx-3)] mb-[5px] tracking-[.03em]">SIZE LIMIT <span class="font-normal">— optional</span></label>
								<input id="vol-tmpfs-size" bind:value={tmpfsSize} placeholder="256m" class={panelInputSmCls} />
								<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Data is lost when the container stops. Useful for caches and scratch space.</div>
							</div>

							<div class="mt-2 bg-[rgba(224,168,62,.08)] border border-[rgba(224,168,62,.2)] rounded-[8px] px-3 py-[10px] flex gap-2">
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#efc060" stroke-width="2" class="shrink-0 mt-[1px]"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
								<div class="text-[12px] text-[#efc060] leading-[1.5]">tmpfs volumes are not persistent. Do not use for databases or uploaded files.</div>
							</div>
						{/if}
					</div>

					<button
						onclick={createVolume}
						disabled={!canCreate || creating}
						class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold shadow-[0_2px_12px_var(--grn-dim)] {(!canCreate || creating) ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
					>
						{#if creating}
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="animate-[bk-spin_1s_linear_infinite]"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
							Creating…
						{:else}
							Create volume
						{/if}
					</button>
				</div>

			{:else}
				<div class="text-center py-6">
					<div class="size-14 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mx-auto mb-4">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div class="font-heading text-[18px] font-bold text-[var(--tx)] mb-[6px]">Volume created!</div>
					<div class="text-[13px] text-[var(--tx-3)] mb-7">Mount it to an app at deploy time or from the app's environment tab.</div>

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[10px] text-left overflow-hidden mb-6">
						{#each [
							{ label: 'Name',   value: volName || 'my-volume', mono: true },
							{ label: 'Driver', value: volDriver, mono: true },
							{ label: 'Host',   value: volHost, mono: false },
							...(volDriver === 'tmpfs' ? [{ label: 'Size limit', value: tmpfsSize || 'unlimited', mono: true }] : []),
							...(volDriver === 'nfs'   ? [{ label: 'NFS server', value: nfsServer || '—', mono: true }, { label: 'Export path', value: nfsPath || '—', mono: true }] : []),
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

{#snippet VolumeIcon()}
	<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6">
		<ellipse cx="12" cy="6" rx="7" ry="3"/>
		<path d="M5 6v12c0 1.6 3.1 3 7 3s7-1.4 7-3V6"/>
		<path d="M5 12c0 1.6 3.1 3 7 3s7-1.4 7-3"/>
	</svg>
{/snippet}
