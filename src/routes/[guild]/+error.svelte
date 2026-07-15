<script lang="ts">
	import { page } from '$app/state';
	import NotFound from '$lib/components/bakery/NotFound.svelte';

	const status = $derived(page.status);
	const message = $derived(page.error?.message ?? 'Something went wrong');
	const is404 = $derived(status === 404);
</script>

<div class="flex-1 h-full">
	<NotFound label={is404 ? message : `Error ${status}`} detail={is404 ? '' : message} />
</div>
