<script lang="ts">
  import { untrack } from 'svelte'
  import { Button, Panel } from '@bakery/ui'
  import Confirm from './Confirm.svelte'
  import type { Colony } from '../lib/colony.svelte'
  import type { SupervisorChat } from '../lib/supervisor.svelte'
  import type { ChatProposal } from '../lib/bindings'

  // The supervisor chat: ask what should happen next, and approve or
  // decline what it proposes. It never acts on its own; each message is one
  // Claude call, and its cost shows under the answer.
  let { chat, colony }: { chat: SupervisorChat; colony: Colony } = $props()

  let text = $state('')
  let confirmForget = $state(false)
  let list = $state<HTMLElement | null>(null)
  // Edited subtasks of split proposals, by "entry:index".
  let edits = $state<Record<string, { title: string; description: string; work_type: string }[]>>({})

  async function send(event?: Event) {
    event?.preventDefault()
    const t = text
    text = ''
    if (!(await chat.send(t))) text = untrack(() => text) || t
  }

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) send(e)
  }

  // Keep the newest message in view.
  $effect(() => {
    void chat.entries.length
    void chat.thinking
    queueMicrotask(() => list?.scrollTo({ top: list.scrollHeight }))
  })

  function taskTitle(id: number | undefined): string {
    for (const c of colony.view?.columns ?? []) for (const t of c.tasks) if (t.id === id) return t.title
    return `task ${id}`
  }

  type Row = { title: string; description: string; work_type: string }

  function proposed(p: ChatProposal): Row[] {
    return (p.subtasks ?? []).map((s) => ({ title: s.title, description: s.description ?? '', work_type: s.work_type ?? '' }))
  }

  // The rows shown: the member's edits once there are any, else the
  // proposal's. Edits are made in event handlers, never while drawing.
  function rows(key: string, p: ChatProposal): Row[] {
    return edits[key] ?? proposed(p)
  }

  function edit(key: string, p: ChatProposal, change: (rows: Row[]) => void) {
    const next = [...rows(key, p).map((r) => ({ ...r }))]
    change(next)
    edits[key] = next
  }

  function outcomeTone(o: string): string {
    if (o.startsWith('refused')) return 'refused'
    if (o === 'declined') return 'declined'
    return 'done'
  }
</script>

<Panel title="Supervisor">
  {#snippet actions()}
    {#if chat.entries.length}<button class="link" onclick={() => (confirmForget = true)}>Forget</button>{/if}
  {/snippet}
  <div class="chat">
    {#if confirmForget}
      <Confirm
        question="Forget this conversation? The supervisor starts afresh; the board is not changed."
        action="Forget"
        onconfirm={() => {
          confirmForget = false
          chat.forget()
        }}
        oncancel={() => (confirmForget = false)}
      />
    {/if}
    <ol class="entries" bind:this={list} aria-label="Conversation">
      {#if chat.entries.length === 0}
        <li class="dim">Ask the supervisor what should happen next. It looks at the board and your agents, and proposes who takes what, what to split and in which order.</li>
      {/if}
      {#each chat.entries as e (e.id)}
        <li class={['entry', e.from]}>
          {#if e.text}<p>{e.text}</p>{/if}
          {#if e.error}<p class="error">{e.error}</p>{/if}
          {#each e.proposals ?? [] as p, i (i)}
            {@const key = `${e.id}:${i}`}
            <div class="proposal">
              <strong>{p.summary}</strong>
              {#if p.why}<span class="dim">{p.why}</span>{/if}
              {#if p.kind === 'split'}
                {#if !p.outcome}
                  <ol class="subs">
                    {#each rows(key, p) as s, j (j)}
                      <li>
                        <input
                          aria-label="Subtask title"
                          maxlength="200"
                          value={s.title}
                          oninput={(ev) => edit(key, p, (r) => (r[j].title = ev.currentTarget.value))}
                        />
                        <button class="x" aria-label="Drop this subtask" onclick={() => edit(key, p, (r) => r.splice(j, 1))}>×</button>
                      </li>
                    {/each}
                  </ol>
                {/if}
              {:else if p.kind === 'reorder' && !p.outcome}
                <ol class="subs">{#each p.task_ids ?? [] as id (id)}<li>{taskTitle(id)}</li>{/each}</ol>
              {/if}
              {#if p.outcome}
                <span class={['outcome', outcomeTone(p.outcome)]}>{p.outcome}</span>
              {:else}
                <div class="row">
                  <Button
                    variant="confirm"
                    disabled={!!chat.busy || (p.kind === 'split' && !rows(key, p).some((s) => s.title.trim()))}
                    onclick={() => chat.approve(e.id, i, p.kind === 'split' && edits[key] ? edits[key].filter((s) => s.title.trim()) : null)}
                    >{chat.busy === key ? 'Working…' : 'Approve'}</Button
                  >
                  <Button disabled={!!chat.busy} onclick={() => chat.decline(e.id, i)}>Decline</Button>
                </div>
              {/if}
            </div>
          {/each}
          {#if e.from === 'supervisor' && e.cost_usd}<span class="cost">${e.cost_usd.toFixed(3)}</span>{/if}
        </li>
      {/each}
      {#if chat.thinking}<li class="entry supervisor dim">Thinking…</li>{/if}
    </ol>
    {#if chat.error}<p class="error" role="alert">{chat.error}</p>{/if}
    <form onsubmit={send}>
      <textarea
        aria-label="Message to the supervisor"
        placeholder="What should happen next?"
        maxlength="4000"
        bind:value={text}
        onkeydown={keydown}
        disabled={!chat.boardId}
      ></textarea>
      <Button type="submit" disabled={chat.thinking || !text.trim() || !chat.boardId}>Send</Button>
    </form>
  </div>
</Panel>

<style>
  .chat {
    display: flex;
    flex-direction: column;
    gap: 6px;
    height: 100%;
    min-height: 0;
  }

  .entries {
    flex: 1;
    min-height: 120px;
    overflow-y: auto;
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .entry {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 6px 8px;
    border-radius: var(--radius);
    font-size: 13px;
  }

  .entry p {
    margin: 0;
    white-space: pre-wrap;
    user-select: text;
  }

  .entry.member {
    align-self: flex-end;
    max-width: 90%;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
  }

  .entry.supervisor {
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
  }

  .proposal {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 6px;
    background: var(--panel);
    border: 1px solid var(--frame);
    border-left: 3px solid var(--steel-bright);
    border-radius: var(--radius);
  }

  .subs {
    margin: 0;
    padding-left: 18px;
    font-size: 12px;
  }

  .subs li {
    display: flex;
    gap: 4px;
    align-items: center;
    list-style: decimal;
  }

  .subs input {
    flex: 1;
    min-width: 0;
    font: inherit;
    font-size: 12px;
    color: var(--text);
    padding: 2px 4px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .x {
    font: inherit;
    color: var(--rust-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .row {
    display: flex;
    gap: 6px;
  }

  .outcome {
    font-size: 12px;
  }

  .outcome.done {
    color: var(--olive-bright);
  }

  .outcome.declined {
    color: var(--text-faint);
  }

  .outcome.refused {
    color: var(--rust-bright);
  }

  .cost {
    align-self: flex-end;
    font-size: 11px;
    color: var(--text-faint);
  }

  .dim {
    color: var(--text-dim);
    font-size: 12px;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  textarea {
    min-height: 54px;
    resize: vertical;
    font: inherit;
    font-size: 13px;
    color: var(--text);
    padding: 5px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  .link {
    font: inherit;
    font-size: 11px;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
    text-decoration: underline;
  }
</style>
