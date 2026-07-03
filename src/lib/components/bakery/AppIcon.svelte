<script lang="ts">
	import type { AppStatus } from '$lib/data/bakery';
	import { statusMeta } from '$lib/data/bakery';

	let { initial, status, size = 34, radius = 10 }: {
		initial: string;
		status: AppStatus;
		size?: number;
		radius?: number;
	} = $props();
	const meta = $derived(statusMeta(status));
</script>

<div style="
	width: {size}px;
	height: {size}px;
	flex: none;
	border-radius: {radius}px;
	background: {meta.bg};
	display: flex;
	align-items: center;
	justify-content: center;
	font-family: 'Bricolage Grotesque', sans-serif;
	font-weight: 700;
	font-size: {Math.round(size * 0.41)}px;
	color: {meta.c};
">{initial}</div>
