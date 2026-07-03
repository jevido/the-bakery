<script lang="ts">
	import type { AppStatus } from '$lib/data/bakery';
	import { statusMeta } from '$lib/data/bakery';

	let { status, size = 9 }: { status: AppStatus; size?: number } = $props();
	const meta = $derived(statusMeta(status));
</script>

<div
	style="
		width: {size}px;
		height: {size}px;
		border-radius: 50%;
		background: {meta.dot};
		flex: none;
		animation: {meta.pulse ? 'bk-pulse 2s infinite' : meta.spin ? 'bk-pulse 1s infinite' : 'none'};
	"
></div>
