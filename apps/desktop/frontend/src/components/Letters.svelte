<script lang="ts">
  import { Portrait } from '@bakery/ui'
  import LetterDialog from './LetterDialog.svelte'
  import type { Letters } from '../lib/letters.svelte'
  import type { Colony } from '../lib/colony.svelte'
  import type { Story } from '../lib/story.svelte'
  import { settings } from '../lib/settings.svelte'
  import { fullTime } from '../lib/activity'

  // RimWorld-style letters: an envelope per run that needs you, stacked on
  // the right edge, newest at the bottom. Click one to answer it. Event
  // letters from the storyteller wait here too until OK; they ask nothing.
  let { letters, colony, story }: { letters: Letters; colony: Colony; story: Story } = $props()

  function who(runId: string) {
    const run = colony.workshop.byId(runId)
    const agent = colony.agents.find((a) => a.slug === run?.agent_slug)
    return { name: run?.agent_name ?? 'An agent', seed: agent?.portrait_seed || run?.agent_slug || runId, task: run?.task_title ?? '' }
  }
</script>

{#if letters.list.length || story.unread.length || (story.history.length && !settings.quiet)}
  <ol class="letters" aria-label="Letters">
    {#each story.unread as l (l.id)}
      <li>
        <button class={['envelope', 'story', l.kind]} title={l.title} onclick={() => (story.openId = l.id)}>
          <span class="mark">✉</span>
          <span class="story-title">{l.title}</span>
        </button>
      </li>
    {/each}
    {#each letters.list as l (l.id)}
      {@const w = who(l.run_id)}
      <li>
        <button
          class={['envelope', l.kind]}
          title={l.kind === 'question' ? `${w.name} has a question` : `${w.name} wants to use ${l.tool_name}`}
          onclick={() => (letters.openId = l.id)}
        >
          <Portrait seed={w.seed} size={20} />
          <span class="mark">{l.kind === 'question' ? '?' : '!'}</span>
        </button>
      </li>
    {/each}
    {#if story.history.length && !settings.quiet}
      <li>
        <button class="history-toggle" onclick={() => (story.showHistory = !story.showHistory)}>
          {story.showHistory ? 'Hide' : 'Past letters'}
        </button>
      </li>
    {/if}
  </ol>
{/if}

{#if story.open && !settings.quiet}
  {@const l = story.open}
  <div class="event-letter" role="dialog" aria-label={l.title}>
    <h3>{l.title}</h3>
    <p>{l.text}</p>
    <div class="row">
      <span class="dim">{fullTime(l.at)}</span>
      <button class="ok" onclick={() => story.ok(l)}>OK</button>
    </div>
  </div>
{/if}

{#if story.showHistory && !settings.quiet}
  <div class="history" role="dialog" aria-label="Past letters">
    <header>
      <strong>Past letters</strong>
      <button class="close" aria-label="Close" onclick={() => (story.showHistory = false)}>×</button>
    </header>
    <ul>
      {#each story.history as l (l.id)}
        <li>
          <strong>{l.title}</strong> <span class="dim">{fullTime(l.at)}</span>
          <p>{l.text}</p>
        </li>
      {/each}
    </ul>
  </div>
{/if}

{#if letters.open}
  {#key letters.open.id}
    <LetterDialog {letters} letter={letters.open} who={who(letters.open.run_id)} onopenrun={(id) => colony.openRun(id)} />
  {/key}
{/if}

<style>
  .letters {
    position: fixed;
    right: 10px;
    bottom: 60px;
    z-index: 30;
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .envelope {
    position: relative;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px 4px 4px;
    border-radius: var(--radius);
    border: 2px solid;
    cursor: pointer;
    box-shadow: 0 2px 6px rgb(0 0 0 / 0.4);
    animation: arrive 0.25s ease-out;
  }

  /* Permission: yellow, like a RimWorld letter that needs a decision. */
  .envelope.permission {
    background: color-mix(in srgb, #c9a227 35%, var(--panel-raised));
    border-color: #c9a227;
  }

  .envelope.question {
    background: color-mix(in srgb, var(--steel) 40%, var(--panel-raised));
    border-color: var(--steel-bright);
  }

  /* Event letters: the storyteller's, olive, asking nothing. */
  .envelope.story {
    background: color-mix(in srgb, var(--olive) 30%, var(--panel-raised));
    border-color: var(--olive-bright);
  }

  .envelope.story.raid {
    background: color-mix(in srgb, var(--rust) 35%, var(--panel-raised));
    border-color: var(--rust-bright);
  }

  .story-title {
    font-size: 12px;
    color: var(--text);
  }

  .history-toggle {
    font: inherit;
    font-size: 11px;
    padding: 2px 6px;
    color: var(--text-dim);
    background: var(--panel);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .event-letter,
  .history {
    position: fixed;
    right: 60px;
    bottom: 60px;
    z-index: 31;
    width: min(340px, calc(100vw - 80px));
    padding: 12px 14px;
    background: var(--panel);
    border: 2px solid var(--olive-bright);
    border-radius: var(--radius);
    box-shadow: 0 4px 14px rgb(0 0 0 / 0.5);
  }

  .event-letter h3 {
    margin: 0 0 6px;
    font-family: var(--font-display);
    font-size: 16px;
  }

  .event-letter p {
    margin: 0 0 10px;
    line-height: 1.4;
  }

  .row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .ok {
    font: inherit;
    font-family: var(--font-display);
    padding: 3px 16px;
    color: var(--text);
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .history {
    max-height: 60vh;
    overflow-y: auto;
    border-color: var(--frame);
  }

  .history header {
    display: flex;
    justify-content: space-between;
    margin-bottom: 6px;
  }

  .history ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .history p {
    margin: 2px 0 0;
    font-size: 12px;
  }

  .close {
    font: inherit;
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
  }

  .dim {
    color: var(--text-dim);
    font-size: 11px;
  }

  .mark {
    font-family: var(--font-display);
    font-weight: 700;
    color: var(--text);
  }

  @keyframes arrive {
    from {
      transform: translateX(40px);
      opacity: 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .envelope {
      animation: none;
    }
  }
</style>
