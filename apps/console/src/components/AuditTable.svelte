<script lang="ts">
  import type { AuditEntry, Names } from '../lib/api'
  import { targetPath, when } from '../lib/format'

  let { entries, names }: { entries: AuditEntry[]; names: Names } = $props()

  function nameOf(kind: string, id: number): string {
    if (kind === 'member') return names.members[id] ?? `member ${id}`
    if (kind === 'guild') return names.guilds[id] ?? `guild ${id}`
    return `${kind} ${id}`
  }
</script>

{#if entries.length === 0}
  <p class="dim">Nothing recorded.</p>
{:else}
  <table>
    <thead><tr><th>When</th><th>Who</th><th>Did</th><th>To</th><th>Reason</th><th>IP</th></tr></thead>
    <tbody>
      {#each entries as e (e.id)}
        <tr>
          <td class="dim">{when(e.at)}</td>
          <td>
            {#if targetPath(e.actor_kind, e.actor_id)}<a href={targetPath(e.actor_kind, e.actor_id)}>{nameOf(e.actor_kind, e.actor_id)}</a>
            {:else}{nameOf(e.actor_kind, e.actor_id)}{/if}
          </td>
          <td><code>{e.action}</code></td>
          <td>
            {#if e.target_kind}
              {#if targetPath(e.target_kind, e.target_id)}<a href={targetPath(e.target_kind, e.target_id)}>{nameOf(e.target_kind, e.target_id)}</a>
              {:else}{nameOf(e.target_kind, e.target_id)}{/if}
            {/if}
          </td>
          <td>{e.reason}</td>
          <td class="dim">{e.ip}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
