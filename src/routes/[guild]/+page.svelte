<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto(`/${page.params.guild}/deploy/overview`, { replaceState: true });
	});
</script>
