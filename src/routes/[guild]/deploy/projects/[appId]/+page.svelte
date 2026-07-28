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
	import { formatBytes, formatDuration, deploymentStatusColor } from '$lib/utils';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import MetricChart from '$lib/components/bakery/MetricChart.svelte';
	import { provideMetricChartCrosshair } from '$lib/components/bakery/metric-chart-crosshair.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	provideMetricChartCrosshair();

	const guildId = $derived(page.params.guild ?? '');
	const guildName = $derived(page.data.organization?.name ?? guildId);
	const appId = $derived(page.params.appId ?? '');
	const mockApp = $derived(GUILD_RESOURCES[guildId]?.apps.find((a) => a.id === appId));

	// Host/cpu/mem/quadlet are all Phase 04/05 concepts that don't exist for a
	// real app yet — a freshly-created real app (task 10) gets placeholder
	// values for those fields rather than fabricating fake ones. `domain`
	// (task 02) is real: every app gets a default subdomain row on creation.
	const realAppDomain = $derived(
		data.domains.find((d) => d.isDefaultSubdomain)?.hostname ?? '— internal —'
	);

	function realAppStatus(): AppStatus {
		// Checked first, not just the latest build's own status — a
		// successful build sits at `status: 'succeeded'`, which matched none
		// of the branches below and fell through to 'stopped' even for an
		// app with a real, currently-running deployment (found live,
		// dogfooding Bakery's own self-hosted deployment, Phase 08).
		if (data.deployments?.some((d) => d.status === 'running')) return 'running';
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
			domain: realAppDomain,
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

	let selectedConnectRepoId = $state('');
	let connectingRepo = $state(false);
	async function connectRepo() {
		if (!selectedConnectRepoId) return;
		connectingRepo = true;
		try {
			const body = new FormData();
			body.set('repoId', selectedConnectRepoId);
			const res = await fetch('?/connectRepo', { method: 'POST', body });
			if (res.ok) {
				toast.success('Repository connected');
				await invalidateAll();
			} else {
				toast.error('Could not connect that repository');
			}
		} finally {
			connectingRepo = false;
		}
	}

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

	let deploying = $state(false);
	async function deployNow() {
		deploying = true;
		try {
			const res = await fetch('?/deploy', { method: 'POST', body: new FormData() });
			if (res.ok) {
				toast.success('Deploy started');
				await invalidateAll();
			} else {
				toast.error('Could not start deploy');
			}
		} finally {
			deploying = false;
		}
	}

	// The most recently-*completed* rollout is still what's actually serving
	// traffic even while a newer one is mid-flight (task 06 never flips until
	// the new unit is healthy) — so "current" is the first `running` entry in
	// startedAt-desc order, not necessarily data.deployments[0].
	// Drives the live-ticking deployment uptime below -- no server round-trip
	// needed, since only "now" changes, not the underlying deployment data
	// (unlike the Containers tab's invalidateAll() live-refresh further down).
	let now = $state(Date.now());
	$effect(() => {
		const interval = setInterval(() => (now = Date.now()), 30_000);
		return () => clearInterval(interval);
	});

	const currentDeployment = $derived(data.deployments?.find((d) => d.status === 'running'));
	const currentDeploymentId = $derived(currentDeployment?.id);
	// `finishedAt` is set the moment a deployment reaches `running`
	// (orchestrator.ts), so a `running` row always has one in practice --
	// falling back to `startedAt` defensively rather than asserting non-null.
	const currentDeploymentUptime = $derived.by(() => {
		const since = currentDeployment?.finishedAt ?? currentDeployment?.startedAt;
		return since ? formatDuration(now - new Date(since).getTime()) : null;
	});

	// Keyed by `buildId` (stable across every row, unlike `deploymentId`,
	// which a still-building row doesn't have yet) so every row in the
	// unified Deployments list is expandable the same way, including one
	// with no deployment at all.
	let expandedDeploymentId = $state<string | null>(null);
	let expandedLogTab = $state<'runtime' | 'build'>('build');
	// Sticks once the user picks a tab by hand, so live-refresh (task 01)
	// re-landing on this row's data below doesn't yank them off a tab
	// they're actively reading -- reset on every fresh expand/collapse so
	// switching rows (or re-expanding the same one) goes back to the
	// row's own computed default rather than remembering a stale choice.
	let expandedLogTabIsManual = false;
	function toggleDeploymentLogs(buildId: string) {
		if (expandedDeploymentId !== buildId) expandedLogTabIsManual = false;
		expandedDeploymentId = expandedDeploymentId === buildId ? null : buildId;
	}
	function selectLogTab(tab: 'runtime' | 'build') {
		expandedLogTab = tab;
		expandedLogTabIsManual = true;
	}

	// Build first, Runtime once that row's build has actually finished (and
	// produced a deployment to tail) -- re-evaluated on every data refresh so
	// a row left open while its build completes (task 01's live-refresh)
	// advances on its own instead of staying stuck showing a build log that
	// will never get another line.
	$effect(() => {
		if (!expandedDeploymentId || expandedLogTabIsManual) return;
		const row = data.deploymentAttempts?.find((r) => r.buildId === expandedDeploymentId);
		if (!row) return;
		const buildFinished = row.buildStatus === 'succeeded' || row.buildStatus === 'failed';
		expandedLogTab = buildFinished && row.deploymentId ? 'runtime' : 'build';
	});

	let rollingBackId = $state<string | null>(null);
	async function rollback(deploymentId: string) {
		rollingBackId = deploymentId;
		try {
			const body = new FormData();
			body.set('deploymentId', deploymentId);
			const res = await fetch('?/rollback', { method: 'POST', body });
			if (res.ok) {
				toast.success('Rollback started');
				await invalidateAll();
			} else {
				toast.error('Could not start rollback');
			}
		} finally {
			rollingBackId = null;
		}
	}

	let stopDialogOpen = $state(false);
	let deploymentIdToStop = $state<string | null>(null);
	let stoppingId = $state<string | null>(null);
	function requestStopDeployment(deploymentId: string) {
		deploymentIdToStop = deploymentId;
		stopDialogOpen = true;
	}
	async function confirmStopDeployment() {
		const deploymentId = deploymentIdToStop;
		if (!deploymentId) return;
		stoppingId = deploymentId;
		try {
			const body = new FormData();
			body.set('deploymentId', deploymentId);
			const res = await fetch('?/stop', { method: 'POST', body });
			if (res.ok) {
				toast.success('Deployment stopped');
				await invalidateAll();
			} else {
				toast.error('Could not stop deployment');
			}
		} finally {
			stoppingId = null;
			deploymentIdToStop = null;
		}
	}

	let domainInput = $state('');
	let addingDomain = $state(false);
	async function addCustomDomain() {
		if (!domainInput.trim()) return;
		addingDomain = true;
		try {
			const body = new FormData();
			body.set('hostname', domainInput.trim());
			const res = await fetch('?/addCustomDomain', { method: 'POST', body });
			if (res.ok) {
				toast.success('Domain added — verify it once DNS is configured');
				domainInput = '';
				await invalidateAll();
			} else {
				toast.error('Could not add that domain');
			}
		} finally {
			addingDomain = false;
		}
	}

	let verifyingDomainId = $state<string | null>(null);
	async function verifyCustomDomain(domainId: string) {
		verifyingDomainId = domainId;
		try {
			const body = new FormData();
			body.set('domainId', domainId);
			const res = await fetch('?/verifyCustomDomain', { method: 'POST', body });
			if (res.ok) {
				toast.success('Domain verified');
				await invalidateAll();
			} else {
				toast.error('DNS not verified yet — check your CNAME record');
			}
		} finally {
			verifyingDomainId = null;
		}
	}

	let removingDomainId = $state<string | null>(null);
	async function removeCustomDomain(domainId: string) {
		removingDomainId = domainId;
		try {
			const body = new FormData();
			body.set('domainId', domainId);
			const res = await fetch('?/removeCustomDomain', { method: 'POST', body });
			if (res.ok) {
				toast.success('Domain removed');
				await invalidateAll();
			} else {
				toast.error('Could not remove domain');
			}
		} finally {
			removingDomainId = null;
		}
	}

	let volName = $state('');
	let volMountPath = $state('');
	let attachingVolume = $state(false);
	async function attachVolume() {
		if (!volName.trim() || !volMountPath.trim()) return;
		attachingVolume = true;
		try {
			const body = new FormData();
			body.set('name', volName.trim());
			body.set('mountPath', volMountPath.trim());
			const res = await fetch('?/attachVolume', { method: 'POST', body });
			if (res.ok) {
				toast.success('Volume attached — takes effect on the next deploy');
				volName = '';
				volMountPath = '';
				await invalidateAll();
			} else {
				toast.error('Could not attach that volume');
			}
		} finally {
			attachingVolume = false;
		}
	}

	let removingVolumeId = $state<string | null>(null);
	async function removeVolume(volumeId: string) {
		removingVolumeId = volumeId;
		try {
			const body = new FormData();
			body.set('volumeId', volumeId);
			const res = await fetch('?/removeVolume', { method: 'POST', body });
			if (res.ok) {
				toast.success('Volume detached');
				await invalidateAll();
			} else {
				toast.error('Could not detach volume');
			}
		} finally {
			removingVolumeId = null;
		}
	}

	type Tab =
		| 'overview'
		| 'containers'
		| 'network'
		| 'deployments'
		| 'logs'
		| 'env'
		| 'domains'
		| 'storage'
		| 'quadlet';
	let envSel = $state<'production' | 'staging'>('production');
	let reveal = $state(false);

	// Bakery's real deploy model is exactly one container per app -- no
	// sidecar concept anywhere in the schema or the Quadlet generator -- so
	// `data.realContainers` (Phase 11) is always 0 or 1 rows. cpu/mem/status
	// are mapped in here rather than stored on `realContainers` itself: they
	// already exist as `latestAppMetricSample`/`app.status`, no need for a
	// second copy of that data.
	const containers = $derived(
		data.realApp
			? data.realContainers.map((c) => ({
					name: c.name,
					image: c.image,
					nets: c.nets,
					cpu:
						data.latestAppMetricSample?.cpuPct != null
							? `${data.latestAppMetricSample.cpuPct.toFixed(1)}%`
							: '—',
					mem:
						data.latestAppMetricSample?.memBytes != null
							? formatBytes(data.latestAppMetricSample.memBytes)
							: '—',
					status: app?.status ?? 'stopped'
				}))
			: (APP_CONTAINERS[appId] ?? [])
	);

	const envVars = $derived(
		(ENV_SETS[envSel] ?? []).map((e) => ({
			...e,
			shown: e.secret && !reveal ? '••••••••••••••••' : e.v
		}))
	);

	type AppMetricSample = (typeof data.appMetricHistory)[number];

	function toAppMetricSeries(history: AppMetricSample[], value: (s: AppMetricSample) => number) {
		return history.map((s) => ({ ts: s.ts, value: value(s) }));
	}

	const appCpuSeries = $derived(toAppMetricSeries(data.appMetricHistory, (s) => s.cpuPct ?? 0));
	const appMemSeries = $derived(
		toAppMetricSeries(data.appMetricHistory, (s) => (s.memBytes ?? 0) / (1024 * 1024))
	);

	const appMetricCards = $derived([
		{
			label: 'CPU',
			big:
				data.latestAppMetricSample?.cpuPct != null
					? `${data.latestAppMetricSample.cpuPct.toFixed(1)}%`
					: '—',
			color: '#3fb984',
			unit: '%',
			series: appCpuSeries
		},
		{
			label: 'Memory',
			big:
				data.latestAppMetricSample?.memBytes != null
					? formatBytes(data.latestAppMetricSample.memBytes)
					: '—',
			color: '#8b5cf6',
			unit: ' MB',
			series: appMemSeries
		}
	]);

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
		{ id: 'storage', label: 'Storage' },
		{ id: 'quadlet', label: 'Quadlet' }
	];

	// Sourced from the URL (not local-only state) so a refresh, or a link
	// shared straight to one tab, lands on that tab instead of always
	// resetting to Overview — an invalid/missing `?tab=` falls back to it.
	const activeTab = $derived(
		(tabs.some((t) => t.id === page.url.searchParams.get('tab'))
			? page.url.searchParams.get('tab')
			: 'overview') as Tab
	);
	function selectTab(tabId: Tab) {
		goto(`?tab=${tabId}`, { replaceState: true, noScroll: true, keepFocus: true });
	}

	// Keeps the Containers tab live while it's open (task's explicit "real
	// time" ask) -- same setInterval(invalidateAll)/clearInterval shape the
	// Hosts page already uses for its own live-refresh, gated here on the
	// tab being active rather than a form step.
	$effect(() => {
		if (activeTab !== 'containers' || !data.realApp) return;
		const interval = setInterval(() => invalidateAll(), 5000);
		return () => clearInterval(interval);
	});

	// A build still queued/building, or a deployment mid-rollout, means
	// something is actively changing server-side that this page's one-shot
	// `load()` won't ever pick up on its own -- without this, a
	// webhook-triggered auto-deploy's own progress (including its
	// completion) is invisible until the user manually reloads, which reads
	// as the deploy being permanently "stuck" on whatever build-log line
	// happened to be last (e.g. "Auto-deploying to <host>...") even once
	// it's actually finished.
	const hasInFlightActivity = $derived(
		Boolean(
			data.builds?.some((b) => b.status === 'queued' || b.status === 'building') ||
			data.deployments?.some(
				(d) =>
					d.status === 'starting_new' ||
					d.status === 'health_checking' ||
					d.status === 'flipping_proxy' ||
					d.status === 'stopping_old'
			)
		)
	);

	// Same setInterval(invalidateAll)/clearInterval shape as the Containers
	// tab above, gated on there being something worth polling for so an app
	// with no active build/deploy doesn't poll forever for nothing.
	$effect(() => {
		if (activeTab !== 'deployments' || !data.realApp || !hasInFlightActivity) return;
		const interval = setInterval(() => invalidateAll(), 5000);
		return () => clearInterval(interval);
	});

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
					onclick={deployNow}
					disabled={deploying || !data.realApp}
					class="flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-[9px] text-[13px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
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
					{deploying ? 'Deploying…' : 'Redeploy'}
				</button>
			</div>

			<!-- Tab bar -->
			<div class="flex gap-0.5 mt-[18px] border-b border-b-[var(--line)]">
				{#each tabs as t (t.id)}
					<button onclick={() => selectTab(t.id)} class={tabCls(t.id)}>{t.label}</button>
				{/each}
			</div>
		</div>

		<!-- Tab content -->
		<div class="px-7 py-[22px]">
			<!-- Overview -->
			{#if activeTab === 'overview'}
				<div class="grid grid-cols-4 gap-[13px] mb-4">
					{#each [{ label: 'CPU', value: app.cpu + '%' }, { label: 'Memory', value: app.mem }, { label: 'Uptime', value: data.realApp ? (currentDeploymentUptime ?? '—') : app.status === 'running' ? '4d 6h' : '—' }, { label: 'Restarts (24h)', value: app.status === 'failed' ? '7' : '0' }] as m (m.label)}
						<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] px-4 py-[15px]">
							<div class="text-[12px] text-[var(--tx-2)]">{m.label}</div>
							<div class="font-mono-jb font-semibold text-[20px] mt-[7px] text-[var(--tx)]">
								{m.value}
							</div>
						</div>
					{/each}
				</div>

				<div class="grid grid-cols-[1.6fr_1fr] gap-4">
					<!-- CPU/mem charts -->
					<div class="grid grid-cols-2 gap-[13px]">
						{#each appMetricCards as m (m.label)}
							<div
								class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] px-[18px] py-4 overflow-hidden"
								style:color={m.color}
							>
								<div class="flex items-center justify-between">
									<span class="text-[13px] font-bold text-[var(--tx)]">{m.label} · last hour</span>
									<span class="font-mono-jb text-[13px] font-semibold" style:color={m.color}
										>{m.big}</span
									>
								</div>
								<div class="h-[92px] mt-3 -mx-[18px] w-[calc(100%+36px)]">
									<MetricChart series={m.series} color={m.color} label={m.label} unit={m.unit} />
								</div>
								<div
									class="flex justify-between font-mono-jb text-[10.5px] text-[var(--tx-3)] mt-1"
								>
									<span>60m ago</span><span>now</span>
								</div>
							</div>
						{/each}
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
					{#if !data.realApp}
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
					{/if}
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
				{#if data.realApp && !data.realApp.repoId}
					<div
						class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[18px] mb-[14px]"
					>
						<div class="text-[13px] font-bold mb-1">No repository connected</div>
						<div class="text-[12.5px] text-[var(--tx-2)] mb-[12px]">
							{app.name} has no source to build from yet — pick one of this guild's connected repos.
							{#if data.connectableRepos.length === 0}
								Connect GitHub on the <a
									href="/{guildId}/deploy/sources"
									class="text-[var(--grn-2)]">Sources page</a
								> first.
							{/if}
						</div>
						{#if data.connectableRepos.length > 0}
							<div class="flex items-center gap-[10px]">
								<select
									bind:value={selectedConnectRepoId}
									class="flex-1 bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[9px] font-mono-jb text-[13px] text-[var(--tx)]"
								>
									<option value="">Select a repository…</option>
									{#each data.connectableRepos as r (r.id)}
										<option value={r.id}>{r.fullName}</option>
									{/each}
								</select>
								<button
									onclick={connectRepo}
									disabled={connectingRepo || !selectedConnectRepoId}
									class="bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-[9px] text-[13px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
									>{connectingRepo ? 'Connecting…' : 'Connect'}</button
								>
							</div>
						{/if}
					</div>
				{:else if data.realApp}
					<div class="flex items-center justify-end mb-[14px]">
						<button
							onclick={buildNow}
							disabled={buildingNow}
							class="flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-[9px] text-[13px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
							>{buildingNow ? 'Queuing…' : 'Build now'}</button
						>
					</div>

					{#if data.deploymentAttempts.length > 0}
						<div class="text-[13px] font-bold mb-[10px]">Deployments</div>
						<div
							class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden"
						>
							{#each data.deploymentAttempts as row (row.buildId)}
								{@const isCurrent = row.deploymentId === currentDeploymentId}
								{@const statusLabel = row.deploymentStatus ?? row.buildStatus}
								{@const { color, pulse } = row.deploymentStatus
									? deploymentStatusColor(row.deploymentStatus)
									: row.buildStatus === 'failed'
										? { color: '#f0836b', pulse: false }
										: {
												color: 'var(--tx-3)',
												pulse: row.buildStatus === 'queued' || row.buildStatus === 'building'
											}}
								<div class="border-b border-b-[var(--line)] last:border-b-0">
									<div
										onclick={() => toggleDeploymentLogs(row.buildId)}
										onkeydown={(e) => {
											if (e.key === 'Enter' || e.key === ' ') {
												e.preventDefault();
												toggleDeploymentLogs(row.buildId);
											}
										}}
										role="button"
										tabindex="0"
										class="flex items-center gap-[14px] px-[18px] py-[12px] cursor-pointer hover:bg-white/[0.02]"
									>
										<div
											class="size-[10px] rounded-full shrink-0"
											style:background={color}
											class:animate-pulse={pulse}
										></div>
										<div class="flex-1 min-w-0">
											<div class="flex items-center gap-[8px]">
												<span class="font-mono-jb text-[12.5px] text-[var(--tx)] font-semibold"
													>{row.commitSha.slice(0, 7)} · {row.branch ?? 'external image'}</span
												>
												{#if isCurrent}
													<span
														class="font-mono-jb text-[10px] font-semibold px-[7px] py-[2px] rounded-[6px] bg-[var(--grn-dim)] text-[var(--grn)]"
														>current</span
													>
												{/if}
											</div>
											<div class="text-[11px] mt-[2px]" style:color>
												{statusLabel} · triggered by {row.deploymentTriggeredBy ??
													row.buildTriggeredBy ??
													'unknown'}
											</div>
										</div>
										<div class="text-[11px] text-[var(--tx-3)] text-right shrink-0">
											{#if isCurrent && currentDeploymentUptime}
												<div class="font-mono-jb text-[var(--grn-2)]">
													up {currentDeploymentUptime}
												</div>
											{/if}
											{#if row.deploymentFinishedAt}
												{new Date(row.deploymentFinishedAt).toLocaleString()}
											{:else if row.deploymentStartedAt}
												{new Date(row.deploymentStartedAt).toLocaleString()}
											{:else if row.buildFinishedAt}
												{new Date(row.buildFinishedAt).toLocaleString()}
											{:else if row.buildStartedAt}
												{new Date(row.buildStartedAt).toLocaleString()}
											{:else}
												queued
											{/if}
										</div>
										{#if row.deploymentStatus === 'stopped'}
											<button
												onclick={(e) => {
													e.stopPropagation();
													if (row.deploymentId) rollback(row.deploymentId);
												}}
												disabled={rollingBackId === row.deploymentId}
												class="shrink-0 bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-[7px] text-[11.5px] font-semibold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
												>{rollingBackId === row.deploymentId
													? 'Rolling back…'
													: 'Rollback to this deploy'}</button
											>
										{/if}
										{#if isCurrent}
											<button
												onclick={(e) => {
													e.stopPropagation();
													if (row.deploymentId) requestStopDeployment(row.deploymentId);
												}}
												disabled={stoppingId === row.deploymentId}
												class="shrink-0 bg-[var(--card-2)] border border-[var(--line)] text-[#f0836b] rounded-[8px] px-3 py-[7px] text-[11.5px] font-semibold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
												>{stoppingId === row.deploymentId ? 'Stopping…' : 'Stop'}</button
											>
										{/if}
									</div>
									{#if expandedDeploymentId === row.buildId}
										<div class="bg-[#080c09] border-t border-t-[var(--line)] overflow-hidden">
											<div
												class="flex items-center justify-between gap-[10px] px-4 py-[10px] border-b border-b-[var(--line)] bg-[var(--card)]"
											>
												<span class="text-[12.5px] font-semibold"
													>Live log · {row.commitSha.slice(0, 7)} · {statusLabel}</span
												>
												{#if row.deploymentId}
													<div
														class="flex gap-[3px] bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] p-[3px] w-fit shrink-0"
													>
														{#each ['runtime', 'build'] as const as logTab (logTab)}
															<button
																onclick={() => selectLogTab(logTab)}
																class="px-[10px] py-[4px] text-[11.5px] rounded-[6px] cursor-pointer {expandedLogTab ===
																logTab
																	? 'bg-[var(--card)] text-[var(--tx)]'
																	: 'text-[var(--tx-2)]'}"
																>{logTab === 'runtime' ? 'Runtime' : 'Build'}</button
															>
														{/each}
													</div>
												{/if}
											</div>
											{#if !row.deploymentId}
												<BuildLogViewer logsUrl={`/api/v1/builds/${row.buildId}/logs`} />
											{:else if expandedLogTab === 'runtime'}
												<BuildLogViewer
													logsUrl={`/api/v1/deployments/${row.deploymentId}/logs/stream`}
												/>
											{:else}
												<BuildLogViewer logsUrl={`/api/v1/builds/${row.buildId}/logs`} />
											{/if}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{:else}
						<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
							No builds yet — click "Build now" to trigger the first one.
						</div>
					{/if}
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
					{#if data.realApp}
						<BuildLogViewer logsUrl={`/api/v1/apps/${data.realApp.id}/logs/stream`} />
					{:else}
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
					{/if}
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
					{#each ['production', 'staging'] as const as tab (tab)}
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
				{#if data.realApp}
					<div class="grid grid-cols-2 gap-4">
						<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[18px]">
							<div class="text-[14px] font-bold mb-1">Domains</div>
							<div class="text-[12.5px] text-[var(--tx-2)] mb-[14px]">
								Every hostname routed to this app.
							</div>
							<div class="flex flex-col gap-2 mb-[14px]">
								{#each data.domains as d (d.id)}
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
										<span class="font-mono-jb text-[13px] text-[var(--tx)]">{d.hostname}</span>

										{#if d.isDefaultSubdomain}
											<span
												class="ml-auto text-[11px] font-semibold px-2 py-[3px] rounded-[6px] bg-[var(--grn-dim)] text-[var(--grn)]"
												>default</span
											>
										{:else if d.verifiedAt}
											<span
												class="ml-auto flex items-center gap-[5px] text-[11.5px] text-[var(--grn)]"
											>
												<svg
													width="13"
													height="13"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2.3"><path d="M20 6L9 17l-5-5" /></svg
												>
												Verified
											</span>
										{:else}
											<span class="ml-auto text-[11.5px] text-[var(--amber)]">Pending DNS</span>
											<button
												onclick={() => verifyCustomDomain(d.id)}
												disabled={verifyingDomainId === d.id}
												class="bg-[var(--card)] border border-[var(--line)] text-[var(--tx)] rounded-[7px] px-[10px] py-[5px] text-[11.5px] font-semibold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
												>{verifyingDomainId === d.id ? 'Checking…' : 'Verify'}</button
											>
										{/if}

										{#if !d.isDefaultSubdomain}
											<button
												onclick={() => removeCustomDomain(d.id)}
												disabled={removingDomainId === d.id}
												aria-label="Remove domain"
												class="text-[var(--tx-3)] cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
											>
												<svg
													width="14"
													height="14"
													viewBox="0 0 24 24"
													fill="none"
													stroke="currentColor"
													stroke-width="2"><path d="M18 6L6 18M6 6l12 12" /></svg
												>
											</button>
										{/if}
									</div>
								{/each}
							</div>

							<div class="flex gap-2">
								<input
									bind:value={domainInput}
									placeholder="app.yourcompany.com"
									class="flex-1 bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-mono-jb"
								/>
								<button
									onclick={addCustomDomain}
									disabled={addingDomain || !domainInput.trim()}
									class="bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-2 text-[12.5px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
									>{addingDomain ? 'Adding…' : 'Add domain'}</button
								>
							</div>
							{#if data.domains.some((d) => !d.isDefaultSubdomain && !d.verifiedAt)}
								<div class="text-[11.5px] text-[var(--tx-3)] mt-2">
									Add a CNAME record pointing to <span class="font-mono-jb text-[var(--grn-2)]"
										>{realAppDomain}</span
									> at your DNS provider, then click Verify.
								</div>
							{/if}
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
									class="text-[var(--grn-2)]">{realAppDomain}</span
								> {'{'}
  reverse_proxy <span class="text-[var(--amber)]">127.0.0.1:{port}</span>
  encode zstd gzip
}</pre>
						</div>
					</div>
				{:else}
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
}</pre>
						</div>
					</div>
				{/if}
			{/if}

			<!-- Storage -->
			{#if activeTab === 'storage'}
				{#if data.realApp}
					<div class="flex items-center justify-between mb-[14px]">
						<div>
							<div class="text-[15px] font-bold">Storage</div>
							<div class="text-[12.5px] text-[var(--tx-2)]">
								Named Podman volumes mounted into <span class="text-[var(--tx)]">{app.name}</span> — persists
								data across redeploys.
							</div>
						</div>
					</div>

					<div
						class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden mb-[18px]"
					>
						{#each data.volumes as v (v.id)}
							<div
								class="grid grid-cols-[1.3fr_1.6fr_100px_auto] gap-[14px] items-center px-[18px] py-[13px] border-b border-b-[var(--line)] last:border-b-0"
							>
								<span class="font-mono-jb text-[13px] text-[var(--tx)] font-semibold">{v.name}</span
								>
								<span class="font-mono-jb text-[12px] text-[var(--tx-2)]">{v.mountPath}</span>
								<span class="font-mono-jb text-[12px] text-[var(--tx-2)]"
									>{formatBytes(v.sizeBytes)}</span
								>
								<button
									onclick={() => removeVolume(v.id)}
									disabled={removingVolumeId === v.id}
									aria-label="Detach volume"
									class="justify-self-end text-[var(--tx-3)] cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
								>
									<svg
										width="14"
										height="14"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="2"><path d="M18 6L6 18M6 6l12 12" /></svg
									>
								</button>
							</div>
						{/each}
						{#if data.volumes.length === 0}
							<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
								No volumes attached — data written inside the container is lost on redeploy.
							</div>
						{/if}
					</div>

					<div
						class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[18px] max-w-[520px]"
					>
						<div class="text-[14px] font-bold mb-[14px]">Attach a volume</div>
						<div class="flex flex-col gap-[10px]">
							<div>
								<label
									for="vol-name"
									class="block text-[11.5px] font-semibold text-[var(--tx-3)] mb-[5px] tracking-[.03em]"
									>NAME</label
								>
								<input
									id="vol-name"
									bind:value={volName}
									placeholder="data"
									class="w-full bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-mono-jb"
								/>
							</div>
							<div>
								<label
									for="vol-mount-path"
									class="block text-[11.5px] font-semibold text-[var(--tx-3)] mb-[5px] tracking-[.03em]"
									>MOUNT PATH</label
								>
								<input
									id="vol-mount-path"
									bind:value={volMountPath}
									placeholder="/data"
									class="w-full bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-3 py-2 text-[12.5px] font-mono-jb"
								/>
							</div>
							<button
								onclick={attachVolume}
								disabled={attachingVolume || !volName.trim() || !volMountPath.trim()}
								class="bg-[var(--grn)] text-[#07130c] rounded-[8px] px-[14px] py-2 text-[12.5px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
								>{attachingVolume ? 'Attaching…' : 'Attach volume'}</button
							>
						</div>
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[10px]">
							Podman-managed, local to the app's host. Takes effect on the next deploy.
						</div>
					</div>
				{:else}
					<div class="p-[30px] text-center text-[var(--tx-3)] text-[13px]">
						Storage isn't available for this demo app.
					</div>
				{/if}
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

<AlertDialog.Root bind:open={stopDialogOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Stop this deployment?</AlertDialog.Title>
			<AlertDialog.Description>
				The app will be unreachable until a new deployment is started or this one is redeployed.
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action onclick={confirmStopDeployment}>Stop deployment</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
