<script lang="ts">
	import { GUILDS, REPOS } from '$lib/data/bakery';

	let {
		open = $bindable(false),
		guildId,
	}: { open: boolean; guildId: string } = $props();

	const guild = $derived(GUILDS[guildId]);
	const hosts = $derived(guild?.hosts ?? []);

	// ---- wizard state ----
	let step = $state(0);
	let wRepo = $state<string | null>(null);
	let wHost = $state<string | null>(null);
	let wQuadlet = $state(0);
	let wDomain = $state('');
	let wProxy = $state(true);
	let wTls = $state(true);
	let wPort = $state('3000');
	let wEnv = $state([
		{ key: 'NODE_ENV', value: 'production', secret: false },
		{ key: 'PORT', value: '3000', secret: false },
		{ key: 'DATABASE_URL', value: 'postgres://crust:•••@proofing-db:5432/app', secret: true },
	]);
	let deployPhase = $state<'idle' | 'running' | 'done'>('idle');
	let deployLog = $state<Array<{ text: string; color: string }>>([]);
	let repoSearch = $state('');

	$effect(() => {
		if (open) {
			step = 0; wRepo = null; wHost = hosts[0]?.name ?? null;
			wQuadlet = 0; wDomain = ''; wProxy = true; wTls = true; wPort = '3000';
			wEnv = [
				{ key: 'NODE_ENV', value: 'production', secret: false },
				{ key: 'PORT', value: '3000', secret: false },
				{ key: 'DATABASE_URL', value: 'postgres://crust:•••@proofing-db:5432/app', secret: true },
			];
			deployPhase = 'idle'; deployLog = []; repoSearch = '';
		}
	});

	const appName = $derived(wRepo ? wRepo.split('/').pop()! : 'new-app');

	const quadletOptions = $derived([
		{ path: `deploy/${appName}.container`, note: 'Podman Quadlet · single container', tag: '.container' },
		{ path: `deploy/${appName}.pod`, note: 'Podman Quadlet · multi-container pod', tag: '.pod' },
		{ path: '—', note: 'Generate a quadlet for me from the Dockerfile', tag: 'auto' },
	]);

	const quadletPreview = $derived(`[Unit]
Description=${appName}
After=network-online.target

[Container]
Image=ghcr.io/${wRepo ?? 'org/my-app'}:latest
PublishPort=${wPort}:${wPort}
EnvironmentFile=/etc/bakery/${appName}.env
AutoUpdate=registry

[Service]
Restart=always
TimeoutStartSec=90

[Install]
WantedBy=default.target`);

	const filteredRepos = $derived(
		REPOS.filter((r) => r.name.toLowerCase().includes(repoSearch.toLowerCase()))
	);

	const canContinue = $derived(
		step === 0 ? !!wRepo :
		step === 1 ? !!wHost :
		step === 3 ? wDomain.trim().length > 0 :
		true
	);

	const wSteps = [
		{ num: '1', label: 'Source' },
		{ num: '2', label: 'Host' },
		{ num: '3', label: 'Quadlet' },
		{ num: '4', label: 'Domain & Proxy' },
		{ num: '5', label: 'Environment' },
		{ num: '6', label: 'Deploy' },
	];

	const stepHeads = [
		{ title: 'Choose a repository', sub: 'Select the GitHub repo to build and deploy.' },
		{ title: 'Pick a host', sub: 'Which machine should run this app?' },
		{ title: 'Quadlet file', sub: 'Select the Podman Quadlet definition for this app.' },
		{ title: 'Domain & proxy', sub: 'Set up a public domain and Caddy reverse proxy.' },
		{ title: 'Environment', sub: 'Inject variables via the quadlet\'s EnvironmentFile.' },
		{ title: 'Review & deploy', sub: 'Everything looks good — bake it.' },
	];

	function close() { open = false; }
	function handleBackdrop(e: MouseEvent) { if (e.target === e.currentTarget) close(); }
	function handleKey(e: KeyboardEvent) { if (e.key === 'Escape') close(); }
	function next() { if (canContinue && step < 5) step++; }
	function back() { if (step > 0) step--; }

	function addEnv() {
		wEnv = [...wEnv, { key: '', value: '', secret: false }];
	}
	function removeEnv(i: number) {
		wEnv = wEnv.filter((_, j) => j !== i);
	}
	function toggleSecret(i: number) {
		wEnv = wEnv.map((e, j) => j === i ? { ...e, secret: !e.secret } : e);
	}
	function updateEnvKey(i: number, val: string) {
		wEnv = wEnv.map((e, j) => j === i ? { ...e, key: val } : e);
	}
	function updateEnvVal(i: number, val: string) {
		wEnv = wEnv.map((e, j) => j === i ? { ...e, value: val } : e);
	}

	const LOG_STEPS = $derived([
		{ text: '→ Connecting to oven-01…', color: 'var(--tx-2)' },
		{ text: '✓ SSH tunnel established', color: 'var(--grn)' },
		{ text: `→ Writing ~/.config/containers/systemd/${appName}.container`, color: 'var(--tx-2)' },
		{ text: '✓ Quadlet file installed', color: 'var(--grn)' },
		{ text: '→ Running systemctl --user daemon-reload', color: 'var(--tx-2)' },
		{ text: '✓ systemd unit registered', color: 'var(--grn)' },
		{ text: `→ Pulling ghcr.io/${wRepo ?? 'org/app'}:latest`, color: 'var(--tx-2)' },
		{ text: '✓ Image pulled (142 MB)', color: 'var(--grn)' },
		{ text: `→ Starting ${appName}.service`, color: 'var(--tx-2)' },
		{ text: '✓ Container started · PID 48291', color: 'var(--grn)' },
		{ text: '→ Configuring Caddy reverse proxy…', color: 'var(--tx-2)' },
		{ text: `✓ TLS certificate provisioned for ${wDomain || 'your domain'}`, color: 'var(--grn)' },
	]);

	async function deploy() {
		if (deployPhase !== 'idle') return;
		deployPhase = 'running';
		deployLog = [];
		for (const line of LOG_STEPS) {
			await new Promise((r) => setTimeout(r, 420));
			deployLog = [...deployLog, line];
		}
		deployPhase = 'done';
	}

	function stepNumStyle(i: number) {
		if (i < step) return 'width:22px;height:22px;border-radius:50%;background:var(--grn);display:flex;align-items:center;justify-content:center;flex:none;';
		if (i === step) return 'width:22px;height:22px;border-radius:50%;background:var(--grn);display:flex;align-items:center;justify-content:center;flex:none;';
		return 'width:22px;height:22px;border-radius:50%;background:rgba(255,255,255,.06);display:flex;align-items:center;justify-content:center;flex:none;';
	}
	function stepLabelStyle(i: number) {
		if (i === step) return 'font-size:13px;font-weight:700;color:var(--tx);';
		if (i < step) return 'font-size:13px;font-weight:600;color:var(--grn-2);';
		return 'font-size:13px;color:var(--tx-3);';
	}
	function stepNumColor(i: number) {
		if (i < step) return 'color:#07130c;';
		if (i === step) return 'color:#07130c;';
		return 'color:var(--tx-3);';
	}

	const reviewRows = $derived([
		{ k: 'Repository', v: wRepo ?? '—' },
		{ k: 'Host', v: wHost ?? '—' },
		{ k: 'Quadlet', v: quadletOptions[wQuadlet]?.path ?? '—' },
		{ k: 'Domain', v: wDomain || 'internal only' },
		{ k: 'Port', v: wProxy ? wPort : '—' },
		{ k: 'TLS', v: wProxy && wTls ? 'Automatic (Let\'s Encrypt)' : wProxy ? 'off' : '—' },
		{ k: 'Env vars', v: `${wEnv.length} variables` },
	]);

	const inputStyle = 'background:var(--card-2);border:1px solid var(--line);border-radius:8px;padding:10px 12px;font-family:"JetBrains Mono",monospace;font-size:13px;color:var(--tx);width:100%;box-sizing:border-box;';
</script>

<svelte:window onkeydown={handleKey} />

{#if open}
	<div
		onclick={handleBackdrop}
		role="presentation"
		style="position:fixed;inset:0;z-index:50;background:rgba(6,9,7,.72);backdrop-filter:blur(6px);display:flex;align-items:center;justify-content:center;padding:28px;"
	>
		<div style="width:960px;max-width:100%;height:640px;max-height:92vh;background:var(--panel);border:1px solid var(--line-2);border-radius:18px;box-shadow:0 30px 80px rgba(0,0,0,.5);display:flex;overflow:hidden;animation:bk-fadeup .22s ease;">

			<!-- Left step rail -->
			<div style="width:238px;flex:none;background:var(--sidebar);border-right:1px solid var(--line);padding:22px 18px;display:flex;flex-direction:column;">
				<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:17px;margin-bottom:3px;">New app</div>
				<div style="font-size:12px;color:var(--tx-2);margin-bottom:22px;">Repo → live in six steps.</div>

				{#each wSteps as s, i (i)}
					<div style="display:flex;align-items:center;gap:11px;padding:8px 0;">
						<div style={stepNumStyle(i)}>
							{#if i < step}
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#07130c" stroke-width="3"><path d="M20 6L9 17l-5-5"/></svg>
							{:else}
								<span style="font-size:11px;font-weight:700;{stepNumColor(i)}">{s.num}</span>
							{/if}
						</div>
						<div style={stepLabelStyle(i)}>{s.label}</div>
					</div>
				{/each}

				<div style="margin-top:auto;font-size:11px;color:var(--tx-3);line-height:1.5;">
					Deploying to <b style="color:var(--tx-2);">{guild?.name ?? guildId}</b> via Podman Quadlets. No Dockerfiles required.
				</div>
			</div>

			<!-- Right content -->
			<div style="flex:1;min-width:0;display:flex;flex-direction:column;">

				<!-- Header -->
				<div style="flex:none;padding:22px 28px 8px;display:flex;align-items:flex-start;justify-content:space-between;">
					<div>
						<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:20px;">{stepHeads[step].title}</div>
						<div style="font-size:13px;color:var(--tx-2);margin-top:2px;">{stepHeads[step].sub}</div>
					</div>
					<button onclick={close} aria-label="Close" style="width:32px;height:32px;flex:none;border-radius:9px;background:var(--card-2);border:none;display:flex;align-items:center;justify-content:center;color:var(--tx-2);cursor:pointer;">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M6 6l12 12M18 6L6 18"/></svg>
					</button>
				</div>

				<!-- Step content -->
				<div style="flex:1;overflow-y:auto;padding:12px 28px 20px;">

					<!-- STEP 0: SOURCE -->
					{#if step === 0}
						<div style="display:flex;align-items:center;gap:11px;background:var(--grn-dim);border:1px solid var(--grn-line);border-radius:10px;padding:11px 14px;margin-bottom:16px;">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="var(--tx)"><path d="M12 1.5A10.5 10.5 0 001.5 12c0 4.6 3 8.5 7.2 9.9.5.1.7-.2.7-.5v-1.9c-2.9.6-3.5-1.2-3.5-1.2-.5-1.2-1.2-1.5-1.2-1.5-.9-.6.1-.6.1-.6 1 .1 1.6 1 1.6 1 .9 1.6 2.4 1.1 3 .9.1-.7.4-1.1.7-1.4-2.3-.3-4.8-1.2-4.8-5.2 0-1.1.4-2 1-2.8-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.7 1a9.4 9.4 0 015 0c1.9-1.3 2.7-1 2.7-1 .5 1.4.2 2.4.1 2.7.6.8 1 1.7 1 2.8 0 4-2.5 4.9-4.8 5.2.4.3.7 1 .7 2v2.9c0 .3.2.6.7.5A10.5 10.5 0 0022.5 12 10.5 10.5 0 0012 1.5z"/></svg>
							<div style="flex:1;">
								<div style="font-size:13px;font-weight:600;color:var(--tx);">Bakery GitHub App installed on <span style="font-family:'JetBrains Mono',monospace;">sourdough-labs</span></div>
								<div style="font-size:11.5px;color:var(--tx-2);">Reading 5 private repositories · last synced 2m ago</div>
							</div>
							<span style="font-size:12px;color:var(--grn-2);font-weight:600;cursor:pointer;">Configure</span>
						</div>

						<div style="display:flex;align-items:center;gap:9px;background:var(--card-2);border:1px solid var(--line);border-radius:9px;padding:9px 12px;color:var(--tx-3);margin-bottom:10px;">
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
							<input
								bind:value={repoSearch}
								placeholder="Search repositories…"
								style="flex:1;background:transparent;border:none;font-size:13px;color:var(--tx);outline:none;"
							/>
						</div>

						{#each filteredRepos as r (r.id)}
							{@const sel = wRepo === r.id}
							<button
								onclick={() => wRepo = r.id}
								style="display:flex;align-items:center;gap:12px;width:100%;padding:11px 13px;border-radius:9px;margin-bottom:6px;cursor:pointer;background:{sel ? 'var(--grn-dim)' : 'var(--card-2)'};border:1px solid {sel ? 'var(--grn-line)' : 'var(--line)'};"
							>
								<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.8" style="flex:none;"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 018 0v3"/></svg>
								<div style="flex:1;min-width:0;text-align:left;">
									<div style="font-family:'JetBrains Mono',monospace;font-size:13px;font-weight:500;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{r.name}</div>
								</div>
								<div style="display:flex;align-items:center;gap:6px;">
									<div style="width:9px;height:9px;border-radius:50%;background:{r.langColor};"></div>
									<span style="font-size:11.5px;color:var(--tx-2);">{r.lang}</span>
								</div>
								<span style="font-size:11.5px;color:var(--tx-3);width:96px;text-align:right;">{r.updated}</span>
								<div style="width:18px;height:18px;border-radius:50%;border:2px solid {sel ? 'var(--grn)' : 'var(--tx-3)'};display:flex;align-items:center;justify-content:center;flex:none;">
									{#if sel}<div style="width:8px;height:8px;border-radius:50%;background:var(--grn);"></div>{/if}
								</div>
							</button>
						{/each}
					{/if}

					<!-- STEP 1: HOST -->
					{#if step === 1}
						{#each hosts as h (h.name)}
							{@const sel = wHost === h.name}
							<button
								onclick={() => wHost = h.name}
								style="display:flex;flex-direction:column;width:100%;padding:14px;border-radius:12px;margin-bottom:10px;cursor:pointer;background:{sel ? 'var(--grn-dim)' : 'var(--card-2)'};border:1px solid {sel ? 'var(--grn-line)' : 'var(--line)'};"
							>
								<div style="display:flex;align-items:center;gap:12px;margin-bottom:12px;">
									<div style="width:38px;height:38px;border-radius:10px;background:var(--card);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;color:var(--grn);flex:none;">
										<svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><rect x="3" y="4" width="18" height="6" rx="1.6"/><rect x="3" y="14" width="18" height="6" rx="1.6"/></svg>
									</div>
									<div style="flex:1;text-align:left;">
										<div style="font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:600;color:var(--tx);">{h.name}</div>
										<div style="font-size:11.5px;color:var(--tx-2);">{h.spec} · {h.apps} apps</div>
									</div>
									<div style="display:flex;align-items:center;gap:6px;font-size:11.5px;color:var(--grn);">
										<div style="width:7px;height:7px;border-radius:50%;background:var(--grn);"></div>online
									</div>
									<div style="width:18px;height:18px;border-radius:50%;border:2px solid {sel ? 'var(--grn)' : 'var(--tx-3)'};display:flex;align-items:center;justify-content:center;flex:none;">
										{#if sel}<div style="width:8px;height:8px;border-radius:50%;background:var(--grn);"></div>{/if}
									</div>
								</div>
								<div style="display:flex;gap:16px;">
									{#each [['CPU', h.cpu], ['MEM', h.mem], ['DISK', h.disk]] as [label, val] (label)}
										<div style="flex:1;">
											<div style="display:flex;justify-content:space-between;font-size:11px;margin-bottom:4px;">
												<span style="color:var(--tx-3);">{label}</span>
												<span style="font-family:'JetBrains Mono',monospace;color:var(--tx-2);">{val}%</span>
											</div>
											<div style="height:5px;border-radius:3px;background:rgba(255,255,255,.07);overflow:hidden;">
												<div style="width:{val}%;height:100%;background:{Number(val) > 80 ? '#f0836b' : 'var(--grn)'};border-radius:3px;"></div>
											</div>
										</div>
									{/each}
								</div>
							</button>
						{/each}

						<button style="width:100%;border:1.5px dashed var(--line-2);border-radius:12px;padding:14px;display:flex;align-items:center;gap:11px;color:var(--tx-2);cursor:pointer;background:transparent;">
							<div style="width:32px;height:32px;border-radius:9px;background:var(--card);display:flex;align-items:center;justify-content:center;color:var(--grn);flex:none;">
								<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>
							</div>
							<div style="text-align:left;">
								<div style="font-size:13px;font-weight:600;color:var(--tx);">Add a new host</div>
								<div style="font-size:11.5px;">Register another machine's Podman socket</div>
							</div>
						</button>
					{/if}

					<!-- STEP 2: QUADLET -->
					{#if step === 2}
						<div style="display:flex;align-items:center;gap:9px;font-size:13px;color:var(--grn);margin-bottom:14px;">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M20 6L9 17l-5-5"/></svg>
							Found 2 Podman Quadlet files in <span style="font-family:'JetBrains Mono',monospace;color:var(--grn-2);">{wRepo ?? 'repo'}</span>
						</div>

						{#each quadletOptions as q, i (i)}
							{@const sel = wQuadlet === i}
							<button
								onclick={() => wQuadlet = i}
								style="display:flex;align-items:center;gap:12px;width:100%;padding:11px 14px;border-radius:9px;margin-bottom:7px;cursor:pointer;background:{sel ? 'var(--grn-dim)' : 'var(--card-2)'};border:1px solid {sel ? 'var(--grn-line)' : 'var(--line)'};"
							>
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.7" style="flex:none;"><path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/><path d="M13 2v7h7"/></svg>
								<div style="flex:1;min-width:0;text-align:left;">
									<div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx);">{q.path}</div>
									<div style="font-size:11px;color:var(--tx-3);margin-top:1px;">{q.note}</div>
								</div>
								<span style="font-family:'JetBrains Mono',monospace;font-size:10.5px;padding:3px 9px;border-radius:6px;background:{sel ? 'var(--grn-dim)' : 'rgba(255,255,255,.06)'};color:{sel ? 'var(--grn)' : 'var(--tx-3)'};">{q.tag}</span>
								<div style="width:18px;height:18px;border-radius:50%;border:2px solid {sel ? 'var(--grn)' : 'var(--tx-3)'};display:flex;align-items:center;justify-content:center;flex:none;">
									{#if sel}<div style="width:8px;height:8px;border-radius:50%;background:var(--grn);"></div>{/if}
								</div>
							</button>
						{/each}

						<div style="font-size:11.5px;color:var(--tx-3);margin:14px 0 8px;">Preview · {quadletOptions[wQuadlet]?.path}</div>
						<pre style="font-family:'JetBrains Mono',monospace;font-size:12px;line-height:1.8;background:#080c09;border:1px solid var(--line);border-radius:11px;padding:16px 18px;overflow-x:auto;color:var(--tx);margin:0;white-space:pre-wrap;">{quadletPreview}</pre>
					{/if}

					<!-- STEP 3: DOMAIN & PROXY -->
					{#if step === 3}
						<div style="font-size:12.5px;color:var(--tx-2);margin-bottom:6px;">Domain</div>
						<div style="display:flex;align-items:center;background:var(--card-2);border:1px solid var(--line);border-radius:9px;overflow:hidden;margin-bottom:6px;">
							<span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx-3);padding:0 0 0 13px;">https://</span>
							<input
								bind:value={wDomain}
								placeholder="api.sourdough.dev"
								style="flex:1;background:transparent;border:none;padding:12px 13px;font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx);outline:none;"
							/>
						</div>
						<div style="font-size:11.5px;color:var(--tx-3);margin-bottom:20px;">Point an A / CNAME record at your host, then Caddy provisions TLS automatically.</div>

						<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px;">
							<div style="display:flex;align-items:center;gap:12px;">
								<div style="flex:1;">
									<div style="font-size:13.5px;font-weight:700;">Add a Caddy reverse proxy</div>
									<div style="font-size:12px;color:var(--tx-2);margin-top:2px;">Route the domain to your app's port with automatic HTTPS.</div>
								</div>
								<button
									onclick={() => wProxy = !wProxy}
									aria-label="Toggle Caddy reverse proxy"
									style="width:40px;height:23px;border-radius:12px;background:{wProxy ? 'var(--grn)' : 'rgba(255,255,255,.1)'};border:none;position:relative;cursor:pointer;flex:none;transition:background .2s;"
								>
									<div style="width:17px;height:17px;border-radius:50%;background:#fff;position:absolute;top:3px;left:{wProxy ? '20px' : '3px'};transition:left .2s;"></div>
								</button>
							</div>

							{#if wProxy}
								<div style="border-top:1px solid var(--line);margin-top:14px;padding-top:14px;">
									<div style="display:flex;gap:14px;margin-bottom:14px;">
										<div style="flex:1;">
											<div style="font-size:12px;color:var(--tx-2);margin-bottom:6px;">Container port</div>
											<input bind:value={wPort} style={inputStyle} />
										</div>
										<div style="flex:1;">
											<div style="font-size:12px;color:var(--tx-2);margin-bottom:6px;">TLS</div>
											<button
												onclick={() => wTls = !wTls}
												style="display:flex;align-items:center;gap:8px;background:var(--card-2);border:1px solid var(--line);border-radius:8px;padding:11px 12px;cursor:pointer;font-size:13px;width:100%;"
											>
												<div style="width:16px;height:16px;border-radius:4px;border:1.5px solid {wTls ? 'var(--grn)' : 'var(--tx-3)'};background:{wTls ? 'var(--grn)' : 'transparent'};display:flex;align-items:center;justify-content:center;flex:none;">
													{#if wTls}<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#07130c" stroke-width="3"><path d="M20 6L9 17l-5-5"/></svg>{/if}
												</div>
												<span style="color:var(--tx);">Automatic (Let's Encrypt)</span>
											</button>
										</div>
									</div>
									{#if wDomain}
										<div style="font-family:'JetBrains Mono',monospace;font-size:12px;background:#080c09;border:1px solid var(--line);border-radius:9px;padding:12px 14px;color:var(--tx-2);">
											<span style="color:var(--grn-2);">{wDomain}</span> → reverse_proxy <span style="color:var(--amber);">127.0.0.1:{wPort}</span>
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/if}

					<!-- STEP 4: ENVIRONMENT -->
					{#if step === 4}
						<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;">
							<div style="font-size:12.5px;color:var(--tx-2);">Injected via the quadlet's <span style="font-family:'JetBrains Mono',monospace;color:var(--grn-2);">EnvironmentFile</span>. Mark sensitive values as secrets.</div>
							<button style="background:var(--card-2);border:1px solid var(--line);color:var(--tx);border-radius:8px;padding:7px 11px;font-size:12px;font-weight:600;cursor:pointer;">Import .env</button>
						</div>

						<div style="display:grid;grid-template-columns:1fr 1.5fr 92px 34px;gap:9px;padding:0 2px 8px;font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
							<div>KEY</div><div>VALUE</div><div>TYPE</div><div></div>
						</div>

						{#each wEnv as e, i (i)}
							<div style="display:grid;grid-template-columns:1fr 1.5fr 92px 34px;gap:9px;margin-bottom:8px;align-items:center;">
								<input
									value={e.key}
									oninput={(ev) => updateEnvKey(i, (ev.target as HTMLInputElement).value)}
									style="{inputStyle}color:var(--grn-2);"
								/>
								<input
									value={e.value}
									oninput={(ev) => updateEnvVal(i, (ev.target as HTMLInputElement).value)}
									type={e.secret ? 'password' : 'text'}
									style={inputStyle}
								/>
								<button
									onclick={() => toggleSecret(i)}
									style="padding:0 10px;height:38px;border-radius:8px;font-size:12px;font-weight:600;cursor:pointer;background:{e.secret ? 'rgba(224,168,62,.14)' : 'rgba(255,255,255,.05)'};border:1px solid {e.secret ? 'rgba(224,168,62,.3)' : 'var(--line)'};color:{e.secret ? '#efc060' : 'var(--tx-3)'};"
								>{e.secret ? 'secret' : 'variable'}</button>
								<button
									onclick={() => removeEnv(i)}
									aria-label="Remove variable"
									style="display:flex;align-items:center;justify-content:center;color:var(--tx-3);cursor:pointer;height:38px;border-radius:8px;background:transparent;border:none;"
								>
									<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6L6 18"/></svg>
								</button>
							</div>
						{/each}

						<button
							onclick={addEnv}
							style="display:inline-flex;align-items:center;gap:7px;color:var(--grn-2);font-size:13px;font-weight:600;cursor:pointer;margin-top:6px;background:none;border:none;"
						>
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M12 5v14M5 12h14"/></svg>
							Add variable
						</button>
					{/if}

					<!-- STEP 5: REVIEW & DEPLOY -->
					{#if step === 5}
						{#if deployPhase === 'idle'}
							<div style="background:var(--card);border:1px solid var(--line);border-radius:12px;overflow:hidden;margin-bottom:18px;">
								{#each reviewRows as row (row.k)}
									<div style="display:flex;align-items:center;gap:12px;padding:12px 18px;border-bottom:1px solid var(--line);">
										<span style="font-size:12.5px;color:var(--tx-2);width:120px;flex:none;">{row.k}</span>
										<span style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx);flex:1;">{row.v}</span>
									</div>
								{/each}
							</div>
							<button
								onclick={deploy}
								style="width:100%;background:var(--grn);color:#07130c;border:none;border-radius:11px;padding:15px;font-size:15px;font-weight:700;cursor:pointer;display:flex;align-items:center;justify-content:center;gap:9px;"
							>
								<span style="font-size:18px;">🥖</span>
								Bake it — deploy {appName}
							</button>
						{:else}
							<div style="font-family:'JetBrains Mono',monospace;font-size:12.5px;line-height:1.9;background:#080c09;border:1px solid var(--line);border-radius:12px;padding:18px 20px;min-height:280px;">
								{#each deployLog as l, li (li)}
									<div style="color:{l.color};">{l.text}</div>
								{/each}
								{#if deployPhase === 'running'}
									<div style="display:flex;align-items:center;gap:8px;color:var(--tx-3);margin-top:4px;">
										<div style="width:10px;height:10px;border-radius:50%;border:2px solid var(--grn);border-top-color:transparent;animation:bk-spin .7s linear infinite;"></div>
										working…
									</div>
								{/if}
							</div>
							{#if deployPhase === 'done'}
								<div style="margin-top:12px;display:flex;align-items:center;gap:10px;color:var(--grn);font-weight:700;font-size:14px;padding:12px 0;">
									<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4"><path d="M20 6L9 17l-5-5"/></svg>
									{appName} is fresh and serving traffic.
								</div>
								<button
									onclick={close}
									style="width:100%;background:var(--grn);color:#07130c;border:none;border-radius:11px;padding:14px;font-size:14.5px;font-weight:700;cursor:pointer;"
								>Open {appName} →</button>
							{/if}
						{/if}
					{/if}
				</div>

				<!-- Footer nav -->
				{#if !(step === 5 && deployPhase !== 'idle')}
					<div style="flex:none;border-top:1px solid var(--line);padding:14px 28px;display:flex;align-items:center;gap:12px;">
						<button
							onclick={back}
							style="padding:9px 16px;border-radius:9px;font-size:13.5px;font-weight:600;cursor:{step > 0 ? 'pointer' : 'not-allowed'};opacity:{step > 0 ? 1 : 0.35};background:var(--card-2);border:1px solid var(--line);color:var(--tx-2);"
						>Back</button>
						<div style="flex:1;"></div>
						<span style="font-size:12px;color:var(--tx-3);">Step {step + 1} of 6</span>
						{#if step < 5}
							<button
								onclick={next}
								style="padding:9px 20px;border-radius:9px;font-size:13.5px;font-weight:700;cursor:{canContinue ? 'pointer' : 'not-allowed'};background:{canContinue ? 'var(--grn)' : 'rgba(63,185,132,.25)'};color:{canContinue ? '#07130c' : 'rgba(63,185,132,.5)'};border:none;transition:background .15s;"
							>{step === 4 ? 'Review' : 'Continue'}</button>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	button { all: unset; box-sizing: border-box; }
	input:focus { outline: none; border-color: var(--grn-line) !important; }
	pre { tab-size: 2; }
</style>
