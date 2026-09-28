<script lang="ts">
  import { untrack } from 'svelte'
  import { Panel, Button, TextField } from '@bakery/ui'
  import type { Letter } from '../lib/bindings'
  import type { Letters } from '../lib/letters.svelte'

  // One letter, to answer: a permission prompt (allow once, always on this
  // board, or deny with a reason) or a run's questions.
  let {
    letters,
    letter,
    who,
    onopenrun,
  }: { letters: Letters; letter: Letter; who: { name: string; task: string }; onopenrun: (runId: string) => void } = $props()

  let reason = $state('')
  let busy = $state(false)
  // Per question: the chosen labels, or free text.
  let picked = $state<Record<string, string[]>>({})
  // Every question starts with empty free text: a text field cannot bind to
  // a missing entry. The dialog is keyed by letter, so the first letter is
  // the only one.
  let other = $state<Record<string, string>>(untrack(() => Object.fromEntries((letter.questions ?? []).map((q) => [q.question, '']))))

  const input = $derived((letter.input ?? {}) as Record<string, unknown>)
  const command = $derived(typeof input.command === 'string' ? input.command : '')
  const detail = $derived(command || JSON.stringify(letter.input, null, 2))

  async function answer(choice: 'allow' | 'allow_always' | 'deny') {
    busy = true
    await letters.answer(letter.id, choice, choice === 'deny' ? reason : '')
    busy = false
  }

  function toggle(q: string, label: string, multi: boolean) {
    const now = picked[q] ?? []
    picked[q] = multi ? (now.includes(label) ? now.filter((l) => l !== label) : [...now, label]) : [label]
  }

  const complete = $derived(
    (letter.questions ?? []).every((q) => (picked[q.question]?.length ?? 0) > 0 || (other[q.question] ?? '').trim()),
  )

  async function sendAnswers() {
    const answers: Record<string, string> = {}
    for (const q of letter.questions ?? []) {
      const free = (other[q.question] ?? '').trim()
      answers[q.question] = free || (picked[q.question] ?? []).join(', ')
    }
    busy = true
    await letters.answer(letter.id, 'allow', '', answers)
    busy = false
  }

  function close() {
    letters.openId = null
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && close()} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="veil" onclick={close}></div>
<div class="dialog" role="dialog" aria-label="Letter">
  <Panel title={letter.kind === 'question' ? `${who.name} has a question` : `${who.name} wants to use ${letter.tool_name}`}>
    {#snippet actions()}
      <button class="close" aria-label="Close" onclick={close}>×</button>
    {/snippet}
    {#if who.task}<p class="dim">Working on “{who.task}”</p>{/if}
    {#if letters.error}<p class="error" role="alert">{letters.error}</p>{/if}

    {#if letter.kind === 'question'}
      {#each letter.questions ?? [] as q (q.question)}
        <section>
          <h3>{q.header ? `${q.header}: ` : ''}{q.question}</h3>
          <div class="options">
            {#each q.options as o (o.label)}
              <button
                class={['option', { on: picked[q.question]?.includes(o.label) }]}
                aria-pressed={picked[q.question]?.includes(o.label) ?? false}
                onclick={() => toggle(q.question, o.label, q.multiSelect)}
              >
                <strong>{o.label}</strong>
                {#if o.description}<span class="dim">{o.description}</span>{/if}
              </button>
            {/each}
          </div>
          <TextField placeholder="Or say something else" bind:value={other[q.question]} />
        </section>
      {/each}
      <div class="row">
        <Button variant="confirm" disabled={busy || !complete} onclick={sendAnswers}>Answer</Button>
        <Button disabled={busy} onclick={() => answer('deny')}>Don't answer</Button>
      </div>
    {:else}
      <pre>{detail}</pre>
      {#if typeof input.description === 'string' && input.description}<p class="dim">{input.description}</p>{/if}
      <div class="row">
        <Button variant="confirm" disabled={busy} onclick={() => answer('allow')}>Allow</Button>
        <Button disabled={busy} onclick={() => answer('allow_always')}>Always allow on this board</Button>
      </div>
      <div class="row deny">
        <TextField placeholder="Why not (optional)" bind:value={reason} />
        <Button variant="danger" disabled={busy} onclick={() => answer('deny')}>Deny</Button>
      </div>
    {/if}
    <button class="link" onclick={() => onopenrun(letter.run_id)}>Open run</button>
  </Panel>
</div>

<style>
  .veil {
    position: fixed;
    inset: 0;
    z-index: 40;
    background: rgb(0 0 0 / 0.35);
  }

  .dialog {
    position: fixed;
    z-index: 41;
    top: 50%;
    left: 50%;
    width: min(520px, calc(100vw - 40px));
    max-height: calc(100vh - 80px);
    overflow: auto;
    transform: translate(-50%, -50%);
  }

  pre {
    margin: 8px 0;
    padding: 8px;
    max-height: 240px;
    overflow: auto;
    font-size: 12px;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: var(--bg-deep);
    border-radius: var(--radius);
    user-select: text;
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 8px 0;
  }

  h3 {
    margin: 0;
    font-size: 14px;
  }

  .options {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .option {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 6px 8px;
    font: inherit;
    text-align: left;
    color: var(--text);
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .option.on {
    border-color: var(--olive-bright);
    background: color-mix(in srgb, var(--olive) 25%, var(--panel-inset));
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 6px;
    margin-top: 8px;
  }

  .row.deny :global(.field) {
    flex: 1;
  }

  .link {
    margin-top: 10px;
    font: inherit;
    font-size: 12px;
    padding: 0;
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

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
    font-size: 12px;
  }

  .error {
    margin: 6px 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
