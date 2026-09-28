<script lang="ts">
  import { Panel, Button } from '@bakery/ui'
  import { Clipboard, Events } from '@wailsio/runtime'
  import Confirm from './Confirm.svelte'
  import { WorkshopService, messageOf, type RunEvent, type RunInfo } from '../lib/bindings'
  import { renderMarkdown, openLinksOutside } from '../lib/markdown'

  // One run as it happens: what the agent says and does, streamed from
  // WorkshopService. The run keeps going when the panel closes.
  let { run, onback, onclose }: { run: RunInfo; onback: () => void; onclose: () => void } = $props()

  let events = $state.raw<RunEvent[]>([])
  let error = $state('')
  let stopping = $state(false)
  let now = $state(Date.now())

  // The run's info is replaced whenever a run starts or ends; only its id
  // decides what to listen to.
  const runId = $derived(run.id)

  // Listen first, then load what was said before; seq drops the doubles.
  $effect(() => {
    const id = runId
    let seen = 0
    let pending: RunEvent[] = []
    let loaded = false
    const add = (list: RunEvent[]) => {
      const fresh = list.filter((e) => e.seq > seen)
      if (!fresh.length) return
      seen = fresh[fresh.length - 1].seq
      events = [...events, ...fresh]
    }
    events = []
    const off = Events.On('run:' + id, (e) => {
      const ev = e.data as RunEvent
      if (loaded) add([ev])
      else pending.push(ev)
    })
    WorkshopService.RunEvents(id)
      .then((past) => {
        add(past ?? [])
        add(pending)
        loaded = true
        pending = []
      })
      .catch((err) => (error = messageOf(err)))
    return () => off()
  })

  $effect(() => {
    if (run.status !== 'running') return
    const timer = setInterval(() => (now = Date.now()), 1000)
    return () => clearInterval(timer)
  })

  const init = $derived(events.find((e) => e.kind === 'init'))
  const result = $derived(events.findLast((e) => e.kind === 'result'))
  const timeline = $derived(events.filter((e) => e.kind !== 'init'))
  // A tool's result, by the call it answers.
  const results = $derived(new Map(events.filter((e) => e.kind === 'tool_result').map((e) => [e.tool_id, e])))

  const STATUS: Record<string, string> = { running: 'Working', succeeded: 'Done', failed: 'Failed', stopped: 'Stopped' }

  function elapsed(): string {
    const ms = run.status === 'running' ? now - new Date(run.started_at).getTime() : (result?.duration_ms ?? 0)
    const s = Math.max(0, Math.round(ms / 1000))
    return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`
  }

  // A short line for a tool call: the command, the file, the pattern.
  function brief(e: RunEvent): string {
    try {
      const input = JSON.parse(e.input || '{}') as Record<string, unknown>
      const v = input.command ?? input.file_path ?? input.path ?? input.pattern ?? input.url ?? input.description ?? ''
      return String(v).split('\n')[0].slice(0, 120)
    } catch {
      return ''
    }
  }

  let confirmingRemove = $state(false)
  let removing = $state(false)
  let copied = $state(false)

  async function removeWorktree() {
    removing = true
    try {
      await WorkshopService.RemoveWorktree(run.id)
      confirmingRemove = false
    } catch (err) {
      error = messageOf(err)
    }
    removing = false
  }

  async function copyBranch() {
    await Clipboard.SetText(run.branch)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  async function stop() {
    stopping = true
    try {
      await WorkshopService.StopRun(run.id)
    } catch (err) {
      error = messageOf(err)
      stopping = false
    }
  }
</script>

<aside class="run-panel" aria-label="Run">
  <Panel>
    {#snippet header()}
      <span class="head">{run.agent_name} · {run.task_title}</span>
      <span class={['badge', run.status]}>{STATUS[run.status] ?? run.status}</span>
    {/snippet}
    {#snippet actions()}
      <button class="link" onclick={onback}>Task</button>
      <button class="close" aria-label="Close" onclick={onclose}>×</button>
    {/snippet}

    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <dl class="facts">
      <dt>Branch</dt>
      <dd><code>{run.branch}</code></dd>
      <dt>Model</dt>
      <dd>{init?.model || run.model} · {run.permission_mode}</dd>
      <dt>Cost</dt>
      <dd>${(result?.cost_usd ?? run.cost_usd).toFixed(2)} · {elapsed()}</dd>
    </dl>
    {#if run.status === 'running'}
      <Button variant="danger" disabled={stopping} onclick={stop}>{stopping ? 'Stopping…' : 'Stop'}</Button>
    {/if}

    {#if init}
      <details class="init">
        <summary>
          {init.skills?.filter((s) => s.startsWith('bakery-')).length ?? 0} skills from this agent and board ·
          {init.mcp_servers?.length ?? 0} MCP servers
        </summary>
        <ul class="chips">
          {#each init.skills ?? [] as s (s)}<li class={{ ours: s.startsWith('bakery-') }}>{s}</li>{/each}
        </ul>
        <ul class="chips">
          {#each init.mcp_servers ?? [] as m (m.name)}<li class={{ ours: m.status === 'connected', bad: m.status !== 'connected' }}>{m.name}: {m.status}</li>{/each}
        </ul>
      </details>
    {:else if run.status === 'running'}
      <p class="dim">Starting Claude…</p>
    {/if}

    <ol class="timeline">
      {#each timeline as e (e.seq)}
        {#if e.kind === 'text'}
          <li class="text">
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="markdown" onclick={openLinksOutside}>{@html renderMarkdown(e.text ?? '')}</div>
          </li>
        {:else if e.kind === 'tool_call'}
          {@const r = results.get(e.tool_id)}
          <li class={['tool', { failed: r?.is_error }]}>
            <details>
              <summary><span class="tool-name">{e.tool}</span> <span class="dim">{brief(e)}</span></summary>
              <pre>{e.input}</pre>
              {#if r}<pre class="result">{r.text}</pre>{/if}
            </details>
          </li>
        {:else if e.kind === 'note'}
          <li class="note">{e.text}</li>
        {:else if e.kind === 'result'}
          <li class="end">
            {e.is_error || e.subtype !== 'success' ? `Ended: ${e.subtype}` : 'Finished'} · ${e.cost_usd?.toFixed(2)} ·
            {e.turns} turns · {Math.round((e.duration_ms ?? 0) / 1000)}s
          </li>
        {/if}
      {/each}
    </ol>

    {#if run.status !== 'running'}
      <footer class="after">
        <p>
          {STATUS[run.status] ?? run.status} · +{run.diff.committed.additions} −{run.diff.committed.deletions} in
          {run.diff.committed.files} committed {run.diff.committed.files === 1 ? 'file' : 'files'}{#if run.diff.uncommitted.files}, {run.diff.uncommitted.files} left uncommitted{/if}
          {#if run.moved_to}· moved to {run.moved_to}{/if}
        </p>
        {#if run.worktree_removed}
          <p class="dim">Worktree removed; the branch <code>{run.branch}</code> is still there.</p>
        {:else if confirmingRemove}
          <Confirm
            question="Remove this run's worktree? Its branch stays."
            action="Remove worktree"
            busy={removing}
            onconfirm={removeWorktree}
            oncancel={() => (confirmingRemove = false)}
          />
        {:else}
          <div class="row">
            <Button onclick={() => WorkshopService.OpenWorktree(run.id).catch((err) => (error = messageOf(err)))}>Open folder</Button>
            <Button onclick={copyBranch}>{copied ? 'Copied' : 'Copy branch name'}</Button>
            <Button variant="danger" onclick={() => (confirmingRemove = true)}>Remove worktree</Button>
          </div>
        {/if}
      </footer>
    {/if}
  </Panel>
</aside>

<style>
  .run-panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .run-panel :global(.panel) {
    min-height: 0;
    overflow: auto;
  }

  .head {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .badge {
    padding: 0 6px;
    font-family: var(--font-body);
    font-size: 11px;
    font-weight: 400;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .badge.running {
    color: var(--olive-bright);
    border-color: var(--olive);
  }

  .badge.failed {
    color: var(--rust-bright);
    border-color: var(--rust);
  }

  .facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2px 10px;
    margin: 0 0 8px;
    font-size: 13px;
  }

  dt {
    color: var(--text-dim);
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .init {
    margin: 8px 0;
    font-size: 12px;
    color: var(--text-dim);
  }

  .chips {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin: 6px 0 0;
    padding: 0;
  }

  .chips li {
    padding: 0 6px;
    border: 1px solid var(--frame-dim);
    border-radius: 999px;
  }

  .chips li.ours {
    color: var(--olive-bright);
    border-color: var(--olive);
  }

  .chips li.bad {
    color: var(--rust-bright);
    border-color: var(--rust);
  }

  .timeline {
    list-style: none;
    margin: 8px 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .text {
    padding: 6px 8px;
    background: var(--panel-inset);
    border-radius: var(--radius);
    overflow-wrap: anywhere;
  }

  .text :global(p) {
    margin: 0 0 6px;
  }

  .text :global(p:last-child) {
    margin: 0;
  }

  .tool {
    font-size: 12px;
    border-left: 2px solid var(--steel);
    padding-left: 6px;
  }

  .tool.failed {
    border-left-color: var(--rust);
  }

  .tool summary {
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tool-name {
    font-weight: 600;
    color: var(--steel-bright);
  }

  pre {
    margin: 4px 0 0;
    padding: 6px;
    max-height: 200px;
    overflow: auto;
    font-size: 11px;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: var(--bg-deep);
    border-radius: var(--radius);
    user-select: text;
  }

  .result {
    color: var(--text-dim);
  }

  .after {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px solid var(--frame-dim);
    font-size: 13px;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .note {
    font-size: 12px;
    font-style: italic;
    color: var(--text-dim);
  }

  .end {
    padding-top: 6px;
    font-size: 12px;
    color: var(--text-dim);
    border-top: 1px solid var(--frame-dim);
  }

  .link {
    font: inherit;
    font-size: 12px;
    padding: 0 6px;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .close {
    font: inherit;
    font-size: 16px;
    padding: 0 4px;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  code {
    font-size: 12px;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    margin-bottom: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
