<script lang="ts">
	import { ROLES, PERM_GROUPS, ROLE_PERMS } from '$lib/data/bakery';

	let selectedRole = $state(ROLES[0]);
	const rolePerms = $derived(new Set(ROLE_PERMS[selectedRole.id] ?? []));
</script>

<div style="padding: 22px 28px;">
	<div style="margin-bottom:18px;">
		<div style="font-family:'Bricolage Grotesque',sans-serif;font-weight:700;font-size:23px;">Roles & Positions</div>
		<div style="font-size:13px;color:var(--tx-2);margin-top:2px;">Configure what each position can do in the guild. Positions are ranked — a baker cannot edit roles above their own.</div>
	</div>

	<div style="display:grid;grid-template-columns:260px 1fr;gap:16px;align-items:start;">
		<!-- Left: role list -->
		<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;">
			<div style="padding:12px 16px;border-bottom:1px solid var(--line);font-size:12px;font-weight:700;letter-spacing:.05em;color:var(--tx-3);">POSITIONS</div>
			{#each ROLES as r}
				{@const active = r.id === selectedRole.id}
				<button
					onclick={() => selectedRole = r}
					style="display:flex;align-items:center;gap:12px;width:100%;padding:13px 16px;border-bottom:1px solid var(--line);cursor:pointer;background:{active ? 'rgba(63,185,132,.08)' : 'transparent'};border-left:3px solid {active ? 'var(--grn)' : 'transparent'};"
				>
					<div style="width:10px;height:10px;border-radius:50%;background:{r.color};flex:none;"></div>
					<div style="flex:1;min-width:0;text-align:left;">
						<div style="font-size:13.5px;font-weight:{active ? '700' : '600'};color:{active ? 'var(--tx)' : 'var(--tx-2)'};">{r.name}</div>
						<div style="font-size:11.5px;color:var(--tx-3);">{r.count} member{r.count !== 1 ? 's' : ''}</div>
					</div>
					{#if r.master}
						<span style="font-size:10px;font-weight:700;padding:2px 7px;border-radius:5px;background:rgba(224,168,62,.14);color:#e0a83e;">OWNER</span>
					{/if}
				</button>
			{/each}
		</div>

		<!-- Right: permissions panel -->
		<div>
			<!-- Role header -->
			<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;padding:16px 18px;margin-bottom:12px;">
				<div style="display:flex;align-items:center;gap:11px;margin-bottom:8px;">
					<div style="width:12px;height:12px;border-radius:50%;background:{selectedRole.color};"></div>
					<div style="font-size:16px;font-weight:700;">{selectedRole.name}</div>
				</div>
				<div style="font-size:13px;color:var(--tx-2);line-height:1.5;">{selectedRole.note}</div>
			</div>

			<!-- Perm groups -->
			{#each PERM_GROUPS as g}
				<div style="background:var(--card);border:1px solid var(--line);border-radius:13px;overflow:hidden;margin-bottom:10px;">
					<div style="padding:10px 18px;border-bottom:1px solid var(--line);font-size:11px;font-weight:700;letter-spacing:.06em;color:var(--tx-3);">{g.name}</div>
					{#each g.perms as p}
						{@const has = rolePerms.has(p.id) || rolePerms.has('administrator')}
						<div style="display:flex;align-items:center;gap:13px;padding:12px 18px;border-bottom:1px solid var(--line);">
							<!-- Toggle -->
							<div style="width:34px;height:20px;border-radius:10px;background:{has ? 'var(--grn)' : 'rgba(255,255,255,.08)'};flex:none;position:relative;cursor:{selectedRole.master ? 'not-allowed' : 'pointer'};">
								<div style="width:14px;height:14px;border-radius:50%;background:#fff;position:absolute;top:3px;left:{has ? '17px' : '3px'};transition:left .15s;"></div>
							</div>
							<div style="flex:1;min-width:0;">
								<div style="font-size:13.5px;font-weight:600;color:{p.danger ? '#f0836b' : 'var(--tx)'};">{p.label}</div>
								<div style="font-size:12px;color:var(--tx-3);margin-top:1px;">{p.desc}</div>
							</div>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	</div>
</div>

<style>
	button { all: unset; box-sizing: border-box; }
</style>
