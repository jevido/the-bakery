<script lang="ts">
	import { superForm } from 'sveltekit-superforms';
	import { zodClient } from '$lib/forms/zod4-adapter';
	import { Field, Control, Label, FieldErrors } from 'formsnap';
	import { z } from 'zod';
	import { page } from '$app/state';

	let { data } = $props();

	const newAppSchema = z.object({
		repoId: z.string().uuid('Pick a repository'),
		name: z.string().trim().min(1, 'App name is required').max(100),
		buildContext: z.string().trim().min(1).max(300).default('.')
	});

	const form = superForm(data.form, { validators: zodClient(newAppSchema) });
	const { form: formData, enhance, submitting } = form;

	const guildId = $derived(page.params.guild ?? '');
	const selectedRepo = $derived(data.repos.find((r) => r.id === $formData.repoId));

	function pickRepo(repoId: string) {
		$formData.repoId = repoId;
		if (!$formData.name) {
			const repo = data.repos.find((r) => r.id === repoId);
			if (repo) $formData.name = repo.fullName.split('/').pop() ?? repo.fullName;
		}
	}

	type Detection =
		| { state: 'idle' }
		| { state: 'checking' }
		| { state: 'found'; filename: string; path: string }
		| { state: 'not-found' };
	let detection = $state<Detection>({ state: 'idle' });

	$effect(() => {
		const repoId = $formData.repoId;
		const buildContext = $formData.buildContext || '.';
		if (!repoId) {
			detection = { state: 'idle' };
			return;
		}

		detection = { state: 'checking' };
		const controller = new AbortController();
		const timer = setTimeout(async () => {
			try {
				const res = await fetch(
					`/api/v1/repos/${repoId}/detect-build-file?buildContext=${encodeURIComponent(buildContext)}`,
					{ signal: controller.signal }
				);
				const body = (await res.json()) as { detected: { filename: string; path: string } | null };
				detection = body.detected ? { state: 'found', ...body.detected } : { state: 'not-found' };
			} catch {
				if (!controller.signal.aborted) detection = { state: 'not-found' };
			}
		}, 400);

		return () => {
			clearTimeout(timer);
			controller.abort();
		};
	});
</script>

<svelte:head><title>New app — The Bakery</title></svelte:head>

<div class="px-7 py-[22px] max-w-[640px]">
	<div class="font-heading font-bold text-[23px] mb-1">New app</div>
	<div class="text-[13px] text-[var(--tx-2)] mb-[22px]">
		Pick a repository and build context — you can trigger a build once it's created.
	</div>

	<form method="post" action="?/create" use:enhance>
		<div
			class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden mb-[14px]"
		>
			<div
				class="px-[18px] py-3 border-b border-b-[var(--line)] text-[12px] font-bold tracking-[.05em] text-[var(--tx-3)]"
			>
				SOURCE
			</div>
			<div class="p-[18px]">
				<Field {form} name="repoId">
					<Control>
						{#snippet children({ props })}
							<Label class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]"
								>Repository</Label
							>
							<input {...props} type="hidden" bind:value={$formData.repoId} />
							{#if data.repos.length === 0}
								<div class="text-[13px] text-[var(--tx-2)]">
									No repositories yet — connect GitHub on the
									<a href="/{guildId}/deploy/sources" class="text-[var(--grn-2)]">Sources page</a> first.
								</div>
							{:else}
								{#each data.repos as r (r.id)}
									{@const sel = $formData.repoId === r.id}
									<button
										type="button"
										onclick={() => pickRepo(r.id)}
										class="flex items-center gap-3 w-full px-[13px] py-[11px] rounded-[9px] mb-[6px] cursor-pointer border {sel
											? 'bg-[var(--grn-dim)] border-[var(--grn-line)]'
											: 'bg-[var(--card-2)] border-[var(--line)]'}"
									>
										<div class="flex-1 min-w-0 text-left">
											<div class="font-mono-jb text-[13px] font-medium text-[var(--tx)]">
												{r.fullName}
											</div>
											<div class="text-[11.5px] text-[var(--tx-3)]">
												default branch: {r.defaultBranch}
											</div>
										</div>
										<div
											class="size-[18px] rounded-full border-2 flex items-center justify-center shrink-0"
											style:border-color={sel ? 'var(--grn)' : 'var(--tx-3)'}
										>
											{#if sel}<div class="size-2 rounded-full bg-[var(--grn)]"></div>{/if}
										</div>
									</button>
								{/each}
							{/if}
						{/snippet}
					</Control>
					<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
				</Field>

				<div class="mt-[14px]">
					<Field {form} name="name">
						<Control>
							{#snippet children({ props })}
								<Label class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]"
									>App name</Label
								>
								<input
									{...props}
									bind:value={$formData.name}
									class="w-full bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] text-[14px] text-[var(--tx)]"
								/>
							{/snippet}
						</Control>
						<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
					</Field>
				</div>

				<div class="mt-[14px]">
					<Field {form} name="buildContext">
						<Control>
							{#snippet children({ props })}
								<Label class="block text-[13px] font-semibold text-[var(--tx-2)] mb-[6px]"
									>Build context</Label
								>
								<input
									{...props}
									bind:value={$formData.buildContext}
									placeholder="."
									class="w-full bg-[var(--card-2)] border border-[var(--line-2)] rounded-[8px] px-[13px] py-[10px] font-mono-jb text-[13px] text-[var(--tx)]"
								/>
								<div class="text-[11.5px] text-[var(--tx-3)] mt-[6px]">
									Subdirectory to build from — use <span class="font-mono-jb">.</span> for the repo root.
								</div>
							{/snippet}
						</Control>
						<FieldErrors class="text-[12px] text-[#f0836b] mt-1" />
					</Field>
				</div>

				{#if $formData.repoId}
					<div
						class="mt-[14px] flex items-center gap-[9px] rounded-[9px] px-[13px] py-[10px] text-[12.5px] {detection.state ===
						'found'
							? 'bg-[var(--grn-dim)] border border-[var(--grn-line)] text-[var(--grn-2)]'
							: detection.state === 'not-found'
								? 'bg-[rgba(229,101,75,.08)] border border-[rgba(229,101,75,.2)] text-[#f0836b]'
								: 'bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx-2)]'}"
					>
						{#if detection.state === 'checking'}
							Checking for a Dockerfile or Containerfile…
						{:else if detection.state === 'found'}
							Found <span class="font-mono-jb">{detection.path}</span>
						{:else if detection.state === 'not-found'}
							No Dockerfile or Containerfile found at branch {selectedRepo?.defaultBranch ?? ''} / {$formData.buildContext ||
								'.'}
						{/if}
					</div>
				{/if}
			</div>
		</div>

		<div class="flex justify-end">
			<button
				type="submit"
				disabled={$submitting || data.repos.length === 0}
				class="px-4 py-[10px] rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13.5px] font-bold cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
				>{$submitting ? 'Creating…' : 'Create app'}</button
			>
		</div>
	</form>
</div>
