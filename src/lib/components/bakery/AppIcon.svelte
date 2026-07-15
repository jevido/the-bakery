<script lang="ts">
	import type { AppStatus } from '$lib/data/bakery';
	import { statusMeta } from '$lib/data/bakery';

	let {
		initial,
		status,
		size = 34,
		radius = 10
	}: {
		initial: string;
		status: AppStatus;
		size?: number;
		radius?: number;
	} = $props();
	const meta = $derived(statusMeta(status));
</script>

<div
	class="shrink-0 flex items-center justify-center font-heading font-bold"
	style:width="{size}px"
	style:height="{size}px"
	style:border-radius="{radius}px"
	style:background={meta.bg}
	style:color={meta.c}
	style:font-size="{Math.round(size * 0.41)}px"
>
	{initial}
</div>
