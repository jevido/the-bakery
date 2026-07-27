<script lang="ts">
	import { AreaChart } from 'layerchart';
	import { ChartContainer, ChartTooltip, type ChartConfig } from '$lib/components/ui/chart';
	import { useMetricChartCrosshair } from './metric-chart-crosshair.svelte';

	interface MetricPoint {
		ts: Date;
		value: number;
	}

	let {
		series,
		color,
		label,
		unit = ''
	}: {
		series: MetricPoint[];
		color: string;
		label: string;
		unit?: string;
	} = $props();

	const config: ChartConfig = $derived({ value: { label, color } });

	// Present only when a page-level provider exists (`provideMetricChartCrosshair`,
	// called once near the top of a page hosting multiple charts) -- a
	// standalone MetricChart used without one still works, just without
	// cross-chart synchronization.
	const crosshair = useMetricChartCrosshair();

	let containerEl = $state<HTMLDivElement>();
	let hovering = $state(false);

	function indexFromPointer(clientX: number): number | null {
		if (!containerEl || series.length === 0) return null;
		const rect = containerEl.getBoundingClientRect();
		if (rect.width === 0) return null;
		const pct = Math.min(1, Math.max(0, (clientX - rect.left) / rect.width));
		return Math.round(pct * (series.length - 1));
	}

	function onPointerMove(e: PointerEvent) {
		hovering = true;
		if (!crosshair) return;
		const idx = indexFromPointer(e.clientX);
		if (idx !== null) crosshair.hoveredIndex = idx;
	}
	function onPointerLeave() {
		hovering = false;
		if (crosshair) crosshair.hoveredIndex = null;
	}

	// Only rendered while *another* chart is driving the shared index --
	// this chart's own native layerchart highlight/tooltip already marks
	// the hovered position while it's the one directly under the pointer,
	// so overlaying both here would be redundant.
	const sharedIndex = $derived(crosshair && !hovering ? crosshair.hoveredIndex : null);
	const sharedPct = $derived(
		sharedIndex !== null && series.length > 1 ? (sharedIndex / (series.length - 1)) * 100 : null
	);

	function formatTimestamp(value: unknown): string {
		if (!(value instanceof Date)) return String(value);
		return value.toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div
	class="relative h-full w-full"
	role="presentation"
	bind:this={containerEl}
	onpointermove={onPointerMove}
	onpointerleave={onPointerLeave}
>
	{#if series.length > 1}
		<ChartContainer {config} class="aspect-auto h-full w-full">
			<AreaChart
				data={series}
				x="ts"
				y="value"
				yDomain={[0, null]}
				axis={false}
				grid={false}
				legend={false}
				rule={true}
				tooltipContext={{ mode: 'bisect-x' }}
				highlight={{ points: true, lines: true }}
				padding={{ top: 4, bottom: 4 }}
				props={{
					area: {
						fillOpacity: 0.13,
						fill: color,
						line: { stroke: color, 'stroke-width': 2 }
					}
				}}
			>
				{#snippet tooltip()}
					<ChartTooltip {color} labelFormatter={formatTimestamp}>
						{#snippet formatter({ value })}
							<div class="flex flex-1 items-center justify-between gap-2">
								<span class="text-muted-foreground">{label}</span>
								<span class="text-foreground font-mono font-medium tabular-nums">{value}{unit}</span
								>
							</div>
						{/snippet}
					</ChartTooltip>
				{/snippet}
			</AreaChart>
		</ChartContainer>
		{#if sharedPct !== null}
			<div
				class="pointer-events-none absolute top-0 bottom-0 w-px bg-[var(--tx-3)]"
				style:left="{sharedPct}%"
			></div>
		{/if}
	{:else}
		<div class="flex h-full items-center justify-center text-[11px] text-[var(--tx-3)]">
			Not enough history yet
		</div>
	{/if}
</div>
