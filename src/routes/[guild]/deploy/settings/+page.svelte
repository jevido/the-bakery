<script lang="ts">
	import { page } from '$app/state';
	import { GUILDS } from '$lib/data/bakery';
	const guildId = $derived(page.params.guild ?? '');
	const guild = $derived(GUILDS[guildId]);
</script>

<div style="padding: 22px 28px;max-width:560px;">
	<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;margin-bottom:4px;">Guild Settings</div>
	<div style="font-size:13px;color:var(--tx-2);margin-bottom:22px;">Manage settings for {guild?.name ?? 'this guild'}.</div>

	<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;margin-bottom:14px;">
		<div style="padding:12px 18px;border-bottom:1px solid var(--line);font-size:12px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">GENERAL</div>
		<div style="padding:18px;">
			<label for="guild-name" style="display:block;font-size:13px;font-weight:600;color:var(--tx-2);margin-bottom:6px;">Guild name</label>
			<input id="guild-name" value={guild?.name ?? ''} style="width:100%;background:var(--card-2);border:1px solid var(--line-2);border-radius:8px;padding:10px 13px;font-size:14px;color:var(--tx);outline:none;box-sizing:border-box;" />
			<label for="invite-code" style="display:block;font-size:13px;font-weight:600;color:var(--tx-2);margin-bottom:6px;margin-top:14px;">Invite code</label>
			<div style="display:flex;gap:8px;">
				<input id="invite-code" value={guild?.invite ?? ''} readonly style="flex:1;background:var(--card-2);border:1px solid var(--line-2);border-radius:8px;padding:10px 13px;font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--tx-2);outline:none;" />
				<button style="padding:10px 14px;background:var(--card-2);border:1px solid var(--line);border-radius:8px;font-size:13px;color:var(--tx-2);cursor:pointer;font-weight:600;">Regenerate</button>
			</div>
		</div>
	</div>

	<div style="background:rgba(229,101,75,.08);border:1px solid rgba(229,101,75,.2);border-radius:13px;padding:18px;">
		<div style="font-size:14px;font-weight:700;color:#f0836b;margin-bottom:4px;">Danger zone</div>
		<div style="font-size:12.5px;color:var(--tx-2);margin-bottom:14px;">These actions are irreversible. Please be certain before proceeding.</div>
		<button style="padding:9px 16px;background:rgba(229,101,75,.14);border:1px solid rgba(229,101,75,.3);border-radius:8px;color:#f0836b;font-size:13px;font-weight:700;cursor:pointer;">Delete guild…</button>
	</div>
</div>

<style>
	button { all: unset; box-sizing: border-box; cursor: pointer; }
</style>
