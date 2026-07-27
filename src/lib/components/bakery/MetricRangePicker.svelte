<script lang="ts">
	import { METRIC_RANGES, type MetricRange } from '$lib/hosts/metric-ranges';

	let {
		value,
		onchange
	}: {
		value: MetricRange['id'];
		onchange: (id: MetricRange['id']) => void;
	} = $props();

	function pillCls(id: MetricRange['id']) {
		return id === value
			? 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx)] rounded-[6px] cursor-pointer bg-[var(--card-2)]'
			: 'px-[11px] py-[5px] text-[12.5px] text-[var(--tx-2)] rounded-[6px] cursor-pointer';
	}
</script>

<div class="flex gap-0.5 bg-[var(--card)] border border-[var(--line)] rounded-[9px] p-[3px]">
	{#each METRIC_RANGES as r (r.id)}
		<button onclick={() => onchange(r.id)} class={pillCls(r.id)}>{r.label}</button>
	{/each}
</div>
