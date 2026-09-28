<script lang="ts">
  import Portrait from './Portrait.svelte'
  import LetterDialog from './LetterDialog.svelte'
  import type { Letters } from '../lib/letters.svelte'
  import type { Colony } from '../lib/colony.svelte'

  // RimWorld-style letters: an envelope per run that needs you, stacked on
  // the right edge, newest at the bottom. Click one to answer it.
  let { letters, colony }: { letters: Letters; colony: Colony } = $props()

  function who(runId: string) {
    const run = colony.workshop.byId(runId)
    const agent = colony.agents.find((a) => a.slug === run?.agent_slug)
    return { name: run?.agent_name ?? 'An agent', seed: agent?.portrait_seed || run?.agent_slug || runId, task: run?.task_title ?? '' }
  }
</script>

{#if letters.list.length}
  <ol class="letters" aria-label="Letters">
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
  </ol>
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
