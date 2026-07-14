<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const totalApps = $derived(data.networks.reduce((n, net) => n + net.apps.length, 0));
</script>

<div class="px-7 py-[22px]">
	<!-- Header -->
	<div class="flex items-start gap-[14px] mb-[22px]">
		<div>
			<div class="font-heading font-bold text-[23px] tracking-[-0.01em] mb-[3px]">Networks</div>
			<div class="text-[13px] text-[var(--tx-2)]">
				{#if data.networks.length > 0}
					{data.networks.length} network{data.networks.length !== 1 ? 's' : ''} · {totalApps} app{totalApps !==
					1
						? 's'
						: ''} attached
				{:else}
					Bakery manages one bridge network per host automatically.
				{/if}
			</div>
		</div>
	</div>

	{#if data.networks.length === 0}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] p-[40px_32px]">
			<div class="font-heading text-[16px] font-bold text-[var(--tx)] mb-[6px]">
				No networks yet
			</div>
			<div class="text-[13px] text-[var(--tx-3)]">
				Every host gets a single managed bridge network automatically once you deploy your first app
				to it — apps in this guild can then reach each other by name. There's no driver to choose or
				network to create.
			</div>
		</div>
	{:else}
		<div class="bg-[var(--card)] border border-[var(--line)] rounded-[13px] overflow-hidden">
			<div
				class="grid grid-cols-[1.6fr_2fr_1.2fr] gap-3 px-[18px] py-[11px] border-b border-b-[var(--line)] text-[11px] font-bold tracking-[.05em] text-[var(--tx-3)]"
			>
				<div>NETWORK</div>
				<div>APPS</div>
				<div>HOST</div>
			</div>

			{#each data.networks as net (net.hostId)}
				<div
					class="px-[18px] py-[13px] grid grid-cols-[1.6fr_2fr_1.2fr] gap-3 items-center border-b border-b-[var(--line)] last:border-b-0"
				>
					<div class="flex items-center gap-[11px] min-w-0">
						<div
							class="size-[34px] rounded-[9px] bg-white/5 border border-[var(--line)] flex items-center justify-center shrink-0"
						>
							<svg
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="var(--tx-3)"
								stroke-width="1.6"
							>
								<circle cx="12" cy="5" r="2" /><circle cx="5" cy="19" r="2" /><circle
									cx="19"
									cy="19"
									r="2"
								/>
								<path d="M12 7v4M12 11l-5 6M12 11l5 6" />
							</svg>
						</div>
						<div class="min-w-0">
							<div
								class="text-[14px] font-semibold text-[var(--tx)] whitespace-nowrap overflow-hidden text-ellipsis"
							>
								{net.name}
							</div>
							<div class="text-[11px] text-[var(--tx-3)]">bridge · managed</div>
						</div>
					</div>

					<div class="flex flex-wrap gap-1 min-w-0">
						{#each net.apps as appName (appName)}
							<span
								class="text-[11px] text-[var(--tx-2)] bg-[var(--card-2)] border border-[var(--line)] rounded-[4px] px-[6px] py-[2px] whitespace-nowrap"
								>{appName}</span
							>
						{/each}
					</div>

					<div class="text-[12.5px] text-[var(--tx-2)]">{net.hostName}</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
