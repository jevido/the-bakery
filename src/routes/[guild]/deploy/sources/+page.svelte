<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getContext, onMount, onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { SOURCES, REPOS, sourceMeta } from '$lib/data/bakery';

	onMount(() => {
		const url = new URL(page.url);
		if (url.searchParams.has('github_connected')) {
			toast.success('GitHub connected', { description: 'The installation is now linked to this guild.' });
			url.searchParams.delete('github_connected');
			goto(url, { replaceState: true, noScroll: true, keepFocus: true });
		} else if (url.searchParams.has('github_error')) {
			toast.error('GitHub connection failed', { description: url.searchParams.get('github_error') ?? undefined });
			url.searchParams.delete('github_error');
			goto(url, { replaceState: true, noScroll: true, keepFocus: true });
		}
	});

	const guildId    = $derived(page.params.guild ?? '');
	const sources    = $derived(SOURCES[guildId] ?? []);
	const totalRepos = $derived(sources.reduce((n, s) => n + s.repoCount, 0));

	let filter = $state<'all' | 'github' | 'selfhosted'>('all');

	const filtered = $derived(
		filter === 'all' ? sources
		: filter === 'github' ? sources.filter((s) => s.provider === 'github_app')
		: sources.filter((s) => s.provider !== 'github_app')
	);

	function filterCls(f: typeof filter) {
		return f === filter
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}

	let panelOpen  = $state(false);
	let step = $state<'pick' | 'github' | 'selfhosted' | 'done'>('pick');

	let shName     = $state('');
	let shUrl      = $state('');
	let shProvider = $state<'gitlab' | 'gitea' | 'bitbucket' | 'generic'>('gitlab');
	let shToken    = $state('');
	let testing    = $state(false);
	let tested     = $state(false);
	let testError  = $state('');

	function openPanel() {
		panelOpen = true;
		step = 'pick';
		shName = '';
		shUrl = '';
		shProvider = 'gitlab';
		shToken = '';
		tested = false;
		testError = '';
	}

	function closePanel() { panelOpen = false; }

	const cta = getContext<{ register(fn: () => void): void; unregister(): void }>('bakery:cta');
	onMount(() => cta?.register(openPanel));
	onDestroy(() => cta?.unregister());

	function finishConnection() {
		const label = shName || 'GitHub App';
		step = 'done';
		toast.success('Source connected', { description: `${label} is now linked` });
	}

	async function testConnection() {
		testing = true;
		tested = false;
		testError = '';
		await new Promise((r) => setTimeout(r, 1100));
		testing = false;
		if (!shUrl.startsWith('http') || !shToken) {
			testError = 'Could not reach the server. Check the URL and token.';
		} else {
			tested = true;
		}
	}

	const canConnect = $derived(tested && shName.trim().length > 0);

	const panelInputCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)]';
	const panelInputMonoCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-mono-jb';
	const panelSelectCls = 'w-full bg-[var(--card)] border border-[var(--line-2)] rounded-[8px] px-3 py-[9px] text-[13.5px] text-[var(--tx)] font-bakery appearance-none cursor-pointer';
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-start gap-[14px] mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Sources</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{#if sources.length > 0}
					{sources.length} source{sources.length !== 1 ? 's' : ''} · {totalRepos} repos accessible
				{:else}
					Connect repositories and container registries as deployment sources.
				{/if}
			</div>
		</div>
	</div>

	{#if sources.length === 0}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[40px_32px]">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)] mb-[6px]">No sources connected</div>
			<div class="text-[13px] text-[var(--tx-3)] mb-8">Follow three steps to connect your first git source and start pulling projects.</div>

			<div class="grid grid-cols-3 gap-4 mb-8">
				{#each [
					{ n: '1', title: 'Choose a provider', desc: 'Select GitHub App for cloud repos, or self-hosted for GitLab, Gitea, or Bitbucket Server.' },
					{ n: '2', title: 'Install & authorize', desc: 'Install the Bakery GitHub App on your org, or provide a personal access token for self-hosted.' },
					{ n: '3', title: 'Pick repos', desc: 'Choose which repositories Bakery can read and deploy from.' },
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
					{ label: 'GitHub App',       icon: 'github'    },
					{ label: 'GitLab',           icon: 'gitlab'    },
					{ label: 'Gitea',            icon: 'gitea'     },
					{ label: 'Bitbucket Server', icon: 'bitbucket' },
				] as p (p.icon)}
					<div class="flex items-center gap-[7px] bg-[var(--card-2)] border border-[var(--line)] rounded-[7px] px-3 py-[6px]">
						{@render ProviderIcon({ icon: p.icon, size: 14 })}
						<span class="text-[12.5px] text-[var(--tx-2)]">{p.label}</span>
					</div>
				{/each}
			</div>

			<button onclick={openPanel} class="inline-flex items-center gap-[7px] bg-[var(--grn)] text-[#07130c] rounded-[9px] px-[18px] py-[9px] text-[13.5px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
				Connect your first source
			</button>
		</div>

	{:else}
		<div class="flex items-center gap-[14px] mb-[14px]">
			<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
				<button onclick={() => filter = 'all'}          class={filterCls('all')}>All</button>
				<button onclick={() => filter = 'github'}       class={filterCls('github')}>GitHub</button>
				<button onclick={() => filter = 'selfhosted'}   class={filterCls('selfhosted')}>Self-hosted</button>
			</div>
		</div>

		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div class="grid grid-cols-[2fr_1.2fr_80px_120px_100px] gap-[14px] px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]">
				<div>SOURCE</div>
				<div>PROVIDER</div>
				<div>REPOS</div>
				<div>CONNECTED</div>
				<div></div>
			</div>

			{#each filtered as s (s.id)}
				{@const m = sourceMeta(s.provider)}
				<div class="group px-[18px] py-[13px] grid grid-cols-[2fr_1.2fr_80px_120px_100px] gap-[14px] items-center border-b border-b-[var(--line)] last:border-b-0 hover:bg-white/[0.02]">
					<div class="flex items-center gap-[11px] min-w-0">
						<div class="size-[34px] rounded-[9px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0">
							{@render ProviderIcon({ icon: s.provider === 'github_app' ? 'github' : s.provider, size: 17 })}
						</div>
						<div class="min-w-0">
							<div class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis">{s.name}</div>
							{#if s.host}
								<div class="font-mono-jb text-[11px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis">{s.host}</div>
							{:else}
								<div class="font-mono-jb text-[11px] text-[var(--tx-3)]">github.com</div>
							{/if}
						</div>
					</div>

					<div>
						<span class="text-[12px] font-semibold rounded-[5px] px-2 py-[3px]" style:color={m.color} style:background={m.bg}>{m.label}</span>
					</div>

					<div class="font-mono-jb text-[13px] text-[var(--tx-2)]">{s.repoCount}</div>

					<div class="text-[12.5px] text-[var(--tx-3)]">{s.connectedAt}</div>

					<div class="text-right">
						<button class="opacity-0 transition-opacity duration-150 group-hover:opacity-100 text-[12.5px] text-[var(--tx-3)] cursor-pointer px-2 py-1 rounded-[6px]">Configure →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div class="p-10 text-center text-[var(--tx-3)] text-[13.5px]">No sources match the current filter.</div>
			{/if}
		</div>

		{#if guildId === 'sourdough'}
			<div class="mt-5">
				<div class="text-[12px] font-bold tracking-[.05em] text-[var(--tx-3)] mb-[10px]">REPOS FROM SOURDOUGH-LABS</div>
				<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
					{#each REPOS as r (r.id)}
						<div class="flex items-center gap-[14px] px-[18px] py-[11px] border-b border-b-[var(--line)] last:border-b-0">
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
							<div class="flex-1 text-[13px] text-[var(--tx)]">{r.name}</div>
							<div class="flex items-center gap-[5px]">
								<div class="size-[10px] rounded-full" style:background={r.langColor}></div>
								<span class="text-[12px] text-[var(--tx-3)]">{r.lang}</span>
							</div>
							<div class="text-[12px] text-[var(--tx-3)] w-[110px] text-right">{r.updated}</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
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
			{#if step !== 'pick'}
				<button onclick={() => { step = 'pick'; tested = false; testError = ''; }} class="text-[var(--tx-3)] cursor-pointer text-[12.5px] flex items-center gap-1">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
					Back
				</button>
				<div class="w-px h-[14px] bg-[var(--line)] shrink-0"></div>
			{/if}
			<div class="font-heading text-[16px] font-bold text-[var(--tx)]">
				{step === 'pick' ? 'Add source' : step === 'github' ? 'GitHub App' : step === 'selfhosted' ? 'Self-hosted Git' : 'Source connected'}
			</div>
			<div class="flex-1"></div>
			<button onclick={closePanel} aria-label="Close panel" class="text-[var(--tx-3)] cursor-pointer flex items-center">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<div class="flex-1 overflow-y-auto p-6">
			{#if step === 'pick'}
				<div class="text-[13px] text-[var(--tx-3)] mb-5">Choose where your code lives. You can add multiple sources.</div>

				<button onclick={() => step = 'github'} class="w-full text-left bg-[var(--card)] border border-[var(--line)] rounded-[11px] p-[18px] mb-[10px] cursor-pointer flex items-start gap-[14px] hover:bg-[var(--card-2)] hover:border-[var(--line-2)] transition-colors">
					<div class="size-10 rounded-[10px] bg-white/[.06] flex items-center justify-center shrink-0 mt-[1px]">
						{@render ProviderIcon({ icon: 'github', size: 20 })}
					</div>
					<div>
						<div class="text-[14px] font-semibold text-[var(--tx)] mb-1">GitHub App</div>
						<div class="text-[12.5px] text-[var(--tx-3)] leading-[1.5]">Install once on your GitHub org or personal account. Bakery gets fine-grained access to selected repos — no secrets stored.</div>
						<div class="mt-[10px] flex gap-[6px]">
							<span class="text-[11px] text-[var(--grn-2)] bg-[var(--grn-dim)] rounded-[4px] px-[7px] py-[2px]">Recommended</span>
							<span class="text-[11px] text-[var(--tx-3)] bg-white/5 rounded-[4px] px-[7px] py-[2px]">github.com</span>
						</div>
					</div>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="2" class="ml-auto shrink-0 mt-[2px]"><path d="M9 18l6-6-6-6"/></svg>
				</button>

				<button onclick={() => step = 'selfhosted'} class="w-full text-left bg-[var(--card)] border border-[var(--line)] rounded-[11px] p-[18px] cursor-pointer flex items-start gap-[14px] hover:bg-[var(--card-2)] hover:border-[var(--line-2)] transition-colors">
					<div class="size-10 rounded-[10px] bg-white/[.06] flex items-center justify-center shrink-0 mt-[1px]">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.6"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
					</div>
					<div>
						<div class="text-[14px] font-semibold text-[var(--tx)] mb-1">Self-hosted Git</div>
						<div class="text-[12.5px] text-[var(--tx-3)] leading-[1.5]">Connect a GitLab, Gitea, or Bitbucket Server instance running on your own infrastructure using a personal access token.</div>
						<div class="mt-[10px] flex gap-[6px]">
							<span class="text-[11px] text-[var(--tx-3)] bg-white/5 rounded-[4px] px-[7px] py-[2px]">GitLab</span>
							<span class="text-[11px] text-[var(--tx-3)] bg-white/5 rounded-[4px] px-[7px] py-[2px]">Gitea</span>
							<span class="text-[11px] text-[var(--tx-3)] bg-white/5 rounded-[4px] px-[7px] py-[2px]">Bitbucket Server</span>
						</div>
					</div>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="2" class="ml-auto shrink-0 mt-[2px]"><path d="M9 18l6-6-6-6"/></svg>
				</button>

			{:else if step === 'github'}
				<div class="bg-[var(--grn-dim)] border border-[var(--grn-line)] rounded-[10px] px-4 py-[14px] mb-6 flex gap-[10px]">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2" class="shrink-0 mt-[1px]"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
					<div class="text-[12.5px] text-[var(--grn-2)] leading-[1.5]">The Bakery GitHub App requests <strong>read-only</strong> access to contents and metadata. No write permissions are ever requested.</div>
				</div>

				<div class="mb-6">
					{#each [
						{ n: '1', label: 'Click "Install on GitHub" below', detail: 'You will be redirected to GitHub to install the Bakery App on your account or organisation.' },
						{ n: '2', label: 'Select repositories', detail: 'Choose "All repositories" or specific repos you want Bakery to access.' },
						{ n: '3', label: 'Return here', detail: 'GitHub will redirect you back and your source will appear in the list automatically.' },
					] as inst (inst.n)}
						<div class="flex gap-[14px] mb-4">
							<div class="size-6 rounded-full bg-[var(--card-2)] border border-[var(--line-2)] flex items-center justify-center shrink-0 mt-[1px]">
								<span class="font-mono-jb text-[11px] font-bold text-[var(--tx-2)]">{inst.n}</span>
							</div>
							<div>
								<div class="text-[13.5px] font-semibold text-[var(--tx)] mb-[3px]">{inst.label}</div>
								<div class="text-[12px] text-[var(--tx-3)] leading-[1.5]">{inst.detail}</div>
							</div>
						</div>
					{/each}
				</div>

				<form method="POST" action="?/connectGithub">
					<button type="submit" class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">
						{@render ProviderIcon({ icon: 'github', size: 16, color: '#07130c' })}
						Install on GitHub →
					</button>
				</form>

			{:else if step === 'selfhosted'}
				<div class="flex flex-col gap-4">
					<div>
						<label for="sh-name" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">DISPLAY NAME</label>
						<input id="sh-name" bind:value={shName} placeholder="my-gitlab" class={panelInputCls} />
					</div>

					<div>
						<label for="sh-url" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">BASE URL</label>
						<input id="sh-url" bind:value={shUrl} placeholder="https://git.example.com" class={panelInputMonoCls} />
					</div>

					<div>
						<label for="sh-provider" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">PROVIDER</label>
						<select id="sh-provider" bind:value={shProvider} class={panelSelectCls}>
							<option value="gitlab">GitLab</option>
							<option value="gitea">Gitea</option>
							<option value="bitbucket">Bitbucket Server</option>
							<option value="generic">Generic Git (SSH)</option>
						</select>
					</div>

					<div>
						<label for="sh-token" class="block text-[12px] font-semibold text-[var(--tx-2)] mb-[6px] tracking-[.03em]">PERSONAL ACCESS TOKEN</label>
						<input id="sh-token" bind:value={shToken} type="password" placeholder="glpat-xxxxxxxxxxxxxxxxxxxx" class={panelInputMonoCls} />
						<div class="text-[11.5px] text-[var(--tx-3)] mt-[5px]">Needs <code class="font-mono-jb bg-white/[.06] px-1 py-[1px] rounded-[3px]">read_repository</code> scope. Stored encrypted at rest.</div>
					</div>

					<div>
						<button
							onclick={testConnection}
							disabled={testing}
							class="flex items-center gap-[7px] bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[14px] py-[9px] text-[13px] font-semibold text-[var(--tx-2)] cursor-pointer {testing ? 'opacity-60' : ''}"
						>
							{#if testing}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="2" class="animate-[bk-spin_1s_linear_infinite]"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
								Testing connection…
							{:else if tested}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
								<span class="text-[var(--grn-2)]">Connection successful</span>
							{:else}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
								Test connection
							{/if}
						</button>
						{#if testError}
							<div class="mt-[7px] text-[12px] text-[#f0836b]">{testError}</div>
						{/if}
					</div>

					<button
						onclick={finishConnection}
						disabled={!canConnect}
						class="w-full flex items-center justify-center gap-2 bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold shadow-[0_2px_12px_var(--grn-dim)] {!canConnect ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
					>
						Connect source
					</button>
				</div>

			{:else if step === 'done'}
				<div class="text-center py-6">
					<div class="size-14 rounded-full bg-[var(--grn-dim)] border border-[var(--grn-line)] flex items-center justify-center mx-auto mb-4">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div class="font-heading text-[18px] font-bold text-[var(--tx)] mb-[6px]">Source connected!</div>
					<div class="text-[13px] text-[var(--tx-3)] mb-7">
						{shName || 'Your GitHub App'} is now linked. Bakery found the following repos:
					</div>

					<div class="bg-[var(--card)] border border-[var(--line)] rounded-[10px] overflow-hidden text-left mb-6">
						{#each REPOS as r (r.id)}
							<div class="flex items-center gap-[10px] px-[14px] py-[10px] border-b border-b-[var(--line)] last:border-b-0">
								<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.8"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
								<div class="flex-1 text-[12.5px] text-[var(--tx)]">{r.name}</div>
								<div class="size-[9px] rounded-full" style:background={r.langColor}></div>
								<span class="text-[11.5px] text-[var(--tx-3)]">{r.lang}</span>
							</div>
						{/each}
					</div>

					<button onclick={closePanel} class="w-full flex items-center justify-center bg-[var(--grn)] text-[#07130c] rounded-[9px] py-[11px] text-[14px] font-bold cursor-pointer shadow-[0_2px_12px_var(--grn-dim)]">Done</button>
				</div>
			{/if}
		</div>
	</div>
{/if}

{#snippet ProviderIcon({ icon, size = 16, color }: { icon: string; size?: number; color?: string })}
	{#if icon === 'github'}
		<svg width={size} height={size} viewBox="0 0 24 24" fill={color ?? 'var(--tx-2)'}>
			<path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
		</svg>
	{:else if icon === 'gitlab'}
		<svg width={size} height={size} viewBox="0 0 24 24" fill={color ?? '#fc6d26'}>
			<path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 4.82 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.49h8.1l2.44-7.51a.42.42 0 0 1 .11-.18.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.51L23 13.45a.84.84 0 0 1-.35.94z"/>
		</svg>
	{:else if icon === 'gitea'}
		<svg width={size} height={size} viewBox="0 0 24 24" fill={color ?? 'var(--grn-2)'}>
			<path d="M3.08 11.98C1.82 8.9 3.07 5.37 6 3.75c2.93-1.62 6.63-.73 8.5 2.03C16.36 3.02 20.06 2.13 23 3.75 25.93 5.37 27.18 8.9 25.92 11.98L14 22 2.08 11.98z" transform="scale(0.857)"/>
		</svg>
	{:else if icon === 'bitbucket'}
		<svg width={size} height={size} viewBox="0 0 24 24" fill={color ?? '#5b8def'}>
			<path d="M.778 1.213a.768.768 0 0 0-.768.892l3.263 19.81c.084.5.515.865 1.022.865h15.007a.77.77 0 0 0 .768-.646l3.263-19.83a.768.768 0 0 0-.768-.891zM14.81 15.538H9.21L7.762 8.465h8.5z"/>
		</svg>
	{/if}
{/snippet}
