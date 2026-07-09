<script lang="ts">
	import { page } from '$app/state';
	import {
		GUILD_RESOURCES,
		statusMeta,
		quadletContent,
		APP_CONTAINERS,
		LOG_LINES,
		ENV_SETS
	} from '$lib/data/bakery';
	import type { App, AppStatus } from '$lib/data/bakery';
	import StatusDot from '$lib/components/bakery/StatusDot.svelte';
	import AppIcon from '$lib/components/bakery/AppIcon.svelte';
	import BuildLogViewer from '$lib/components/bakery/BuildLogViewer.svelte';
	import NotFound from '$lib/components/bakery/NotFound.svelte';
	import { goto, invalidateAll } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const guildId = $derived(page.params.guild ?? '');
	const guildName = $derived(page.data.organization?.name ?? guildId);
	const appId = $derived(page.params.appId ?? '');
	const mockApp = $derived(GUILD_RESOURCES[guildId]?.apps.find((a) => a.id === appId));

	// Host/domain/cpu/mem/quadlet are all Phase 04/05 concepts that don't
	// exist for a real app yet — a freshly-created real app (task 10) gets
	// placeholder values for those fields rather than fabricating fake ones.
	function realAppStatus(): AppStatus {
		const latest = data.builds[0];
		if (!latest) return 'stopped';
		if (latest.status === 'building' || latest.status === 'queued') return 'building';
		if (latest.status === 'failed') return 'failed';
		return 'stopped';
	}

	const realAppDisplay = $derived.by((): App | undefined => {
		if (!data.realApp) return undefined;
		return {
			id: data.realApp.id,
			name: data.realApp.name,
			type: data.repo?.fullName ?? 'app',
			status: realAppStatus(),
			host: '— unassigned —',
			domain: '— internal —',
			cpu: 0,
			mem: '—',
			deployed: data.builds[0]?.finishedAt ? 'built' : data.builds[0] ? 'building' : 'never built',
			port: '—',
			initial: data.realApp.name.charAt(0).toUpperCase() || 'A',
			unit: `${data.realApp.name}.container`,
			quadletPath: `deploy/${data.realApp.name}.container`
		};
	});

	const app = $derived(mockApp ?? realAppDisplay);
	const meta = $derived(app ? statusMeta(app.status) : null);

	let buildingNow = $state(false);
	async function buildNow() {
		buildingNow = true;
		try {
			const res = await fetch('?/buildNow', { method: 'POST', body: new FormData() });
			if (res.ok) {
				toast.success('Build queued');
				await invalidateAll();
			} else {
				toast.error('Could not queue build');
			}
		} finally {
			buildingNow = false;
		}
	}

	type Tab =
		'overview' | 'containers' | 'network' | 'deployments' | 'logs' | 'env' | 'domains' | 'quadlet';
	let activeTab = $state<Tab>('overview');
	let envSel = $state<'production' | 'staging'>('production');
	let reveal = $state(false);

	const containers = $derived(APP_CONTAINERS[appId] ?? []);
	const envVars = $derived(
		(ENV_SETS[envSel] ?? []).map((e) => ({
			...e,
			shown: e.secret && !reveal ? '••••••••••••••••' : e.v
		}))
	);

	const sparkbars = Array.from({ length: 42 }, (_, i) => {
		const h = Math.min(
			100,
			25 + Math.round(35 * Math.abs(Math.sin(i * 0.7)) + 20 * Math.abs(Math.cos(i * 0.4)))
		);
		return { h, recent: i > 36 };
	});

	function tabCls(t: Tab) {
		return t === activeTab
			? 'px-[14px] py-[10px] text-[13.5px] font-semibold text-[var(--grn)] cursor-pointer border-b-2 border-b-[var(--grn)] -mb-px'
			: 'px-[14px] py-[10px] text-[13.5px] font-semibold text-[var(--tx-2)] cursor-pointer border-b-2 border-b-transparent -mb-px';
	}

	const tabs: Array<{ id: Tab; label: string }> = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'containers', label: 'Containers' },
		{ id: 'network', label: 'Network' },
		{ id: 'deployments', label: 'Deployments' },
		{ id: 'logs', label: 'Logs' },
		{ id: 'env', label: 'Environment' },
		{ id: 'domains', label: 'Domains & Proxy' },
		{ id: 'quadlet', label: 'Quadlet' }
	];

	const port = $derived(app?.port === '—' ? '3000' : (app?.port ?? '3000'));
</script>

<svelte:head><title>{app?.name ?? 'Project'} · {guildName} — The Bakery</title></svelte:head>

{#if !app}
	<NotFound label="App not found" detail="There's no app at this address" />
{:else}
	<div>
		<!-- App header -->
		<div class="px-7 pt-5 pb-0">
			<button
				onclick={() => goto(`/${guildId}/deploy/projects`)}
				class="inline-flex items-center gap-[6px] text-[12.5px] text-[var(--tx-2)] cursor-pointer mb-[14px]"
			>
				<svg
					width="14"
					height="14"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.2"><path d="M15 18l-6-6 6-6" /></svg
				>
				All projects
			</button>

			<div class="flex items-center gap-[15px]">
				<AppIcon initial={app.initial} status={app.status} size={52} radius={14} />
				<div class="flex-1 min-w-0">
					<div class="flex items-center gap-[11px]">
						<div class="font-heading font-bold text-[23px]">{app.name}</div>
						<div
							class="flex items-center gap-[7px] px-[10px] py-1 rounded-[20px]"
							style:background={meta?.bg}
						>
							<StatusDot status={app.status} />
							<span class="text-[12px] font-semibold" style:color={meta?.c}>{meta?.label}</span>
						</div>
					</div>
					<div class="font-mono-jb text-[12px] text-[var(--tx-3)] mt-[3px]">
						{app.type} · {app.host}
					</div>
				</div>

				{#if app.domain !== '— internal —'}
					<a
						href="https://{app.domain}"
						target="_blank"
						rel="noreferrer"
						class="flex items-center gap-[6px] text-[12.5px] text-[var(--grn-2)] no-underline font-semibold px-3 py-2 border border-[var(--grn-line)] rounded-[8px]"
					>
						<svg
							width="14"
							height="14"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"><path d="M7 17L17 7M9 7h8v8" /></svg
						>
						{app.domain}
					</a>
				{/if}

				<button
					class="flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-[9px] text-[13px] font-bold cursor-pointer"
				>
					<svg
						width="15"
						height="15"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2.3"
					>
						<path d="M21 12a9 9 0 11-3-6.7L21 8" />
						<path d="M21 3v5h-5" />
					</svg>
					Redeploy
				</button>
			</div>

			<!-- Tab bar -->
			<div class="flex gap-0.5 mt-[18px] border-b border-b-[var(--line)]">
				{#each tabs as t}
					<button onclick={() => (activeTab = t.id)} class={tabCls(t.id)}>{t.label}</button>
				{/each}
			</div>
		</div>

		<!-- Tab content -->
		<div class="px-7 py-[22px]">
			<!-- Overview -->
			{#if activeTab === 'overview'}
				<div class="grid grid-cols-4 gap-[13px] mb-4">
					{#each [{ label: 'CPU', value: app.cpu + '%' }, { label: 'Memory', value: app.mem }, { label: 'Uptime', value: app.status === 'running' ? '4d 6h' : '—' }, { label: 'Restarts (24h)', value: app.status === 'failed' ? '7' : '0' }] as m (m.label)}
						<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] px-4 py-[15px]">
							<div class="text-[12px] text-[var(--tx-2)]">{m.label}</div>
							<div class="font-mono-jb font-semibold text-[20px] mt-[7px] text-[var(--tx)]">
								{m.value}
							</div>
						</div>
					{/each}
				</div>

				<div class="grid grid-cols-[1.6fr_1fr] gap-4">
					<!-- CPU chart -->
					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] px-[18px] py-4">
						<div class="text-[13px] font-bold mb-[14px]">CPU & Memory · last hour</div>
						<div class="flex items-end gap-[3px] h-[110px]">
							{#each sparkbars as b}
								<div
									class="flex-1 rounded-t-[2px]"
									class:bg-[var(--grn)]={b.recent}
									class:bg-[var(--grn-dim)]={!b.recent}
									style:height="{b.h}%"
								></div>
							{/each}
						</div>
						<div class="flex justify-between font-mono-jb text-[10.5px] text-[var(--tx-3)] mt-2">
							<span>60m ago</span><span>now</span>
						</div>
					</div>

					<!-- Details -->
					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] px-[18px] py-4">
						<div class="text-[13px] font-bold mb-3">Details</div>
						{#each [{ k: 'Host', v: app.host }, { k: 'Image', v: 'ghcr.io/…/' + app.name }, { k: 'Port', v: port }, { k: 'Quadlet', v: app.quadletPath
									.split('/')
									.pop() }, { k: 'Restart policy', v: 'always' }, { k: 'Auto-update', v: 'registry' }] as d (d.k)}
							<div class="flex justify-between gap-[10px] py-[7px] border-b border-b-[var(--line)]">
								<span class="text-[12.5px] text-[var(--tx-2)]">{d.k}</span>
								<span class="font-mono-jb text-[12px] text-[var(--tx)] text-right">{d.v}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Containers -->
			{#if activeTab === 'containers'}
				<div class="flex items-center justify-between mb-[14px]">
					<div>
						<div class="text-[15px] font-bold">Containers</div>
						<div class="text-[12.5px] text-[var(--tx-2)]">
							Every container running inside <span class="text-[var(--tx)]">{app.name}</span>, from
							one quadlet pod.
						</div>
					</div>
					<button
						class="flex items-center gap-[7px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-semibold cursor-pointer"
					>
						<svg
							width="14"
							height="14"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2.2"><path d="M12 5v14M5 12h14" /></svg
						>
						Add container
					</button>
				</div>
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden">
					<div
						class="grid grid-cols-[1.4fr_1.4fr_1fr_90px_90px] gap-[14px] px-[18px] py-[10px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]"
					>
						<div>CONTAINER</div>
						<div>IMAGE</div>
						<div>NETWORKS</div>
						<div>CPU</div>
						<div>MEM</div>
					</div>
					{#each containers as c (c.name)}
						<div
							class="grid grid-cols-[1.4fr_1.4fr_1fr_90px_90px] gap-[14px] items-center px-[18px] py-3 border-b border-b-[var(--line)] last:border-b-0"
						>
							<div class="flex items-center gap-[9px] min-w-0">
								<StatusDot status={c.status} size={8} />
								<span
									class="font-mono-jb text-[13px] text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis"
									>{c.name}</span
								>
							</div>
							<span
								class="font-mono-jb text-[11.5px] text-[var(--tx-2)] whitespace-nowrap overflow-hidden text-ellipsis"
								>{c.image}</span
							>
							<span class="text-[11.5px] text-[var(--tx-2)]">{c.nets}</span>
							<span class="font-mono-jb text-[12px] text-[var(--tx)]">{c.cpu}</span>
							<span class="font-mono-jb text-[12px] text-[var(--tx)]">{c.mem}</span>
						</div>
					{/each}
					{#if containers.length === 0}
						<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
							No container data available.
						</div>
					{/if}
				</div>
			{/if}

			<!-- Network (visual placeholder) -->
			{#if activeTab === 'network'}
				<div
					class="bg-[#0c0d13] border border-[var(--line)] rounded-[14px] p-[60px] text-center text-[var(--tx-3)]"
				>
					<svg
						width="48"
						height="48"
						viewBox="0 0 24 24"
						fill="none"
						stroke="var(--tx-3)"
						stroke-width="1.2"
						class="mx-auto mb-3"
					>
						<circle cx="12" cy="12" r="3" /><circle cx="4" cy="6" r="2" /><circle
							cx="20"
							cy="6"
							r="2"
						/>
						<circle cx="4" cy="18" r="2" /><circle cx="20" cy="18" r="2" />
						<path d="M6 6l4 4M14 14l4 4M6 18l4-4M14 10l4-4" />
					</svg>
					<div class="text-[14px] font-semibold text-[var(--tx-2)]">Network topology</div>
					<div class="text-[12.5px] mt-1">Interactive network graph for {app.name}</div>
				</div>
			{/if}

			<!-- Deployments / Build history -->
			{#if activeTab === 'deployments'}
				{#if data.realApp}
					<div class="flex items-center justify-between mb-[14px]">
						<div class="text-[12.5px] text-[var(--tx-2)]">
							Real builds for <span class="text-[var(--tx)]">{app.name}</span>, from {data.repo
								?.fullName ?? 'its repo'}.
						</div>
						<button
							onclick={buildNow}
							disabled={buildingNow}
							class="flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-[9px] text-[13px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
							>{buildingNow ? 'Queuing…' : 'Build now'}</button
						>
					</div>

					{#if data.builds.length > 0}
						{@const top = data.builds[0]}
						{#if top.status === 'queued' || top.status === 'building'}
							<div
								class="mb-[14px] bg-[#080c09] border border-[var(--line)] rounded-[12px] overflow-hidden"
							>
								<div
									class="px-4 py-[10px] border-b border-b-[var(--line)] bg-[var(--card)] text-[12.5px] font-semibold"
								>
									Live log · {top.commitSha.slice(0, 7)} · {top.status}
								</div>
								<BuildLogViewer logsUrl={`/api/v1/builds/${top.id}/logs`} />
							</div>
						{/if}
					{/if}

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden">
						{#each data.builds as b (b.id)}
							<div
								class="grid grid-cols-[auto_1fr_auto] gap-[14px] items-center px-[18px] py-[14px] border-b border-b-[var(--line)] last:border-b-0"
							>
								<div
									class="size-[10px] rounded-full shrink-0"
									style:background={b.status === 'succeeded'
										? 'var(--grn)'
										: b.status === 'failed'
											? '#e5654b'
											: '#e0a83e'}
								></div>
								<div>
									<div class="font-mono-jb text-[13.5px] text-[var(--tx)] font-semibold">
										{b.commitSha.slice(0, 7)} · {b.branch}
									</div>
									<div class="text-[11.5px] text-[var(--tx-3)] mt-[2px]">
										triggered by {b.triggeredBy ?? 'unknown'}
									</div>
								</div>
								<div class="text-right">
									<div
										class="text-[12px] font-semibold"
										style:color={b.status === 'succeeded'
											? '#52cc96'
											: b.status === 'failed'
												? '#f0836b'
												: '#e0a83e'}
									>
										{b.status}
									</div>
									<div class="text-[11px] text-[var(--tx-3)] mt-[2px]">
										{b.finishedAt
											? new Date(b.finishedAt).toLocaleString()
											: b.startedAt
												? new Date(b.startedAt).toLocaleString()
												: 'queued'}
									</div>
								</div>
							</div>
						{/each}
						{#if data.builds.length === 0}
							<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
								No builds yet — click "Build now" to trigger the first one.
							</div>
						{/if}
					</div>
				{:else}
					<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
						Build history isn't available for this demo app.
					</div>
				{/if}
			{/if}

			<!-- Logs -->
			{#if activeTab === 'logs'}
				<div class="bg-[#080c09] border border-[var(--line)] rounded-[12px] overflow-hidden">
					<div
						class="flex items-center gap-[10px] px-4 py-[10px] border-b border-b-[var(--line)] bg-[var(--card)]"
					>
						<div class="flex items-center gap-[7px]">
							<div class="size-2 rounded-full bg-[var(--grn)] animate-[bk-pulse_2s_infinite]"></div>
							<span class="text-[12.5px] font-semibold">Live</span>
						</div>
						<span class="font-mono-jb text-[11.5px] text-[var(--tx-3)]"
							>journalctl --user -u {app.unit} -f</span
						>
						<div class="flex-1"></div>
						<span class="text-[11.5px] text-[var(--tx-3)]">stdout · stderr</span>
					</div>
					<div
						class="px-4 py-[14px] font-mono-jb text-[12px] leading-[1.7] max-h-[360px] overflow-y-auto"
					>
						{#each LOG_LINES as l (l.t + l.msg)}
							<div class="flex gap-3">
								<span class="text-[var(--tx-3)] shrink-0">{l.t}</span>
								<span class="shrink-0 w-[52px]" style:color={l.color}>{l.lvl}</span>
								<span class="text-[var(--tx)] whitespace-pre-wrap">{l.msg}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Environment -->
			{#if activeTab === 'env'}
				<div class="flex items-center justify-between mb-[14px]">
					<div>
						<div class="text-[15px] font-bold">Environment</div>
						<div class="text-[12.5px] text-[var(--tx-2)]">
							Scoped to <span class="text-[var(--tx)]">{app.name}</span> only · written to
							<span class="font-mono-jb text-[var(--grn-2)]">/etc/bakery/{app.unit}.env</span>
						</div>
					</div>
					<div class="flex gap-2">
						<button
							class="bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-semibold cursor-pointer"
							>Import .env</button
						>
						<button
							onclick={() => (reveal = !reveal)}
							class="bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-semibold cursor-pointer"
							>{reveal ? 'Hide values' : 'Reveal values'}</button
						>
					</div>
				</div>
				<div
					class="flex gap-[3px] bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px] w-fit mb-[14px]"
				>
					{#each ['production', 'staging'] as const as tab}
						<button
							onclick={() => (envSel = tab)}
							class="px-[11px] py-[5px] text-[12.5px] rounded-[6px] cursor-pointer {envSel === tab
								? 'bg-[var(--card-2)] text-[var(--tx)]'
								: 'text-[var(--tx-2)]'}">{tab.charAt(0).toUpperCase() + tab.slice(1)}</button
						>
					{/each}
				</div>
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden">
					{#each envVars as e (e.key)}
						<div
							class="grid grid-cols-[1fr_1.6fr_auto] gap-[14px] items-center px-[18px] py-[11px] border-b border-b-[var(--line)] last:border-b-0"
						>
							<span class="font-mono-jb text-[12.5px] text-[var(--grn-2)] font-medium">{e.key}</span
							>
							<span
								class="font-mono-jb text-[12.5px] text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis"
								>{e.shown}</span
							>
							<span
								class="font-mono-jb text-[10.5px] font-semibold px-2 py-[3px] rounded-[6px]"
								style:background={e.secret ? 'rgba(224,168,62,.14)' : 'rgba(255,255,255,.06)'}
								style:color={e.secret ? '#efc060' : 'var(--tx-3)'}
								>{e.secret ? 'secret' : 'variable'}</span
							>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Domains & Proxy -->
			{#if activeTab === 'domains'}
				<div class="grid grid-cols-2 gap-4">
					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[18px]">
						<div class="text-[14px] font-bold mb-1">Domain</div>
						<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">
							Public address routed to this app.
						</div>
						<div
							class="flex items-center gap-[10px] bg-[var(--card-2)] border border-[var(--line)] rounded-[9px] px-[14px] py-[11px]"
						>
							<svg
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="var(--grn)"
								stroke-width="1.8"
							>
								<circle cx="12" cy="12" r="9" />
								<path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18" />
							</svg>
							<span class="font-mono-jb text-[13px] text-[var(--tx)]">{app.domain}</span>
							{#if app.domain !== '— internal —'}
								<span class="ml-auto flex items-center gap-[5px] text-[11.5px] text-[var(--grn)]">
									<svg
										width="13"
										height="13"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="2.3"><path d="M20 6L9 17l-5-5" /></svg
									>
									DNS ok
								</span>
							{/if}
						</div>
					</div>

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[18px]">
						<div class="flex items-center justify-between mb-1">
							<div class="text-[14px] font-bold">Caddy reverse proxy</div>
							<div class="flex items-center gap-[7px] text-[11.5px] text-[var(--grn)]">
								<div class="size-[7px] rounded-full bg-[var(--grn)]"></div>
								Active · auto HTTPS
							</div>
						</div>
						<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">
							Bakery manages a single Caddy instance per host.
						</div>
						<pre
							class="font-mono-jb text-[12px] leading-[1.8] bg-[#080c09] border border-[var(--line)] rounded-[9px] px-[15px] py-[13px] text-[var(--tx-2)] overflow-x-auto m-0"><span
								class="text-[var(--grn-2)]">{app.domain}</span
							> {'{'}
  reverse_proxy <span class="text-[var(--amber)]">127.0.0.1:{port}</span>
  encode zstd gzip
{'}'}</pre>
					</div>
				</div>
			{/if}

			<!-- Quadlet -->
			{#if activeTab === 'quadlet'}
				<div class="flex items-center justify-between mb-3">
					<div>
						<div class="text-[15px] font-bold">Quadlet</div>
						<div class="text-[12.5px] text-[var(--tx-2)]">
							The Podman Quadlet that defines this app, sourced from <span
								class="font-mono-jb text-[var(--grn-2)]">{app.quadletPath}</span
							>.
						</div>
					</div>
					<span
						class="font-mono-jb text-[11px] px-[10px] py-[5px] rounded-[6px] bg-[var(--grn-dim)] text-[var(--grn)]"
						>.container</span
					>
				</div>
				<pre
					class="font-mono-jb text-[12.5px] leading-[1.85] bg-[#080c09] border border-[var(--line)] rounded-[12px] px-5 py-[18px] overflow-x-auto text-[var(--tx)] m-0 [tab-size:2]">{data.realApp
						? (data.realQuadletContent ??
							'No successful build yet — the Quadlet unit is generated from the latest built image.')
						: quadletContent(app)}</pre>
			{/if}
		</div>
	</div>
{/if}
