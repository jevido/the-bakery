<script lang="ts">
	// Written generically enough to reuse for app runtime logs later (task 09
	// note) — it just needs an SSE endpoint at `logsUrl` emitting `data: "<line>"`
	// events and closing the stream when there's nothing left to send.
	let { logsUrl }: { logsUrl: string } = $props();

	let lines = $state<string[]>([]);
	let container: HTMLDivElement | undefined = $state();
	let stickToBottom = $state(true);

	function onScroll() {
		if (!container) return;
		const distanceFromBottom =
			container.scrollHeight - container.scrollTop - container.clientHeight;
		stickToBottom = distanceFromBottom < 24;
	}

	$effect(() => {
		lines = [];
		stickToBottom = true;
		const source = new EventSource(logsUrl);
		source.onmessage = (event) => {
			lines.push(JSON.parse(event.data) as string);
		};
		return () => source.close();
	});

	$effect(() => {
		void lines.length;
		if (stickToBottom && container) {
			container.scrollTop = container.scrollHeight;
		}
	});
</script>

<div
	bind:this={container}
	onscroll={onScroll}
	class="max-h-[360px] overflow-y-auto px-4 py-[14px] font-mono-jb text-[12px] leading-[1.7]"
>
	{#each lines as line, i (i)}
		<div class="whitespace-pre-wrap text-[var(--tx)]">{line}</div>
	{/each}
	{#if lines.length === 0}
		<div class="text-[var(--tx-3)]">Waiting for logs…</div>
	{/if}
</div>
