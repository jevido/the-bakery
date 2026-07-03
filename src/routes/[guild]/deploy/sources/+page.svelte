<script lang="ts">
	import { page } from '$app/state';
	import { SOURCES, REPOS, sourceMeta } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const sources = $derived(SOURCES[guildId] ?? []);
	const totalRepos = $derived(sources.reduce((n, s) => n + s.repoCount, 0));

	let filter = $state<'all' | 'github' | 'selfhosted'>('all');

	const filtered = $derived(
		filter === 'all' ? sources
		: filter === 'github' ? sources.filter((s) => s.provider === 'github_app')
		: sources.filter((s) => s.provider !== 'github_app')
	);

	function filterStyle(f: typeof filter) {
		return f === filter
			? 'padding:5px 11px;font-size:12.5px;color:var(--tx);border-radius:6px;cursor:pointer;background:var(--card-2);'
			: 'padding:5px 11px;font-size:12.5px;color:var(--tx-2);border-radius:6px;cursor:pointer;';
	}

	// ── Add source panel ──────────────────────────────────────────────
	let panelOpen = $state(false);
	let step = $state<'pick' | 'github' | 'selfhosted' | 'done'>('pick');

	// Self-hosted form state
	let shName = $state('');
	let shUrl = $state('');
	let shProvider = $state<'gitlab' | 'gitea' | 'bitbucket' | 'generic'>('gitlab');
	let shToken = $state('');
	let testing = $state(false);
	let tested = $state(false);
	let testError = $state('');

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

	function closePanel() {
		panelOpen = false;
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
</script>

<div style="padding: 22px 28px;">
	<!-- Header -->
	<div style="display:flex;align-items:flex-start;gap:14px;margin-bottom:22px;">
		<div>
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;letter-spacing:-.01em;margin-bottom:3px;">Sources</div>
			<div style="font-size:13px;color:var(--tx-2);">
				{#if sources.length > 0}
					{sources.length} source{sources.length !== 1 ? 's' : ''} · {totalRepos} repos accessible
				{:else}
					Connect repositories and container registries as deployment sources.
				{/if}
			</div>
		</div>
		<div style="flex:1"></div>
		<button onclick={openPanel} style="display:flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border-radius:9px;padding:8px 14px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
			Add source
		</button>
	</div>

	{#if sources.length === 0}
		<!-- Empty / onboarding state -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;padding:40px 32px;">
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);margin-bottom:6px;">No sources connected</div>
			<div style="font-size:13px;color:var(--tx-3);margin-bottom:32px;">Follow three steps to connect your first git source and start pulling projects.</div>

			<!-- 3-step guide -->
			<div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:16px;margin-bottom:32px;">
				{#each [
					{ n: '1', title: 'Choose a provider', desc: 'Select GitHub App for cloud repos, or self-hosted for GitLab, Gitea, or Bitbucket Server.' },
					{ n: '2', title: 'Install & authorize', desc: 'Install the Bakery GitHub App on your org, or provide a personal access token for self-hosted.' },
					{ n: '3', title: 'Pick repos', desc: 'Choose which repositories Bakery can read and deploy from.' },
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

			<!-- Provider pills -->
			<div style="display:flex;flex-wrap:wrap;gap:8px;margin-bottom:28px;">
				{#each [
					{ label: 'GitHub App',       icon: 'github'    },
					{ label: 'GitLab',           icon: 'gitlab'    },
					{ label: 'Gitea',            icon: 'gitea'     },
					{ label: 'Bitbucket Server', icon: 'bitbucket' },
				] as p (p.icon)}
					<div style="display:flex;align-items:center;gap:7px;background:var(--card-2);border:1px solid var(--line);border-radius:7px;padding:6px 12px;">
						{@render ProviderIcon({ icon: p.icon, size: 14 })}
						<span style="font-size:12.5px;color:var(--tx-2);">{p.label}</span>
					</div>
				{/each}
			</div>

			<button onclick={openPanel} style="display:inline-flex;align-items:center;gap:7px;background:var(--grn);color:#07130c;border-radius:9px;padding:9px 18px;font-size:13.5px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 5v14M5 12h14"/></svg>
				Connect your first source
			</button>
		</div>

	{:else}
		<!-- Filter tabs -->
		<div style="display:flex;align-items:center;gap:14px;margin-bottom:14px;">
			<div style="display:flex;gap:2px;background:var(--card);border:1px solid var(--line);border-radius:9px;padding:3px;">
				<button onclick={() => filter = 'all'} style={filterStyle('all')}>All</button>
				<button onclick={() => filter = 'github'} style={filterStyle('github')}>GitHub</button>
				<button onclick={() => filter = 'selfhosted'} style={filterStyle('selfhosted')}>Self-hosted</button>
			</div>
		</div>

		<!-- Sources table -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
			<!-- Header row -->
			<div style="display:grid;grid-template-columns:2fr 1.2fr 80px 120px 100px;gap:14px;padding:11px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
				<div>SOURCE</div>
				<div>PROVIDER</div>
				<div>REPOS</div>
				<div>CONNECTED</div>
				<div></div>
			</div>

			{#each filtered as s (s.id)}
				{@const m = sourceMeta(s.provider)}
				<div class="source-row" style="padding:13px 18px;display:grid;grid-template-columns:2fr 1.2fr 80px 120px 100px;gap:14px;align-items:center;border-bottom:1px solid var(--line);">
					<!-- Name + host -->
					<div style="display:flex;align-items:center;gap:11px;min-width:0;">
						<div style="width:34px;height:34px;border-radius:9px;background:rgba(255,255,255,.05);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;flex:none;">
							{@render ProviderIcon({ icon: s.provider === 'github_app' ? 'github' : s.provider, size: 17 })}
						</div>
						<div style="min-width:0;">
							<div style="font-size:14px;font-weight:600;color:var(--tx);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{s.name}</div>
							{#if s.host}
								<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--tx-3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">{s.host}</div>
							{:else}
								<div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--tx-3);">github.com</div>
							{/if}
						</div>
					</div>

					<!-- Provider badge -->
					<div>
						<span style="font-size:12px;font-weight:600;color:{m.color};background:{m.bg};border-radius:5px;padding:3px 8px;">{m.label}</span>
					</div>

					<!-- Repo count -->
					<div style="font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx-2);">{s.repoCount}</div>

					<!-- Connected at -->
					<div style="font-size:12.5px;color:var(--tx-3);">{s.connectedAt}</div>

					<!-- Configure button -->
					<div style="text-align:right;">
						<button class="configure-btn" style="font-size:12.5px;color:var(--tx-3);cursor:pointer;padding:4px 8px;border-radius:6px;">Configure →</button>
					</div>
				</div>
			{/each}

			{#if filtered.length === 0}
				<div style="padding:40px;text-align:center;color:var(--tx-3);font-size:13.5px;">No sources match the current filter.</div>
			{/if}
		</div>

		<!-- Repo list preview -->
		{#if guildId === 'sourdough'}
			<div style="margin-top:20px;">
				<div style="font-size:12px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);margin-bottom:10px;">REPOS FROM SOURDOUGH-LABS</div>
				<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
					{#each REPOS as r (r.id)}
						<div style="display:flex;align-items:center;gap:14px;padding:11px 18px;border-bottom:1px solid var(--line);">
							<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.6"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
							<div style="flex:1;font-size:13px;color:var(--tx);">{r.name}</div>
							<div style="display:flex;align-items:center;gap:5px;">
								<div style="width:10px;height:10px;border-radius:50%;background:{r.langColor};"></div>
								<span style="font-size:12px;color:var(--tx-3);">{r.lang}</span>
							</div>
							<div style="font-size:12px;color:var(--tx-3);width:110px;text-align:right;">{r.updated}</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>

<!-- ── Add Source Panel ─────────────────────────────────────────────── -->
{#if panelOpen}
	<!-- Backdrop -->
	<div
		role="button"
		tabindex="-1"
		onclick={closePanel}
		onkeydown={(e) => e.key === 'Escape' && closePanel()}
		style="position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:200;"
	></div>

	<!-- Slide panel -->
	<div style="position:fixed;top:0;right:0;width:480px;height:100vh;background:var(--panel);border-left:1px solid var(--line);z-index:201;display:flex;flex-direction:column;overflow:hidden;">
		<!-- Panel header -->
		<div style="padding:20px 24px 16px;border-bottom:1px solid var(--line);display:flex;align-items:center;gap:12px;">
			{#if step !== 'pick'}
				<button onclick={() => { step = 'pick'; tested = false; testError = ''; }} style="color:var(--tx-3);cursor:pointer;font-size:12.5px;display:flex;align-items:center;gap:4px;">
					<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 5l-7 7 7 7"/></svg>
					Back
				</button>
				<div style="width:1px;height:14px;background:var(--line);"></div>
			{/if}
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:16px;font-weight:700;color:var(--tx);">
				{step === 'pick' ? 'Add source' : step === 'github' ? 'GitHub App' : step === 'selfhosted' ? 'Self-hosted Git' : 'Source connected'}
			</div>
			<div style="flex:1"></div>
			<button onclick={closePanel} aria-label="Close panel" style="color:var(--tx-3);cursor:pointer;display:flex;align-items:center;">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
			</button>
		</div>

		<!-- Panel body -->
		<div style="flex:1;overflow-y:auto;padding:24px;">

			{#if step === 'pick'}
				<!-- Provider picker -->
				<div style="font-size:13px;color:var(--tx-3);margin-bottom:20px;">Choose where your code lives. You can add multiple sources.</div>

				<!-- GitHub App card -->
				<button onclick={() => step = 'github'} class="provider-card" style="width:100%;text-align:left;background:var(--card);border:1px solid var(--line);border-radius:11px;padding:18px;margin-bottom:10px;cursor:pointer;display:flex;align-items:flex-start;gap:14px;">
					<div style="width:40px;height:40px;border-radius:10px;background:rgba(255,255,255,.06);display:flex;align-items:center;justify-content:center;flex:none;margin-top:1px;">
						{@render ProviderIcon({ icon: 'github', size: 20 })}
					</div>
					<div>
						<div style="font-size:14px;font-weight:600;color:var(--tx);margin-bottom:4px;">GitHub App</div>
						<div style="font-size:12.5px;color:var(--tx-3);line-height:1.5;">Install once on your GitHub org or personal account. Bakery gets fine-grained access to selected repos — no secrets stored.</div>
						<div style="margin-top:10px;display:flex;gap:6px;">
							<span style="font-size:11px;color:var(--grn-2);background:var(--grn-dim);border-radius:4px;padding:2px 7px;">Recommended</span>
							<span style="font-size:11px;color:var(--tx-3);background:rgba(255,255,255,.05);border-radius:4px;padding:2px 7px;">github.com</span>
						</div>
					</div>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="2" style="margin-left:auto;flex:none;margin-top:2px;"><path d="M9 18l6-6-6-6"/></svg>
				</button>

				<!-- Self-hosted card -->
				<button onclick={() => step = 'selfhosted'} class="provider-card" style="width:100%;text-align:left;background:var(--card);border:1px solid var(--line);border-radius:11px;padding:18px;cursor:pointer;display:flex;align-items:flex-start;gap:14px;">
					<div style="width:40px;height:40px;border-radius:10px;background:rgba(255,255,255,.06);display:flex;align-items:center;justify-content:center;flex:none;margin-top:1px;">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--tx-2)" stroke-width="1.6"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
					</div>
					<div>
						<div style="font-size:14px;font-weight:600;color:var(--tx);margin-bottom:4px;">Self-hosted Git</div>
						<div style="font-size:12.5px;color:var(--tx-3);line-height:1.5;">Connect a GitLab, Gitea, or Bitbucket Server instance running on your own infrastructure using a personal access token.</div>
						<div style="margin-top:10px;display:flex;gap:6px;">
							<span style="font-size:11px;color:var(--tx-3);background:rgba(255,255,255,.05);border-radius:4px;padding:2px 7px;">GitLab</span>
							<span style="font-size:11px;color:var(--tx-3);background:rgba(255,255,255,.05);border-radius:4px;padding:2px 7px;">Gitea</span>
							<span style="font-size:11px;color:var(--tx-3);background:rgba(255,255,255,.05);border-radius:4px;padding:2px 7px;">Bitbucket Server</span>
						</div>
					</div>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="2" style="margin-left:auto;flex:none;margin-top:2px;"><path d="M9 18l6-6-6-6"/></svg>
				</button>

			{:else if step === 'github'}
				<!-- GitHub App instructions -->
				<div style="background:var(--grn-dim);border:1px solid var(--grn-line);border-radius:10px;padding:14px 16px;margin-bottom:24px;display:flex;gap:10px;">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2" style="flex:none;margin-top:1px;"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>
					<div style="font-size:12.5px;color:var(--grn-2);line-height:1.5;">The Bakery GitHub App requests <strong>read-only</strong> access to contents and metadata. No write permissions are ever requested.</div>
				</div>

				<div style="margin-bottom:24px;">
					{#each [
						{ n: '1', label: 'Click "Install on GitHub" below', detail: 'You will be redirected to GitHub to install the Bakery App on your account or organisation.' },
						{ n: '2', label: 'Select repositories', detail: 'Choose "All repositories" or specific repos you want Bakery to access.' },
						{ n: '3', label: 'Return here', detail: 'GitHub will redirect you back and your source will appear in the list automatically.' },
					] as inst (inst.n)}
						<div style="display:flex;gap:14px;margin-bottom:16px;">
							<div style="width:24px;height:24px;border-radius:50%;background:var(--card-2);border:1px solid var(--line-2);display:flex;align-items:center;justify-content:center;flex:none;margin-top:1px;">
								<span style="font-family:'JetBrains Mono',monospace;font-size:11px;font-weight:700;color:var(--tx-2);">{inst.n}</span>
							</div>
							<div>
								<div style="font-size:13.5px;font-weight:600;color:var(--tx);margin-bottom:3px;">{inst.label}</div>
								<div style="font-size:12px;color:var(--tx-3);line-height:1.5;">{inst.detail}</div>
							</div>
						</div>
					{/each}
				</div>

				<button style="width:100%;display:flex;align-items:center;justify-content:center;gap:8px;background:var(--grn);color:#07130c;border-radius:9px;padding:11px;font-size:14px;font-weight:700;cursor:pointer;box-shadow:0 2px 12px var(--grn-dim);">
					{@render ProviderIcon({ icon: 'github', size: 16, color: '#07130c' })}
					Install on GitHub →
				</button>

				<div style="margin-top:14px;text-align:center;">
					<button onclick={() => step = 'done'} style="font-size:12px;color:var(--tx-3);cursor:pointer;text-decoration:underline;">I've already installed it — continue</button>
				</div>

			{:else if step === 'selfhosted'}
				<!-- Self-hosted form -->
				<div style="display:flex;flex-direction:column;gap:16px;">
					<!-- Display name -->
					<div>
						<label for="sh-name" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">DISPLAY NAME</label>
						<input
							id="sh-name"
							bind:value={shName}
							placeholder="my-gitlab"
							style="width:100%;box-sizing:border-box;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;"
						/>
					</div>

					<!-- Base URL -->
					<div>
						<label for="sh-url" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">BASE URL</label>
						<input
							id="sh-url"
							bind:value={shUrl}
							placeholder="https://git.example.com"
							style="width:100%;box-sizing:border-box;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
						/>
					</div>

					<!-- Provider type -->
					<div>
						<label for="sh-provider" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">PROVIDER</label>
						<select
							id="sh-provider"
							bind:value={shProvider}
							style="width:100%;box-sizing:border-box;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'Instrument Sans',sans-serif;outline:none;appearance:none;cursor:pointer;"
						>
							<option value="gitlab">GitLab</option>
							<option value="gitea">Gitea</option>
							<option value="bitbucket">Bitbucket Server</option>
							<option value="generic">Generic Git (SSH)</option>
						</select>
					</div>

					<!-- Personal access token -->
					<div>
						<label for="sh-token" style="display:block;font-size:12px;font-weight:600;color:var(--tx-2);margin-bottom:6px;letter-spacing:.03em;">PERSONAL ACCESS TOKEN</label>
						<input
							id="sh-token"
							bind:value={shToken}
							type="password"
							placeholder="glpat-xxxxxxxxxxxxxxxxxxxx"
							style="width:100%;box-sizing:border-box;background:var(--card);border:1px solid var(--line-2);border-radius:8px;padding:9px 12px;font-size:13.5px;color:var(--tx);font-family:'JetBrains Mono',monospace;outline:none;"
						/>
						<div style="font-size:11.5px;color:var(--tx-3);margin-top:5px;">Needs <code style="font-family:'JetBrains Mono',monospace;background:rgba(255,255,255,.06);padding:1px 4px;border-radius:3px;">read_repository</code> scope. Stored encrypted at rest.</div>
					</div>

					<!-- Test connection -->
					<div>
						<button
							onclick={testConnection}
							disabled={testing}
							style="display:flex;align-items:center;gap:7px;background:var(--card-2);border:1px solid var(--line-2);border-radius:8px;padding:9px 14px;font-size:13px;font-weight:600;color:var(--tx-2);cursor:pointer;{testing ? 'opacity:.6;' : ''}"
						>
							{#if testing}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--grn)" stroke-width="2" style="animation:bk-spin 1s linear infinite;"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
								Testing connection…
							{:else if tested}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
								<span style="color:var(--grn-2);">Connection successful</span>
							{:else}
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>
								Test connection
							{/if}
						</button>
						{#if testError}
							<div style="margin-top:7px;font-size:12px;color:#f0836b;">{testError}</div>
						{/if}
					</div>

					<!-- Connect button -->
					<button
						onclick={() => step = 'done'}
						disabled={!canConnect}
						style="width:100%;display:flex;align-items:center;justify-content:center;gap:8px;background:var(--grn);color:#07130c;border-radius:9px;padding:11px;font-size:14px;font-weight:700;box-shadow:0 2px 12px var(--grn-dim);{!canConnect ? 'opacity:.4;cursor:not-allowed;' : 'cursor:pointer;'}"
					>
						Connect source
					</button>
				</div>

			{:else if step === 'done'}
				<!-- Success -->
				<div style="text-align:center;padding:24px 0;">
					<div style="width:56px;height:56px;border-radius:50%;background:var(--grn-dim);border:1px solid var(--grn-line);display:flex;align-items:center;justify-content:center;margin:0 auto 16px;">
						<svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="var(--grn-2)" stroke-width="2.5"><path d="M20 6L9 17l-5-5"/></svg>
					</div>
					<div style="font-family:'Bricolage Grotesque',sans-serif;font-size:18px;font-weight:700;color:var(--tx);margin-bottom:6px;">Source connected!</div>
					<div style="font-size:13px;color:var(--tx-3);margin-bottom:28px;">
						{shName || 'Your GitHub App'} is now linked. Bakery found the following repos:
					</div>

					<div style="background:var(--card);border:1px solid var(--line);border-radius:10px;overflow:hidden;text-align:left;margin-bottom:24px;">
						{#each REPOS as r (r.id)}
							<div style="display:flex;align-items:center;gap:10px;padding:10px 14px;border-bottom:1px solid var(--line);">
								<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--tx-3)" stroke-width="1.8"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
								<div style="flex:1;font-size:12.5px;color:var(--tx);">{r.name}</div>
								<div style="width:9px;height:9px;border-radius:50%;background:{r.langColor};"></div>
								<span style="font-size:11.5px;color:var(--tx-3);">{r.lang}</span>
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

<!-- ── Provider icon snippet ───────────────────────────────────────── -->
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
	.source-row:hover {
		background: rgba(255,255,255,.02);
	}
	.source-row:last-child {
		border-bottom: none;
	}
	.configure-btn {
		opacity: 0;
		transition: opacity 0.15s;
	}
	.source-row:hover .configure-btn {
		opacity: 1;
	}
	.provider-card:hover {
		background: var(--card-2) !important;
		border-color: var(--line-2) !important;
	}
</style>
