<script lang="ts">
	import { page } from '$app/state';
	import { MEMBERS, ROLES } from '$lib/data/bakery';

	const guildId = $derived(page.params.guild ?? '');
	const members = $derived(MEMBERS[guildId] ?? []);

	function roleColor(roleName: string) {
		const r = ROLES.find(r => r.name === roleName);
		return r?.color ?? 'var(--tx-3)';
	}
</script>

<div style="padding: 22px 28px;">
	<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:18px;">
		<div>
			<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;">Members</div>
			<div style="font-size:13px;color:var(--tx-2);margin-top:2px;">{members.length} members in this guild</div>
		</div>
		<button style="display:flex;align-items:center;gap:8px;background:var(--grn);color:#07130c;border:none;border-radius:9px;padding:10px 16px;font-size:13.5px;font-weight:700;cursor:pointer;">
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3"><path d="M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"/></svg>
			Invite member
		</button>
	</div>

	<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
		<div style="display:grid;grid-template-columns:2fr 1.3fr 1fr 80px;gap:14px;padding:10px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">
			<div>MEMBER</div>
			<div>ROLE</div>
			<div>JOINED</div>
			<div></div>
		</div>

		{#each members as m}
			{@const rc = roleColor(m.role)}
			<div style="display:grid;grid-template-columns:2fr 1.3fr 1fr 80px;gap:14px;align-items:center;padding:13px 18px;border-bottom:1px solid var(--line);">
				<!-- Avatar + name -->
				<div style="display:flex;align-items:center;gap:12px;">
					<div style="width:36px;height:36px;border-radius:9px;background:{m.avBg};display:flex;align-items:center;justify-content:center;font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:14px;color:{m.avColor};flex:none;">{m.initials}</div>
					<div>
						<div style="font-size:14px;font-weight:600;color:var(--tx);">{m.name}</div>
						<div style="font-size:11.5px;color:var(--tx-3);">{m.handle}</div>
					</div>
				</div>

				<!-- Role badge -->
				<div>
					<span style="font-size:12.5px;font-weight:600;padding:4px 10px;border-radius:20px;background:{rc}22;color:{rc};">{m.role}</span>
				</div>

				<!-- Joined -->
				<div style="font-size:12.5px;color:var(--tx-2);">{m.joinedAt ?? '—'}</div>

				<!-- Actions -->
				<div style="display:flex;justify-content:flex-end;gap:4px;">
					{#if !m.master}
						<button style="padding:6px 11px;font-size:12px;background:var(--card-2);border:1px solid var(--line);border-radius:7px;color:var(--tx-2);cursor:pointer;font-weight:600;">Edit</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
</div>

<style>
	button { all: unset; box-sizing: border-box; }
</style>
