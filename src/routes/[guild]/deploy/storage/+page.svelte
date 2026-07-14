<script lang="ts">
	import { page } from '$app/state';
	import { formatBytes } from '$lib/utils';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const guildId = $derived(page.params.guild ?? '');
	const totalSize = $derived(data.volumes.reduce((n, v) => n + v.sizeBytes, 0));
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-start gap-[14px] mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Storage</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{#if data.volumes.length > 0}
					{data.volumes.length} volume{data.volumes.length !== 1 ? 's' : ''} · {formatBytes(
						totalSize
					)}
				{:else}
					Persistent volumes for your container deployments.
				{/if}
			</div>
		</div>
	</div>

	{#if data.volumes.length === 0}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[40px_32px]">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)] mb-[6px]">No volumes yet</div>
			<div class="text-[13px] text-[var(--tx-3)]">
				Bakery manages Podman-local volumes automatically — attach one from an app's Storage tab to
				persist its data across redeploys. There's no driver to choose or volume to create here.
			</div>
		</div>
	{:else}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div
				class="grid grid-cols-[1.4fr_100px_2fr_1fr_1fr] gap-3 px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]"
			>
				<div>VOLUME</div>
				<div>SIZE</div>
				<div>MOUNT PATH</div>
				<div>APP</div>
				<div>HOST</div>
			</div>

			{#each data.volumes as vol (vol.id)}
				<div
					class="group px-[18px] py-[13px] grid grid-cols-[1.4fr_100px_2fr_1fr_1fr] gap-3 items-center border-b border-b-[var(--line)] last:border-b-0 hover:bg-white/[0.02]"
				>
					<div class="flex items-center gap-[11px] min-w-0">
						<div
							class="size-[34px] rounded-[9px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0"
						>
							{@render VolumeIcon()}
						</div>
						<div
							class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis"
						>
							{vol.name}
						</div>
					</div>

					<div class="font-mono-jb text-[12.5px] text-[var(--tx-2)]">
						{formatBytes(vol.sizeBytes)}
					</div>

					<div
						class="font-mono-jb text-[11px] text-[var(--tx-3)] whitespace-nowrap overflow-hidden text-ellipsis"
						title={vol.mountPath}
					>
						{vol.mountPath}
					</div>

					<div class="min-w-0">
						<a
							href="/{guildId}/deploy/projects/{vol.appId}"
							class="text-[12px] text-[var(--tx-2)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px] whitespace-nowrap no-underline"
							>{vol.appName}</a
						>
					</div>

					<div class="text-[12.5px] text-[var(--tx-2)]">{vol.hostName}</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

{#snippet VolumeIcon()}
	<svg
		width="16"
		height="16"
		viewBox="0 0 24 24"
		fill="none"
		stroke="var(--tx-3)"
		stroke-width="1.6"
	>
		<ellipse cx="12" cy="6" rx="7" ry="3" />
		<path d="M5 6v12c0 1.6 3.1 3 7 3s7-1.4 7-3V6" />
		<path d="M5 12c0 1.6 3.1 3 7 3s7-1.4 7-3" />
	</svg>
{/snippet}
