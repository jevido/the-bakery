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

<div style="padding: 22px 28px;">
	<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;margin-bottom:18px;">Board</div>

	<div style="display:grid;grid-template-columns:repeat(4,1fr);gap:13px;align-items:start;">
		{#each columns as col}
			<div>
				<!-- Column header -->
				<div style="display:flex;align-items:center;gap:8px;margin-bottom:11px;padding:0 4px;">
					<div style="width:9px;height:9px;border-radius:50%;background:{col.color};flex:none;"></div>
					<span style="font-size:13px;font-weight:700;color:var(--tx-2);">{col.label}</span>
					<span style="font-size:11.5px;font-weight:700;padding:2px 8px;border-radius:12px;background:var(--card);border:1px solid var(--line);color:var(--tx-3);margin-left:auto;">{col.cards.length}</span>
				</div>

				<!-- Cards -->
				<div style="display:flex;flex-direction:column;gap:8px;">
					{#each col.cards as card}
						{@const pb = priBadge[card.priority]}
						<div style="background:var(--card);border:1px solid var(--line);border-radius:11px;padding:13px 14px;cursor:pointer;">
							<div style="display:flex;align-items:center;justify-content:space-between;gap:8px;margin-bottom:9px;">
								<span style="font-size:10.5px;font-weight:700;padding:3px 8px;border-radius:6px;background:{tagColor[card.tag] ?? 'rgba(255,255,255,.06)'};color:{tagText[card.tag] ?? 'var(--tx-3)'};">{card.tag}</span>
								<span style="font-size:10.5px;font-weight:600;padding:3px 8px;border-radius:6px;background:{pb.bg};color:{pb.c};">{card.priority}</span>
							</div>
							<div style="font-size:13.5px;font-weight:600;line-height:1.4;">{card.title}</div>
						</div>
					{/each}

					<!-- Add card button -->
					<button style="display:flex;align-items:center;gap:6px;padding:9px 12px;background:transparent;border:1px dashed rgba(255,255,255,.08);border-radius:10px;cursor:pointer;color:var(--tx-3);font-size:12.5px;">
						<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M12 5v14M5 12h14"/></svg>
						Add card
					</button>
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	button { all: unset; box-sizing: border-box; width: 100%; }
</style>
