<script lang="ts">
	const columns = [
		{
			id: 'backlog', label: 'Backlog', color: 'var(--tx-3)',
			cards: [
				{ id: 1, title: 'Add OIDC login provider', tag: 'FEAT', priority: 'medium' },
				{ id: 2, title: 'S3 backup for proofing-db', tag: 'INFRA', priority: 'high' },
				{ id: 3, title: 'Rate limiting on crumb-api', tag: 'FEAT', priority: 'medium' },
			]
		},
		{
			id: 'in-progress', label: 'In Progress', color: '#efc060',
			cards: [
				{ id: 4, title: 'Fix yeast-worker build', tag: 'BUG', priority: 'high' },
				{ id: 5, title: 'Migrate to Postgres 16', tag: 'INFRA', priority: 'medium' },
			]
		},
		{
			id: 'review', label: 'In Review', color: '#7aa6f5',
			cards: [
				{ id: 6, title: 'loaf-web dark mode improvements', tag: 'UI', priority: 'low' },
			]
		},
		{
			id: 'done', label: 'Done', color: '#52cc96',
			cards: [
				{ id: 7, title: 'Deploy Caddy with auto HTTPS', tag: 'INFRA', priority: 'high' },
				{ id: 8, title: 'Set up sourdough guild', tag: 'SETUP', priority: 'high' },
			]
		},
	];

	const tagColor: Record<string, string> = {
		FEAT: 'rgba(63,185,132,.18)',
		BUG: 'rgba(229,101,75,.18)',
		INFRA: 'rgba(91,141,239,.18)',
		UI: 'rgba(169,120,230,.18)',
		SETUP: 'rgba(224,168,62,.18)',
	};
	const tagText: Record<string, string> = {
		FEAT: '#52cc96', BUG: '#f0836b', INFRA: '#7aa6f5', UI: '#c09bf0', SETUP: '#efc060',
	};
	const priBadge: Record<string, { bg: string; c: string }> = {
		high:   { bg: 'rgba(229,101,75,.15)', c: '#f0836b' },
		medium: { bg: 'rgba(224,168,62,.15)', c: '#efc060' },
		low:    { bg: 'rgba(255,255,255,.06)', c: 'var(--tx-3)' },
	};
</script>

<div class="px-7 py-[22px]">
	<div class="font-heading font-bold text-[23px] mb-[18px]">Board</div>

	<div class="grid grid-cols-4 gap-[13px] items-start">
		{#each columns as col}
			<div>
				<!-- Column header -->
				<div class="flex items-center gap-2 mb-[11px] px-1">
					<div class="size-[9px] rounded-full shrink-0" style:background={col.color}></div>
					<span class="text-[13px] font-bold text-[var(--tx-2)]">{col.label}</span>
					<span class="text-[11.5px] font-bold px-2 py-[2px] rounded-[12px] bg-[var(--card)] border border-[var(--line)] text-[var(--tx-3)] ml-auto">{col.cards.length}</span>
				</div>

				<!-- Cards -->
				<div class="flex flex-col gap-2">
					{#each col.cards as card}
						{@const pb = priBadge[card.priority]}
						<div class="bg-[var(--card)] border border-[var(--line)] rounded-[11px] p-[13px_14px] cursor-pointer">
							<div class="flex items-center justify-between gap-2 mb-[9px]">
								<span class="text-[10.5px] font-bold px-2 py-[3px] rounded-[6px]" style:background={tagColor[card.tag] ?? 'rgba(255,255,255,.06)'} style:color={tagText[card.tag] ?? 'var(--tx-3)'}>{card.tag}</span>
								<span class="text-[10.5px] font-semibold px-2 py-[3px] rounded-[6px]" style:background={pb.bg} style:color={pb.c}>{card.priority}</span>
							</div>
							<div class="text-[13.5px] font-semibold leading-[1.4]">{card.title}</div>
						</div>
					{/each}

					<!-- Add card button -->
					<button class="flex items-center gap-[6px] w-full px-3 py-[9px] bg-transparent border border-dashed border-white/[0.08] rounded-[10px] cursor-pointer text-[var(--tx-3)] text-[12.5px]">
						<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>
						Add card
					</button>
				</div>
			</div>
		{/each}
	</div>
</div>
