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

	function stepNumClass(i: number) {
		const base = 'size-[22px] rounded-full flex items-center justify-center shrink-0';
		return i <= step ? `${base} bg-[var(--grn)]` : `${base} bg-white/[0.06]`;
	}
	function stepLabelClass(i: number) {
		if (i === step) return 'text-[13px] font-bold text-[var(--tx)]';
		if (i < step) return 'text-[13px] font-semibold text-[var(--grn-2)]';
		return 'text-[13px] text-[var(--tx-3)]';
	}
	function stepNumIconColor(i: number) {
		return i <= step ? '#07130c' : 'var(--tx-3)';
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

	const inputCls = 'bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] px-3 py-[10px] font-mono-jb text-[13px] w-full';
</script>

<svelte:window onkeydown={handleKey} />

{#if open}
	<div
		onclick={handleBackdrop}
		role="presentation"
		class="fixed inset-0 z-50 bg-[rgba(6,9,7,.72)] backdrop-blur-[6px] flex items-center justify-center p-7"
	>
		<div class="w-[960px] max-w-full h-[640px] max-h-[92vh] bg-[var(--panel)] border border-[var(--line-2)] rounded-[18px] shadow-[0_30px_80px_rgba(0,0,0,.5)] flex overflow-hidden animate-[bk-fadeup_.22s_ease]">

			<!-- Left step rail -->
			<div class="w-[238px] shrink-0 bg-[var(--sidebar)] border-r border-r-[var(--line)] p-[22px_18px] flex flex-col">
				<div class="font-heading font-bold text-[17px] mb-[3px]">New app</div>
				<div class="text-[12px] text-[var(--tx-2)] mb-[22px]">Repo → live in six steps.</div>

				{#each wSteps as s, i (i)}
					<div class="flex items-center gap-[11px] py-2">
						<div class={stepNumClass(i)}>
							{#if i < step}
								<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#07130c" stroke-width="3"><path d="M20 6L9 17l-5-5"/></svg>
							{:else}
								<span class="text-[11px] font-bold" style:color={stepNumIconColor(i)}>{s.num}</span>
							{/if}
						</div>
						<div class={stepLabelClass(i)}>{s.label}</div>
					</div>
				{/each}

				<div class="mt-auto text-[11px] text-[var(--tx-3)] leading-[1.5]">
					Deploying to <b class="text-[var(--tx-2)]">{guild?.name ?? guildId}</b> via Podman Quadlets. No Dockerfiles required.
				</div>
			</div>

			<!-- Right content -->
			<div class="flex-1 min-w-0 flex flex-col">

				<!-- Header -->
				<div class="shrink-0 pt-[22px] px-7 pb-2 flex items-start justify-between">
					<div>
						<div class="font-heading font-bold text-[20px]">{stepHeads[step].title}</div>
						<div class="text-[13px] text-[var(--tx-2)] mt-0.5">{stepHeads[step].sub}</div>
					</div>
					<button onclick={close} aria-label="Close" class="size-8 shrink-0 rounded-[9px] bg-[var(--card-2)] flex items-center justify-center text-[var(--tx-2)] cursor-pointer">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M6 6l12 12M18 6L6 18"/></svg>
					</button>
				</div>

				<!-- Step content -->
				<div class="flex-1 overflow-y-auto px-7 pt-3 pb-5">

					<!-- STEP 0: SOURCE -->
					{#if step === 0}
						<div class="flex items-center gap-[11px] bg-[var(--grn-dim)] border border-[var(--grn-line)] rounded-[10px] px-[14px] py-[11px] mb-4">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="var(--tx)"><path d="M12 1.5A10.5 10.5 0 001.5 12c0 4.6 3 8.5 7.2 9.9.5.1.7-.2.7-.5v-1.9c-2.9.6-3.5-1.2-3.5-1.2-.5-1.2-1.2-1.5-1.2-1.5-.9-.6.1-.6.1-.6 1 .1 1.6 1 1.6 1 .9 1.6 2.4 1.1 3 .9.1-.7.4-1.1.7-1.4-2.3-.3-4.8-1.2-4.8-5.2 0-1.1.4-2 1-2.8-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.7 1a9.4 9.4 0 015 0c1.9-1.3 2.7-1 2.7-1 .5 1.4.2 2.4.1 2.7.6.8 1 1.7 1 2.8 0 4-2.5 4.9-4.8 5.2.4.3.7 1 .7 2v2.9c0 .3.2.6.7.5A10.5 10.5 0 0022.5 12 10.5 10.5 0 0012 1.5z"/></svg>
							<div class="flex-1">
								<div class="text-[13px] font-semibold text-[var(--tx)]">Bakery GitHub App installed on <span class="font-mono-jb">sourdough-labs</span></div>
								<div class="text-[11.5px] text-[var(--tx-2)]">Reading 5 private repositories · last synced 2m ago</div>
							</div>
							<span class="text-[12px] text-[var(--grn-2)] font-semibold cursor-pointer">Configure</span>
						</div>

						<div class="flex items-center gap-[9px] bg-[var(--card-2)] border border-[var(--line)] rounded-[9px] px-3 py-[9px] text-[var(--tx-3)] mb-[10px]">
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="M20 20l-3.5-3.5"/></svg>
							<input
								bind:value={repoSearch}
								placeholder="Search repositories…"
								class="flex-1 bg-transparent border-none text-[13px] text-[var(--tx)] outline-none"
							/>
						</div>

						{#each filteredRepos as r (r.id)}
							{@const sel = wRepo === r.id}
							<button
								onclick={() => wRepo = r.id}
								class="flex items-center gap-3 w-full px-[13px] py-[11px] rounded-[9px] mb-[6px] cursor-pointer border {sel ? 'bg-[var(--grn-dim)] border-[var(--grn-line)]' : 'bg-[var(--card-2)] border-[var(--line)]'}"
							>
								<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.8" class="shrink-0"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 018 0v3"/></svg>
								<div class="flex-1 min-w-0 text-left">
									<div class="font-mono-jb text-[13px] font-medium text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{r.name}</div>
								</div>
								<div class="flex items-center gap-[6px]">
									<div class="size-[9px] rounded-full" style:background={r.langColor}></div>
									<span class="text-[11.5px] text-[var(--tx-2)]">{r.lang}</span>
								</div>
								<span class="text-[11.5px] text-[var(--tx-3)] w-24 text-right">{r.updated}</span>
								<div class="size-[18px] rounded-full border-2 flex items-center justify-center shrink-0" style:border-color={sel ? 'var(--grn)' : 'var(--tx-3)'}>
									{#if sel}<div class="size-2 rounded-full bg-[var(--grn)]"></div>{/if}
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
								class="flex flex-col w-full p-[14px] rounded-[12px] mb-[10px] cursor-pointer border {sel ? 'bg-[var(--grn-dim)] border-[var(--grn-line)]' : 'bg-[var(--card-2)] border-[var(--line)]'}"
							>
								<div class="flex items-center gap-3 mb-3">
									<div class="size-[38px] rounded-[10px] bg-[var(--card)] border border-[var(--line)] flex items-center justify-center text-[var(--grn)] shrink-0">
										<svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><rect x="3" y="4" width="18" height="6" rx="1.6"/><rect x="3" y="14" width="18" height="6" rx="1.6"/></svg>
									</div>
									<div class="flex-1 text-left">
										<div class="font-mono-jb text-[14px] font-semibold text-[var(--tx)]">{h.name}</div>
										<div class="text-[11.5px] text-[var(--tx-2)]">{h.spec} · {h.apps} apps</div>
									</div>
									<div class="flex items-center gap-[6px] text-[11.5px] text-[var(--grn)]">
										<div class="size-[7px] rounded-full bg-[var(--grn)]"></div>online
									</div>
									<div class="size-[18px] rounded-full border-2 flex items-center justify-center shrink-0" style:border-color={sel ? 'var(--grn)' : 'var(--tx-3)'}>
										{#if sel}<div class="size-2 rounded-full bg-[var(--grn)]"></div>{/if}
									</div>
								</div>
								<div class="flex gap-4">
									{#each [['CPU', h.cpu], ['MEM', h.mem], ['DISK', h.disk]] as [label, val] (label)}
										<div class="flex-1">
											<div class="flex justify-between text-[11px] mb-1">
												<span class="text-[var(--tx-3)]">{label}</span>
												<span class="font-mono-jb text-[var(--tx-2)]">{val}%</span>
											</div>
											<div class="h-[5px] rounded-[3px] bg-white/[0.07] overflow-hidden">
												<div class="h-full rounded-[3px]" style:width="{val}%" style:background={Number(val) > 80 ? '#f0836b' : 'var(--grn)'}></div>
											</div>
										</div>
									{/each}
								</div>
							</button>
						{/each}

						<button class="w-full border-[1.5px] border-dashed border-[var(--line-2)] rounded-[12px] p-[14px] flex items-center gap-[11px] text-[var(--tx-2)] cursor-pointer bg-transparent">
							<div class="size-8 rounded-[9px] bg-[var(--card)] flex items-center justify-center text-[var(--grn)] shrink-0">
								<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>
							</div>
							<div class="text-left">
								<div class="text-[13px] font-semibold text-[var(--tx)]">Add a new host</div>
								<div class="text-[11.5px]">Register another machine's Podman socket</div>
							</div>
						</button>
					{/if}

					<!-- STEP 2: QUADLET -->
					{#if step === 2}
						<div class="flex items-center gap-[9px] text-[13px] text-[var(--grn)] mb-[14px]">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M20 6L9 17l-5-5"/></svg>
							Found 2 Podman Quadlet files in <span class="font-mono-jb text-[var(--grn-2)]">{wRepo ?? 'repo'}</span>
						</div>

						{#each quadletOptions as q, i (i)}
							{@const sel = wQuadlet === i}
							<button
								onclick={() => wQuadlet = i}
								class="flex items-center gap-3 w-full px-[14px] py-[11px] rounded-[9px] mb-[7px] cursor-pointer border {sel ? 'bg-[var(--grn-dim)] border-[var(--grn-line)]' : 'bg-[var(--card-2)] border-[var(--line)]'}"
							>
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.7" class="shrink-0"><path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/><path d="M13 2v7h7"/></svg>
								<div class="flex-1 min-w-0 text-left">
									<div class="font-mono-jb text-[13px] text-[var(--tx)]">{q.path}</div>
									<div class="text-[11px] text-[var(--tx-3)] mt-[1px]">{q.note}</div>
								</div>
								<span class="font-mono-jb text-[10.5px] px-[9px] py-[3px] rounded-[6px] {sel ? 'bg-[var(--grn-dim)] text-[var(--grn)]' : 'bg-white/[0.06] text-[var(--tx-3)]'}">{q.tag}</span>
								<div class="size-[18px] rounded-full border-2 flex items-center justify-center shrink-0" style:border-color={sel ? 'var(--grn)' : 'var(--tx-3)'}>
									{#if sel}<div class="size-2 rounded-full bg-[var(--grn)]"></div>{/if}
								</div>
							</button>
						{/each}

						<div class="text-[11.5px] text-[var(--tx-3)] mt-[14px] mb-2">Preview · {quadletOptions[wQuadlet]?.path}</div>
						<pre class="font-mono-jb text-[12px] leading-[1.8] bg-[#080c09] border border-[var(--line)] rounded-[11px] px-[18px] py-4 overflow-x-auto text-[var(--tx)] m-0 whitespace-pre-wrap [tab-size:2]">{quadletPreview}</pre>
					{/if}

					<!-- STEP 3: DOMAIN & PROXY -->
					{#if step === 3}
						<div class="text-[12.5px] text-[var(--tx-2)] mb-[6px]">Domain</div>
						<div class="flex items-center bg-[var(--card-2)] border border-[var(--line)] rounded-[9px] overflow-hidden mb-[6px]">
							<span class="font-mono-jb text-[13px] text-[var(--tx-3)] pl-[13px]">https://</span>
							<input
								bind:value={wDomain}
								placeholder="api.sourdough.dev"
								class="flex-1 bg-transparent border-none px-[13px] py-3 font-mono-jb text-[13px] text-[var(--tx)] outline-none"
							/>
						</div>
						<div class="text-[11.5px] text-[var(--tx-3)] mb-5">Point an A / CNAME record at your host, then Caddy provisions TLS automatically.</div>

						<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] p-[16px_18px]">
							<div class="flex items-center gap-3">
								<div class="flex-1">
									<div class="text-[13.5px] font-bold">Add a Caddy reverse proxy</div>
									<div class="text-[12px] text-[var(--tx-2)] mt-0.5">Route the domain to your app's port with automatic HTTPS.</div>
								</div>
								<button
									onclick={() => wProxy = !wProxy}
									aria-label="Toggle Caddy reverse proxy"
									class="relative w-10 h-[23px] rounded-[12px] border-none cursor-pointer shrink-0 transition-[background] duration-200 {wProxy ? 'bg-[var(--grn)]' : 'bg-white/10'}"
								>
									<div
										class="absolute top-[3px] size-[17px] rounded-full bg-white transition-[left] duration-200"
										style:left={wProxy ? '20px' : '3px'}
									></div>
								</button>
							</div>

							{#if wProxy}
								<div class="border-t border-t-[var(--line)] mt-[14px] pt-[14px]">
									<div class="flex gap-[14px] mb-[14px]">
										<div class="flex-1">
											<div class="text-[12px] text-[var(--tx-2)] mb-[6px]">Container port</div>
											<input bind:value={wPort} class="{inputCls} text-[var(--tx)]" />
										</div>
										<div class="flex-1">
											<div class="text-[12px] text-[var(--tx-2)] mb-[6px]">TLS</div>
											<button
												onclick={() => wTls = !wTls}
												class="flex items-center gap-2 bg-[var(--card-2)] border border-[var(--line)] rounded-[8px] px-3 py-[11px] cursor-pointer text-[13px] w-full"
											>
												<div
													class="size-4 rounded-[4px] flex items-center justify-center shrink-0 border-[1.5px] {wTls ? 'border-[var(--grn)] bg-[var(--grn)]' : 'border-[var(--tx-3)] bg-transparent'}"
												>
													{#if wTls}<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#07130c" stroke-width="3"><path d="M20 6L9 17l-5-5"/></svg>{/if}
												</div>
												<span class="text-[var(--tx)]">Automatic (Let's Encrypt)</span>
											</button>
										</div>
									</div>
									{#if wDomain}
										<div class="font-mono-jb text-[12px] bg-[#080c09] border border-[var(--line)] rounded-[9px] px-[14px] py-3 text-[var(--tx-2)]">
											<span class="text-[var(--grn-2)]">{wDomain}</span> → reverse_proxy <span class="text-[var(--amber)]">127.0.0.1:{wPort}</span>
										</div>
									{/if}
								</div>
							{/if}
						</div>
					{/if}

					<!-- STEP 4: ENVIRONMENT -->
					{#if step === 4}
						<div class="flex items-center justify-between mb-3">
							<div class="text-[12.5px] text-[var(--tx-2)]">Injected via the quadlet's <span class="font-mono-jb text-[var(--grn-2)]">EnvironmentFile</span>. Mark sensitive values as secrets.</div>
							<button class="bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] rounded-[8px] px-[11px] py-[7px] text-[12px] font-semibold cursor-pointer">Import .env</button>
						</div>

						<div class="grid grid-cols-[1fr_1.5fr_92px_34px] gap-[9px] px-0.5 pb-2 text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
							<div>KEY</div><div>VALUE</div><div>TYPE</div><div></div>
						</div>

						{#each wEnv as e, i (i)}
							<div class="grid grid-cols-[1fr_1.5fr_92px_34px] gap-[9px] mb-2 items-center">
								<input
									value={e.key}
									oninput={(ev) => updateEnvKey(i, (ev.target as HTMLInputElement).value)}
									class="{inputCls} text-[var(--grn-2)]"
								/>
								<input
									value={e.value}
									oninput={(ev) => updateEnvVal(i, (ev.target as HTMLInputElement).value)}
									type={e.secret ? 'password' : 'text'}
									class="{inputCls} text-[var(--tx)]"
								/>
								<button
									onclick={() => toggleSecret(i)}
									class="h-[38px] px-[10px] rounded-[8px] text-[12px] font-semibold cursor-pointer border {e.secret ? 'bg-[rgba(224,168,62,.14)] border-[rgba(224,168,62,.3)] text-[#efc060]' : 'bg-white/5 border-[var(--line)] text-[var(--tx-3)]'}"
								>{e.secret ? 'secret' : 'variable'}</button>
								<button
									onclick={() => removeEnv(i)}
									aria-label="Remove variable"
									class="flex items-center justify-center text-[var(--tx-3)] cursor-pointer h-[38px] rounded-[8px] bg-transparent border-none"
								>
									<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 6l12 12M18 6L6 18"/></svg>
								</button>
							</div>
						{/each}

						<button
							onclick={addEnv}
							class="inline-flex items-center gap-[7px] text-[var(--grn-2)] text-[13px] font-semibold cursor-pointer mt-[6px] bg-none border-none"
						>
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M12 5v14M5 12h14"/></svg>
							Add variable
						</button>
					{/if}

					<!-- STEP 5: REVIEW & DEPLOY -->
					{#if step === 5}
						{#if deployPhase === 'idle'}
							<div class="bg-[var(--card)] border border-[var(--line)] rounded-[12px] overflow-hidden mb-[18px]">
								{#each reviewRows as row (row.k)}
									<div class="flex items-center gap-3 px-[18px] py-3 border-b border-b-[var(--line)]">
										<span class="text-[12.5px] text-[var(--tx-2)] w-[120px] shrink-0">{row.k}</span>
										<span class="font-mono-jb text-[13px] text-[var(--tx)] flex-1">{row.v}</span>
									</div>
								{/each}
							</div>
							<button
								onclick={deploy}
								class="w-full bg-[var(--grn)] text-[#07130c] border-none rounded-[11px] py-[15px] text-[15px] font-bold cursor-pointer flex items-center justify-center gap-[9px]"
							>
								<span class="text-[18px]">🥖</span>
								Bake it — deploy {appName}
							</button>
						{:else}
							<div class="font-mono-jb text-[12.5px] leading-[1.9] bg-[#080c09] border border-[var(--line)] rounded-[12px] p-[18px_20px] min-h-[280px]">
								{#each deployLog as l, li (li)}
									<div style:color={l.color}>{l.text}</div>
								{/each}
								{#if deployPhase === 'running'}
									<div class="flex items-center gap-2 text-[var(--tx-3)] mt-1">
										<div class="size-[10px] rounded-full border-2 border-[var(--grn)] border-t-transparent animate-[bk-spin_.7s_linear_infinite]"></div>
										working…
									</div>
								{/if}
							</div>
							{#if deployPhase === 'done'}
								<div class="mt-3 flex items-center gap-[10px] text-[var(--grn)] font-bold text-[14px] py-3">
									<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4"><path d="M20 6L9 17l-5-5"/></svg>
									{appName} is fresh and serving traffic.
								</div>
								<button
									onclick={close}
									class="w-full bg-[var(--grn)] text-[#07130c] border-none rounded-[11px] py-[14px] text-[14.5px] font-bold cursor-pointer"
								>Open {appName} →</button>
							{/if}
						{/if}
					{/if}
				</div>

				<!-- Footer nav -->
				{#if !(step === 5 && deployPhase !== 'idle')}
					<div class="shrink-0 border-t border-t-[var(--line)] px-7 py-[14px] flex items-center gap-3">
						<button
							onclick={back}
							class="px-4 py-[9px] rounded-[9px] text-[13.5px] font-semibold bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx-2)] {step > 0 ? 'cursor-pointer opacity-100' : 'cursor-not-allowed opacity-35'}"
						>Back</button>
						<div class="flex-1"></div>
						<span class="text-[12px] text-[var(--tx-3)]">Step {step + 1} of 6</span>
						{#if step < 5}
							<button
								onclick={next}
								class="px-5 py-[9px] rounded-[9px] text-[13.5px] font-bold border-none transition-[background] duration-150 {canContinue ? 'bg-[var(--grn)] text-[#07130c] cursor-pointer' : 'bg-[rgba(63,185,132,.25)] text-[rgba(63,185,132,.5)] cursor-not-allowed'}"
							>{step === 4 ? 'Review' : 'Continue'}</button>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}
