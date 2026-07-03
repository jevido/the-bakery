<script lang="ts">
	import type { AppStatus } from '$lib/data/bakery';
	import { statusMeta } from '$lib/data/bakery';

	let { status, size = 9 }: { status: AppStatus; size?: number } = $props();
	const meta = $derived(statusMeta(status));
</script>

<div
	class="shrink-0 rounded-full"
	style:width="{size}px"
	style:height="{size}px"
	style:background={meta.dot}
	style:animation={meta.pulse ? 'bk-pulse 2s infinite' : meta.spin ? 'bk-pulse 1s infinite' : 'none'}
></div>
