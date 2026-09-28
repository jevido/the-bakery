<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { api, ApiError } from '../lib/api'
  import { when } from '../lib/format'

  // Abuse signals: who signs up, founds guilds or hits rate limits in
  // volume. A row is a lead to look at, not a verdict.
  type Count = { ip?: string; member_id?: number; name?: string; count: number; limits?: string[]; last: string }
  type Summary = {
    since: string
    sign_ups_by_ip: Count[]
    limit_hits_by_member: Count[]
    limit_hits_by_ip: Count[]
    guilds_founded_by: Count[]
  }

  let hours = $state(24)
  let summary = $state.raw<Summary | null>(null)
  let error = $state('')

  async function load() {
    error = ''
    try {
      summary = await api<Summary>('GET', `/api/console/signals?hours=${hours}`)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  load()
</script>

<svelte:head><title>Signals · Console</title></svelte:head>

{#snippet who(c: Count)}
  {#if c.member_id}
    <a href="/members/{c.member_id}">{c.name || `member ${c.member_id}`}</a>
  {:else}
    <code>{c.ip}</code>
  {/if}
{/snippet}

{#snippet table(title: string, rows: Count[], unit: string, showLimits = false)}
  <Panel {title}>
    {#if rows.length === 0}
      <p class="dim">Nothing.</p>
    {:else}
      <table>
        <thead>
          <tr><th>Who</th><th>{unit}</th>{#if showLimits}<th>Limits</th>{/if}<th>Last</th></tr>
        </thead>
        <tbody>
          {#each rows as c (c.member_id ?? c.ip)}
            <tr>
              <td>{@render who(c)}</td>
              <td>{c.count}</td>
              {#if showLimits}<td>{c.limits?.join(', ')}</td>{/if}
              <td class="dim">{when(c.last)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </Panel>
{/snippet}

<div class="head">
  <label>
    Last
    <select bind:value={() => hours, (v) => ((hours = v), load())} aria-label="Window">
      <option value={1}>hour</option>
      <option value={24}>24 hours</option>
      <option value={168}>7 days</option>
      <option value={720}>30 days</option>
    </select>
  </label>
  {#if summary}<span class="dim">since {when(summary.since)}</span>{/if}
</div>
{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if summary}
  <div class="grid">
    {@render table('Sign-ups by IP', summary.sign_ups_by_ip, 'Sign-ups')}
    {@render table('Guilds founded by member', summary.guilds_founded_by, 'Guilds')}
    {@render table('Rate limits hit by member', summary.limit_hits_by_member, 'Refusals', true)}
    {@render table('Rate limits hit by IP', summary.limit_hits_by_ip, 'Refusals', true)}
  </div>
{/if}

<style>
  .head {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 12px;
  }

  select {
    font: inherit;
    color: var(--text);
    padding: 2px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
    gap: 12px;
  }
</style>
