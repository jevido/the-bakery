<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS, statusMeta, quadletContent, APP_CONTAINERS, APP_DEPLOYMENTS, LOG_LINES, ENV_SETS } from '$lib/data/bakery';
	import StatusDot from '$lib/components/bakery/StatusDot.svelte';
	import AppIcon from '$lib/components/bakery/AppIcon.svelte';
	import { goto } from '$app/navigation';

	const guildId = $derived(page.params.guild ?? '');
	const appId = $derived(page.params.appId ?? '');
	const guild = $derived(GUILDS[guildId]);
	const app = $derived(guild?.apps.find((a) => a.id === appId));
	const meta = $derived(app ? statusMeta(app.status) : null);

	type Tab = 'overview' | 'containers' | 'network' | 'deployments' | 'logs' | 'env' | 'domains' | 'quadlet';
	let activeTab = $state<Tab>('overview');
	let envSel = $state<'production' | 'staging'>('production');
	let reveal = $state(false);

	const containers = $derived(APP_CONTAINERS[appId] ?? []);
	const envVars = $derived((ENV_SETS[envSel] ?? []).map(e => ({
		...e,
		shown: e.secret && !reveal ? '••••••••••••••••' : e.v,
	})));

	// Spark bars for CPU chart
	const sparkbars = Array.from({ length: 42 }, (_, i) => {
		const h = Math.min(100, 25 + Math.round(35 * Math.abs(Math.sin(i * 0.7)) + 20 * Math.abs(Math.cos(i * 0.4))));
		return { h, recent: i > 36 };
	});

	function tabStyle(t: Tab) {
		const on = 'padding:10px 14px;font-size:13.5px;font-weight:600;color:var(--grn);cursor:pointer;border-bottom:2px solid var(--grn);background:none;border-top:none;border-left:none;border-right:none;margin-bottom:-1px;';
		const off = 'padding:10px 14px;font-size:13.5px;font-weight:600;color:var(--tx-2);cursor:pointer;border-bottom:2px solid transparent;background:none;border-top:none;border-left:none;border-right:none;margin-bottom:-1px;';
		return t === activeTab ? on : off;
	}

	const tabs: Array<{id: Tab; label: string}> = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'containers', label: 'Containers' },
		{ id: 'network', label: 'Network' },
		{ id: 'deployments', label: 'Deployments' },
		{ id: 'logs', label: 'Logs' },
		{ id: 'env', label: 'Environment' },
		{ id: 'domains', label: 'Domains & Proxy' },
		{ id: 'quadlet', label: 'Quadlet' },
	];

	const port = $derived(app?.port === '—' ? '3000' : (app?.port ?? '3000'));
</script>

{#if !app}
	<div style="padding:40px;color:var(--tx-2);">Project not found.</div>
{:else}
<div>
	<!-- App header -->
	<div style="padding: 20px 28px 0;">
		<button
			onclick={() => goto(`/${guildId}/deploy/projects`)}
			style="display:inline-flex;align-items:center;gap:6px;font-size:12.5px;color:var(--tx-2);cursor:pointer;margin-bottom:14px;background:none;border:none;padding:0;"
		>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
				<path d="M15 18l-6-6 6-6"/>
			</svg>
			All projects
		</button>

		<div style="display:flex;align-items:center;gap:15px;">
			<AppIcon initial={app.initial} status={app.status} size={52} radius={14} />
			<div style="flex:1;min-width:0;">
				<div style="display:flex;align-items:center;gap:11px;">
					<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;">{app.name}</div>
					<div style="display:flex;align-items:center;gap:7px;padding:4px 10px;border-radius:20px;background:{meta?.bg};">
						<StatusDot status={app.status} />
						<span style="font-size:12px;font-weight:600;color:{meta?.c};">{meta?.label}</span>
					</div>
				</div>
				<div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx-3);margin-top:3px;">{app.type} · {app.host}</div>
			</div>

			{#if app.domain !== '— internal —'}
				<a href="https://{app.domain}" target="_blank" rel="noreferrer" style="display:flex;align-items:center;gap:6px;font-size:12.5px;color:var(--grn-2);text-decoration:none;font-weight:600;padding:8px 12px;border:1px solid var(--grn-line);border-radius:8px;">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M7 17L17 7M9 7h8v8"/>
					</svg>
					{app.domain}
				</a>
			{/if}

			<button style="display:flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border:none;border-radius:8px;padding:9px 14px;font-size:13px;font-weight:700;cursor:pointer;">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
					<path d="M21 12a9 9 0 11-3-6.7L21 8"/>
					<path d="M21 3v5h-5"/>
				</svg>
				Redeploy
			</button>
		</div>

		<!-- Tab bar -->
		<div style="display:flex;gap:2px;margin-top:18px;border-bottom:1px solid var(--line);">
			{#each tabs as t}
				<button onclick={() => activeTab = t.id} style={tabStyle(t.id)}>{t.label}</button>
			{/each}
		</div>
	</div>

	<!-- Tab content -->
	<div style="padding: 22px 28px;">

		<!-- Overview -->
		{#if activeTab === 'overview'}
			<div style="display:grid;grid-template-columns:repeat(4,1fr);gap:13px;margin-bottom:16px;">
				{#each [
					{ label: 'CPU', value: app.cpu + '%' },
					{ label: 'Memory', value: app.mem },
					{ label: 'Uptime', value: app.status === 'running' ? '4d 6h' : '—' },
					{ label: 'Restarts (24h)', value: app.status === 'failed' ? '7' : '0' },
				] as m}
					<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:15px 16px;">
						<div style="font-size:12px;color:var(--tx-2);">{m.label}</div>
						<div style="font-family:'JetBrains Mono',monospace;font-weight:600;font-size:20px;margin-top:7px;color:var(--tx);">{m.value}</div>
					</div>
				{/each}
			</div>

			<div style="display:grid;grid-template-columns:1.6fr 1fr;gap:16px;">
				<!-- CPU chart -->
				<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px;">
					<div style="font-size:13px;font-weight:700;margin-bottom:14px;">CPU & Memory · last hour</div>
					<div style="display:flex;align-items:flex-end;gap:3px;height:110px;">
						{#each sparkbars as b}
							<div style="flex:1;height:{b.h}%;background:{b.recent ? 'var(--grn)' : 'var(--grn-dim)'};border-radius:2px 2px 0 0;"></div>
						{/each}
					</div>
					<div style="display:flex;justify-content:space-between;font-family:'JetBrains Mono',monospace;font-size:10.5px;color:var(--tx-3);margin-top:8px;">
						<span>60m ago</span><span>now</span>
					</div>
				</div>

				<!-- Details -->
				<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px;">
					<div style="font-size:13px;font-weight:700;margin-bottom:12px;">Details</div>
					{#each [
						{ k: 'Host', v: app.host },
						{ k: 'Image', v: 'ghcr.io/…/' + app.name },
						{ k: 'Port', v: port },
						{ k: 'Quadlet', v: app.quadletPath.split('/').pop() },
						{ k: 'Restart policy', v: 'always' },
						{ k: 'Auto-update', v: 'registry' },
					] as d}
						<div style="display:flex;justify-content:space-between;gap:10px;padding:7px 0;border-bottom:1px solid var(--line);">
							<span style="font-size:12.5px;color:var(--tx-2);">{d.k}</span>
							<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx);text-align:right;">{d.v}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Containers -->
		{#if activeTab === 'containers'}
			<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px;">
				<div>
					<div style="font-size:15px;font-weight:700;">Containers</div>
					<div style="font-size:12.5px;color:var(--tx-2);">Every container running inside <span style="color:var(--tx);">{app.name}</span>, from one quadlet pod.</div>
				</div>
				<button style="display:flex;align-items:center;gap:7px;background:var(--card-2);border:1px solid var(--line);color:var(--tx);border-radius:8px;padding:8px 12px;font-size:12.5px;font-weight:600;cursor:pointer;">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>
					Add container
				</button>
			</div>
			<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;overflow:hidden;">
				<div style="display:grid;grid-template-columns:1.4fr 1.4fr 1fr 90px 90px;gap:14px;padding:10px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
					<div>CONTAINER</div><div>IMAGE</div><div>NETWORKS</div><div>CPU</div><div>MEM</div>
				</div>
				{#each containers as c}
					<div style="display:grid;grid-template-columns:1.4fr 1.4fr 1fr 90px 90px;gap:14px;align-items:center;padding:12px 18px;border-bottom:1px solid var(--line);">
						<div style="display:flex;align-items:center;gap:9px;min-width:0;">
							<StatusDot status={c.status} size={8} />
							<span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{c.name}</span>
						</div>
						<span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--tx-2);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{c.image}</span>
						<span style="font-size:11.5px;color:var(--tx-2);">{c.nets}</span>
						<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx);">{c.cpu}</span>
						<span style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--tx);">{c.mem}</span>
					</div>
				{/each}
				{#if containers.length === 0}
					<div style="padding:30px;text-align:center;color:var(--tx-3);font-size:13px;">No container data available.</div>
				{/if}
			</div>
		{/if}

		<!-- Network (visual placeholder) -->
		{#if activeTab === 'network'}
			<div style="background:#0c0d13;border:1px solid var(--line);border-radius:14px;padding:60px;text-align:center;color:var(--tx-3);">
				<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.2" style="margin:0 auto 12px;">
					<circle cx="12" cy="12" r="3"/><circle cx="4" cy="6" r="2"/><circle cx="20" cy="6" r="2"/>
					<circle cx="4" cy="18" r="2"/><circle cx="20" cy="18" r="2"/>
					<path d="M6 6l4 4M14 14l4 4M6 18l4-4M14 10l4-4"/>
				</svg>
				<div style="font-size:14px;font-weight:600;color:var(--tx-2);">Network topology</div>
				<div style="font-size:12.5px;margin-top:4px;">Interactive network graph for {app.name}</div>
			</div>
		{/if}

		<!-- Deployments -->
		{#if activeTab === 'deployments'}
			<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;overflow:hidden;">
				{#each APP_DEPLOYMENTS as d}
					<div style="display:grid;grid-template-columns:auto 1fr auto;gap:14px;align-items:center;padding:14px 18px;border-bottom:1px solid var(--line);">
						<div style="width:10px;height:10px;border-radius:50%;background:{d.status === 'Deployed' ? 'var(--grn)' : '#e5654b'};flex:none;"></div>
						<div>
							<div style="font-size:13.5px;color:var(--tx);font-weight:600;">{d.msg}</div>
							<div style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--tx-3);margin-top:2px;">{d.sha} · {d.branch} · by {d.by}</div>
						</div>
						<div style="text-align:right;">
							<div style="font-size:12px;color:{d.status === 'Deployed' ? '#52cc96' : '#f0836b'};font-weight:600;">{d.status}</div>
							<div style="font-size:11px;color:var(--tx-3);margin-top:2px;">{d.time}</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Logs -->
		{#if activeTab === 'logs'}
			<div style="background:#080c09;border:1px solid var(--line);border-radius:12px;overflow:hidden;">
				<div style="display:flex;align-items:center;gap:10px;padding:10px 16px;border-bottom:1px solid var(--line);background:var(--card);">
					<div style="display:flex;align-items:center;gap:7px;">
						<div style="width:8px;height:8px;border-radius:50%;background:var(--grn);animation:bk-pulse 2s infinite;"></div>
						<span style="font-size:12.5px;font-weight:600;">Live</span>
					</div>
					<span style="font-family:'JetBrains Mono',monospace;font-size:11.5px;color:var(--tx-3);">journalctl --user -u {app.unit} -f</span>
					<div style="flex:1"></div>
					<span style="font-size:11.5px;color:var(--tx-3);">stdout · stderr</span>
				</div>
				<div style="padding:14px 16px;font-family:'JetBrains Mono',monospace;font-size:12px;line-height:1.7;max-height:360px;overflow-y:auto;">
					{#each LOG_LINES as l}
						<div style="display:flex;gap:12px;">
							<span style="color:var(--tx-3);flex:none;">{l.t}</span>
							<span style="color:{l.color};flex:none;width:52px;">{l.lvl}</span>
							<span style="color:var(--tx);white-space:pre-wrap;">{l.msg}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Environment -->
		{#if activeTab === 'env'}
			<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px;">
				<div>
					<div style="font-size:15px;font-weight:700;">Environment</div>
					<div style="font-size:12.5px;color:var(--tx-2);">Scoped to <span style="color:var(--tx);">{app.name}</span> only · written to <span style="font-family:'JetBrains Mono',monospace;color:var(--grn-2);">/etc/bakery/{app.unit}.env</span></div>
				</div>
				<div style="display:flex;gap:8px;">
					<button style="background:var(--card-2);border:1px solid var(--line);color:var(--tx);border-radius:8px;padding:8px 12px;font-size:12.5px;font-weight:600;cursor:pointer;">Import .env</button>
					<button onclick={() => reveal = !reveal} style="background:var(--card-2);border:1px solid var(--line);color:var(--tx);border-radius:8px;padding:8px 12px;font-size:12.5px;font-weight:600;cursor:pointer;">{reveal ? 'Hide values' : 'Reveal values'}</button>
				</div>
			</div>
			<!-- Env filter tabs -->
			<div style="display:flex;gap:3px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:3px;width:fit-content;margin-bottom:14px;">
				{#each (['production', 'staging'] as const) as tab}
					<button
						onclick={() => envSel = tab}
						style="padding:5px 11px;font-size:12.5px;border-radius:6px;cursor:pointer;background:{envSel === tab ? 'var(--card-2)' : 'transparent'};color:{envSel === tab ? 'var(--tx)' : 'var(--tx-2)'};border:none;"
					>{tab.charAt(0).toUpperCase() + tab.slice(1)}</button>
				{/each}
			</div>
			<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;overflow:hidden;">
				{#each envVars as e}
					<div style="display:grid;grid-template-columns:1fr 1.6fr auto;gap:14px;align-items:center;padding:11px 18px;border-bottom:1px solid var(--line);">
						<span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:var(--grn-2);font-weight:500;">{e.key}</span>
						<span style="font-family:'JetBrains Mono',monospace;font-size:12.5px;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{e.shown}</span>
						<span style="font-family:'JetBrains Mono',monospace;font-size:10.5px;font-weight:600;padding:3px 8px;border-radius:6px;background:{e.secret ? 'rgba(224,168,62,.14)' : 'rgba(255,255,255,.06)'};color:{e.secret ? '#efc060' : 'var(--tx-3)'};">{e.secret ? 'secret' : 'variable'}</span>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Domains & Proxy -->
		{#if activeTab === 'domains'}
			<div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;">
				<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px;">
					<div style="font-size:14px;font-weight:700;margin-bottom:4px;">Domain</div>
					<div style="font-size:12.5px;color:var(--tx-2);margin-bottom:14px;">Public address routed to this app.</div>
					<div style="display:flex;align-items:center;gap:10px;background:var(--card-2);border:1px solid var(--line);border-radius:9px;padding:11px 14px;">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="1.8">
							<circle cx="12" cy="12" r="9"/>
							<path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18"/>
						</svg>
						<span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx);">{app.domain}</span>
						{#if app.domain !== '— internal —'}
							<span style="margin-left:auto;display:flex;align-items:center;gap:5px;font-size:11.5px;color:var(--grn);">
								<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M20 6L9 17l-5-5"/></svg>
								DNS ok
							</span>
						{/if}
					</div>
				</div>

				<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px;">
					<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;">
						<div style="font-size:14px;font-weight:700;">Caddy reverse proxy</div>
						<div style="display:flex;align-items:center;gap:7px;font-size:11.5px;color:var(--grn);">
							<div style="width:7px;height:7px;border-radius:50%;background:var(--grn);"></div>Active · auto HTTPS
						</div>
					</div>
					<div style="font-size:12.5px;color:var(--tx-2);margin-bottom:14px;">Bakery manages a single Caddy instance per host.</div>
					<div style="font-family:'JetBrains Mono',monospace;font-size:12px;line-height:1.8;background:#080c09;border:1px solid var(--line);border-radius:9px;padding:13px 15px;color:var(--tx-2);">
						<span style="color:var(--grn-2);">{app.domain}</span> {'{'}
						<br>&nbsp;&nbsp;reverse_proxy <span style="color:var(--amber);">127.0.0.1:{port}</span>
						<br>&nbsp;&nbsp;encode zstd gzip
						<br>{'}'}
					</div>
				</div>
			</div>
		{/if}

		<!-- Quadlet -->
		{#if activeTab === 'quadlet'}
			<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;">
				<div>
					<div style="font-size:15px;font-weight:700;">Quadlet</div>
					<div style="font-size:12.5px;color:var(--tx-2);">The Podman Quadlet that defines this app, sourced from <span style="font-family:'JetBrains Mono',monospace;color:var(--grn-2);">{app.quadletPath}</span>.</div>
				</div>
				<span style="font-family:'JetBrains Mono',monospace;font-size:11px;padding:5px 10px;border-radius:6px;background:var(--grn-dim);color:var(--grn);">.container</span>
			</div>
			<pre style="font-family:'JetBrains Mono',monospace;font-size:12.5px;line-height:1.85;background:#080c09;border:1px solid var(--line);border-radius:12px;padding:18px 20px;overflow-x:auto;color:var(--tx);margin:0;">{quadletContent(app)}</pre>
		{/if}
	</div>
</div>
{/if}

<style>
	button {
		all: unset;
		box-sizing: border-box;
	}
	button:not([style*="cursor:pointer"]) {
		cursor: pointer;
	}
</style>
