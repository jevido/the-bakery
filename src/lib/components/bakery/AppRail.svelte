<script lang="ts">
	import { GUILD_LIST } from '$lib/data/bakery';
	import { goto } from '$app/navigation';

	let { guildId }: { guildId: string } = $props();

	function selectGuild(id: string) {
		goto(`/${id}/deploy/overview`);
	}
</script>

<div style="
	width: 76px;
	flex: none;
	background: var(--rail);
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 12px 0 14px;
	gap: 8px;
	border-right: 1px solid rgba(0,0,0,.4);
	z-index: 2;
">
	<!-- Logo -->
	<div
		title="The Bakery"
		style="
			width: 48px; height: 48px; border-radius: 15px;
			background: linear-gradient(150deg, var(--grn-2), var(--grn));
			display: flex; align-items: center; justify-content: center;
			font-family: 'Bricolage Grotesque', sans-serif; font-weight: 800;
			font-size: 24px; color: #fff;
			box-shadow: 0 4px 16px var(--grn-dim); cursor: pointer;
		"
		role="button"
		tabindex="0"
		onclick={() => selectGuild(guildId)}
		onkeydown={(e) => e.key === 'Enter' && selectGuild(guildId)}
	>B</div>

	<div style="width:32px;height:2px;background:rgba(255,255,255,.09);border-radius:2px;margin:2px 0"></div>

	<!-- Guild list -->
	{#each GUILD_LIST as gd}
		{@const active = gd.id === guildId}
		<div
			title={gd.name}
			onclick={() => selectGuild(gd.id)}
			onkeydown={(e) => e.key === 'Enter' && selectGuild(gd.id)}
			role="button"
			tabindex="0"
			style="position:relative;width:48px;height:48px;cursor:pointer;display:flex;align-items:center;justify-content:center;"
		>
			<!-- Active pill -->
			<div style="
				position: absolute; left: -14px; width: 4px;
				border-radius: 0 4px 4px 0;
				background: var(--tx);
				transition: all .15s;
				height: {active ? '32px' : '0px'};
			"></div>
			<!-- Avatar -->
			<div style="
				width: 48px; height: 48px;
				border-radius: {active ? '15px' : '24px'};
				background: {active ? 'linear-gradient(150deg, var(--grn-2), var(--grn))' : 'var(--card-2)'};
				display: flex; align-items: center; justify-content: center;
				font-family: 'Bricolage Grotesque', sans-serif;
				font-weight: 700; font-size: 19px;
				color: {active ? '#07130c' : 'var(--tx)'};
				transition: all .15s;
				border: {active ? 'none' : '1px solid var(--line)'};
			">{gd.letter}</div>
		</div>
	{/each}

	<!-- Add guild -->
	<div
		title="Create or join a guild"
		style="
			width:48px;height:48px;border-radius:15px;
			background:rgba(255,255,255,.05);color:var(--grn-2);
			display:flex;align-items:center;justify-content:center;
			font-size:26px;font-weight:400;cursor:pointer;
		"
	>+</div>

	<!-- Discover -->
	<div
		title="Discover"
		style="
			width:48px;height:48px;border-radius:15px;
			background:rgba(255,255,255,.05);color:var(--tx-2);
			display:flex;align-items:center;justify-content:center;cursor:pointer;
		"
	>
		<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
			<circle cx="12" cy="12" r="9"/>
			<path d="M15 9l-2.5 5L9 15l2.5-5z" fill="currentColor" stroke="none"/>
		</svg>
	</div>

	<!-- Settings -->
	<div
		title="Settings"
		style="
			margin-top:auto;width:40px;height:40px;border-radius:12px;
			background:rgba(255,255,255,.04);color:var(--tx-2);
			display:flex;align-items:center;justify-content:center;cursor:pointer;
		"
	>
		<svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
			<circle cx="12" cy="12" r="3.2"/>
			<path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/>
		</svg>
	</div>
</div>
